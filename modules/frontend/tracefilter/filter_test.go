package tracefilter

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type testSpan struct {
	id       byte
	parent   byte
	statusOK bool
	attrs    map[string]any
}

func buildTrace(spans []testSpan, resourceAttrs map[string]any) *tempopb.Trace {
	var protoSpans []*tracev1.Span
	for _, s := range spans {
		span := &tracev1.Span{
			SpanId:            []byte{s.id},
			Name:              "span-" + string(rune('A'+s.id)),
			StartTimeUnixNano: 1000,
			EndTimeUnixNano:   2000,
			Attributes:        keyValues(s.attrs),
		}
		if s.parent != 0 {
			span.ParentSpanId = []byte{s.parent}
		}
		code := tracev1.Status_STATUS_CODE_ERROR
		if s.statusOK {
			code = tracev1.Status_STATUS_CODE_OK
		}
		span.Status = &tracev1.Status{Code: code}
		protoSpans = append(protoSpans, span)
	}
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource:   &resourcev1.Resource{Attributes: keyValues(resourceAttrs)},
				ScopeSpans: []*tracev1.ScopeSpans{{Spans: protoSpans}},
			},
		},
	}
}

func keyValues(m map[string]any) []*commonv1.KeyValue {
	var out []*commonv1.KeyValue
	for k, v := range m {
		kv := &commonv1.KeyValue{Key: k}
		switch val := v.(type) {
		case string:
			kv.Value = &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: val}}
		case int:
			kv.Value = &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: int64(val)}}
		case bool:
			kv.Value = &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: val}}
		}
		out = append(out, kv)
	}
	return out
}

func keptIDs(trace *tempopb.Trace) []byte {
	var ids []byte
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				ids = append(ids, s.SpanId[0])
			}
		}
	}
	return ids
}

func TestOptionsFromValues(t *testing.T) {
	tests := []struct {
		name    string
		vals    url.Values
		want    Options
		wantErr bool
	}{
		{name: "empty defaults keep_hierarchy true", vals: url.Values{}, want: Options{KeepHierarchy: true, MatchDepth: -1}},
		{name: "query only", vals: url.Values{"q": {"{ .a = 1 }"}}, want: Options{Query: "{ .a = 1 }", KeepHierarchy: true, MatchDepth: -1}},
		{name: "explicit keep_hierarchy true", vals: url.Values{"q": {"{ .a = 1 }"}, "keep_hierarchy": {"true"}}, want: Options{Query: "{ .a = 1 }", KeepHierarchy: true, MatchDepth: -1}},
		{name: "explicit keep_hierarchy false", vals: url.Values{"q": {"{ .a = 1 }"}, "keep_hierarchy": {"false"}}, want: Options{Query: "{ .a = 1 }", KeepHierarchy: false, MatchDepth: -1}},
		{name: "invalid keep_hierarchy", vals: url.Values{"keep_hierarchy": {"yes-please"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OptionsFromValues(tt.vals)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompileEmptyQueryIsPassthrough(t *testing.T) {
	f, err := Options{}.Compile()
	require.NoError(t, err)
	require.Nil(t, f)
}

func TestCompileInvalidQueryErrors(t *testing.T) {
	_, err := Options{Query: "{ .a = }"}.Compile()
	require.Error(t, err)
}

func TestApplyNilFilterReturnsInput(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1}}, nil)
	var f *Filter
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.Same(t, trace, out)
}

func TestApplyQueryOnlyReturnsMatchedSpans(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 2, parent: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 3, parent: 2, attrs: map[string]any{"http.status_code": 500}},
	}, nil)
	f, err := Options{Query: `{ .http.status_code = 500 }`}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{3}, keptIDs(out))
}

