package ingester

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/tracesizes"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var (
	errTraceTooLarge = errors.New(overrides.ErrorPrefixTraceTooLarge)
	errMaxLiveTraces = errors.New(overrides.ErrorPrefixLiveTracesExceeded)
)

const (
	traceDataType             = "trace"
	maxTraceLogLinesPerSecond = 10
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
	metricLiveTraceBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "ingester_live_trace_bytes",
		Help:      "The current number of bytes consumed by lives traces per tenant.",
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
	tracesMtx      sync.Mutex
	traces         map[uint64]*liveTrace
	traceSizes     *tracesizes.Tracker
	traceSizeBytes uint64

	headBlockMtx sync.RWMutex
	headBlock    common.WALBlock

	blocksMtx        sync.RWMutex
	completingBlocks []common.WALBlock
	completeBlocks   []*LocalBlock

	lastBlockCut time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	bytesReceivedTotal *prometheus.CounterVec
	limiter            Limiter
	writer             tempodb.Writer

	dedicatedColumns backend.DedicatedColumns
	overrides        ingesterOverrides

	local       *local.Backend
	localReader backend.Reader
	localWriter backend.Writer

	logger         kitlog.Logger
	maxTraceLogger *log.RateLimitedLogger
}

func newInstance(instanceID string, limiter Limiter, overrides ingesterOverrides, writer tempodb.Writer, l *local.Backend, dedicatedColumns backend.DedicatedColumns) (*instance, error) {
	logger := kitlog.With(log.Logger, "tenant", instanceID)
	i := &instance{
		traces:     map[uint64]*liveTrace{},
		traceSizes: tracesizes.New(),

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesReceivedTotal: metricBytesReceivedTotal,
		limiter:            limiter,
		writer:             writer,

		dedicatedColumns: dedicatedColumns,
		overrides:        overrides,

		local:       l,
		localReader: backend.NewReader(l),
		localWriter: backend.NewWriter(l),

		logger:         logger,
		maxTraceLogger: log.NewRateLimitedLogger(maxTraceLogLinesPerSecond, level.Warn(logger)),
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *instance) PushBytesRequest(ctx context.Context, req *tempopb.PushBytesRequest) *tempopb.PushResponse {
	pr := &tempopb.PushResponse{}

	for j := range req.Traces {
		err := i.PushBytes(ctx, req.Ids[j], req.Traces[j].Slice)
		pr.ErrorsByTrace = i.addTraceError(pr.ErrorsByTrace, err, len(req.Traces), j)
	}

	return pr
}

func (i *instance) addTraceError(errorsByTrace []tempopb.PushErrorReason, pushError error, numTraces int, traceIndex int) []tempopb.PushErrorReason {
	if pushError != nil {
		// only make list if there is at least one error
		if len(errorsByTrace) == 0 {
			errorsByTrace = make([]tempopb.PushErrorReason, 0, numTraces)
			// because this is the first error, fill list with NO_ERROR for the traces before this one
			for k := 0; k < traceIndex; k++ {
				errorsByTrace = append(errorsByTrace, tempopb.PushErrorReason_NO_ERROR)
			}
		}
		if errors.Is(pushError, errMaxLiveTraces) {
			errorsByTrace = append(errorsByTrace, tempopb.PushErrorReason_MAX_LIVE_TRACES)
			return errorsByTrace
		}

		if errors.Is(pushError, errTraceTooLarge) {
			errorsByTrace = append(errorsByTrace, tempopb.PushErrorReason_TRACE_TOO_LARGE)
			return errorsByTrace
		}

		// error is not either MaxLiveTraces or TraceTooLarge
		level.Error(i.logger).Log("msg", "Unexpected error during PushBytes", "error", pushError)
		errorsByTrace = append(errorsByTrace, tempopb.PushErrorReason_UNKNOWN_ERROR)
		return errorsByTrace

	} else if len(errorsByTrace) > 0 {
		errorsByTrace = append(errorsByTrace, tempopb.PushErrorReason_NO_ERROR)
	}

	return errorsByTrace
}

// PushBytes is used to push an unmarshalled tempopb.Trace to the instance
func (i *instance) PushBytes(ctx context.Context, id, traceBytes []byte) error {
	i.measureReceivedBytes(traceBytes)

	if !validation.ValidTraceID(id) {
		return status.Errorf(codes.InvalidArgument, "%s is not a valid traceid", hex.EncodeToString(id))
	}

	return i.push(ctx, id, traceBytes)
}

func (i *instance) push(ctx context.Context, id, traceBytes []byte) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, len(i.traces))
	if err != nil {
		return errMaxLiveTraces
	}

	maxBytes := i.limiter.Limits().MaxBytesPerTrace(i.instanceID)
	reqSize := len(traceBytes)

	if maxBytes > 0 && !i.traceSizes.Allow(id, reqSize, maxBytes) {
		i.maxTraceLogger.Log("msg", overrides.ErrorPrefixTraceTooLarge, "max", maxBytes, "size", reqSize, "trace", hex.EncodeToString(id))
		return errTraceTooLarge
	}

	tkn := util.HashForTraceID(id)
	trace := i.getOrCreateTrace(id, tkn)

	err = trace.Push(ctx, i.instanceID, traceBytes)
	if err != nil {
		return err
	}

	i.traceSizeBytes += uint64(reqSize)

	return nil
}

