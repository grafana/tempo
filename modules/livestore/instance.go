package livestore

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/tracesizes"
	util_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	traceDataType              = "trace"
	reasonWaitingForLiveTraces = "waiting_for_live_traces"
	reasonWaitingForWAL        = "waiting_for_wal"
	maxTraceLogLinesPerSecond  = 10
)

var (
	// Instance-level metrics (similar to ingester instance.go)
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricLiveTraces = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo_live_store",
		Name:      "live_traces",
		Help:      "The current number of live traces per tenant.",
	}, []string{"tenant"})
	metricLiveTraceBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo_live_store",
		Name:      "live_trace_bytes",
		Help:      "The current number of bytes consumed by live traces per tenant.",
	}, []string{"tenant"})
	metricBytesReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "bytes_received_total",
		Help:      "The total bytes received per tenant.",
	}, []string{"tenant", "data_type"})
	metricBlocksClearedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	}, []string{"block_type"})
	metricCompletionSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo_live_store",
		Name:      "completion_size_bytes",
		Help:      "Size in bytes of blocks completed.",
		Buckets:   prometheus.ExponentialBuckets(1024*1024, 2, 10), // from 1MB up to 1GB
	})
	metricBackPressure = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "back_pressure_seconds_total",
		Help:      "The total amount of time spent waiting to process data from queue",
	}, []string{"reason"})
	metricCompleteBlockCleanupFailures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "complete_block_cleanup_failures_total",
		Help:      "Total number of complete block cleanup failures after WAL clear errors",
	})
	metricOrphanedBlocksCleaned = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "orphaned_blocks_cleaned_total",
		Help:      "Total number of orphaned complete blocks cleaned up (startup or retry)",
	})
	metricWALBlockCleanupFailures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "wal_block_cleanup_failures_total",
		Help:      "Total number of WAL block cleanup failures during deleteOldBlocks",
	})
)

type instance struct {
	tenantID string
	logger   log.Logger

	// Configuration
	Cfg Config

	// WAL and encoding
	wal *wal.WAL
	enc encoding.VersionedEncoding

	// Block management
	blocksMtx        sync.RWMutex
	headBlock        common.WALBlock
	walBlocks        map[uuid.UUID]common.WALBlock
	completeBlocks   map[uuid.UUID]*ingester.LocalBlock
	completingBlocks map[uuid.UUID]bool // Tracks blocks currently being completed to prevent TOCTOU races
	lastCutTime      time.Time

	// Live traces
	liveTracesMtx  sync.Mutex
	liveTraces     *livetraces.LiveTraces[*v1.ResourceSpans]
	traceSizes     *tracesizes.Tracker
	maxTraceLogger *util_log.RateLimitedLogger

	// Metrics
	tracesCreatedTotal prometheus.Counter
	bytesReceivedTotal *prometheus.CounterVec

	overrides overrides.Interface
}

func newInstance(instanceID string, cfg Config, wal *wal.WAL, overrides overrides.Interface, logger log.Logger) (*instance, error) {
	enc := encoding.DefaultEncoding()
	logger = log.With(logger, "tenant", instanceID)

	i := &instance{
		tenantID:           instanceID,
		logger:             logger,
		Cfg:                cfg,
		wal:                wal,
		enc:                enc,
		walBlocks:          map[uuid.UUID]common.WALBlock{},
		completeBlocks:     map[uuid.UUID]*ingester.LocalBlock{},
		completingBlocks:   map[uuid.UUID]bool{}, // Track blocks being completed
		liveTraces:         livetraces.New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, cfg.MaxTraceIdle, cfg.MaxTraceLive, instanceID),
		traceSizes:         tracesizes.New(),
		maxTraceLogger:     util_log.NewRateLimitedLogger(maxTraceLogLinesPerSecond, level.Warn(logger)),
		overrides:          overrides,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesReceivedTotal: metricBytesReceivedTotal,
		// blockOffsetMeta:   make(map[uuid.UUID]offsetMetadata),
	}

	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (i *instance) backpressure(ctx context.Context) bool {
	if i.Cfg.MaxLiveTracesBytes > 0 {
		// Check live traces
		i.liveTracesMtx.Lock()
		// Guard against nil during shutdown
		var sz uint64
		if i.liveTraces != nil {
			sz = i.liveTraces.Size()
		}
		i.liveTracesMtx.Unlock()

		if sz >= i.Cfg.MaxLiveTracesBytes {
			// Live traces exceeds the expected amount of data in per wal flush,
			// so wait a bit.
			select {
			case <-ctx.Done():
				return false
			case <-time.After(1 * time.Second):
			}

			metricBackPressure.WithLabelValues(reasonWaitingForLiveTraces).Inc()
			return true
		}
	}

	// Check outstanding wal blocks
	i.blocksMtx.RLock()
	count := len(i.walBlocks)
	i.blocksMtx.RUnlock()

	if count > 1 {
		// There are multiple outstanding WAL blocks that need completion
		// so wait a bit.
		select {
		case <-ctx.Done():
			return false
		case <-time.After(1 * time.Second):
		}

		metricBackPressure.WithLabelValues(reasonWaitingForWAL).Inc()
		return true
	}

	return false
}

