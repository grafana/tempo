package bufferer

import (
	"context"
	"fmt"
	"sync"
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
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	ConsumerGroup       = "bufferer"
	buffererServiceName = "bufferer"
)

type Overrides interface {
	MaxLocalTracesPerUser(userID string) int
	MaxBytesPerTrace(userID string) int
	DedicatedColumns(userID string) backend.DedicatedColumns
}

type Buffer struct {
	services.Service

	cfg    Config
	logger log.Logger
	reg    prometheus.Registerer

	subservicesWatcher *services.FailureWatcher

	ingestPartitionID         int32
	ingestPartitionLifecycler *ring.PartitionInstanceLifecycler

	client  *kgo.Client
	decoder *ingest.Decoder

	reader *PartitionReader

	// Multi-tenant instances
	instancesMtx sync.RWMutex
	instances    map[string]*instance
	wal          *wal.WAL
	overrides    Overrides

	// Global offset tracking and cut coordination
	blockStartOffset            int64 // TODO: Should be fetched from replay
	totalBlockBytesSinceLastCut int64
	lastCutTime                 time.Time
	cutInterval                 time.Duration
	cutDataSize                 int64
	lastKafkaOffset             int64

	// Background processing
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
}

func New(cfg Config, overrides Overrides, logger log.Logger, reg prometheus.Registerer, singlePartition bool) (*Buffer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	b := &Buffer{
		cfg:       cfg,
		logger:    logger,
		reg:       reg,
		decoder:   ingest.NewDecoder(),
		ctx:       ctx,
		cancel:    cancel,
		instances: make(map[string]*instance),
		overrides: overrides,

		// Cut configuration - 5 minutes or 100MB
		cutInterval: 5 * time.Minute,
		cutDataSize: 100 * 1024 * 1024, // 100MB
		lastCutTime: time.Now(),
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

	// Start background loops for completing blocks
	b.wg.Add(3)
	go b.cutLoop()
	go b.deleteLoop()
	go b.completeLoop()

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

	// Cut all remaining traces and blocks for all tenants
	b.instancesMtx.RLock()
	for tenantID, inst := range b.instances {
		if cutErr := inst.cutIdleTraces(true); cutErr != nil {
			level.Error(b.logger).Log("msg", "failed to cut remaining traces on shutdown", "tenant", tenantID, "err", cutErr)
		}
		if _, cutErr := inst.cutBlocks(true); cutErr != nil {
			level.Error(b.logger).Log("msg", "failed to cut head block on shutdown", "tenant", tenantID, "err", cutErr)
		}
	}
	b.instancesMtx.RUnlock()

	return b.reader.stop(err)
}

func (b *Buffer) completeLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Complete blocks for all tenants
			b.instancesMtx.RLock()
			for _, inst := range b.instances {
				// Look for WAL blocks that need completion
				inst.blocksMtx.RLock()
				// TODO: Concurrency? Maybe only signal the instance but continue going?
				for id := range inst.walBlocks {
					err := inst.completeBlock(id)
					if err != nil { // TODO: What to do?
						level.Error(b.logger).Log("msg", "failed to complete block", "id", id, "err", err)
					}
				}
				inst.blocksMtx.RUnlock()
			}
			b.instancesMtx.RUnlock()

		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Buffer) consume(_ context.Context, rs []record, minOffset, maxOffset, totalBytes int64) error {
	// Update global offset tracking
	if b.blockStartOffset == 0 {
		b.blockStartOffset = minOffset
	}
	b.totalBlockBytesSinceLastCut += totalBytes
	b.lastKafkaOffset = maxOffset

	// Process records by tenant
	for _, record := range rs {
		var pushReq *tempopb.PushBytesRequest
		pushReq, err := b.decoder.Decode(record.content)
		if err != nil {
			return fmt.Errorf("decoding record: %w", err)
		}

		// Get or create tenant instance
		inst, err := b.getOrCreateInstance(record.tenantID)
		if err != nil {
			level.Error(b.logger).Log("msg", "failed to get instance for tenant", "tenant", record.tenantID, "err", err)
			continue
		}

		// Push data to tenant instance (no offset tracking at instance level)
		inst.pushBytes(time.Now(), pushReq)
	}

	// Check if we should trigger a coordinated cut (global decision)
	if b.shouldTriggerCut() {
		level.Info(b.logger).Log("msg", "triggering coordinated cut",
			"start_offset", b.blockStartOffset, "end_offset", maxOffset, "bytes", b.totalBlockBytesSinceLastCut)
		b.performCut()
	}

	return nil
}

