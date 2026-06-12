package tracefilter

import (
	"maps"
	"time"

	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
)

// protoSpan adapts an OTLP proto span to traceql.Span so the engine can match a filter against a
// proto trace. Bridging reuses the real SpansetFilter instead of reimplementing TraceQL semantics.
type protoSpan struct {
	span               *tracev1.Span
	startTimeUnixNanos uint64
	durationNanos      uint64
	attributes         map[traceql.Attribute]traceql.Static
}

// newProtoSpan builds a protoSpan. resourceAttrs and traceAttrs are shared across the batch/trace;
// scope carries the span's instrumentation scope (may be nil).
func newProtoSpan(span *tracev1.Span, resourceAttrs, traceAttrs map[traceql.Attribute]traceql.Static, scope *commonv1.InstrumentationScope) *protoSpan {
	duration := uint64(0)
	if span.EndTimeUnixNano > span.StartTimeUnixNano {
		duration = span.EndTimeUnixNano - span.StartTimeUnixNano
	}

	attrs := make(map[traceql.Attribute]traceql.Static, len(span.Attributes)+len(resourceAttrs)+len(traceAttrs)+10)

	// resource attributes first so a same-named span attribute wins, matching TraceQL precedence.
	maps.Copy(attrs, resourceAttrs)
	maps.Copy(attrs, traceAttrs)
	for _, kv := range span.Attributes {
		attrs[traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, kv.Key)] = staticFromKeyValue(kv)
	}

	// HACK: this hand-maps each proto field to a traceql intrinsic, duplicating value extraction the
	// engine already does for parquet spans. TODO: refactor to reuse the engine's intrinsic resolution.
	attrs[traceql.IntrinsicNameAttribute] = traceql.NewStaticString(span.Name)
	attrs[traceql.IntrinsicDurationAttribute] = traceql.NewStaticDuration(time.Duration(duration))
	attrs[traceql.IntrinsicKindAttribute] = traceql.NewStaticKind(spanKindToTraceql(span.Kind))
	// hex spellings match how the storage layer surfaces these ids to TraceQL.
	attrs[traceql.IntrinsicSpanIDAttribute] = traceql.NewStaticString(util.SpanIDToHexString(span.SpanId))
	attrs[traceql.IntrinsicParentIDAttribute] = traceql.NewStaticString(util.SpanIDToHexString(span.ParentSpanId))
	attrs[traceql.IntrinsicTraceIDAttribute] = traceql.NewStaticString(util.TraceIDToHexString(span.TraceId))
	if span.Status != nil {
		attrs[traceql.IntrinsicStatusAttribute] = traceql.NewStaticStatus(spanStatusToTraceql(span.Status.Code))
		attrs[traceql.IntrinsicStatusMessageAttribute] = traceql.NewStaticString(span.Status.Message)
	} else {
		// nil status is unset; populate both so statusMessage resolves consistently.
		attrs[traceql.IntrinsicStatusAttribute] = traceql.NewStaticStatus(traceql.StatusUnset)
		attrs[traceql.IntrinsicStatusMessageAttribute] = traceql.NewStaticString("")
	}

	if scope != nil {
		attrs[traceql.IntrinsicInstrumentationNameAttribute] = traceql.NewStaticString(scope.Name)
		attrs[traceql.IntrinsicInstrumentationVersionAttribute] = traceql.NewStaticString(scope.Version)
	}

	// the map holds one value per intrinsic, so only the first event/link is queryable; the storage
	// layer's AttributeFor likewise returns the first match.
	if len(span.Events) > 0 {
		e := span.Events[0]
		attrs[traceql.IntrinsicEventNameAttribute] = traceql.NewStaticString(e.Name)
		attrs[traceql.IntrinsicEventTimeSinceStartAttribute] = traceql.NewStaticDuration(time.Duration(e.TimeUnixNano - span.StartTimeUnixNano))
	}
	if len(span.Links) > 0 {
		l := span.Links[0]
		attrs[traceql.IntrinsicLinkTraceIDAttribute] = traceql.NewStaticString(util.TraceIDToHexString(l.TraceId))
		attrs[traceql.IntrinsicLinkSpanIDAttribute] = traceql.NewStaticString(util.SpanIDToHexString(l.SpanId))
	}

	return &protoSpan{
		span:               span,
		startTimeUnixNanos: span.StartTimeUnixNano,
		durationNanos:      duration,
		attributes:         attrs,
	}
}

