package dataquality

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	reasonOutsideIngestionSlack = "outside_ingestion_time_slack"
	reasonDisconnectedTrace     = "disconnected_trace"

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

func WarnDisconnectedTrace(tenant string, phase string) {
	metric.WithLabelValues(tenant, reasonDisconnectedTrace+phase).Inc()
}
