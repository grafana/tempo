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
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	buffererServiceName = "bufferer"
)

type Overrides interface {
	MaxLocalTracesPerUser(userID string) int
	MaxBytesPerTrace(userID string) int
	DedicatedColumns(userID string) backend.DedicatedColumns
}

var metricCompleteQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "bufferer",
	Name:      "complete_queue_length",
	Help:      "Number of wal blocks waiting for completion",
})

type Bufferer struct {
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

	flushqueues *flushqueues.PriorityQueue

	// Background processing
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
}

func New(cfg Config, overrides Overrides, logger log.Logger, reg prometheus.Registerer, singlePartition bool) (*Bufferer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	b := &Bufferer{
		cfg:         cfg,
		logger:      logger,
		reg:         reg,
		decoder:     ingest.NewDecoder(),
		ctx:         ctx,
		cancel:      cancel,
		instances:   make(map[string]*instance),
		overrides:   overrides,
		flushqueues: flushqueues.NewPriorityQueue(metricCompleteQueueLength),
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

	// TODO: It's probably easier to just use the ID directly
	//  https://raintank-corp.slack.com/archives/C05CAA0ULUF/p1752847274420489
	b.cfg.IngestConfig.Kafka.ConsumerGroup, err = ingest.BuffererConsumerGroupID(cfg.LifecyclerConfig.ID)
	if err != nil {
		return nil, fmt.Errorf("calculating ingester consumer group ID: %w", err)
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

func (b *Bufferer) starting(ctx context.Context) error {
	var err error
	b.wal, err = wal.New(&b.cfg.WAL)
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}

	err = services.StartAndAwaitRunning(ctx, b.ingestPartitionLifecycler)
	if err != nil {
		return fmt.Errorf("failed to start partition lifecycler: %w", err)
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

	b.reader, err = NewPartitionReaderForPusher(b.client, b.ingestPartitionID, b.cfg.IngestConfig.Kafka, b.consume, b.logger, b.reg)
	if err != nil {
		return fmt.Errorf("failed to create partition reader: %w", err)
	}
	err = services.StartAndAwaitRunning(ctx, b.reader)
	if err != nil {
		return fmt.Errorf("failed to start partition reader: %w", err)
	}

	b.wg.Add(1)
	go b.completeLoop()

	return nil
}

func (b *Bufferer) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-b.subservicesWatcher.Chan():
		return fmt.Errorf("bufferer subservice failed: %w", err)
	}
}

func (b *Bufferer) stopping(error) error {
	// Stop consuming
	err := services.StopAndAwaitTerminated(b.ctx, b.reader)
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to stop reader", "err", err)
		return err
	}

	// Cancel and wait for async loops to return
	b.cancel()
	b.wg.Wait()

	// Flush all data to disk
	b.flushRemaining()

	return nil
}

func (b *Bufferer) flushRemaining() {
	b.cutAllInstancesToWal()
	for b.flushqueues.Length() > 0 {
		time.Sleep(100 * time.Millisecond)
	}
}

func (b *Bufferer) consume(_ context.Context, rs []record) error {
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

		// Push data to tenant instance
		inst.pushBytes(time.Now(), pushReq, record.offset)
	}

	return nil
}

func (b *Bufferer) getOrCreateInstance(tenantID string) (*instance, error) {
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

	b.wg.Add(2)
	go b.cutToWalLoop(inst)
	go b.cleanupLoop(inst)

	return inst, nil
}

func (b *Bufferer) cutToWalLoop(instance *instance) {

	defer b.wg.Done()

	// TODO: We don't reply blocks atm, we should do this when we replay blocks.
	// // wait for the signal to start. we need the wal to be completely replayed
	// // before we start cutting to WAL
	// select {
	// case <-i.cutToWalStart:
	// case <-i.cutToWalStop:
	// 	return
	// }

	// ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.cutOneInstanceToWal(instance, false)
		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Bufferer) cleanupLoop(inst *instance) {
	defer b.wg.Done()

	// ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// dump any blocks that have been flushed for a while
			err := inst.deleteOldBlocks()
			if err != nil {
				level.Error(b.logger).Log("msg", "failed to delete old blocks", "err", err)
			}
		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Bufferer) getInstances() []*instance {
	b.instancesMtx.RLock()
	defer b.instancesMtx.RUnlock()
	instances := make([]*instance, 0, len(b.instances))
	for _, inst := range b.instances {
		instances = append(instances, inst)
	}
	return instances
}

// cutAllInstancesToWal periodically schedules series for flushing and garbage collects instances with no series
func (b *Bufferer) cutAllInstancesToWal() {
	instances := b.getInstances()

	for _, instance := range instances {
		b.cutOneInstanceToWal(instance, true)
	}
}

func (b *Bufferer) cutOneInstanceToWal(inst *instance, immedate bool) {
	// Regular trace cuts (live traces -> head block)
	err := inst.cutIdleTraces(immedate)
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to cut idle traces", "tenant", inst.tenantID, "err", err)
	}

	// Regular block cuts
	blockID, err := inst.cutBlocks(immedate)
	if err != nil {
		level.Error(b.logger).Log("msg", "failed to cut blocks", "tenant", inst.tenantID, "err", err)
	}

	// If head block is cut, enqueue complete operation
	if blockID != uuid.Nil {
		err = b.enqueueCompleteOp(inst.tenantID, blockID)
		if err != nil {
			return
		}
	}
}

func (b *Bufferer) enqueueCompleteOp(tenantID string, blockID uuid.UUID) error {
	op := &completeOp{
		tenantID: tenantID,
		blockID:  blockID,
		// Initial priority and backoff
		at: time.Now(),
		bo: 30 * time.Second,
	}
	_, err := b.flushqueues.Enqueue(op)
	return err
}

func (b *Bufferer) completeLoop() {
	defer b.wg.Done()

	// TODO: Add concurrency
	for {
		select {
		case <-b.ctx.Done():
			b.flushqueues.Close()
			return
		default:
			o := b.flushqueues.Dequeue()
			if o == nil {
				return // queue is closed
			}
			op := o.(*completeOp)
			op.attempts++

			if op.attempts > maxFlushAttempts {
				level.Error(b.logger).Log("msg", "failed to complete operation", "tenant", op.tenantID, "block", op.blockID, "attemps", op.attempts)
				continue
			}

			inst, err := b.getOrCreateInstance(op.tenantID)
			if err != nil {
				// TODO: How to handle?
				level.Error(b.logger).Log("msg", "failed to retrieve instance for completion", "tenant", op.tenantID, "err", err)
				return
			}
			err = inst.completeBlock(op.blockID)
			if err != nil {
				level.Error(b.logger).Log("msg", "failed to complete block", "tenant", op.tenantID, "block", op.blockID, "err", err)

				delay := op.backoff()
				op.at = time.Now().Add(delay)

				go func() {
					time.Sleep(delay)

					if _, err := b.flushqueues.Enqueue(op); err != nil {
						_ = level.Error(b.logger).Log("msg", "failed to requeue block for flushing", "tenant", op.tenantID, "block", op.blockID, "err", err)
					}
				}()

			}
		}
	}
}
