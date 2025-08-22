package overrides

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const discardReasonLabel = "reason"

// Exported reasons for discarded spans to be shared across packages.
const (
	// ReasonRateLimited indicates that the tenant's spans/second exceeded their limits.
	ReasonRateLimited = "rate_limited"
	// ReasonTraceTooLarge indicates that a single trace has too many spans.
	ReasonTraceTooLarge = "trace_too_large"
	// ReasonLiveTracesExceeded indicates Tempo is already tracking too many live traces for this tenant.
	ReasonLiveTracesExceeded = "live_traces_exceeded"
	// ReasonUnknown indicates an unknown error when pushing spans.
	ReasonUnknown = "unknown_error"
	// ReasonTraceTooLargeToCompact indicates a trace is too large for the compactor to combine/compact.
	ReasonCompactorDiscardedSpans = "trace_too_large_to_compact"
)

var metricDiscardedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "discarded_spans_total",
	Help:      "The total number of samples that were discarded.",
}, []string{discardReasonLabel, "tenant"})

func RecordDiscardedSpans(spansDiscarded int, reason string, tenant string) {
	metricDiscardedSpans.WithLabelValues(reason, tenant).Add(float64(spansDiscarded))
}
