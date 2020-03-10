package distributor

import (
	"context"
	"net/http"
	"sync"
	"time"

	cortex_client "github.com/cortexproject/cortex/pkg/ingester/client"
	"github.com/cortexproject/cortex/pkg/ring"
	cortex_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/limiter"

	"github.com/go-kit/kit/log/level"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"
	"github.com/weaveworks/common/user"
	uber_atomic "go.uber.org/atomic"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/frigg/pkg/distributor/receiver"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/ingester/client"
	"github.com/grafana/frigg/pkg/util"
	"github.com/grafana/frigg/pkg/util/validation"
)

var (
	metricIngesterAppends = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "distributor_ingester_appends_total",
		Help:      "The total number of batch appends sent to ingesters.",
	}, []string{"ingester"})
	metricIngesterAppendFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "distributor_ingester_append_failures_total",
		Help:      "The total number of failed batch appends sent to ingesters.",
	}, []string{"ingester"})
	metricSpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "distributor_spans_received_total",
		Help:      "The total number of spans received per tenant",
	}, []string{"tenant"})

	readinessProbeSuccess = []byte("Ready")
)

// Distributor coordinates replicates and distribution of log streams.
type Distributor struct {
	cfg           Config
	clientCfg     client.Config
	ingestersRing ring.ReadRing
	overrides     *validation.Overrides
	pool          *cortex_client.Pool

	// receiver shims
	receivers receiver.Receivers

	// The global rate limiter requires a distributors ring to count
	// the number of healthy instances.
	distributorsRing *ring.Lifecycler

	// Per-user rate limiter.
	ingestionRateLimiter *limiter.RateLimiter
}

