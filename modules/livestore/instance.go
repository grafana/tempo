package livestore

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
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
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const traceDataType = "trace"

// blocksSnapshot represents an immutable snapshot of all blocks
type blocksSnapshot struct {
	headBlock      common.WALBlock
	walBlocks      map[uuid.UUID]common.WALBlock
	completeBlocks map[uuid.UUID]*ingester.LocalBlock
}

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
)

type instance struct {
	tenantID string
	logger   log.Logger

	// Configuration
	Cfg Config

	// WAL and encoding
	wal *wal.WAL
	enc encoding.VersionedEncoding

	// Block management - using atomic pointer for lock-free reads
	blocksMtx      sync.Mutex                     // Regular mutex for write operations only
	blocksSnapshot atomic.Pointer[blocksSnapshot] // Atomic pointer to immutable snapshot
	lastCutTime    time.Time

	// Live traces
	liveTracesMtx sync.Mutex
	liveTraces    *livetraces.LiveTraces[*v1.ResourceSpans]
	traceSizes    *tracesizes.Tracker

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
		liveTraces:         livetraces.New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, cfg.MaxTraceLive, cfg.MaxTraceIdle),
		traceSizes:         tracesizes.New(),
		overrides:          overrides,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesReceivedTotal: metricBytesReceivedTotal,
		// blockOffsetMeta:   make(map[uuid.UUID]offsetMetadata),
	}

	// Initialize the blocks snapshot
	i.blocksSnapshot.Store(&blocksSnapshot{
		walBlocks:      make(map[uuid.UUID]common.WALBlock),
		completeBlocks: make(map[uuid.UUID]*ingester.LocalBlock),
	})

	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}

	return i, nil
}

// getBlocksSnapshot returns the current immutable snapshot of blocks - lock-free!
func (i *instance) getBlocksSnapshot() *blocksSnapshot {
	return i.blocksSnapshot.Load()
}

