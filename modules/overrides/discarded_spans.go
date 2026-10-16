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
	// ReasonTooFarInFuture indicates a span's start or end time was further into the future than the tenant's creation_grace_period allows.
	ReasonTooFarInFuture = "too_far_in_future"
	// ReasonInvalidDuration indicates a span was discarded because its end time was before its start time.
	ReasonInvalidDuration = "invalid_duration"
	// ReasonInvalidTraceID indicates a batch was rejected because it contained an invalid trace ID.
	ReasonInvalidTraceID = "invalid_trace_id"
	// ReasonInvalidSpanID indicates a batch was rejected because it contained an invalid span ID.
	ReasonInvalidSpanID = "invalid_span_id"
	// ReasonUnknown indicates an unknown error when pushing spans.
	ReasonUnknown = "unknown_error"
	// ReasonTraceTooLargeToCompact indicates a trace is too large for the backend-worker to combine/compact.
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
