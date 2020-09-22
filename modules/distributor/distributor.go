package distributor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	ring_client "github.com/cortexproject/cortex/pkg/ring/client"
	cortex_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/limiter"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/gogo/status"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/distributor/receiver"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/validation"
)

const (
	discardReasonLabel = "reason"

	// RateLimited is one of the values for the reason to discard samples.
	// Declared here to avoid duplication in ingester and distributor.
	rateLimited = "rate_limited"
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
	metricSpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_spans_received_total",
		Help:      "The total number of spans received per tenant",
	}, []string{"tenant"})
	metricTracesPerBatch = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "distributor_traces_per_batch",
		Help:      "The number of traces in each batch",
		Buckets:   prometheus.LinearBuckets(0, 3, 5),
	})
	metricIngesterClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "distributor_ingester_clients",
		Help:      "The current number of ingester clients.",
	})
	metricDiscardedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "discarded_spans_total",
		Help:      "The total number of samples that were discarded.",
	}, []string{discardReasonLabel, "tenant"})
)

// Distributor coordinates replicates and distribution of log streams.
type Distributor struct {
	services.Service

	cfg           Config
	clientCfg     ingester_client.Config
	ingestersRing ring.ReadRing
	pool          *ring_client.Pool

	// Per-user rate limiter.
	ingestionRateLimiter *limiter.RateLimiter

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New a distributor creates.
func New(cfg Config, clientCfg ingester_client.Config, ingestersRing ring.ReadRing, o *overrides.Overrides, authEnabled bool) (*Distributor, error) {
	factory := cfg.factory
	if factory == nil {
		factory = func(addr string) (ring_client.PoolClient, error) {
			return ingester_client.New(addr, clientCfg)
		}
	}

	subservices := []services.Service(nil)

	// Create the configured ingestion rate limit strategy (local or global).
	var ingestionRateStrategy limiter.RateLimiterStrategy
	var distributorsRing *ring.Lifecycler

	if o.IngestionRateStrategy() == overrides.GlobalIngestionRateStrategy {
		var err error
		distributorsRing, err = ring.NewLifecycler(cfg.DistributorRing.ToLifecyclerConfig(), nil, "distributor", ring.DistributorRingKey, false, prometheus.DefaultRegisterer)
		if err != nil {
			return nil, err
		}

		subservices = append(subservices, distributorsRing)
		ingestionRateStrategy = newGlobalIngestionRateStrategy(o, distributorsRing)
	} else {
		ingestionRateStrategy = newLocalIngestionRateStrategy(o)
	}

	pool := ring_client.NewPool("distributor_pool",
		clientCfg.PoolConfig,
		ring_client.NewRingServiceDiscovery(ingestersRing),
		factory,
		metricIngesterClients,
		cortex_util.Logger)

	subservices = append(subservices, pool)

	d := &Distributor{
		cfg:                  cfg,
		clientCfg:            clientCfg,
		ingestersRing:        ingestersRing,
		pool:                 pool,
		ingestionRateLimiter: limiter.NewRateLimiter(ingestionRateStrategy, 10*time.Second),
	}

	cfgReceivers := cfg.Receivers
	if len(cfgReceivers) == 0 {
		cfgReceivers = defaultReceivers
	}

	receivers, err := receiver.New(cfgReceivers, d, authEnabled)
	if err != nil {
		return nil, err
	}
	subservices = append(subservices, receivers)

	d.subservices, err = services.NewManager(subservices...)
	if err != nil {
		return nil, fmt.Errorf("failed to create subservices %w", err)
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
		return fmt.Errorf("failed to start subservices %w", err)
	}

	return nil
}

func (d *Distributor) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-d.subservicesWatcher.Chan():
		return fmt.Errorf("distributor subservices failed %w", err)
	}
}

// Called after distributor is asked to stop via StopAsync.
func (d *Distributor) stopping(_ error) error {
	return services.StopManagerAndAwaitStopped(context.Background(), d.subservices)
}

