package spanpruning

import (
	"testing"
	"time"

	spanpruningprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

// ---------------------------------------------------------------------------
// helpers: constructors
//
// Span/trace construction and summary-inspection helpers live in pkg/util/test
// (test.MakeSpanPruningSpan, test.FindSpanPruningSummary, etc.) so they can be
// shared with modules/frontend/combiner and the integration/api span pruning tests.
// ---------------------------------------------------------------------------

var fixedTraceID = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

func intAttr(v int64) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: "db.retries", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: v}}}
}

func boolAttr(v bool) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: "db.cached", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: v}}}
}

func defaultCfg(minSpans int) *spanpruningprocessor.Config {
	cfg := spanpruningprocessor.NewFactory().CreateDefaultConfig().(*spanpruningprocessor.Config)
	cfg.MinSpansToAggregate = minSpans
	return cfg
}

// ---------------------------------------------------------------------------
// basic aggregation
// ---------------------------------------------------------------------------

func TestPruneTrace_EmptyTrace(t *testing.T) {
	result, _, err := PruneTrace(defaultCfg(2), &tempopb.Trace{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 0, test.CountSpans(result))
}

func TestPruneTrace_BasicAggregation(t *testing.T) {
	// 3 identical leaf spans ≥ min=2 → 1 parent + 1 summary
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 1_000_000_000+uint64(i)*100, 1_000_000_100+uint64(i)*100, test.MakeAttribute("db.operation", "select")))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 2, test.CountSpans(result))

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)
	require.Equal(t, int64(3), test.SpanAttrInt(sum, "aggregation.span_count"))
}

func TestPruneTrace_BelowThreshold(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	// 1 child — below min=2
	child := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100, test.MakeAttribute("db.operation", "select"))

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(parent, child))
	require.NoError(t, err)
	require.Equal(t, 2, test.CountSpans(result))
}

func TestPruneTrace_MixedLeafAndNonLeaf(t *testing.T) {
	// root → intermediate → 3 leaf spans: only leaves should be aggregated
	cfg := defaultCfg(2)

	root := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "root", 0, 1000)
	inter := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "intermediate", 0, 500)
	spans := []*tracev1.Span{root, inter}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(2, 0), "SELECT", 0, 100))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	// root + intermediate + 1 summary
	require.Equal(t, 3, test.CountSpans(result))
}

func TestPruneTrace_SingleSpanTrace(t *testing.T) {
	root := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "root", 0, 100)
	result, _, err := PruneTrace(defaultCfg(2), test.WrapSpansAsTrace(root))
	require.NoError(t, err)
	require.Equal(t, 1, test.CountSpans(result))
}

// ---------------------------------------------------------------------------
// status separation
// ---------------------------------------------------------------------------

func TestPruneTrace_StatusAggregation(t *testing.T) {
	// 4 OK + 2 Error with same name → two separate summary spans
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 4; i++ {
		spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 100, tracev1.Status_STATUS_CODE_OK))
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 100, tracev1.Status_STATUS_CODE_ERROR))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 3, test.CountSpans(result)) // parent + ok summary + err summary

	okSum, found := test.FindSpanPruningSummaryByNameAndStatus(result, "SELECT", tracev1.Status_STATUS_CODE_OK)
	require.True(t, found)
	require.Equal(t, int64(4), test.SpanAttrInt(okSum, "aggregation.span_count"))

	errSum, found := test.FindSpanPruningSummaryByNameAndStatus(result, "SELECT", tracev1.Status_STATUS_CODE_ERROR)
	require.True(t, found)
	require.Equal(t, int64(2), test.SpanAttrInt(errSum, "aggregation.span_count"))
}

func TestPruneTrace_StatusBelowThreshold(t *testing.T) {
	// 1 OK + 1 Error: each group below min=2, nothing aggregated
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	ok := test.MakeSpanPruningSpanWithStatus(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "SELECT", 100, tracev1.Status_STATUS_CODE_OK)
	er := test.MakeSpanPruningSpanWithStatus(fixedTraceID, test.MakeSpanPruningSpanID(2, 1), test.MakeSpanPruningSpanID(1, 0), "SELECT", 100, tracev1.Status_STATUS_CODE_ERROR)

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(parent, ok, er))
	require.NoError(t, err)
	require.Equal(t, 3, test.CountSpans(result))
}

