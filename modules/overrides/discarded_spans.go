package overrides

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const discardReasonLabel = "reason"

var metricDiscardedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "discarded_spans_total",
	Help:      "The total number of samples that were discarded.",
}, []string{discardReasonLabel, "tenant"})

func RecordDiscardedSpans(spansDiscarded int, reason string, tenant string) {
	metricDiscardedSpans.WithLabelValues(reason, tenant).Add(float64(spansDiscarded))
}
