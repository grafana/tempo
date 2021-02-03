package ingester

import (
	"context"
	"fmt"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

// Errors returned on Query.
var (
	ErrTraceMissing = errors.New("Trace missing")
)

var (
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricBytesWrittenTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_bytes_written_total",
		Help:      "The total bytes written per tenant.",
	}, []string{"tenant"})
	metricBlocksClearedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	})
)

type instance struct {
	tracesMtx sync.Mutex
	traces    map[uint32]*trace

	blocksMtx       sync.RWMutex
	headBlock       *wal.AppendBlock
	completingBlock *wal.AppendBlock
	completeBlocks  []*encoding.CompleteBlock

	lastBlockCut time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	bytesWrittenTotal  prometheus.Counter
	limiter            *Limiter
	writer             tempodb.Writer

	hash hash.Hash32
}

func newInstance(instanceID string, limiter *Limiter, writer tempodb.Writer) (*instance, error) {
	i := &instance{
		traces: map[uint32]*trace{},

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesWrittenTotal:  metricBytesWrittenTotal.WithLabelValues(instanceID),
		limiter:            limiter,
		writer:             writer,

		hash: fnv.New32(),
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *instance) Push(ctx context.Context, req *tempopb.PushRequest) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	trace, err := i.getOrCreateTrace(req)
	if err != nil {
		return err
	}

	if err := trace.Push(ctx, req); err != nil {
		return err
	}

	return nil
}

// PushBytes is used by the wal replay code and so it can push directly into the head block with 0 shenanigans
func (i *instance) PushBytes(ctx context.Context, id []byte, object []byte) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	return i.headBlock.Write(id, object)
}

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	tracesToCut := i.tracesToCut(cutoff, immediate)

	for _, t := range tracesToCut {
		out, err := proto.Marshal(t.trace)
		if err != nil {
			return err
		}

		err = i.writeTraceToHeadBlock(t.traceID, out)
		if err != nil {
			return err
		}
		i.bytesWrittenTotal.Add(float64(len(out)))
	}

	return nil
}

func (i *instance) CutBlockIfReady(maxBlockLifetime time.Duration, maxBlockBytes uint64, immediate bool) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return nil
	}

	now := time.Now()
	if i.lastBlockCut.Add(maxBlockLifetime).Before(now) || i.headBlock.DataLength() >= maxBlockBytes || immediate {
		if i.completingBlock != nil {
			return fmt.Errorf("unable to complete head block for %s b/c there is already a completing block.  Will try again next cycle", i.instanceID)
		}

		i.completingBlock = i.headBlock
		err := i.resetHeadBlock()
		if err != nil {
			return fmt.Errorf("failed to resetHeadBlock: %w", err)
		}

		// todo : this should be a queue of blocks to complete with workers
		go func() {
			completeBlock, err := i.writer.CompleteBlock(i.completingBlock, i)
			i.blocksMtx.Lock()
			defer i.blocksMtx.Unlock()

			if err != nil {
				// this is a really bad error that results in data loss.  most likely due to disk full
				_ = i.completingBlock.Clear()
				metricFailedFlushes.Inc()
				i.completingBlock = nil
				level.Error(cortex_util.Logger).Log("msg", "unable to complete block.  THIS BLOCK WAS LOST", "tenantID", i.instanceID, "err", err)
				return
			}
			i.completingBlock = nil
			i.completeBlocks = append(i.completeBlocks, completeBlock)
		}()
	}

	return nil
}

func (i *instance) GetBlockToBeFlushed() *encoding.CompleteBlock {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for _, c := range i.completeBlocks {
		if c.FlushedTime().IsZero() {
			return c
		}
	}

	return nil
}

func (i *instance) ClearFlushedBlocks(completeBlockTimeout time.Duration) error {
	var err error

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for idx, b := range i.completeBlocks {
		flushedTime := b.FlushedTime()
		if flushedTime.IsZero() {
			continue
		}

		if flushedTime.Add(completeBlockTimeout).Before(time.Now()) {
			i.completeBlocks = append(i.completeBlocks[:idx], i.completeBlocks[idx+1:]...)
			err = b.Clear() // todo: don't remove from complete blocks slice until after clear succeeds?
			if err == nil {
				metricBlocksClearedTotal.Inc()
			}
			break
		}
	}

	return err
}

