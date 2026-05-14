package livestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/shutdownmarker"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
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
	ErrStarting = errors.New("live-store is starting")
	ErrStopping = errors.New("live-store is stopping")
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

	// Readiness metrics
	metricCatchUpDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo_live_store",
		Name:      "catch_up_duration_seconds",
		Help:      "Time spent catching up at startup",
	})

	metricReady = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo_live_store",
		Name:      "ready",
		Help:      "1 if ready to serve queries, 0 otherwise",
	})

	metricLaggedRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "lagged_requests_total",
		Help:      "Requests where the live-store could not guarantee complete results due to Kafka lag.",
	}, []string{"route"})
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
	instancesMtx           sync.RWMutex
	instances              map[string]*instance
	wal                    *wal.WAL
	completeBlockEncoding  encoding.VersionedEncoding
	completeBlockLifecycle completeBlockLifecycle
	overrides              overrides.Interface

	// Background processing
	ctx                 context.Context // context for the service. all background processes should exit if this is cancelled
	cancel              func()
	wg                  sync.WaitGroup
	completeQueues      *flushqueues.ExclusiveQueues[*completeOp]
	startupComplete     chan struct{} // channel to signal that the starting function has finished. allows background processes to block until the service is fully started
	lagCancel           context.CancelFunc
	readyErr            atomic.Pointer[error] // nil when ready to serve queries
	lastRecordTimeNanos atomic.Int64          // stores timestamp of last consumed record as UnixNano, -1 means not set

	cutToWalStop chan struct{}  // closed to stop perTenantCutToWalLoop goroutines before shutdown flush
	cutToWalWg   sync.WaitGroup // tracks active perTenantCutToWalLoop goroutines
}

