package secrets

import (
	"strconv"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type TraceLocation struct {
	Resource int
	Scope    int
}

type TraceField struct {
	Kind     FieldKind
	Value    string
	TraceID  []byte
	SpanID   []byte
	Location TraceLocation
}

type TraceFieldVisitor func(TraceField) bool

func WalkPushSpansRequest(request *tempopb.PushSpansRequest, visit TraceFieldVisitor) {
	if request == nil {
		return
	}
	walkResourceSpans(request.Batches, visit)
}

func walkResourceSpans(resources []*tracev1.ResourceSpans, visit TraceFieldVisitor) {
	for resourceIndex, resourceSpans := range resources {
		if resourceSpans == nil {
			continue
		}
		base := TraceLocation{Resource: resourceIndex, Scope: -1}
		if resourceSpans.SchemaUrl != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: resourceSpans.SchemaUrl, Location: base}) {
			return
		}
		if resource := resourceSpans.Resource; resource != nil {
			if !walkAttributes(resource.Attributes, FieldKindResourceAttribute, nil, nil, base, visit) {
				return
			}
			for _, entity := range resource.EntityRefs {
				if entity == nil {
					continue
				}
				if entity.SchemaUrl != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: entity.SchemaUrl, Location: base}) {
					return
				}
				if entity.Type != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: entity.Type, Location: base}) {
					return
				}
			}
		}

		for scopeIndex, scopeSpans := range resourceSpans.ScopeSpans {
			if scopeSpans == nil {
				continue
			}
			location := base
			location.Scope = scopeIndex
			if scopeSpans.SchemaUrl != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: scopeSpans.SchemaUrl, Location: location}) {
				return
			}
			if scope := scopeSpans.Scope; scope != nil {
				if scope.Name != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: scope.Name, Location: location}) {
					return
				}
				if scope.Version != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: scope.Version, Location: location}) {
					return
				}
				if !walkAttributes(scope.Attributes, FieldKindScopeAttribute, nil, nil, location, visit) {
					return
				}
			}
			for _, span := range scopeSpans.Spans {
				if span == nil {
					continue
				}
				if !walkSpan(span, location, visit) {
					return
				}
			}
		}
	}
}

func walkSpan(span *tracev1.Span, location TraceLocation, visit TraceFieldVisitor) bool {
	if span.TraceState != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: span.TraceState, TraceID: span.TraceId, SpanID: span.SpanId, Location: location}) {
		return false
	}
	if span.Name != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: span.Name, TraceID: span.TraceId, SpanID: span.SpanId, Location: location}) {
		return false
	}
	if span.Status != nil && span.Status.Message != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: span.Status.Message, TraceID: span.TraceId, SpanID: span.SpanId, Location: location}) {
		return false
	}
	if !walkAttributes(span.Attributes, FieldKindSpanAttribute, span.TraceId, span.SpanId, location, visit) {
		return false
	}
	for _, event := range span.Events {
		if event == nil {
			continue
		}
		if event.Name != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: event.Name, TraceID: span.TraceId, SpanID: span.SpanId, Location: location}) {
			return false
		}
		if !walkAttributes(event.Attributes, FieldKindEventAttribute, span.TraceId, span.SpanId, location, visit) {
			return false
		}
	}
	for _, link := range span.Links {
		if link == nil {
			continue
		}
		if link.TraceState != "" && !visit(TraceField{Kind: FieldKindTraceField, Value: link.TraceState, TraceID: span.TraceId, SpanID: span.SpanId, Location: location}) {
			return false
		}
		if !walkAttributes(link.Attributes, FieldKindLinkAttribute, span.TraceId, span.SpanId, location, visit) {
			return false
		}
	}
	return true
}

func walkAttributes(attributes []*commonv1.KeyValue, kind FieldKind, traceID, spanID []byte, location TraceLocation, visit TraceFieldVisitor) bool {
	for _, attribute := range attributes {
		if attribute == nil {
			continue
		}
		if !walkAnyValue(attribute.Value, kind, traceID, spanID, location, visit) {
			return false
		}
	}
	return true
}

func walkAnyValue(value *commonv1.AnyValue, kind FieldKind, traceID, spanID []byte, location TraceLocation, visit TraceFieldVisitor) bool {
	if value == nil {
		return true
	}
	switch typed := value.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return visit(TraceField{Kind: kind, Value: typed.StringValue, TraceID: traceID, SpanID: spanID, Location: location})
	case *commonv1.AnyValue_BytesValue:
		return visit(TraceField{Kind: kind, Value: string(typed.BytesValue), TraceID: traceID, SpanID: spanID, Location: location})
	case *commonv1.AnyValue_IntValue:
		return visit(TraceField{Kind: kind, Value: strconv.FormatInt(typed.IntValue, 10), TraceID: traceID, SpanID: spanID, Location: location})
	case *commonv1.AnyValue_DoubleValue:
		return visit(TraceField{Kind: kind, Value: strconv.FormatFloat(typed.DoubleValue, 'g', -1, 64), TraceID: traceID, SpanID: spanID, Location: location})
	case *commonv1.AnyValue_BoolValue:
		return visit(TraceField{Kind: kind, Value: strconv.FormatBool(typed.BoolValue), TraceID: traceID, SpanID: spanID, Location: location})
	case *commonv1.AnyValue_ArrayValue:
		if typed.ArrayValue == nil {
			return true
		}
		for _, child := range typed.ArrayValue.Values {
			if !walkAnyValue(child, kind, traceID, spanID, location, visit) {
				return false
			}
		}
	case *commonv1.AnyValue_KvlistValue:
		if typed.KvlistValue == nil {
			return true
		}
		for _, child := range typed.KvlistValue.Values {
			if child == nil {
				continue
			}
			if !walkAnyValue(child.Value, kind, traceID, spanID, location, visit) {
				return false
			}
		}
	}
	return true
}
