package livestore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
)

const (
	liveStoreServiceName = "live-store"

	// ringAutoForgetUnhealthyPeriods is how many consecutive timeout periods an unhealthy instance
	// in the ring will be automatically removed.
	ringAutoForgetUnhealthyPeriods = 2

	PartitionRingKey  = "livestore-partitions"
	PartitionRingName = "livestore-partitions"

	droppedRecordReasonTooOld           = "too_old"
	droppedRecordReasonDecodingFailed   = "decoding_failed"
	droppedRecordReasonInstanceNotFound = "instance_not_found"
)

var (
	// Queue management metrics
	metricCompleteQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo_live_store",
		Name:      "complete_queue_length",
		Help:      "Number of wal blocks waiting for completion",
	})

	// Block completion metrics
	metricBlocksCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "blocks_completed_total",
		Help:      "The total number of blocks completed",
	})
	metricFailedCompletions = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "failed_completions_total",
		Help:      "The total number of failed block completions",
	})
	metricCompletionRetries = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "completion_retries_total",
		Help:      "The total number of retries after a failed completion",
	})
	metricCompletionFailedRetries = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "completion_failed_retries_total",
		Help:      "The total number of failed retries after a failed completion",
	})
	metricCompletionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo_live_store",
		Name:                            "completion_duration_seconds",
		Help:                            "Records the amount of time to complete a block.",
		Buckets:                         prometheus.ExponentialBuckets(1, 2, 10),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})

	// Kafka/Ingest specific metrics
	metricRecordsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "records_processed_total",
		Help:      "The total number of kafka records processed per tenant.",
	}, []string{"tenant"})

	metricRecordsDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "records_dropped_total",
		Help:      "The total number of kafka records dropped per tenant.",
	}, []string{"tenant", "reason"})
)

type LiveStore struct {
	services.Service

	cfg    Config
	logger log.Logger
	reg    prometheus.Registerer

	subservicesWatcher *services.FailureWatcher

	ingestPartitionID         int32
	ingestPartitionLifecycler *ring.PartitionInstanceLifecycler
	livestoreLifecycler       *ring.BasicLifecycler

	client  *kgo.Client
	decoder *ingest.Decoder

	reader *PartitionReader

	// Multi-tenant instances
	instancesMtx sync.RWMutex
	instances    map[string]*instance
	wal          *wal.WAL
	overrides    overrides.Interface

	// Background processing
	ctx             context.Context // context for the service. all background processes should exit if this is cancelled
	cancel          func()
	wg              sync.WaitGroup
	completeQueues  *flushqueues.ExclusiveQueues
	startupComplete chan struct{} // channel to signal that the starting function has finished. allows background processes to block until the service is fully started
}

func New(cfg Config, overridesService overrides.Interface, logger log.Logger, reg prometheus.Registerer, singlePartition bool) (*LiveStore, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &LiveStore{
		cfg:             cfg,
		logger:          logger,
		reg:             reg,
		decoder:         ingest.NewDecoder(),
		ctx:             ctx,
		cancel:          cancel,
		instances:       make(map[string]*instance),
		overrides:       overridesService,
		completeQueues:  flushqueues.New(cfg.CompleteBlockConcurrency, metricCompleteQueueLength),
		startupComplete: make(chan struct{}),
	}

	var err error
	if singlePartition {
		// For single-binary don't require hostname to identify a partition.
		// Assume partition 0.
		s.ingestPartitionID = 0
	} else {
		s.ingestPartitionID, err = ingest.IngesterPartitionID(cfg.Ring.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("calculating livestore partition ID: %w", err)
		}
	}

	// TODO: It's probably easier to just use the ID directly
	//  https://raintank-corp.slack.com/archives/C05CAA0ULUF/p1752847274420489
	s.cfg.IngestConfig.Kafka.ConsumerGroup, err = ingest.LiveStoreConsumerGroupID(cfg.Ring.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("calculating livestore consumer group ID: %w", err)
	}

	// setup partition ring
	partitionRingKV := cfg.PartitionRing.KVStore.Mock
	if partitionRingKV == nil {
		partitionRingKV, err = kv.NewClient(cfg.PartitionRing.KVStore, ring.GetPartitionRingCodec(), kv.RegistererWithKVName(reg, PartitionRingName+"-lifecycler"), logger)
		if err != nil {
			return nil, fmt.Errorf("creating KV store for livestore partition ring: %w", err)
		}
	}

	s.ingestPartitionLifecycler = ring.NewPartitionInstanceLifecycler(
		s.cfg.PartitionRing.ToLifecyclerConfig(s.ingestPartitionID, cfg.Ring.InstanceID),
		PartitionRingName,
		PartitionRingKey,
		partitionRingKV,
		logger,
		prometheus.WrapRegistererWithPrefix("tempo_", reg))

	// setup live store read ring
	ringStore := cfg.Ring.KVStore.Mock
	if ringStore == nil {
		ringStore, err = kv.NewClient(
			cfg.Ring.KVStore,
			ring.GetCodec(),
			kv.RegistererWithKVName(prometheus.WrapRegistererWithPrefix("tempo_", reg), "livestore"),
			s.logger,
		)
		if err != nil {
			return nil, fmt.Errorf("create KV store client: %w", err)
		}
	}

	lifecyclerCfg, err := cfg.Ring.ToLifecyclerConfig(0)
	if err != nil {
		return nil, fmt.Errorf("invalid ring lifecycler config: %w", err)
	}

	// Define lifecycler delegates in reverse order (last to be called defined first because they're
	// chained via "next delegate").
	delegate := ring.BasicLifecyclerDelegate(s)
	delegate = ring.NewLeaveOnStoppingDelegate(delegate, s.logger)
	delegate = ring.NewAutoForgetDelegate(ringAutoForgetUnhealthyPeriods*cfg.Ring.HeartbeatTimeout, delegate, s.logger)

	s.livestoreLifecycler, err = ring.NewBasicLifecycler(lifecyclerCfg, liveStoreServiceName, liveStoreServiceName, ringStore, delegate, s.logger, prometheus.WrapRegistererWithPrefix("tempo_", reg))
	if err != nil {
		return nil, fmt.Errorf("create ring lifecycler: %w", err)
	}

	s.subservicesWatcher = services.NewFailureWatcher()
	s.subservicesWatcher.WatchService(s.ingestPartitionLifecycler)
	s.subservicesWatcher.WatchService(s.livestoreLifecycler)

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)

	return s, nil
}

