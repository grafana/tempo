package ingester

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	tempodb_encoding "github.com/grafana/tempo/tempodb/encoding"
	tempodb_wal "github.com/grafana/tempo/tempodb/wal"
)

type traceFingerprint uint64

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
	metricBlocksClearedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	})
)

type instance struct {
	tracesMtx sync.Mutex
	traces    map[traceFingerprint]*trace

	blocksMtx       sync.RWMutex
	headBlock       *tempodb_wal.AppendBlock
	completingBlock *tempodb_wal.AppendBlock
	completeBlocks  []*tempodb_wal.CompleteBlock
	lastBlockCut    time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	limiter            *Limiter
	wal                *tempodb_wal.WAL
}

func newInstance(instanceID string, limiter *Limiter, wal *tempodb_wal.WAL) (*instance, error) {
	i := &instance{
		traces: map[traceFingerprint]*trace{},

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		limiter:            limiter,
		wal:                wal,
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
func (i *instance) PushBytes(ctx context.Context, id tempodb_encoding.ID, object []byte) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	return i.headBlock.Write(id, object)
}

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	now := time.Now()
	for key, trace := range i.traces {
		if now.Add(cutoff).After(trace.lastAppend) || immediate {
			out, err := proto.Marshal(trace.trace)
			if err != nil {
				return err
			}

			err = i.headBlock.Write(trace.traceID, out)
			if err != nil {
				return err
			}

			delete(i.traces, key)
		}
	}

	return nil
}

func (i *instance) CutBlockIfReady(maxTracesPerBlock int, maxBlockLifetime time.Duration, immediate bool) (bool, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.Length() == 0 {
		return false, nil
	}

	now := time.Now()
	ready := i.headBlock.Length() >= maxTracesPerBlock || i.lastBlockCut.Add(maxBlockLifetime).Before(now) || immediate
	if ready {
		if i.completingBlock != nil {
			return false, fmt.Errorf("unable to complete head block for %s b/c there is already a completing block.  Will try again next cycle", i.instanceID)
		}

		i.completingBlock = i.headBlock
		i.resetHeadBlock()

		// todo : this should be a queue of blocks to complete with workers
		go func(toComplete *tempodb_wal.AppendBlock) {
			completeBlock, err := i.completingBlock.Complete(i.wal, i)

			i.blocksMtx.Lock()
			defer i.blocksMtx.Unlock()

			if err != nil {
				// this is a really bad error that results in data loss.  most likely due to disk full
				i.completingBlock.Clear()
				i.completingBlock = nil
				level.Error(cortex_util.Logger).Log("msg", "unable to complete block.  THIS BLOCK WAS LOST", "tenantID", i.instanceID, "err", err)
				return
			}
			i.completingBlock = nil
			i.completeBlocks = append(i.completeBlocks, completeBlock)
		}(i.completingBlock)
	}

	return ready, nil
}

func (i *instance) GetBlockToBeFlushed() *tempodb_wal.CompleteBlock {
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
	if liveTrace, ok := i.traces[traceFingerprint(util.Fingerprint(id))]; ok {
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

func (i *instance) getOrCreateTrace(req *tempopb.PushRequest) (*trace, error) {
	traceID, err := pushRequestTraceID(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get trace id %w", err)
	}

	fp := traceFingerprint(util.Fingerprint(traceID))

	trace, ok := i.traces[fp]
	if ok {
		return trace, nil
	}

	err = i.limiter.AssertMaxTracesPerUser(i.instanceID, len(i.traces))
	if err != nil {
		return nil, httpgrpc.Errorf(http.StatusTooManyRequests, err.Error())
	}

	trace = newTrace(fp, traceID)
	i.traces[fp] = trace
	i.tracesCreatedTotal.Inc()

	return trace, nil
}

// resetHeadBlock() should be called under lock
func (i *instance) resetHeadBlock() error {
	var err error
	i.headBlock, err = i.wal.NewBlock(uuid.New(), i.instanceID)
	i.lastBlockCut = time.Now()
	return err
}

func (i *instance) Combine(objA []byte, objB []byte) []byte {
	return util.CombineTraces(objA, objB)
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
