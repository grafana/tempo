package distributor

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/grafana/dskit/limiter"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/distributor/receiver"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	_ "github.com/grafana/tempo/pkg/gogocodec" // force gogo codec registration
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/validation"
)

const (
	discardReasonLabel = "reason"

	// reasonRateLimited indicates that the tenants spans/second exceeded their limits
	reasonRateLimited = "rate_limited"
	// reasonTraceTooLarge indicates that a single trace has too many spans
	reasonTraceTooLarge = "trace_too_large"
	// reasonLiveTracesExceeded indicates that tempo is already tracking too many live traces in the ingesters for this user
	reasonLiveTracesExceeded = "live_traces_exceeded"
	// reasonInternalError indicates an unexpected error occurred processing these spans. analogous to a 500
	reasonInternalError = "internal_error"
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
	metricDiscardedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "discarded_spans_total",
		Help:      "The total number of samples that were discarded.",
	}, []string{discardReasonLabel, "tenant"})
)

// Distributor coordinates replicates and distribution of log streams.
type Distributor struct {
	services.Service

	cfg             Config
	clientCfg       ingester_client.Config
	ingestersRing   ring.ReadRing
	pool            *ring_client.Pool
	DistributorRing *ring.Ring
	overrides       *overrides.Overrides

	// search
	searchEnabled    bool
	globalTagsToDrop map[string]struct{}

	// Per-user rate limiter.
	ingestionRateLimiter *limiter.RateLimiter

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New a distributor creates.
func New(cfg Config, clientCfg ingester_client.Config, ingestersRing ring.ReadRing, o *overrides.Overrides, middleware receiver.Middleware, level logging.Level, searchEnabled bool, reg prometheus.Registerer) (*Distributor, error) {
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
		lifecycler, err := ring.NewLifecycler(lifecyclerCfg, nil, "distributor", cfg.OverrideRingKey, false, log.Logger, prometheus.WrapRegistererWithPrefix("cortex_", reg))
		if err != nil {
			return nil, err
		}
		subservices = append(subservices, lifecycler)
		ingestionRateStrategy = newGlobalIngestionRateStrategy(o, lifecycler)

		ring, err := ring.New(lifecyclerCfg.RingConfig, "distributor", cfg.OverrideRingKey, log.Logger, prometheus.WrapRegistererWithPrefix("cortex_", reg))
		if err != nil {
			return nil, errors.Wrap(err, "unable to initialize distributor ring")
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
		log.Logger)

	subservices = append(subservices, pool)

	// turn list into map for efficient checking
	tagsToDrop := map[string]struct{}{}
	for _, tag := range cfg.SearchTagsDenyList {
		tagsToDrop[tag] = struct{}{}
	}

	d := &Distributor{
		cfg:                  cfg,
		clientCfg:            clientCfg,
		ingestersRing:        ingestersRing,
		pool:                 pool,
		DistributorRing:      distributorRing,
		ingestionRateLimiter: limiter.NewRateLimiter(ingestionRateStrategy, 10*time.Second),
		searchEnabled:        searchEnabled,
		globalTagsToDrop:     tagsToDrop,
		overrides:            o,
	}

	cfgReceivers := cfg.Receivers
	if len(cfgReceivers) == 0 {
		cfgReceivers = defaultReceivers
	}

	receivers, err := receiver.New(cfgReceivers, d, middleware, level)
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

// PushBatches pushes a batch of traces
func (d *Distributor) PushBatches(ctx context.Context, batches []*v1.ResourceSpans) (*tempopb.PushResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "distributor.PushBatches")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		// can't record discarded spans here b/c there's no tenant
		return nil, err
	}

	if d.cfg.LogReceivedTraces {
		logTraces(batches)
	}

	// metric size
	size := 0
	spanCount := 0
	for _, b := range batches {
		size += b.Size()
		for _, ils := range b.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}
	}
	if spanCount == 0 {
		return &tempopb.PushResponse{}, nil
	}
	metricBytesIngested.WithLabelValues(userID).Add(float64(size))
	metricSpansIngested.WithLabelValues(userID).Add(float64(spanCount))

	// check limits
	now := time.Now()
	if !d.ingestionRateLimiter.AllowN(now, userID, size) {
		metricDiscardedSpans.WithLabelValues(reasonRateLimited, userID).Add(float64(spanCount))
		return nil, status.Errorf(codes.ResourceExhausted,
			"%s ingestion rate limit (%d bytes) exceeded while adding %d bytes",
			overrides.ErrorPrefixRateLimited,
			int(d.ingestionRateLimiter.Limit(now, userID)),
			size)
	}

	keys, traces, ids, err := requestsByTraceID(batches, userID, spanCount)
	if err != nil {
		metricDiscardedSpans.WithLabelValues(reasonInternalError, userID).Add(float64(spanCount))
		return nil, err
	}

	var searchData [][]byte
	if d.searchEnabled {
		perTenantAllowedTags := d.overrides.SearchTagsAllowList(userID)
		searchData = extractSearchDataAll(traces, ids, func(tag string) bool {
			// if in per tenant override, extract
			if _, ok := perTenantAllowedTags[tag]; ok {
				return true
			}
			// if in global deny list, drop
			if _, ok := d.globalTagsToDrop[tag]; ok {
				return false
			}
			// allow otherwise
			return true
		})
	}

	err = d.sendToIngestersViaBytes(ctx, userID, traces, searchData, keys, ids)
	if err != nil {
		recordDiscaredSpans(err, userID, spanCount)
	}

	return nil, err // PushRequest is ignored, so no reason to create one
}