// ---------------------------------------------------------------------------
// duration stats
// ---------------------------------------------------------------------------

func TestPruneTrace_DurationStats(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 10_000)
	spans := []*tracev1.Span{parent}
	base := uint64(1_000_000_000)
	for i, dur := range []uint64{100, 200, 300} {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, byte(i)), test.MakeSpanPruningSpanID(1, 0), "SELECT", base, base+dur))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)
	require.Equal(t, int64(100), test.SpanAttrInt(sum, "aggregation.duration_min_ns"))
	require.Equal(t, int64(300), test.SpanAttrInt(sum, "aggregation.duration_max_ns"))
	require.Equal(t, int64(200), test.SpanAttrInt(sum, "aggregation.duration_avg_ns")) // (100+200+300)/3 = 200
	require.Equal(t, int64(600), test.SpanAttrInt(sum, "aggregation.duration_total_ns"))
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
	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1_000_000_000)
	spans := []*tracev1.Span{parent}
	base := uint64(1_000_000_000)
	for i, d := range durations {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, byte(i)), test.MakeSpanPruningSpanID(1, 0), "SELECT", base, base+d))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 2, test.CountSpans(result))

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)

	bounds := test.SpanAttrDoubleSlice(sum, "aggregation.histogram_bucket_bounds_s")
	require.Len(t, bounds, 3)
	require.InDelta(t, 0.01, bounds[0], 1e-9)
	require.InDelta(t, 0.05, bounds[1], 1e-9)
	require.InDelta(t, 0.10, bounds[2], 1e-9)

	counts := test.SpanAttrIntSlice(sum, "aggregation.histogram_bucket_counts")
	require.Len(t, counts, 4) // len(buckets)+1 (overflow)
	require.Equal(t, []int64{1, 3, 4, 5}, counts)
}

func TestPruneTrace_HistogramDisabled(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0
	cfg.AggregationHistogramBuckets = []time.Duration{} // disabled

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1_000_000_000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 5; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)
	require.Nil(t, test.SpanAttr(sum, "aggregation.histogram_bucket_bounds_s"))
	require.Nil(t, test.SpanAttr(sum, "aggregation.histogram_bucket_counts"))
}

// ---------------------------------------------------------------------------
// group_by_attributes
// ---------------------------------------------------------------------------

func TestPruneTrace_GroupByAttributes(t *testing.T) {
	// 5 mysql + 5 postgres with group_by=[db.system] → 2 separate summaries
	cfg := defaultCfg(5)
	cfg.GroupByAttributes = []string{"db.system"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 5; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "db.query", 0, 100, test.MakeAttribute("db.system", "mysql")))
	}
	for i := byte(0); i < 5; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(1, 0), "db.query", 0, 100, test.MakeAttribute("db.system", "postgres")))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 3, test.CountSpans(result)) // parent + 2 summaries
}

func TestPruneTrace_GroupByNonStringAttributes(t *testing.T) {
	// Group by int and bool attributes
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.retries", "db.cached"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	// 2 spans: retries=1, cached=true
	spans = append(
		spans,
		test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, intAttr(1), boolAttr(true)),
		test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 1), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, intAttr(1), boolAttr(true)),
	)
	// 2 spans: retries=2, cached=true
	spans = append(
		spans,
		test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, 0), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, intAttr(2), boolAttr(true)),
		test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, 1), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, intAttr(2), boolAttr(true)),
	)
	// 2 spans: retries=1, cached=false — same retries as the first group, but must not merge
	// with it, or grouping isn't actually keying on the bool attribute.
	spans = append(
		spans,
		test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(4, 0), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, intAttr(1), boolAttr(false)),
		test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(4, 1), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, intAttr(1), boolAttr(false)),
	)

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	// parent + 3 summaries (one per group)
	require.Equal(t, 4, test.CountSpans(result))
	require.Len(t, test.SpanPruningSummaries(result), 3)
}

