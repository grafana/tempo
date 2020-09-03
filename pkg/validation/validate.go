package validation

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	discardReasonLabel = "reason"

	// RateLimited is one of the values for the reason to discard samples.
	// Declared here to avoid duplication in ingester and distributor.
	RateLimited = "rate_limited"
)

// DiscardedSpans is a metric of the number of discarded samples, by reason.
var DiscardedSpans = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "discarded_spans_total",
		Help:      "The total number of samples that were discarded.",
	},
	[]string{discardReasonLabel, "tenant"},
)

func init() {
	prometheus.MustRegister(DiscardedSpans)
}

// ValidTraceID confirms that trace ids are 128 bits
func ValidTraceID(id []byte) bool {
	return len(id) == 16
}
