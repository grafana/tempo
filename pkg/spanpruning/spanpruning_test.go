package spanpruning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	spanpruningprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/tempopb"
)

// ---------------------------------------------------------------------------
// helpers: constructors
// ---------------------------------------------------------------------------

var fixedTraceID = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

func sid(a, b byte) []byte { return []byte{a, b, 0, 0, 0, 0, 0, 0} }

func newSpan(id, parentID []byte, name string, startNs, endNs uint64) *tracev1.Span {
	return &tracev1.Span{
		TraceId:           fixedTraceID,
		SpanId:            id,
		ParentSpanId:      parentID,
		Name:              name,
		StartTimeUnixNano: startNs,
		EndTimeUnixNano:   endNs,
	}
}

func newSpanWithStatus(id, parentID []byte, name string, startNs, endNs uint64, code tracev1.Status_StatusCode) *tracev1.Span {
	s := newSpan(id, parentID, name, startNs, endNs)
	s.Status = &tracev1.Status{Code: code}
	return s
}

func newSpanWithAttrs(id, parentID []byte, name string, startNs, endNs uint64, attrs ...*commonv1.KeyValue) *tracev1.Span {
	s := newSpan(id, parentID, name, startNs, endNs)
	s.Attributes = attrs
	return s
}

func strAttr(k, v string) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: k, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: v}}}
}

func intAttr(k string, v int64) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: k, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: v}}}
}

func boolAttr(k string, v bool) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: k, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: v}}}
}

func wrapTrace(spans ...*tracev1.Span) *tempopb.Trace {
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{ScopeSpans: []*tracev1.ScopeSpans{{Spans: spans}}},
		},
	}
}

func defaultCfg(min int) *spanpruningprocessor.Config {
	cfg := spanpruningprocessor.NewFactory().CreateDefaultConfig().(*spanpruningprocessor.Config)
	cfg.MinSpansToAggregate = min
	return cfg
}

// ---------------------------------------------------------------------------
// helpers: inspection
// ---------------------------------------------------------------------------

func countSpans(t *tempopb.Trace) int {
	n := 0
	for _, rs := range t.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			n += len(ss.Spans)
		}
	}
	return n
}

func allSpans(t *tempopb.Trace) []*tracev1.Span {
	var out []*tracev1.Span
	for _, rs := range t.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			out = append(out, ss.Spans...)
		}
	}
	return out
}

func getAttr(s *tracev1.Span, key string) *commonv1.AnyValue {
	for _, kv := range s.Attributes {
		if kv.Key == key {
			return kv.Value
		}
	}
	return nil
}

func isSummary(s *tracev1.Span) bool {
	v := getAttr(s, "aggregation.is_summary")
	return v != nil && v.GetBoolValue()
}

func findSummary(t *tempopb.Trace) (*tracev1.Span, bool) {
	for _, s := range allSpans(t) {
		if isSummary(s) {
			return s, true
		}
	}
	return nil, false
}

func findAllSummaries(t *tempopb.Trace) []*tracev1.Span {
	var out []*tracev1.Span
	for _, s := range allSpans(t) {
		if isSummary(s) {
			out = append(out, s)
		}
	}
	return out
}

func findSummaryByName(t *tempopb.Trace, name string) (*tracev1.Span, bool) {
	for _, s := range allSpans(t) {
		if isSummary(s) && s.Name == name {
			return s, true
		}
	}
	return nil, false
}

func findSummaryByNameStatus(t *tempopb.Trace, name string, code tracev1.Status_StatusCode) (*tracev1.Span, bool) {
	for _, s := range allSpans(t) {
		if isSummary(s) && s.Name == name && s.Status != nil && s.Status.Code == code {
			return s, true
		}
	}
	return nil, false
}

func existsByName(t *tempopb.Trace, name string) bool {
	for _, s := range allSpans(t) {
		if s.Name == name {
			return true
		}
	}
	return false
}

func allByName(t *tempopb.Trace, name string) []*tracev1.Span {
	var out []*tracev1.Span
	for _, s := range allSpans(t) {
		if s.Name == name {
			out = append(out, s)
		}
	}
	return out
}

func findByID(t *tempopb.Trace, id []byte) *tracev1.Span {
	for _, s := range allSpans(t) {
		if string(s.SpanId) == string(id) {
			return s
		}
	}
	return nil
}

// attrIntVal returns the int64 value of an attribute, panics if missing/wrong type.
func attrInt(s *tracev1.Span, key string) int64 {
	v := getAttr(s, key)
	if v == nil {
		return -1
	}
	return v.GetIntValue()
}