func TestPruneTrace_DifferentGroups(t *testing.T) {
	// 3 SELECT + 2 INSERT with group_by=[db.operation] → 2 separate summaries
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.operation"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, test.MakeAttribute("db.operation", "select")))
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100, test.MakeAttribute("db.operation", "insert")))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 3, test.CountSpans(result)) // parent + select summary + insert summary
}

// ---------------------------------------------------------------------------
// glob patterns
// ---------------------------------------------------------------------------

func TestPruneTrace_GlobPatternWildcard(t *testing.T) {
	// "db.*" should match db.operation, db.name, db.system
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.*"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(
			fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100,
			test.MakeAttribute("db.operation", "select"),
			test.MakeAttribute("db.name", "users"),
			test.MakeAttribute("db.system", "postgresql"),
		))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	// All 3 have identical db.* attrs → 1 summary
	require.Equal(t, 2, test.CountSpans(result))
	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)
	require.Equal(t, int64(3), test.SpanAttrInt(sum, "aggregation.span_count"))
}

func TestPruneTrace_GlobPatternSeparatesGroups(t *testing.T) {
	// "db.*" with different db.operation values → 2 summaries
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.*"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100,
			test.MakeAttribute("db.operation", "select"), test.MakeAttribute("db.name", "users")))
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100,
			test.MakeAttribute("db.operation", "insert"), test.MakeAttribute("db.name", "users")))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 3, test.CountSpans(result)) // parent + 2 summaries
}

func TestPruneTrace_GlobPatternMultiplePatterns(t *testing.T) {
	// ["db.*", "http.*"] — all spans have identical db.* and http.*, so 1 summary
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.*", "http.*"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(
			fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "api_call", 0, 100,
			test.MakeAttribute("db.operation", "select"),
			test.MakeAttribute("db.name", "users"),
			test.MakeAttribute("http.method", "GET"),
			test.MakeAttribute("http.route", "/api/users"),
		))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 2, test.CountSpans(result))
}

func TestPruneTrace_GlobPatternExactMatch(t *testing.T) {
	// Exact "db.operation" without wildcard still groups correctly
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.operation"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(
			fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "db_query", 0, 100,
			test.MakeAttribute("db.operation", "select"),
			test.MakeAttribute("db.name", "users"),
			test.MakeAttribute("db.system", "postgresql"),
		))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 2, test.CountSpans(result))
}

func TestPruneTrace_InvalidGlobPattern(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"[invalid"}

	_, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "root", 0, 100)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid glob pattern")
}

// ---------------------------------------------------------------------------
// template: longest duration span is used as template
// ---------------------------------------------------------------------------

func TestPruneTrace_LongestDurationTemplate(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.operation"}
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 10_000)
	base := uint64(1_000_000_000)

	// durations 100, 500, 200 → the 500ns span should become template
	spans := []*tracev1.Span{parent}
	type spanDef struct {
		dur   uint64
		ident string
	}
	for i, d := range []spanDef{{100, "short"}, {500, "longest"}, {200, "medium"}} {
		s := test.MakeSpanPruningSpan(
			fixedTraceID, test.MakeSpanPruningSpanID(2, byte(i)), test.MakeSpanPruningSpanID(1, 0), "db_query", base, base+d.dur,
			test.MakeAttribute("db.operation", "select"),
			test.MakeAttribute("span.identifier", d.ident),
		)
		spans = append(spans, s)
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)
	id := test.SpanAttr(sum, "span.identifier")
	require.NotNil(t, id)
	require.Equal(t, "longest", id.GetStringValue())
	require.Equal(t, int64(100), test.SpanAttrInt(sum, "aggregation.duration_min_ns"))
	require.Equal(t, int64(500), test.SpanAttrInt(sum, "aggregation.duration_max_ns"))
}

// ---------------------------------------------------------------------------
// template: events and links preserved
// ---------------------------------------------------------------------------

