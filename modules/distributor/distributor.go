package distributor

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/grafana/dskit/limiter"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/util/strutil"
	"github.com/segmentio/fasthash/fnv1a"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/distributor/receiver"
	"github.com/grafana/tempo/modules/distributor/usage"
	"github.com/grafana/tempo/modules/generator"
	generator_client "github.com/grafana/tempo/modules/generator/client"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/dataquality"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/validation"
)

const (
	// reasonRateLimited indicates that the tenants spans/second exceeded their limits
	reasonRateLimited = "rate_limited"
	// reasonTraceTooLarge indicates that a single trace has too many spans
	reasonTraceTooLarge = "trace_too_large"
	// reasonLiveTracesExceeded indicates that tempo is already tracking too many live traces in the ingesters for this user
	reasonLiveTracesExceeded = "live_traces_exceeded"
	// reasonUnknown indicates a pushByte error at the ingester level not related to GRPC
	reasonUnknown = "unknown_error"

	distributorRingKey = "distributor"
)

var (
	metricIngesterAppends = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_ingester_appends_total",
		Help:      "The total number of batch appends sent to ingesters.",
	}, []string{"ingester"})
	metricIngesterAppendFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_ingester_append_failures_total",
		Help:      "The total number of failed batch appends sent to ingesters.",
	}, []string{"ingester"})
	metricGeneratorPushes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_metrics_generator_pushes_total",
		Help:      "The total number of span pushes sent to metrics-generators.",
	}, []string{"metrics_generator"})
	metricGeneratorPushesFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_metrics_generator_pushes_failures_total",
		Help:      "The total number of failed span pushes sent to metrics-generators.",
	}, []string{"metrics_generator"})
	metricSpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_spans_received_total",
		Help:      "The total number of spans received per tenant",
	}, []string{"tenant"})
	metricDebugSpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_debug_spans_received_total",
		Help:      "Debug counters for spans received per tenant",
	}, []string{"tenant", "name", "service"})
	metricBytesIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_bytes_received_total",
		Help:      "The total number of proto bytes received per tenant",
	}, []string{"tenant"})
	metricTracesPerBatch = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Name:                            "distributor_traces_per_batch",
		Help:                            "The number of traces in each batch",
		Buckets:                         prometheus.ExponentialBuckets(2, 2, 10),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
	metricIngesterClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "distributor_ingester_clients",
		Help:      "The current number of ingester clients.",
	})
	metricGeneratorClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "distributor_metrics_generator_clients",
		Help:      "The current number of metrics-generator clients.",
	})
	metricAttributesTruncated = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_attributes_truncated_total",
		Help:      "The total number of attribute keys or values truncated per tenant",
	}, []string{"tenant"})
	metricKafkaRecordsPerRequest = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "kafka_records_per_request",
		Help:      "The number of records in each kafka request",
	})
	metricKafkaWriteLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "kafka_write_latency_seconds",
		Help:      "The latency of writing to kafka",
	})
	metricKafkaWriteBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "kafka_write_bytes_total",
		Help:      "The total number of bytes written to kafka",
	}, []string{"partition"})
	metricKafkaAppends = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "kafka_appends_total",
		Help:      "The total number of appends sent to kafka",
	}, []string{"partition", "status"})

	statBytesReceived = usagestats.NewCounter("distributor_bytes_received")
	statSpansReceived = usagestats.NewCounter("distributor_spans_received")
)

var tracer = otel.Tracer("modules/distributor")

// rebatchedTrace is used to more cleanly pass the set of data
type rebatchedTrace struct {
	id        []byte
	trace     *tempopb.Trace
	start     uint32 // unix epoch seconds
	end       uint32 // unix epoch seconds
	spanCount int
}

// Distributor coordinates replicates and distribution of log streams.
type Distributor struct {
	services.Service

	cfg             Config
	clientCfg       ingester_client.Config
	ingestersRing   ring.ReadRing
	pool            *ring_client.Pool
	DistributorRing *ring.Ring
	overrides       overrides.Interface
	traceEncoder    model.SegmentDecoder

	// metrics-generator
	generatorClientCfg generator_client.Config
	generatorsRing     ring.ReadRing
	generatorsPool     *ring_client.Pool
	generatorForwarder *generatorForwarder

	// Generic Forwarder
	forwardersManager *forwarder.Manager

	// Kafka
	kafkaProducer *ingest.Producer
	partitionRing ring.PartitionRingReader

	// Per-user rate limiter.
	ingestionRateLimiter *limiter.RateLimiter

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	usage *usage.Tracker

	logger log.Logger

	// For testing functionality that relies on timing without having to sleep in unit tests.
	sleep func(time.Duration)
	now   func() time.Time
}