// sliceDoubles returns the double values of an ArrayValue attribute.
func sliceDoubles(s *tracev1.Span, key string) []float64 {
	v := getAttr(s, key)
	if v == nil {
		return nil
	}
	arr := v.GetArrayValue()
	if arr == nil {
		return nil
	}
	out := make([]float64, len(arr.Values))
	for i, elem := range arr.Values {
		out[i] = elem.GetDoubleValue()
	}
	return out
}

// sliceInts returns the int64 values of an ArrayValue attribute.
func sliceInts(s *tracev1.Span, key string) []int64 {
	v := getAttr(s, key)
	if v == nil {
		return nil
	}
	arr := v.GetArrayValue()
	if arr == nil {
		return nil
	}
	out := make([]int64, len(arr.Values))
	for i, elem := range arr.Values {
		out[i] = elem.GetIntValue()
	}
	return out
}

// ---------------------------------------------------------------------------
// basic aggregation
// ---------------------------------------------------------------------------

func TestPruneTrace_EmptyTrace(t *testing.T) {
	result, err := PruneTrace(defaultCfg(2), &tempopb.Trace{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, countSpans(result))
}

func TestPruneTrace_BasicAggregation(t *testing.T) {
	// 3 identical leaf spans ≥ min=2 → 1 parent + 1 summary
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpanWithAttrs(sid(2, i), sid(1, 0), "SELECT", 1_000_000_000+uint64(i)*100, 1_000_000_100+uint64(i)*100, strAttr("db.operation", "select")))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))

	sum, found := findSummary(result)
	require.True(t, found)
	assert.Equal(t, int64(3), attrInt(sum, "aggregation.span_count"))
}

func TestPruneTrace_BelowThreshold(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	// 1 child — below min=2
	child := newSpanWithAttrs(sid(2, 0), sid(1, 0), "SELECT", 0, 100, strAttr("db.operation", "select"))

	result, err := PruneTrace(cfg, wrapTrace(parent, child))
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))
}

func TestPruneTrace_MixedLeafAndNonLeaf(t *testing.T) {
	// root → intermediate → 3 leaf spans: only leaves should be aggregated
	cfg := defaultCfg(2)

	root := newSpan(sid(1, 0), nil, "root", 0, 1000)
	inter := newSpan(sid(2, 0), sid(1, 0), "intermediate", 0, 500)
	spans := []*tracev1.Span{root, inter}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(3, i), sid(2, 0), "SELECT", 0, 100))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	// root + intermediate + 1 summary
	assert.Equal(t, 3, countSpans(result))
}

func TestPruneTrace_SingleSpanTrace(t *testing.T) {
	root := newSpan(sid(1, 0), nil, "root", 0, 100)
	result, err := PruneTrace(defaultCfg(2), wrapTrace(root))
	require.NoError(t, err)
	assert.Equal(t, 1, countSpans(result))
}

// ---------------------------------------------------------------------------
// status separation
// ---------------------------------------------------------------------------

func TestPruneTrace_StatusAggregation(t *testing.T) {
	// 4 OK + 2 Error with same name → two separate summary spans
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 4; i++ {
		spans = append(spans, newSpanWithStatus(sid(2, i), sid(1, 0), "SELECT", 0, 100, tracev1.Status_STATUS_CODE_OK))
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, newSpanWithStatus(sid(3, i), sid(1, 0), "SELECT", 0, 100, tracev1.Status_STATUS_CODE_ERROR))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 3, countSpans(result)) // parent + ok summary + err summary

	okSum, found := findSummaryByNameStatus(result, "SELECT", tracev1.Status_STATUS_CODE_OK)
	require.True(t, found)
	assert.Equal(t, int64(4), attrInt(okSum, "aggregation.span_count"))

	errSum, found := findSummaryByNameStatus(result, "SELECT", tracev1.Status_STATUS_CODE_ERROR)
	require.True(t, found)
	assert.Equal(t, int64(2), attrInt(errSum, "aggregation.span_count"))
}

func TestPruneTrace_StatusBelowThreshold(t *testing.T) {
	// 1 OK + 1 Error: each group below min=2, nothing aggregated
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	ok := newSpanWithStatus(sid(2, 0), sid(1, 0), "SELECT", 0, 100, tracev1.Status_STATUS_CODE_OK)
	er := newSpanWithStatus(sid(2, 1), sid(1, 0), "SELECT", 0, 100, tracev1.Status_STATUS_CODE_ERROR)

	result, err := PruneTrace(cfg, wrapTrace(parent, ok, er))
	require.NoError(t, err)
	assert.Equal(t, 3, countSpans(result))
}

