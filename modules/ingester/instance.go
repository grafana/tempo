package ingester

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type traceTooLargeError struct {
	traceID           common.ID
	instanceID        string
	maxBytes, reqSize int
}

func newTraceTooLargeError(traceID common.ID, instanceID string, maxBytes, reqSize int) *traceTooLargeError {
	return &traceTooLargeError{
		traceID:    traceID,
		instanceID: instanceID,
		maxBytes:   maxBytes,
		reqSize:    reqSize,
	}
}

func (e traceTooLargeError) Error() string {
	return fmt.Sprintf(
		"%s max size of trace (%d) exceeded while adding %d bytes to trace %s for tenant %s",
		overrides.ErrorPrefixTraceTooLarge, e.maxBytes, e.reqSize, hex.EncodeToString(e.traceID), e.instanceID)
}

// Errors returned on Query.
var (
	ErrTraceMissing = errors.New("Trace missing")
)

const (
	traceDataType = "trace"
)

var (
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricLiveTraces = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "ingester_live_traces",
		Help:      "The current number of lives traces per tenant.",
	}, []string{"tenant"})
	metricBlocksClearedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	})
	metricBytesReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_bytes_received_total",
		Help:      "The total bytes received per tenant.",
	}, []string{"tenant", "data_type"})
	metricReplayErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_replay_errors_total",
		Help:      "The total number of replay errors received per tenant.",
	}, []string{"tenant"})
)

type instance struct {
	tracesMtx  sync.Mutex
	traces     map[uint32]*liveTrace
	traceSizes map[uint32]uint32
	traceCount atomic.Int32

	blocksMtx        sync.RWMutex
	headBlock        common.WALBlock
	completingBlocks []common.WALBlock
	completeBlocks   []*localBlock

	lastBlockCut time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	bytesReceivedTotal *prometheus.CounterVec
	limiter            *Limiter
	writer             tempodb.Writer

	local       *local.Backend
	localReader backend.Reader
	localWriter backend.Writer

	hash hash.Hash32
}

func newInstance(instanceID string, limiter *Limiter, writer tempodb.Writer, l *local.Backend) (*instance, error) {
	i := &instance{
		traces:     map[uint32]*liveTrace{},
		traceSizes: map[uint32]uint32{},

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesReceivedTotal: metricBytesReceivedTotal,
		limiter:            limiter,
		writer:             writer,

		local:       l,
		localReader: backend.NewReader(l),
		localWriter: backend.NewWriter(l),

		hash: fnv.New32(),
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *instance) PushBytesRequest(ctx context.Context, req *tempopb.PushBytesRequest) error {
	for j := range req.Traces {
		err := i.PushBytes(ctx, req.Ids[j].Slice, req.Traces[j].Slice)
		if err != nil {
			return err
		}
	}
	return nil
}

// PushBytes is used to push an unmarshalled tempopb.Trace to the instance
func (i *instance) PushBytes(ctx context.Context, id []byte, traceBytes []byte) error {
	i.measureReceivedBytes(traceBytes)

	if !validation.ValidTraceID(id) {
		return status.Errorf(codes.InvalidArgument, "%s is not a valid traceid", hex.EncodeToString(id))
	}

	// check for max traces before grabbing the lock to better load shed
	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, int(i.traceCount.Load()))
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "%s max live traces exceeded for tenant %s: %v", overrides.ErrorPrefixLiveTracesExceeded, i.instanceID, err)
	}

	return i.push(ctx, id, traceBytes)
}

func (i *instance) push(ctx context.Context, id, traceBytes []byte) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	tkn := i.tokenForTraceID(id)
	maxBytes := i.limiter.limits.MaxBytesPerTrace(i.instanceID)

	if maxBytes > 0 {
		prevSize := int(i.traceSizes[tkn])
		reqSize := len(traceBytes)
		if prevSize+reqSize > maxBytes {
			return status.Errorf(codes.FailedPrecondition, (newTraceTooLargeError(id, i.instanceID, maxBytes, reqSize).Error()))
		}
	}

	trace := i.getOrCreateTrace(id, tkn, maxBytes)

	err := trace.Push(ctx, i.instanceID, traceBytes)
	if err != nil {
		if e, ok := err.(*traceTooLargeError); ok {
			return status.Errorf(codes.FailedPrecondition, e.Error())
		}
		return err
	}

	if maxBytes > 0 {
		i.traceSizes[tkn] += uint32(len(traceBytes))
	}

	return nil
}

func (i *instance) measureReceivedBytes(traceBytes []byte) {
	// measure received bytes as sum of slice lengths
	// type byte is guaranteed to be 1 byte in size
	// ref: https://golang.org/ref/spec#Size_and_alignment_guarantees
	i.bytesReceivedTotal.WithLabelValues(i.instanceID, traceDataType).Add(float64(len(traceBytes)))
}

