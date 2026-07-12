package tracefilter

import (
	"encoding/binary"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// testSpan is a compact span spec for building test traces.
type testSpan struct {
	id       byte
	parent   byte // 0 means root (no parent)
	statusOK bool
	attrs    map[string]any
}

// buildTrace builds a single-resource, single-scope trace from the given span specs.
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

// keptIDs returns the set of span ids present in a trace, as bytes.
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
		{name: "empty defaults keep_hierarchy false", vals: url.Values{}, want: Options{}},
		{name: "query only defaults keep_hierarchy false", vals: url.Values{"q": {"{ .a = 1 }"}}, want: Options{Query: "{ .a = 1 }"}},
		// a whitespace-only q is trimmed to empty, so it is treated as no filter and keep_hierarchy is ignored.
		{name: "whitespace-only q is treated as empty", vals: url.Values{"q": {"   "}, "keep_hierarchy": {"true"}}, want: Options{}},
		// surrounding whitespace on a real query is trimmed, not passed to the parser.
		{name: "surrounding whitespace on q is trimmed", vals: url.Values{"q": {"  { .a = 1 }  "}}, want: Options{Query: "{ .a = 1 }"}},
		{
			name: "query and explicit keep_hierarchy true",
			vals: url.Values{"q": {"{ .a = 1 }"}, "keep_hierarchy": {"true"}},
			want: Options{Query: "{ .a = 1 }", KeepHierarchy: true},
		},
		{
			name: "explicit keep_hierarchy false overrides default",
			vals: url.Values{"q": {"{ .a = 1 }"}, "keep_hierarchy": {"false"}},
			want: Options{Query: "{ .a = 1 }", KeepHierarchy: false},
		},
		{
			name:    "invalid keep_hierarchy with query",
			vals:    url.Values{"q": {"{ .a = 1 }"}, "keep_hierarchy": {"yes-please"}},
			wantErr: true,
		},
		{
			name: "invalid keep_hierarchy ignored without query",
			vals: url.Values{"keep_hierarchy": {"yes-please"}},
			want: Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OptionsFromValues(tt.vals)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCompileEmptyQueryIsPassthrough(t *testing.T) {
	f, err := Options{}.Compile()
	require.NoError(t, err)
	require.Nil(t, f, "empty query must compile to a nil filter (passthrough)")
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
	require.Same(t, trace, out)
}

func TestApplyQueryOnlyReturnsMatchedSpans(t *testing.T) {
	// trace: A(root) -> B -> C, only C has http.status_code=500.
	trace := buildTrace([]testSpan{
		{id: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 2, parent: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 3, parent: 2, attrs: map[string]any{"http.status_code": 500}},
	}, nil)

	f, err := Options{Query: `{ .http.status_code = 500 }`}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{3}, keptIDs(out), "only the matching span is returned without keep_hierarchy")
}

func TestApplyKeepHierarchyAddsAncestors(t *testing.T) {
	// trace: A(root) -> B -> C, only C matches. keep_hierarchy should also return A and B.
	trace := buildTrace([]testSpan{
		{id: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 2, parent: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 3, parent: 2, attrs: map[string]any{"http.status_code": 500}},
	}, nil)

	f, err := Options{Query: `{ .http.status_code = 500 }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{1, 2, 3}, keptIDs(out), "matched span plus its full ancestor path is returned")
}

func TestApplyKeepHierarchyMultipleBranches(t *testing.T) {
	// A(root) -> {B -> D(match), C -> E}; expect A, B, D (C and E dropped).
	trace := buildTrace([]testSpan{
		{id: 1},
		{id: 2, parent: 1},
		{id: 3, parent: 1},
		{id: 4, parent: 2, attrs: map[string]any{"match": true}},
		{id: 5, parent: 3},
	}, nil)

	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{1, 2, 4}, keptIDs(out))
}

func TestApplyKeepHierarchyFollowsAllParentsOfDuplicateID(t *testing.T) {
	// two spans share id 3 but have different parents (1 and 2); a match on id 3 must keep both
	// ancestor branches, not an arbitrary last-writer one.
	trace := buildTrace([]testSpan{
		{id: 1},
		{id: 2},
		{id: 3, parent: 1, attrs: map[string]any{"match": true}},
		{id: 3, parent: 2, attrs: map[string]any{"match": true}},
	}, nil)

	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{1, 2, 3, 3}, keptIDs(out))
}

func TestApplyNoMatchReturnsEmptyTrace(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1, attrs: map[string]any{"http.status_code": 200}}}, nil)

	f, err := Options{Query: `{ .http.status_code = 500 }`}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Empty(t, keptIDs(out))
}

func TestApplyDoesNotMutateInput(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1, attrs: map[string]any{"http.status_code": 200}},
		{id: 2, parent: 1, attrs: map[string]any{"http.status_code": 500}},
	}, nil)
	// snapshot the full wire encoding so any field mutation, not just a dropped span, is caught.
	before, err := trace.Marshal()
	require.NoError(t, err)

	f, err := Options{Query: `{ .http.status_code = 500 }`}.Compile()
	require.NoError(t, err)

	_, err = f.Process(trace)
	require.NoError(t, err)

	after, err := trace.Marshal()
	require.NoError(t, err)
	require.Equal(t, before, after, "input trace must be untouched")
}

func TestApplyMatchesOnResourceAttribute(t *testing.T) {
	trace := buildTrace([]testSpan{{id: 1}}, map[string]any{"service.name": "checkout"})

	f, err := Options{Query: `{ resource.service.name = "checkout" }`}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{1}, keptIDs(out))
}

func TestApplyMatchesOnIntrinsicStatus(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1, statusOK: true},
		{id: 2, statusOK: false}, // error
	}, nil)

	f, err := Options{Query: `{ status = error }`}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{2}, keptIDs(out))
}

func TestApplyKeepHierarchyToleratesMissingParent(t *testing.T) {
	// parent id 9 is absent; the walk must terminate and the phantom parent must not be emitted.
	trace := buildTrace([]testSpan{
		{id: 2, parent: 9, attrs: map[string]any{"match": true}},
	}, nil)

	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{2}, keptIDs(out))
}

func TestApplyKeepHierarchyTerminatesOnCycle(t *testing.T) {
	// 1 -> 2 -> 1 is a parent cycle. The walk must terminate (not hang) and keep both.
	trace := buildTrace([]testSpan{
		{id: 1, parent: 2, attrs: map[string]any{"match": true}},
		{id: 2, parent: 1},
	}, nil)

	f, err := Options{Query: `{ .match = true }`, KeepHierarchy: true}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{1, 2}, keptIDs(out))
}

func TestApplyToleratesNilAttributeValue(t *testing.T) {
	// a span attribute with a nil Value is representable OTLP; it must not panic.
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{{Key: "bad", Value: nil}}},
				ScopeSpans: []*tracev1.ScopeSpans{
					{Spans: []*tracev1.Span{
						{SpanId: []byte{1}, Attributes: []*commonv1.KeyValue{{Key: "x", Value: nil}, {Key: "match", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: true}}}}},
					}},
				},
			},
		},
	}

	f, err := Options{Query: `{ .match = true }`}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{1}, keptIDs(out))
}

func TestApplyTraceLevelIntrinsicRejected(t *testing.T) {
	// trace-level intrinsics need a whole-trace pass the per-span filter does not do, so they are
	// rejected at compile rather than silently matching nothing.
	for _, q := range []string{`{ trace:rootName = "span-B" }`, `{ trace:rootService = "checkout" }`, `{ trace:duration > 1s }`} {
		_, err := Options{Query: q}.Compile()
		require.Error(t, err, q)
	}
}

func TestApplySpanIDIntrinsic(t *testing.T) {
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				ScopeSpans: []*tracev1.ScopeSpans{
					{Spans: []*tracev1.Span{
						{SpanId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
						{SpanId: []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}},
					}},
				},
			},
		},
	}

	f, err := Options{Query: `{ span:id = "0102030405060708" }`}.Compile()
	require.NoError(t, err)

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.Len(t, out.ResourceSpans, 1)
	require.Len(t, out.ResourceSpans[0].ScopeSpans, 1)
	require.Len(t, out.ResourceSpans[0].ScopeSpans[0].Spans, 1)
	require.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, out.ResourceSpans[0].ScopeSpans[0].Spans[0].SpanId)
}

func TestApplyEmptyOrMatchAllReturnsFullTrace(t *testing.T) {
	// no q, empty q, {}, and { true } must all return the full trace.
	trace := buildTrace([]testSpan{
		{id: 1},
		{id: 2, parent: 1},
		{id: 3, parent: 2},
	}, nil)

	for _, q := range []string{"", "{}", "{ true }"} {
		f, err := Options{Query: q}.Compile()
		require.NoError(t, err)
		out, err := f.Process(trace) // Process is nil-safe, covering the empty-q passthrough.
		require.NoError(t, err)
		require.ElementsMatch(t, []byte{1, 2, 3}, keptIDs(out), "query %q should return the full trace", q)
	}
}

func TestKeepHierarchyIgnoredWhenQueryAbsent(t *testing.T) {
	trace := buildTrace([]testSpan{
		{id: 1},
		{id: 2, parent: 1},
		{id: 3, parent: 2},
	}, nil)

	f, err := Options{KeepHierarchy: true}.Compile()
	require.NoError(t, err)
	require.Nil(t, f, "no query means no filter regardless of keep_hierarchy")

	out, err := f.Process(trace)
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{1, 2, 3}, keptIDs(out))
}

// benchSpanID encodes i into an 8-byte span id so a large trace has unique ids.
func benchSpanID(i int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i)+1)
	return b
}

// buildBenchTrace builds an n-span trace: one root with n-1 children, every 10th child matching.
func buildBenchTrace(n int) *tempopb.Trace {
	spans := make([]*tracev1.Span, 0, n)
	spans = append(spans, &tracev1.Span{
		SpanId: benchSpanID(0), Name: "root", StartTimeUnixNano: 1000, EndTimeUnixNano: 2000,
	})
	for i := 1; i < n; i++ {
		code := int64(200)
		if i%10 == 0 {
			code = 500
		}
		spans = append(spans, &tracev1.Span{
			SpanId:            benchSpanID(i),
			ParentSpanId:      benchSpanID(0),
			Name:              "child",
			StartTimeUnixNano: 1000,
			EndTimeUnixNano:   2000,
			Attributes: []*commonv1.KeyValue{{
				Key:   "http.status_code",
				Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: code}},
			}},
		})
	}
	return &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{{
		Resource:   &resourcev1.Resource{Attributes: keyValues(map[string]any{"service.name": "bench"})},
		ScopeSpans: []*tracev1.ScopeSpans{{Spans: spans}},
	}}}
}

func BenchmarkProcess(b *testing.B) {
	trace := buildBenchTrace(2000)

	keep, err := Options{Query: `{ .http.status_code = 500 }`, KeepHierarchy: true}.Compile()
	require.NoError(b, err)
	flat, err := Options{Query: `{ .http.status_code = 500 }`, KeepHierarchy: false}.Compile()
	require.NoError(b, err)

	b.Run("keep_hierarchy=true", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := keep.Process(trace); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("keep_hierarchy=false", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := flat.Process(trace); err != nil {
				b.Fatal(err)
			}
		}
	})
}
