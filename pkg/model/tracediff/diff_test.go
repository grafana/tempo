package tracediff

import (
	"errors"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffRejectsNilInputs(t *testing.T) {
	tests := []struct {
		name    string
		base    *tempopb.Trace
		compare *tempopb.Trace
	}{
		{
			name:    "nil base",
			base:    nil,
			compare: &tempopb.Trace{},
		},
		{
			name:    "nil compare",
			base:    &tempopb.Trace{},
			compare: nil,
		},
		{
			name:    "both nil",
			base:    nil,
			compare: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Diff(tt.base, tt.compare, FormatTracePatchV0)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrNilTrace))
			assert.Nil(t, got)
		})
	}
}

func TestDiffEmptyTracesReturnsEmptyResult(t *testing.T) {
	got, err := Diff(&tempopb.Trace{}, &tempopb.Trace{}, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, &Result{
		Version:  VersionTracePatchV0,
		Modified: []ModifiedSpan{},
		Added:    []SpanChange{},
		Removed:  []SpanChange{},
		Warnings: []Warning{},
	}, got)
}

func TestDiffPopulatesTraceMeta(t *testing.T) {
	base := traceWithSpans(
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		[]byte{0x01},
		[]byte{0x02},
	)
	compare := traceWithSpans(
		[]byte{0x10, 0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
		[]byte{0x03},
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, TraceMeta{TraceID: "0102030405060708090a0b0c0d0e0f10", SpanCount: 2}, got.Base)
	assert.Equal(t, TraceMeta{TraceID: "100f0e0d0c0b0a090807060504030201", SpanCount: 1}, got.Compare)
}

func TestDiffIdenticalTracesHaveNoChanges(t *testing.T) {
	base := traceWithSpans(
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		[]byte("root"),
		[]byte("child"),
	)
	compare := traceWithSpans(
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		[]byte("root"),
		[]byte("child"),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:   2,
		SpanCountB:   2,
		MatchedSpans: 2,
	}, got.Stats)
	assert.Empty(t, got.Modified)
	assert.Empty(t, got.Added)
	assert.Empty(t, got.Removed)
}

func TestDiffReportsAddedAndRemovedSpans(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "old_child", "root", "inventory", "old reserve", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "new_child", "root", "inventory", "new reserve", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:   2,
		SpanCountB:   2,
		MatchedSpans: 1,
		AddedSpans:   1,
		RemovedSpans: 1,
	}, got.Stats)
	require.Len(t, got.Added, 1)
	assert.Equal(t, "new reserve", got.Added[0].Span.Name)
	assert.Equal(t, SpanTarget{Type: TargetSpan, ParentPath: []int{0}, Index: intPtr(0)}, got.Added[0].Target)
	require.Len(t, got.Removed, 1)
	assert.Equal(t, "old reserve", got.Removed[0].Span.Name)
	assert.Equal(t, SpanTarget{Type: TargetSpan, Path: []int{0, 0}}, got.Removed[0].Target)
}

func TestDiffMatchesUnchangedSiblingsAfterInsertedSibling(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "base-root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "base-a", "base-root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "base-b", "base-root", "inventory", "reserve", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "compare-root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "compare-x", "compare-root", "cache", "warm", tracev1.Span_SPAN_KIND_CLIENT, 5, 10, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "compare-a", "compare-root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "compare-b", "compare-root", "inventory", "reserve", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:   3,
		SpanCountB:   4,
		MatchedSpans: 3,
		AddedSpans:   1,
	}, got.Stats)
	require.Len(t, got.Added, 1)
	assert.Equal(t, "warm", got.Added[0].Span.Name)
	assert.Equal(t, SpanTarget{Type: TargetSpan, ParentPath: []int{0}, Index: intPtr(0)}, got.Added[0].Target)
	assert.Empty(t, got.Removed)
	assert.Empty(t, got.Modified)
}

