package tracediff

import (
	"math"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTraceAssignsStructuralPaths(t *testing.T) {
	trace := traceForNormalizeTest()

	got, warnings := normalizeTrace(trace)
	require.Empty(t, warnings)

	paths := map[string][]int{}
	for _, span := range got.spans {
		paths[span.spanID] = span.ref.Path
	}

	require.Len(t, paths, 5)
	assert.Equal(t, []int{0}, paths["root"])
	assert.Equal(t, []int{0, 0}, paths["auth"])
	assert.Equal(t, []int{0, 1}, paths["inventory"])
	assert.Equal(t, []int{0, 1, 0}, paths["reserve"])
	assert.Equal(t, []int{0, 2}, paths["payment"])
}

func TestNormalizeTraceBuildsSpanRefsAndSnapshots(t *testing.T) {
	trace := traceForNormalizeTest()

	got, warnings := normalizeTrace(trace)
	require.Empty(t, warnings)

	byID := map[string]normalizedSpan{}
	for _, span := range got.spans {
		byID[span.spanID] = span
	}

	reserve := byID["reserve"]
	assert.Equal(t, SpanRef{
		Path:    []int{0, 1, 0},
		Service: "inventory",
		Name:    "reserve",
		Kind:    "client",
	}, reserve.ref)
	assert.Equal(t, SpanSnapshot{
		Path:          []int{0, 1, 0},
		Service:       "inventory",
		Name:          "reserve",
		Kind:          "client",
		DurationNanos: 25_000_000,
		Status:        "error",
	}, reserve.snapshot)
}

func TestNormalizeTraceHandlesCyclicParents(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							spanForNormalizeTest(traceID, "a", "b", "svc-a", "a", tracev1.Span_SPAN_KIND_CLIENT, 0, 10, tracev1.Status_STATUS_CODE_OK),
							spanForNormalizeTest(traceID, "b", "a", "svc-b", "b", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
						},
					},
				},
			},
		},
	}

	got, warnings := normalizeTrace(trace)
	require.Empty(t, warnings)

	paths := map[string][]int{}
	for _, span := range got.spans {
		paths[span.spanID] = span.ref.Path
	}
	require.Len(t, paths, 2)
	assert.NotEmpty(t, paths["a"])
	assert.NotEmpty(t, paths["b"])
}

func traceForNormalizeTest() *tempopb.Trace {
	traceID := []byte("trace-id-0000001")
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						stringAttribute("service.name", "checkout"),
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							spanForNormalizeTest(traceID, "payment", "root", "payment", "charge", tracev1.Span_SPAN_KIND_CLIENT, 30, 45, tracev1.Status_STATUS_CODE_OK),
							spanForNormalizeTest(traceID, "reserve", "inventory", "inventory", "reserve", tracev1.Span_SPAN_KIND_CLIENT, 20, 45, tracev1.Status_STATUS_CODE_ERROR),
							spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
							spanForNormalizeTest(traceID, "inventory", "root", "inventory", "reserve inventory", tracev1.Span_SPAN_KIND_CLIENT, 20, 50, tracev1.Status_STATUS_CODE_OK),
							spanForNormalizeTest(traceID, "auth", "root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
						},
					},
				},
			},
		},
	}
}

const normalizeTestTimeOffsetMs = uint64(1_700_000_000_000)

func spanForNormalizeTest(traceID []byte, spanID, parentID, service, name string, kind tracev1.Span_SpanKind, start, end uint64, status tracev1.Status_StatusCode) *tracev1.Span {
	return &tracev1.Span{
		TraceId:           traceID,
		SpanId:            []byte(spanID),
		ParentSpanId:      []byte(parentID),
		Name:              name,
		Kind:              kind,
		StartTimeUnixNano: (normalizeTestTimeOffsetMs + start) * 1_000_000,
		EndTimeUnixNano:   (normalizeTestTimeOffsetMs + end) * 1_000_000,
		Attributes: []*commonv1.KeyValue{
			stringAttribute("service.name", service),
		},
		Status: &tracev1.Status{Code: status},
	}
}

func stringAttribute(key, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		},
	}
}

func TestSpanNameHasHighCardinalityToken(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "canonical UUID", in: "GET /users/550e8400-e29b-41d4-a716-446655440000", want: true},
		{name: "trace ID hex token", in: "GET /user/3f2a1b9c0d1e2f3a", want: true},
		{name: "explicit ID key with hex value", in: "SELECT id=3f2a1b9c-0001", want: true},
		{name: "explicit ID key with numeric value", in: "SELECT id=123456", want: true},
		{name: "long numeric path segment", in: "GET /orders/123456", want: true},
		{name: "low-cardinality route", in: "GET /checkout", want: false},
		{name: "short numeric path segment", in: "GET /orders/42", want: false},
		{name: "versioned route", in: "GET /api/v10/users", want: false},
		{name: "single-digit product name", in: "s3 upload", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, spanNameHasHighCardinalityToken(tc.in))
		})
	}
}

func TestInvalidDurationExamplesAreDeterministic(t *testing.T) {
	invalid := []spanLogicalKey{
		{service: "z", name: "beta", kind: "client"},
		{service: "a", name: "zeta", kind: "server"},
		{service: "a", name: "alpha", kind: "producer"},
		{service: "a", name: "alpha", kind: "consumer"},
	}

	assert.Equal(
		t,
		`["alpha" service "a" kind "consumer"; "alpha" service "a" kind "producer"; "zeta" service "a" kind "server"; and 1 more]`,
		invalidDurationExamples(invalid, 3),
	)
}

func TestDurationNanos(t *testing.T) {
	// A realistic (~2023) start in unix nanos; an unset (0) start against an end
	// this large would report ~decades if not guarded.
	const realisticStart = uint64(1_700_000_000_000_000_000)
	tests := []struct {
		name  string
		start uint64
		end   uint64
		want  int64
	}{
		{name: "normal span", start: realisticStart, end: realisticStart + 100_000_000, want: 100_000_000},
		{name: "zero-length span", start: realisticStart, end: realisticStart, want: 0},
		{name: "unset start with realistic end", start: 0, end: realisticStart, want: 0},
		{name: "unset start with small end", start: 0, end: 90_000_000, want: 0},
		{name: "both unset", start: 0, end: 0, want: 0},
		{name: "end before start", start: realisticStart + 100, end: realisticStart, want: 0},
		{name: "maximum int64 duration", start: 1, end: uint64(math.MaxInt64) + 1, want: math.MaxInt64},
		{name: "duration exceeds int64", start: 1, end: uint64(math.MaxInt64) + 2, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := &tracev1.Span{StartTimeUnixNano: tt.start, EndTimeUnixNano: tt.end}
			assert.Equal(t, tt.want, durationNanos(span))
		})
	}
}
