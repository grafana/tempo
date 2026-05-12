package livestore

import (
	"context"
	"encoding/hex"
	"errors"
	"iter"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/tracesizes"
	util_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	traceDataType              = "trace"
	reasonWaitingForLiveTraces = "waiting_for_live_traces"
	reasonWaitingForWAL        = "waiting_for_wal"
	maxTraceLogLinesPerSecond  = 10
	// walBackpressureLimit is the maximum number of outstanding WAL blocks before
	// backpressure is applied. In the ideal case, shutdown can leave up to 2
	// uncompleted WAL blocks on disk, and after restart ingestion may outpace WAL
	// completion, so we use 4 to avoid unnecessary backpressure during catch-up.
	walBackpressureLimit = 4
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
		Namespace:                       "tempo_live_store",
		Name:                            "completion_size_bytes",
		Help:                            "Size in bytes of blocks completed.",
		Buckets:                         prometheus.ExponentialBuckets(1024*1024, 2, 10), // from 1MB up to 512MB
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
	metricBlocksCutTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "blocks_cut_total",
		Help:      "The total number of blocks cut by reason.",
	}, []string{"reason"})
	metricBackPressure = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "back_pressure_seconds_total",
		Help:      "The total amount of time spent waiting to process data from queue",
	}, []string{"reason"})
	metricTotalBackPressure = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Subsystem:                       "live_store",
		Name:                            "back_pressure_duration_seconds",
		Help:                            "Duration of backpressure wait per push",
		Buckets:                         prometheus.DefBuckets,
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
)

type instance struct {
	tenantID string
	logger   log.Logger

	// Configuration
	Cfg Config

	// WAL and encoding
	wal                    *wal.WAL
	completeBlockEncoding  encoding.VersionedEncoding
	completeBlockLifecycle completeBlockLifecycle

	// blocksMtx serializes writers. Readers iterate the immutable snapshot
	// via atomic.Load; for the headBlock's mutating BlockMeta they must use
	// WALBlock.MetaSnapshot.
	blocksMtx   sync.Mutex
	blocks      atomic.Pointer[blockSnapshot]
	lastCutTime time.Time

	// reclaim holds blocks removed from the snapshot whose files must
	// outlive in-flight readers; drained after the grace window.
	reclaim *quarantine

	// Live traces
	liveTracesMtx      sync.Mutex
	liveTraces         *livetraces.LiveTraces[*v1.ResourceSpans]
	traceSizes         *tracesizes.Tracker
	maxTraceLogger     *util_log.RateLimitedLogger
	liveTracesIterNext func() (*livetraces.LiveTrace[*v1.ResourceSpans], bool)
	liveTracesIterStop func()

	// Metrics
	tracesCreatedTotal prometheus.Counter
	bytesReceivedTotal *prometheus.CounterVec

	overrides overrides.Interface
}

func newInstance(instanceID string, cfg Config, wal *wal.WAL, completeBlockEncoding encoding.VersionedEncoding, completeBlockLifecycle completeBlockLifecycle, overrides overrides.Interface, logger log.Logger) (*instance, error) {
	logger = log.With(logger, "tenant", instanceID)
	if completeBlockLifecycle == nil {
		var err error
		completeBlockLifecycle, err = newCompleteBlockLifecycle(cfg, nil, logger)
		if err != nil {
			return nil, err
		}
	}
	i := &instance{
		tenantID:               instanceID,
		logger:                 logger,
		Cfg:                    cfg,
		wal:                    wal,
		completeBlockEncoding:  completeBlockEncoding,
		completeBlockLifecycle: completeBlockLifecycle,
		liveTraces:             livetraces.New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, cfg.MaxTraceIdle, cfg.MaxTraceLive, instanceID),
		traceSizes:             tracesizes.New(),
		maxTraceLogger:         util_log.NewRateLimitedLogger(maxTraceLogLinesPerSecond, level.Warn(logger)),
		overrides:              overrides,
		tracesCreatedTotal:     metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesReceivedTotal:     metricBytesReceivedTotal,
		reclaim:                newQuarantine(cfg.BlockReclaimGrace),
	}
	i.blocks.Store(emptyBlockSnapshot())

	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (i *instance) waitBackpressure(ctx context.Context) {
	span := oteltrace.SpanFromContext(ctx)
	start := time.Now()
	var hadBackpressure bool
	for i.backpressure(ctx) {
		hadBackpressure = true
	}
	if !hadBackpressure {
		return // no need to measure backpressure
	}
	duration := time.Since(start)
	span.AddEvent("backpressure done", oteltrace.WithAttributes(
		attribute.Int("duration_ms", int(duration.Milliseconds())),
	))
	metricTotalBackPressure.Observe(duration.Seconds())
}