func (i *instance) measureReceivedBytes(traceBytes []byte) {
	// measure received bytes as sum of slice lengths
	// type byte is guaranteed to be 1 byte in size
	// ref: https://golang.org/ref/spec#Size_and_alignment_guarantees
	i.bytesReceivedTotal.WithLabelValues(i.instanceID, traceDataType).Add(float64(len(traceBytes)))
}

// CutCompleteTraces moves any complete traces out of the map to complete traces.
func (i *instance) CutCompleteTraces(idleCutoff time.Duration, liveCutoff time.Duration, immediate bool) error {
	tracesToCut := i.tracesToCut(time.Now(), idleCutoff, liveCutoff, immediate)
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

	i.headBlockMtx.Lock()
	defer i.headBlockMtx.Unlock()
	return i.headBlock.Flush()
}

// CutBlockIfReady cuts a completingBlock from the HeadBlock if ready.
// Returns the ID of a block if one was cut or a nil ID if one was not cut, along with the error (if any).
func (i *instance) CutBlockIfReady(maxBlockLifetime time.Duration, maxBlockBytes uint64, immediate bool) (uuid.UUID, error) {
	i.headBlockMtx.Lock()
	defer i.headBlockMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	now := time.Now()
	if i.lastBlockCut.Add(maxBlockLifetime).Before(now) || i.headBlock.DataLength() >= maxBlockBytes || immediate {
		// Reset trace sizes when cutting block
		i.traceSizes.ClearIdle(i.lastBlockCut)

		// Final flush
		err := i.headBlock.Flush()
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to flush head block: %w", err)
		}

		completingBlock := i.headBlock

		// Now that we are adding a new block take the blocks mutex.
		// A warning about deadlocks!!  This area does a hard-acquire of both mutexes.
		// To avoid deadlocks this function and all others must acquire them in
		// the ** same_order ** or else!!! i.e. another function can't acquire blocksMtx
		// then headblockMtx. Even if the likelihood is low it is a statistical certainly
		// that eventually a deadlock will occur.
		i.blocksMtx.Lock()
		defer i.blocksMtx.Unlock()

		i.completingBlocks = append(i.completingBlocks, completingBlock)

		err = i.resetHeadBlock()
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to resetHeadBlock: %w", err)
		}

		return (uuid.UUID)(completingBlock.BlockMeta().BlockID), nil
	}

	return uuid.Nil, nil
}