// New a distributor creates.
func New(
	cfg Config,
	clientCfg ingester_client.Config,
	ingestersRing ring.ReadRing,
	generatorClientCfg generator_client.Config,
	generatorsRing ring.ReadRing,
	partitionRing ring.PartitionRingReader,
	o overrides.Interface,
	middleware receiver.Middleware,
	logger log.Logger,
	loggingLevel dslog.Level,
	reg prometheus.Registerer,
) (*Distributor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	factory := cfg.factory
	if factory == nil {
		factory = func(addr string) (ring_client.PoolClient, error) {
			return ingester_client.New(addr, clientCfg)
		}
	}

	subservices := []services.Service(nil)

	// Create the configured ingestion rate limit strategy (local or global).
	var ingestionRateStrategy limiter.RateLimiterStrategy
	var distributorRing *ring.Ring

	if o.IngestionRateStrategy() == overrides.GlobalIngestionRateStrategy {
		lifecyclerCfg := cfg.DistributorRing.ToLifecyclerConfig()
		lifecycler, err := ring.NewLifecycler(lifecyclerCfg, nil, "distributor", cfg.OverrideRingKey, false, logger, prometheus.WrapRegistererWithPrefix("tempo_", reg))
		if err != nil {
			return nil, err
		}
		subservices = append(subservices, lifecycler)
		ingestionRateStrategy = newGlobalIngestionRateStrategy(o, lifecycler)

		ring, err := ring.New(lifecyclerCfg.RingConfig, "distributor", cfg.OverrideRingKey, logger, prometheus.WrapRegistererWithPrefix("tempo_", reg))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize distributor ring: %w", err)
		}
		distributorRing = ring
		subservices = append(subservices, distributorRing)
	} else {
		ingestionRateStrategy = newLocalIngestionRateStrategy(o)
	}

	pool := ring_client.NewPool("distributor_pool",
		clientCfg.PoolConfig,
		ring_client.NewRingServiceDiscovery(ingestersRing),
		factory,
		metricIngesterClients,
		logger)

	subservices = append(subservices, pool)

	d := &Distributor{
		cfg:                  cfg,
		clientCfg:            clientCfg,
		ingestersRing:        ingestersRing,
		pool:                 pool,
		DistributorRing:      distributorRing,
		ingestionRateLimiter: limiter.NewRateLimiter(ingestionRateStrategy, 10*time.Second),
		generatorClientCfg:   generatorClientCfg,
		generatorsRing:       generatorsRing,
		partitionRing:        partitionRing,
		overrides:            o,
		traceEncoder:         model.MustNewSegmentDecoder(model.CurrentEncoding),
		logger:               logger,
		sleep:                time.Sleep,
		now:                  time.Now,
	}

	if cfg.Usage.CostAttribution.Enabled {
		usage, err := usage.NewTracker(cfg.Usage.CostAttribution, "cost-attribution", o.CostAttributionDimensions, o.CostAttributionMaxCardinality)
		if err != nil {
			return nil, fmt.Errorf("creating usage tracker: %w", err)
		}
		d.usage = usage
	}

	var generatorsPoolFactory ring_client.PoolAddrFunc = func(addr string) (ring_client.PoolClient, error) {
		return generator_client.New(addr, generatorClientCfg)
	}
	d.generatorsPool = ring_client.NewPool(
		"distributor_metrics_generator_pool",
		generatorClientCfg.PoolConfig,
		ring_client.NewRingServiceDiscovery(generatorsRing),
		generatorsPoolFactory,
		metricGeneratorClients,
		logger,
	)

	subservices = append(subservices, d.generatorsPool)

	d.generatorForwarder = newGeneratorForwarder(logger, d.sendToGenerators, o)
	subservices = append(subservices, d.generatorForwarder)

	forwardersManager, err := forwarder.NewManager(d.cfg.Forwarders, logger, o, loggingLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create forwarders manager: %w", err)
	}

	d.forwardersManager = forwardersManager
	subservices = append(subservices, d.forwardersManager)

	cfgReceivers := cfg.Receivers
	if len(cfgReceivers) == 0 {
		cfgReceivers = defaultReceivers
	}

	receivers, err := receiver.New(cfgReceivers, d, middleware, cfg.RetryAfterOnResourceExhausted, loggingLevel, reg)
	if err != nil {
		return nil, err
	}
	subservices = append(subservices, receivers)

	if cfg.KafkaWritePathEnabled {
		client, err := ingest.NewWriterClient(cfg.KafkaConfig, 10, logger, prometheus.WrapRegistererWithPrefix("tempo_distributor_", reg))
		if err != nil {
			return nil, fmt.Errorf("failed to create kafka writer client: %w", err)
		}
		d.kafkaProducer = ingest.NewProducer(client, d.cfg.KafkaConfig.ProducerMaxBufferedBytes, reg)
	}

	d.subservices, err = services.NewManager(subservices...)
	if err != nil {
		return nil, fmt.Errorf("failed to create subservices: %w", err)
	}
	d.subservicesWatcher = services.NewFailureWatcher()
	d.subservicesWatcher.WatchManager(d.subservices)

	d.Service = services.NewBasicService(d.starting, d.running, d.stopping)
	return d, nil
}