func TestDiffMatchesUnchangedSubtreeAfterInsertedAncestor(t *testing.T) {
	// root -> A -> B -> C   vs   root -> X -> A -> B -> C
	// X is inserted between the root and A; A, B, C are byte-identical, just one
	// level deeper. The only real change is "X added". A matcher that keys on the
	// full root-to-span path shifts every descendant's key and reports them all as
	// removed+added (a cascade). The correct result is 1 added, the rest matched.
	baseTraceID := []byte("trace-id-0000001")
	cmpTraceID := []byte("trace-id-0000002")
	base := traceWithNamedSpans(
		spanForNormalizeTest(baseTraceID, "base-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-a", "base-root", "auth", "authorize", tracev1.Span_SPAN_KIND_SERVER, 10, 180, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-b", "base-a", "inventory", "reserve", tracev1.Span_SPAN_KIND_SERVER, 20, 150, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-c", "base-b", "db", "persist", tracev1.Span_SPAN_KIND_SERVER, 30, 120, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(cmpTraceID, "cmp-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(cmpTraceID, "cmp-x", "cmp-root", "proxy", "forward", tracev1.Span_SPAN_KIND_SERVER, 5, 190, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(cmpTraceID, "cmp-a", "cmp-x", "auth", "authorize", tracev1.Span_SPAN_KIND_SERVER, 10, 180, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(cmpTraceID, "cmp-b", "cmp-a", "inventory", "reserve", tracev1.Span_SPAN_KIND_SERVER, 20, 150, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(cmpTraceID, "cmp-c", "cmp-b", "db", "persist", tracev1.Span_SPAN_KIND_SERVER, 30, 120, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:   4,
		SpanCountB:   5,
		MatchedSpans: 4, // root, authorize, reserve, persist
		AddedSpans:   1, // forward (X)
	}, got.Stats)
	require.Len(t, got.Added, 1)
	assert.Equal(t, "forward", got.Added[0].Span.Name)
	assert.Empty(t, got.Removed)
	assert.Empty(t, got.Modified)
}

func TestDiffReportsAddedSubtreeInParentBeforeChildOrder(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "base-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "cmp-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "cmp-recommend", "cmp-root", "recommendations", "recommend", tracev1.Span_SPAN_KIND_SERVER, 10, 120, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "cmp-cache", "cmp-recommend", "cache", "lookup recommendations", tracev1.Span_SPAN_KIND_CLIENT, 20, 60, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "cmp-db", "cmp-cache", "db", "SELECT recommendations", tracev1.Span_SPAN_KIND_CLIENT, 30, 50, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:   1,
		SpanCountB:   4,
		MatchedSpans: 1,
		AddedSpans:   3,
	}, got.Stats)
	require.Len(t, got.Added, 3)
	assert.Equal(t, []string{"recommend", "lookup recommendations", "SELECT recommendations"}, []string{
		got.Added[0].Span.Name,
		got.Added[1].Span.Name,
		got.Added[2].Span.Name,
	})
	assert.Equal(t, SpanTarget{Type: TargetSpan, ParentPath: []int{0}, Index: intPtr(0)}, got.Added[0].Target)
	assert.Equal(t, SpanTarget{Type: TargetSpan, ParentPath: []int{0, 0}, Index: intPtr(0)}, got.Added[1].Target)
	assert.Equal(t, SpanTarget{Type: TargetSpan, ParentPath: []int{0, 0, 0}, Index: intPtr(0)}, got.Added[2].Target)
	assert.Empty(t, got.Removed)
	assert.Empty(t, got.Modified)
}

