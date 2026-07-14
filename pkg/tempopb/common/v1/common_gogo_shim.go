package v1

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
//
//nolint:revive // name is dictated by gogo's reflection lookup, see file doc comment above.
func (*AnyValue) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*AnyValue_StringValue)(nil),
		(*AnyValue_BoolValue)(nil),
		(*AnyValue_IntValue)(nil),
		(*AnyValue_DoubleValue)(nil),
		(*AnyValue_ArrayValue)(nil),
		(*AnyValue_KvlistValue)(nil),
		(*AnyValue_BytesValue)(nil),
		(*AnyValue_StringValueStrindex)(nil),
	}
}