func New(cfg Config, overridesService overrides.Interface, completeBlockFlusher completeBlockFlusher, logger log.Logger, reg prometheus.Registerer) (*LiveStore, error) {
	completeBlockEncoding, encErr := encoding.FromVersionForWrites(cfg.BlockConfig.Version)
	if encErr != nil {
		return nil, fmt.Errorf("block version validation failed: %w", encErr)
	}

	completeBlockLifecycle, lifecycleErr := newCompleteBlockLifecycle(cfg, completeBlockFlusher, logger)
	if lifecycleErr != nil {
		return nil, fmt.Errorf("create complete block lifecycle: %w", lifecycleErr)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &LiveStore{
		cfg:                    cfg,
		logger:                 logger,
		reg:                    reg,
		decoder:                ingest.NewDecoder(),
		completeBlockEncoding:  completeBlockEncoding,
		ctx:                    ctx,
		cancel:                 cancel,
		instances:              make(map[string]*instance),
		overrides:              overridesService,
		completeBlockLifecycle: completeBlockLifecycle,
		completeQueues:         flushqueues.New[*completeOp](metricCompleteQueueLength),
		startupComplete:        make(chan struct{}),
		cutToWalStop:           make(chan struct{}),
	}

	// Initialize ready state to starting
	s.readyErr.Store(&ErrStarting)
	metricReady.Set(0)
	s.lastRecordTimeNanos.Store(-1)

	var err error
	if cfg.ConsumeFromKafka {
		s.ingestPartitionID, err = ingest.IngesterPartitionID(cfg.Ring.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("calculating livestore partition ID: %w", err)
		}

		// TODO: It's probably easier to just use the ID directly
		//  https://raintank-corp.slack.com/archives/C05CAA0ULUF/p1752847274420489
		s.cfg.IngestConfig.Kafka.ConsumerGroup, err = ingest.LiveStoreConsumerGroupID(cfg.Ring.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("calculating livestore consumer group ID: %w", err)
		}
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
	// First of all we have to check if the shutdown marker is set. This needs to be done
	// as first thing because, if found, it may change the behaviour of the live-store startup.

	if _, err := os.Stat(s.cfg.ShutdownMarkerDir); os.IsNotExist(err) {
		err := os.MkdirAll(s.cfg.ShutdownMarkerDir, 0o700)
		if err != nil {
			return fmt.Errorf("failed to create shutdown marker directory: %w", err)
		}
	}

	shutdownMarkerPath := shutdownmarker.GetPath(s.cfg.ShutdownMarkerDir)
	if exists, err := shutdownmarker.Exists(shutdownMarkerPath); err != nil {
		return fmt.Errorf("failed to check live-store shutdown marker: %w", err)
	} else if exists {
		level.Info(s.logger).Log("msg", "detected existing shutdown marker, setting prepare for shutdown", "path", shutdownMarkerPath)
		s.setPrepareShutdown()
	}

	var err error
	s.wal, err = wal.New(&s.cfg.WAL)
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}

	err = s.reloadBlocks()
	if err != nil {
		return fmt.Errorf("failed to reload blocks from wal: %w", err)
	}

	level.Info(s.logger).Log("msg", "deleting old blocks")
	for _, inst := range s.getInstances() {
		err = inst.deleteOldBlocks()
		if err != nil {
			level.Warn(s.logger).Log("msg", "failed to delete old blocks", "err", err, "tenant", inst.tenantID)
		}
	}
	level.Info(s.logger).Log("msg", "done deleting old blocks")

	// Set eagerly so the flag is already in place when the lifecycler's stopping()
	// checks it. Setting it in our own stopping() races with context-cancellation
	// that triggers the lifecycler's shutdown first.
	if s.cfg.RemoveOwnerOnShutdown {
		s.ingestPartitionLifecycler.SetRemoveOwnerOnShutdown(true)
	}

	err = services.StartAndAwaitRunning(ctx, s.ingestPartitionLifecycler)
	if err != nil {
		return fmt.Errorf("failed to start partition lifecycler: %w", err)
	}

	err = services.StartAndAwaitRunning(ctx, s.livestoreLifecycler)
	if err != nil {
		return fmt.Errorf("failed to start livestore lifecycler: %w", err)
	}

	if err := s.startIngestPath(ctx); err != nil {
		return err
	}

	for i := range s.cfg.CompleteBlockConcurrency {
		idx := i
		s.runInBackground(func() {
			s.globalCompleteLoop(idx)
		})
	}

	// allow background processes to start
	level.Info(s.logger).Log("msg", "starting all background processes")
	s.startAllBackgroundProcesses()

	level.Info(s.logger).Log("msg", "waiting for ingestion catching up")
	if err := s.waitForIngestPathReady(ctx); err != nil {
		return err
	}
	level.Info(s.logger).Log("msg", "done waiting for ingestion catching up")

	// Mark as ready at end of starting()
	s.readyErr.Store(nil)
	metricReady.Set(1)
	level.Info(s.logger).Log("msg", "live-store ready to serve queries")

	return nil
}

func (s *LiveStore) waitForIngestPathReady(ctx context.Context) error {
	if !s.cfg.ConsumeFromKafka {
		return nil
	}
	// Wait for catch-up before marking ready (if enabled)
	if err := s.waitForCatchUp(ctx); err != nil {
		return fmt.Errorf("failed to catch up: %w", err)
	}
	return nil
}

func (s *LiveStore) startIngestPath(ctx context.Context) error {
	if !s.cfg.ConsumeFromKafka {
		return nil
	}
	return s.startKafkaIngestPath(ctx)
}

// shouldForceFromLookback decides whether to re-read the Kafka lookback to rebuild query state.
// Skipped when local data exists, or when the partition is Inactive (prior pod already drained).
func (s *LiveStore) shouldForceFromLookback(ctx context.Context) bool {
	if len(s.getInstances()) > 0 {
		return false
	}
	state, _, err := s.ingestPartitionLifecycler.GetPartitionState(ctx)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to read partition state, defaulting to lookback replay", "err", err)
		return true
	}
	if state == ring.PartitionInactive {
		level.Info(s.logger).Log("msg", "skipping lookback replay because partition is Inactive")
		return false
	}
	level.Info(s.logger).Log("msg", "no local data found after reload, will force reading from lookback period")
	return true
}

