package tracediff

const (
	VersionTracePatchV0 = "trace-patch-v0"
	// VersionTraceSummaryV0Native is the single summary format: a compact
	// triage/localization document computed from normalized traces and matcher
	// results, with complete changed-service enumeration, uncapped rollups, and
	// capped patterns.
	VersionTraceSummaryV0Native = "trace-summary-v0-native"
)

// Format selects a diff format implemented directly by Diff.
type Format string

const (
	FormatTracePatchV0 Format = VersionTracePatchV0
)

// Operation is a mechanical change verb.
type Operation string

const (
	OperationAdd    Operation = "add"
	OperationRemove Operation = "remove"
	OperationModify Operation = "modify"
)

// TargetType identifies what a change affects.
type TargetType string

const (
	TargetAttribute TargetType = "attribute"
	TargetField     TargetType = "field"
	TargetSpan      TargetType = "span"
)

// Field names (Target.Name) and attribute scopes (Target.Scope) in the
// trace-patch-v0 vocabulary.
const (
	FieldDurationNanos = "duration_nanos"
	FieldStatus        = "status"
	ScopeSpan          = "span"
)

// Warning codes emitted by trace diff and trace-diff CLI outputs.
const (
	WarningHighCardinalitySpanName = "high_cardinality_span_name"
	WarningPartialTrace            = "partial_trace"
	WarningInvalidDuration         = "invalid_duration"
	WarningZeroSpanTrace           = "zero_span_trace"
	WarningDuplicateSpanID         = "duplicate_span_id"
	WarningAmbiguousSpanMatch      = "ambiguous_span_match"
)

// Result is the trace-patch-v0 diff document.
type Result struct {
	Version  string         `json:"version"`
	Base     TraceMeta      `json:"base"`
	Compare  TraceMeta      `json:"compare"`
	Stats    Stats          `json:"stats"`
	Modified []ModifiedSpan `json:"modified"`
	Added    []SpanChange   `json:"added"`
	Removed  []SpanChange   `json:"removed"`
	Warnings []Warning      `json:"warnings"`
}

// TraceMeta identifies one side of the comparison.
type TraceMeta struct {
	TraceID   string `json:"traceId"`
	SpanCount int    `json:"spanCount"`
}

// Stats contains factual counts for the diff.
type Stats struct {
	SpanCountA       int  `json:"spanCountA"`
	SpanCountB       int  `json:"spanCountB"`
	MatchedSpans     int  `json:"matchedSpans"`
	ModifiedSpans    int  `json:"modifiedSpans"`
	AddedSpans       int  `json:"addedSpans"`
	RemovedSpans     int  `json:"removedSpans"`
	FieldChanges     int  `json:"fieldChanges"`
	AttributeChanges int  `json:"attributeChanges"`
	EventChanges     int  `json:"eventChanges"`
	Truncated        bool `json:"truncated"`
}

// ModifiedSpan groups field and attribute changes for one matched span.
type ModifiedSpan struct {
	Span    SpanRef  `json:"span"`
	Changes []Change `json:"changes"`
}

// SpanRef is a stable-ish logical span locator.
type SpanRef struct {
	// Path is zero-based sibling indexes from root to span.
	Path    []int  `json:"path"`
	Service string `json:"service"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
}

// Change describes one change inside a matched span.
type Change struct {
	Op     Operation `json:"op"`
	Target Target    `json:"target"`
	Before any       `json:"before"`
	After  any       `json:"after"`
}

// Target identifies the field or attribute being changed.
type Target struct {
	Type  TargetType `json:"type"`
	Name  string     `json:"name,omitempty"`
	Scope string     `json:"scope,omitempty"`
	Key   string     `json:"key,omitempty"`
}

// SpanChange describes a whole-span add or remove.
type SpanChange struct {
	Target SpanTarget   `json:"target"`
	Span   SpanSnapshot `json:"span"`
}

// SpanTarget locates where a span is added or removed.
type SpanTarget struct {
	Type       TargetType `json:"type"`
	Path       []int      `json:"path,omitempty"`
	ParentPath []int      `json:"parentPath,omitempty"`
	Index      *int       `json:"index,omitempty"`
}

// SpanSnapshot is the minimal span data needed to render an add/remove.
type SpanSnapshot struct {
	Path          []int  `json:"path"`
	Service       string `json:"service"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	DurationNanos int64  `json:"duration_nanos"`
	Status        string `json:"status"`
}

// Warning reports non-fatal limits or matcher caveats.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
