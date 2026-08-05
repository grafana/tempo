package tracesummary

// Summary is a condensed, human-readable view of a trace: its root, overall
// duration, critical path, per-service breakdown, and the spans most likely
// to matter for triage (slowest spans, error spans).
type Summary struct {
	TraceID       string             `json:"traceId"`
	RootService   string             `json:"rootService"`
	RootSpanName  string             `json:"rootSpanName"`
	DurationNanos int64              `json:"durationNanos"`
	SpanCount     int                `json:"spanCount"`
	ErrorCount    int                `json:"errorCount"`
	CriticalPath  []PathSpan         `json:"criticalPath"`
	Services      []ServiceBreakdown `json:"services"`
	SlowestSpans  []SpanSummary      `json:"slowestSpans"`
	ErrorSpans    []ErrorSpanSummary `json:"errorSpans"`
}

// PathSpan is a span on the trace's critical path, in root-to-leaf order.
type PathSpan struct {
	SpanID            string `json:"spanId"`
	Service           string `json:"service"`
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	StartTimeUnixNano uint64 `json:"startTimeUnixNano"`
	DurationNanos     int64  `json:"durationNanos"`
}

// ServiceBreakdown aggregates span/error counts and cumulative duration for
// a single resolved service name.
type ServiceBreakdown struct {
	Service       string `json:"service"`
	SpanCount     int    `json:"spanCount"`
	ErrorCount    int    `json:"errorCount"`
	DurationNanos int64  `json:"durationNanos"`
}

// SpanSummary describes a single span, e.g. one of the trace's slowest spans.
type SpanSummary struct {
	SpanID            string         `json:"spanId"`
	Service           string         `json:"service"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	StartTimeUnixNano uint64         `json:"startTimeUnixNano"`
	DurationNanos     int64          `json:"durationNanos"`
	Attributes        map[string]any `json:"attributes,omitempty"`
}

// ErrorSpanSummary describes a single span with an error status.
type ErrorSpanSummary struct {
	SpanID            string         `json:"spanId"`
	Service           string         `json:"service"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	StartTimeUnixNano uint64         `json:"startTimeUnixNano"`
	DurationNanos     int64          `json:"durationNanos"`
	StatusMessage     string         `json:"statusMessage"`
	Attributes        map[string]any `json:"attributes,omitempty"`
}