// New a distributor creates.
func New(cfg Config, clientCfg client.Config, ingestersRing ring.ReadRing, overrides *validation.Overrides, authEnabled bool) (*Distributor, error) {
	factory := cfg.factory
	if factory == nil {
		factory = func(addr string) (grpc_health_v1.HealthClient, error) {
			return client.New(clientCfg, addr)
		}
	}

	// Create the configured ingestion rate limit strategy (local or global).
	var ingestionRateStrategy limiter.RateLimiterStrategy
	var distributorsRing *ring.Lifecycler

	if overrides.IngestionRateStrategy() == validation.GlobalIngestionRateStrategy {
		var err error
		distributorsRing, err = ring.NewLifecycler(cfg.DistributorRing.ToLifecyclerConfig(), nil, "distributor", ring.DistributorRingKey, false)
		if err != nil {
			return nil, err
		}

		distributorsRing.Start()

		ingestionRateStrategy = newGlobalIngestionRateStrategy(overrides, distributorsRing)
	} else {
		ingestionRateStrategy = newLocalIngestionRateStrategy(overrides)
	}

	d := &Distributor{
		cfg:                  cfg,
		clientCfg:            clientCfg,
		ingestersRing:        ingestersRing,
		distributorsRing:     distributorsRing,
		overrides:            overrides,
		pool:                 cortex_client.NewPool(clientCfg.PoolConfig, ingestersRing, factory, cortex_util.Logger),
		ingestionRateLimiter: limiter.NewRateLimiter(ingestionRateStrategy, 10*time.Second),
	}

	if len(cfg.Receivers) > 0 {
		var err error
		d.receivers, err = receiver.New(cfg.Receivers, d, authEnabled)
		if err != nil {
			return nil, err
		}
		err = d.receivers.Start()
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

func (d *Distributor) Stop() {
	if d.distributorsRing != nil {
		d.distributorsRing.Shutdown()
	}

	if d.receivers != nil {
		err := d.receivers.Shutdown()
		if err != nil {
			level.Error(cortex_util.Logger).Log("msg", "error stopping receivers", "error", err)
		}
	}
}

// ReadinessHandler is used to indicate to k8s when the distributor is ready.
// Returns 200 when the distributor is ready, 500 otherwise.
func (d *Distributor) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	_, err := d.ingestersRing.GetAll()
	if err != nil {
		http.Error(w, "Not ready: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(readinessProbeSuccess); err != nil {
		level.Error(cortex_util.Logger).Log("msg", "error writing success message", "error", err)
	}
}

// Push a set of streams.
func (d *Distributor) Push(ctx context.Context, req *friggpb.PushRequest) (*friggpb.PushResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	// Track metrics.
	if req.Batch == nil {
		return &friggpb.PushResponse{}, nil
	}
	spanCount := len(req.Batch.Spans)
	if spanCount == 0 {
		return &friggpb.PushResponse{}, nil
	}
	metricSpansIngested.WithLabelValues(userID).Add(float64(spanCount))

	now := time.Now()
	if !d.ingestionRateLimiter.AllowN(now, userID, spanCount) {
		// Return a 4xx here to have the client discard the data and not retry. If a client
		// is sending too much data consistently we will unlikely ever catch up otherwise.
		validation.DiscardedSamples.WithLabelValues(validation.RateLimited, userID).Add(float64(spanCount))
		return nil, httpgrpc.Errorf(http.StatusTooManyRequests, "ingestion rate limit (%d bytes) exceeded while adding %d spans", int(d.ingestionRateLimiter.Limit(now, userID)), spanCount)
	}

	requestsByIngester, err := d.routeRequest(req, userID, spanCount)
	if err != nil {
		return nil, err
	}

	atomicErr := uber_atomic.NewError(nil)
	done := make(chan struct{})
	wg := sync.WaitGroup{}
	for ingester, reqs := range requestsByIngester {
		wg.Add(1)
		go func(ingesterAddr string, reqs []*friggpb.PushRequest) {
			// Use a background context to make sure all ingesters get samples even if we return early
			localCtx, cancel := context.WithTimeout(context.Background(), d.clientCfg.RemoteTimeout)
			defer cancel()
			localCtx = user.InjectOrgID(localCtx, userID)
			if sp := opentracing.SpanFromContext(ctx); sp != nil {
				localCtx = opentracing.ContextWithSpan(localCtx, sp)
			}
			err := d.sendSamples(localCtx, ingesterAddr, reqs)
			atomicErr.Store(err)
			wg.Done()
		}(ingester, reqs)
	}

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		resetPushRequests(requestsByIngester)
		return &friggpb.PushResponse{}, atomicErr.Load()
	case <-ctx.Done():
		resetPushRequests(requestsByIngester)
		return nil, ctx.Err()
	}
}

// TODO taken from Loki taken from Cortex, see if we can refactor out an usable interface.
func (d *Distributor) sendSamples(ctx context.Context, ingesterAddr string, batches []*friggpb.PushRequest) error {
	var warning error

	for _, b := range batches {
		err := d.sendSamplesErr(ctx, ingesterAddr, b)

		if err != nil {
			warning = err
		}
	}

	return warning
}

// TODO taken from Loki taken from Cortex, see if we can refactor out an usable interface.
func (d *Distributor) sendSamplesErr(ctx context.Context, ingesterAddr string, req *friggpb.PushRequest) error {
	c, err := d.pool.GetClientFor(ingesterAddr)
	if err != nil {
		return err
	}

	_, err = c.(friggpb.PusherClient).Push(ctx, req)
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

func (d *Distributor) routeRequest(req *friggpb.PushRequest, userID string, spanCount int) (map[string][]*friggpb.PushRequest, error) {
	const maxExpectedReplicationSet = 3 // 3.  b/c frigg it
	var descs [maxExpectedReplicationSet]ring.IngesterDesc

	// todo: add a metric to track traces per batch
	requests := make(map[uint32]*friggpb.PushRequest)
	for _, span := range req.Batch.Spans {
		if !validation.ValidTraceID(span.TraceId) {
			return nil, httpgrpc.Errorf(http.StatusBadRequest, "trace ids must be 128 bit")
		}

		key := util.TokenFor(userID, span.TraceId)

		routedReq, ok := requests[key]
		if !ok {
			routedReq = pushRequestPool.Get().(*friggpb.PushRequest)
			resourceSpans := resourceSpansPool.Get().(*opentelemetry_proto_trace_v1.ResourceSpans)

			resourceSpans.Spans = make([]*opentelemetry_proto_trace_v1.Span, 0, spanCount), // assume most spans belong to the same trace
			resourceSpans.Resource = req.Batch.Resource
			routedReq.Batch = resourceSpans

			requests[key] = routedReq
		}
		routedReq.Batch.Spans = append(routedReq.Batch.Spans, span)
	}

	requestsByIngester := make(map[string][]*friggpb.PushRequest)
	for key, routedReq := range requests {
		// now map to ingesters
		replicationSet, err := d.ingestersRing.Get(key, ring.Write, descs[:0])
		if err != nil {
			return nil, err
		}
		for _, ingester := range replicationSet.Ingesters {
			requestsByIngester[ingester.Addr] = append(requestsByIngester[ingester.Addr], routedReq)
		}
	}

	return requestsByIngester, nil
}
