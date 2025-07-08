package bufferer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/tracesizes"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	ConsumerGroup       = "bufferer"
	buffererServiceName = "bufferer"
)

type Buffer struct {
	services.Service

	cfg    Config
	logger log.Logger
	reg    prometheus.Registerer

	subservicesWatcher *services.FailureWatcher

	ingestPartitionID         int32
	ingestPartitionLifecycler *ring.PartitionInstanceLifecycler

	client      *kgo.Client
	adminClient *kadm.Client
	decoder     *ingest.Decoder

	reader *PartitionReader

	// Data processing components
	wal    *wal.WAL
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
	enc    encoding.VersionedEncoding

	// Block management
	blocksMtx      sync.RWMutex
	headBlock      common.WALBlock
	walBlocks      map[uuid.UUID]common.WALBlock
	completeBlocks map[uuid.UUID]*ingester.LocalBlock
	lastCutTime    time.Time

	// Live traces
	liveTracesMtx sync.Mutex
	liveTraces    *livetraces.LiveTraces[*v1.ResourceSpans]
	traceSizes    *tracesizes.Tracker

	// Hard cut tracking
	blockStartOffset   int64
	totalBlockBytes    int64
	lastHardCutTime    time.Time
	hardCutInterval    time.Duration
	hardCutDataSize    int64
	hardCutRequested   atomic.Bool // Signal that a hard cut is needed
	lastKafkaOffset    atomic.Int64 // Last processed Kafka offset
}

func New(cfg Config, logger log.Logger, reg prometheus.Registerer, singlePartition bool) (*Buffer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	b := &Buffer{
		cfg:            cfg,
		logger:         logger,
		reg:            reg,
		decoder:        ingest.NewDecoder(),
		ctx:            ctx,
		cancel:         cancel,
		enc:            encoding.DefaultEncoding(), // TODO(mapno): configurable?
		walBlocks:      map[uuid.UUID]common.WALBlock{},
		completeBlocks: map[uuid.UUID]*ingester.LocalBlock{},
		liveTraces:     livetraces.New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, 30*time.Second, 5*time.Minute),
		traceSizes:     tracesizes.New(),

		// Hard cut configuration - 5 minutes or 100MB
		hardCutInterval: 5 * time.Minute,
		hardCutDataSize: 100 * 1024 * 1024, // 100MB
		lastHardCutTime: time.Now(),
	}

	var err error
	if singlePartition {
		// For single-binary don't require hostname to identify a partition.
		// Assume partition 0.
		b.ingestPartitionID = 0
	} else {
		b.ingestPartitionID, err = ingest.IngesterPartitionID(cfg.LifecyclerConfig.ID)
		if err != nil {
			return nil, fmt.Errorf("calculating ingester partition ID: %w", err)
		}
	}

	partitionRingKV := cfg.PartitionRing.KVStore.Mock
	if partitionRingKV == nil {
		partitionRingKV, err = kv.NewClient(cfg.PartitionRing.KVStore, ring.GetPartitionRingCodec(), kv.RegistererWithKVName(reg, ingester.PartitionRingName+"-lifecycler"), logger)
		if err != nil {
			return nil, fmt.Errorf("creating KV store for ingester partition ring: %w", err)
		}
	}

	b.ingestPartitionLifecycler = ring.NewPartitionInstanceLifecycler(
		b.cfg.PartitionRing.ToLifecyclerConfig(b.ingestPartitionID, cfg.LifecyclerConfig.ID),
		ingester.PartitionRingName,
		ingester.PartitionRingKey,
		partitionRingKV,
		logger,
		prometheus.WrapRegistererWithPrefix("tempo_", reg))

	b.subservicesWatcher = services.NewFailureWatcher()
	b.subservicesWatcher.WatchService(b.ingestPartitionLifecycler)

	b.Service = services.NewBasicService(b.starting, b.running, b.stopping)

	return b, nil
}

