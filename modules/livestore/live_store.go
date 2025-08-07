package livestore

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
)

const (
	liveStoreServiceName = "live-store"
)

type Overrides interface {
	MaxLocalTracesPerUser(userID string) int
	MaxBytesPerTrace(userID string) int
	DedicatedColumns(userID string) backend.DedicatedColumns
}

var metricCompleteQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "live_store",
	Name:      "complete_queue_length",
	Help:      "Number of wal blocks waiting for completion",
})

type LiveStore struct {
	services.Service

	cfg    Config
	logger log.Logger
	reg    prometheus.Registerer

	subservicesWatcher *services.FailureWatcher

	ingestPartitionID         int32
	ingestPartitionLifecycler *ring.PartitionInstanceLifecycler

	client        KafkaClient
	clientFactory KafkaClientFactory
	decoder       *ingest.Decoder

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

func New(cfg Config, overrides Overrides, logger log.Logger, reg prometheus.Registerer, singlePartition bool, clientFactory KafkaClientFactory) (*LiveStore, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &LiveStore{
		cfg:           cfg,
		logger:        logger,
		reg:           reg,
		decoder:       ingest.NewDecoder(),
		ctx:           ctx,
		cancel:        cancel,
		instances:     make(map[string]*instance),
		overrides:     overrides,
		flushqueues:   flushqueues.NewPriorityQueue(metricCompleteQueueLength),
		clientFactory: clientFactory,
	}

	var err error
	if singlePartition {
		// For single-binary don't require hostname to identify a partition.
		// Assume partition 0.
		s.ingestPartitionID = 0
	} else {
		s.ingestPartitionID, err = ingest.IngesterPartitionID(cfg.LifecyclerConfig.ID)
		if err != nil {
			return nil, fmt.Errorf("calculating ingester partition ID: %w", err)
		}
	}

	// TODO: It's probably easier to just use the ID directly
	//  https://raintank-corp.slack.com/archives/C05CAA0ULUF/p1752847274420489
	s.cfg.IngestConfig.Kafka.ConsumerGroup, err = ingest.LiveStoreConsumerGroupID(cfg.LifecyclerConfig.ID)
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

	s.ingestPartitionLifecycler = ring.NewPartitionInstanceLifecycler(
		s.cfg.PartitionRing.ToLifecyclerConfig(s.ingestPartitionID, cfg.LifecyclerConfig.ID),
		ingester.PartitionRingName,
		ingester.PartitionRingKey,
		partitionRingKV,
		logger,
		prometheus.WrapRegistererWithPrefix("tempo_", reg))

	s.subservicesWatcher = services.NewFailureWatcher()
	s.subservicesWatcher.WatchService(s.ingestPartitionLifecycler)

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)

	return s, nil
}

func (s *LiveStore) starting(ctx context.Context) error {
	var err error
	s.wal, err = wal.New(&s.cfg.WAL)
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}

	err = services.StartAndAwaitRunning(ctx, s.ingestPartitionLifecycler)
	if err != nil {
		return fmt.Errorf("failed to start partition lifecycler: %w", err)
	}

	s.client, err = s.clientFactory(
		s.cfg.IngestConfig.Kafka,
		ingest.NewReaderClientMetrics(liveStoreServiceName, s.reg),
		s.logger,
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
		err := s.client.Ping(ctx)
		if err == nil {
			break
		}
		level.Warn(s.logger).Log("msg", "ping kafka; will retry", "err", err)
		boff.Wait()
	}
	if err := boff.ErrCause(); err != nil {
		return fmt.Errorf("failed to ping kafka: %w", err)
	}

	s.reader, err = NewPartitionReaderForPusher(s.client, s.ingestPartitionID, s.cfg.IngestConfig.Kafka, s.consume, s.logger, s.reg)
	if err != nil {
		return fmt.Errorf("failed to create partition reader: %w", err)
	}
	err = services.StartAndAwaitRunning(ctx, s.reader)
	if err != nil {
		return fmt.Errorf("failed to start partition reader: %w", err)
	}

	s.wg.Add(1)
	go s.completeLoop()

	return nil
}

func (s *LiveStore) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-s.subservicesWatcher.Chan():
		return fmt.Errorf("live-store subservice failed: %w", err)
	}
}

func (s *LiveStore) stopping(error) error {
	// Stop consuming
	err := services.StopAndAwaitTerminated(context.Background(), s.reader)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to stop reader", "err", err)
		return err
	}

	// Cancel and wait for async loops to return
	s.cancel()
	s.flushqueues.Close()
	s.wg.Wait()

	// Flush all data to disk
	s.flushRemaining()

	return nil
}