func (s *LiveStore) starting(ctx context.Context) error {
	var err error
	s.wal, err = wal.New(&s.cfg.WAL)
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}

	err = s.reloadBlocks()
	if err != nil {
		return fmt.Errorf("failed to reload blocks from wal: %w", err)
	}

	for _, inst := range s.getInstances() {
		err = inst.deleteOldBlocks()
		if err != nil {
			level.Warn(s.logger).Log("msg", "failed to delete old blocks", "err", err, "tenant", inst.tenantID)
		}
	}

	err = services.StartAndAwaitRunning(ctx, s.ingestPartitionLifecycler)
	if err != nil {
		return fmt.Errorf("failed to start partition lifecycler: %w", err)
	}

	err = services.StartAndAwaitRunning(ctx, s.livestoreLifecycler)
	if err != nil {
		return fmt.Errorf("failed to start livestore lifecycler: %w", err)
	}

	s.client, err = ingest.NewReaderClient(
		s.cfg.IngestConfig.Kafka,
		ingest.NewReaderClientMetrics(liveStoreServiceName, s.reg),
		s.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create kafka reader client: %w", err)
	}

	err = ingest.WaitForKafkaBroker(ctx, s.client, s.logger)
	if err != nil {
		return fmt.Errorf("failed to start livestore: %w", err)
	}

	s.reader, err = NewPartitionReaderForPusher(s.client, s.ingestPartitionID, s.cfg.IngestConfig.Kafka, s.cfg.CommitInterval, s.consume, s.logger, s.reg)
	if err != nil {
		return fmt.Errorf("failed to create partition reader: %w", err)
	}
	err = services.StartAndAwaitRunning(ctx, s.reader)
	if err != nil {
		return fmt.Errorf("failed to start partition reader: %w", err)
	}

	for i := range s.cfg.CompleteBlockConcurrency {
		idx := i
		s.runInBackground(func() {
			s.globalCompleteLoop(idx)
		})
	}

	// allow background processes to start
	s.startAllBackgroundProcesses()

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

	// Flush all data to disk
	s.cutAllInstancesToWal()

	if s.cfg.holdAllBackgroundProcesses { // nothing to do
		return nil
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(s.cfg.InstanceCleanupPeriod)
	defer timeout.Stop()

	s.stopAllBackgroundProcesses()

	return nil
}