func (b *Buffer) starting(ctx context.Context) error {
	var err error
	b.wal, err = wal.New(&b.cfg.WAL)
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}

	b.client, err = ingest.NewReaderClient(
		b.cfg.IngestConfig.Kafka,
		ingest.NewReaderClientMetrics(buffererServiceName, b.reg),
		b.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create kafka reader client: %w", err)
	}

	boff := backoff.New(ctx, backoff.Config{
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: time.Minute, // If there is a network hiccup, we prefer to wait longer retrying, than fail the service.
		MaxRetries: 10,
	})

	for boff.Ongoing() {
		err := b.client.Ping(ctx)
		if err == nil {
			break
		}
		level.Warn(b.logger).Log("msg", "ping kafka; will retry", "err", err)
		boff.Wait()
	}
	if err := boff.ErrCause(); err != nil {
		return fmt.Errorf("failed to ping kafka: %w", err)
	}

	b.reader, err = NewPartitionReaderForPusher(b.client, b.ingestPartitionID, b.cfg.IngestConfig.Kafka.Topic, b.consume, b.logger, b.reg)
	if err != nil {
		return fmt.Errorf("failed to create partition reader: %w", err)
	}

	if err := b.reader.start(ctx); err != nil {
		return fmt.Errorf("failed to start partition reader: %w", err)
	}

	// Start background loops
	b.wg.Add(2)
	go b.cutLoop()
	go b.deleteLoop()

	return nil
}

func (b *Buffer) running(ctx context.Context) error {
	go func() {
		_ = b.reader.running(ctx)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-b.subservicesWatcher.Chan():
		return fmt.Errorf("bufferer subservice failed: %w", err)
	}
}

func (b *Buffer) stopping(err error) error {
	b.cancel()
	b.wg.Wait()

	// Cut all remaining traces
	if cutErr := b.cutIdleTraces(true); cutErr != nil {
		level.Error(b.logger).Log("msg", "failed to cut remaining traces on shutdown", "err", cutErr)
	}

	// Cut head block
	if cutErr := b.cutBlocks(true); cutErr != nil {
		level.Error(b.logger).Log("msg", "failed to cut head block on shutdown", "err", cutErr)
	}

	return b.reader.stop(err)
}

func (b *Buffer) consume(_ context.Context, rs []record, minOffset, maxOffset, totalBytes int64) error {
	// Initialize block offset tracking if this is the first batch
	if b.blockStartOffset == 0 {
		b.blockStartOffset = minOffset
	}

	// Update tracking (non-blocking)
	b.totalBlockBytes += totalBytes
	b.lastKafkaOffset.Store(maxOffset)

	// Process records
	for _, record := range rs {
		pushReq, err := b.decoder.Decode(record.content)
		if err != nil {
			return fmt.Errorf("decoding record: %w", err)
		}

		b.pushBytes(time.Now(), pushReq, record.tenantID)
	}

	// Check if we should signal a hard cut (non-blocking check)
	if b.shouldTriggerHardCut() {
		// Just set a flag - don't do the actual cutting here
		if b.hardCutRequested.CompareAndSwap(false, true) {
			level.Info(b.logger).Log("msg", "hard cut requested due to time/size limits",
				"start_offset", b.blockStartOffset, "end_offset", maxOffset, "bytes", b.totalBlockBytes)
		}
	}

	return nil
}

func (b *Buffer) shouldTriggerHardCut() bool {
	// Time-based cut
	if time.Since(b.lastHardCutTime) >= b.hardCutInterval {
		return true
	}

	// Size-based cut
	if b.totalBlockBytes >= b.hardCutDataSize {
		return true
	}

	return false
}

func (b *Buffer) pushBytes(ts time.Time, req *tempopb.PushBytesRequest, tenantID string) {
	// For each pre-marshalled trace, we need to unmarshal it and push to live traces
	for i, traceBytes := range req.Traces {
		if i >= len(req.Ids) {
			level.Warn(b.logger).Log("msg", "mismatched traces and ids length")
			break
		}

		traceID := req.Ids[i]

		// Unmarshal the trace
		trace := &tempopb.Trace{}
		if err := trace.Unmarshal(traceBytes.Slice); err != nil {
			level.Error(b.logger).Log("msg", "failed to unmarshal trace", "err", err)
			continue
		}

		b.liveTracesMtx.Lock()
		// Push each batch in the trace to live traces
		for _, batch := range trace.ResourceSpans {
			if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
				continue
			}

			// Push to live traces with basic limits
			if !b.liveTraces.PushWithTimestampAndLimits(ts, traceID, batch, 10000, 0) {
				level.Warn(b.logger).Log("msg", "dropped trace due to live traces limit", "tenant", tenantID)
				continue
			}
		}
		b.liveTracesMtx.Unlock()
	}
}