func (s *LiveStore) startKafkaIngestPath(ctx context.Context) error {
	forceFromLookback := s.shouldForceFromLookback(ctx)

	client, err := ingest.NewReaderClient(
		s.cfg.IngestConfig.Kafka,
		ingest.NewReaderClientMetrics(liveStoreServiceName, s.reg),
		s.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create kafka reader client: %w", err)
	}
	s.client = client

	if err := ingest.WaitForKafkaBroker(ctx, s.client, s.logger); err != nil {
		return fmt.Errorf("failed to start livestore: %w", err)
	}

	lookbackPeriod := 2 * s.cfg.CompleteBlockTimeout
	reader, err := NewPartitionReaderForPusher(s.client, s.ingestPartitionID, s.cfg.IngestConfig.Kafka, s.cfg.CommitInterval, lookbackPeriod, forceFromLookback, s.consume, s.logger, s.reg)
	if err != nil {
		return fmt.Errorf("failed to create partition reader: %w", err)
	}
	s.reader = reader

	if err := services.StartAndAwaitRunning(ctx, s.reader); err != nil {
		return fmt.Errorf("failed to start partition reader: %w", err)
	}

	lagCtx, cncl := context.WithCancel(s.ctx)
	s.lagCancel = cncl
	// Start exporting partition lag metrics
	ingest.ExportPartitionLagMetrics(
		lagCtx,
		s.client,
		s.logger,
		s.cfg.IngestConfig,
		func() []int32 { return []int32{s.ingestPartitionID} },
		s.client.ForceMetadataRefresh,
	)

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
	level.Info(s.logger).Log("msg", "live store stopping", "remove_partition_owner", s.ingestPartitionLifecycler.RemoveOwnerOnShutdown())

	// Reject new queries early in shutdown, before tearing down the reader.
	s.readyErr.Store(&ErrStopping)
	metricReady.Set(0)

	var stopErr error

	if s.cfg.ConsumeFromKafka {
		// Stop the kafka lag background worker.
		if s.lagCancel != nil {
			s.lagCancel()
		}
		// Stop consuming.
		if err := services.StopAndAwaitTerminated(context.Background(), s.reader); err != nil {
			level.Warn(s.logger).Log("msg", "failed to stop reader", "err", err)
			stopErr = errors.Join(stopErr, err)
		}

		// Reset lag metrics for our partition when stopping.
		ingest.ResetLagMetricsForRevokedPartitions(s.cfg.IngestConfig.Kafka.ConsumerGroup, []int32{s.ingestPartitionID})
	}

	// Stop both the membership ring and partition ring even if an earlier shutdown step failed.
	if err := services.StopAndAwaitTerminated(context.Background(), s.livestoreLifecycler); err != nil {
		level.Warn(s.logger).Log("msg", "failed to stop livestore lifecycler", "err", err)
		stopErr = errors.Join(stopErr, err)
	}

	if err := services.StopAndAwaitTerminated(context.Background(), s.ingestPartitionLifecycler); err != nil {
		level.Warn(s.logger).Log("msg", "failed to stop partition lifecycler", "err", err)
		stopErr = errors.Join(stopErr, err)
	}

	level.Info(s.logger).Log("msg", "stopping periodic WAL flush goroutines")
	s.stopAllCutToWalLoops()
	level.Info(s.logger).Log("msg", "periodic WAL flush goroutines stopped")

	// Flush all data to disk.
	level.Info(s.logger).Log("msg", "cutting all instances to WAL")
	s.cutAllInstancesToWal()
	level.Info(s.logger).Log("msg", "done cutting all instances to WAL")

	// Remove the shutdown marker if it exists since we are shutting down.
	shutdownMarkerPath := shutdownmarker.GetPath(s.cfg.ShutdownMarkerDir)
	if err := shutdownmarker.Remove(shutdownMarkerPath); err != nil {
		level.Warn(s.logger).Log("msg", "failed to remove shutdown marker", "path", shutdownMarkerPath, "err", err)
		stopErr = errors.Join(stopErr, err)
	}

	if s.cfg.holdAllBackgroundProcesses { // nothing to do
		return stopErr
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(s.cfg.InstanceCleanupPeriod)
	defer timeout.Stop()

	level.Info(s.logger).Log("msg", "stopping all background processes")
	s.stopAllBackgroundProcesses()

	return stopErr
}

func (s *LiveStore) waitForCatchUp(ctx context.Context) error {
	// If disabled (ReadinessTargetLag == 0), mark ready immediately
	// This preserves backward compatibility
	if s.cfg.ReadinessTargetLag == 0 {
		level.Info(s.logger).Log("msg", "catch-up waiting disabled (readiness_target_lag=0)")
		return nil
	}

	startTime := time.Now()

	ticker := time.NewTicker(time.Second) // Check every second
	defer ticker.Stop()

	level.Info(s.logger).Log(
		"msg", "waiting for Kafka catch-up",
		"target_lag", s.cfg.ReadinessTargetLag,
		"max_wait", s.cfg.ReadinessMaxWait,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			elapsed := time.Since(startTime)

			// Check max wait timeout
			if elapsed >= s.cfg.ReadinessMaxWait {
				level.Warn(s.logger).Log(
					"msg", "max catch-up wait exceeded, proceeding anyway",
					"elapsed", elapsed,
					"max_wait", s.cfg.ReadinessMaxWait,
				)
				metricCatchUpDuration.Set(elapsed.Seconds())
				return nil
			}

			// Calculate current lag
			lag := s.calculateTimeLag(1000)
			if lag == nil {
				level.Debug(s.logger).Log("msg", "catch-up lag could not be determined, waiting")
				continue
			}

			level.Debug(s.logger).Log(
				"msg", "catch-up progress",
				"current_lag", *lag,
				"target_lag", s.cfg.ReadinessTargetLag,
				"elapsed", elapsed,
			)

			if *lag <= s.cfg.ReadinessTargetLag {
				level.Info(s.logger).Log(
					"msg", "caught up with Kafka",
					"final_lag", *lag,
					"target_lag", s.cfg.ReadinessTargetLag,
					"elapsed", elapsed,
				)
				metricCatchUpDuration.Set(elapsed.Seconds())
				return nil
			}
		}
	}
}