func (i *instance) pushBytes(ctx context.Context, ts time.Time, req *tempopb.PushBytesRequest) {
	if len(req.Traces) != len(req.Ids) {
		level.Error(i.logger).Log("msg", "mismatched traces and ids length", "IDs", len(req.Ids), "traces", len(req.Traces))
		return
	}

	// Wait for room in pipeline if needed
	for i.backpressure(ctx) {
	}

	if err := ctx.Err(); err != nil {
		// Fixed C-7: Record dropped traces when context is cancelled
		level.Error(i.logger).Log("msg", "failed to push bytes to instance due to context cancellation",
			"err", err, "dropped_traces", len(req.Traces))
		// Estimate 10 spans per trace since we can't unmarshal during shutdown
		overrides.RecordDiscardedSpans(len(req.Traces)*10, "context_cancelled", i.tenantID)
		return
	}

	// Check tenant limits
	maxBytes := i.overrides.MaxBytesPerTrace(i.tenantID)
	maxLiveTraces := i.overrides.MaxLocalTracesPerUser(i.tenantID)

	// For each pre-marshalled trace, we need to unmarshal it and push to live traces
	for j, traceBytes := range req.Traces {
		// Fixed C-16: Check context before processing each trace to enable fast shutdown
		if err := ctx.Err(); err != nil {
			remainingTraces := len(req.Traces) - j
			level.Info(i.logger).Log("msg", "batch processing cancelled by context",
				"processed", j, "remaining", remainingTraces)
			overrides.RecordDiscardedSpans(remainingTraces*10, "context_cancelled", i.tenantID)
			return
		}

		traceID := req.Ids[j]
		// measure received bytes as sum of slice lengths
		// type byte is guaranteed to be 1 byte in size
		// ref: https://golang.org/ref/spec#Size_and_alignment_guarantees
		i.bytesReceivedTotal.WithLabelValues(i.tenantID, traceDataType).Add(float64(len(traceBytes.Slice)))

		// Unmarshal the trace
		trace := &tempopb.Trace{}
		if err := trace.Unmarshal(traceBytes.Slice); err != nil {
			level.Error(i.logger).Log("msg", "failed to unmarshal trace", "err", err)
			continue
		}

		// Reuse the byte slice now that we've unmarshalled it
		tempopb.ReuseByteSlices([][]byte{traceBytes.Slice})

		// test max trace size. use trace sizes over liveTraces b/c it tracks large traces across multiple flushes
		if maxBytes > 0 {
			traceSz := trace.Size()
			allowResult := i.traceSizes.Allow(traceID, traceSz, maxBytes)
			if !allowResult.IsAllowed {
				i.maxTraceLogger.Log("msg", overrides.ErrorPrefixTraceTooLarge, "max", maxBytes, "traceSz", traceSz, "totalSize", allowResult.CurrentTotalSize, "trace", hex.EncodeToString(traceID), "insight", true)
				overrides.RecordDiscardedSpans(countSpans(trace), overrides.ReasonTraceTooLarge, i.tenantID)
				continue
			}
		}

		// Push each batch in the trace to live traces with fine-grained locking
		// Lock is acquired/released per batch to reduce contention
		for _, batch := range trace.ResourceSpans {
			if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
				continue
			}

			// Acquire lock for this single batch only
			i.liveTracesMtx.Lock()
			// Guard against nil during shutdown
			if i.liveTraces == nil {
				i.liveTracesMtx.Unlock()
				// During shutdown, drop this batch but continue with others
				level.Warn(i.logger).Log("msg", "dropping batch during shutdown", "traces", len(batch.ScopeSpans))
				continue // Continue processing remaining batches
			}
			err := i.liveTraces.PushWithTimestampAndLimits(ts, traceID, batch, uint64(maxLiveTraces), 0)
			i.liveTracesMtx.Unlock()

			if err != nil {
				var reason string
				switch {
				case errors.Is(err, livetraces.ErrMaxLiveTracesExceeded):
					reason = overrides.ReasonLiveTracesExceeded
				case errors.Is(err, livetraces.ErrMaxTraceSizeExceeded):
					reason = overrides.ReasonTraceTooLarge
				default:
					reason = overrides.ReasonUnknown
				}
				level.Debug(i.logger).Log("msg", "dropped spans due to limits", "tenant", i.tenantID, "reason", reason)
				overrides.RecordDiscardedSpans(countSpans(trace), reason, i.tenantID)
				continue
			}
		}
	}
}