func (d *Distributor) starting(ctx context.Context) error {
	// Only report success if all sub-services start properly
	err := services.StartManagerAndAwaitHealthy(ctx, d.subservices)
	if err != nil {
		return fmt.Errorf("failed to start subservices: %w", err)
	}

	return nil
}

func (d *Distributor) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-d.subservicesWatcher.Chan():
		return fmt.Errorf("distributor subservices failed: %w", err)
	}
}

// Called after distributor is asked to stop via StopAsync.
func (d *Distributor) stopping(_ error) error {
	return services.StopManagerAndAwaitStopped(context.Background(), d.subservices)
}

// checkForRateLimits checks if the trace batch size exceeds the ingestion rate limit.
// it will use the ingestion rate limits based on the ingestion strategy configured.
//
// LocalIngestionRateStrategy: the ingestion rate limit is applied as is in each distributor.
// example: if the ingestion rate limit is 10MB/s and the burst size is 20MB, then each distributor
// will allow 10MB/s with a burst of 20MB.
//
// GlobalIngestionRateStrategy: the ingestion rate limit is divided by the number of healthy distributors.
// example: if the ingestion rate limit is 10MB/s and the burst size is 20MB, and there are 5 healthy distributors,
// then each distributor will allow 2MB/s with a burst of 20MB.
func (d *Distributor) checkForRateLimits(tracesSize, spanCount int, userID string) error {
	now := time.Now()
	if !d.ingestionRateLimiter.AllowN(now, userID, tracesSize) {
		overrides.RecordDiscardedSpans(spanCount, reasonRateLimited, userID)
		// limit: number of bytes per second allowed for the user, as per ingestion rate strategy
		limit := int(d.ingestionRateLimiter.Limit(now, userID))
		burst := d.ingestionRateLimiter.Burst(now, userID)

		// globalLimit will be 0 when using local ingestion rate strategy
		var globalLimit int
		if d.overrides.IngestionRateStrategy() == overrides.GlobalIngestionRateStrategy {
			// note: global limit should be calculated using healthy distributors count,
			// but we are using it in logs, instance count is good enough.
			globalLimit = limit * d.DistributorRing.InstancesCount()
		}

		// batch size is too big if it's more than the limit and burst both
		if tracesSize > limit && tracesSize > burst {
			return status.Errorf(codes.ResourceExhausted,
				"%s: batch size (%d bytes) exceeds ingestion limit (local: %d bytes/s, global: %d bytes/s, burst: %d bytes) while adding %d bytes for user %s. consider reducing batch size or increasing rate limit.",
				overrides.ErrorPrefixRateLimited, tracesSize, limit, globalLimit, burst, tracesSize, userID)
		}

		return status.Errorf(codes.ResourceExhausted,
			"%s: ingestion rate limit (local: %d bytes/s, global: %d bytes/s, burst: %d bytes) exceeded while adding %d bytes for user %s. consider increasing the limit or reducing ingestion rate.",
			overrides.ErrorPrefixRateLimited, limit, globalLimit, burst, tracesSize, userID)
	}

	return nil
}

func (d *Distributor) extractBasicInfo(ctx context.Context, traces ptrace.Traces) (userID string, spanCount, tracesSize int, err error) {
	user, e := user.ExtractOrgID(ctx)
	if e != nil {
		return "", 0, 0, e
	}

	return user, traces.SpanCount(), (&ptrace.ProtoMarshaler{}).TracesSize(traces), nil
}

