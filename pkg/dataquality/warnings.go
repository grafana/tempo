package dataquality

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const reasonOutsideIngestionSlack = "outside_ingestion_time_slack"

var metric = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "warnings_total",
	Help:      "The total number of warnings per tenant with reason.",
}, []string{"tenant", "reason"})

func WarnOutsideIngestionSlack(tenant string) {
	metric.WithLabelValues(tenant, reasonOutsideIngestionSlack).Inc()
}