// CutCompleteTraces moves any complete traces out of the map to complete traces.
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	tracesToCut := i.tracesToCut(cutoff, immediate)
	segmentDecoder := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// Sort by ID
	sort.Slice(tracesToCut, func(i, j int) bool {
		return bytes.Compare(tracesToCut[i].traceID, tracesToCut[j].traceID) == -1
	})

	for _, t := range tracesToCut {
		// sort batches before cutting to reduce combinations during compaction
		sortByteSlices(t.batches)

		out, err := segmentDecoder.ToObject(t.batches)
		if err != nil {
			return err
		}

		err = i.writeTraceToHeadBlock(t.traceID, out, t.start, t.end)
		if err != nil {
			return err
		}

		// return trace byte slices to be reused by proto marshalling
		//  WARNING: can't reuse traceid's b/c the appender takes ownership of byte slices that are passed to it
		tempopb.ReuseByteSlices(t.batches)
	}

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()
	return i.headBlock.Flush()
}

// CutBlockIfReady cuts a completingBlock from the HeadBlock if ready.
// Returns the ID of a block if one was cut or a nil ID if one was not cut, along with the error (if any).
func (i *instance) CutBlockIfReady(maxBlockLifetime time.Duration, maxBlockBytes uint64, immediate bool) (uuid.UUID, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	now := time.Now()
	if i.lastBlockCut.Add(maxBlockLifetime).Before(now) || i.headBlock.DataLength() >= maxBlockBytes || immediate {

		// Final flush
		err := i.headBlock.Flush()
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to flush head block: %w", err)
		}

		completingBlock := i.headBlock

		i.completingBlocks = append(i.completingBlocks, completingBlock)

		err = i.resetHeadBlock()
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to resetHeadBlock: %w", err)
		}

		return completingBlock.BlockMeta().BlockID, nil
	}

	return uuid.Nil, nil
}

// CompleteBlock moves a completingBlock to a completeBlock. The new completeBlock has the same ID.
func (i *instance) CompleteBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()
	var completingBlock common.WALBlock
	for _, iterBlock := range i.completingBlocks {
		if iterBlock.BlockMeta().BlockID == blockID {
			completingBlock = iterBlock
			break
		}
	}
	i.blocksMtx.Unlock()

	if completingBlock == nil {
		return fmt.Errorf("error finding completingBlock")
	}

	ctx := context.Background()

	backendBlock, err := i.writer.CompleteBlockWithBackend(ctx, completingBlock, i.localReader, i.localWriter)
	if err != nil {
		return errors.Wrap(err, "error completing wal block with local backend")
	}

	ingesterBlock := newLocalBlock(ctx, backendBlock, i.local)

	i.blocksMtx.Lock()
	i.completeBlocks = append(i.completeBlocks, ingesterBlock)
	i.blocksMtx.Unlock()

	return nil
}

func (i *instance) ClearCompletingBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	var completingBlock common.WALBlock
	for j, iterBlock := range i.completingBlocks {
		if iterBlock.BlockMeta().BlockID == blockID {
			completingBlock = iterBlock
			i.completingBlocks = append(i.completingBlocks[:j], i.completingBlocks[j+1:]...)
			break
		}
	}

	if completingBlock != nil {
		return completingBlock.Clear()
	}

	return errors.New("Error finding wal completingBlock to clear")
}

// GetBlockToBeFlushed gets a list of blocks that can be flushed to the backend.
func (i *instance) GetBlockToBeFlushed(blockID uuid.UUID) *localBlock {
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

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

func (i *instance) FindTraceByID(ctx context.Context, id []byte) (*tempopb.Trace, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.FindTraceByID")
	defer span.Finish()

	var err error
	var completeTrace *tempopb.Trace

	// live traces
	i.tracesMtx.Lock()
	if liveTrace, ok := i.traces[i.tokenForTraceID(id)]; ok {
		completeTrace, err = model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForRead(liveTrace.batches)
		if err != nil {
			i.tracesMtx.Unlock()
			return nil, fmt.Errorf("unable to unmarshal liveTrace: %w", err)
		}
	}
	i.tracesMtx.Unlock()

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	combiner := trace.NewCombiner()
	combiner.Consume(completeTrace)

	// headBlock
	tr, err := i.headBlock.FindTraceByID(ctx, id, common.DefaultSearchOptions())
	if err != nil {
		return nil, fmt.Errorf("headBlock.FindTraceByID failed: %w", err)
	}
	combiner.Consume(tr)

	// completingBlock
	for _, c := range i.completingBlocks {
		tr, err = c.FindTraceByID(ctx, id, common.DefaultSearchOptions())
		if err != nil {
			return nil, fmt.Errorf("completingBlock.FindTraceByID failed: %w", err)
		}
		combiner.Consume(tr)
	}

	// completeBlock
	for _, c := range i.completeBlocks {
		found, err := c.FindTraceByID(ctx, id, common.DefaultSearchOptions())
		if err != nil {
			return nil, fmt.Errorf("completeBlock.FindTraceByID failed: %w", err)
		}
		combiner.Consume(found)
	}

	result, _ := combiner.Result()
	return result, nil
}

// AddCompletingBlock adds an AppendBlock directly to the slice of completing blocks.
// This is used during wal replay. It is expected that calling code will add the appropriate
// jobs to the queue to eventually flush these.
func (i *instance) AddCompletingBlock(b common.WALBlock) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	i.completingBlocks = append(i.completingBlocks, b)
}

