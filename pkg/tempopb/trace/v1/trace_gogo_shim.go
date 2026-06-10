package v1

import "github.com/gogo/protobuf/proto"

// Register enums with gogo/protobuf so jsonpb (and any other consumer that
// goes through proto.EnumValueMap) can look up the symbolic names that
// wiresmith only exposes via the local *_value/*_name maps.

func init() {
	proto.RegisterEnum("tempopb.trace.v1.SpanFlags", SpanFlags_name, SpanFlags_value)
	proto.RegisterEnum("tempopb.trace.v1.Span.SpanKind", Span_SpanKind_name, Span_SpanKind_value)
	proto.RegisterEnum("tempopb.trace.v1.Status.StatusCode", Status_StatusCode_name, Status_StatusCode_value)
}
