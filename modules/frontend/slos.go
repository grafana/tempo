package frontend

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	traceByIDOp = "traces"
	searchOp    = "search"
)

var (
	// be careful about adding or removing labels from this metric. this, along with the
	// query_frontend_queries_total metric are used to calculate budget burns.
	// the labels need to be aligned for accurate calculations
	sloQueriesPerTenant = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_within_slo_total",
		Help:      "Total Queries within SLO per tenant",
	}, []string{"tenant", "op"})

	sloTraceByIDCounter = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": traceByIDOp})
	sloSearchCounter    = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": searchOp})

	// be careful about adding or removing labels from this metric. this, along with the
	// query_frontend_queries_within_slo_total metric are used to calculate budget burns.
	// the labels need to be aligned for accurate calculations
	queriesPerTenant = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant", "op"})

	traceByIDCounter = queriesPerTenant.MustCurryWith(prometheus.Labels{"op": traceByIDOp})
	searchCounter    = queriesPerTenant.MustCurryWith(prometheus.Labels{"op": searchOp})

	queryThroughput = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "query_frontend_bytes_processed_per_second",
		Help:      "Bytes processed per second in the query per tenant",
		Buckets:   prometheus.ExponentialBuckets(1024*1024, 2, 10), // from 1MB up to 1GB
	}, []string{"tenant", "op"})

	searchThroughput = queryThroughput.MustCurryWith(prometheus.Labels{"op": searchOp})
)

// todo: remove post hooks and implement as a handler
func traceByIDSLOPostHook(cfg SLOConfig) handlerPostHook {
	return sloHook(traceByIDCounter, sloTraceByIDCounter, cfg)
}

func searchSLOPostHook(cfg SLOConfig) handlerPostHook {
	return sloHook(searchCounter, sloSearchCounter, cfg)
}

func sloHook(allByTenantCounter, withinSLOByTenantCounter *prometheus.CounterVec, cfg SLOConfig) handlerPostHook {
	return func(resp *http.Response, tenant string, bytesProcessed uint64, latency time.Duration, err error) {
		// first record all queries
		allByTenantCounter.WithLabelValues(tenant).Inc()

		// now check conditions to see if we should record within SLO
		if err != nil {
			return
		}

		// all 200s/300s/400s are success
		if resp != nil && resp.StatusCode >= 500 {
			return
		}

		passedThroughput := false
		// final check is throughput
		if cfg.ThroughputBytesSLO > 0 {
			throughput := 0.0
			seconds := latency.Seconds()
			if seconds > 0 {
				throughput = float64(bytesProcessed) / seconds
			}

			searchThroughput.WithLabelValues(tenant).Observe(throughput)
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

		withinSLOByTenantCounter.WithLabelValues(tenant).Inc()
	}
}
