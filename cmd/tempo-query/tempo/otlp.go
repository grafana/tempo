package tempo

import (
	otlpcommonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	otlpresourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	otlptracev1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/grafana/tempo/pkg/tempopb"
	tempocommonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	temporesourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tempotracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func getOtlpTraceData(td *tempopb.TraceByIDResponse) *otlptracev1.TracesData {
	if td == nil || td.Trace == nil {
		return nil
	}
	tempoResourceSpans := td.Trace.ResourceSpans
	otlpResourceSpans := make([]*otlptracev1.ResourceSpans, len(tempoResourceSpans))
	for i, tempoResourceSpan := range tempoResourceSpans {
		otlpResourceSpans[i] = &otlptracev1.ResourceSpans{
			Resource:   toOtlpResource(tempoResourceSpan.Resource),
			ScopeSpans: toOtlpScopeSpans(tempoResourceSpan.ScopeSpans),
		}
	}
	return &otlptracev1.TracesData{
		ResourceSpans: otlpResourceSpans,
	}
}

func toOtlpResource(resource *temporesourcev1.Resource) *otlpresourcev1.Resource {
	return &otlpresourcev1.Resource{
		Attributes:             toOtlpAttributes(resource.Attributes),
		DroppedAttributesCount: resource.DroppedAttributesCount,
	}
}

func toOtlpAttributes(attributes []*tempocommonv1.KeyValue) []*otlpcommonv1.KeyValue {
	otlpAttributes := make([]*otlpcommonv1.KeyValue, len(attributes))
	for i, attribute := range attributes {
		otlpAttributes[i] = &otlpcommonv1.KeyValue{
			Key:   attribute.Key,
			Value: toOtlpAnyValue(attribute.Value),
		}
	}
	return otlpAttributes
}

func toOtlpAnyValue(value *tempocommonv1.AnyValue) *otlpcommonv1.AnyValue {
	if value == nil {
		return nil
	}
	switch v := value.Value.(type) {
	case *tempocommonv1.AnyValue_StringValue:
		return &otlpcommonv1.AnyValue{Value: &otlpcommonv1.AnyValue_StringValue{StringValue: v.StringValue}}
	case *tempocommonv1.AnyValue_BoolValue:
		return &otlpcommonv1.AnyValue{Value: &otlpcommonv1.AnyValue_BoolValue{BoolValue: v.BoolValue}}
	case *tempocommonv1.AnyValue_IntValue:
		return &otlpcommonv1.AnyValue{Value: &otlpcommonv1.AnyValue_IntValue{IntValue: v.IntValue}}
	case *tempocommonv1.AnyValue_DoubleValue:
		return &otlpcommonv1.AnyValue{Value: &otlpcommonv1.AnyValue_DoubleValue{DoubleValue: v.DoubleValue}}
	case *tempocommonv1.AnyValue_ArrayValue:
		otlpArrayValues := make([]*otlpcommonv1.AnyValue, len(v.ArrayValue.Values))
		for i, element := range v.ArrayValue.Values {
			otlpArrayValues[i] = toOtlpAnyValue(element)
		}
		return &otlpcommonv1.AnyValue{Value: &otlpcommonv1.AnyValue_ArrayValue{ArrayValue: &otlpcommonv1.ArrayValue{Values: otlpArrayValues}}}
	case *tempocommonv1.AnyValue_KvlistValue:
		otlpKvlistValues := make([]*otlpcommonv1.KeyValue, len(v.KvlistValue.Values))
		for i, element := range v.KvlistValue.Values {
			otlpKvlistValues[i] = &otlpcommonv1.KeyValue{
				Key:   element.Key,
				Value: toOtlpAnyValue(element.Value),
			}
		}
		return &otlpcommonv1.AnyValue{Value: &otlpcommonv1.AnyValue_KvlistValue{KvlistValue: &otlpcommonv1.KeyValueList{Values: otlpKvlistValues}}}
	case *tempocommonv1.AnyValue_BytesValue:
		return &otlpcommonv1.AnyValue{Value: &otlpcommonv1.AnyValue_BytesValue{BytesValue: v.BytesValue}}
	default:
		return nil
	}
}

func toOtlpScopeSpans(tempoScopeSpans []*tempotracev1.ScopeSpans) []*otlptracev1.ScopeSpans {
	otlpScopeSpans := make([]*otlptracev1.ScopeSpans, len(tempoScopeSpans))
	for i, tempoScopeSpan := range tempoScopeSpans {
		otlpScopeSpans[i] = &otlptracev1.ScopeSpans{
			Scope:     toOtlpInstrumentationScope(tempoScopeSpan.Scope),
			Spans:     toOtlpSpans(tempoScopeSpan.Spans),
			SchemaUrl: tempoScopeSpan.SchemaUrl,
		}
	}
	return otlpScopeSpans
}

func toOtlpInstrumentationScope(scope *tempocommonv1.InstrumentationScope) *otlpcommonv1.InstrumentationScope {
	if scope == nil {
		return nil
	}
	return &otlpcommonv1.InstrumentationScope{
		Name:                   scope.Name,
		Version:                scope.Version,
		Attributes:             toOtlpAttributes(scope.Attributes),
		DroppedAttributesCount: scope.DroppedAttributesCount,
	}
}

func toOtlpSpans(tempoSpans []*tempotracev1.Span) []*otlptracev1.Span {
	otlpSpans := make([]*otlptracev1.Span, len(tempoSpans))
	for i, tempoSpan := range tempoSpans {
		otlpSpans[i] = &otlptracev1.Span{
			TraceId:                tempoSpan.TraceId,
			SpanId:                 tempoSpan.SpanId,
			TraceState:             tempoSpan.TraceState,
			ParentSpanId:           tempoSpan.ParentSpanId,
			Name:                   tempoSpan.Name,
			Kind:                   otlptracev1.Span_SpanKind(tempoSpan.Kind),
			StartTimeUnixNano:      tempoSpan.StartTimeUnixNano,
			EndTimeUnixNano:        tempoSpan.EndTimeUnixNano,
			Attributes:             toOtlpAttributes(tempoSpan.Attributes),
			DroppedAttributesCount: tempoSpan.DroppedAttributesCount,
			Events:                 toOtlpEvents(tempoSpan.Events),
			DroppedEventsCount:     tempoSpan.DroppedEventsCount,
			Links:                  toOtlpLinks(tempoSpan.Links),
			DroppedLinksCount:      tempoSpan.DroppedLinksCount,
			Status:                 toOtlpStatus(tempoSpan.Status),
		}
	}
	return otlpSpans
}

func toOtlpEvents(tempoEvents []*tempotracev1.Span_Event) []*otlptracev1.Span_Event {
	otlpEvents := make([]*otlptracev1.Span_Event, len(tempoEvents))
	for i, tempoEvent := range tempoEvents {
		otlpEvents[i] = &otlptracev1.Span_Event{
			TimeUnixNano:           tempoEvent.TimeUnixNano,
			Name:                   tempoEvent.Name,
			Attributes:             toOtlpAttributes(tempoEvent.Attributes),
			DroppedAttributesCount: tempoEvent.DroppedAttributesCount,
		}
	}
	return otlpEvents
}

func toOtlpLinks(tempoLinks []*tempotracev1.Span_Link) []*otlptracev1.Span_Link {
	otlpLinks := make([]*otlptracev1.Span_Link, len(tempoLinks))
	for i, tempoLink := range tempoLinks {
		otlpLinks[i] = &otlptracev1.Span_Link{
			TraceId:                tempoLink.TraceId,
			SpanId:                 tempoLink.SpanId,
			TraceState:             tempoLink.TraceState,
			Attributes:             toOtlpAttributes(tempoLink.Attributes),
			DroppedAttributesCount: tempoLink.DroppedAttributesCount,
		}
	}
	return otlpLinks
}

func toOtlpStatus(tempoStatus *tempotracev1.Status) *otlptracev1.Status {
	if tempoStatus == nil {
		return nil
	}
	return &otlptracev1.Status{
		Message: tempoStatus.Message,
		Code:    otlptracev1.Status_StatusCode(tempoStatus.Code),
	}
}
