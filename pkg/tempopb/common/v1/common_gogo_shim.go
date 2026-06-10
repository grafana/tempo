package v1

import (
	"fmt"
	"strconv"
)

// Hand-written compatibility shims for gogo/protobuf's jsonpb and proto
// reflection paths. The wiresmith generator emits the wire-format
// Marshal/Unmarshal/Size methods and Google's protoreflect.Message, but
// gogo's jsonpb walks types via the older proto.Properties machinery which
// looks up oneof variants through XXX_OneofWrappers. Without this method
// jsonpb fails to dispatch JSON field names like "stringValue" to the
// correct AnyValue_StringValue wrapper. The on-wire representation is
// untouched.

// XXX_OneofWrappers exposes the AnyValue oneof variants so gogo's
// proto.GetProperties can build the OneofTypes map.
func (*AnyValue) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*AnyValue_StringValue)(nil),
		(*AnyValue_BoolValue)(nil),
		(*AnyValue_IntValue)(nil),
		(*AnyValue_DoubleValue)(nil),
		(*AnyValue_ArrayValue)(nil),
		(*AnyValue_KvlistValue)(nil),
		(*AnyValue_BytesValue)(nil),
	}
}

// StableString returns a deterministic representation of an AnyValue.
// wiresmith's generated String() emits fmt.Sprintf("%v", *m) which, for
// oneof fields, prints the interface's underlying pointer address (and
// can't be overridden — the generator emits String() itself). Tempo relies
// on a stable AnyValue identity for sort keys and span-set IDs, so callers
// that need determinism use this helper instead of String().
func (m *AnyValue) StableString() string {
	if m == nil {
		return "<nil>"
	}
	switch x := m.Value.(type) {
	case *AnyValue_StringValue:
		return x.StringValue
	case *AnyValue_BoolValue:
		return strconv.FormatBool(x.BoolValue)
	case *AnyValue_IntValue:
		return strconv.FormatInt(x.IntValue, 10)
	case *AnyValue_DoubleValue:
		return strconv.FormatFloat(x.DoubleValue, 'g', -1, 64)
	case *AnyValue_BytesValue:
		return string(x.BytesValue)
	case *AnyValue_ArrayValue:
		return fmt.Sprintf("%v", x.ArrayValue.Values)
	case *AnyValue_KvlistValue:
		return fmt.Sprintf("%v", x.KvlistValue.Values)
	}
	return ""
}