func (i *instance) FindTraceByID(id []byte) (*tempopb.Trace, error) {
	var allBytes []byte

	// live traces
	i.tracesMtx.Lock()
	if liveTrace, ok := i.traces[i.tokenForTraceID(id)]; ok {
		foundBytes, err := proto.Marshal(liveTrace.trace)
		if err != nil {
			i.tracesMtx.Unlock()
			return nil, fmt.Errorf("unable to marshal liveTrace: %w", err)
		}

		allBytes = i.Combine(foundBytes, allBytes)
	}
	i.tracesMtx.Unlock()

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	// headBlock
	foundBytes, err := i.headBlock.Find(id, i)
	if err != nil {
		return nil, fmt.Errorf("headBlock.Find failed: %w", err)
	}
	allBytes = i.Combine(foundBytes, allBytes)

	// completingBlock
	if i.completingBlock != nil {
		foundBytes, err = i.completingBlock.Find(id, i)
		if err != nil {
			return nil, fmt.Errorf("completingBlock.Find failed: %w", err)
		}
		allBytes = i.Combine(foundBytes, allBytes)
	}

	// completeBlock
	for _, c := range i.completeBlocks {
		foundBytes, err = c.Find(id, i)
		if err != nil {
			return nil, fmt.Errorf("completeBlock.Find failed: %w", err)
		}
		allBytes = i.Combine(foundBytes, allBytes)
	}

	// now marshal it all
	if allBytes != nil {
		out := &tempopb.Trace{}

		err = proto.Unmarshal(allBytes, out)
		if err != nil {
			return nil, err
		}

		return out, nil
	}

	return nil, nil
}

// getOrCreateTrace will return a new trace object for the given request
//  It must be called under the i.tracesMtx lock
func (i *instance) getOrCreateTrace(req *tempopb.PushRequest) (*trace, error) {
	traceID, err := pushRequestTraceID(req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to extract traceID: %v", err)
	}

	fp := i.tokenForTraceID(traceID)
	trace, ok := i.traces[fp]
	if ok {
		return trace, nil
	}

	err = i.limiter.AssertMaxTracesPerUser(i.instanceID, len(i.traces))
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "max live traces per tenant exceeded: %v", err)
	}

	maxSpans := i.limiter.limits.MaxSpansPerTrace(i.instanceID)
	trace = newTrace(maxSpans, fp, traceID)
	i.traces[fp] = trace
	i.tracesCreatedTotal.Inc()

	return trace, nil
}

// tokenForTraceID hash trace ID, should be called under lock
func (i *instance) tokenForTraceID(id []byte) uint32 {
	i.hash.Reset()
	_, _ = i.hash.Write(id)
	return i.hash.Sum32()
}

// resetHeadBlock() should be called under lock
func (i *instance) resetHeadBlock() error {
	var err error
	i.headBlock, err = i.writer.WAL().NewBlock(uuid.New(), i.instanceID)
	i.lastBlockCut = time.Now()
	return err
}

func (i *instance) tracesToCut(cutoff time.Duration, immediate bool) []*trace {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	cutoffTime := time.Now().Add(cutoff)
	tracesToCut := make([]*trace, 0, len(i.traces))

	for key, trace := range i.traces {
		if cutoffTime.After(trace.lastAppend) || immediate {
			tracesToCut = append(tracesToCut, trace)
			delete(i.traces, key)
		}
	}

	return tracesToCut
}

func (i *instance) writeTraceToHeadBlock(id common.ID, b []byte) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	return i.headBlock.Write(id, b)
}

func (i *instance) Combine(objA []byte, objB []byte) []byte {
	combinedTrace, err := util.CombineTraces(objA, objB)
	if err != nil {
		level.Error(cortex_util.Logger).Log("msg", "error combining trace protos", "err", err.Error())
	}
	return combinedTrace
}

// pushRequestTraceID gets the TraceID of the first span in the batch and assumes its the trace ID throughout
//  this assumption should hold b/c the distributors make sure each batch all belong to the same trace
func pushRequestTraceID(req *tempopb.PushRequest) ([]byte, error) {
	if req == nil || req.Batch == nil {
		return nil, errors.New("req or req.Batch nil")
	}

	if len(req.Batch.InstrumentationLibrarySpans) == 0 {
		return nil, errors.New("InstrumentationLibrarySpans has length 0")
	}

	if len(req.Batch.InstrumentationLibrarySpans[0].Spans) == 0 {
		return nil, errors.New("Spans has length 0")
	}

	return req.Batch.InstrumentationLibrarySpans[0].Spans[0].TraceId, nil
}
