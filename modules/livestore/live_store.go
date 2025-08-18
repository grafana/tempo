package livestore

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	liveStoreServiceName = "live-store"

	ringNumTokens = 1 // we only need 1 token in the read ring per instance b/c it's for service discovery only. sharding is done with the parttition ring.

	// ringAutoForgetUnhealthyPeriods is how many consecutive timeout periods an unhealthy instance
	// in the ring will be automatically removed.
	ringAutoForgetUnhealthyPeriods = 2
)

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
	livestoreLifecycler       *ring.BasicLifecycler

	client  *kgo.Client
	decoder *ingest.Decoder

	reader *PartitionReader

	// Multi-tenant instances
	instancesMtx sync.RWMutex
	instances    map[string]*instance
	wal          *wal.WAL
	overrides    overrides.Interface

	flushqueues *flushqueues.PriorityQueue

	// Background processing
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
}

func New(cfg Config, overridesService overrides.Interface, logger log.Logger, reg prometheus.Registerer, singlePartition bool) (*LiveStore, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &LiveStore{
		cfg:         cfg,
		logger:      logger,
		reg:         reg,
		decoder:     ingest.NewDecoder(),
		ctx:         ctx,
		cancel:      cancel,
		instances:   make(map[string]*instance),
		overrides:   overridesService,
		flushqueues: flushqueues.NewPriorityQueue(metricCompleteQueueLength),
	}

	var err error
	if singlePartition {
		// For single-binary don't require hostname to identify a partition.
		// Assume partition 0.
		s.ingestPartitionID = 0
	} else {
		s.ingestPartitionID, err = ingest.IngesterPartitionID(cfg.Ring.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("calculating ingester partition ID: %w", err)
		}
	}

	// TODO: It's probably easier to just use the ID directly
	//  https://raintank-corp.slack.com/archives/C05CAA0ULUF/p1752847274420489
	s.cfg.IngestConfig.Kafka.ConsumerGroup, err = ingest.LiveStoreConsumerGroupID(cfg.Ring.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("calculating ingester consumer group ID: %w", err)
	}

	// setup partition ring
	partitionRingKV := cfg.PartitionRing.KVStore.Mock
	if partitionRingKV == nil {
		partitionRingKV, err = kv.NewClient(cfg.PartitionRing.KVStore, ring.GetPartitionRingCodec(), kv.RegistererWithKVName(reg, ingester.PartitionRingName+"-lifecycler"), logger)
		if err != nil {
			return nil, fmt.Errorf("creating KV store for ingester partition ring: %w", err)
		}
	}

	s.ingestPartitionLifecycler = ring.NewPartitionInstanceLifecycler(
		s.cfg.PartitionRing.ToLifecyclerConfig(s.ingestPartitionID, cfg.Ring.InstanceID),
		ingester.PartitionRingName,
		ingester.PartitionRingKey,
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

	lifecyclerCfg, err := cfg.Ring.ToLifecyclerConfig(ringNumTokens)
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
	inst, err := newInstance(tenantID, s.cfg, s.wal, s.overrides, s.logger)
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

// OnRingInstanceRegister implements ring.BasicLifecyclerDelegate
func (s *LiveStore) OnRingInstanceRegister(_ *ring.BasicLifecycler, _ ring.Desc, _ bool, _ string, _ ring.InstanceDesc) (ring.InstanceState, ring.Tokens) {
	// tokens don't matter for the livestore ring, we just need to be in the ring for service discovery
	token := rand.Uint32() //nolint: gosec // G404 we don't need a cryptographic random number here
	level.Info(s.logger).Log("msg", "registered in livestore ring", "token", token, "partition", s.ingestPartitionID)

	return ring.ACTIVE, []uint32{token}
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
func (s *LiveStore) FindTraceByID(_ context.Context, _ *tempopb.TraceByIDRequest) (*tempopb.TraceByIDResponse, error) {
	return nil, fmt.Errorf("FindTraceByID not implemented in livestore")
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

	return inst.SearchTagValues(ctx, req.TagName, req.MaxTagValues, req.StaleValueThreshold)
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
	e := traceql.NewEngine()

	// Compile the raw version of the query for head and wal blocks
	// These aren't cached and we put them all into the same evaluator
	// for efficiency.
	// TODO MRD look into how to propogate unsafe query hints.
	rawEval, err := e.CompileMetricsQueryRange(req, int(req.Exemplars), s.cfg.TimeOverlapCutoff, false)
	if err != nil {
		return nil, err
	}

	// This is a summation version of the query for complete blocks
	// which can be cached. They are timeseries, so they need the job-level evaluator.
	jobEval, err := traceql.NewEngine().CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
	if err != nil {
		return nil, err
	}
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := s.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}
	err = inst.QueryRange(ctx, req, rawEval, jobEval)
	if err != nil {
		return nil, err
	}

	// The code below here is taken from modules/generator/instance.go, where it combines the results from the processor.
	// We only have one "processor" so directly evaluating the results.

	// Combine the raw results into the job results
	walResults := rawEval.Results().ToProto(req)
	jobEval.ObserveSeries(walResults)

	r := jobEval.Results()
	rr := r.ToProto(req)

	maxSeries := int(req.MaxSeries)
	if maxSeries > 0 && len(rr) > maxSeries {
		return &tempopb.QueryRangeResponse{
			Series: rr[:maxSeries],
			Status: tempopb.PartialStatus_PARTIAL,
		}, nil
	}

	return &tempopb.QueryRangeResponse{
		Series: rr,
	}, nil
}
