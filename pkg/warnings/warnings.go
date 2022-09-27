package warnings

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const ReasonOutsideIngestionSlack = "outside_ingestion_time_slack"

var (
	Metric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "warnings_total",
		Help:      "The total number of warnings per tenant with reason.",
	}, []string{"tenant", "reason"})
)