func (i *instance) backpressure(ctx context.Context) bool {
	span := oteltrace.SpanFromContext(ctx)

	if i.Cfg.MaxLiveTracesBytes > 0 {
		// Check live traces

		i.liveTracesMtx.Lock()
		sz := i.liveTraces.Size()
		i.liveTracesMtx.Unlock()

		if sz >= i.Cfg.MaxLiveTracesBytes {
			// Live traces exceeds the expected amount of data in per wal flush,
			// so wait a bit.
			select {
			case <-ctx.Done():
				return false
			case <-time.After(1 * time.Second):
			}

			span.AddEvent("backpressure", oteltrace.WithAttributes(
				attribute.String("reason", reasonWaitingForLiveTraces),
			))
			metricBackPressure.WithLabelValues(reasonWaitingForLiveTraces).Inc()
			return true
		}
	}

	// Check outstanding wal blocks
	count := len(i.blocks.Load().walBlocks)

	if count > walBackpressureLimit {
		// There are multiple outstanding WAL blocks that need completion
		// so wait a bit.
		select {
		case <-ctx.Done():
			return false
		case <-time.After(1 * time.Second):
		}

		metricBackPressure.WithLabelValues(reasonWaitingForWAL).Inc()
		span.AddEvent("backpressure", oteltrace.WithAttributes(
			attribute.String("reason", reasonWaitingForWAL),
		))
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
	i.waitBackpressure(ctx)

	if err := ctx.Err(); err != nil {
		level.Error(i.logger).Log("msg", "failed to push bytes to instance", "err", err)
		return
	}

	// Check tenant limits
	maxBytes := i.overrides.MaxBytesPerTrace(i.tenantID)
	maxLiveTraces := i.overrides.MaxLocalTracesPerUser(i.tenantID)

	// Reuse a single Trace across iterations to preserve slice capacity.
	// The ResourceSpans pointers are handed off to liveTraces, but the
	// Trace wrapper and its backing array can be recycled.
	trace := &tempopb.Trace{}

	// For each pre-marshalled trace, we need to unmarshal it and push to live traces
	for j, traceBytes := range req.Traces {
		traceID := req.Ids[j]
		// measure received bytes as sum of slice lengths
		// type byte is guaranteed to be 1 byte in size
		// ref: https://golang.org/ref/spec#Size_and_alignment_guarantees
		i.bytesReceivedTotal.WithLabelValues(i.tenantID, traceDataType).Add(float64(len(traceBytes.Slice)))

		// Capture proto size before returning bytes to pool.
		traceSz := len(traceBytes.Slice)

		// Clear stale pointers so prior iterations' ResourceSpans can be
		// GC'd, then truncate so Unmarshal appends into the existing backing array.
		clear(trace.ResourceSpans)
		trace.ResourceSpans = trace.ResourceSpans[:0]
		if err := trace.Unmarshal(traceBytes.Slice); err != nil {
			level.Error(i.logger).Log("msg", "failed to unmarshal trace", "err", err)
			continue
		}

		// Reuse the byte slice now that we've unmarshalled it
		tempopb.ReuseByteSlices([][]byte{traceBytes.Slice})

		// test max trace size. use trace sizes over liveTraces b/c it tracks large traces across multiple flushes
		if maxBytes > 0 {
			allowResult := i.traceSizes.Allow(traceID, traceSz, maxBytes)
			if !allowResult.IsAllowed {
				i.maxTraceLogger.Log("msg", overrides.ErrorPrefixTraceTooLarge, "max", maxBytes, "traceSz", traceSz, "totalSize", allowResult.CurrentTotalSize, "trace", hex.EncodeToString(traceID), "insight", true)
				overrides.RecordDiscardedSpans(countSpans(trace), overrides.ReasonTraceTooLarge, i.tenantID)
				continue
			}
		}

		i.liveTracesMtx.Lock()
		// Push each batch in the trace to live traces
		for _, batch := range trace.ResourceSpans {
			if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
				continue
			}

			// Push to live traces with tenant-specific limits
			if err := i.liveTraces.PushWithTimestampAndLimits(ts, traceID, batch, uint64(maxLiveTraces), 0); err != nil {
				var reason string
				switch {
				case errors.Is(err, livetraces.ErrMaxLiveTracesExceeded):
					reason = overrides.ReasonLiveTracesExceeded
				case errors.Is(err, livetraces.ErrMaxTraceSizeExceeded): // this should technically never happen b/c we are passing 0 as max trace sz
					reason = overrides.ReasonTraceTooLarge
				default:
					reason = overrides.ReasonUnknown
				}
				level.Debug(i.logger).Log("msg", "dropped spans due to limits", "tenant", i.tenantID, "reason", reason)
				overrides.RecordDiscardedSpans(countSpans(trace), reason, i.tenantID)
				continue
			}
		}
		i.liveTracesMtx.Unlock()
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

func (i *instance) cutIdleTraces(ctx context.Context, immediate bool) (bool, error) {
	_, span := tracer.Start(ctx, "instance.cutIdleTraces",
		oteltrace.WithAttributes(attribute.String("tenant", i.tenantID)))
	defer span.End()

	i.liveTracesMtx.Lock()
	span.AddEvent("acquired liveTracesMtx")

	// Set metrics before cutting (similar to ingester)
	metricLiveTraces.WithLabelValues(i.tenantID).Set(float64(i.liveTraces.Len()))
	metricLiveTraceBytes.WithLabelValues(i.tenantID).Set(float64(i.liveTraces.Size()))
	if i.liveTracesIterNext == nil {
		i.liveTracesIterNext, i.liveTracesIterStop = iter.Pull(i.liveTraces.CutIdle(time.Now(), immediate))
	}

	i.liveTracesMtx.Unlock()
	span.AddEvent("released liveTracesMtx")

	var tracesCut int
	defer func() { span.SetAttributes(attribute.Int("traces_cut", tracesCut)) }()

	// Write traces to head block, cutting when MaxBlockBytes is reached.
	span.AddEvent("writing traces to head block")
	for {
		t, ok := i.liveTracesIterNext()
		if !ok {
			break
		}
		blockSize, err := i.writeHeadBlock(t.ID, t)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			i.liveTracesIterStop()
			i.liveTracesIterNext, i.liveTracesIterStop = nil, nil
			return false, err
		}

		tracesCut++
		i.tracesCreatedTotal.Inc()

		// if the head block has reached max block bytes,
		// we exit earlier in order to cut the block
		if blockSize >= i.Cfg.MaxBlockBytes {
			return false, nil
		}
	}

	i.liveTracesIterStop()
	i.liveTracesIterNext, i.liveTracesIterStop = nil, nil

	if tracesCut == 0 { // no traces to process
		return true, nil
	}

	span.AddEvent("wrote traces to head block")

	i.blocksMtx.Lock()
	span.AddEvent("acquired blocksMtx")
	defer func() {
		i.blocksMtx.Unlock()
		span.AddEvent("released blocksMtx")
	}()
	if hb := i.blocks.Load().headBlock; hb != nil {
		err := hb.Flush()
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return false, err
		}
	}
	return true, nil
}