// getOrCreateTrace will return a new trace object for the given request
//
//	It must be called under the i.tracesMtx lock
func (i *instance) getOrCreateTrace(traceID []byte, fp uint32, maxBytes int) *liveTrace {
	trace, ok := i.traces[fp]
	if ok {
		return trace
	}

	trace = newTrace(traceID, maxBytes)
	i.traces[fp] = trace
	i.tracesCreatedTotal.Inc()
	i.traceCount.Inc()

	return trace
}

// tokenForTraceID hash trace ID, should be called under lock
func (i *instance) tokenForTraceID(id []byte) uint32 {
	i.hash.Reset()
	_, _ = i.hash.Write(id)
	return i.hash.Sum32()
}

// resetHeadBlock() should be called under lock
func (i *instance) resetHeadBlock() error {

	// Reset trace sizes when cutting block
	i.tracesMtx.Lock()
	i.traceSizes = make(map[uint32]uint32, len(i.traceSizes))
	i.tracesMtx.Unlock()

	newHeadBlock, err := i.writer.WAL().NewBlock(uuid.New(), i.instanceID, model.CurrentEncoding)
	if err != nil {
		return err
	}

	i.headBlock = newHeadBlock
	i.lastBlockCut = time.Now()

	return nil
}

func (i *instance) tracesToCut(cutoff time.Duration, immediate bool) []*liveTrace {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	// Set this before cutting to give a more accurate number.
	metricLiveTraces.WithLabelValues(i.instanceID).Set(float64(len(i.traces)))

	cutoffTime := time.Now().Add(cutoff)
	tracesToCut := make([]*liveTrace, 0, len(i.traces))

	for key, trace := range i.traces {
		if cutoffTime.After(trace.lastAppend) || immediate {
			tracesToCut = append(tracesToCut, trace)
			delete(i.traces, key)
		}
	}
	i.traceCount.Store(int32(len(i.traces)))

	return tracesToCut
}

func (i *instance) writeTraceToHeadBlock(id common.ID, b []byte, start, end uint32) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	err := i.headBlock.Append(id, b, start, end)
	if err != nil {
		return err
	}

	return nil
}

func (i *instance) rediscoverLocalBlocks(ctx context.Context) ([]*localBlock, error) {
	ids, err := i.localReader.Blocks(ctx, i.instanceID)
	if err != nil {
		return nil, err
	}

	hasWal := func(id uuid.UUID) bool {
		i.blocksMtx.RLock()
		defer i.blocksMtx.RUnlock()
		for _, b := range i.completingBlocks {
			if b.BlockMeta().BlockID == id {
				return true
			}
		}
		return false
	}

	var rediscoveredBlocks []*localBlock

	for _, id := range ids {

		// Ignore blocks that have a matching wal. The wal will be replayed and the local block recreated.
		// NOTE - Wal replay must be done beforehand.
		if hasWal(id) {
			continue
		}

		// See if block is intact by checking for meta, which is written last.
		// If meta missing then block was not successfully written.
		meta, err := i.localReader.BlockMeta(ctx, id, i.instanceID)
		if err != nil {
			if err == backend.ErrDoesNotExist {
				// Partial/incomplete block found, remove, it will be recreated from data in the wal.
				level.Warn(log.Logger).Log("msg", "Unable to reload meta for local block. This indicates an incomplete block and will be deleted", "tenant", i.instanceID, "block", id.String())
				err = i.local.ClearBlock(id, i.instanceID)
				if err != nil {
					return nil, errors.Wrapf(err, "deleting bad local block tenant %v block %v", i.instanceID, id.String())
				}
			} else {
				// Block with unknown error
				level.Error(log.Logger).Log("msg", "Unexpected error reloading meta for local block. Ignoring and continuing. This block should be investigated.", "tenant", i.instanceID, "block", id.String(), "error", err)
				metricReplayErrorsTotal.WithLabelValues(i.instanceID).Inc()
			}

			continue
		}

		b, err := encoding.OpenBlock(meta, i.localReader)
		if err != nil {
			return nil, err
		}

		ib := newLocalBlock(ctx, b, i.local)
		rediscoveredBlocks = append(rediscoveredBlocks, ib)

		level.Info(log.Logger).Log("msg", "reloaded local block", "tenantID", i.instanceID, "block", id.String(), "flushed", ib.FlushedTime())
	}

	i.blocksMtx.Lock()
	i.completeBlocks = append(i.completeBlocks, rediscoveredBlocks...)
	i.blocksMtx.Unlock()

	return rediscoveredBlocks, nil
}

// sortByteSlices sorts a []byte
func sortByteSlices(buffs [][]byte) {
	sort.Slice(buffs, func(i, j int) bool {
		traceI := buffs[i]
		traceJ := buffs[j]

		return bytes.Compare(traceI, traceJ) == -1
	})
}