// ---------------------------------------------------------------------------
// duration stats
// ---------------------------------------------------------------------------

func TestPruneTrace_DurationStats(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 10_000)
	spans := []*tracev1.Span{parent}
	base := uint64(1_000_000_000)
	for i, dur := range []uint64{100, 200, 300} {
		spans = append(spans, newSpan(sid(2, byte(i)), sid(1, 0), "SELECT", base, base+dur))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found)
	assert.Equal(t, int64(100), attrInt(sum, "aggregation.duration_min_ns"))
	assert.Equal(t, int64(300), attrInt(sum, "aggregation.duration_max_ns"))
	assert.Equal(t, int64(200), attrInt(sum, "aggregation.duration_avg_ns")) // (100+200+300)/3 = 200
	assert.Equal(t, int64(600), attrInt(sum, "aggregation.duration_total_ns"))
}

// ---------------------------------------------------------------------------
// histogram
// ---------------------------------------------------------------------------

func TestPruneTrace_HistogramEnabled(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0
	cfg.AggregationHistogramBuckets = []time.Duration{
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
	}

	durations := []uint64{
		uint64(5 * time.Millisecond),
		uint64(15 * time.Millisecond),
		uint64(25 * time.Millisecond),
		uint64(75 * time.Millisecond),
		uint64(150 * time.Millisecond),
	}
	parent := newSpan(sid(1, 0), nil, "parent", 0, 1_000_000_000)
	spans := []*tracev1.Span{parent}
	base := uint64(1_000_000_000)
	for i, d := range durations {
		spans = append(spans, newSpan(sid(2, byte(i)), sid(1, 0), "SELECT", base, base+d))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))

	sum, found := findSummary(result)
	require.True(t, found)

	bounds := sliceDoubles(sum, "aggregation.histogram_bucket_bounds_s")
	require.Len(t, bounds, 3)
	assert.InDelta(t, 0.01, bounds[0], 1e-9)
	assert.InDelta(t, 0.05, bounds[1], 1e-9)
	assert.InDelta(t, 0.10, bounds[2], 1e-9)

	counts := sliceInts(sum, "aggregation.histogram_bucket_counts")
	require.Len(t, counts, 4) // len(buckets)+1 (overflow)
	assert.Equal(t, []int64{1, 3, 4, 5}, counts)
}

func TestPruneTrace_HistogramDisabled(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0
	cfg.AggregationHistogramBuckets = []time.Duration{} // disabled

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1_000_000_000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 5; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "SELECT", 0, 100))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found)
	assert.Nil(t, getAttr(sum, "aggregation.histogram_bucket_bounds_s"))
	assert.Nil(t, getAttr(sum, "aggregation.histogram_bucket_counts"))
}

// ---------------------------------------------------------------------------
// group_by_attributes
// ---------------------------------------------------------------------------

func TestPruneTrace_GroupByAttributes(t *testing.T) {
	// 5 mysql + 5 postgres with group_by=[db.system] → 2 separate summaries
	cfg := defaultCfg(5)
	cfg.GroupByAttributes = []string{"db.system"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 5; i++ {
		spans = append(spans, newSpanWithAttrs(sid(2, i), sid(1, 0), "db.query", 0, 100, strAttr("db.system", "mysql")))
	}
	for i := byte(0); i < 5; i++ {
		spans = append(spans, newSpanWithAttrs(sid(3, i), sid(1, 0), "db.query", 0, 100, strAttr("db.system", "postgres")))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 3, countSpans(result)) // parent + 2 summaries
}

func TestPruneTrace_GroupByNonStringAttributes(t *testing.T) {
	// Group by int and bool attributes
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.retries", "db.cached"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	// 2 spans: retries=1, cached=true
	spans = append(spans,
		newSpanWithAttrs(sid(2, 0), sid(1, 0), "db_query", 0, 100, intAttr("db.retries", 1), boolAttr("db.cached", true)),
		newSpanWithAttrs(sid(2, 1), sid(1, 0), "db_query", 0, 100, intAttr("db.retries", 1), boolAttr("db.cached", true)),
	)
	// 2 spans: retries=2, cached=true
	spans = append(spans,
		newSpanWithAttrs(sid(3, 0), sid(1, 0), "db_query", 0, 100, intAttr("db.retries", 2), boolAttr("db.cached", true)),
		newSpanWithAttrs(sid(3, 1), sid(1, 0), "db_query", 0, 100, intAttr("db.retries", 2), boolAttr("db.cached", true)),
	)

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	// parent + 2 summaries (one per group)
	assert.Equal(t, 3, countSpans(result))
	assert.Len(t, findAllSummaries(result), 2)
}

func TestPruneTrace_DifferentGroups(t *testing.T) {
	// 3 SELECT + 2 INSERT with group_by=[db.operation] → 2 separate summaries
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.operation"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpanWithAttrs(sid(2, i), sid(1, 0), "db_query", 0, 100, strAttr("db.operation", "select")))
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, newSpanWithAttrs(sid(3, i), sid(1, 0), "db_query", 0, 100, strAttr("db.operation", "insert")))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 3, countSpans(result)) // parent + select summary + insert summary
}