func TestPruneTrace_TemplateEventsAndLinksPreserved(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)

	// Template span: longer duration → becomes template
	template := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 500, test.MakeAttribute("db.operation", "select"))
	template.Events = []*tracev1.Span_Event{
		{Name: "template_event", Attributes: []*commonv1.KeyValue{test.MakeAttribute("event.attr", "value")}},
	}
	template.Links = []*tracev1.Span_Link{
		{
			TraceId:    []byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
			SpanId:     []byte{9, 9, 9, 9, 9, 9, 9, 9},
			Attributes: []*commonv1.KeyValue{test.MakeAttribute("link.kind", "template")},
		},
	}

	// Shorter span
	other := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 1), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100, test.MakeAttribute("db.operation", "select"))

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(parent, template, other))
	require.NoError(t, err)

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)

	require.Len(t, sum.Events, 1)
	require.Equal(t, "template_event", sum.Events[0].Name)

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
	require.Equal(t, "template", linkAttr.GetStringValue())
}

// ---------------------------------------------------------------------------
// parent aggregation
// ---------------------------------------------------------------------------

func TestPruneTrace_ParentNotAggregatedIfChildrenMixed(t *testing.T) {
	// handler1 has 3 SELECTs (aggregated); handler2 has 1 INSERT (not aggregated)
	// → handlers cannot be aggregated (mixed children)
	cfg := defaultCfg(2)

	root := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "root", 0, 1000)
	h1 := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "handler", 0, 500)
	h2 := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 1), test.MakeSpanPruningSpanID(1, 0), "handler", 0, 500)

	spans := []*tracev1.Span{root, h1, h2}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(2, 0), "SELECT", 0, 100))
	}
	spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(4, 0), test.MakeSpanPruningSpanID(2, 1), "INSERT", 0, 100))

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	// root + h1 + h2 + SELECT summary + INSERT = 5
	require.Equal(t, 5, test.CountSpans(result))
	_, found := test.FindSpanPruningSummaryByName(result, "handler")
	require.False(t, found, "handler should NOT be aggregated")
	_, found = test.FindSpanPruningSummaryByName(result, "SELECT")
	require.True(t, found)
	require.Len(t, test.SpansByName(result, "handler"), 2)
}

func TestPruneTrace_RootSpansNotAggregated(t *testing.T) {
	// 3 root spans each with 2 SELECT children → SELECTs aggregate, roots do not
	cfg := defaultCfg(2)

	spans := []*tracev1.Span{}
	for i := byte(0); i < 3; i++ {
		rootID := test.MakeSpanPruningSpanID(i+1, 0)
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, rootID, nil, "root", 0, 1000))
		for j := byte(0); j < 2; j++ {
			spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(i+4, j), rootID, "SELECT", 0, 100))
		}
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	// 3 roots + 1 SELECT summary = 4
	require.Equal(t, 4, test.CountSpans(result))
	require.Len(t, test.SpansByName(result, "root"), 3)
	sum, found := test.FindSpanPruningSummaryByName(result, "SELECT")
	require.True(t, found)
	require.Equal(t, int64(6), test.SpanAttrInt(sum, "aggregation.span_count"))
}