// Calculate lag based on parameters populated by PartitionReader
// Edge cases:
// - empty partition = no lag
// - nothing has been fetched yet = indeterminate
// - we know the watermark but nothing has been consumed yet = indeterminate
//
// It takes lagShortcutThreshold to shortcut calculations if the lag is close to the end of the partition.
// To disable the shortcut, set lagShortcutThreshold to a negative value.
func (s *LiveStore) calculateTimeLag(lagShortcutThreshold int64) *time.Duration {
	// reader is nil only before starting() creates it. After stopping(), reader
	// is a stopped service but not nil, and its atomic fields remain safe to read.
	if s.reader == nil {
		level.Debug(s.logger).Log("msg", "Partition reader not initialized")
		return nil
	}

	// Use cached high watermark from fetch responses (avoids extra API call)
	lag := s.reader.lag.Load()
	zero := time.Duration(0)

	// If we haven't performed any fetches yet, we can't determine lag
	if lag < 0 {
		level.Debug(s.logger).Log("msg", "High watermark not set yet")
		return nil
	}

	// Check if we are near end or partition is empty
	// Arbitrary value picked to shortcut calculations
	if lagShortcutThreshold >= 0 && lag <= lagShortcutThreshold {
		level.Debug(s.logger).Log(
			"msg", "At or close to partition end",
			"lag", lag)
		return &zero
	}

	nanos := s.lastRecordTimeNanos.Load()
	if nanos < 0 {
		level.Debug(s.logger).Log("msg", "No last record yet")
		// Haven't consumed records - check if offset at end
		return nil // Not caught up yet, can't determine lag
	}

	// Potential race condition that can result in negative lag?
	// Assuming strictly monotonic timestamps in Kafka, can't cause an issue
	lastRecordTime := time.Unix(0, nanos)
	recordLag := time.Since(lastRecordTime)
	return &recordLag
}