// ---------------------------------------------------------------------------
// glob patterns
// ---------------------------------------------------------------------------

func TestPruneTrace_GlobPatternWildcard(t *testing.T) {
	// "db.*" should match db.operation, db.name, db.system
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.*"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpanWithAttrs(sid(2, i), sid(1, 0), "db_query", 0, 100,
			strAttr("db.operation", "select"),
			strAttr("db.name", "users"),
			strAttr("db.system", "postgresql"),
		))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	// All 3 have identical db.* attrs → 1 summary
	assert.Equal(t, 2, countSpans(result))
	sum, found := findSummary(result)
	require.True(t, found)
	assert.Equal(t, int64(3), attrInt(sum, "aggregation.span_count"))
}

func TestPruneTrace_GlobPatternSeparatesGroups(t *testing.T) {
	// "db.*" with different db.operation values → 2 summaries
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.*"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, newSpanWithAttrs(sid(2, i), sid(1, 0), "db_query", 0, 100,
			strAttr("db.operation", "select"), strAttr("db.name", "users")))
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, newSpanWithAttrs(sid(3, i), sid(1, 0), "db_query", 0, 100,
			strAttr("db.operation", "insert"), strAttr("db.name", "users")))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 3, countSpans(result)) // parent + 2 summaries
}

func TestPruneTrace_GlobPatternMultiplePatterns(t *testing.T) {
	// ["db.*", "http.*"] — all spans have identical db.* and http.*, so 1 summary
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.*", "http.*"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpanWithAttrs(sid(2, i), sid(1, 0), "api_call", 0, 100,
			strAttr("db.operation", "select"),
			strAttr("db.name", "users"),
			strAttr("http.method", "GET"),
			strAttr("http.route", "/api/users"),
		))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))
}

func TestPruneTrace_GlobPatternExactMatch(t *testing.T) {
	// Exact "db.operation" without wildcard still groups correctly
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.operation"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpanWithAttrs(sid(2, i), sid(1, 0), "db_query", 0, 100,
			strAttr("db.operation", "select"),
			strAttr("db.name", "users"),
			strAttr("db.system", "postgresql"),
		))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))
}

func TestPruneTrace_InvalidGlobPattern(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"[invalid"}

	_, err := PruneTrace(cfg, wrapTrace(newSpan(sid(1, 0), nil, "root", 0, 100)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid glob pattern")
}

// ---------------------------------------------------------------------------
// template: longest duration span is used as template
// ---------------------------------------------------------------------------

func TestPruneTrace_LongestDurationTemplate(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.operation"}
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 10_000)
	base := uint64(1_000_000_000)

	// durations 100, 500, 200 → the 500ns span should become template
	spans := []*tracev1.Span{parent}
	type spanDef struct {
		dur   uint64
		ident string
	}
	for i, d := range []spanDef{{100, "short"}, {500, "longest"}, {200, "medium"}} {
		s := newSpanWithAttrs(sid(2, byte(i)), sid(1, 0), "db_query", base, base+d.dur,
			strAttr("db.operation", "select"),
			strAttr("span.identifier", d.ident),
		)
		spans = append(spans, s)
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found)
	id := getAttr(sum, "span.identifier")
	require.NotNil(t, id)
	assert.Equal(t, "longest", id.GetStringValue())
	assert.Equal(t, int64(100), attrInt(sum, "aggregation.duration_min_ns"))
	assert.Equal(t, int64(500), attrInt(sum, "aggregation.duration_max_ns"))
}

// ---------------------------------------------------------------------------
// template: events and links preserved
// ---------------------------------------------------------------------------