// resourceAttributes converts a resource's attributes into the scoped traceql map once per batch.
func resourceAttributes(resource *resourcev1.Resource) map[traceql.Attribute]traceql.Static {
	if resource == nil {
		return nil
	}
	out := make(map[traceql.Attribute]traceql.Static, len(resource.Attributes))
	for _, kv := range resource.Attributes {
		out[traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, kv.Key)] = staticFromKeyValue(kv)
	}
	return out
}

// staticFromKeyValue guards a nil value: StaticFromAnyValue would panic on a nil AnyValue.
func staticFromKeyValue(kv *commonv1.KeyValue) traceql.Static {
	if kv.Value == nil {
		return traceql.NewStaticNil()
	}
	return traceql.StaticFromAnyValue(kv.Value)
}

func (s *protoSpan) AttributeFor(a traceql.Attribute) (traceql.Static, bool) {
	if v, ok := s.attributes[a]; ok {
		return v, true
	}
	// unscoped lookups fall back to span then resource scope, matching the engine.
	if a.Scope == traceql.AttributeScopeNone && a.Intrinsic == traceql.IntrinsicNone {
		aSpan := a
		aSpan.Scope = traceql.AttributeScopeSpan
		if v, ok := s.attributes[aSpan]; ok {
			return v, true
		}
		aRes := a
		aRes.Scope = traceql.AttributeScopeResource
		if v, ok := s.attributes[aRes]; ok {
			return v, true
		}
	}
	return traceql.NewStaticNil(), false
}

func (s *protoSpan) AllAttributes() map[traceql.Attribute]traceql.Static {
	return s.attributes
}

func (s *protoSpan) AllAttributesFunc(cb func(traceql.Attribute, traceql.Static)) {
	for k, v := range s.attributes {
		cb(k, v)
	}
}

func (s *protoSpan) ID() []byte                 { return s.span.SpanId }
func (s *protoSpan) StartTimeUnixNanos() uint64 { return s.startTimeUnixNanos }
func (s *protoSpan) DurationNanos() uint64      { return s.durationNanos }

// Structural methods are unreachable: CompileSpansetFilter rejects structural operators.
func (s *protoSpan) DescendantOf(_, _ []traceql.Span, _, _, _ bool, _ []traceql.Span) []traceql.Span {
	return nil
}

func (s *protoSpan) SiblingOf(_, _ []traceql.Span, _, _ bool, _ []traceql.Span) []traceql.Span {
	return nil
}

func (s *protoSpan) ChildOf(_, _ []traceql.Span, _, _, _ bool, _ []traceql.Span) []traceql.Span {
	return nil
}

// spanKindToTraceql maps OTLP span kinds to traceql kinds; the enums are not numerically aligned.
func spanKindToTraceql(k tracev1.Span_SpanKind) traceql.Kind {
	switch k {
	case tracev1.Span_SPAN_KIND_INTERNAL:
		return traceql.KindInternal
	case tracev1.Span_SPAN_KIND_CLIENT:
		return traceql.KindClient
	case tracev1.Span_SPAN_KIND_SERVER:
		return traceql.KindServer
	case tracev1.Span_SPAN_KIND_PRODUCER:
		return traceql.KindProducer
	case tracev1.Span_SPAN_KIND_CONSUMER:
		return traceql.KindConsumer
	default:
		return traceql.KindUnspecified
	}
}

// spanStatusToTraceql maps OTLP status codes to traceql statuses; the enum orderings differ.
func spanStatusToTraceql(c tracev1.Status_StatusCode) traceql.Status {
	switch c {
	case tracev1.Status_STATUS_CODE_OK:
		return traceql.StatusOk
	case tracev1.Status_STATUS_CODE_ERROR:
		return traceql.StatusError
	default:
		return traceql.StatusUnset
	}
}

// compile-time assertion that protoSpan satisfies the engine's Span interface.
var _ traceql.Span = (*protoSpan)(nil)