func TestPruneTrace_RecursiveParentAggregation(t *testing.T) {
	// 3× handler(OK)→SELECT(OK) + 2× handler(Error)→SELECT(Error) + 1 handler(OK)→INSERT + 1 worker→SELECT
	// Expected: root + 4 summaries + 1 INSERT + 1 worker + 1 isolated SELECT = 9
	cfg := defaultCfg(2)
	cfg.GroupByAttributes = []string{"db.op"}

	rootID := test.MakeSpanPruningSpanID(1, 0)
	root := test.MakeSpanPruningSpan(fixedTraceID, rootID, nil, "root", 0, 10_000)
	spans := []*tracev1.Span{root}

	// 3× handler(OK) → SELECT(OK)
	for i := byte(0); i < 3; i++ {
		hID := test.MakeSpanPruningSpanID(2, i)
		h := test.MakeSpanPruningSpanWithStatus(fixedTraceID, hID, rootID, "handler", 500, tracev1.Status_STATUS_CODE_OK)
		s := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), hID, "SELECT", 0, 100, test.MakeAttribute("db.op", "select"))
		s.Status = &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK}
		spans = append(spans, h, s)
	}
	// 2× handler(Error) → SELECT(Error)
	for i := byte(0); i < 2; i++ {
		hID := test.MakeSpanPruningSpanID(4, i)
		h := test.MakeSpanPruningSpanWithStatus(fixedTraceID, hID, rootID, "handler", 500, tracev1.Status_STATUS_CODE_ERROR)
		s := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(5, i), hID, "SELECT", 0, 100, test.MakeAttribute("db.op", "select"))
		s.Status = &tracev1.Status{Code: tracev1.Status_STATUS_CODE_ERROR}
		spans = append(spans, h, s)
	}
	// 1× handler(OK) → INSERT
	hID := test.MakeSpanPruningSpanID(6, 0)
	spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, hID, rootID, "handler", 500, tracev1.Status_STATUS_CODE_OK))
	spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(7, 0), hID, "INSERT", 0, 100, test.MakeAttribute("db.op", "insert")))
	// 1× worker → SELECT (different parent name, won't merge with handler groups)
	wID := test.MakeSpanPruningSpanID(8, 0)
	spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, wID, rootID, "worker", 0, 500))
	spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(9, 0), wID, "SELECT", 0, 100, test.MakeAttribute("db.op", "select")))

	require.Equal(t, 15, len(spans))

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 9, test.CountSpans(result))

	hOK, found := test.FindSpanPruningSummaryByNameAndStatus(result, "handler", tracev1.Status_STATUS_CODE_OK)
	require.True(t, found)
	require.Equal(t, int64(3), test.SpanAttrInt(hOK, "aggregation.span_count"))

	hErr, found := test.FindSpanPruningSummaryByNameAndStatus(result, "handler", tracev1.Status_STATUS_CODE_ERROR)
	require.True(t, found)
	require.Equal(t, int64(2), test.SpanAttrInt(hErr, "aggregation.span_count"))

	sOK, found := test.FindSpanPruningSummaryByNameAndStatus(result, "SELECT", tracev1.Status_STATUS_CODE_OK)
	require.True(t, found)
	require.Equal(t, int64(3), test.SpanAttrInt(sOK, "aggregation.span_count"))

	sErr, found := test.FindSpanPruningSummaryByNameAndStatus(result, "SELECT", tracev1.Status_STATUS_CODE_ERROR)
	require.True(t, found)
	require.Equal(t, int64(2), test.SpanAttrInt(sErr, "aggregation.span_count"))

	// SELECT(OK) summary should be a child of handler(OK) summary
	require.Equal(t, string(hOK.SpanId), string(sOK.ParentSpanId))
	require.Equal(t, string(hErr.SpanId), string(sErr.ParentSpanId))

	require.True(t, test.SpanExistsWithName(result, "INSERT"))
	require.True(t, test.SpanExistsWithName(result, "worker"))
}

func TestPruneTrace_ThreeLevelAggregation(t *testing.T) {
	// 1 root → 2 middleware → 2 handler each → 2 SELECT each = 15 spans
	// With MaxParentDepth=-1 (unlimited) all three levels should collapse
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = -1

	rootID := test.MakeSpanPruningSpanID(1, 0)
	root := test.MakeSpanPruningSpan(fixedTraceID, rootID, nil, "root", 0, 10_000)
	spans := []*tracev1.Span{root}

	n := byte(2)
	for i := byte(0); i < 2; i++ {
		mwID := test.MakeSpanPruningSpanID(n, 0)
		n++
		spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, mwID, rootID, "middleware", 1000, tracev1.Status_STATUS_CODE_OK))
		for j := byte(0); j < 2; j++ {
			hID := test.MakeSpanPruningSpanID(n, 0)
			n++
			spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, hID, mwID, "handler", 500, tracev1.Status_STATUS_CODE_OK))
			for k := byte(0); k < 2; k++ {
				spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, test.MakeSpanPruningSpanID(n, k), hID, "SELECT", 100, tracev1.Status_STATUS_CODE_OK))
			}
			n++
		}
	}

	require.Equal(t, 15, len(spans))

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	// root + middleware summary + handler summary + SELECT summary = 4
	require.Equal(t, 4, test.CountSpans(result))

	mwSum, found := test.FindSpanPruningSummaryByName(result, "middleware")
	require.True(t, found)
	hSum, found := test.FindSpanPruningSummaryByName(result, "handler")
	require.True(t, found)
	sSum, found := test.FindSpanPruningSummaryByName(result, "SELECT")
	require.True(t, found)

	// Parent-child chain: root → middleware_summary → handler_summary → SELECT_summary
	require.Equal(t, string(mwSum.SpanId), string(hSum.ParentSpanId))
	require.Equal(t, string(hSum.SpanId), string(sSum.ParentSpanId))

	require.Equal(t, int64(2), test.SpanAttrInt(mwSum, "aggregation.span_count"))
	require.Equal(t, int64(4), test.SpanAttrInt(hSum, "aggregation.span_count"))
	require.Equal(t, int64(8), test.SpanAttrInt(sSum, "aggregation.span_count"))
}