func (s *LiveStore) consume(ctx context.Context, rs recordIter, now time.Time) (*kadm.Offset, error) {
	defer s.decoder.Reset()
	ctx, span := tracer.Start(ctx, "LiveStore.consume")
	defer span.End()

	recordCount := 0
	var lastRecord *kgo.Record

	cutoff := now.Add(-s.cfg.CompleteBlockTimeout)
	// Process records by tenant
	for !rs.Done() {
		record := rs.Next()
		tenant := string(record.Key)

		// Track partition lag in seconds
		lag := now.Sub(record.Timestamp)
		ingest.SetPartitionLagSeconds(s.cfg.IngestConfig.Kafka.ConsumerGroup, record.Partition, lag)

		if record.Timestamp.Before(cutoff) {
			metricRecordsDropped.WithLabelValues(tenant, droppedRecordReasonTooOld).Inc()
			lastRecord = record
			continue
		}

		s.decoder.Reset()
		pushReq, err := s.decoder.Decode(record.Value)
		if err != nil {
			metricRecordsDropped.WithLabelValues(tenant, droppedRecordReasonDecodingFailed).Inc()
			level.Error(s.logger).Log("msg", "failed to decoded record", "tenant", tenant, "err", err)
			span.RecordError(err)
			lastRecord = record
			continue
		}

		// Get or create tenant instance
		inst, err := s.getOrCreateInstance(tenant)
		if err != nil {
			metricRecordsDropped.WithLabelValues(tenant, droppedRecordReasonInstanceNotFound).Inc()
			level.Error(s.logger).Log("msg", "failed to get instance for tenant", "tenant", tenant, "err", err)
			span.RecordError(err)
			lastRecord = record
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

	// Store the timestamp of the last consumed record for lag calculation
	s.lastRecordTimeNanos.Store(lastRecord.Timestamp.UnixNano())

	offset := kadm.NewOffsetFromRecord(lastRecord)
	return &offset, nil
}

func (s *LiveStore) getInstance(tenantID string) (*instance, bool) {
	s.instancesMtx.RLock()
	defer s.instancesMtx.RUnlock()
	inst, ok := s.instances[tenantID]
	return inst, ok
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
	inst, err := newInstance(tenantID, s.cfg, s.wal, s.completeBlockEncoding, s.completeBlockLifecycle, s.overrides, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance for tenant %s: %w", tenantID, err)
	}

	s.instances[tenantID] = inst

	s.startPerTenantCutToWalLoop(inst)
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
		s.cutOneInstanceToWal(s.ctx, instance, true)
	}
}

func (s *LiveStore) cutOneInstanceToWal(ctx context.Context, inst *instance, immediate bool) {
	ctx, span := tracer.Start(ctx, "LiveStore.cutOneInstanceToWal",
		oteltrace.WithAttributes(
			attribute.String("tenant", inst.tenantID),
			attribute.Bool("immediate", immediate),
		))
	defer span.End()

	var liveTracesDrained bool
	var err error
	for !liveTracesDrained {
		// Regular trace cuts (live traces -> head block)
		liveTracesDrained, err = inst.cutIdleTraces(ctx, immediate)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to cut idle traces", "tenant", inst.tenantID, "err", err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			break
		}
		id, err := inst.cutBlocks(ctx, immediate)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to cut blocks", "tenant", inst.tenantID, "err", err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			break
		}
		if id != uuid.Nil {
			span.AddEvent("block enqueued for completion",
				oteltrace.WithAttributes(attribute.String("blockID", id.String())))
			if err := s.enqueueCompleteOp(inst.tenantID, id, false); err != nil {
				level.Error(s.logger).Log("msg", "failed to enqueue complete operation", "tenant", inst.tenantID, "err", err)
				span.RecordError(err)
			}
		}
	}
}

// CheckReady returns nil if the live-store is ready to serve queries
func (s *LiveStore) CheckReady(_ context.Context) error {
	if err := s.readyErr.Load(); err != nil {
		return *err
	}
	return nil
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

// PushBytes ingests pre-marshaled traces directly into the local live-store.
func (s *LiveStore) PushBytes(ctx context.Context, req *tempopb.PushBytesRequest) (*tempopb.PushResponse, error) {
	if err := s.CheckReady(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New("nil push bytes request")
	}
	if len(req.Traces) != len(req.Ids) {
		return nil, fmt.Errorf("mismatched traces and ids length: traces=%d ids=%d", len(req.Traces), len(req.Ids))
	}
	if len(req.Traces) == 0 {
		return &tempopb.PushResponse{}, nil
	}

	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(tenantID)
	if err != nil {
		return nil, err
	}

	inst.pushBytes(ctx, time.Now(), req)
	return &tempopb.PushResponse{}, nil
}

// FindTraceByID implements tempopb.Querier
func (s *LiveStore) FindTraceByID(ctx context.Context, req *tempopb.TraceByIDRequest) (*tempopb.TraceByIDResponse, error) {
	return withInstance(ctx, s, func(inst *instance) (*tempopb.TraceByIDResponse, error) {
		return inst.FindByTraceID(ctx, req.TraceID, req.AllowPartialTrace)
	})
}

// SearchRecent implements tempopb.Querier
func (s *LiveStore) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	return withInstance(ctx, s, func(inst *instance) (*tempopb.SearchResponse, error) {
		if s.isLagged(int64(req.End)*1e9, "/tempopb.Querier/SearchRecent", req.Query) { // convert seconds to nanoseconds
			metricLaggedRequests.WithLabelValues("/tempopb.Querier/SearchRecent").Inc()
			if s.cfg.FailOnHighLag {
				return nil, errLagged
			}
		}
		return inst.Search(ctx, req)
	})
}

