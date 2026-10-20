package distributor

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/entity"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

// directTraces copies mutable data because pdata may be reused after PushTraces returns.
func directTraces(td ptrace.Traces) *tempopb.Trace {
	out := &tempopb.Trace{}
	rss := td.ResourceSpans()
	if rss.Len() > 0 {
		out.ResourceSpans = make([]*trace.ResourceSpans, rss.Len())
	}
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		r := rs.Resource()
		batch := &trace.ResourceSpans{
			SchemaUrl: rs.SchemaUrl(),
			Resource: &resource.Resource{
				Attributes:             directAttributes(r.Attributes()),
				DroppedAttributesCount: r.DroppedAttributesCount(),
				EntityRefs:             directEntityRefs(r),
			},
		}
		sss := rs.ScopeSpans()
		if sss.Len() > 0 {
			batch.ScopeSpans = make([]*trace.ScopeSpans, sss.Len())
		}
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			sc := ss.Scope()
			scope := &trace.ScopeSpans{
				SchemaUrl: ss.SchemaUrl(),
				Scope: &common.InstrumentationScope{
					Name:                   sc.Name(),
					Version:                sc.Version(),
					Attributes:             directAttributes(sc.Attributes()),
					DroppedAttributesCount: sc.DroppedAttributesCount(),
				},
			}
			spans := ss.Spans()
			if spans.Len() > 0 {
				scope.Spans = make([]*trace.Span, spans.Len())
			}
			for k := 0; k < spans.Len(); k++ {
				scope.Spans[k] = directSpan(spans.At(k))
			}
			batch.ScopeSpans[j] = scope
		}
		out.ResourceSpans[i] = batch
	}
	return out
}

func directSpan(s ptrace.Span) *trace.Span {
	out := &trace.Span{
		TraceId:                directTraceID(s.TraceID()),
		SpanId:                 directSpanID(s.SpanID()),
		ParentSpanId:           directSpanID(s.ParentSpanID()),
		TraceState:             s.TraceState().AsRaw(),
		Flags:                  s.Flags(),
		Name:                   s.Name(),
		Kind:                   trace.Span_SpanKind(s.Kind()),
		StartTimeUnixNano:      uint64(s.StartTimestamp()),
		EndTimeUnixNano:        uint64(s.EndTimestamp()),
		Attributes:             directAttributes(s.Attributes()),
		DroppedAttributesCount: s.DroppedAttributesCount(),
		DroppedEventsCount:     s.DroppedEventsCount(),
		DroppedLinksCount:      s.DroppedLinksCount(),
		Status:                 &trace.Status{Code: trace.Status_StatusCode(s.Status().Code()), Message: s.Status().Message()},
	}
	events := s.Events()
	if events.Len() > 0 {
		out.Events = make([]*trace.Span_Event, events.Len())
	}
	for i := 0; i < events.Len(); i++ {
		ev := events.At(i)
		out.Events[i] = &trace.Span_Event{
			TimeUnixNano:           uint64(ev.Timestamp()),
			Name:                   ev.Name(),
			Attributes:             directAttributes(ev.Attributes()),
			DroppedAttributesCount: ev.DroppedAttributesCount(),
		}
	}
	links := s.Links()
	if links.Len() > 0 {
		out.Links = make([]*trace.Span_Link, links.Len())
	}
	for i := 0; i < links.Len(); i++ {
		lk := links.At(i)
		out.Links[i] = &trace.Span_Link{
			TraceId:                directTraceID(lk.TraceID()),
			SpanId:                 directSpanID(lk.SpanID()),
			TraceState:             lk.TraceState().AsRaw(),
			Flags:                  lk.Flags(),
			Attributes:             directAttributes(lk.Attributes()),
			DroppedAttributesCount: lk.DroppedAttributesCount(),
		}
	}
	return out
}

func directTraceID(id pcommon.TraceID) []byte {
	if id.IsEmpty() {
		return []byte{}
	}
	return id[:]
}

func directSpanID(id pcommon.SpanID) []byte {
	if id.IsEmpty() {
		return []byte{}
	}
	return id[:]
}

func directAttributes(m pcommon.Map) []*common.KeyValue {
	if m.Len() == 0 {
		return nil
	}
	out := make([]*common.KeyValue, 0, m.Len())
	m.Range(func(k string, v pcommon.Value) bool {
		out = append(out, &common.KeyValue{Key: k, Value: directValue(v)})
		return true
	})
	return out
}

func directValue(v pcommon.Value) *common.AnyValue {
	out := &common.AnyValue{}
	switch v.Type() {
	case pcommon.ValueTypeStr:
		out.Value = &common.AnyValue_StringValue{StringValue: v.Str()}
	case pcommon.ValueTypeInt:
		out.Value = &common.AnyValue_IntValue{IntValue: v.Int()}
	case pcommon.ValueTypeDouble:
		out.Value = &common.AnyValue_DoubleValue{DoubleValue: v.Double()}
	case pcommon.ValueTypeBool:
		out.Value = &common.AnyValue_BoolValue{BoolValue: v.Bool()}
	case pcommon.ValueTypeBytes:
		data := v.Bytes().AsRaw()
		if data == nil {
			// A nil payload would lose the bytes oneof when Tempo marshals it.
			data = []byte{}
		}
		out.Value = &common.AnyValue_BytesValue{BytesValue: data}
	case pcommon.ValueTypeMap:
		out.Value = &common.AnyValue_KvlistValue{KvlistValue: &common.KeyValueList{Values: directAttributes(v.Map())}}
	case pcommon.ValueTypeSlice:
		a := v.Slice()
		arr := &common.ArrayValue{}
		if a.Len() > 0 {
			arr.Values = make([]*common.AnyValue, a.Len())
		}
		for i := 0; i < a.Len(); i++ {
			arr.Values[i] = directValue(a.At(i))
		}
		out.Value = &common.AnyValue_ArrayValue{ArrayValue: arr}
	}
	return out
}

func directEntityRefs(r pcommon.Resource) []*common.EntityRef {
	refs := entity.ResourceEntityRefs(r)
	if refs.Len() == 0 {
		return nil
	}
	out := make([]*common.EntityRef, refs.Len())
	for i := 0; i < refs.Len(); i++ {
		ref := refs.At(i)
		out[i] = &common.EntityRef{
			SchemaUrl:       ref.SchemaUrl(),
			Type:            ref.Type(),
			IdKeys:          ref.IdKeys().AsRaw(),
			DescriptionKeys: ref.DescriptionKeys().AsRaw(),
		}
	}
	return out
}