func TestPruneTrace_TemplateEventsAndLinksPreserved(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)

	// Template span: longer duration → becomes template
	template := newSpanWithAttrs(sid(2, 0), sid(1, 0), "SELECT", 0, 500, strAttr("db.operation", "select"))
	template.Events = []*tracev1.Span_Event{
		{Name: "template_event", Attributes: []*commonv1.KeyValue{strAttr("event.attr", "value")}},
	}
	template.Links = []*tracev1.Span_Link{
		{
			TraceId:    []byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
			SpanId:     []byte{9, 9, 9, 9, 9, 9, 9, 9},
			Attributes: []*commonv1.KeyValue{strAttr("link.kind", "template")},
		},
	}

	// Shorter span
	other := newSpanWithAttrs(sid(2, 1), sid(1, 0), "SELECT", 0, 100, strAttr("db.operation", "select"))

	result, err := PruneTrace(cfg, wrapTrace(parent, template, other))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found)

	require.Len(t, sum.Events, 1)
	assert.Equal(t, "template_event", sum.Events[0].Name)

	require.Len(t, sum.Links, 1)
	linkAttr := func() *commonv1.AnyValue {
		for _, kv := range sum.Links[0].Attributes {
			if kv.Key == "link.kind" {
				return kv.Value
			}
		}
		return nil
	}()
	require.NotNil(t, linkAttr)
	assert.Equal(t, "template", linkAttr.GetStringValue())
}

// ---------------------------------------------------------------------------
// parent aggregation
// ---------------------------------------------------------------------------

func TestPruneTrace_ParentNotAggregatedIfChildrenMixed(t *testing.T) {
	// handler1 has 3 SELECTs (aggregated); handler2 has 1 INSERT (not aggregated)
	// → handlers cannot be aggregated (mixed children)
	cfg := defaultCfg(2)

	root := newSpan(sid(1, 0), nil, "root", 0, 1000)
	h1 := newSpan(sid(2, 0), sid(1, 0), "handler", 0, 500)
	h2 := newSpan(sid(2, 1), sid(1, 0), "handler", 0, 500)

	spans := []*tracev1.Span{root, h1, h2}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(3, i), sid(2, 0), "SELECT", 0, 100))
	}
	spans = append(spans, newSpan(sid(4, 0), sid(2, 1), "INSERT", 0, 100))

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	// root + h1 + h2 + SELECT summary + INSERT = 5
	assert.Equal(t, 5, countSpans(result))
	_, found := findSummaryByName(result, "handler")
	assert.False(t, found, "handler should NOT be aggregated")
	_, found = findSummaryByName(result, "SELECT")
	assert.True(t, found)
	assert.Len(t, allByName(result, "handler"), 2)
}