// SearchBlock implements tempopb.Querier
func (s *LiveStore) SearchBlock(_ context.Context, _ *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	return nil, fmt.Errorf("SearchBlock not implemented in livestore")
}

// SearchTags implements tempopb.Querier
func (s *LiveStore) SearchTags(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsResponse, error) {
	return withInstance(ctx, s, func(inst *instance) (*tempopb.SearchTagsResponse, error) {
		return inst.SearchTags(ctx, req.Scope)
	})
}

// SearchTagsV2 implements tempopb.Querier
func (s *LiveStore) SearchTagsV2(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsV2Response, error) {
	return withInstance(ctx, s, func(inst *instance) (*tempopb.SearchTagsV2Response, error) {
		return inst.SearchTagsV2(ctx, req)
	})
}

// SearchTagValues implements tempopb.Querier
func (s *LiveStore) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	return withInstance(ctx, s, func(inst *instance) (*tempopb.SearchTagValuesResponse, error) {
		return inst.SearchTagValues(ctx, req)
	})
}

// SearchTagValuesV2 implements tempopb.Querier
func (s *LiveStore) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	return withInstance(ctx, s, func(inst *instance) (*tempopb.SearchTagValuesV2Response, error) {
		return inst.SearchTagValuesV2(ctx, req)
	})
}

// QueryRange implements tempopb.MetricsServer
func (s *LiveStore) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	return withInstance(ctx, s, func(inst *instance) (*tempopb.QueryRangeResponse, error) {
		if s.isLagged(int64(req.End), "/tempopb.Metrics/QueryRange", req.Query) { // end param is already nanos, no need to convert
			metricLaggedRequests.WithLabelValues("/tempopb.Metrics/QueryRange").Inc()
			if s.cfg.FailOnHighLag {
				return nil, errLagged
			}
		}
		return inst.QueryRange(ctx, req)
	})
}

var errLagged = errors.New("cannot guarantee complete results")

func (s *LiveStore) isLagged(endNanos int64, route, query string) bool {
	if !s.cfg.ConsumeFromKafka { // if config disabled or no kafka consumption, never lagged
		return false
	}
	lag := s.calculateTimeLag(0)
	// prefer fail when lag is unknown; otherwise lagged means queryEnd is more recent than the last consumed record.
	lagged := lag == nil || time.Since(time.Unix(0, endNanos)) < *lag
	if lagged {
		timeLagSec := -1.0
		if lag != nil {
			timeLagSec = lag.Seconds()
		}
		level.Info(s.logger).Log(
			"msg", "isLagged tripped",
			"route", route,
			"query", query,
			"end_unix_nano", endNanos,
			"now_unix_nano", time.Now().UnixNano(),
			"time_lag_sec", timeLagSec,
			"offset_lag", s.reader.lag.Load(),
			"last_record_unix_nano", s.lastRecordTimeNanos.Load(),
		)
	}
	return lagged
}

// withInstance extracts the tenant ID from the context, gets the instance,
// and calls the provided function if the instance exists. If the instance
// doesn't exist, it returns the zero value.
func withInstance[T any](ctx context.Context, s *LiveStore, fn func(*instance) (*T, error)) (*T, error) {
	var defaultValue T

	// Check readiness before processing query
	if err := s.CheckReady(ctx); err != nil {
		return &defaultValue, err
	}

	instanceID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return &defaultValue, err
	}

	inst, found := s.getInstance(instanceID)
	if inst == nil || !found {
		return &defaultValue, nil
	}

	return fn(inst)
}