// ---------------------------------------------------------------------------
// TraceState grouping
// ---------------------------------------------------------------------------

func TestPruneTrace_TraceState_SameGrouped(t *testing.T) {
	// 3 spans with identical TraceState → aggregate to 1 summary
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0
	ts := "ot=th:fd70a4;rv:12345"

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		s := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100)
		s.TraceState = ts
		spans = append(spans, s)
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 2, test.CountSpans(result))

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)
	require.Equal(t, int64(3), test.SpanAttrInt(sum, "aggregation.span_count"))
	require.Equal(t, ts, sum.TraceState)
}

func TestPruneTrace_TraceState_DifferentSeparated(t *testing.T) {
	// 3 spans with th:fd70a4 + 2 spans with th:fa00 → 2 separate summaries
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		s := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100)
		s.TraceState = "ot=th:fd70a4;rv:12345"
		spans = append(spans, s)
	}
	for i := byte(0); i < 2; i++ {
		s := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100)
		s.TraceState = "ot=th:fa00;rv:12345"
		spans = append(spans, s)
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 3, test.CountSpans(result)) // parent + 2 summaries

	sums := test.SpanPruningSummaries(result)
	require.Len(t, sums, 2)
	tsCounts := map[string]int64{}
	for _, s := range sums {
		tsCounts[s.TraceState] = test.SpanAttrInt(s, "aggregation.span_count")
	}
	require.Equal(t, int64(3), tsCounts["ot=th:fd70a4;rv:12345"])
	require.Equal(t, int64(2), tsCounts["ot=th:fa00;rv:12345"])
}

func TestPruneTrace_TraceState_MixedWithEmpty(t *testing.T) {
	// 3 spans with TraceState + 2 without → 2 summaries
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		s := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100)
		s.TraceState = "ot=th:fd70a4;rv:12345"
		spans = append(spans, s)
	}
	for i := byte(0); i < 2; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(3, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 3, test.CountSpans(result))

	sums := test.SpanPruningSummaries(result)
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
	require.Equal(t, "ot=th:fd70a4;rv:12345", withTS.TraceState)
	require.Equal(t, int64(3), test.SpanAttrInt(withTS, "aggregation.span_count"))
	require.Equal(t, int64(2), test.SpanAttrInt(withoutTS, "aggregation.span_count"))
}

func TestPruneTrace_TraceState_Empty(t *testing.T) {
	// 3 spans all with empty TraceState → aggregate normally
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)
	require.Equal(t, 2, test.CountSpans(result))
	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)
	require.Empty(t, sum.TraceState)
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

	rootID := test.MakeSpanPruningSpanID(1, 0)
	root := test.MakeSpanPruningSpan(fixedTraceID, rootID, nil, "root", 0, 10_000)
	spans := []*tracev1.Span{root}

	n := byte(2)
	nextID := func() []byte { id := test.MakeSpanPruningSpanID(n, 0); n++; return id }

	for i := 0; i < 3; i++ {
		outerID := nextID()
		spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, outerID, rootID, "svc", 1000, tracev1.Status_STATUS_CODE_OK))
		for j := 0; j < 2; j++ {
			innerID := nextID()
			spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, innerID, outerID, "svc", 500, tracev1.Status_STATUS_CODE_OK))
			for k := 0; k < 2; k++ {
				spans = append(spans, test.MakeSpanPruningSpanWithStatus(fixedTraceID, nextID(), innerID, "SELECT", 100, tracev1.Status_STATUS_CODE_OK))
			}
		}
	}

	require.Equal(t, 22, len(spans)) // 1 root + 3 outer + 6 inner + 12 leaf

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)

	// No dangling parents: every non-root span must have its parent in the result.
	byID := map[string]*tracev1.Span{}
	for _, s := range test.AllSpansInTrace(result) {
		byID[string(s.SpanId)] = s
	}
	for _, s := range test.AllSpansInTrace(result) {
		if len(s.ParentSpanId) == 0 {
			continue
		}
		_, ok := byID[string(s.ParentSpanId)]
		require.True(t, ok, "span %q has dangling parent", s.Name)
	}

	// Expected: root + 3 summaries (outer-svc, inner-svc, SELECT)
	require.Equal(t, 4, test.CountSpans(result))
}