func (s *LiveStore) consume(ctx context.Context, rs recordIter, now time.Time) (*kadm.Offset, error) {
	defer s.decoder.Reset()
	_, span := tracer.Start(ctx, "LiveStore.consume")
	defer span.End()

	recordCount := 0
	var lastRecord *kgo.Record

	cutoff := now.Add(-s.cfg.CompleteBlockTimeout)
	// Process records by tenant
	for !rs.Done() {
		record := rs.Next()
		tenant := string(record.Key)

		if record.Timestamp.Before(cutoff) {
			metricRecordsDropped.WithLabelValues(tenant, droppedRecordReasonTooOld).Inc()
			continue
		}

		s.decoder.Reset()
		pushReq, err := s.decoder.Decode(record.Value)
		if err != nil {
			metricRecordsDropped.WithLabelValues(tenant, droppedRecordReasonDecodingFailed).Inc()
			level.Error(s.logger).Log("msg", "failed to decoded record", "tenant", tenant, "err", err)
			span.RecordError(err)
			continue
		}

		// Get or create tenant instance
		inst, err := s.getOrCreateInstance(tenant)
		if err != nil {
			metricRecordsDropped.WithLabelValues(tenant, droppedRecordReasonInstanceNotFound).Inc()
			level.Error(s.logger).Log("msg", "failed to get instance for tenant", "tenant", tenant, "err", err)
			span.RecordError(err)
			continue
		}

		// Push data to tenant instance
		inst.pushBytes(ctx, record.Timestamp, pushReq)

		metricRecordsProcessed.WithLabelValues(tenant).Inc()
		recordCount++
		lastRecord = record
	}

	span.SetAttributes(attribute.Int("records_count", recordCount))

	if lastRecord == nil {
		return nil, nil
	}

	offset := kadm.NewOffsetFromRecord(lastRecord)
	return &offset, nil
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
	inst, err := newInstance(tenantID, s.cfg, s.wal, s.overrides, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance for tenant %s: %w", tenantID, err)
	}

	s.instances[tenantID] = inst

	s.runInBackground(func() {
		s.perTenantCutToWalLoop(inst)
	})
	s.runInBackground(func() {
		s.perTenantCleanupLoop(inst)
	})

	return inst, nil
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
		err = s.enqueueCompleteOp(inst.tenantID, blockID, false)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to enqueue complete operation", "tenant", inst.tenantID, "err", err)
			return
		}
	}
}

// OnRingInstanceRegister implements ring.BasicLifecyclerDelegate
func (s *LiveStore) OnRingInstanceRegister(_ *ring.BasicLifecycler, _ ring.Desc, _ bool, _ string, _ ring.InstanceDesc) (ring.InstanceState, ring.Tokens) {
	return ring.ACTIVE, nil // no tokens needed for the livestore ring, we just need to be in the ring for service discovery
}

// OnRingInstanceTokens implements ring.BasicLifecyclerDelegate
func (s *LiveStore) OnRingInstanceTokens(*ring.BasicLifecycler, ring.Tokens) {
}

// OnRingInstanceStopping implements ring.BasicLifecyclerDelegate
func (s *LiveStore) OnRingInstanceStopping(*ring.BasicLifecycler) {
}

// OnRingInstanceHeartbeat implements ring.BasicLifecyclerDelegate
func (s *LiveStore) OnRingInstanceHeartbeat(*ring.BasicLifecycler, *ring.Desc, *ring.InstanceDesc) {
}

// FindTraceByID implements tempopb.Querier
func (s *LiveStore) FindTraceByID(ctx context.Context, req *tempopb.TraceByIDRequest) (*tempopb.TraceByIDResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}
	return inst.FindByTraceID(ctx, req.TraceID, req.AllowPartialTrace)
}

// SearchRecent implements tempopb.Querier
func (s *LiveStore) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	return inst.Search(ctx, req)
}

// SearchBlock implements tempopb.Querier
func (s *LiveStore) SearchBlock(_ context.Context, _ *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	return nil, fmt.Errorf("SearchBlock not implemented in livestore")
}

// SearchTags implements tempopb.Querier
func (s *LiveStore) SearchTags(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	return inst.SearchTags(ctx, req.Scope)
}

// SearchTagsV2 implements tempopb.Querier
func (s *LiveStore) SearchTagsV2(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsV2Response, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	return inst.SearchTagsV2(ctx, req)
}

// SearchTagValues implements tempopb.Querier
func (s *LiveStore) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	return inst.SearchTagValues(ctx, req)
}

// SearchTagValuesV2 implements tempopb.Querier
func (s *LiveStore) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	return inst.SearchTagValuesV2(ctx, req)
}

// PushSpans implements tempopb.MetricsGeneratorServer
func (s *LiveStore) PushSpans(_ context.Context, _ *tempopb.PushSpansRequest) (*tempopb.PushResponse, error) {
	return nil, fmt.Errorf("PushSpans not implemented in livestore")
}

// GetMetrics implements tempopb.MetricsGeneratorServer
func (s *LiveStore) GetMetrics(_ context.Context, _ *tempopb.SpanMetricsRequest) (*tempopb.SpanMetricsResponse, error) {
	return nil, fmt.Errorf("GetMetrics not implemented in livestore") // todo: this is metrics summary, are we allowed to remove this or do we need to continue to support?
}

// QueryRange implements tempopb.MetricsGeneratorServer
func (s *LiveStore) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}
	return inst.QueryRange(ctx, req)
}
