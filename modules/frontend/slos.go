package frontend

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	traceByIDOp = "traces"
	searchOp    = "search"

	throughputKey
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
)

func traceByIDSLOHook(cfg *SLOConfig) requestHook {
	return sloHook(sloTraceByIDCounter, traceByIDCounter, cfg)
}

func searchSLOHook(cfg *SLOConfig) requestHook {
	return sloHook(sloSearchCounter, searchCounter, cfg)
}

func sloHook(allByTenantCounter, withinSLOByTenantCounter *prometheus.CounterVec, cfg *SLOConfig) requestHook {
	return func(ctx context.Context, resp *http.Response, tenant string, latency time.Duration, err error) { // jpe add tenant?
		// first record all queries
		allByTenantCounter.WithLabelValues(tenant).Inc()

		// now check conditions to see if we should record within SLO
		if err != nil {
			return
		}

		// all 200s/300s/400s are success
		if resp.StatusCode >= 500 {
			return
		}

		passedThroughput := true
		// final check is throughput
		if cfg.ThroughputBytesSLO > 0 {
			throughput, ok := ctx.Value(throughputKey).(float64)

			// if we didn't find the key, but expected it, we consider throughput a failure
			passedThroughput = !ok && throughput >= cfg.ThroughputBytesSLO
		}

		passedDuration := cfg.DurationSLO == 0 || latency < cfg.DurationSLO

		// throughput and latency are evaluated simultaneously. if either pass then we're good
		// if both fail then bail out
		if !passedDuration && !passedThroughput {
			return
		}

		withinSLOByTenantCounter.WithLabelValues(tenant).Inc()
	}
}

func addThroughputToContext(ctx context.Context, throughput float64) context.Context {
	return context.WithValue(ctx, throughputKey, throughput)
}