// ---------------------------------------------------------------------------
// conversion roundtrip
// ---------------------------------------------------------------------------

func TestPruneTrace_ConversionRoundtrip(t *testing.T) {
	s := test.MakeSpanPruningSpan(
		fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "root", 100, 200,
		test.MakeAttribute("service.name", "my-service"),
	)
	trace := test.WrapSpansAsTrace(s)

	td, err := tempopbToTraces(trace)
	require.NoError(t, err)
	require.Equal(t, 1, td.SpanCount())

	back, err := tracesToTempopb(td)
	require.NoError(t, err)
	require.Equal(t, 1, test.CountSpans(back))
	require.Equal(t, "root", back.ResourceSpans[0].ScopeSpans[0].Spans[0].Name)
}

// ---------------------------------------------------------------------------
// summary attributes present
// ---------------------------------------------------------------------------

func TestPruneTrace_SummaryAttributesPresent(t *testing.T) {
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "child", 0, 100))
	}

	result, _, err := PruneTrace(cfg, test.WrapSpansAsTrace(spans...))
	require.NoError(t, err)

	sum, found := test.FindSpanPruningSummary(result)
	require.True(t, found)

	keys := map[string]struct{}{}
	for _, kv := range sum.Attributes {
		keys[kv.Key] = struct{}{}
	}
	require.Contains(t, keys, "aggregation.span_count")
	require.Contains(t, keys, "aggregation.is_summary")
}

// ---------------------------------------------------------------------------
// already-pruned traces
// ---------------------------------------------------------------------------

func summaryMarkerAttr(prefix string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key:   prefix + "is_summary",
		Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: true}},
	}
}

func TestPruneTrace_SkipsAlreadyPrunedTrace(t *testing.T) {
	// 3 identical leaf spans, above the aggregation threshold, but one already carries the
	// upstream-pruning summary marker: pruning must be a no-op on the whole trace, not just the
	// already-summarized span.
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	alreadySummarized := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100, summaryMarkerAttr(cfg.AggregationAttributePrefix))
	spans := []*tracev1.Span{parent, alreadySummarized}
	for i := byte(1); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100))
	}

	trace := test.WrapSpansAsTrace(spans...)
	result, status, err := PruneTrace(cfg, trace)
	require.NoError(t, err)
	require.Same(t, trace, result)
	require.Equal(t, 4, test.CountSpans(result))
	require.Equal(t, StatusPrunedOnWrite, status)
}

func TestPruneTrace_HonorsCustomAggregationAttributePrefix(t *testing.T) {
	// the marker key is derived from AggregationAttributePrefix, not hardcoded.
	cfg := defaultCfg(2)
	cfg.MaxParentDepth = 0
	cfg.AggregationAttributePrefix = "custom."

	parent := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 1000)
	alreadySummarized := test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, 0), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100, summaryMarkerAttr("custom."))
	spans := []*tracev1.Span{parent, alreadySummarized}
	for i := byte(1); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(fixedTraceID, test.MakeSpanPruningSpanID(2, i), test.MakeSpanPruningSpanID(1, 0), "SELECT", 0, 100))
	}

	trace := test.WrapSpansAsTrace(spans...)
	result, _, err := PruneTrace(cfg, trace)
	require.NoError(t, err)
	require.Same(t, trace, result)
}