func (i *instance) writeHeadBlock(id []byte, liveTrace *livetraces.LiveTrace[*v1.ResourceSpans]) (uint64, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	hb := i.blocks.Load().headBlock
	if hb == nil {
		if err := i.resetHeadBlock(); err != nil {
			return 0, err
		}
		hb = i.blocks.Load().headBlock
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

	err := hb.AppendTrace(id, tr, startSeconds, endSeconds, false)
	return hb.DataLength(), err
}

func (i *instance) getDedicatedColumns() backend.DedicatedColumns {
	if cols := i.overrides.DedicatedColumns(i.tenantID); cols != nil {
		_, err := cols.Validate()
		if err != nil {
			level.Error(i.logger).Log("msg", "Unable to apply overrides for dedicated attribute columns. Columns invalid.", "error", err)
			return i.Cfg.BlockConfig.DedicatedColumns
		}
		return cols
	}
	return i.Cfg.BlockConfig.DedicatedColumns
}

func (i *instance) resetHeadBlock() error {
	dedicatedColumns := i.getDedicatedColumns()

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
	i.blocks.Store(i.blocks.Load().withHeadBlock(block))
	i.lastCutTime = time.Now()
	return nil
}

type cutReason int

const (
	cutReasonNone             cutReason = 0
	cutReasonImmediate        cutReason = 1 << iota
	cutReasonMaxBlockDuration cutReason = 1 << iota
	cutReasonMaxBlockBytes    cutReason = 1 << iota
)

// shouldCutHead checks whether the head block should be cut and returns the
// reason(s). Caller must hold blocksMtx.
func (i *instance) shouldCutHead(immediate bool) cutReason {
	hb := i.blocks.Load().headBlock
	if hb == nil || hb.DataLength() == 0 {
		return cutReasonNone
	}

	var reason cutReason

	if immediate {
		reason |= cutReasonImmediate
	}
	if time.Since(i.lastCutTime) >= i.Cfg.MaxBlockDuration {
		reason |= cutReasonMaxBlockDuration
	}
	if hb.DataLength() >= i.Cfg.MaxBlockBytes {
		reason |= cutReasonMaxBlockBytes
	}

	return reason
}

func recordBlockCutMetric(reason cutReason) {
	if reason&cutReasonImmediate != 0 {
		metricBlocksCutTotal.WithLabelValues("immediate").Inc()
	}
	if reason&cutReasonMaxBlockDuration != 0 {
		metricBlocksCutTotal.WithLabelValues("max_block_duration").Inc()
	}
	if reason&cutReasonMaxBlockBytes != 0 {
		metricBlocksCutTotal.WithLabelValues("max_block_bytes").Inc()
	}
}

func (i *instance) cutBlocks(ctx context.Context, immediate bool) (uuid.UUID, error) {
	_, span := tracer.Start(ctx, "instance.cutBlocks",
		oteltrace.WithAttributes(attribute.String("tenant", i.tenantID)))
	defer span.End()

	i.blocksMtx.Lock()
	span.AddEvent("acquired blocksMtx")
	defer func() {
		i.blocksMtx.Unlock()
		span.AddEvent("released blocksMtx")
	}()

	reason := i.shouldCutHead(immediate)
	if reason == cutReasonNone {
		return uuid.Nil, nil
	}

	i.traceSizes.ClearIdle(i.lastCutTime)

	snap := i.blocks.Load()
	hb := snap.headBlock

	// Final flush
	if err := hb.Flush(); err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return uuid.Nil, err
	}

	id := (uuid.UUID)(hb.BlockMeta().BlockID)
	blockSize := hb.DataLength()

	// Move headBlock into walBlocks and clear head, in one published snapshot.
	i.blocks.Store(snap.withWALBlockAdded(id, hb).withHeadBlock(nil))

	span.SetAttributes(
		attribute.String("blockID", id.String()),
		attribute.Int64("block_size", int64(blockSize)),
	)

	level.Info(i.logger).Log("msg", "queueing wal block for completion", "block", id.String(), "size", blockSize)

	if err := i.resetHeadBlock(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return uuid.Nil, err
	}

	recordBlockCutMetric(reason)

	return id, nil
}

