package parquet

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/parquet-go/jsonlite"
	"github.com/parquet-go/parquet-go/format"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func writeProtoTimestamp(col ColumnBuffer, levels columnLevels, ts *timestamppb.Timestamp, node Node) {
	if ts == nil {
		col.writeNull(levels)
		return
	}
	var typ = node.Type()
	var unit format.TimeUnit
	if lt := typ.LogicalType(); lt != nil && lt.Timestamp != nil {
		unit = lt.Timestamp.Unit
	} else {
		unit = Nanosecond.TimeUnit()
	}
	var t = ts.AsTime()
	var value int64
	switch {
	case unit.Millis != nil:
		value = t.UnixMilli()
	case unit.Micros != nil:
		value = t.UnixMicro()
	default:
		value = t.UnixNano()
	}
	switch kind := typ.Kind(); kind {
	case Int32, Int64:
		col.writeInt64(levels, value)
	case Float, Double:
		col.writeDouble(levels, t.Sub(time.Unix(0, 0)).Seconds())
	case ByteArray:
		col.writeByteArray(levels, t.AppendFormat(nil, time.RFC3339Nano))
	default:
		panic(fmt.Sprintf("unsupported physical type for timestamp: %v", kind))
	}
}

func writeProtoDuration(col ColumnBuffer, levels columnLevels, dur *durationpb.Duration, node Node) {
	if dur == nil {
		col.writeNull(levels)
		return
	}
	d := dur.AsDuration()
	switch kind := node.Type().Kind(); kind {
	case Int32, Int64:
		col.writeInt64(levels, d.Nanoseconds())
	case Float, Double:
		col.writeDouble(levels, d.Seconds())
	case ByteArray:
		col.writeByteArray(levels, unsafeByteArrayFromString(d.String()))
	default:
		panic(fmt.Sprintf("unsupported physical type for duration: %v", kind))
	}
}

func writeProtoStruct(col ColumnBuffer, levels columnLevels, s *structpb.Struct, node Node) {
	buf := buffers.get(2 * proto.Size(s))
	buf.data = appendProtoStructJSON(buf.data[:0], s)
	col.writeByteArray(levels, buf.data)
	buf.unref()
}

func writeProtoList(col ColumnBuffer, levels columnLevels, l *structpb.ListValue, node Node) {
	buf := buffers.get(2 * proto.Size(l))
	buf.data = appendProtoListValueJSON(buf.data[:0], l)
	col.writeByteArray(levels, buf.data)
	buf.unref()
}

func appendProtoStructJSON(b []byte, s *structpb.Struct) []byte {
	if s == nil {
		return append(b, "null"...)
	}

	fields := s.GetFields()
	if len(fields) == 0 {
		return append(b, "{}"...)
	}

	keys := make([]string, 0, 20)
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	b = append(b, '{')
	for i, key := range keys {
		if i > 0 {
			b = append(b, ',')
		}
		b = jsonlite.AppendQuote(b, key)
		b = append(b, ':')
		b = appendProtoValueJSON(b, fields[key])
	}
	b = append(b, '}')
	return b
}

func appendProtoValueJSON(b []byte, v *structpb.Value) []byte {
	switch k := v.GetKind().(type) {
	case *structpb.Value_NumberValue:
		return strconv.AppendFloat(b, k.NumberValue, 'g', -1, 64)
	case *structpb.Value_StringValue:
		return jsonlite.AppendQuote(b, k.StringValue)
	case *structpb.Value_BoolValue:
		return strconv.AppendBool(b, k.BoolValue)
	case *structpb.Value_StructValue:
		return appendProtoStructJSON(b, k.StructValue)
	case *structpb.Value_ListValue:
		return appendProtoListValueJSON(b, k.ListValue)
	default:
		return append(b, "null"...)
	}
}

func appendProtoListValueJSON(b []byte, l *structpb.ListValue) []byte {
	if l == nil {
		return append(b, "null"...)
	}
	values := l.GetValues()
	b = append(b, '[')
	for i, v := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = appendProtoValueJSON(b, v)
	}
	b = append(b, ']')
	return b
}

func writeProtoAny(col ColumnBuffer, levels columnLevels, a *anypb.Any, node Node) {
	if a == nil {
		col.writeNull(levels)
		return
	}
	data, err := proto.Marshal(a)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal anypb.Any: %v", err))
	}
	col.writeByteArray(levels, data)
}

// makeNestedMap creates a nested map structure from a dot-separated path.
// For example, "testproto.ProtoPayload" with value v creates:
// map["testproto"] = map["ProtoPayload"] = v
func makeNestedMap(path string, value any) any {
	components := make([]string, 0, 8)
	for component := range strings.SplitSeq(path, ".") {
		components = append(components, component)
	}

	result := value
	for i := len(components) - 1; i >= 0; i-- {
		result = map[string]any{
			components[i]: result,
		}
	}
	return result
}

// navigateToNestedGroup walks through a nested group structure following the given path.
// The path is expected to be a dot-separated string (e.g., "testproto.ProtoPayload").
// Returns the node at the end of the path, or panics if the path doesn't match the schema.
func navigateToNestedGroup(node Node, path string) Node {
	for component := range strings.SplitSeq(path, ".") {
		var found bool
		for _, field := range node.Fields() {
			if field.Name() == component {
				node, found = field, true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("field %q not found in schema while navigating path %q", component, path))
		}
	}
	return node
}