func TestPruneTrace_RootSpansNotAggregated(t *testing.T) {
	// 3 root spans each with 2 SELECT children → SELECTs aggregate, roots do not
	cfg := defaultCfg(2)

	spans := []*tracev1.Span{}
	for i := byte(0); i < 3; i++ {
		rootID := sid(i+1, 0)
		spans = append(spans, newSpan(rootID, nil, "root", 0, 1000))
		for j := byte(0); j < 2; j++ {
			spans = append(spans, newSpan(sid(i+4, j), rootID, "SELECT", 0, 100))
		}
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	// 3 roots + 1 SELECT summary = 4
	assert.Equal(t, 4, countSpans(result))
	assert.Len(t, allByName(result, "root"), 3)
	sum, found := findSummaryByName(result, "SELECT")
	require.True(t, found)
	assert.Equal(t, int64(6), attrInt(sum, "aggregation.span_count"))
}

func TestPruneTrace_RecursiveParentAggregation(t *testing.T) {
	// 3× handler(OK)→SELECT(OK) + 2× handler(Error)→SELECT(Error) + 1 handler(OK)→INSERT + 1 worker→SELECT
	// Expected: root + 4 summaries + 1 INSERT + 1 worker + 1 isolated SELECT = 9
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.op"}

	rootID := sid(1, 0)
	root := newSpan(rootID, nil, "root", 0, 10_000)
	spans := []*tracev1.Span{root}

	// 3× handler(OK) → SELECT(OK)
	for i := byte(0); i < 3; i++ {
		hID := sid(2, i)
		h := newSpanWithStatus(hID, rootID, "handler", 0, 500, tracev1.Status_STATUS_CODE_OK)
		s := newSpanWithAttrs(sid(3, i), hID, "SELECT", 0, 100, strAttr("db.op", "select"))
		s.Status = &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK}
		spans = append(spans, h, s)
	}
	// 2× handler(Error) → SELECT(Error)
	for i := byte(0); i < 2; i++ {
		hID := sid(4, i)
		h := newSpanWithStatus(hID, rootID, "handler", 0, 500, tracev1.Status_STATUS_CODE_ERROR)
		s := newSpanWithAttrs(sid(5, i), hID, "SELECT", 0, 100, strAttr("db.op", "select"))
		s.Status = &tracev1.Status{Code: tracev1.Status_STATUS_CODE_ERROR}
		spans = append(spans, h, s)
	}
	// 1× handler(OK) → INSERT
	hID := sid(6, 0)
	spans = append(spans, newSpanWithStatus(hID, rootID, "handler", 0, 500, tracev1.Status_STATUS_CODE_OK))
	spans = append(spans, newSpanWithAttrs(sid(7, 0), hID, "INSERT", 0, 100, strAttr("db.op", "insert")))
	// 1× worker → SELECT (different parent name, won't merge with handler groups)
	wID := sid(8, 0)
	spans = append(spans, newSpan(wID, rootID, "worker", 0, 500))
	spans = append(spans, newSpanWithAttrs(sid(9, 0), wID, "SELECT", 0, 100, strAttr("db.op", "select")))

	assert.Equal(t, 15, len(spans))

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 9, countSpans(result))

	hOK, found := findSummaryByNameStatus(result, "handler", tracev1.Status_STATUS_CODE_OK)
	require.True(t, found)
	assert.Equal(t, int64(3), attrInt(hOK, "aggregation.span_count"))

	hErr, found := findSummaryByNameStatus(result, "handler", tracev1.Status_STATUS_CODE_ERROR)
	require.True(t, found)
	assert.Equal(t, int64(2), attrInt(hErr, "aggregation.span_count"))

	sOK, found := findSummaryByNameStatus(result, "SELECT", tracev1.Status_STATUS_CODE_OK)
	require.True(t, found)
	assert.Equal(t, int64(3), attrInt(sOK, "aggregation.span_count"))

	sErr, found := findSummaryByNameStatus(result, "SELECT", tracev1.Status_STATUS_CODE_ERROR)
	require.True(t, found)
	assert.Equal(t, int64(2), attrInt(sErr, "aggregation.span_count"))

	// SELECT(OK) summary should be a child of handler(OK) summary
	assert.Equal(t, string(hOK.SpanId), string(sOK.ParentSpanId))
	assert.Equal(t, string(hErr.SpanId), string(sErr.ParentSpanId))

	assert.True(t, existsByName(result, "INSERT"))
	assert.True(t, existsByName(result, "worker"))
}

func TestPruneTrace_ThreeLevelAggregation(t *testing.T) {
	// 1 root → 2 middleware → 2 handler each → 2 SELECT each = 15 spans
	// With MaxParentDepth=-1 (unlimited) all three levels should collapse
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = -1

	rootID := sid(1, 0)
	root := newSpan(rootID, nil, "root", 0, 10_000)
	spans := []*tracev1.Span{root}

	n := byte(2)
	for i := byte(0); i < 2; i++ {
		mwID := sid(n, 0)
		n++
		spans = append(spans, newSpanWithStatus(mwID, rootID, "middleware", 0, 1000, tracev1.Status_STATUS_CODE_OK))
		for j := byte(0); j < 2; j++ {
			hID := sid(n, 0)
			n++
			spans = append(spans, newSpanWithStatus(hID, mwID, "handler", 0, 500, tracev1.Status_STATUS_CODE_OK))
			for k := byte(0); k < 2; k++ {
				spans = append(spans, newSpanWithStatus(sid(n, k), hID, "SELECT", 0, 100, tracev1.Status_STATUS_CODE_OK))
			}
			n++
		}
	}

	assert.Equal(t, 15, len(spans))

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	// root + middleware summary + handler summary + SELECT summary = 4
	assert.Equal(t, 4, countSpans(result))

	mwSum, found := findSummaryByName(result, "middleware")
	require.True(t, found)
	hSum, found := findSummaryByName(result, "handler")
	require.True(t, found)
	sSum, found := findSummaryByName(result, "SELECT")
	require.True(t, found)

	// Parent-child chain: root → middleware_summary → handler_summary → SELECT_summary
	assert.Equal(t, string(mwSum.SpanId), string(hSum.ParentSpanId))
	assert.Equal(t, string(hSum.SpanId), string(sSum.ParentSpanId))

	assert.Equal(t, int64(2), attrInt(mwSum, "aggregation.span_count"))
	assert.Equal(t, int64(4), attrInt(hSum, "aggregation.span_count"))
	assert.Equal(t, int64(8), attrInt(sSum, "aggregation.span_count"))
}

// ---------------------------------------------------------------------------
// TraceState grouping
// ---------------------------------------------------------------------------