func (b *Buffer) getOrCreateInstance(tenantID string) (*instance, error) {
	b.instancesMtx.RLock()
	inst, ok := b.instances[tenantID]
	b.instancesMtx.RUnlock()

	if ok {
		return inst, nil
	}

	b.instancesMtx.Lock()
	defer b.instancesMtx.Unlock()

	// Double-check in case another goroutine created it
	if inst, ok := b.instances[tenantID]; ok {
		return inst, nil
	}

	// Create new instance
	inst, err := newInstance(tenantID, b.wal, b.overrides, b.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance for tenant %s: %w", tenantID, err)
	}

	b.instances[tenantID] = inst
	return inst, nil
}

func (b *Buffer) shouldTriggerCut() bool {
	// Time-based cut
	if time.Since(b.lastCutTime) >= b.cutInterval {
		return true
	}

	// Size-based cut
	if b.totalBlockBytesSinceLastCut >= b.cutDataSize {
		return true
	}

	return false
}

func (b *Buffer) performCut() {
	level.Info(b.logger).Log("msg", "performing coordinated cut",
		"start_offset", b.blockStartOffset, "end_offset", b.lastKafkaOffset, "bytes", b.totalBlockBytesSinceLastCut)

	// Cut all tenant instances at the same global offset
	b.instancesMtx.RLock()
	// TODO: Add concurrency? Only signal to instance?
	for tenantID, inst := range b.instances {
		// Cut idle traces first
		err := inst.cutIdleTraces(true) // Immediate cut
		if err != nil {
			level.Error(b.logger).Log("msg", "failed to cut idle traces during cut", "tenant", tenantID, "err", err)
		}

		// Then cut blocks - all tenants cut at the same global offset
		blockID, err := inst.cutBlocks(true) // Immediate cut
		if err != nil {
			level.Error(b.logger).Log("msg", "failed to cut blocks during cut", "tenant", tenantID, "err", err)
		} else if blockID != uuid.Nil {
			// Store offset metadata for this block (global offsets)
			inst.setBlockOffsetMetadata(blockID, b.blockStartOffset, b.lastKafkaOffset)
		}
	}
	b.instancesMtx.RUnlock()

	// Reset global tracking for next block
	b.blockStartOffset = b.lastKafkaOffset + 1
	b.totalBlockBytesSinceLastCut = 0
	b.lastCutTime = time.Now()
}

func (b *Buffer) cutLoop() {
	defer b.wg.Done()

	flushTicker := time.NewTicker(30 * time.Second) // Cut every 30 seconds
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			// Process all tenant instances for regular cuts
			b.instancesMtx.RLock()
			// TODO(mapno): Add concurrency or make it async for when we're handling a lot of tenants
			for tenantID, inst := range b.instances {
				// Regular trace cuts (live traces -> head block)
				err := inst.cutIdleTraces(false)
				if err != nil {
					level.Error(b.logger).Log("msg", "failed to cut idle traces", "tenant", tenantID, "err", err)
				}

				// Regular block cuts
				_, err = inst.cutBlocks(false)
				if err != nil {
					level.Error(b.logger).Log("msg", "failed to cut blocks", "tenant", tenantID, "err", err)
				}
			}
			b.instancesMtx.RUnlock()

		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Buffer) deleteLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Clean up old blocks for all tenants
			b.instancesMtx.RLock()
			for tenantID, inst := range b.instances {
				err := inst.deleteOldBlocks()
				if err != nil {
					level.Error(b.logger).Log("msg", "failed to delete old blocks", "tenant", tenantID, "err", err)
				}
			}
			b.instancesMtx.RUnlock()

		case <-b.ctx.Done():
			return
		}
	}
}