func (b *Buffer) cutLoop() {
	defer b.wg.Done()

	flushTicker := time.NewTicker(30 * time.Second) // Cut every 30 seconds
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			// Regular trace cuts (live traces -> head block)
			err := b.cutIdleTraces(false)
			if err != nil {
				level.Error(b.logger).Log("msg", "failed to cut idle traces", "err", err)
			}
			
			// Check for requested hard cuts (non-blocking)
			if b.hardCutRequested.Load() {
				b.performHardCut()
			}
			
			// Safety check: if we haven't had a hard cut in too long, force one
			// This prevents unbounded block growth if Kafka consumption stalls
			if time.Since(b.lastHardCutTime) > 2*b.hardCutInterval {
				level.Warn(b.logger).Log("msg", "forcing hard cut due to stale block", 
					"last_cut_ago", time.Since(b.lastHardCutTime))
				
				b.performHardCut()
			}

		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Buffer) performHardCut() {
	// Clear the request flag first to prevent duplicate cuts
	b.hardCutRequested.Store(false)
	
	endOffset := b.lastKafkaOffset.Load()
	
	level.Info(b.logger).Log("msg", "performing hard cut",
		"start_offset", b.blockStartOffset, "end_offset", endOffset, "bytes", b.totalBlockBytes)

	// Cut idle traces first
	err := b.cutIdleTraces(true) // Immediate cut
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to cut idle traces during hard cut", "err", err)
	}

	// Then cut blocks
	err = b.cutBlocks(true) // Immediate cut
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to cut blocks during hard cut", "err", err)
	} else {
		// Reset tracking for next block only on successful cut
		b.blockStartOffset = endOffset + 1
		b.totalBlockBytes = 0
		b.lastHardCutTime = time.Now()
	}
}

func (b *Buffer) deleteLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := b.deleteOldBlocks()
			if err != nil {
				level.Error(b.logger).Log("msg", "failed to delete old blocks", "err", err)
			}

		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Buffer) cutIdleTraces(immediate bool) error {
	b.liveTracesMtx.Lock()
	tracesToCut := b.liveTraces.CutIdle(time.Now(), immediate)
	b.liveTracesMtx.Unlock()

	if len(tracesToCut) == 0 {
		return nil
	}

	for _, t := range tracesToCut {
		tr := &tempopb.Trace{
			ResourceSpans: t.Batches,
		}

		err := b.writeHeadBlock(t.ID, tr)
		if err != nil {
			return err
		}
	}

	b.blocksMtx.Lock()
	defer b.blocksMtx.Unlock()
	if b.headBlock != nil {
		return b.headBlock.Flush()
	}
	return nil
}