func TestPruneTrace_TraceState_SameGrouped(t *testing.T) {
	// 3 spans with identical TraceState → aggregate to 1 summary
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0
	ts := "ot=th:fd70a4;rv:12345"

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		s := newSpan(sid(2, i), sid(1, 0), "SELECT", 0, 100)
		s.TraceState = ts
		spans = append(spans, s)
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))

	sum, found := findSummary(result)
	require.True(t, found)
	assert.Equal(t, int64(3), attrInt(sum, "aggregation.span_count"))
	assert.Equal(t, ts, sum.TraceState)
}

func TestPruneTrace_TraceState_DifferentSeparated(t *testing.T) {
	// 3 spans with th:fd70a4 + 2 spans with th:fa00 → 2 separate summaries
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		s := newSpan(sid(2, i), sid(1, 0), "SELECT", 0, 100)
		s.TraceState = "ot=th:fd70a4;rv:12345"
		spans = append(spans, s)
	}
	for i := byte(0); i < 2; i++ {
		s := newSpan(sid(3, i), sid(1, 0), "SELECT", 0, 100)
		s.TraceState = "ot=th:fa00;rv:12345"
		spans = append(spans, s)
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 3, countSpans(result)) // parent + 2 summaries

	sums := findAllSummaries(result)
	require.Len(t, sums, 2)
	tsCounts := map[string]int64{}
	for _, s := range sums {
		tsCounts[s.TraceState] = attrInt(s, "aggregation.span_count")
	}
	assert.Equal(t, int64(3), tsCounts["ot=th:fd70a4;rv:12345"])
	assert.Equal(t, int64(2), tsCounts["ot=th:fa00;rv:12345"])
}

func TestPruneTrace_TraceState_MixedWithEmpty(t *testing.T) {
	// 3 spans with TraceState + 2 without → 2 summaries
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		s := newSpan(sid(2, i), sid(1, 0), "SELECT", 0, 100)
		s.TraceState = "ot=th:fd70a4;rv:12345"
		spans = append(spans, s)
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, newSpan(sid(3, i), sid(1, 0), "SELECT", 0, 100))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 3, countSpans(result))

	sums := findAllSummaries(result)
	require.Len(t, sums, 2)
	var withTS, withoutTS *tracev1.Span
	for _, s := range sums {
		if s.TraceState == "" {
			withoutTS = s
		} else {
			withTS = s
		}
	}
	require.NotNil(t, withTS)
	require.NotNil(t, withoutTS)
	assert.Equal(t, "ot=th:fd70a4;rv:12345", withTS.TraceState)
	assert.Equal(t, int64(3), attrInt(withTS, "aggregation.span_count"))
	assert.Equal(t, int64(2), attrInt(withoutTS, "aggregation.span_count"))
}

func TestPruneTrace_TraceState_Empty(t *testing.T) {
	// 3 spans all with empty TraceState → aggregate normally
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "SELECT", 0, 100))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))
	sum, found := findSummary(result)
	require.True(t, found)
	assert.Empty(t, sum.TraceState)
}

// ---------------------------------------------------------------------------
// regression: parent key collision across depths
// ---------------------------------------------------------------------------

// TestPruneTrace_ParentKeyCollisionRegression is a regression test for a bug where spans
// with the same name at different tree depths caused group key collisions, leaving nodes
// marked for removal but absent from the final plan (dangling parent refs).
//
// Structure (3 copies): root → svc(outer) → svc(inner) → SELECT(leaf) ×2
func TestPruneTrace_ParentKeyCollisionRegression(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = -1

	rootID := sid(1, 0)
	root := newSpan(rootID, nil, "root", 0, 10_000)
	spans := []*tracev1.Span{root}

	n := byte(2)
	nextID := func() []byte { id := sid(n, 0); n++; return id }

	for i := 0; i < 3; i++ {
		outerID := nextID()
		spans = append(spans, newSpanWithStatus(outerID, rootID, "svc", 0, 1000, tracev1.Status_STATUS_CODE_OK))
		for j := 0; j < 2; j++ {
			innerID := nextID()
			spans = append(spans, newSpanWithStatus(innerID, outerID, "svc", 0, 500, tracev1.Status_STATUS_CODE_OK))
			for k := 0; k < 2; k++ {
				spans = append(spans, newSpanWithStatus(nextID(), innerID, "SELECT", 0, 100, tracev1.Status_STATUS_CODE_OK))
			}
		}
	}

	assert.Equal(t, 22, len(spans)) // 1 root + 3 outer + 6 inner + 12 leaf

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	// No dangling parents: every non-root span must have its parent in the result.
	byID := map[string]*tracev1.Span{}
	for _, s := range allSpans(result) {
		byID[string(s.SpanId)] = s
	}
	for _, s := range allSpans(result) {
		if len(s.ParentSpanId) == 0 {
			continue
		}
		_, ok := byID[string(s.ParentSpanId)]
		assert.True(t, ok, "span %q has dangling parent", s.Name)
	}

	// Expected: root + 3 summaries (outer-svc, inner-svc, SELECT)
	assert.Equal(t, 4, countSpans(result))
}

