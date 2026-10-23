package tracesummary

// Nanosecond timestamps and durations are encoded as JSON strings. They are
// 64-bit values — an epoch-nanosecond timestamp is already ~1.7e18 — and so
// exceed the 2^53 integer range a JSON number survives in JavaScript clients.
// This matches the protobuf JSON convention Tempo's trace endpoints already
// follow for 64-bit fields.

// TraceOverview is a condensed, human-readable view of a trace: its root,
// overall duration, critical path, per-service breakdown, and the spans most
// likely to matter for triage (slowest spans, error spans).
//
// Named to stay distinct from tracediff's TraceSummary, which is a compact
// per-side scalar rollup used only as an input to a trace comparison — not a
// general-purpose single-trace summary.
type TraceOverview struct {
	TraceID        string             `json:"traceId"`
	RootService    string             `json:"rootService"`
	RootSpanName   string             `json:"rootSpanName"`
	DurationNanos  int64              `json:"durationNanos,string"`
	SpanCount      int                `json:"spanCount"`
	ErrorSpanCount int                `json:"errorSpanCount"`
	CriticalPath   []PathSpan         `json:"criticalPath"`
	Services       []ServiceBreakdown `json:"services"`
	SlowestSpans   []SpanSummary      `json:"slowestSpans"`
	ErrorSpans     []ErrorSpanSummary `json:"errorSpans"`
}

// PathSpan is a span on the trace's critical path, in root-to-leaf order.
//
// SelfDurationNanos is what this hop actually contributed: on a critical path
// every span nests inside the one above it, so the wall durations decline only
// slightly from hop to hop and don't say which level spent the time.
type PathSpan struct {
	SpanID            string         `json:"spanId"`
	Service           string         `json:"service"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	StartTimeUnixNano uint64         `json:"startTimeUnixNano,string"`
	DurationNanos     int64          `json:"durationNanos,string"`
	SelfDurationNanos int64          `json:"selfDurationNanos,string"`
	Attributes        map[string]any `json:"attributes,omitempty"`
}

// ServiceBreakdown aggregates span/error counts and duration for a single
// resolved service name.
//
// DurationNanos is inclusive of child span time and so double-counts nested
// work; summed across services it can far exceed the trace duration. Rank
// services by SelfDurationNanos, which excludes time covered by child spans.
type ServiceBreakdown struct {
	Service           string `json:"service"`
	SpanCount         int    `json:"spanCount"`
	ErrorSpanCount    int    `json:"errorSpanCount"`
	DurationNanos     int64  `json:"durationNanos,string"`
	SelfDurationNanos int64  `json:"selfDurationNanos,string"`
}

// SpanSummary describes a single span, e.g. one of the trace's slowest spans.
type SpanSummary struct {
	SpanID            string         `json:"spanId"`
	ParentSpanID      string         `json:"parentSpanId,omitempty"`
	Service           string         `json:"service"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	StartTimeUnixNano uint64         `json:"startTimeUnixNano,string"`
	DurationNanos     int64          `json:"durationNanos,string"`
	SelfDurationNanos int64          `json:"selfDurationNanos,string"`
	Attributes        map[string]any `json:"attributes,omitempty"`
}

// ErrorSpanSummary describes a single span with an error status.
type ErrorSpanSummary struct {
	SpanID            string         `json:"spanId"`
	ParentSpanID      string         `json:"parentSpanId,omitempty"`
	Service           string         `json:"service"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	StartTimeUnixNano uint64         `json:"startTimeUnixNano,string"`
	DurationNanos     int64          `json:"durationNanos,string"`
	StatusMessage     string         `json:"statusMessage"`
	Attributes        map[string]any `json:"attributes,omitempty"`
}