func countSpans(trace *tempopb.Trace) int {
	count := 0
	for _, b := range trace.ResourceSpans {
		for _, ss := range b.ScopeSpans {
			count += len(ss.Spans)
		}
	}
	return count
}

func (i *instance) cutIdleTraces(immediate bool) error {
	i.liveTracesMtx.Lock()

	// Guard against nil during shutdown
	if i.liveTraces == nil {
		i.liveTracesMtx.Unlock()
		return nil
	}

	// Set metrics before cutting (similar to ingester)
	metricLiveTraces.WithLabelValues(i.tenantID).Set(float64(i.liveTraces.Len()))
	metricLiveTraceBytes.WithLabelValues(i.tenantID).Set(float64(i.liveTraces.Size()))

	tracesToCut := i.liveTraces.CutIdle(time.Now(), immediate)
	i.liveTracesMtx.Unlock()

	if len(tracesToCut) == 0 {
		return nil
	}
	// Sort by ID
	sort.Slice(tracesToCut, func(i, j int) bool {
		return bytes.Compare(tracesToCut[i].ID, tracesToCut[j].ID) == -1
	})
	// Collect the trace IDs that will be flushed
	for _, t := range tracesToCut {
		err := i.writeHeadBlock(t.ID, t)
		if err != nil {
			return err
		}

		i.tracesCreatedTotal.Inc()
	}

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()
	if i.headBlock != nil {
		err := i.headBlock.Flush()
		if err != nil {
			return err
		}

		return nil
	}
	return nil
}

func (i *instance) writeHeadBlock(id []byte, liveTrace *livetraces.LiveTrace[*v1.ResourceSpans]) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil {
		err := i.resetHeadBlock()
		if err != nil {
			return err
		}
	}

	tr := &tempopb.Trace{
		ResourceSpans: liveTrace.Batches,
	}

	// Get trace timestamp bounds
	var start, end uint64
	for _, batch := range tr.ResourceSpans {
		for _, ss := range batch.ScopeSpans {
			for _, s := range ss.Spans {
				if start == 0 || s.StartTimeUnixNano < start {
					start = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > end {
					end = s.EndTimeUnixNano
				}
			}
		}
	}

	// Convert from unix nanos to unix seconds
	startSeconds := uint32(start / uint64(time.Second))
	endSeconds := uint32(end / uint64(time.Second))

	// constrain start/end with ingestion slack calculated off of liveTrace.createdAt and lastAppend
	// createdAt and lastAppend are set via the record.Timestamp from kafka so they are "time.Now()" for the
	// ingestion of this trace
	slackDuration := i.Cfg.WAL.IngestionSlack
	minStart := uint32(liveTrace.CreatedAt.Add(-slackDuration).Unix())
	maxEnd := uint32(liveTrace.LastAppend.Add(slackDuration).Unix())

	if startSeconds < minStart {
		startSeconds = minStart
	}
	if endSeconds > maxEnd {
		endSeconds = maxEnd
	}

	return i.headBlock.AppendTrace(id, tr, startSeconds, endSeconds, false)
}

