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
			Target: Target{Type: TargetField, Name: "duration_ms"},
			Before: int64(100),
			After:  int64(250),
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