// CompleteBlock moves a completingBlock to a completeBlock. The new completeBlock has the same ID.
func (i *instance) CompleteBlock(ctx context.Context, blockID uuid.UUID) error {
	i.blocksMtx.Lock()
	var completingBlock common.WALBlock
	for _, iterBlock := range i.completingBlocks {
		if (uuid.UUID)(iterBlock.BlockMeta().BlockID) == blockID {
			completingBlock = iterBlock
			break
		}
	}
	i.blocksMtx.Unlock()

	if completingBlock == nil {
		return fmt.Errorf("error finding completingBlock")
	}

	backendBlock, err := i.writer.CompleteBlockWithBackend(ctx, completingBlock, i.localReader, i.localWriter)
	if err != nil {
		return fmt.Errorf("error completing wal block with local backend: %w", err)
	}

	ingesterBlock := NewLocalBlock(ctx, backendBlock, i.local)

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
		if (uuid.UUID)(iterBlock.BlockMeta().BlockID) == blockID {
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
func (i *instance) GetBlockToBeFlushed(blockID uuid.UUID) *LocalBlock {
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	for _, c := range i.completeBlocks {
		if (uuid.UUID)(c.BlockMeta().BlockID) == blockID && c.FlushedTime().IsZero() {
			return c
		}
	}

	return nil
}

func (i *instance) ClearOldBlocks(flushObjectStorage bool, completeBlockTimeout time.Duration) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	cutoff := time.Now().Add(-completeBlockTimeout)
	for idx, numBlocks := 0, len(i.completeBlocks); idx < numBlocks; idx++ {
		b := i.completeBlocks[idx]

		// Keep blocks awaiting flush if we are flushing to object storage.
		if flushObjectStorage && b.FlushedTime().IsZero() {
			continue
		}

		// Keep blocks within retention.
		if b.BlockMeta().EndTime.After(cutoff) {
			continue
		}

		// This block can be deleted.
		err := i.local.ClearBlock((uuid.UUID)(b.BlockMeta().BlockID), i.instanceID)
		if err != nil {
			return err
		}
		i.completeBlocks = append(i.completeBlocks[:idx], i.completeBlocks[idx+1:]...)
		idx--
		numBlocks--
		metricBlocksClearedTotal.Inc()
	}

	return nil
}

func (i *instance) FindTraceByID(ctx context.Context, id []byte, allowPartialTrace bool) (*tempopb.TraceByIDResponse, error) {
	ctx, span := tracer.Start(ctx, "instance.FindTraceByID")
	defer span.End()

	var err error
	var completeTrace *tempopb.Trace
	metrics := tempopb.TraceByIDMetrics{}

	// live traces
	i.tracesMtx.Lock()
	if liveTrace, ok := i.traces[util.HashForTraceID(id)]; ok {
		completeTrace, err = model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForRead(liveTrace.batches)
		for _, b := range liveTrace.batches {
			metrics.InspectedBytes += uint64(len(b))
		}
		if err != nil {
			i.tracesMtx.Unlock()
			return nil, fmt.Errorf("unable to unmarshal liveTrace: %w", err)
		}
	}
	i.tracesMtx.Unlock()

	maxBytes := i.limiter.Limits().MaxBytesPerTrace(i.instanceID)
	searchOpts := common.DefaultSearchOptionsWithMaxBytes(maxBytes)

	combiner := trace.NewCombiner(maxBytes, allowPartialTrace)
	_, err = combiner.Consume(completeTrace)
	if err != nil {
		return nil, err
	}

	// headBlock
	i.headBlockMtx.RLock()
	tr, err := i.headBlock.FindTraceByID(ctx, id, searchOpts)
	i.headBlockMtx.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("headBlock.FindTraceByID failed: %w", err)
	}
	if tr != nil {
		_, err = combiner.Consume(tr.Trace)
		if err != nil {
			return nil, err
		}
		metrics.InspectedBytes += tr.Metrics.InspectedBytes
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	// completingBlock
	for _, c := range i.completingBlocks {
		tr, err = c.FindTraceByID(ctx, id, searchOpts)
		if err != nil {
			return nil, fmt.Errorf("completingBlock.FindTraceByID failed: %w", err)
		}
		if tr == nil {
			continue
		}
		_, err = combiner.Consume(tr.Trace)
		if err != nil {
			return nil, err
		}
		if tr.Metrics != nil {
			metrics.InspectedBytes += tr.Metrics.InspectedBytes
		}
	}

	// completeBlock
	for _, c := range i.completeBlocks {
		found, err := c.FindTraceByID(ctx, id, searchOpts)
		if err != nil {
			return nil, fmt.Errorf("completeBlock.FindTraceByID failed: %w", err)
		}
		if found == nil {
			continue
		}
		_, err = combiner.Consume(found.Trace)
		if err != nil {
			return nil, err
		}
		if found.Metrics != nil {
			metrics.InspectedBytes += found.Metrics.InspectedBytes
		}
	}

	result, _ := combiner.Result()
	response := &tempopb.TraceByIDResponse{
		Trace:   result,
		Metrics: &metrics,
	}
	return response, nil
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
func (i *instance) getOrCreateTrace(traceID []byte, fp uint64) *liveTrace {
	trace, ok := i.traces[fp]
	if ok {
		return trace
	}

	trace = newTrace(traceID)
	i.traces[fp] = trace

	return trace
}

// resetHeadBlock() should be called under lock
func (i *instance) resetHeadBlock() error {
	dedicatedColumns := i.getDedicatedColumns()

	meta := &backend.BlockMeta{
		BlockID:          backend.NewUUID(),
		TenantID:         i.instanceID,
		DedicatedColumns: dedicatedColumns,
	}
	newHeadBlock, err := i.writer.WAL().NewBlock(meta, model.CurrentEncoding)
	if err != nil {
		return err
	}

	i.headBlock = newHeadBlock
	i.lastBlockCut = time.Now()

	return nil
}

func (i *instance) getDedicatedColumns() backend.DedicatedColumns {
	if cols := i.overrides.DedicatedColumns(i.instanceID); cols != nil {
		err := cols.Validate()
		if err != nil {
			level.Error(i.logger).Log("msg", "Unable to apply overrides for dedicated attribute columns. Columns invalid.", "error", err)
			return i.dedicatedColumns
		}

		return cols
	}
	return i.dedicatedColumns
}

func (i *instance) tracesToCut(now time.Time, idleCutoff time.Duration, liveCutoff time.Duration, immediate bool) []*liveTrace {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	// Set this before cutting to give a more accurate number.
	metricLiveTraces.WithLabelValues(i.instanceID).Set(float64(len(i.traces)))
	metricLiveTraceBytes.WithLabelValues(i.instanceID).Set(float64(i.traceSizeBytes))

	idleCutoffTime := now.Add(-idleCutoff)
	liveCutoffTime := now.Add(-liveCutoff)
	tracesToCut := make([]*liveTrace, 0, len(i.traces))

	for key, trace := range i.traces {
		if idleCutoffTime.After(trace.lastAppend) ||
			liveCutoffTime.After(trace.createdAt) ||
			immediate {

			tracesToCut = append(tracesToCut, trace)

			// decrease live trace bytes
			i.traceSizeBytes -= trace.Size()

			delete(i.traces, key)
		}
	}

	return tracesToCut
}

func (i *instance) writeTraceToHeadBlock(id common.ID, b []byte, start, end uint32) error {
	i.headBlockMtx.Lock()
	defer i.headBlockMtx.Unlock()

	i.tracesCreatedTotal.Inc()
	err := i.headBlock.Append(id, b, start, end, true)
	if err != nil {
		return err
	}

	return nil
}

func (i *instance) rediscoverLocalBlocks(ctx context.Context) ([]*LocalBlock, error) {
	ids, _, err := i.localReader.Blocks(ctx, i.instanceID)
	if err != nil {
		return nil, err
	}

	hasWal := func(id uuid.UUID) bool {
		i.blocksMtx.RLock()
		defer i.blocksMtx.RUnlock()
		for _, b := range i.completingBlocks {
			if (uuid.UUID)(b.BlockMeta().BlockID) == id {
				return true
			}
		}
		return false
	}

	var rediscoveredBlocks []*LocalBlock

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
			if errors.Is(err, backend.ErrDoesNotExist) {
				// Partial/incomplete block found, remove, it will be recreated from data in the wal.
				level.Warn(i.logger).Log("msg", "Unable to reload meta for local block. This indicates an incomplete block and will be deleted", "block", id.String())
				err = i.local.ClearBlock(id, i.instanceID)
				if err != nil {
					return nil, fmt.Errorf("deleting bad local block tenant %v block %v: %w", i.instanceID, id.String(), err)
				}
			} else {
				// Block with unknown error
				level.Error(i.logger).Log("msg", "Unexpected error reloading meta for local block. Ignoring and continuing. This block should be investigated.", "block", id.String(), "error", err)
				metricReplayErrorsTotal.WithLabelValues(i.instanceID).Inc()
			}

			continue
		}

		b, err := encoding.OpenBlock(meta, i.localReader)
		if err != nil {
			return nil, err
		}

		// validate the block before adding it to the list. if we drop a block here and its not in the wal this is data loss, but there is no way to recover. this is likely due to disk
		// level corruption
		err = b.Validate(ctx)
		if err != nil && !errors.Is(err, common.ErrUnsupported) {
			level.Error(i.logger).Log("msg", "local block failed validation, dropping", "block", id.String(), "error", err)
			metricReplayErrorsTotal.WithLabelValues(i.instanceID).Inc()

			err = i.local.ClearBlock(id, i.instanceID)
			if err != nil {
				return nil, fmt.Errorf("deleting invalid local block tenant %v block %v: %w", i.instanceID, id.String(), err)
			}

			continue
		}

		ib := NewLocalBlock(ctx, b, i.local)
		rediscoveredBlocks = append(rediscoveredBlocks, ib)

		level.Info(i.logger).Log("msg", "reloaded local block", "block", id.String(), "flushed", ib.FlushedTime())
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
