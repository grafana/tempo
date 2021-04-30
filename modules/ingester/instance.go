package ingester

import (
	"context"
	"fmt"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
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
	tracesMtx  sync.Mutex
	traces     map[uint32]*trace
	traceCount atomic.Int32

	blocksMtx        sync.RWMutex
	headBlock        *wal.AppendBlock
	completingBlocks []*wal.AppendBlock
	completeBlocks   []*wal.LocalBlock

	lastBlockCut time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	bytesWrittenTotal  prometheus.Counter
	limiter            *Limiter
	writer             tempodb.Writer
	local              *local.Backend

	hash hash.Hash32
}

func newInstance(instanceID string, limiter *Limiter, writer tempodb.Writer, l *local.Backend) (*instance, error) {
	i := &instance{
		traces: map[uint32]*trace{},

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesWrittenTotal:  metricBytesWrittenTotal.WithLabelValues(instanceID),
		limiter:            limiter,
		writer:             writer,
		local:              l,

		hash: fnv.New32(),
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *instance) Push(ctx context.Context, req *tempopb.PushRequest) error {
	// check for max traces before grabbing the lock to better load shed
	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, int(i.traceCount.Load()))
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "%s max live traces per tenant exceeded: %v", overrides.ErrorPrefixLiveTracesExceeded, err)
	}

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

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	tracesToCut := i.tracesToCut(cutoff, immediate)

	for _, t := range tracesToCut {

		util.SortTrace(t.trace)

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

// CutBlockIfReady cuts a completingBlock from the HeadBlock if ready
// Returns a bool indicating if a block was cut along with the error (if any).
func (i *instance) CutBlockIfReady(maxBlockLifetime time.Duration, maxBlockBytes uint64, immediate bool) (uuid.UUID, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	now := time.Now()
	if i.lastBlockCut.Add(maxBlockLifetime).Before(now) || i.headBlock.DataLength() >= maxBlockBytes || immediate {
		completingBlock := i.headBlock

		i.completingBlocks = append(i.completingBlocks, completingBlock)

		err := i.resetHeadBlock()
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to resetHeadBlock: %w", err)
		}

		return completingBlock.BlockID(), nil
	}

	return uuid.Nil, nil
}

// CompleteBlock() moves a completingBlock to a completeBlock. The new completeBlock has the same ID
func (i *instance) CompleteBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()

	var completingBlock *wal.AppendBlock
	for _, iterBlock := range i.completingBlocks {
		if iterBlock.BlockID() == blockID {
			completingBlock = iterBlock
			break
		}
	}
	i.blocksMtx.Unlock()

	if completingBlock == nil {
		return fmt.Errorf("error finding completingBlock")
	}

	ctx := context.Background()

	backendBlock, err := i.writer.CompleteBlockWithBackend(ctx, completingBlock, i, i.local, i.local)
	if err != nil {
		return errors.Wrap(err, "error completing wal block with local backend")
	}

	ingesterBlock, err := wal.NewLocalBlock(ctx, backendBlock, i.local)
	if err != nil {
		return errors.Wrap(err, "error creating ingester block")
	}

	i.blocksMtx.Lock()
	i.completeBlocks = append(i.completeBlocks, ingesterBlock)
	i.blocksMtx.Unlock()

	return nil
}

// nolint:interfacer
func (i *instance) ClearCompletingBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()
	var completingBlock *wal.AppendBlock
	for j, iterBlock := range i.completingBlocks {
		if iterBlock.BlockID() == blockID {
			completingBlock = iterBlock
			i.completingBlocks = append(i.completingBlocks[:j], i.completingBlocks[j+1:]...)
			break
		}
	}
	i.blocksMtx.Unlock()

	if completingBlock != nil {
		return completingBlock.Clear()
	}

	return fmt.Errorf("Error finding wal completingBlock to clear")
}

// GetBlockToBeFlushed gets a list of blocks that can be flushed to the backend
func (i *instance) GetBlockToBeFlushed(blockID uuid.UUID) *wal.LocalBlock {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for _, c := range i.completeBlocks {
		if c.BlockMeta().BlockID == blockID && c.FlushedTime().IsZero() {
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
			err = i.local.ClearBlock(b.BlockMeta().BlockID, i.instanceID)
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
	for _, c := range i.completingBlocks {
		foundBytes, err = c.Find(id, i)
		if err != nil {
			return nil, fmt.Errorf("completingBlock.Find failed: %w", err)
		}
		allBytes = i.Combine(foundBytes, allBytes)
	}

	// completeBlock
	for _, c := range i.completeBlocks {
		foundBytes, err = c.Find(context.TODO(), id)
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

// AddCompletingBlock adds an AppendBlock directly to the slice of completing blocks.
// This is used during wal replay. It is expected that calling code will add the appropriate
// jobs to the queue to eventually flush these.
func (i *instance) AddCompletingBlock(b *wal.AppendBlock) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	i.completingBlocks = append(i.completingBlocks, b)
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

	maxBytes := i.limiter.limits.MaxBytesPerTrace(i.instanceID)
	trace = newTrace(maxBytes, fp, traceID)
	i.traces[fp] = trace
	i.tracesCreatedTotal.Inc()
	i.traceCount.Inc()

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
	i.traceCount.Store(int32(len(i.traces)))

	return tracesToCut
}

func (i *instance) writeTraceToHeadBlock(id common.ID, b []byte) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	return i.headBlock.Write(id, b)
}

func (i *instance) Combine(objA []byte, objB []byte) []byte {
	combinedTrace, _, err := util.CombineTraces(objA, objB)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error combining trace protos", "err", err.Error())
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

func (i *instance) rediscoverLocalBlocks(ctx context.Context) error {
	ids, err := i.local.Blocks(ctx, i.instanceID)
	if err != nil {
		return err
	}

	for _, id := range ids {
		meta, err := i.local.BlockMeta(ctx, id, i.instanceID)
		if err != nil {
			if err == backend.ErrMetaDoesNotExist {
				// Partial/incomplete block found, remove, it will be recreated from data in the wal.
				level.Warn(log.Logger).Log("msg", "Unable to reload meta for local block. This indicates an incomplete block and will be deleted", "tenant", i.instanceID, "block", id.String())
				err = i.local.ClearBlock(id, i.instanceID)
				if err != nil {
					return errors.Wrapf(err, "deleting bad local block tenant %v block %v", i.instanceID, id.String())
				}
				continue
			}

			return err
		}

		b, err := encoding.NewBackendBlock(meta, i.local)
		if err != nil {
			return err
		}

		ib, err := wal.NewLocalBlock(ctx, b, i.local)
		if err != nil {
			return err
		}

		i.blocksMtx.Lock()
		i.completeBlocks = append(i.completeBlocks, ib)
		i.blocksMtx.Unlock()

		level.Info(log.Logger).Log("msg", "reloaded local block", "tenantID", i.instanceID, "block", id.String(), "flushed", ib.FlushedTime())
	}

	return nil
}
