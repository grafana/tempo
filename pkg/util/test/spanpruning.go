package test

import (
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// MakeSpanPruningSpanID builds a deterministic 8-byte span ID from two bytes, handy for
// constructing readable trace trees in span pruning tests.
func MakeSpanPruningSpanID(a, b byte) []byte {
	return []byte{a, b, 0, 0, 0, 0, 0, 0}
}

// MakeSpanPruningSpan builds a span with an explicit parent, name and time range. Unlike
// MakeSpan, it doesn't randomize anything, so tests can construct precise trace shapes
// (e.g. N identical leaf spans under one parent) to exercise span pruning/aggregation.
func MakeSpanPruningSpan(traceID, spanID, parentID []byte, name string, startNs, endNs uint64, attrs ...*v1_common.KeyValue) *v1_trace.Span {
	return &v1_trace.Span{
		TraceId:           traceID,
		SpanId:            spanID,
		ParentSpanId:      parentID,
		Name:              name,
		StartTimeUnixNano: startNs,
		EndTimeUnixNano:   endNs,
		Attributes:        attrs,
	}
}

// MakeSpanPruningSpanWithStatus is MakeSpanPruningSpan with a status code attached, for tests
// that group aggregation by span status (e.g. OK vs ERROR).
func MakeSpanPruningSpanWithStatus(traceID, spanID, parentID []byte, name string, endNs uint64, code v1_trace.Status_StatusCode) *v1_trace.Span {
	s := MakeSpanPruningSpan(traceID, spanID, parentID, name, 0, endNs)
	s.Status = &v1_trace.Status{Code: code}
	return s
}

// WrapSpansAsTrace nests the given spans into a single-batch tempopb.Trace.
func WrapSpansAsTrace(spans ...*v1_trace.Span) *tempopb.Trace {
	return &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{
			{ScopeSpans: []*v1_trace.ScopeSpans{{Spans: spans}}},
		},
	}
}

// AllSpansInTrace flattens every span across all resource/scope spans in a trace.
func AllSpansInTrace(tr *tempopb.Trace) []*v1_trace.Span {
	var out []*v1_trace.Span
	for _, rs := range tr.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			out = append(out, ss.Spans...)
		}
	}
	return out
}

// CountSpans returns the total number of spans in a trace.
func CountSpans(tr *tempopb.Trace) int {
	return len(AllSpansInTrace(tr))
}

// SpansByName returns every span in a trace with the given name.
func SpansByName(tr *tempopb.Trace, name string) []*v1_trace.Span {
	var out []*v1_trace.Span
	for _, s := range AllSpansInTrace(tr) {
		if s.Name == name {
			out = append(out, s)
		}
	}
	return out
}

// SpanExistsWithName reports whether any span in the trace has the given name.
func SpanExistsWithName(tr *tempopb.Trace, name string) bool {
	for _, s := range AllSpansInTrace(tr) {
		if s.Name == name {
			return true
		}
	}
	return false
}

// SpanAttr returns the value of the named attribute on a span, or nil if absent.
func SpanAttr(s *v1_trace.Span, key string) *v1_common.AnyValue {
	for _, kv := range s.Attributes {
		if kv.Key == key {
			return kv.Value
		}
	}
	return nil
}

// SpanAttrInt returns the int64 value of the named attribute, or -1 if absent.
func SpanAttrInt(s *v1_trace.Span, key string) int64 {
	v := SpanAttr(s, key)
	if v == nil {
		return -1
	}
	return v.GetIntValue()
}

// SpanAttrString returns the string value of the named attribute, or "" if absent.
func SpanAttrString(s *v1_trace.Span, key string) string {
	v := SpanAttr(s, key)
	if v == nil {
		return ""
	}
	return v.GetStringValue()
}

// SpanAttrDoubleSlice returns the float64 values of an ArrayValue attribute.
func SpanAttrDoubleSlice(s *v1_trace.Span, key string) []float64 {
	v := SpanAttr(s, key)
	if v == nil {
		return nil
	}
	arr := v.GetArrayValue()
	if arr == nil {
		return nil
	}
	out := make([]float64, len(arr.Values))
	for i, elem := range arr.Values {
		out[i] = elem.GetDoubleValue()
	}
	return out
}

// SpanAttrIntSlice returns the int64 values of an ArrayValue attribute.
func SpanAttrIntSlice(s *v1_trace.Span, key string) []int64 {
	v := SpanAttr(s, key)
	if v == nil {
		return nil
	}
	arr := v.GetArrayValue()
	if arr == nil {
		return nil
	}
	out := make([]int64, len(arr.Values))
	for i, elem := range arr.Values {
		out[i] = elem.GetIntValue()
	}
	return out
}

// IsSpanPruningSummary reports whether a span is an aggregated summary produced by the span
// pruning processor, i.e. it carries aggregation.is_summary=true.
func IsSpanPruningSummary(s *v1_trace.Span) bool {
	v := SpanAttr(s, "aggregation.is_summary")
	return v != nil && v.GetBoolValue()
}

// SpanPruningSummaries returns every summary span in a trace.
func SpanPruningSummaries(tr *tempopb.Trace) []*v1_trace.Span {
	var out []*v1_trace.Span
	for _, s := range AllSpansInTrace(tr) {
		if IsSpanPruningSummary(s) {
			out = append(out, s)
		}
	}
	return out
}

// FindSpanPruningSummary returns the first summary span found in a trace.
func FindSpanPruningSummary(tr *tempopb.Trace) (*v1_trace.Span, bool) {
	for _, s := range AllSpansInTrace(tr) {
		if IsSpanPruningSummary(s) {
			return s, true
		}
	}
	return nil, false
}

// FindSpanPruningSummaryByName returns the first summary span with the given name.
func FindSpanPruningSummaryByName(tr *tempopb.Trace, name string) (*v1_trace.Span, bool) {
	for _, s := range AllSpansInTrace(tr) {
		if IsSpanPruningSummary(s) && s.Name == name {
			return s, true
		}
	}
	return nil, false
}

// FindSpanPruningSummaryByNameAndStatus returns the first summary span matching both name and status code.
func FindSpanPruningSummaryByNameAndStatus(tr *tempopb.Trace, name string, code v1_trace.Status_StatusCode) (*v1_trace.Span, bool) {
	for _, s := range AllSpansInTrace(tr) {
		if IsSpanPruningSummary(s) && s.Name == name && s.Status != nil && s.Status.Code == code {
			return s, true
		}
	}
	return nil, false
}