func TestDiffMatchesDuplicateSpanIdentityByParent(t *testing.T) {
	// Both branches contain a db span with the same logical identity. When the
	// alpha branch is removed, the db span under beta should match the db span
	// under beta, not the first db span in pre-order.
	baseTraceID := []byte("trace-id-0000001")
	cmpTraceID := []byte("trace-id-0000002")
	base := traceWithNamedSpans(
		spanForNormalizeTest(baseTraceID, "base-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-alpha", "base-root", "alpha", "call alpha", tracev1.Span_SPAN_KIND_SERVER, 10, 80, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-alpha-db", "base-alpha", "db", "SELECT users", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-beta", "base-root", "beta", "call beta", tracev1.Span_SPAN_KIND_SERVER, 100, 180, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-beta-db", "base-beta", "db", "SELECT users", tracev1.Span_SPAN_KIND_CLIENT, 110, 150, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(cmpTraceID, "cmp-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(cmpTraceID, "cmp-beta", "cmp-root", "beta", "call beta", tracev1.Span_SPAN_KIND_SERVER, 100, 180, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(cmpTraceID, "cmp-beta-db", "cmp-beta", "db", "SELECT users", tracev1.Span_SPAN_KIND_CLIENT, 110, 150, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:   5,
		SpanCountB:   3,
		MatchedSpans: 3,
		RemovedSpans: 2,
	}, got.Stats)
	assert.Empty(t, got.Added)
	assert.Empty(t, got.Modified)
	require.Len(t, got.Removed, 2)
	removedDBDurations := make([]int64, 0, 1)
	for _, removed := range got.Removed {
		if removed.Span.Service == "db" && removed.Span.Name == "SELECT users" {
			removedDBDurations = append(removedDBDurations, removed.Span.DurationNanos)
		}
	}
	assert.Equal(t, []int64{10_000_000}, removedDBDurations)
}

func TestDiffWarnsAndUsesRawHighCardinalitySpanName(t *testing.T) {
	// The span name embeds a per-request ID. That is an instrumentation problem:
	// IDs belong in attributes, not span names. The diff should preserve raw-name
	// matching for correctness and warn that equivalent operations may be reported
	// as added/removed.
	baseTraceID := []byte("trace-id-0000001")
	cmpTraceID := []byte("trace-id-0000002")
	base := traceWithNamedSpans(
		spanForNormalizeTest(baseTraceID, "base-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(baseTraceID, "base-db", "base-root", "db", "SELECT id=3f2a1b9c-0001", tracev1.Span_SPAN_KIND_CLIENT, 10, 80, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(cmpTraceID, "cmp-root", "", "gateway", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 200, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(cmpTraceID, "cmp-db", "cmp-root", "db", "SELECT id=7c4e88d0-0002", tracev1.Span_SPAN_KIND_CLIENT, 10, 80, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:   2,
		SpanCountB:   2,
		MatchedSpans: 1,
		AddedSpans:   1,
		RemovedSpans: 1,
	}, got.Stats)
	require.Len(t, got.Added, 1)
	assert.Equal(t, "SELECT id=7c4e88d0-0002", got.Added[0].Span.Name)
	require.Len(t, got.Removed, 1)
	assert.Equal(t, "SELECT id=3f2a1b9c-0001", got.Removed[0].Span.Name)
	assert.Empty(t, got.Modified)
	require.Len(t, got.Warnings, 1)
	assert.Equal(t, WarningHighCardinalitySpanName, got.Warnings[0].Code)
	assert.Contains(t, got.Warnings[0].Message, "SELECT id=3f2a1b9c-0001")
	assert.Contains(t, got.Warnings[0].Message, "raw span names")
}

func TestDiffReportsModifiedSpanFields(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 250, tracev1.Status_STATUS_CODE_ERROR),
	)

	got, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:    1,
		SpanCountB:    1,
		MatchedSpans:  1,
		ModifiedSpans: 1,
		FieldChanges:  2,
	}, got.Stats)
	require.Len(t, got.Modified, 1)
	assert.Equal(t, SpanRef{Path: []int{0}, Service: "checkout", Name: "POST /checkout", Kind: "server"}, got.Modified[0].Span)
	assert.Equal(t, []Change{
		{
			Op:     OperationModify,
			Target: Target{Type: TargetField, Name: "duration_nanos"},
			Before: int64(100_000_000),
			After:  int64(250_000_000),
		},
		{
			Op:     OperationModify,
			Target: Target{Type: TargetField, Name: "status"},
			Before: "ok",
			After:  "error",
		},
	}, got.Modified[0].Changes)
}

func TestDiffReportsSpanAttributeChanges(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	baseSpan := spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	baseSpan.Attributes = append(baseSpan.Attributes,
		stringAttribute("removed.attr", "gone"),
		stringAttribute("version", "v1"),
	)
	compareSpan := spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	compareSpan.Attributes = append(compareSpan.Attributes,
		stringAttribute("added.attr", "new"),
		stringAttribute("version", "v2"),
	)

	got, err := Diff(traceWithNamedSpans(baseSpan), traceWithNamedSpans(compareSpan), FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:       1,
		SpanCountB:       1,
		MatchedSpans:     1,
		ModifiedSpans:    1,
		AttributeChanges: 3,
	}, got.Stats)
	require.Len(t, got.Modified, 1)
	assert.Equal(t, []Change{
		{
			Op:     OperationAdd,
			Target: Target{Type: TargetAttribute, Scope: "span", Key: "added.attr"},
			Before: nil,
			After:  "new",
		},
		{
			Op:     OperationRemove,
			Target: Target{Type: TargetAttribute, Scope: "span", Key: "removed.attr"},
			Before: "gone",
			After:  nil,
		},
		{
			Op:     OperationModify,
			Target: Target{Type: TargetAttribute, Scope: "span", Key: "version"},
			Before: "v1",
			After:  "v2",
		},
	}, got.Modified[0].Changes)
}