func (i *instance) pushBytes(ts time.Time, req *tempopb.PushBytesRequest) {
	if len(req.Traces) != len(req.Ids) {
		level.Error(i.logger).Log("msg", "mismatched traces and ids length", "IDs", len(req.Ids), "traces", len(req.Traces))
		return
	}

	// Check tenant limits
	maxBytes := i.overrides.MaxBytesPerTrace(i.tenantID)
	maxLiveTraces := i.overrides.MaxLocalTracesPerUser(i.tenantID)

	// For each pre-marshalled trace, we need to unmarshal it and push to live traces
	for j, traceBytes := range req.Traces {
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

		i.liveTracesMtx.Lock()
		// Push each batch in the trace to live traces
		for _, batch := range trace.ResourceSpans {
			if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
				continue
			}

			// Push to live traces with tenant-specific limits
			if err := i.liveTraces.PushWithTimestampAndLimits(ts, traceID, batch, uint64(maxLiveTraces), uint64(maxBytes)); err != nil {
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

// flushHeadBlock must be called under blocksMtx's lock
func (i *instance) flushHeadBlock() error {
	// Get current snapshot to access headBlock
	snapshot := i.blocksSnapshot.Load()
	if snapshot.headBlock != nil {
		err := snapshot.headBlock.Flush()
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (i *instance) cutIdleTraces(immediate bool) error {
	i.liveTracesMtx.Lock()
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

	return i.flushHeadBlock()
}

func (i *instance) writeHeadBlock(id []byte, liveTrace *livetraces.LiveTrace[*v1.ResourceSpans]) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	snapshot := i.blocksSnapshot.Load()
	if snapshot.headBlock == nil {
		err := i.resetHeadBlock()
		if err != nil {
			return err
		}
		snapshot = i.blocksSnapshot.Load()
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

	return snapshot.headBlock.AppendTrace(id, tr, startSeconds, endSeconds, false)
}

// resetHeadBlock must be called under blocksMtx's lock
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

	// Get current snapshot and create new snapshot with new headBlock
	oldSnapshot := i.blocksSnapshot.Load()
	newSnapshot := &blocksSnapshot{
		headBlock:      block,
		walBlocks:      oldSnapshot.walBlocks,      // Share immutable map reference
		completeBlocks: oldSnapshot.completeBlocks, // Share immutable map reference
	}
	i.blocksSnapshot.Store(newSnapshot)

	i.lastCutTime = time.Now()
	return nil
}

func (i *instance) cutBlocks(immediate bool) (uuid.UUID, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	// Get current snapshot
	snapshot := i.blocksSnapshot.Load()
	headBlock := snapshot.headBlock
	if headBlock == nil || headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	if !immediate && time.Since(i.lastCutTime) < i.Cfg.MaxBlockDuration && headBlock.DataLength() < i.Cfg.MaxBlockBytes {
		return uuid.Nil, nil
	}

	// Final flush
	err := headBlock.Flush()
	if err != nil {
		return uuid.Nil, err
	}

	id := (uuid.UUID)(headBlock.BlockMeta().BlockID)

	// Create new map with the head block moved to walBlocks
	newWalBlocks := make(map[uuid.UUID]common.WALBlock, len(snapshot.walBlocks)+1)
	for k, v := range snapshot.walBlocks {
		newWalBlocks[k] = v
	}
	newWalBlocks[id] = headBlock

	newSnapshot := &blocksSnapshot{
		headBlock:      nil, // Will be set by resetHeadBlock
		walBlocks:      newWalBlocks,
		completeBlocks: snapshot.completeBlocks,
	}
	i.blocksSnapshot.Store(newSnapshot)

	level.Info(i.logger).Log("msg", "queueing wal block for completion", "block", id.String())

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

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	snapshot := i.blocksSnapshot.Load()
	walBlock := snapshot.walBlocks[id]

	if walBlock == nil {
		level.Warn(i.logger).Log("msg", "WAL block disappeared before being completed", "id", id)
		span.AddEvent("WAL block not found")
		return nil
	}

	blockSize := walBlock.DataLength()
	metricCompletionSize.Observe(float64(blockSize))
	span.SetAttributes(attribute.Int64("block_size", int64(blockSize)))

	// Create completed block
	reader := backend.NewReader(i.wal.LocalBackend())
	writer := backend.NewWriter(i.wal.LocalBackend())

	iter, err := walBlock.Iterator()
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to get WAL block iterator", "id", id, "err", err)
		span.RecordError(err)
		return err
	}
	defer iter.Close()

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

	// Create new maps with the completed block and without the WAL block
	newWalBlocks := make(map[uuid.UUID]common.WALBlock, len(snapshot.walBlocks))
	for k, v := range snapshot.walBlocks {
		if k != id {
			newWalBlocks[k] = v
		}
	}

	newCompleteBlocks := make(map[uuid.UUID]*ingester.LocalBlock, len(snapshot.completeBlocks)+1)
	for k, v := range snapshot.completeBlocks {
		newCompleteBlocks[k] = v
	}
	newCompleteBlocks[id] = ingester.NewLocalBlock(ctx, newBlock, i.wal.LocalBackend())

	newSnapshot := &blocksSnapshot{
		headBlock:      snapshot.headBlock,
		walBlocks:      newWalBlocks,
		completeBlocks: newCompleteBlocks,
	}
	i.blocksSnapshot.Store(newSnapshot)

	err = walBlock.Clear()
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to clear WAL block", "id", id, "err", err)
		span.RecordError(err)
	}

	level.Info(i.logger).Log("msg", "completed block", "id", id.String())
	span.AddEvent("block completed successfully")
	return nil
}

func (i *instance) deleteOldBlocks() error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	cutoff := time.Now().Add(-i.Cfg.CompleteBlockTimeout) // Delete blocks older than Complete Block Timeout
	snapshot := i.blocksSnapshot.Load()

	needsUpdate := false
	newWalBlocks := make(map[uuid.UUID]common.WALBlock, len(snapshot.walBlocks))
	newCompleteBlocks := make(map[uuid.UUID]*ingester.LocalBlock, len(snapshot.completeBlocks))

	// Copy current blocks, excluding old ones
	for id, walBlock := range snapshot.walBlocks {
		if walBlock.BlockMeta().EndTime.Before(cutoff) {
			if _, ok := snapshot.completeBlocks[id]; !ok {
				level.Warn(i.logger).Log("msg", "deleting WAL block that was never completed", "block", id.String())
			}
			err := walBlock.Clear()
			if err != nil {
				return err
			}
			metricBlocksClearedTotal.WithLabelValues("wal").Inc()
			needsUpdate = true
		} else {
			newWalBlocks[id] = walBlock
		}
	}

	for id, completeBlock := range snapshot.completeBlocks {
		if completeBlock.BlockMeta().EndTime.Before(cutoff) {
			level.Info(i.logger).Log("msg", "deleting complete block", "block", id.String())
			err := i.wal.LocalBackend().ClearBlock(id, i.tenantID)
			if err != nil {
				return err
			}
			metricBlocksClearedTotal.WithLabelValues("complete").Inc()
			needsUpdate = true
		} else {
			newCompleteBlocks[id] = completeBlock
		}
	}

	if needsUpdate {
		newSnapshot := &blocksSnapshot{
			headBlock:      snapshot.headBlock,
			walBlocks:      newWalBlocks,
			completeBlocks: newCompleteBlocks,
		}
		i.blocksSnapshot.Store(newSnapshot)
	}

	return nil
}