func (i *instance) completeBlock(ctx context.Context, id uuid.UUID) (*LocalBlock, error) {
	ctx, span := tracer.Start(ctx, "instance.completeBlock",
		oteltrace.WithAttributes(
			attribute.String("tenant", i.tenantID),
			attribute.String("blockID", id.String()),
		))
	defer span.End()

	walBlock := i.blocks.Load().walBlocks[id]

	if walBlock == nil {
		level.Warn(i.logger).Log("msg", "WAL block disappeared before being completed", "id", id)
		span.AddEvent("WAL block not found")
		return nil, nil
	}

	blockSize := walBlock.DataLength()

	level.Info(i.logger).Log("msg", "completing WAL block", "blockSize", blockSize, "blockId", id.String())
	metricCompletionSize.Observe(float64(blockSize))
	span.SetAttributes(attribute.Int64("wal_block_size", int64(blockSize)))

	// Create completed block
	reader := backend.NewReader(i.wal.LocalBackend())
	writer := backend.NewWriter(i.wal.LocalBackend())

	iter, err := walBlock.Iterator(ctx)
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to get WAL block iterator", "id", id, "err", err)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, err
	}
	defer iter.Close()

	span.AddEvent("creating block")
	newMeta, err := i.completeBlockEncoding.CreateBlock(ctx, &i.Cfg.BlockConfig, walBlock.BlockMeta(), iter, reader, writer)
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to create complete block", "id", id, "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.AddEvent("created block")
	span.SetAttributes(
		attribute.Int64("block_size", int64(newMeta.Size_)),
		attribute.Int64("total_objects", newMeta.TotalObjects),
	)

	level.Info(i.logger).Log("msg", "opening newly completed block", "blockId", newMeta.BlockID.String())
	span.AddEvent("opening block")
	newBlock, err := i.completeBlockEncoding.OpenBlock(newMeta, reader)
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to open complete block", "id", id, "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.AddEvent("opened block")

	level.Info(i.logger).Log("msg", "swapping wal block with newly completed block")

	i.blocksMtx.Lock()
	span.AddEvent("acquired blocksMtx")
	defer func() {
		i.blocksMtx.Unlock()
		span.AddEvent("released blocksMtx")
	}()

	snap := i.blocks.Load()

	// Verify the WAL block still exists
	if _, ok := snap.walBlocks[id]; !ok {
		level.Warn(i.logger).Log("msg", "WAL block disappeared while being completed, deleting complete block", "id", id)
		err := i.wal.LocalBackend().ClearBlock(id, i.tenantID)
		if err != nil {
			level.Error(i.logger).Log("msg", "failed to clear complete block after WAL disappeared", "block", id, "err", err)
			span.RecordError(err)
		}
		span.AddEvent("WAL block disappeared during completion")
		return nil, nil
	}

	completeBlock := NewLocalBlock(ctx, newBlock, i.wal.LocalBackend())

	// Tombstone the WAL block before publishing the new snapshot. If we
	// crash between here and the deferred Clear, replay sees the
	// meta.deleted.json marker and reclaims the dir.
	if err := walBlock.Tombstone(); err != nil {
		level.Error(i.logger).Log("msg", "failed to tombstone WAL block", "id", id, "err", err)
		span.RecordError(err)
	}

	walID := (uuid.UUID)(walBlock.BlockMeta().BlockID)
	i.blocks.Store(snap.withCompleteBlockAdded(id, completeBlock).withWALBlockRemoved(walID))

	// Defer file deletion past the grace window so readers on the previous
	// snapshot don't hit ENOENT.
	wb := walBlock
	i.reclaim.add(walID, i.tenantID, "wal", func() error { return wb.Clear() })

	level.Info(i.logger).Log("msg", "completed block", "id", id.String())
	span.AddEvent("block completed successfully")
	return completeBlock, nil
}

