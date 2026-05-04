package distributor

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/grafana/dskit/limiter"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/util/strutil"
	"github.com/segmentio/fasthash/fnv1a"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/distributor/receiver"
	"github.com/grafana/tempo/modules/distributor/usage"
	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/dataquality"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/pkg/util"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
)

const (
	distributorRingKey = "distributor"

	truncationLogsPerSecond = 1
)

var (
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
		Help:      "The total number of proto bytes received per tenant, after limits",
	}, []string{"tenant"})
	metricIngressBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_ingress_bytes_total",
		Help:      "The total number of bytes received per tenant, before limits",
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
	metricAttributesTruncated = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_attributes_truncated_total",
		Help:      "The total number of attribute keys or values truncated per tenant and scope",
	}, []string{"tenant", "scope"})
	metricKafkaRecordsPerRequest = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Subsystem:                       "distributor",
		Name:                            "kafka_records_per_request",
		Help:                            "The number of records in each kafka request",
		Buckets:                         prometheus.DefBuckets,
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
	metricKafkaWriteLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Subsystem:                       "distributor",
		Name:                            "kafka_write_latency_seconds",
		Help:                            "The latency of writing to kafka",
		Buckets:                         prometheus.DefBuckets,
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
	metricKafkaWriteBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "kafka_write_bytes_total",
		Help:      "The total number of bytes written to kafka",
	}, []string{"partition"})

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

// PushSpansFunc is a callback used to push spans to a local in-process consumer
// without gRPC/ring indirection (single-binary mode).
type PushSpansFunc func(ctx context.Context, req *tempopb.PushSpansRequest) (*tempopb.PushResponse, error)

// PushBytesFunc is a callback used to push pre-marshaled traces to a local
// in-process consumer without Kafka indirection (single-binary mode).
type PushBytesFunc func(ctx context.Context, req *tempopb.PushBytesRequest) (*tempopb.PushResponse, error)

// LocalPushTargets contains optional local in-process push callbacks.
type LocalPushTargets struct {
	Generator PushSpansFunc
	LiveStore PushBytesFunc
}

type truncatedAttributesCount struct {
	Resource int
	Scope    int
	Span     int
	Event    int
	Link     int
}

func (c truncatedAttributesCount) Total() int {
	return c.Resource + c.Scope + c.Span + c.Event + c.Link
}

type truncatedAttrInfo struct {
	scope    string
	name     string
	field    string // "key" or "value"
	origSize int    // original byte length before truncation; 0 means no example captured yet
}

// Distributor coordinates replicates and distribution of log streams.
type Distributor struct {
	services.Service

	cfg             Config
	DistributorRing *ring.Ring
	overrides       overrides.Interface

	// Local in-process push targets used in single-binary mode.
	localPushTargets   LocalPushTargets
	generatorForwarder *generatorForwarder

	// Generic Forwarder
	forwardersManager *forwarder.Manager

	// Kafka
	kafkaProducer *ingest.Producer
	partitionRing ring.PartitionRingReader

	pushSpansToKafka bool

	// Per-user rate limiter.
	ingestionRateLimiter *limiter.RateLimiter

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	usage *usage.Tracker

	logger log.Logger

	// TracePushMiddlewares are hooks called when a trace push request is received.
	// Middleware errors are logged but don't fail the push (fail open behavior).
	tracePushMiddlewares []TracePushMiddleware

	truncationLogger *tempo_log.RateLimitedLogger

	// For testing functionality that relies on timing without having to sleep in unit tests.
	sleep func(time.Duration)
	now   func() time.Time
}

