package distributor

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/grafana/dskit/limiter"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/util/strutil"
	"github.com/segmentio/fasthash/fnv1a"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/distributor/receiver"
	generator_client "github.com/grafana/tempo/modules/generator/client"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"

	"github.com/grafana/tempo/pkg/validation"
)

const (
	// reasonRateLimited indicates that the tenants spans/second exceeded their limits
	reasonRateLimited = "rate_limited"
	// reasonTraceTooLarge indicates that a single trace has too many spans
	reasonTraceTooLarge = "trace_too_large"
	// reasonLiveTracesExceeded indicates that tempo is already tracking too many live traces in the ingesters for this user
	reasonLiveTracesExceeded = "live_traces_exceeded"
	// reasonInternalError indicates an unexpected error occurred processing these spans. analogous to a 500
	reasonInternalError = "internal_error"
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
		Namespace: "tempo",
		Name:      "distributor_traces_per_batch",
		Help:      "The number of traces in each batch",
		Buckets:   prometheus.ExponentialBuckets(2, 2, 10),
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
)

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

	// Per-user rate limiter.
	ingestionRateLimiter *limiter.RateLimiter

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	logger log.Logger
}

// New a distributor creates.
func New(cfg Config, clientCfg ingester_client.Config, ingestersRing ring.ReadRing, generatorClientCfg generator_client.Config, generatorsRing ring.ReadRing, o overrides.Interface, middleware receiver.Middleware, logger log.Logger, loggingLevel dslog.Level, reg prometheus.Registerer) (*Distributor, error) {
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
		overrides:            o,
		traceEncoder:         model.MustNewSegmentDecoder(model.CurrentEncoding),
		logger:               logger,
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

	receivers, err := receiver.New(cfgReceivers, d, middleware, cfg.RetryAfterOnResourceExhausted, loggingLevel)
	if err != nil {
		return nil, err
	}
	subservices = append(subservices, receivers)

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

func (d *Distributor) checkForRateLimits(tracesSize, spanCount int, userID string) error {
	now := time.Now()
	if !d.ingestionRateLimiter.AllowN(now, userID, tracesSize) {
		overrides.RecordDiscardedSpans(spanCount, reasonRateLimited, userID)
		return status.Errorf(codes.ResourceExhausted,
			"%s: ingestion rate limit (%d bytes) exceeded while adding %d bytes for user %s",
			overrides.ErrorPrefixRateLimited,
			int(d.ingestionRateLimiter.Limit(now, userID)),
			tracesSize, userID)
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "distributor.PushTraces")
	defer span.Finish()

	userID, spanCount, size, err := d.extractBasicInfo(ctx, traces)
	if err != nil {
		// can't record discarded spans here b/c there's no tenant
		return nil, err
	}
	if spanCount == 0 {
		return &tempopb.PushResponse{}, nil
	}
	// check limits
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

	batches := trace.Batches

	if d.cfg.LogReceivedSpans.Enabled {
		logSpans(batches, &d.cfg.LogReceivedSpans, d.logger)
	}
	if d.cfg.MetricReceivedSpans.Enabled {
		metricSpans(batches, userID, &d.cfg.MetricReceivedSpans)
	}

	metricBytesIngested.WithLabelValues(userID).Add(float64(size))
	metricSpansIngested.WithLabelValues(userID).Add(float64(spanCount))

	keys, rebatchedTraces, err := requestsByTraceID(batches, userID, spanCount)
	if err != nil {
		overrides.RecordDiscardedSpans(spanCount, reasonInternalError, userID)
		return nil, err
	}

	err = d.sendToIngestersViaBytes(ctx, userID, spanCount, rebatchedTraces, keys)
	if err != nil {
		return nil, err
	}

	if len(d.overrides.MetricsGeneratorProcessors(userID)) > 0 {
		d.generatorForwarder.SendTraces(ctx, userID, keys, rebatchedTraces)
	}

	if err := d.forwardersManager.ForTenant(userID).ForwardTraces(ctx, traces); err != nil {
		_ = level.Warn(d.logger).Log("msg", "failed to forward batches for tenant=%s: %w", userID, err)
	}

	return nil, nil // PushRequest is ignored, so no reason to create one
}

func (d *Distributor) sendToIngestersViaBytes(ctx context.Context, userID string, totalSpanCount int, traces []*rebatchedTrace, keys []uint32) error {
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

	err := ring.DoBatch(ctx, op, writeRing, keys, func(ingester ring.InstanceDesc, indexes []int) error {
		localCtx, cancel := context.WithTimeout(ctx, d.clientCfg.RemoteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		req := tempopb.PushBytesRequest{
			Traces:     make([]tempopb.PreallocBytes, len(indexes)),
			Ids:        make([]tempopb.PreallocBytes, len(indexes)),
			SearchData: nil, // support for flatbuffer/v2 search has been removed. todo: cleanup the proto
		}

		for i, j := range indexes {
			req.Traces[i].Slice = marshalledTraces[j][0:]
			req.Ids[i].Slice = traces[j].id
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
	}, func() {})
	// if err != nil, we discarded everything because of an internal error
	if err != nil {
		overrides.RecordDiscardedSpans(totalSpanCount, reasonInternalError, userID)
		return err
	}

	// count discarded span count
	mu.Lock()
	defer mu.Unlock()

	maxLiveDiscardedCount, traceTooLargeDiscardedCount, unknownErrorCount := countDiscaredSpans(numSuccessByTraceIndex, lastErrorReasonByTraceIndex, traces, writeRing.ReplicationFactor())
	overrides.RecordDiscardedSpans(maxLiveDiscardedCount, reasonLiveTracesExceeded, userID)
	overrides.RecordDiscardedSpans(traceTooLargeDiscardedCount, reasonTraceTooLarge, userID)
	overrides.RecordDiscardedSpans(unknownErrorCount, reasonUnknown, userID)

	return nil
}

func (d *Distributor) sendToGenerators(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) error {
	// If an instance is unhealthy write to the next one (i.e. write extend is enabled)
	op := ring.Write

	readRing := d.generatorsRing.ShuffleShard(userID, d.overrides.MetricsGeneratorRingSize(userID))

	err := ring.DoBatch(ctx, op, readRing, keys, func(generator ring.InstanceDesc, indexes []int) error {
		localCtx, cancel := context.WithTimeout(ctx, d.generatorClientCfg.RemoteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		req := tempopb.PushSpansRequest{
			Batches: nil,
		}
		for _, j := range indexes {
			req.Batches = append(req.Batches, traces[j].trace.Batches...)
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
	}, func() {})

	return err
}

// Check implements the grpc healthcheck
func (*Distributor) Check(_ context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

// requestsByTraceID takes an incoming tempodb.PushRequest and creates a set of keys for the hash ring
// and traces to pass onto the ingesters.
func requestsByTraceID(batches []*v1.ResourceSpans, userID string, spanCount int) ([]uint32, []*rebatchedTrace, error) {
	const tracesPerBatch = 20 // p50 of internal env
	tracesByID := make(map[uint32]*rebatchedTrace, tracesPerBatch)

	for _, b := range batches {
		spansByILS := make(map[uint32]*v1.ScopeSpans)

		for _, ils := range b.ScopeSpans {
			for _, span := range ils.Spans {
				traceID := span.TraceId
				if !validation.ValidTraceID(traceID) {
					return nil, nil, status.Errorf(codes.InvalidArgument, "trace ids must be 128 bit")
				}

				traceKey := tempo_util.TokenFor(userID, traceID)
				ilsKey := traceKey
				if ils.Scope != nil {
					ilsKey = fnv1a.AddString32(ilsKey, ils.Scope.Name)
					ilsKey = fnv1a.AddString32(ilsKey, ils.Scope.Version)
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
							Batches: make([]*v1.ResourceSpans, 0, spanCount/tracesPerBatch),
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
					existingTrace.trace.Batches = append(existingTrace.trace.Batches, &v1.ResourceSpans{
						Resource:   b.Resource,
						ScopeSpans: []*v1.ScopeSpans{existingILS},
					})
				}

				// increase span count for trace
				existingTrace.spanCount = existingTrace.spanCount + 1
			}
		}
	}

	metricTracesPerBatch.Observe(float64(len(tracesByID)))

	keys := make([]uint32, 0, len(tracesByID))
	traces := make([]*rebatchedTrace, 0, len(tracesByID))

	for k, r := range tracesByID {
		keys = append(keys, k)
		traces = append(traces, r)
	}

	return keys, traces, nil
}

func countDiscaredSpans(numSuccessByTraceIndex []int, lastErrorReasonByTraceIndex []tempopb.PushErrorReason, traces []*rebatchedTrace, repFactor int) (maxLiveDiscardedCount, traceTooLargeDiscardedCount, unknownErrorCount int) {
	quorum := int(math.Floor(float64(repFactor)/2)) + 1 // min success required

	for traceIndex, numSuccess := range numSuccessByTraceIndex {
		// we will count anything that did not receive min success as discarded
		if numSuccess >= quorum {
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

func logSpans(batches []*v1.ResourceSpans, cfg *LogReceivedSpansConfig, logger log.Logger) {
	for _, b := range batches {
		loggerWithAtts := logger

		if cfg.IncludeAllAttributes {
			for _, a := range b.Resource.GetAttributes() {
				loggerWithAtts = log.With(
					loggerWithAtts,
					"span_"+strutil.SanitizeLabelName(a.GetKey()),
					tempo_util.StringifyAnyValue(a.GetValue()))
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
				tempo_util.StringifyAnyValue(a.GetValue()))
		}

		latencySeconds := float64(s.GetEndTimeUnixNano()-s.GetStartTimeUnixNano()) / float64(time.Second.Nanoseconds())
		logger = log.With(
			logger,
			"span_name", s.Name,
			"span_duration_seconds", latencySeconds,
			"span_kind", s.GetKind().String(),
			"span_status", s.GetStatus().GetCode().String())
	}

	level.Info(logger).Log("msg", "received", "spanid", hex.EncodeToString(s.SpanId), "traceid", hex.EncodeToString(s.TraceId))
}

// startEndFromSpan returns a unix epoch timestamp in seconds for the start and end of a span
func startEndFromSpan(span *v1.Span) (uint32, uint32) {
	return uint32(span.StartTimeUnixNano / uint64(time.Second)), uint32(span.EndTimeUnixNano / uint64(time.Second))
}