func (i *instance) deleteOldBlocks() error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	cutoff := time.Now().Add(-i.Cfg.CompleteBlockTimeout) // Delete blocks older than Complete Block Timeout

	snap := i.blocks.Load()
	newSnap := snap
	defer func() {
		if newSnap != snap {
			i.blocks.Store(newSnap)
		}
	}()

	for id, walBlock := range snap.walBlocks {
		if walBlock.BlockMeta().EndTime.Before(cutoff) {
			if _, ok := snap.completeBlocks[id]; !ok {
				level.Warn(i.logger).Log("msg", "deleting WAL block that was never completed", "block", id.String())
			}
			if err := walBlock.Tombstone(); err != nil {
				return err
			}
			newSnap = newSnap.withWALBlockRemoved(id)
			wb := walBlock
			i.reclaim.add(id, i.tenantID, "wal", func() error { return wb.Clear() })
		}
	}

	for id, completeBlock := range snap.completeBlocks {
		if !i.completeBlockLifecycle.shouldDeleteCompleteBlock(completeBlock, cutoff) {
			continue
		}

		level.Info(i.logger).Log("msg", "deleting complete block", "block", id.String())
		if err := i.wal.LocalBackend().TombstoneBlock(id, i.tenantID); err != nil {
			return err
		}
		newSnap = newSnap.withCompleteBlockRemoved(id)
		bid := id
		tenant := i.tenantID
		bk := i.wal.LocalBackend()
		i.reclaim.add(bid, tenant, "complete", func() error { return bk.ClearBlock(bid, tenant) })
	}

	return nil
}