// ---------------------------------------------------------------------------
// conversion roundtrip
// ---------------------------------------------------------------------------

func TestPruneTrace_ConversionRoundtrip(t *testing.T) {
	s := newSpanWithAttrs(sid(1, 0), nil, "root", 100, 200,
		strAttr("service.name", "my-service"),
	)
	trace := wrapTrace(s)

	td, err := tempopbToTraces(trace)
	require.NoError(t, err)
	require.Equal(t, 1, td.SpanCount())

	back, err := tracesToTempopb(td)
	require.NoError(t, err)
	require.Equal(t, 1, countSpans(back))
	assert.Equal(t, "root", back.ResourceSpans[0].ScopeSpans[0].Spans[0].Name)
}

// ---------------------------------------------------------------------------
// summary attributes present
// ---------------------------------------------------------------------------

func TestPruneTrace_SummaryAttributesPresent(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "child", 0, 100))
	}

	result, err := PruneTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found)

	keys := map[string]struct{}{}
	for _, kv := range sum.Attributes {
		keys[kv.Key] = struct{}{}
	}
	assert.Contains(t, keys, "aggregation.span_count")
	assert.Contains(t, keys, "aggregation.is_summary")
}

// ---------------------------------------------------------------------------
// SummaryOnlyTrace tests
// ---------------------------------------------------------------------------

func TestSummaryOnlyTrace_KeepsAllOriginalSpans(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "child", 0, 100))
	}
	trace := wrapTrace(spans...)

	result, err := SummaryOnlyTrace(cfg, trace)
	require.NoError(t, err)

	// Original spans are all present.
	require.Equal(t, 4, countSpans(trace), "original unchanged")
	// Result has all originals + 1 summary = 5 spans.
	assert.Equal(t, 5, countSpans(result))
}

func TestSummaryOnlyTrace_SummarySpanHasAggregationAttrs(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "child", 0, 100))
	}

	result, err := SummaryOnlyTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found, "summary span should be present")
	assert.Equal(t, int64(3), attrInt(sum, "aggregation.span_count"))
	assert.NotNil(t, getAttr(sum, "aggregation.duration_min_ns"))
	assert.NotNil(t, getAttr(sum, "aggregation.is_summary"))
}

func TestSummaryOnlyTrace_SummarySpanIDDiffersFromOriginals(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "child", 0, 100))
	}

	result, err := SummaryOnlyTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found)

	// Collect all original span IDs.
	originalIDs := map[string]struct{}{}
	for _, s := range spans {
		originalIDs[string(s.SpanId)] = struct{}{}
	}

	_, collides := originalIDs[string(sum.SpanId)]
	assert.False(t, collides, "summary SpanID must not collide with any original span")
}

func TestSummaryOnlyTrace_SummaryPlacedUnderCorrectParent(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "child", 0, 100))
	}

	result, err := SummaryOnlyTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)

	sum, found := findSummary(result)
	require.True(t, found)

	// The summary span's parent is the original parent span.
	assert.Equal(t, string(sid(1, 0)), string(sum.ParentSpanId))
}

func TestSummaryOnlyTrace_NoPruningBelowThreshold(t *testing.T) {
	cfg := defaultCfg(5)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	// Only 3 children — below min=5, so nothing is pruned and no summary is added.
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "child", 0, 100))
	}

	result, err := SummaryOnlyTrace(cfg, wrapTrace(spans...))
	require.NoError(t, err)
	// No summary added; result equals original.
	assert.Equal(t, 4, countSpans(result))
	_, found := findSummary(result)
	assert.False(t, found)
}

func TestSummaryOnlyTrace_DoesNotMutateInput(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := newSpan(sid(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, newSpan(sid(2, i), sid(1, 0), "child", 0, 100))
	}
	trace := wrapTrace(spans...)
	originalCount := countSpans(trace)

	_, err := SummaryOnlyTrace(cfg, trace)
	require.NoError(t, err)
	assert.Equal(t, originalCount, countSpans(trace), "input trace must not be mutated")
}