// PushTraces pushes a batch of traces
func (d *Distributor) PushTraces(ctx context.Context, traces ptrace.Traces) (*tempopb.PushResponse, error) {
	reqStart := time.Now()

	ctx, span := tracer.Start(ctx, "distributor.PushBytes")
	defer span.End()

	userID, spanCount, size, err := d.extractBasicInfo(ctx, traces)
	if err != nil {
		// can't record discarded spans here b/c there's no tenant
		return nil, err
	}
	defer d.padWithArtificialDelay(reqStart, userID)

	if spanCount == 0 {
		return &tempopb.PushResponse{}, nil
	}

	// check limits
	// todo - usage tracker include discarded bytes?
	err = d.checkForRateLimits(size, spanCount, userID)
	if err != nil {
		return nil, err
	}

	// Convert to bytes and back. This is unfortunate for efficiency, but it works
	// around the otel-collector internalization of otel-proto which Tempo also uses.
	convert, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(traces)
	if err != nil {
		return nil, err
	}

	// tempopb.Trace is wire-compatible with ExportTraceServiceRequest
	// used by ToOtlpProtoBytes
	trace := tempopb.Trace{}
	err = trace.Unmarshal(convert)
	if err != nil {
		return nil, err
	}

	batches := trace.ResourceSpans

	logReceivedSpans(batches, &d.cfg.LogReceivedSpans, d.logger)
	if d.cfg.MetricReceivedSpans.Enabled {
		metricSpans(batches, userID, &d.cfg.MetricReceivedSpans)
	}

	metricBytesIngested.WithLabelValues(userID).Add(float64(size))
	metricSpansIngested.WithLabelValues(userID).Add(float64(spanCount))

	statBytesReceived.Inc(int64(size))
	statSpansReceived.Inc(int64(spanCount))

	// Usage tracking
	if d.usage != nil {
		d.usage.Observe(userID, batches)
	}

	maxAttributeBytes := d.getMaxAttributeBytes(userID)

	ringTokens, rebatchedTraces, truncatedAttributeCount, err := requestsByTraceID(batches, userID, spanCount, maxAttributeBytes)
	if err != nil {
		logDiscardedResourceSpans(batches, userID, &d.cfg.LogDiscardedSpans, d.logger)
		return nil, err
	}

	if truncatedAttributeCount > 0 {
		metricAttributesTruncated.WithLabelValues(userID).Add(float64(truncatedAttributeCount))
	}

	err = d.sendToIngestersViaBytes(ctx, userID, rebatchedTraces, ringTokens)
	if err != nil {
		return nil, err
	}

	if err := d.forwardersManager.ForTenant(userID).ForwardTraces(ctx, traces); err != nil {
		_ = level.Warn(d.logger).Log("msg", "failed to forward batches for tenant=%s: %w", userID, err)
	}

	if d.kafkaProducer != nil {
		err := d.sendToKafka(ctx, userID, ringTokens, rebatchedTraces)
		if err != nil {
			level.Error(d.logger).Log("msg", "failed to write to kafka", "err", err)
			return nil, err
		}
	} else {
		// See if we need to send to the generators
		if len(d.overrides.MetricsGeneratorProcessors(userID)) > 0 {
			d.generatorForwarder.SendTraces(ctx, userID, ringTokens, rebatchedTraces)
		}
	}

	return nil, nil // PushRequest is ignored, so no reason to create one
}