func (i *instance) resetHeadBlock() error {
	dedicatedColumns := i.overrides.DedicatedColumns(i.tenantID)

	meta := &backend.BlockMeta{
		BlockID:           backend.NewUUID(),
		TenantID:          i.tenantID,
		DedicatedColumns:  dedicatedColumns,
		ReplicationFactor: backend.LiveStoreReplicationFactor,
	}
	block, err := i.wal.NewBlock(meta, model.CurrentEncoding)
	if err != nil {
		return err
	}
	i.headBlock = block
	i.lastCutTime = time.Now()
	return nil
}

func (i *instance) cutBlocks(immediate bool) (uuid.UUID, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	if !immediate && time.Since(i.lastCutTime) < i.Cfg.MaxBlockDuration && i.headBlock.DataLength() < i.Cfg.MaxBlockBytes {
		return uuid.Nil, nil
	}

	i.traceSizes.ClearIdle(i.lastCutTime)

	// Final flush
	err := i.headBlock.Flush()
	if err != nil {
		return uuid.Nil, err
	}

	id := (uuid.UUID)(i.headBlock.BlockMeta().BlockID)
	blockSize := i.headBlock.DataLength()
	i.walBlocks[id] = i.headBlock

	level.Info(i.logger).Log("msg", "queueing wal block for completion", "block", id.String(), "size", blockSize)

	err = i.resetHeadBlock()
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (i *instance) completeBlock(ctx context.Context, id uuid.UUID) error {
	ctx, span := tracer.Start(ctx, "instance.completeBlock",
		oteltrace.WithAttributes(
			attribute.String("tenant", i.tenantID),
			attribute.String("blockID", id.String()),
		))
	defer span.End()

	// STEP 1: Check and mark as in-progress (TOCTOU prevention)
	i.blocksMtx.Lock()
	walBlock := i.walBlocks[id]
	if walBlock == nil {
		i.blocksMtx.Unlock()
		level.Warn(i.logger).Log("msg", "WAL block disappeared before being completed", "id", id)
		span.AddEvent("WAL block not found")
		return nil
	}

	// Check if already being completed
	if i.completingBlocks[id] {
		i.blocksMtx.Unlock()
		err := fmt.Errorf("block already being completed")
		span.RecordError(err)
		return err
	}

	// Mark as in-progress
	i.completingBlocks[id] = true
	i.blocksMtx.Unlock()

	// Ensure cleanup on exit (even if function panics)
	defer func() {
		i.blocksMtx.Lock()
		delete(i.completingBlocks, id)
		i.blocksMtx.Unlock()
	}()

	// STEP 2: Perform I/O without lock (safe - marked in-progress)
	blockSize := walBlock.DataLength()
	metricCompletionSize.Observe(float64(blockSize))
	span.SetAttributes(attribute.Int64("block_size", int64(blockSize)))

	// Create completed block
	localBackend := i.wal.LocalBackend()
	if localBackend == nil {
		err := fmt.Errorf("WAL local backend not available for block %s", id)
		level.Error(i.logger).Log("msg", "local backend unavailable", "id", id, "err", err)
		span.RecordError(err)
		return err
	}
	reader := backend.NewReader(localBackend)
	writer := backend.NewWriter(localBackend)

	iter, err := walBlock.Iterator()
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to get WAL block iterator", "id", id, "err", err)
		span.RecordError(err)
		return err
	}
	defer iter.Close()

	// Check if complete block already exists from previous failed attempt
	_, metaErr := reader.BlockMeta(ctx, id, i.tenantID)
	if metaErr == nil {
		// Block exists - clear before retry
		level.Warn(i.logger).Log("msg", "found existing complete block from failed attempt, clearing before retry",
			"id", id, "tenant", i.tenantID)

		clearErr := localBackend.ClearBlock(id, i.tenantID) // I/O without lock
		if clearErr != nil {
			level.Error(i.logger).Log("msg", "cannot retry - failed to clear existing complete block",
				"id", id, "tenant", i.tenantID, "err", clearErr)
			metricCompleteBlockCleanupFailures.Inc()
			return fmt.Errorf("cannot retry completion - failed to clear existing complete block: %w", clearErr)
		}

		level.Info(i.logger).Log("msg", "cleared orphaned complete block, proceeding with retry",
			"id", id, "tenant", i.tenantID)
		metricOrphanedBlocksCleaned.Inc()
	}

	// Create new complete block (I/O without lock)
	newMeta, err := i.enc.CreateBlock(ctx, &i.Cfg.BlockConfig, walBlock.BlockMeta(), iter, reader, writer)
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to create complete block", "id", id, "err", err)
		span.RecordError(err)
		return err
	}

	newBlock, err := i.enc.OpenBlock(newMeta, reader)
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to open complete block", "id", id, "err", err)
		span.RecordError(err)
		return err
	}

	// Create LocalBlock wrapper (cheap operation, but do outside lock anyway)
	localBlock := ingester.NewLocalBlock(ctx, newBlock, localBackend)

	// STEP 3: Final state update with lock (double-check pattern)
	i.blocksMtx.Lock()

	// Verify WAL block still exists (could have been deleted during I/O)
	walBlock, ok := i.walBlocks[id]
	if !ok {
		i.blocksMtx.Unlock()
		// Block disappeared during completion - clean up orphan
		level.Warn(i.logger).Log("msg", "WAL block disappeared while being completed, deleting complete block", "id", id)
		err := localBackend.ClearBlock(id, i.tenantID) // I/O without lock
		if err != nil {
			level.Error(i.logger).Log("msg", "failed to clear complete block after WAL disappeared", "block", id, "err", err)
		}
		span.AddEvent("WAL block disappeared during completion")
		return nil
	}

	// Add to complete blocks and remove from WAL blocks atomically
	// This prevents the dual-map window that would cause duplicate search results
	i.completeBlocks[id] = localBlock
	delete(i.walBlocks, id)
	i.blocksMtx.Unlock()

	// STEP 4: Clear WAL block (I/O outside lock)
	err = walBlock.Clear()
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to clear WAL block", "id", id, "err", err)
		span.RecordError(err)

		// Clean up orphaned complete block files
		cleanupErr := localBackend.ClearBlock(id, i.tenantID) // I/O without lock
		if cleanupErr != nil {
			// CRITICAL: Both WAL clear and complete block clear failed
			// Do NOT remove from maps - files still exist and need retry
			level.Error(i.logger).Log("msg", "both WAL clear and complete block clear failed",
				"id", id, "wal_clear_err", err, "complete_clear_err", cleanupErr)
			metricCompleteBlockCleanupFailures.Inc()

			// Remove from completeBlocks map but keep state dirty for retry
			i.blocksMtx.Lock()
			delete(i.completeBlocks, id)
			i.blocksMtx.Unlock()

			return fmt.Errorf("both WAL clear and complete block clear failed for %s (will retry): wal=%w, complete=%v", id, err, cleanupErr)
		}

		// Complete block clear succeeded, so only WAL clear failed
		// Remove from completeBlocks map (walBlocks already removed above)
		i.blocksMtx.Lock()
		delete(i.completeBlocks, id)
		i.blocksMtx.Unlock()

		return fmt.Errorf("failed to clear WAL block %s: %w", id, err)
	}

	// STEP 5: Final cleanup - completingBlocks map (walBlocks already removed atomically above)
	// Note: No need to remove from walBlocks here - already done atomically with completeBlocks add

	level.Info(i.logger).Log("msg", "completed block", "id", id.String())
	span.AddEvent("block completed successfully")
	return nil
}