// Push a set of streams.
func (d *Distributor) Push(ctx context.Context, req *tempopb.PushRequest) (*tempopb.PushResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	// Track metrics.
	if req.Batch == nil {
		return &tempopb.PushResponse{}, nil
	}
	spanCount := 0
	for _, ils := range req.Batch.InstrumentationLibrarySpans {
		spanCount += len(ils.Spans)
	}
	if spanCount == 0 {
		return &tempopb.PushResponse{}, nil
	}
	metricSpansIngested.WithLabelValues(userID).Add(float64(spanCount))

	now := time.Now()
	if !d.ingestionRateLimiter.AllowN(now, userID, spanCount) {
		// Return a 4xx here to have the client discard the data and not retry. If a client
		// is sending too much data consistently we will unlikely ever catch up otherwise.
		metricDiscardedSpans.WithLabelValues(rateLimited, userID).Add(float64(spanCount))

		return nil, status.Errorf(codes.ResourceExhausted, "ingestion rate limit (%d spans) exceeded while adding %d spans", int(d.ingestionRateLimiter.Limit(now, userID)), spanCount)
	}

	keys, traces, err := requestsByTraceID(req, userID, spanCount)
	if err != nil {
		return nil, err
	}

	// keys -> slice of uint32 tokens used below
	// traces -> tempopb.PushRequests aligned with uint32 tokens
	// replication strategy for 1x,2x,3x

	// change push request to take a batch of batches
	err = ring.DoBatch(ctx, d.ingestersRing, keys, func(ingester ring.IngesterDesc, indexes []int) error {
		localCtx, cancel := context.WithTimeout(context.Background(), d.clientCfg.RemoteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		var err error
		for _, idx := range indexes {
			pushRequest := traces[idx]
			err = d.send(ctx, ingester.Addr, pushRequest)
			if err != nil {
				break
			}
		}

		return err
	}, func() {})

	return nil, err // PushRequest is ignored, so no reason to create one
}

func (d *Distributor) send(ctx context.Context, ingesterAddr string, req *tempopb.PushRequest) error {
	c, err := d.pool.GetClientFor(ingesterAddr)
	if err != nil {
		return err
	}

	_, err = c.(tempopb.PusherClient).Push(ctx, req)
	metricIngesterAppends.WithLabelValues(ingesterAddr).Inc()
	if err != nil {
		metricIngesterAppendFailures.WithLabelValues(ingesterAddr).Inc()
	}
	return err
}

// Check implements the grpc healthcheck
func (*Distributor) Check(_ context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func requestsByTraceID(req *tempopb.PushRequest, userID string, spanCount int) ([]uint32, []*tempopb.PushRequest, error) {
	const expectedTracesPerBatch = 10 // roughly what we're seeing through metrics
	expectedSpansPerTrace := spanCount / expectedTracesPerBatch

	requestsByTrace := make(map[uint32]*tempopb.PushRequest)
	spansByILS := make(map[string]*opentelemetry_proto_trace_v1.InstrumentationLibrarySpans)

	for _, ils := range req.Batch.InstrumentationLibrarySpans {
		for _, span := range ils.Spans {
			if !validation.ValidTraceID(span.TraceId) {
				return nil, nil, status.Errorf(codes.InvalidArgument, "trace ids must be 128 bit")
			}

			traceKey := util.TokenFor(userID, span.TraceId)
			ilsKey := strconv.Itoa(int(traceKey))
			if ils.InstrumentationLibrary != nil {
				ilsKey = ilsKey + ils.InstrumentationLibrary.Name + ils.InstrumentationLibrary.Version
			}
			existingILS, ok := spansByILS[ilsKey]
			if !ok {
				existingILS = &opentelemetry_proto_trace_v1.InstrumentationLibrarySpans{
					InstrumentationLibrary: ils.InstrumentationLibrary,
					Spans:                  make([]*opentelemetry_proto_trace_v1.Span, 0, expectedSpansPerTrace),
				}
				spansByILS[ilsKey] = existingILS
			}
			existingILS.Spans = append(existingILS.Spans, span)

			// if we found an ILS we assume its already part of a request and can go to the next span
			if ok {
				continue
			}

			existingReq, ok := requestsByTrace[traceKey]
			if !ok {
				existingReq = &tempopb.PushRequest{
					Batch: &opentelemetry_proto_trace_v1.ResourceSpans{
						InstrumentationLibrarySpans: make([]*opentelemetry_proto_trace_v1.InstrumentationLibrarySpans, 0, len(req.Batch.InstrumentationLibrarySpans)), // assume most spans belong to the same trace
						Resource:                    req.Batch.Resource,
					},
				}
				requestsByTrace[traceKey] = existingReq
			}
			existingReq.Batch.InstrumentationLibrarySpans = append(existingReq.Batch.InstrumentationLibrarySpans, existingILS)
		}
	}

	metricTracesPerBatch.Observe(float64(len(requestsByTrace)))

	keys := make([]uint32, 0, len(requestsByTrace))
	pushRequests := make([]*tempopb.PushRequest, 0, len(requestsByTrace))

	for k, r := range requestsByTrace {
		keys = append(keys, k)
		pushRequests = append(pushRequests, r)
	}

	return keys, pushRequests, nil
}
