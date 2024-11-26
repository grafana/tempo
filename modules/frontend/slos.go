package frontend

import (
	"net/http"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/dskit/grpcutil"
	"github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
)

const (
	traceByIDOp = "traces"
	searchOp    = "search"
	metadataOp  = "metadata"
	metricsOp   = "metrics"

	resultCompleted = "completed"
	resultCanceled  = "canceled"
)

var (
	// be careful about adding or removing labels from this metric. this, along with the
	// query_frontend_queries_total metric are used to calculate budget burns.
	// the labels need to be aligned for accurate calculations
	sloQueriesPerTenant = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_within_slo_total",
		Help:      "Total Queries within SLO per tenant",
	}, []string{"tenant", "op", "result"})

	sloTraceByIDCounter = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": traceByIDOp})
	sloSearchCounter    = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": searchOp})
	sloMetadataCounter  = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": metadataOp})
	sloMetricsCounter   = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": metricsOp})

	// be careful about adding or removing labels from this metric. this, along with the
	// query_frontend_queries_within_slo_total metric are used to calculate budget burns.
	// the labels need to be aligned for accurate calculations
	queriesPerTenant = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant", "op", "result"})

	traceByIDCounter = queriesPerTenant.MustCurryWith(prometheus.Labels{"op": traceByIDOp})
	searchCounter    = queriesPerTenant.MustCurryWith(prometheus.Labels{"op": searchOp})
	metadataCounter  = queriesPerTenant.MustCurryWith(prometheus.Labels{"op": metadataOp})
	metricsCounter   = queriesPerTenant.MustCurryWith(prometheus.Labels{"op": metricsOp})

	queryThroughput = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Name:                            "query_frontend_bytes_processed_per_second",
		Help:                            "Bytes processed per second in the query per tenant",
		Buckets:                         prometheus.ExponentialBuckets(8*1024*1024, 2, 12), // from 8MB up to 16GB
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{"tenant", "op"})

	searchThroughput   = queryThroughput.MustCurryWith(prometheus.Labels{"op": searchOp})
	metadataThroughput = queryThroughput.MustCurryWith(prometheus.Labels{"op": metadataOp})
	metricsThroughput  = queryThroughput.MustCurryWith(prometheus.Labels{"op": metricsOp})
)

type (
	handlerPostHook func(resp *http.Response, tenant string, bytesProcessed uint64, latency time.Duration, err error)
)

// todo: remove post hooks and implement as a handler
func traceByIDSLOPostHook(cfg SLOConfig) handlerPostHook {
	return sloHook(traceByIDCounter, sloTraceByIDCounter, nil, cfg)
}

func searchSLOPostHook(cfg SLOConfig) handlerPostHook {
	return sloHook(searchCounter, sloSearchCounter, searchThroughput, cfg)
}

func metadataSLOPostHook(cfg SLOConfig) handlerPostHook {
	return sloHook(metadataCounter, sloMetadataCounter, metadataThroughput, cfg)
}

func metricsSLOPostHook(cfg SLOConfig) handlerPostHook {
	return sloHook(metricsCounter, sloMetricsCounter, metricsThroughput, cfg)
}

func sloHook(allByTenantCounter, withinSLOByTenantCounter *prometheus.CounterVec, throughputVec prometheus.ObserverVec, cfg SLOConfig) handlerPostHook {
	return func(resp *http.Response, tenant string, bytesProcessed uint64, latency time.Duration, err error) {
		// most errors are SLO violations but we have few exceptions.
		if err != nil {
			// However, gRPC resource exhausted error (429), invalid argument (400), not found (404) and
			// request cancellations are considered within the SLO.
			switch status.Code(err) {
			case codes.ResourceExhausted, codes.InvalidArgument, codes.NotFound:
				allByTenantCounter.WithLabelValues(tenant, resultCompleted).Inc()
				withinSLOByTenantCounter.WithLabelValues(tenant, resultCompleted).Inc()
				return
			}

			if grpcutil.IsCanceled(err) {
				allByTenantCounter.WithLabelValues(tenant, resultCanceled).Inc()
				withinSLOByTenantCounter.WithLabelValues(tenant, resultCanceled).Inc()
				return
			}

			// check for the response and 499 in the status code, can come from http pipeline along with error
			if resp != nil && resp.StatusCode == util.StatusClientClosedRequest {
				allByTenantCounter.WithLabelValues(tenant, resultCanceled).Inc()
				withinSLOByTenantCounter.WithLabelValues(tenant, resultCanceled).Inc()
				return
			}

			// in case we have error, that doesn't fall into the above categories, it's a SLO violation
			// so only increment the allByTenantCounter
			allByTenantCounter.WithLabelValues(tenant, resultCompleted).Inc()
			return
		}

		// we don't always get error in case of http pipeline, check for 499 status code
		if resp != nil && resp.StatusCode == util.StatusClientClosedRequest {
			allByTenantCounter.WithLabelValues(tenant, resultCanceled).Inc()
			withinSLOByTenantCounter.WithLabelValues(tenant, resultCanceled).Inc()
			return
		}

		// record all queries
		allByTenantCounter.WithLabelValues(tenant, resultCompleted).Inc()

		// all 200s/300s/400s are success
		if resp != nil && resp.StatusCode >= 500 {
			return
		}

		passedThroughput := false
		// final check is throughput
		// throughputVec is nil for TraceByIDSLO
		if cfg.ThroughputBytesSLO > 0 && throughputVec != nil {
			throughput := 0.0
			seconds := latency.Seconds()
			if seconds > 0 {
				throughput = float64(bytesProcessed) / seconds
			}

			throughputVec.WithLabelValues(tenant).Observe(throughput)
			passedThroughput = throughput >= cfg.ThroughputBytesSLO
		}

		passedDuration := false
		if cfg.DurationSLO > 0 {
			passedDuration = cfg.DurationSLO == 0 || latency < cfg.DurationSLO
		}

		hasConfiguredSLO := cfg.DurationSLO > 0 || cfg.ThroughputBytesSLO > 0
		// throughput and latency are evaluated simultaneously. if either pass then we're good
		// if both fail then bail out
		// only bail out if they were actually configured
		if !passedDuration && !passedThroughput && hasConfiguredSLO {
			return
		}

		withinSLOByTenantCounter.WithLabelValues(tenant, resultCompleted).Inc()
	}
}