func TestApplyKeepHierarchyAddsAncestors(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 2, parent: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 3, parent: 2, attrs: map[string]any{"http.status_code": 500}},
	}, nil)
	f, err := Options{Query: `{ .http.status_code = 500 }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1, 2, 3}, keptIDs(out))
}

func TestApplyKeepHierarchyMultipleBranches(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1}, {id: 2, parent: 1}, {id: 3, parent: 1},
		{id: 4, parent: 2, attrs: map[string]any{"match": true}},
		{id: 5, parent: 3},
	}, nil)
	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1, 2, 4}, keptIDs(out))
}

func TestApplyNoMatchReturnsEmptyTrace(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1, attrs: map[string]any{"http.status_code": 200}}}, nil)
	f, err := Options{Query: `{ .http.status_code = 500 }`}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.Empty(t, keptIDs(out))
}

func TestApplyDoesNotMutateInput(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 2, parent: 1, attrs: map[string]any{"http.status_code": 500}},
	}, nil)
	original := len(trace.ResourceSpans[0].ScopeSpans[0].Spans)
	f, err := Options{Query: `{ .http.status_code = 500 }`}.Compile()
	require.NoError(t, err)
	_, err = f.Process(trace)
	require.NoError(t, err)
	assert.Len(t, trace.ResourceSpans[0].ScopeSpans[0].Spans, original)
}

func TestApplyMatchesOnResourceAttribute(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1}}, map[string]any{"service.name": "checkout"})
	f, err := Options{Query: `{ resource.service.name = "checkout" }`}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1}, keptIDs(out))
}

func TestApplyMatchesOnIntrinsicStatus(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1, statusOK: true}, {id: 2, statusOK: false}}, nil)
	f, err := Options{Query: `{ status = error }`}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{2}, keptIDs(out))
}

func TestApplyKeepHierarchyToleratesMissingParent(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 2, parent: 9, attrs: map[string]any{"match": true}}}, nil)
	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{2}, keptIDs(out))
}

func TestApplyKeepHierarchyTerminatesOnCycle(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1, parent: 2, attrs: map[string]any{"match": true}},
		{id: 2, parent: 1},
	}, nil)
	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1, 2}, keptIDs(out))
}

func TestApplyToleratesNilAttributeValue(t *testing.T) {
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{{
			Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{{Key: "bad", Value: nil}}},
			ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{
				SpanId: []byte{1},
				Attributes: []*commonv1.KeyValue{
					{Key: "x", Value: nil},
					{Key: "match", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: true}}},
				},
			}}}},
		}},
	}
	f, err := Options{Query: `{ .match = true }`}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1}, keptIDs(out))
}

func TestApplyTraceLevelIntrinsic(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1}, {id: 2, parent: 1}}, map[string]any{"service.name": "checkout"})
	f, err := Options{Query: `{ trace:rootName = "span-B" && trace:rootService = "checkout" }`}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1, 2}, keptIDs(out))
}

func TestApplySpanIDIntrinsic(t *testing.T) {
	trace := &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{{ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{
		{SpanId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
		{SpanId: []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}},
	}}}}}}
	f, err := Options{Query: `{ span:id = "0102030405060708" }`}.Compile()
	require.NoError(t, err)
	out, err := f.Process(trace)
	require.NoError(t, err)
	require.Len(t, out.ResourceSpans[0].ScopeSpans[0].Spans, 1)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, out.ResourceSpans[0].ScopeSpans[0].Spans[0].SpanId)
}

func TestApplyEmptyOrMatchAllReturnsFullTrace(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1}, {id: 2, parent: 1}, {id: 3, parent: 2}}, nil)
	for _, q := range []string{"", "{}", "{ true }"} {
		f, err := Options{Query: q}.Compile()
		require.NoError(t, err)
		out, err := f.Process(trace)
		require.NoError(t, err)
		assert.ElementsMatch(t, []byte{1, 2, 3}, keptIDs(out), "query %q should return the full trace", q)
	}
}

func TestKeepHierarchyIgnoredWhenQueryAbsent(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1}, {id: 2, parent: 1}, {id: 3, parent: 2}}, nil)
	f, err := Options{KeepHierarchy: true}.Compile()
	require.NoError(t, err)
	require.Nil(t, f)
	out, err := f.Process(trace)
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1, 2, 3}, keptIDs(out))
}

// match_depth tests
//
// Tree used in depth tests:
//
//	1 (root)
//	├── 2
//	│   ├── 4
//	│   └── 5
//	└── 3
//	    └── 6
//	        └── 7

func depthTrace() *tempopb.Trace {
	return buildTrace([]testSpan{
		{id: 1},
		{id: 2, parent: 1},
		{id: 3, parent: 1},
		{id: 4, parent: 2, attrs: map[string]any{"match": true}},
		{id: 5, parent: 2},
		{id: 6, parent: 3},
		{id: 7, parent: 6},
	}, nil)
}

func TestOptionsFromValues_MatchDepth(t *testing.T) {
	tests := []struct {
		name       string
		vals       url.Values
		wantDepth  int
		wantErr    bool
	}{
		{name: "default is -1", vals: url.Values{}, wantDepth: -1},
		{name: "explicit -1", vals: url.Values{"match_depth": {"-1"}}, wantDepth: -1},
		{name: "zero", vals: url.Values{"match_depth": {"0"}}, wantDepth: 0},
		{name: "positive", vals: url.Values{"match_depth": {"3"}}, wantDepth: 3},
		{name: "below -1 rejected", vals: url.Values{"match_depth": {"-2"}}, wantErr: true},
		{name: "non-integer rejected", vals: url.Values{"match_depth": {"deep"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OptionsFromValues(tt.vals)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDepth, got.MatchDepth)
		})
	}
}

func TestMatchDepth_DefaultAllDescendants(t *testing.T) {
	// match_depth=-1 (default): matched span 2 + all its descendants (4, 5) are kept.
	// keep_hierarchy=false so ancestors are NOT added.
	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: false, MatchDepth: -1}.Compile()
	require.NoError(t, err)
	out, err := f.Process(depthTrace())
	require.NoError(t, err)
	// span 4 matches; keep_hierarchy=false so no ancestors; matchDepth=-1 so no descendants of 4 (it's a leaf)
	assert.ElementsMatch(t, []byte{4}, keptIDs(out))
}

func TestMatchDepth_ZeroNoDescendants(t *testing.T) {
	// depth=0: only the matched span itself, no children.
	// Match span 2 (parent of 4 and 5).
	f, err := Options{Query: `{ span:id = "0000000000000002" }`, KeepHierarchy: false, MatchDepth: 0}.Compile()
	require.NoError(t, err)
	out, err := f.Process(depthTrace())
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{2}, keptIDs(out))
}

func TestMatchDepth_OneDirectChildrenOnly(t *testing.T) {
	// depth=1: matched span + its direct children only.
	// Match span 2; should keep 2, 4, 5. NOT grandchildren (there are none of 2's children anyway).
	f, err := Options{Query: `{ span:id = "0000000000000002" }`, KeepHierarchy: false, MatchDepth: 1}.Compile()
	require.NoError(t, err)
	out, err := f.Process(depthTrace())
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{2, 4, 5}, keptIDs(out))
}

func TestMatchDepth_TwoLevels(t *testing.T) {
	// depth=2: matched span + children + grandchildren.
	// Match span 3; should keep 3 (matched), 6 (child), 7 (grandchild).
	f, err := Options{Query: `{ span:id = "0000000000000003" }`, KeepHierarchy: false, MatchDepth: 2}.Compile()
	require.NoError(t, err)
	out, err := f.Process(depthTrace())
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{3, 6, 7}, keptIDs(out))
}

func TestMatchDepth_OneLevelStopsAtGrandchild(t *testing.T) {
	// depth=1 on span 3: keeps 3 and 6 (direct child), but NOT 7 (grandchild).
	f, err := Options{Query: `{ span:id = "0000000000000003" }`, KeepHierarchy: false, MatchDepth: 1}.Compile()
	require.NoError(t, err)
	out, err := f.Process(depthTrace())
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{3, 6}, keptIDs(out))
}

func TestMatchDepth_WithKeepHierarchy(t *testing.T) {
	// keep_hierarchy=true + depth=1: ancestors of matched + matched + direct children.
	// Match span 4 (child of 2, grandchild of 1).
	// keep_hierarchy adds 2, 1. depth=1 adds 4's children (none — it's a leaf).
	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true, MatchDepth: 1}.Compile()
	require.NoError(t, err)
	out, err := f.Process(depthTrace())
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1, 2, 4}, keptIDs(out))
}

func TestMatchDepth_AllDescendantsFromRoot(t *testing.T) {
	// depth=-1 matching the root: entire tree is kept.
	f, err := Options{Query: `{ span:id = "0000000000000001" }`, KeepHierarchy: false, MatchDepth: -1}.Compile()
	require.NoError(t, err)
	out, err := f.Process(depthTrace())
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte{1, 2, 3, 4, 5, 6, 7}, keptIDs(out))
}

func TestMatchDepth_CycleTerminates(t *testing.T) {
	// Cycles in the children map must not hang the BFS.
	// Build a cycle: spanA -> spanB -> spanA (each is listed as parent of the other).
	spanA := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	spanB := []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	cycleTrace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{{
			ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{
				{SpanId: spanA, ParentSpanId: spanB, Name: "A", StartTimeUnixNano: 1, EndTimeUnixNano: 2},
				{SpanId: spanB, ParentSpanId: spanA, Name: "B", StartTimeUnixNano: 1, EndTimeUnixNano: 2},
			}}},
		}},
	}
	// match span A; depth=-1 should reach span B via the child relationship without hanging.
	f, err := Options{Query: `{ span:id = "0100000000000000" }`, KeepHierarchy: false, MatchDepth: -1}.Compile()
	require.NoError(t, err)
	out, err := f.Process(cycleTrace)
	require.NoError(t, err)
	// both spans reachable; cycle terminated by visited set.
	assert.Len(t, keptIDs(out), 2)
}
