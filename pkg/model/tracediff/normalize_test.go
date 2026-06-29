package tracediff

import (
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

func spanForNormalizeTest(traceID []byte, spanID, parentID, service, name string, kind tracev1.Span_SpanKind, start, end uint64, status tracev1.Status_StatusCode) *tracev1.Span {
	return &tracev1.Span{
		TraceId:           traceID,
		SpanId:            []byte(spanID),
		ParentSpanId:      []byte(parentID),
		Name:              name,
		Kind:              kind,
		StartTimeUnixNano: start * 1_000_000,
		EndTimeUnixNano:   end * 1_000_000,
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