// New a distributor creates.
func New(
	cfg Config,
	localPushTargets LocalPushTargets,
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

	pushSpansToKafka := cfg.PushSpansToKafka

	d := &Distributor{
		cfg:                  cfg,
		DistributorRing:      distributorRing,
		ingestionRateLimiter: limiter.NewRateLimiter(ingestionRateStrategy, 10*time.Second),
		localPushTargets:     localPushTargets,
		partitionRing:        partitionRing,
		pushSpansToKafka:     pushSpansToKafka,
		overrides:            o,
		tracePushMiddlewares: cfg.TracePushMiddlewares,
		truncationLogger:     tempo_log.NewRateLimitedLogger(truncationLogsPerSecond, level.Warn(logger)),
		logger:               logger,
		sleep:                time.Sleep,
		now:                  time.Now,
	}

	if cfg.Usage.CostAttribution.Enabled {
		tracker, err := usage.NewTracker(cfg.Usage.CostAttribution, "cost-attribution", o.CostAttributionDimensions, o.CostAttributionMaxCardinality, logger)
		if err != nil {
			return nil, fmt.Errorf("creating usage tracker: %w", err)
		}
		d.usage = tracker
	}

	if d.localPushTargets.Generator != nil {
		d.generatorForwarder = newGeneratorForwarder(logger, d.sendToGenerators, o)
		subservices = append(subservices, d.generatorForwarder)
	}

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

	if d.pushSpansToKafka {
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
		overrides.RecordDiscardedSpans(spanCount, overrides.ReasonRateLimited, userID)
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
	orgID, e := validation.ExtractValidTenantID(ctx)
	if e != nil {
		return "", 0, 0, status.Error(codes.InvalidArgument, e.Error())
	}

	return orgID, traces.SpanCount(), (&ptrace.ProtoMarshaler{}).TracesSize(traces), nil
}

// PushTraces pushes a batch of traces
func (d *Distributor) PushTraces(ctx context.Context, traces ptrace.Traces) (*tempopb.PushResponse, error) {
	reqStart := time.Now()

	ctx, span := tracer.Start(ctx, "distributor.PushTraces")
	defer span.End()

	// Call trace push middlewares
	for _, mw := range d.tracePushMiddlewares {
		if err := mw(ctx, traces); err != nil {
			_ = level.Warn(d.logger).Log("msg", "trace push middleware failed", "err", err)
		}
	}

	userID, spanCount, size, err := d.extractBasicInfo(ctx, traces)
	if err != nil {
		// can't record discarded spans here b/c there's no tenant
		return nil, err
	}
	span.SetAttributes(attribute.String("orgID", userID))
	defer d.padWithArtificialDelay(reqStart, userID)
	metricIngressBytes.WithLabelValues(userID).Add(float64(size))

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

	ringTokens, rebatchedTraces, truncatedAttributesCount, truncationExample, err := requestsByTraceID(batches, userID, spanCount, maxAttributeBytes)
	if err != nil {
		logDiscardedResourceSpans(batches, userID, &d.cfg.LogDiscardedSpans, d.logger)
		return nil, err
	}

	if truncatedAttributesCount.Total() > 0 {
		metricAttributesTruncated.WithLabelValues(userID, "resource").Add(float64(truncatedAttributesCount.Resource))
		metricAttributesTruncated.WithLabelValues(userID, "scope").Add(float64(truncatedAttributesCount.Scope))
		metricAttributesTruncated.WithLabelValues(userID, "span").Add(float64(truncatedAttributesCount.Span))
		metricAttributesTruncated.WithLabelValues(userID, "event").Add(float64(truncatedAttributesCount.Event))
		metricAttributesTruncated.WithLabelValues(userID, "link").Add(float64(truncatedAttributesCount.Link))

		if truncationExample != nil {
			d.truncationLogger.Log("msg", "attributes truncated",
				"tenant", userID,
				"total_truncated", truncatedAttributesCount.Total(),
				"max_size_bytes", maxAttributeBytes,
				"example_scope", truncationExample.scope,
				"example_name", truncationExample.name,
				"example_field", truncationExample.field,
				"example_orig_size", truncationExample.origSize)
		}
	}

	if err := d.forwardersManager.ForTenant(userID).ForwardTraces(ctx, traces); err != nil {
		_ = level.Warn(d.logger).Log("msg", "failed to forward batches for tenant=%s: %w", userID, err)
	}

	if d.pushSpansToKafka {
		if err := d.pushTracesKafka(ctx, userID, ringTokens, rebatchedTraces); err != nil {
			level.Error(d.logger).Log("msg", "failed to write to kafka", "err", err, "tenant", userID)
			return nil, err
		}
	} else {
		if err := d.pushLocal(ctx, userID, ringTokens, rebatchedTraces); err != nil {
			level.Error(d.logger).Log("msg", "failed to push to local consumers", "err", err, "tenant", userID)
			return nil, err
		}
	}

	return nil, nil // PushRequest is ignored, so no reason to create one
}

func (d *Distributor) pushTracesKafka(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) error {
	skipMetricsGeneration := generator.ExtractNoGenerateMetrics(ctx)
	return d.sendToKafka(ctx, userID, keys, traces, skipMetricsGeneration)
}

func (d *Distributor) pushLocal(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) error {
	if err := d.pushTracesToLiveStore(ctx, userID, traces); err != nil {
		return err
	}

	d.pushTracesToGenerator(ctx, userID, keys, traces)
	return nil
}

func (d *Distributor) pushTracesToGenerator(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) {
	if d.localPushTargets.Generator == nil {
		return
	}
	if len(d.overrides.MetricsGeneratorProcessors(userID)) > 0 {
		d.generatorForwarder.SendTraces(ctx, userID, keys, traces)
	}
}

func (d *Distributor) pushTracesToLiveStore(ctx context.Context, userID string, traces []*rebatchedTrace) error {
	if d.localPushTargets.LiveStore == nil {
		return nil
	}

	req := &tempopb.PushBytesRequest{
		Traces: make([]tempopb.PreallocBytes, len(traces)),
		Ids:    make([][]byte, len(traces)),
	}
	for i, tr := range traces {
		b, err := proto.Marshal(tr.trace)
		if err != nil {
			return fmt.Errorf("failed to marshal trace for local live-store push: %w", err)
		}
		req.Traces[i].Slice = b
		req.Ids[i] = tr.id
	}

	localCtx := user.InjectOrgID(ctx, userID)
	_, err := d.localPushTargets.LiveStore(localCtx, req)
	if err != nil {
		return fmt.Errorf("failed to push spans to local live-store: %w", err)
	}
	return nil
}

func (d *Distributor) sendToGenerators(ctx context.Context, userID string, _ []uint32, traces []*rebatchedTrace, noGenerateMetrics bool) error {
	req := tempopb.PushSpansRequest{
		Batches:               nil,
		SkipMetricsGeneration: noGenerateMetrics,
	}
	for _, tr := range traces {
		req.Batches = append(req.Batches, tr.trace.ResourceSpans...)
	}

	localCtx := user.InjectOrgID(ctx, userID)
	_, err := d.localPushTargets.Generator(localCtx, &req)
	metricGeneratorPushes.WithLabelValues("local").Inc()
	if err != nil {
		metricGeneratorPushesFailures.WithLabelValues("local").Inc()
		return fmt.Errorf("failed to push spans to local generator: %w", err)
	}
	return nil
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

func (d *Distributor) sendToKafka(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace, skipMetricsGeneration bool) error {
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
		localCtx, cancel := context.WithTimeout(ctx, d.cfg.KafkaConfig.WriteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		req := &tempopb.PushBytesRequest{
			Traces:                make([]tempopb.PreallocBytes, len(indexes)),
			Ids:                   make([][]byte, len(indexes)),
			SkipMetricsGeneration: skipMetricsGeneration,
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
				_ = level.Error(d.logger).Log("msg", "failed to write to kafka", "err", result.Err, "tenant", userID)
			} else {
				count++
				sizeBytes += len(result.Record.Value)
			}
		}

		if count > 0 {
			metricKafkaWriteBytesTotal.WithLabelValues(partitionLabel).Add(float64(sizeBytes))
		}

		_ = level.Debug(d.logger).Log("msg", "kafka write success stats", "count", count, "size_bytes", sizeBytes, "partition", partitionLabel)

		return produceResults.FirstErr()
	}, ring.DoBatchOptions{})
}