func (b *Buffer) writeHeadBlock(id []byte, tr *tempopb.Trace) error {
	b.blocksMtx.Lock()
	defer b.blocksMtx.Unlock()

	if b.headBlock == nil {
		err := b.resetHeadBlock()
		if err != nil {
			return err
		}
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

	return b.headBlock.AppendTrace(id, tr, startSeconds, endSeconds, false)
}

func (b *Buffer) resetHeadBlock() error {
	meta := &backend.BlockMeta{
		BlockID:           backend.NewUUID(),
		TenantID:          "default", // Use default tenant for now
		ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
		// Store offset range in custom metadata (this is a simplified approach)
		// In a real implementation, you might want to extend the BlockMeta or use a different mechanism
	}
	block, err := b.wal.NewBlock(meta, "v2") // Use v2 encoding
	if err != nil {
		return err
	}
	b.headBlock = block
	b.lastCutTime = time.Now()
	return nil
}

func (b *Buffer) cutBlocks(immediate bool) error {
	b.blocksMtx.Lock()
	defer b.blocksMtx.Unlock()

	if b.headBlock == nil || b.headBlock.DataLength() == 0 {
		return nil
	}

	maxBlockDuration := 5 * time.Minute
	maxBlockBytes := uint64(100 * 1024 * 1024) // 100MB

	if !immediate && time.Since(b.lastCutTime) < maxBlockDuration && b.headBlock.DataLength() < maxBlockBytes {
		return nil
	}

	// Final flush
	err := b.headBlock.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush head block: %w", err)
	}

	id := (uuid.UUID)(b.headBlock.BlockMeta().BlockID)
	b.walBlocks[id] = b.headBlock

	// Log the offset range for this block
	level.Info(b.logger).Log("msg", "queueing wal block for completion",
		"block", id.String(),
		"start_offset", b.blockStartOffset,
		"bytes", b.totalBlockBytes)

	err = b.resetHeadBlock()
	if err != nil {
		return fmt.Errorf("failed to resetHeadBlock: %w", err)
	}

	go b.completeBlock(id)

	return nil
}

func (b *Buffer) completeBlock(id uuid.UUID) {
	b.blocksMtx.RLock()
	walBlock := b.walBlocks[id]
	b.blocksMtx.RUnlock()

	if walBlock == nil {
		level.Warn(b.logger).Log("msg", "WAL block disappeared before being completed", "id", id)
		return
	}

	// Create completed block
	reader := backend.NewReader(b.wal.LocalBackend())
	writer := backend.NewWriter(b.wal.LocalBackend())

	iter, err := walBlock.Iterator()
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to get WAL block iterator", "id", id, "err", err)
		return
	}
	defer iter.Close()

	newMeta, err := b.enc.CreateBlock(b.ctx, &common.BlockConfig{}, walBlock.BlockMeta(), iter, reader, writer)
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to create complete block", "id", id, "err", err)
		return
	}

	newBlock, err := b.enc.OpenBlock(newMeta, reader)
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to open complete block", "id", id, "err", err)
		return
	}

	b.blocksMtx.Lock()
	// Verify the WAL block still exists
	if _, ok := b.walBlocks[id]; !ok {
		level.Warn(b.logger).Log("msg", "WAL block disappeared while being completed, deleting complete block", "id", id)
		err := b.wal.LocalBackend().ClearBlock(id, "default")
		if err != nil {
			level.Error(b.logger).Log("msg", "failed to clear complete block after WAL disappeared", "block", id, "err", err)
		}
		b.blocksMtx.Unlock()
		return
	}

	b.completeBlocks[id] = ingester.NewLocalBlock(b.ctx, newBlock, b.wal.LocalBackend())

	err = walBlock.Clear()
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to clear WAL block", "id", id, "err", err)
	}
	delete(b.walBlocks, (uuid.UUID)(walBlock.BlockMeta().BlockID))
	b.blocksMtx.Unlock()

	level.Info(b.logger).Log("msg", "completed block", "id", id.String())
}

func (b *Buffer) deleteOldBlocks() error {
	b.blocksMtx.Lock()
	defer b.blocksMtx.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour) // Delete blocks older than 1 hour

	for id, walBlock := range b.walBlocks {
		if walBlock.BlockMeta().EndTime.Before(cutoff) {
			if _, ok := b.completeBlocks[id]; !ok {
				level.Warn(b.logger).Log("msg", "deleting WAL block that was never completed", "block", id.String())
			}
			err := walBlock.Clear()
			if err != nil {
				return err
			}
			delete(b.walBlocks, id)
		}
	}

	for id, completeBlock := range b.completeBlocks {
		if completeBlock.BlockMeta().EndTime.Before(cutoff) {
			level.Info(b.logger).Log("msg", "deleting complete block", "block", id.String())
			err := b.wal.LocalBackend().ClearBlock(id, "default")
			if err != nil {
				return err
			}
			delete(b.completeBlocks, id)
		}
	}

	return nil
}