func (s *LiveStore) flushRemaining() {
	s.cutAllInstancesToWal()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(30 * time.Second) // TODO: Configurable?
	defer timeout.Stop()

	for s.flushqueues.Length() > 0 {
		select {
		case <-ticker.C:
		case <-timeout.C:
			level.Error(s.logger).Log("msg", "flush remaining blocks timed out", "remaining", s.flushqueues.Length())
			return // shutdown timeout reached
		}
	}
}

func (s *LiveStore) consume(_ context.Context, rs []record) error {
	// Process records by tenant
	for _, record := range rs {
		var pushReq *tempopb.PushBytesRequest
		pushReq, err := s.decoder.Decode(record.content)
		if err != nil {
			return fmt.Errorf("decoding record: %w", err)
		}

		// Get or create tenant instance
		inst, err := s.getOrCreateInstance(record.tenantID)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to get instance for tenant", "tenant", record.tenantID, "err", err)
			continue
		}

		// Push data to tenant instance
		inst.pushBytes(time.Now(), pushReq)
	}

	return nil
}

func (s *LiveStore) getOrCreateInstance(tenantID string) (*instance, error) {
	s.instancesMtx.RLock()
	inst, ok := s.instances[tenantID]
	s.instancesMtx.RUnlock()

	if ok {
		return inst, nil
	}

	s.instancesMtx.Lock()
	defer s.instancesMtx.Unlock()

	// Double-check in case another goroutine created it
	if inst, ok := s.instances[tenantID]; ok {
		return inst, nil
	}

	// Create new instance
	inst, err := newInstance(tenantID, s.wal, s.overrides, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance for tenant %s: %w", tenantID, err)
	}

	s.instances[tenantID] = inst

	s.wg.Add(2)
	go s.cutToWalLoop(inst)
	go s.cleanupLoop(inst)

	return inst, nil
}

func (s *LiveStore) cutToWalLoop(instance *instance) {
	defer s.wg.Done()

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
			s.cutOneInstanceToWal(instance, false)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *LiveStore) cleanupLoop(inst *instance) {
	defer s.wg.Done()

	// ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// dump any blocks that have been flushed for a while
			err := inst.deleteOldBlocks()
			if err != nil {
				level.Error(s.logger).Log("msg", "failed to delete old blocks", "err", err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *LiveStore) getInstances() []*instance {
	s.instancesMtx.RLock()
	defer s.instancesMtx.RUnlock()
	instances := make([]*instance, 0, len(s.instances))
	for _, inst := range s.instances {
		instances = append(instances, inst)
	}
	return instances
}

func (s *LiveStore) cutAllInstancesToWal() {
	instances := s.getInstances()

	for _, instance := range instances {
		s.cutOneInstanceToWal(instance, true)
	}
}

func (s *LiveStore) cutOneInstanceToWal(inst *instance, immediate bool) {
	// Regular trace cuts (live traces -> head block)
	err := inst.cutIdleTraces(immediate)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to cut idle traces", "tenant", inst.tenantID, "err", err)
	}

	// Regular block cuts
	blockID, err := inst.cutBlocks(immediate)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to cut blocks", "tenant", inst.tenantID, "err", err)
	}

	// If head block is cut, enqueue complete operation
	if blockID != uuid.Nil {
		err = s.enqueueCompleteOp(inst.tenantID, blockID)
		if err != nil {
			return
		}
	}
}

func (s *LiveStore) enqueueCompleteOp(tenantID string, blockID uuid.UUID) error {
	op := &completeOp{
		tenantID: tenantID,
		blockID:  blockID,
		// Initial priority and backoff
		at: time.Now(),
		bo: 30 * time.Second,
	}
	_, err := s.flushqueues.Enqueue(op)
	return err
}

func (s *LiveStore) completeLoop() {
	defer s.wg.Done()

	// TODO: Add concurrency
	for {
		o := s.flushqueues.Dequeue()
		if o == nil {
			return // queue is closed
		}
		op := o.(*completeOp)
		op.attempts++

		if op.attempts > maxFlushAttempts {
			level.Error(s.logger).Log("msg", "failed to complete operation", "tenant", op.tenantID, "block", op.blockID, "attempts", op.attempts)
			continue
		}

		inst, err := s.getOrCreateInstance(op.tenantID)
		if err != nil {
			// TODO: How to handle?
			level.Error(s.logger).Log("msg", "failed to retrieve instance for completion", "tenant", op.tenantID, "err", err)
			return
		}
		err = inst.completeBlock(s.ctx, op.blockID)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to complete block", "tenant", op.tenantID, "block", op.blockID, "err", err)

			delay := op.backoff()
			op.at = time.Now().Add(delay)

			go func() {
				time.Sleep(delay)

				if _, err := s.flushqueues.Enqueue(op); err != nil {
					_ = level.Error(s.logger).Log("msg", "failed to requeue block for flushing", "tenant", op.tenantID, "block", op.blockID, "err", err)
				}
			}()

		}
	}
}
