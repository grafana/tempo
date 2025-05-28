package dataquality

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	reasonOutsideIngestionSlack             = "outside_ingestion_time_slack"
	reasonBlockBuilderOutsideIngestionSlack = "blockbuilder_outside_ingestion_time_slack"
	reasonDisconnectedTrace                 = "disconnected_trace"
	reasonRootlessTrace                     = "rootless_trace"

	PhaseTraceFlushedToWal     = "_flushed_to_wal"
	PhaseTraceWalToComplete    = "_wal_to_complete"
	PhaseTraceCompactorCombine = "_compactor_combine"
)

var metric = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "warnings_total",
	Help:      "The total number of warnings per tenant with reason.",
}, []string{"tenant", "reason"})

func WarnOutsideIngestionSlack(tenant string) {
	metric.WithLabelValues(tenant, reasonOutsideIngestionSlack).Inc()
}

func WarnBlockBuilderOutsideIngestionSlack(tenant string) {
	metric.WithLabelValues(tenant, reasonBlockBuilderOutsideIngestionSlack).Inc()
}

func WarnDisconnectedTrace(tenant string, phase string) {
	metric.WithLabelValues(tenant, reasonDisconnectedTrace+phase).Inc()
}

func WarnRootlessTrace(tenant string, phase string) {
	metric.WithLabelValues(tenant, reasonRootlessTrace+phase).Inc()
}

var MetricSpanInFuture = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "tempo",
	Name:      "spans_distance_in_future_seconds",
	Help:      "The number of seconds in the future of the span end time in relation to the ingestion time.",
	Buckets:   []float64{300, 1800, 3600}, // 5m, 30m, 1h
}, []string{"tenant"})

var MetricSpanInPast = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "tempo",
	Name:      "spans_distance_in_past_seconds",
	Help:      "The number of seconds in the past of the span end time in relation to the ingestion time.",
	Buckets:   []float64{300, 1800, 3600}, // 5m, 30m, 1h
}, []string{"tenant"})
