package tracefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
)

func TestProtoSpanAttributeResolution(t *testing.T) {
	resourceAttrs := resourceAttributes(nil) // exercise the nil-resource path
	require.Nil(t, resourceAttrs)

	span := &tracev1.Span{
		TraceId:           []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
		SpanId:            []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		ParentSpanId:      []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
		Name:              "GET /api",
		Kind:              tracev1.Span_SPAN_KIND_SERVER,
		StartTimeUnixNano: 1000,
		EndTimeUnixNano:   1500,
		Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_ERROR},
		Attributes: []*commonv1.KeyValue{
			{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "GET"}}},
		},
		Events: []*tracev1.Span_Event{
			{Name: "exception", TimeUnixNano: 1200},
		},
		Links: []*tracev1.Span_Link{
			{
				TraceId: []byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf, 0xb0},
				SpanId:  []byte{0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8},
			},
		},
	}

	scope := &commonv1.InstrumentationScope{Name: "otelgo", Version: "1.2.3"}
	ps := newProtoSpan(span, map[traceql.Attribute]traceql.Static{
		traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, "service.name"): traceql.NewStaticString("checkout"),
	}, nil, scope)

	assert.Equal(t, span.SpanId, ps.ID())
	assert.Equal(t, uint64(500), ps.DurationNanos())

	// span attribute by explicit span scope.
	v, ok := ps.AttributeFor(traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, "http.method"))
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString("GET"), v)

	// span attribute via unscoped fallback.
	v, ok = ps.AttributeFor(traceql.NewScopedAttribute(traceql.AttributeScopeNone, false, "http.method"))
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString("GET"), v)

	// resource attribute via unscoped fallback.
	v, ok = ps.AttributeFor(traceql.NewScopedAttribute(traceql.AttributeScopeNone, false, "service.name"))
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString("checkout"), v)

	// intrinsics.
	v, ok = ps.AttributeFor(traceql.IntrinsicNameAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString("GET /api"), v)

	v, ok = ps.AttributeFor(traceql.IntrinsicStatusAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticStatus(traceql.StatusError), v)

	v, ok = ps.AttributeFor(traceql.IntrinsicKindAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticKind(traceql.KindServer), v)

	// missing attribute.
	_, ok = ps.AttributeFor(traceql.NewScopedAttribute(traceql.AttributeScopeNone, false, "does.not.exist"))
	assert.False(t, ok)

	// id intrinsics are hex strings, matching the storage layer.
	v, ok = ps.AttributeFor(traceql.IntrinsicSpanIDAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString(util.SpanIDToHexString(span.SpanId)), v)

	v, ok = ps.AttributeFor(traceql.IntrinsicParentIDAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString(util.SpanIDToHexString(span.ParentSpanId)), v)

	v, ok = ps.AttributeFor(traceql.IntrinsicTraceIDAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString(util.TraceIDToHexString(span.TraceId)), v)

	// instrumentation scope intrinsics.
	v, ok = ps.AttributeFor(traceql.IntrinsicInstrumentationNameAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString("otelgo"), v)

	v, ok = ps.AttributeFor(traceql.IntrinsicInstrumentationVersionAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString("1.2.3"), v)

	// event intrinsics from the first event.
	v, ok = ps.AttributeFor(traceql.IntrinsicEventNameAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString("exception"), v)

	v, ok = ps.AttributeFor(traceql.IntrinsicEventTimeSinceStartAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticDuration(200), v)

	// link intrinsics from the first link.
	v, ok = ps.AttributeFor(traceql.IntrinsicLinkTraceIDAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString(util.TraceIDToHexString(span.Links[0].TraceId)), v)

	v, ok = ps.AttributeFor(traceql.IntrinsicLinkSpanIDAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticString(util.SpanIDToHexString(span.Links[0].SpanId)), v)
}

func TestProtoSpanAllAttributesMergesSharedMaps(t *testing.T) {
	// span-local, resource, and trace maps are stored separately; AllAttributes/AllAttributesFunc must
	// present the merged union, and the two methods must agree.
	resourceAttrs := map[traceql.Attribute]traceql.Static{
		traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, "service.name"): traceql.NewStaticString("checkout"),
	}
	traceAttrs := map[traceql.Attribute]traceql.Static{
		traceql.IntrinsicTraceRootSpanAttribute: traceql.NewStaticString("span-A"),
	}
	span := &tracev1.Span{
		SpanId: []byte{0x01},
		Name:   "GET /api",
		Attributes: []*commonv1.KeyValue{
			{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "GET"}}},
		},
	}
	ps := newProtoSpan(span, resourceAttrs, traceAttrs, nil)

	// trace intrinsic resolves via the shared trace map (the unit resolution test uses a nil trace map).
	v, ok := ps.AttributeFor(traceql.IntrinsicTraceRootSpanAttribute)
	require.True(t, ok)
	require.Equal(t, traceql.NewStaticString("span-A"), v)

	all := ps.AllAttributes()
	require.Equal(t, traceql.NewStaticString("GET"), all[traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, "http.method")], "span attr present")
	require.Equal(t, traceql.NewStaticString("checkout"), all[traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, "service.name")], "resource attr present")
	require.Equal(t, traceql.NewStaticString("span-A"), all[traceql.IntrinsicTraceRootSpanAttribute], "trace intrinsic present")
	require.Equal(t, traceql.NewStaticString("GET /api"), all[traceql.IntrinsicNameAttribute], "per-span intrinsic present")

	// AllAttributesFunc must visit exactly the same set AllAttributes returns.
	seen := make(map[traceql.Attribute]traceql.Static)
	ps.AllAttributesFunc(func(a traceql.Attribute, s traceql.Static) { seen[a] = s })
	require.Equal(t, all, seen)
}

func TestSpanKindMapping(t *testing.T) {
	cases := map[tracev1.Span_SpanKind]traceql.Kind{
		tracev1.Span_SPAN_KIND_UNSPECIFIED: traceql.KindUnspecified,
		tracev1.Span_SPAN_KIND_INTERNAL:    traceql.KindInternal,
		tracev1.Span_SPAN_KIND_CLIENT:      traceql.KindClient,
		tracev1.Span_SPAN_KIND_SERVER:      traceql.KindServer,
		tracev1.Span_SPAN_KIND_PRODUCER:    traceql.KindProducer,
		tracev1.Span_SPAN_KIND_CONSUMER:    traceql.KindConsumer,
	}
	for in, want := range cases {
		assert.Equal(t, want, spanKindToTraceql(in), "kind %v", in)
	}
}

func TestSpanStatusMapping(t *testing.T) {
	cases := map[tracev1.Status_StatusCode]traceql.Status{
		tracev1.Status_STATUS_CODE_UNSET: traceql.StatusUnset,
		tracev1.Status_STATUS_CODE_OK:    traceql.StatusOk,
		tracev1.Status_STATUS_CODE_ERROR: traceql.StatusError,
	}
	for in, want := range cases {
		assert.Equal(t, want, spanStatusToTraceql(in), "status %v", in)
	}
}

func TestProtoSpanNilStatusIsUnset(t *testing.T) {
	ps := newProtoSpan(&tracev1.Span{SpanId: []byte{0x01}}, nil, nil, nil)
	v, ok := ps.AttributeFor(traceql.IntrinsicStatusAttribute)
	require.True(t, ok)
	assert.Equal(t, traceql.NewStaticStatus(traceql.StatusUnset), v)
}