func (i *instance) deleteOldBlocks() error {
	cutoff := time.Now().Add(-i.Cfg.CompleteBlockTimeout) // Delete blocks older than Complete Block Timeout
	cleanupErrors := 0

	// Phase 1: Collect blocks to delete (fast, under lock)
	i.blocksMtx.Lock()
	walBlocksToDelete := make([]uuid.UUID, 0)
	completeBlocksToDelete := make([]uuid.UUID, 0)

	for id, walBlock := range i.walBlocks {
		if walBlock.BlockMeta().EndTime.Before(cutoff) {
			// Skip blocks that are currently being completed
			if i.completingBlocks[id] {
				level.Debug(i.logger).Log("msg", "skipping WAL block deletion, completion in progress",
					"block", id.String())
				continue
			}

			if _, ok := i.completeBlocks[id]; !ok {
				level.Warn(i.logger).Log("msg", "deleting old WAL block (may have had completion issues)",
					"block", id.String(), "tenant", i.tenantID)
			}

			walBlocksToDelete = append(walBlocksToDelete, id)
		}
	}

	for id, completeBlock := range i.completeBlocks {
		if completeBlock.BlockMeta().EndTime.Before(cutoff) {
			// Skip blocks that are currently being completed
			// This shouldn't happen in normal operation, but protects against edge cases
			if i.completingBlocks[id] {
				level.Debug(i.logger).Log("msg", "skipping complete block deletion, completion in progress",
					"block", id.String())
				continue
			}

			completeBlocksToDelete = append(completeBlocksToDelete, id)
		}
	}
	i.blocksMtx.Unlock()

	// Phase 2 & 3: Delete WAL block files (slow, NO lock) then remove from map (fast, under lock)
	for _, id := range walBlocksToDelete {
		// Get block reference under read lock
		i.blocksMtx.RLock()
		walBlock, exists := i.walBlocks[id]
		i.blocksMtx.RUnlock()

		if !exists {
			continue // Block already deleted
		}

		// Clear block file (OUTSIDE lock)
		err := walBlock.Clear()
		if err != nil {
			if errors.Is(err, backend.ErrDoesNotExist) {
				// Block already deleted - this is OK
				level.Debug(i.logger).Log("msg", "WAL block already deleted", "id", id)
				// Still remove from map
				i.blocksMtx.Lock()
				delete(i.walBlocks, id)
				i.blocksMtx.Unlock()
				continue
			}
			// Real error - log and continue
			level.Error(i.logger).Log("msg", "failed to clear old WAL block, skipping",
				"id", id, "tenant", i.tenantID, "err", err)
			metricWALBlockCleanupFailures.Inc()
			cleanupErrors++

			// Circuit breaker: if too many failures, something is seriously wrong
			if cleanupErrors > 10 {
				return fmt.Errorf("too many WAL block deletion failures (%d), possible disk issue", cleanupErrors)
			}
			continue // Skip this block but continue with others
		}

		// Remove from map (fast, under lock)
		i.blocksMtx.Lock()
		delete(i.walBlocks, id)
		i.blocksMtx.Unlock()

		metricBlocksClearedTotal.WithLabelValues("wal").Inc()
	}

	// Phase 2 & 3: Delete complete block files (slow, NO lock) then remove from map (fast, under lock)
	for _, id := range completeBlocksToDelete {
		level.Info(i.logger).Log("msg", "deleting complete block", "block", id.String())

		// Clear block file (OUTSIDE lock)
		err := i.wal.LocalBackend().ClearBlock(id, i.tenantID)
		if err != nil {
			if errors.Is(err, backend.ErrDoesNotExist) {
				// Block already deleted - this is OK
				level.Debug(i.logger).Log("msg", "complete block already deleted", "id", id)
				// Still remove from map
				i.blocksMtx.Lock()
				delete(i.completeBlocks, id)
				i.blocksMtx.Unlock()
				continue
			}
			// Real error - log and continue
			level.Error(i.logger).Log("msg", "failed to clear old complete block, skipping",
				"id", id, "tenant", i.tenantID, "err", err)
			metricWALBlockCleanupFailures.Inc()
			cleanupErrors++

			// Circuit breaker: if too many failures, something is seriously wrong
			if cleanupErrors > 10 {
				return fmt.Errorf("too many complete block deletion failures (%d), possible disk issue", cleanupErrors)
			}
			continue // Skip this block but continue with others
		}

		// Remove from map (fast, under lock)
		i.blocksMtx.Lock()
		delete(i.completeBlocks, id)
		i.blocksMtx.Unlock()

		metricBlocksClearedTotal.WithLabelValues("complete").Inc()
	}

	// Log summary if any blocks failed
	if cleanupErrors > 0 {
		level.Warn(i.logger).Log("msg", "some blocks failed to clean up",
			"failed_count", cleanupErrors,
			"total_attempted", len(walBlocksToDelete)+len(completeBlocksToDelete),
			"tenant", i.tenantID)
	}

	return nil
}
