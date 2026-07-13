package tracefilter

import (
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
)

// protoSpan is a proto-backed traceql.Span.
// A span is presented to the engine as one protoSpan per (event, link) combination.
type protoSpan struct {
	span            *tracev1.Span
	resource        *resourcev1.Resource
	instrumentation *commonv1.InstrumentationScope
	event           *tracev1.Span_Event // nil when the span has no events.
	link            *tracev1.Span_Link  // nil when the span has no links.
}

var _ traceql.Span = (*protoSpan)(nil)

func (s *protoSpan) AttributeFor(a traceql.Attribute) (traceql.Static, bool) {
	if a.Intrinsic != traceql.IntrinsicNone {
		return boundIntrinsicFor(s.span, s.instrumentation, s.event, s.link, a.Intrinsic)
	}
	switch a.Scope {
	case traceql.AttributeScopeSpan:
		return findKV(s.span.Attributes, a.Name)
	case traceql.AttributeScopeResource:
		return findKV(s.resource.GetAttributes(), a.Name)
	case traceql.AttributeScopeEvent:
		if s.event == nil {
			return traceql.NewStaticNil(), false
		}
		return findKV(s.event.Attributes, a.Name)
	case traceql.AttributeScopeLink:
		if s.link == nil {
			return traceql.NewStaticNil(), false
		}
		return findKV(s.link.Attributes, a.Name)
	case traceql.AttributeScopeInstrumentation:
		return findKV(s.instrumentation.GetAttributes(), a.Name)
	case traceql.AttributeScopeNone:
		// unscoped .x resolves against span, resource, then instrumentation - not event/link, mirroring a
		// real search's handling of an unscoped attribute.
		if v, ok := findKV(s.span.Attributes, a.Name); ok {
			return v, true
		}
		if v, ok := findKV(s.resource.GetAttributes(), a.Name); ok {
			return v, true
		}
		if v, ok := findKV(s.instrumentation.GetAttributes(), a.Name); ok {
			return v, true
		}
	}
	return traceql.NewStaticNil(), false
}

func (s *protoSpan) ID() []byte                 { return s.span.SpanId }
func (s *protoSpan) StartTimeUnixNanos() uint64 { return s.span.StartTimeUnixNano }
func (s *protoSpan) DurationNanos() uint64 {
	return s.span.EndTimeUnixNano - s.span.StartTimeUnixNano
}

// AllAttributes / AllAttributesFunc satisfy traceql.Span but are unreachable on the MatchSpans path.
func (s *protoSpan) AllAttributes() map[traceql.Attribute]traceql.Static { return nil }

func (s *protoSpan) AllAttributesFunc(func(traceql.Attribute, traceql.Static)) {}

// Stubs for the traceql.Span interface. Structural operators are rejected at compile, so these are unreachable.

func (s *protoSpan) DescendantOf(_, _ []traceql.Span, _, _, _ bool, _ []traceql.Span) []traceql.Span {
	return nil
}

func (s *protoSpan) SiblingOf(_, _ []traceql.Span, _, _ bool, _ []traceql.Span) []traceql.Span {
	return nil
}

func (s *protoSpan) ChildOf(_, _ []traceql.Span, _, _, _ bool, _ []traceql.Span) []traceql.Span {
	return nil
}