func TestDiffReportsArraySpanAttributeChanges(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	baseSpan := spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	baseSpan.Attributes = append(baseSpan.Attributes,
		stringArrayAttribute("feature.flags", "a", "b"),
	)
	compareSpan := spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	compareSpan.Attributes = append(compareSpan.Attributes,
		stringArrayAttribute("feature.flags", "a", "c"),
	)

	got, err := Diff(traceWithNamedSpans(baseSpan), traceWithNamedSpans(compareSpan), FormatTracePatchV0)
	require.NoError(t, err)

	assert.Equal(t, Stats{
		SpanCountA:       1,
		SpanCountB:       1,
		MatchedSpans:     1,
		ModifiedSpans:    1,
		AttributeChanges: 1,
	}, got.Stats)
	require.Len(t, got.Modified, 1)
	assert.Equal(t, []Change{
		{
			Op:     OperationModify,
			Target: Target{Type: TargetAttribute, Scope: "span", Key: "feature.flags"},
			Before: []any{"a", "b"},
			After:  []any{"a", "c"},
		},
	}, got.Modified[0].Changes)
}

func traceWithSpans(traceID []byte, spanIDs ...[]byte) *tempopb.Trace {
	spans := make([]*tracev1.Span, 0, len(spanIDs))
	for _, spanID := range spanIDs {
		spans = append(spans, &tracev1.Span{
			TraceId: traceID,
			SpanId:  spanID,
		})
	}

	return traceWithNamedSpans(spans...)
}

func traceWithNamedSpans(spans ...*tracev1.Span) *tempopb.Trace {
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				ScopeSpans: []*tracev1.ScopeSpans{
					{Spans: spans},
				},
			},
		},
	}
}

func intPtr(v int) *int {
	return &v
}

func stringArrayAttribute(key string, values ...string) *commonv1.KeyValue {
	arrayValues := make([]*commonv1.AnyValue, 0, len(values))
	for _, value := range values {
		arrayValues = append(arrayValues, &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		})
	}
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_ArrayValue{
				ArrayValue: &commonv1.ArrayValue{Values: arrayValues},
			},
		},
	}
}

func TestValuesEqual(t *testing.T) {
	// 2^53 is the largest integer float64 represents exactly; 2^53 and 2^53+1
	// collapse to the same float64, so comparing int64s as float would miss the
	// difference.
	const maxExactFloat = int64(1) << 53

	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{name: "nil equal", a: nil, b: nil, want: true},
		{name: "nil vs value", a: nil, b: int64(0), want: false},
		{name: "string equal", a: "x", b: "x", want: true},
		{name: "string unequal", a: "x", b: "y", want: false},
		{name: "bool equal", a: true, b: true, want: true},
		{name: "int64 equal", a: int64(5), b: int64(5), want: true},
		{name: "int64 unequal", a: int64(5), b: int64(6), want: false},
		{name: "float64 equal", a: 2.5, b: 2.5, want: true},
		{name: "float64 unequal", a: 2.5, b: 2.6, want: false},
		{name: "int64 vs float64 same value", a: int64(80), b: 80.0, want: true},
		{name: "float64 vs int64 same value", a: 80.0, b: int64(80), want: true},
		{name: "int64 vs float64 different value", a: int64(80), b: 81.0, want: false},
		{name: "large near-equal int64 stays exact", a: maxExactFloat, b: maxExactFloat + 1, want: false},
		{name: "numeric vs string", a: int64(80), b: "80", want: false},
		{name: "bytes equal", a: []byte{1, 2}, b: []byte{1, 2}, want: true},
		{name: "bytes unequal", a: []byte{1, 2}, b: []byte{1, 3}, want: false},
		{name: "array with cross-type numeric equal", a: []any{int64(1), "x"}, b: []any{1.0, "x"}, want: true},
		{name: "array unequal", a: []any{int64(1)}, b: []any{int64(2)}, want: false},
		{name: "map with cross-type numeric equal", a: map[string]any{"k": int64(3)}, b: map[string]any{"k": 3.0}, want: true},
		{name: "map unequal", a: map[string]any{"k": int64(3)}, b: map[string]any{"k": int64(4)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, valuesEqual(tt.a, tt.b))
		})
	}
}