func (d *Distributor) sendToIngestersViaBytes(ctx context.Context, userID string, traces []*tempopb.Trace, searchData [][]byte, keys []uint32, ids [][]byte) error {
	// Marshal to bytes once
	marshalledTraces := make([][]byte, len(traces))
	for i, t := range traces {
		b, err := t.Marshal()
		if err != nil {
			return errors.Wrap(err, "failed to marshal PushRequest")
		}
		marshalledTraces[i] = b
	}

	op := ring.WriteNoExtend
	if d.cfg.ExtendWrites {
		op = ring.Write
	}

	err := ring.DoBatch(ctx, op, d.ingestersRing, keys, func(ingester ring.InstanceDesc, indexes []int) error {
		localCtx, cancel := context.WithTimeout(ctx, d.clientCfg.RemoteTimeout)
		defer cancel()
		localCtx = user.InjectOrgID(localCtx, userID)

		req := tempopb.PushBytesRequest{
			Traces:     make([]tempopb.PreallocBytes, len(indexes)),
			Ids:        make([]tempopb.PreallocBytes, len(indexes)),
			SearchData: make([]tempopb.PreallocBytes, len(indexes)),
		}

		for i, j := range indexes {
			req.Traces[i].Slice = marshalledTraces[j][0:]
			req.Ids[i].Slice = ids[j]

			// Search data optional
			if len(searchData) > j {
				req.SearchData[i].Slice = searchData[j]
			}
		}

		c, err := d.pool.GetClientFor(ingester.Addr)
		if err != nil {
			return err
		}

		_, err = c.(tempopb.PusherClient).PushBytes(localCtx, &req)
		metricIngesterAppends.WithLabelValues(ingester.Addr).Inc()
		if err != nil {
			metricIngesterAppendFailures.WithLabelValues(ingester.Addr).Inc()
		}
		return err
	}, func() {})

	return err
}

// Check implements the grpc healthcheck
func (*Distributor) Check(_ context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

// requestsByTraceID takes an incoming tempodb.PushRequest and creates a set of keys for the hash ring
// and traces to pass onto the ingesters.
func requestsByTraceID(batches []*v1.ResourceSpans, userID string, spanCount int) ([]uint32, []*tempopb.Trace, [][]byte, error) {
	type traceAndID struct {
		id    []byte
		trace *tempopb.Trace
	}

	const tracesPerBatch = 20 // p50 of internal env
	tracesByID := make(map[uint32]*traceAndID, tracesPerBatch)

	for _, b := range batches {
		spansByILS := make(map[uint32]*v1.InstrumentationLibrarySpans)

		for _, ils := range b.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				traceID := span.TraceId
				if !validation.ValidTraceID(traceID) {
					return nil, nil, nil, status.Errorf(codes.InvalidArgument, "trace ids must be 128 bit")
				}

				traceKey := util.TokenFor(userID, traceID)
				ilsKey := traceKey
				if ils.InstrumentationLibrary != nil {
					ilsKey = fnv1a.AddString32(ilsKey, ils.InstrumentationLibrary.Name)
					ilsKey = fnv1a.AddString32(ilsKey, ils.InstrumentationLibrary.Version)
				}

				existingILS, ok := spansByILS[ilsKey]
				if !ok {
					existingILS = &v1.InstrumentationLibrarySpans{
						InstrumentationLibrary: ils.InstrumentationLibrary,
						Spans:                  make([]*v1.Span, 0, spanCount/tracesPerBatch),
					}
					spansByILS[ilsKey] = existingILS
				}
				existingILS.Spans = append(existingILS.Spans, span)

				// if we found an ILS we assume its already part of a request and can go to the next span
				if ok {
					continue
				}

				existingTrace, ok := tracesByID[traceKey]
				if !ok {
					existingTrace = &traceAndID{
						id: traceID,
						trace: &tempopb.Trace{
							Batches: make([]*v1.ResourceSpans, 0, spanCount/tracesPerBatch),
						},
					}

					tracesByID[traceKey] = existingTrace
				}

				existingTrace.trace.Batches = append(existingTrace.trace.Batches, &v1.ResourceSpans{
					Resource:                    b.Resource,
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{existingILS},
				})
			}
		}
	}

	metricTracesPerBatch.Observe(float64(len(tracesByID)))

	keys := make([]uint32, 0, len(tracesByID))
	traces := make([]*tempopb.Trace, 0, len(tracesByID))
	ids := make([][]byte, 0, len(tracesByID))

	for k, r := range tracesByID {
		keys = append(keys, k)
		traces = append(traces, r.trace)
		ids = append(ids, r.id)
	}

	return keys, traces, ids, nil
}

func recordDiscaredSpans(err error, userID string, spanCount int) {
	s := status.Convert(err)
	if s == nil {
		return
	}
	desc := s.Message()

	if strings.HasPrefix(desc, overrides.ErrorPrefixLiveTracesExceeded) {
		metricDiscardedSpans.WithLabelValues(reasonLiveTracesExceeded, userID).Add(float64(spanCount))
	} else if strings.HasPrefix(desc, overrides.ErrorPrefixTraceTooLarge) {
		metricDiscardedSpans.WithLabelValues(reasonTraceTooLarge, userID).Add(float64(spanCount))
	} else {
		metricDiscardedSpans.WithLabelValues(reasonInternalError, userID).Add(float64(spanCount))
	}
}

func logTraces(batches []*v1.ResourceSpans) {
	for _, b := range batches {
		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				level.Info(log.Logger).Log("msg", "received", "spanid", hex.EncodeToString(s.SpanId), "traceid", hex.EncodeToString(s.TraceId))
			}
		}
	}
}