// boundIntrinsicFor resolves an intrinsic against the bound event/link, not a first-match across all of
// them, so an event intrinsic condition is decided by the same event as that binding's event attrs.
func boundIntrinsicFor(span *tracev1.Span, scope *commonv1.InstrumentationScope, event *tracev1.Span_Event, link *tracev1.Span_Link, ic traceql.Intrinsic) (traceql.Static, bool) {
	switch ic {
	case traceql.IntrinsicName:
		return traceql.NewStaticString(span.Name), true
	case traceql.IntrinsicDuration:
		// unguarded like storage's DurationNano: on a skewed span (end<start) the uint64 subtraction wraps, same as a real search.
		return traceql.NewStaticDuration(time.Duration(span.EndTimeUnixNano - span.StartTimeUnixNano)), true
	case traceql.IntrinsicKind:
		return traceql.NewStaticKind(kindFromOTLP(span.Kind)), true
	case traceql.IntrinsicSpanID:
		return traceql.NewStaticString(util.SpanIDToHexString(span.SpanId)), true
	case traceql.IntrinsicParentID:
		return traceql.NewStaticString(util.SpanIDToHexString(span.ParentSpanId)), true
	case traceql.IntrinsicStatus:
		if span.Status == nil {
			return traceql.NewStaticStatus(traceql.StatusUnset), true
		}
		return traceql.NewStaticStatus(statusFromOTLP(span.Status.Code)), true
	case traceql.IntrinsicStatusMessage:
		if span.Status == nil {
			return traceql.NewStaticString(""), true
		}
		return traceql.NewStaticString(span.Status.Message), true
	case traceql.IntrinsicInstrumentationName:
		if scope == nil {
			return traceql.NewStaticNil(), false
		}
		return traceql.NewStaticString(scope.Name), true
	case traceql.IntrinsicInstrumentationVersion:
		if scope == nil {
			return traceql.NewStaticNil(), false
		}
		return traceql.NewStaticString(scope.Version), true
	case traceql.IntrinsicEventName:
		if event == nil {
			return traceql.NewStaticNil(), false
		}
		return traceql.NewStaticString(event.Name), true
	case traceql.IntrinsicEventTimeSinceStart:
		if event == nil {
			return traceql.NewStaticNil(), false
		}
		// unguarded subtraction mirrors storage's eventToParquet - wraps on skewed timestamps like a real search.
		return traceql.NewStaticDuration(time.Duration(event.TimeUnixNano - span.StartTimeUnixNano)), true
	case traceql.IntrinsicLinkTraceID:
		if link == nil {
			return traceql.NewStaticNil(), false
		}
		return traceql.NewStaticString(util.TraceIDToHexString(link.TraceId)), true
	case traceql.IntrinsicLinkSpanID:
		if link == nil {
			return traceql.NewStaticNil(), false
		}
		return traceql.NewStaticString(util.SpanIDToHexString(link.SpanId)), true
	default:
		return traceql.NewStaticNil(), false
	}
}

// kindFromOTLP maps an OTLP span kind to the TraceQL Kind, mirroring the vparquet read path. Unknown
// values fall through to the raw integer.
func kindFromOTLP(k tracev1.Span_SpanKind) traceql.Kind {
	switch k {
	case tracev1.Span_SPAN_KIND_UNSPECIFIED:
		return traceql.KindUnspecified
	case tracev1.Span_SPAN_KIND_INTERNAL:
		return traceql.KindInternal
	case tracev1.Span_SPAN_KIND_SERVER:
		return traceql.KindServer
	case tracev1.Span_SPAN_KIND_CLIENT:
		return traceql.KindClient
	case tracev1.Span_SPAN_KIND_PRODUCER:
		return traceql.KindProducer
	case tracev1.Span_SPAN_KIND_CONSUMER:
		return traceql.KindConsumer
	default:
		return traceql.Kind(k)
	}
}

// statusFromOTLP maps an OTLP status code to the TraceQL Status, mirroring the vparquet read path.
func statusFromOTLP(c tracev1.Status_StatusCode) traceql.Status {
	switch c {
	case tracev1.Status_STATUS_CODE_UNSET:
		return traceql.StatusUnset
	case tracev1.Status_STATUS_CODE_OK:
		return traceql.StatusOk
	case tracev1.Status_STATUS_CODE_ERROR:
		return traceql.StatusError
	default:
		return traceql.Status(c)
	}
}

// maxBindingsPerSpan caps the event x link fan-out. The cap is far above any real span,
// so real traces are never truncated.
// A span past the cap is truncated and may under-match, the safe trade against unbounded allocation on the read path.
const maxBindingsPerSpan = 100_000

// expandSpanBindings appends one protoSpan per (event, link) combination of span. When expandElements
// is false (the filter reads no event/link scope) it appends a single binding, skipping the fan-out.
// The bool return reports whether the fan-out was truncated at maxBindingsPerSpan.
func expandSpanBindings(dst []traceql.Span, span *tracev1.Span, resource *resourcev1.Resource, scope *commonv1.InstrumentationScope, expandElements bool) ([]traceql.Span, bool) {
	if !expandElements {
		return append(dst, &protoSpan{span: span, resource: resource, instrumentation: scope}), false
	}

	eventCount := len(span.Events)
	if eventCount == 0 {
		eventCount = 1
	}
	linkCount := len(span.Links)
	if linkCount == 0 {
		linkCount = 1
	}
	emitted := 0
	for ei := 0; ei < eventCount; ei++ {
		var e *tracev1.Span_Event
		if len(span.Events) > 0 {
			e = span.Events[ei]
		}
		for li := 0; li < linkCount; li++ {
			if emitted >= maxBindingsPerSpan {
				return dst, true // truncate pathological fan-out - may under-match this span, but bounds memory.
			}
			var l *tracev1.Span_Link
			if len(span.Links) > 0 {
				l = span.Links[li]
			}
			dst = append(dst, &protoSpan{span: span, resource: resource, instrumentation: scope, event: e, link: l})
			emitted++
		}
	}
	return dst, false
}