// requestsByTraceID groups ResourceSpans by trace ID, producing hash-ring tokens and
// rebatched traces for downstream write-path processing. It truncates oversized attributes
// and returns the first truncation example (if any) for diagnostic logging.
func requestsByTraceID(batches []*v1.ResourceSpans, userID string, spanCount, maxSpanAttrSize int) ([]uint32, []*rebatchedTrace, truncatedAttributesCount, *truncatedAttrInfo, error) {
	const tracesPerBatch = 20 // p50 of internal env
	tracesByID := make(map[uint64]*rebatchedTrace, tracesPerBatch)
	truncatedCount := truncatedAttributesCount{}

	// truncationExample captures one example of a truncated attribute for rate-limited logging.
	var truncationExample truncatedAttrInfo

	currentTime := uint32(time.Now().Unix())
	for _, b := range batches {
		spansByILS := make(map[uint64]*v1.ScopeSpans)
		// check resource for large attributes
		if maxSpanAttrSize > 0 && b.Resource != nil {
			truncatedCount.Resource += processAttributes(b.Resource.Attributes, maxSpanAttrSize, &truncationExample, "resource")
		}

		for _, ils := range b.ScopeSpans {

			// check instrumentation for large attributes
			if maxSpanAttrSize > 0 && ils.Scope != nil {
				truncatedCount.Scope += processAttributes(ils.Scope.Attributes, maxSpanAttrSize, &truncationExample, "scope")
			}

			for _, span := range ils.Spans {
				// check spans for large attributes
				if maxSpanAttrSize > 0 {
					truncatedCount.Span += processAttributes(span.Attributes, maxSpanAttrSize, &truncationExample, "span")

					// check large attributes for events and links
					for _, event := range span.Events {
						truncatedCount.Event += processAttributes(event.Attributes, maxSpanAttrSize, &truncationExample, "event")
					}

					for _, link := range span.Links {
						truncatedCount.Link += processAttributes(link.Attributes, maxSpanAttrSize, &truncationExample, "link")
					}
				}
				traceID := span.TraceId
				if !validation.ValidTraceID(traceID) {
					overrides.RecordDiscardedSpans(spanCount, overrides.ReasonInvalidTraceID, userID)
					return nil, nil, truncatedAttributesCount{}, nil, status.Errorf(codes.InvalidArgument, "trace ids must be 128 bit, received %d bits", len(traceID)*8)
				}

				if !validation.ValidSpanID(span.SpanId) {
					overrides.RecordDiscardedSpans(spanCount, overrides.ReasonInvalidSpanID, userID)
					return nil, nil, truncatedAttributesCount{}, nil, status.Errorf(codes.InvalidArgument, "span ids must be 64 bit and not all zero, received %d bits", len(span.SpanId)*8)
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
				existingTrace.spanCount++

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

	if truncationExample.origSize > 0 {
		return ringTokens, traces, truncatedCount, &truncationExample, nil
	}
	return ringTokens, traces, truncatedCount, nil, nil
}

// processAttributes finds and truncates attribute keys/values that exceed maxAttrSize.
func processAttributes(attributes []*v1_common.KeyValue, maxAttrSize int, truncationExample *truncatedAttrInfo, scope string) int {
	count := 0
	for _, attr := range attributes {
		if len(attr.Key) > maxAttrSize {
			origSize := len(attr.Key)
			attr.Key = attr.Key[:maxAttrSize]
			if truncationExample != nil && truncationExample.origSize == 0 { // only capture the first truncation
				// name is the truncated prefix; origSize records the full original length.
				*truncationExample = truncatedAttrInfo{scope: scope, name: attr.Key, field: "key", origSize: origSize}
			}
			count++
		}

		switch value := attr.GetValue().Value.(type) {
		case *v1_common.AnyValue_StringValue:
			if len(value.StringValue) > maxAttrSize {
				if truncationExample != nil && truncationExample.origSize == 0 { // only capture the first truncation
					*truncationExample = truncatedAttrInfo{scope: scope, name: attr.Key, field: "value", origSize: len(value.StringValue)}
				}
				value.StringValue = value.StringValue[:maxAttrSize]
				count++
			}
		default:
			continue
		}
	}

	return count
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

func (d *Distributor) RetryInfoEnabled(ctx context.Context) (bool, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return false, err
	}

	// if disabled at cluster level, just return false
	if d.cfg.RetryAfterOnResourceExhausted <= 0 {
		return false, nil
	}

	// cluster level is enabled, check per-tenant override and respect that.
	return d.overrides.IngestionRetryInfoEnabled(userID), nil
}

// TracePushMiddleware is a hook called when a trace push request is received.
// Middlewares are invoked after the request is decoded but before it's processed.
// Errors returned by middleware are logged but don't fail the push (fail open behavior).
type TracePushMiddleware func(ctx context.Context, td ptrace.Traces) error