func (d *Distributor) sendToIngestersViaBytes(ctx context.Context, userID string, traces []*rebatchedTrace, keys []uint32) error {
	marshalledTraces := make([][]byte, len(traces))
	for i, t := range traces {
		b, err := d.traceEncoder.PrepareForWrite(t.trace, t.start, t.end)
		if err != nil {
			return fmt.Errorf("failed to marshal PushRequest: %w", err)
		}
		marshalledTraces[i] = b
	}

	op := ring.WriteNoExtend
	if d.cfg.ExtendWrites {
		op = ring.Write
	}

	numOfTraces := len(keys)
	numSuccessByTraceIndex := make([]int, numOfTraces)
	lastErrorReasonByTraceIndex := make([]tempopb.PushErrorReason, numOfTraces)

	var mu sync.Mutex

	writeRing := d.ingestersRing.ShuffleShard(userID, d.overrides.IngestionTenantShardSize(userID))

	err := ring.DoBatchWithOptions(ctx, op, writeRing, keys, func(ingester ring.InstanceDesc, indexes []int) error {
		localCtx, cancel := context.WithTimeout(ctx, d.clientCfg.RemoteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		req := tempopb.PushBytesRequest{
			Traces: make([]tempopb.PreallocBytes, len(indexes)),
			Ids:    make([][]byte, len(indexes)),
		}

		for i, j := range indexes {
			req.Traces[i].Slice = marshalledTraces[j][0:]
			req.Ids[i] = traces[j].id
		}

		c, err := d.pool.GetClientFor(ingester.Addr)
		if err != nil {
			return err
		}

		pushResponse, err := c.(tempopb.PusherClient).PushBytesV2(localCtx, &req)
		metricIngesterAppends.WithLabelValues(ingester.Addr).Inc()

		if err != nil { // internal error, drop entire batch
			metricIngesterAppendFailures.WithLabelValues(ingester.Addr).Inc()
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		d.processPushResponse(pushResponse, numSuccessByTraceIndex, lastErrorReasonByTraceIndex, numOfTraces, indexes)

		return nil
	}, ring.DoBatchOptions{})
	// if err != nil, we discarded everything because of an internal error (like "context cancelled")
	if err != nil {
		logDiscardedRebatchedSpans(traces, userID, &d.cfg.LogDiscardedSpans, d.logger)
		return err
	}

	// count discarded span count
	mu.Lock()
	defer mu.Unlock()
	recordDiscardedSpans(numSuccessByTraceIndex, lastErrorReasonByTraceIndex, traces, writeRing, userID)
	logDiscardedSpans(numSuccessByTraceIndex, lastErrorReasonByTraceIndex, traces, writeRing, userID, &d.cfg.LogDiscardedSpans, d.logger)

	return nil
}

func (d *Distributor) sendToGenerators(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace, noGenerateMetrics bool) error {
	// If an instance is unhealthy write to the next one (i.e. write extend is enabled)
	op := ring.Write

	readRing := d.generatorsRing.ShuffleShard(userID, d.overrides.MetricsGeneratorRingSize(userID))

	err := ring.DoBatchWithOptions(ctx, op, readRing, keys, func(generator ring.InstanceDesc, indexes []int) error {
		localCtx, cancel := context.WithTimeout(ctx, d.generatorClientCfg.RemoteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		req := tempopb.PushSpansRequest{
			Batches:               nil,
			SkipMetricsGeneration: noGenerateMetrics,
		}
		for _, j := range indexes {
			req.Batches = append(req.Batches, traces[j].trace.ResourceSpans...)
		}

		c, err := d.generatorsPool.GetClientFor(generator.Addr)
		if err != nil {
			return fmt.Errorf("failed to get client for generator: %w", err)
		}

		_, err = c.(tempopb.MetricsGeneratorClient).PushSpans(localCtx, &req)
		metricGeneratorPushes.WithLabelValues(generator.Addr).Inc()
		if err != nil {
			metricGeneratorPushesFailures.WithLabelValues(generator.Addr).Inc()
			return fmt.Errorf("failed to push spans to generator: %w", err)
		}
		return nil
	}, ring.DoBatchOptions{})

	return err
}

// Check implements the grpc healthcheck
func (*Distributor) Check(_ context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (d *Distributor) UsageTrackerHandler() http.Handler {
	if d.usage != nil {
		return d.usage.Handler()
	}

	return nil
}

func (d *Distributor) sendToKafka(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) error {
	marshalledTraces := make([][]byte, len(traces))
	for i, t := range traces {
		b, err := proto.Marshal(t.trace)
		if err != nil {
			return fmt.Errorf("failed to marshal trace: %w", err)
		}
		marshalledTraces[i] = b
	}

	partitionRing, err := d.partitionRing.PartitionRing().ShuffleShard(userID, d.overrides.IngestionTenantShardSize(userID))
	if err != nil {
		return fmt.Errorf("failed to shuffle shard: %w", err)
	}
	return ring.DoBatchWithOptions(ctx, ring.Write, ring.NewActivePartitionBatchRing(partitionRing), keys, func(partition ring.InstanceDesc, indexes []int) error {
		localCtx, cancel := context.WithTimeout(ctx, d.clientCfg.RemoteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		req := &tempopb.PushBytesRequest{
			Traces:                make([]tempopb.PreallocBytes, len(indexes)),
			Ids:                   make([][]byte, len(indexes)),
			SkipMetricsGeneration: generator.ExtractNoGenerateMetrics(ctx),
		}

		for i, j := range indexes {
			req.Traces[i].Slice = marshalledTraces[j][0:]
			req.Ids[i] = traces[j].id
		}

		// The partition ID is stored in the ring.InstanceDesc ID.
		partitionID, err := strconv.ParseInt(partition.Id, 10, 32)
		if err != nil {
			return err
		}

		records, err := ingest.Encode(int32(partitionID), userID, req, d.cfg.KafkaConfig.ProducerMaxRecordSizeBytes)
		if err != nil {
			return fmt.Errorf("failed to encode PushSpansRequest: %w", err)
		}

		metricKafkaRecordsPerRequest.Observe(float64(len(records)))

		startTime := time.Now()
		produceResults := d.kafkaProducer.ProduceSync(localCtx, records)
		metricKafkaWriteLatency.Observe(time.Since(startTime).Seconds())

		partitionLabel := fmt.Sprintf("partition_%d", partitionID)
		count := 0
		sizeBytes := 0
		for _, result := range produceResults {
			if result.Err != nil {
				_ = level.Error(d.logger).Log("msg", "failed to write to kafka", "err", result.Err)
				metricKafkaAppends.WithLabelValues(partitionLabel, "fail").Inc()
			} else {
				count++
				sizeBytes += len(result.Record.Value)
			}
		}

		if count > 0 {
			metricKafkaWriteBytesTotal.WithLabelValues(partitionLabel).Add(float64(sizeBytes))
			metricKafkaAppends.WithLabelValues(partitionLabel, "success").Add(float64(count))
		}

		_ = level.Debug(d.logger).Log("msg", "kafka write success stats", "count", count, "size_bytes", sizeBytes, "partition", partitionLabel)

		return produceResults.FirstErr()
	}, ring.DoBatchOptions{})
}

// requestsByTraceID takes an incoming tempodb.PushRequest and creates a set of keys for the hash ring
// and traces to pass onto the ingesters.
func requestsByTraceID(batches []*v1.ResourceSpans, userID string, spanCount, maxSpanAttrSize int) ([]uint32, []*rebatchedTrace, int, error) {
	const tracesPerBatch = 20 // p50 of internal env
	tracesByID := make(map[uint64]*rebatchedTrace, tracesPerBatch)
	truncatedAttributeCount := 0
	currentTime := uint32(time.Now().Unix())
	for _, b := range batches {
		spansByILS := make(map[uint64]*v1.ScopeSpans)
		// check resource for large attributes
		if maxSpanAttrSize > 0 && b.Resource != nil {
			resourceAttrTruncatedCount := processAttributes(b.Resource.Attributes, maxSpanAttrSize)
			truncatedAttributeCount += resourceAttrTruncatedCount
		}

		for _, ils := range b.ScopeSpans {

			// check instrumentation for large attributes
			if maxSpanAttrSize > 0 && ils.Scope != nil {
				scopeAttrTruncatedCount := processAttributes(ils.Scope.Attributes, maxSpanAttrSize)
				truncatedAttributeCount += scopeAttrTruncatedCount
			}

			for _, span := range ils.Spans {
				// check spans for large attributes
				if maxSpanAttrSize > 0 {
					spanAttrTruncatedCount := processAttributes(span.Attributes, maxSpanAttrSize)
					truncatedAttributeCount += spanAttrTruncatedCount

					// check large attributes for events and links
					for _, event := range span.Events {
						eventAttrTruncatedCount := processAttributes(event.Attributes, maxSpanAttrSize)
						truncatedAttributeCount += eventAttrTruncatedCount
					}

					for _, link := range span.Links {
						linkAttrTruncatedCount := processAttributes(link.Attributes, maxSpanAttrSize)
						truncatedAttributeCount += linkAttrTruncatedCount
					}
				}
				traceID := span.TraceId
				if !validation.ValidTraceID(traceID) {
					return nil, nil, 0, status.Errorf(codes.InvalidArgument, "trace ids must be 128 bit, received %d bits", len(traceID)*8)
				}

				traceKey := util.HashForTraceID(traceID)
				ilsKey := traceKey
				if ils.Scope != nil {
					ilsKey = fnv1a.AddString64(ilsKey, ils.Scope.Name)
					ilsKey = fnv1a.AddString64(ilsKey, ils.Scope.Version)
				}

				existingILS, ilsAdded := spansByILS[ilsKey]
				if !ilsAdded {
					existingILS = &v1.ScopeSpans{
						Scope: ils.Scope,
						Spans: make([]*v1.Span, 0, spanCount/tracesPerBatch),
					}
					spansByILS[ilsKey] = existingILS
				}
				existingILS.Spans = append(existingILS.Spans, span)

				// now find and update the rebatchedTrace with a new start and end
				existingTrace, ok := tracesByID[traceKey]
				if !ok {
					existingTrace = &rebatchedTrace{
						id: traceID,
						trace: &tempopb.Trace{
							ResourceSpans: make([]*v1.ResourceSpans, 0, spanCount/tracesPerBatch),
						},
						start:     math.MaxUint32,
						end:       0,
						spanCount: 0,
					}

					tracesByID[traceKey] = existingTrace
				}

				start, end := startEndFromSpan(span)
				if existingTrace.end < end {
					existingTrace.end = end
				}
				if existingTrace.start > start {
					existingTrace.start = start
				}
				if !ilsAdded {
					existingTrace.trace.ResourceSpans = append(existingTrace.trace.ResourceSpans, &v1.ResourceSpans{
						Resource:   b.Resource,
						ScopeSpans: []*v1.ScopeSpans{existingILS},
					})
				}

				// increase span count for trace
				existingTrace.spanCount = existingTrace.spanCount + 1

				// Count spans with timestamps in the future
				if end > currentTime {
					dataquality.MetricSpanInFuture.WithLabelValues(userID).Observe(float64(end - currentTime))
				} else {
					dataquality.MetricSpanInPast.WithLabelValues(userID).Observe(float64(currentTime - end))
				}
			}
		}
	}

	metricTracesPerBatch.Observe(float64(len(tracesByID)))

	ringTokens := make([]uint32, 0, len(tracesByID))
	traces := make([]*rebatchedTrace, 0, len(tracesByID))

	for _, tr := range tracesByID {
		ringTokens = append(ringTokens, util.TokenFor(userID, tr.id))
		traces = append(traces, tr)
	}

	return ringTokens, traces, truncatedAttributeCount, nil
}

// find and truncate the span attributes that are too large
func processAttributes(attributes []*v1_common.KeyValue, maxAttrSize int) int {
	count := 0
	for _, attr := range attributes {
		if len(attr.Key) > maxAttrSize {
			attr.Key = attr.Key[:maxAttrSize]
			count++
		}

		switch value := attr.GetValue().Value.(type) {
		case *v1_common.AnyValue_StringValue:
			if len(value.StringValue) > maxAttrSize {
				value.StringValue = value.StringValue[:maxAttrSize]
				count++
			}
		default:
			continue
		}
	}

	return count
}

// discardedPredicate determines if a trace is discarded based on the number of successful replications.
type discardedPredicate func(int) bool

func newDiscardedPredicate(repFactor int) discardedPredicate {
	quorum := int(math.Floor(float64(repFactor)/2)) + 1 // min success required
	return func(numSuccess int) bool {
		return numSuccess < quorum
	}
}

func countDiscardedSpans(numSuccessByTraceIndex []int, lastErrorReasonByTraceIndex []tempopb.PushErrorReason, traces []*rebatchedTrace, repFactor int) (maxLiveDiscardedCount, traceTooLargeDiscardedCount, unknownErrorCount int) {
	discarded := newDiscardedPredicate(repFactor)

	for traceIndex, numSuccess := range numSuccessByTraceIndex {
		if !discarded(numSuccess) {
			continue
		}
		spanCount := traces[traceIndex].spanCount
		switch lastErrorReasonByTraceIndex[traceIndex] {
		case tempopb.PushErrorReason_MAX_LIVE_TRACES:
			maxLiveDiscardedCount += spanCount
		case tempopb.PushErrorReason_TRACE_TOO_LARGE:
			traceTooLargeDiscardedCount += spanCount
		case tempopb.PushErrorReason_UNKNOWN_ERROR:
			unknownErrorCount += spanCount
		}
	}

	return maxLiveDiscardedCount, traceTooLargeDiscardedCount, unknownErrorCount
}

func (d *Distributor) processPushResponse(pushResponse *tempopb.PushResponse, numSuccessByTraceIndex []int, lastErrorReasonByTraceIndex []tempopb.PushErrorReason, numOfTraces int, indexes []int) {
	// no errors
	if len(pushResponse.ErrorsByTrace) == 0 {
		for _, reqBatchIndex := range indexes {
			if reqBatchIndex > numOfTraces {
				level.Warn(d.logger).Log("msg", fmt.Sprintf("batch index %d out of bound for length %d", reqBatchIndex, numOfTraces))
				continue
			}
			numSuccessByTraceIndex[reqBatchIndex]++
		}
		return
	}

	for ringIndex, pushError := range pushResponse.ErrorsByTrace {
		// translate index of ring batch and req batch
		// since the request batch gets split up into smaller batches based on the indexes
		// like [0,1] [1] [2] [0,2]
		reqBatchIndex := indexes[ringIndex]
		if reqBatchIndex > numOfTraces {
			level.Warn(d.logger).Log("msg", fmt.Sprintf("batch index %d out of bound for length %d", reqBatchIndex, numOfTraces))
			continue
		}

		// if no error, record number of success
		if pushError == tempopb.PushErrorReason_NO_ERROR {
			numSuccessByTraceIndex[reqBatchIndex]++
			continue
		}
		// else record last error
		lastErrorReasonByTraceIndex[reqBatchIndex] = pushError
	}
}

func metricSpans(batches []*v1.ResourceSpans, tenantID string, cfg *MetricReceivedSpansConfig) {
	for _, b := range batches {
		serviceName := ""
		if b.Resource != nil {
			for _, a := range b.Resource.GetAttributes() {
				if a.GetKey() == "service.name" {
					serviceName = a.Value.GetStringValue()
					break
				}
			}
		}

		for _, ils := range b.ScopeSpans {
			for _, s := range ils.Spans {
				if cfg.RootOnly && len(s.ParentSpanId) != 0 {
					continue
				}

				metricDebugSpansIngested.WithLabelValues(tenantID, s.Name, serviceName).Inc()
			}
		}
	}
}

func recordDiscardedSpans(numSuccessByTraceIndex []int, lastErrorReasonByTraceIndex []tempopb.PushErrorReason, traces []*rebatchedTrace, writeRing ring.ReadRing, userID string) {
	maxLiveDiscardedCount, traceTooLargeDiscardedCount, unknownErrorCount := countDiscardedSpans(numSuccessByTraceIndex, lastErrorReasonByTraceIndex, traces, writeRing.ReplicationFactor())
	overrides.RecordDiscardedSpans(maxLiveDiscardedCount, reasonLiveTracesExceeded, userID)
	overrides.RecordDiscardedSpans(traceTooLargeDiscardedCount, reasonTraceTooLarge, userID)
	overrides.RecordDiscardedSpans(unknownErrorCount, reasonUnknown, userID)
}

func logDiscardedSpans(numSuccessByTraceIndex []int, lastErrorReasonByTraceIndex []tempopb.PushErrorReason, traces []*rebatchedTrace, writeRing ring.ReadRing, userID string, cfg *LogSpansConfig, logger log.Logger) {
	if !cfg.Enabled {
		return
	}
	discarded := newDiscardedPredicate(writeRing.ReplicationFactor())
	for traceIndex, numSuccess := range numSuccessByTraceIndex {
		if !discarded(numSuccess) {
			continue
		}
		errorReason := lastErrorReasonByTraceIndex[traceIndex]
		if errorReason != tempopb.PushErrorReason_NO_ERROR {
			loggerWithAtts := logger
			loggerWithAtts = log.With(
				loggerWithAtts,
				"push_error_reason", fmt.Sprintf("%v", errorReason),
			)
			logDiscardedResourceSpans(traces[traceIndex].trace.ResourceSpans, userID, cfg, loggerWithAtts)
		}
	}
}

func logDiscardedRebatchedSpans(batches []*rebatchedTrace, userID string, cfg *LogSpansConfig, logger log.Logger) {
	if !cfg.Enabled {
		return
	}
	for _, b := range batches {
		logDiscardedResourceSpans(b.trace.ResourceSpans, userID, cfg, logger)
	}
}

func logDiscardedResourceSpans(batches []*v1.ResourceSpans, userID string, cfg *LogSpansConfig, logger log.Logger) {
	if !cfg.Enabled {
		return
	}
	loggerWithAtts := logger
	loggerWithAtts = log.With(
		loggerWithAtts,
		"msg", "discarded",
		"tenant", userID,
	)
	logSpans(batches, cfg, loggerWithAtts)
}

func logReceivedSpans(batches []*v1.ResourceSpans, cfg *LogSpansConfig, logger log.Logger) {
	if !cfg.Enabled {
		return
	}
	loggerWithAtts := logger
	loggerWithAtts = log.With(
		loggerWithAtts,
		"msg", "received",
	)
	logSpans(batches, cfg, loggerWithAtts)
}

func logSpans(batches []*v1.ResourceSpans, cfg *LogSpansConfig, logger log.Logger) {
	for _, b := range batches {
		loggerWithAtts := logger

		if cfg.IncludeAllAttributes {
			for _, a := range b.Resource.GetAttributes() {
				loggerWithAtts = log.With(
					loggerWithAtts,
					"span_"+strutil.SanitizeLabelName(a.GetKey()),
					util.StringifyAnyValue(a.GetValue()))
			}
		}

		for _, ils := range b.ScopeSpans {
			for _, s := range ils.Spans {
				if cfg.FilterByStatusError && s.Status.Code != v1.Status_STATUS_CODE_ERROR {
					continue
				}

				logSpan(s, cfg.IncludeAllAttributes, loggerWithAtts)
			}
		}
	}
}

func logSpan(s *v1.Span, allAttributes bool, logger log.Logger) {
	if allAttributes {
		for _, a := range s.GetAttributes() {
			logger = log.With(
				logger,
				"span_"+strutil.SanitizeLabelName(a.GetKey()),
				util.StringifyAnyValue(a.GetValue()))
		}

		latencySeconds := float64(s.GetEndTimeUnixNano()-s.GetStartTimeUnixNano()) / float64(time.Second.Nanoseconds())
		logger = log.With(
			logger,
			"span_name", s.Name,
			"span_duration_seconds", latencySeconds,
			"span_kind", s.GetKind().String(),
			"span_status", s.GetStatus().GetCode().String())
	}

	level.Info(logger).Log("spanid", hex.EncodeToString(s.SpanId), "traceid", hex.EncodeToString(s.TraceId))
}

// startEndFromSpan returns a unix epoch timestamp in seconds for the start and end of a span
func startEndFromSpan(span *v1.Span) (uint32, uint32) {
	return uint32(span.StartTimeUnixNano / uint64(time.Second)), uint32(span.EndTimeUnixNano / uint64(time.Second))
}

func (d *Distributor) getMaxAttributeBytes(userID string) int {
	if tenantMaxAttrByte := d.overrides.IngestionMaxAttributeBytes(userID); tenantMaxAttrByte > 0 {
		return tenantMaxAttrByte
	}

	return d.cfg.MaxAttributeBytes
}