// countSpans returns the total span count so newSpanIndex can pre-size idx.spans in one pass.
func countSpans(trace *tempopb.Trace) int {
	n := 0
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			n += len(ss.Spans)
		}
	}
	return n
}

// findKV linearly scans kvs for name, staticFromKeyValue handles nil/array collapsing.
func findKV(kvs []*commonv1.KeyValue, name string) (traceql.Static, bool) {
	for _, kv := range kvs {
		if kv.Key == name {
			return staticFromKeyValue(kv), true
		}
	}
	return traceql.NewStaticNil(), false
}

// staticFromKeyValue converts a proto attribute to a traceql.Static, collapsing single-element arrays
// to scalars the way vp5's read-side attributeCollector does.
func staticFromKeyValue(kv *commonv1.KeyValue) traceql.Static {
	if kv.Value == nil {
		return traceql.NewStaticNil()
	}
	if arr, ok := kv.Value.Value.(*commonv1.AnyValue_ArrayValue); ok {
		return staticFromArray(arr.ArrayValue)
	}
	return traceql.StaticFromAnyValue(kv.Value)
}

// staticFromArray builds a homogeneous array Static. A single element collapses to a scalar and a
// mixed-type (or empty) array is surfaced as nil, matching vp5's array storage.
func staticFromArray(arr *commonv1.ArrayValue) traceql.Static {
	if arr == nil || len(arr.Values) == 0 {
		return traceql.NewStaticNil()
	}
	// a nil element would panic the type assertions below (unreachable via proto unmarshal, guarded like staticFromKeyValue).
	for _, e := range arr.Values {
		if e == nil {
			return traceql.NewStaticNil()
		}
	}
	switch arr.Values[0].Value.(type) {
	case *commonv1.AnyValue_StringValue:
		out := make([]string, 0, len(arr.Values))
		for _, e := range arr.Values {
			ev, ok := e.Value.(*commonv1.AnyValue_StringValue)
			if !ok {
				return traceql.NewStaticNil()
			}
			out = append(out, ev.StringValue)
		}
		if len(out) == 1 {
			return traceql.NewStaticString(out[0])
		}
		return traceql.NewStaticStringArray(out)
	case *commonv1.AnyValue_IntValue:
		out := make([]int, 0, len(arr.Values))
		for _, e := range arr.Values {
			ev, ok := e.Value.(*commonv1.AnyValue_IntValue)
			if !ok {
				return traceql.NewStaticNil()
			}
			out = append(out, int(ev.IntValue))
		}
		if len(out) == 1 {
			return traceql.NewStaticInt(out[0])
		}
		return traceql.NewStaticIntArray(out)
	case *commonv1.AnyValue_DoubleValue:
		out := make([]float64, 0, len(arr.Values))
		for _, e := range arr.Values {
			ev, ok := e.Value.(*commonv1.AnyValue_DoubleValue)
			if !ok {
				return traceql.NewStaticNil()
			}
			out = append(out, ev.DoubleValue)
		}
		if len(out) == 1 {
			return traceql.NewStaticFloat(out[0])
		}
		return traceql.NewStaticFloatArray(out)
	case *commonv1.AnyValue_BoolValue:
		out := make([]bool, 0, len(arr.Values))
		for _, e := range arr.Values {
			ev, ok := e.Value.(*commonv1.AnyValue_BoolValue)
			if !ok {
				return traceql.NewStaticNil()
			}
			out = append(out, ev.BoolValue)
		}
		if len(out) == 1 {
			return traceql.NewStaticBool(out[0])
		}
		return traceql.NewStaticBooleanArray(out)
	default:
		return traceql.NewStaticNil()
	}
}
