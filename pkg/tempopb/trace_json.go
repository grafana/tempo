package tempopb

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"unicode/utf8"

	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

// The functions in this file encode trace-by-ID responses to JSON without
// jsonpb's per-message reflection. The output matches the default
// jsonpb.Marshaler byte for byte

var errMarshalNil = errors.New("Marshal called with nil")

// MarshalTraceByIDResponseToJSON encodes a TraceByIDResponse to JSON.
func MarshalTraceByIDResponseToJSON(r *TraceByIDResponse) ([]byte, error) {
	if r == nil {
		return nil, errMarshalNil
	}
	e := newTraceJSONEncoder(r.Size())
	e.response(r)
	return e.result()
}

// traceJSONEncoder appends JSON to buf. When err is set, result discards buf,
// so encoding continues to the end without checking err.
type traceJSONEncoder struct {
	buf []byte
	err error
}

// newTraceJSONEncoder sizes the buffer from the protobuf size of the message.
// Trace JSON measured 1.8-2.8x its protobuf size, so 3x avoids growing the
// buffer, which otherwise allocates about 5x the output size in total.
func newTraceJSONEncoder(protoSize int) traceJSONEncoder {
	return traceJSONEncoder{buf: make([]byte, 0, 3*protoSize)}
}

func (e *traceJSONEncoder) response(v *TraceByIDResponse) {
	e.buf = append(e.buf, '{')
	if v.Trace != nil {
		e.field("trace")
		e.trace(v.Trace, "resourceSpans")
	}
	if v.Metrics != nil {
		e.field("metrics")
		e.traceMetrics(v.Metrics)
	}
	e.enumField("status", int32(v.Status), PartialStatus_name)
	e.stringField("message", v.Message)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) result() ([]byte, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.buf, nil
}

// message writes null for an absent message, or opens its object.
func (e *traceJSONEncoder) message(present bool) bool {
	if !present {
		e.buf = append(e.buf, "null"...)
		return false
	}
	e.buf = append(e.buf, '{')
	return true
}

// field writes a schema field name. It is only called inside an open object,
// so a preceding '{' means this is the first field.
func (e *traceJSONEncoder) field(name string) {
	if e.buf[len(e.buf)-1] != '{' {
		e.buf = append(e.buf, ',')
	}
	e.buf = append(e.buf, '"')
	e.buf = append(e.buf, name...)
	e.buf = append(e.buf, '"', ':')
}

func (e *traceJSONEncoder) stringField(name, value string) {
	if value != "" {
		e.field(name)
		e.stringValue(value)
	}
}

func (e *traceJSONEncoder) uint32Field(name string, value uint32) {
	if value != 0 {
		e.field(name)
		e.buf = strconv.AppendUint(e.buf, uint64(value), 10)
	}
}

func (e *traceJSONEncoder) int32Field(name string, value int32) {
	if value != 0 {
		e.field(name)
		e.buf = strconv.AppendInt(e.buf, int64(value), 10)
	}
}

func (e *traceJSONEncoder) uint64Field(name string, value uint64) {
	if value != 0 {
		e.field(name)
		e.buf = append(e.buf, '"')
		e.buf = strconv.AppendUint(e.buf, value, 10)
		e.buf = append(e.buf, '"')
	}
}

func (e *traceJSONEncoder) bytesField(name string, value []byte) {
	if value != nil {
		e.field(name)
		e.bytesValue(value)
	}
}

func (e *traceJSONEncoder) enumField(name string, value int32, names map[int32]string) {
	if value == 0 {
		return
	}
	e.field(name)
	if symbol, ok := names[value]; ok {
		e.stringValue(symbol)
	} else {
		e.buf = strconv.AppendInt(e.buf, int64(value), 10)
	}
}

func appendJSONArray[T any](e *traceJSONEncoder, name string, values []*T, encode func(*traceJSONEncoder, *T)) {
	if values == nil {
		return
	}
	e.field(name)
	e.buf = append(e.buf, '[')
	for i, value := range values {
		if i != 0 {
			e.buf = append(e.buf, ',')
		}
		encode(e, value)
	}
	e.buf = append(e.buf, ']')
}

func (e *traceJSONEncoder) stringsField(name string, values []string) {
	if values == nil {
		return
	}
	e.field(name)
	e.buf = append(e.buf, '[')
	for i, value := range values {
		if i != 0 {
			e.buf = append(e.buf, ',')
		}
		e.stringValue(value)
	}
	e.buf = append(e.buf, ']')
}

func (e *traceJSONEncoder) traceMetrics(v *TraceByIDMetrics) {
	e.buf = append(e.buf, '{')
	e.uint64Field("inspectedBytes", v.InspectedBytes)
	e.uint64Field("backendReads", v.BackendReads)
	e.uint64Field("backendBytes", v.BackendBytes)
	if v.AdditionalMetrics != nil {
		e.field("additionalMetrics")
		e.buf = append(e.buf, '{')
		// jsonpb sorts map keys.
		keys := make([]string, 0, len(v.AdditionalMetrics))
		for key := range v.AdditionalMetrics {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for i, key := range keys {
			if i != 0 {
				e.buf = append(e.buf, ',')
			}
			e.stringValue(key)
			e.buf = append(e.buf, ':', '"')
			e.buf = strconv.AppendInt(e.buf, v.AdditionalMetrics[key], 10)
			e.buf = append(e.buf, '"')
		}
		e.buf = append(e.buf, '}')
	}
	e.buf = append(e.buf, '}')
}

// trace takes the root field name because the V1 API uses the legacy
// "batches" name and the V2 response uses "resourceSpans".
func (e *traceJSONEncoder) trace(v *Trace, rootField string) {
	e.buf = append(e.buf, '{')
	appendJSONArray(e, rootField, v.ResourceSpans, (*traceJSONEncoder).resourceSpans)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) resourceSpans(v *tracev1.ResourceSpans) {
	if !e.message(v != nil) {
		return
	}
	if v.Resource != nil {
		e.field("resource")
		e.resource(v.Resource)
	}
	appendJSONArray(e, "scopeSpans", v.ScopeSpans, (*traceJSONEncoder).scopeSpans)
	e.stringField("schemaUrl", v.SchemaUrl)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) resource(v *resourcev1.Resource) {
	e.buf = append(e.buf, '{')
	appendJSONArray(e, "attributes", v.Attributes, (*traceJSONEncoder).keyValue)
	e.uint32Field("droppedAttributesCount", v.DroppedAttributesCount)
	appendJSONArray(e, "entityRefs", v.EntityRefs, (*traceJSONEncoder).entityRef)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) entityRef(v *commonv1.EntityRef) {
	if !e.message(v != nil) {
		return
	}
	e.stringField("schemaUrl", v.SchemaUrl)
	e.stringField("type", v.Type)
	e.stringsField("idKeys", v.IdKeys)
	e.stringsField("descriptionKeys", v.DescriptionKeys)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) scopeSpans(v *tracev1.ScopeSpans) {
	if !e.message(v != nil) {
		return
	}
	if v.Scope != nil {
		e.field("scope")
		e.scope(v.Scope)
	}
	appendJSONArray(e, "spans", v.Spans, (*traceJSONEncoder).span)
	e.stringField("schemaUrl", v.SchemaUrl)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) scope(v *commonv1.InstrumentationScope) {
	e.buf = append(e.buf, '{')
	e.stringField("name", v.Name)
	e.stringField("version", v.Version)
	appendJSONArray(e, "attributes", v.Attributes, (*traceJSONEncoder).keyValue)
	e.uint32Field("droppedAttributesCount", v.DroppedAttributesCount)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) span(v *tracev1.Span) {
	if !e.message(v != nil) {
		return
	}
	e.bytesField("traceId", v.TraceId)
	e.bytesField("spanId", v.SpanId)
	e.stringField("traceState", v.TraceState)
	e.bytesField("parentSpanId", v.ParentSpanId)
	e.uint32Field("flags", v.Flags)
	e.stringField("name", v.Name)
	e.enumField("kind", int32(v.Kind), tracev1.Span_SpanKind_name)
	e.uint64Field("startTimeUnixNano", v.StartTimeUnixNano)
	e.uint64Field("endTimeUnixNano", v.EndTimeUnixNano)
	appendJSONArray(e, "attributes", v.Attributes, (*traceJSONEncoder).keyValue)
	e.uint32Field("droppedAttributesCount", v.DroppedAttributesCount)
	appendJSONArray(e, "events", v.Events, (*traceJSONEncoder).event)
	e.uint32Field("droppedEventsCount", v.DroppedEventsCount)
	appendJSONArray(e, "links", v.Links, (*traceJSONEncoder).link)
	e.uint32Field("droppedLinksCount", v.DroppedLinksCount)
	if v.Status != nil {
		e.field("status")
		e.status(v.Status)
	}
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) event(v *tracev1.Span_Event) {
	if !e.message(v != nil) {
		return
	}
	e.uint64Field("timeUnixNano", v.TimeUnixNano)
	e.stringField("name", v.Name)
	appendJSONArray(e, "attributes", v.Attributes, (*traceJSONEncoder).keyValue)
	e.uint32Field("droppedAttributesCount", v.DroppedAttributesCount)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) link(v *tracev1.Span_Link) {
	if !e.message(v != nil) {
		return
	}
	e.bytesField("traceId", v.TraceId)
	e.bytesField("spanId", v.SpanId)
	e.stringField("traceState", v.TraceState)
	appendJSONArray(e, "attributes", v.Attributes, (*traceJSONEncoder).keyValue)
	e.uint32Field("droppedAttributesCount", v.DroppedAttributesCount)
	e.uint32Field("flags", v.Flags)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) status(v *tracev1.Status) {
	e.buf = append(e.buf, '{')
	e.stringField("message", v.Message)
	e.enumField("code", int32(v.Code), tracev1.Status_StatusCode_name)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) keyValue(v *commonv1.KeyValue) {
	if !e.message(v != nil) {
		return
	}
	e.stringField("key", v.Key)
	if v.Value != nil {
		e.field("value")
		e.anyValue(v.Value)
	}
	e.int32Field("keyStrindex", v.KeyStrindex)
	e.buf = append(e.buf, '}')
}

// oneof rejects a typed-nil oneof wrapper. Protobuf decoding cannot produce
// one, but jsonpb panics on it, so return an error instead.
func (e *traceJSONEncoder) oneof(present bool) bool {
	if !present {
		e.err = errors.New("typed-nil AnyValue oneof wrapper")
	}
	return present
}

func (e *traceJSONEncoder) anyValue(v *commonv1.AnyValue) {
	if !e.message(v != nil) {
		return
	}
	switch value := v.Value.(type) {
	case nil:
	case *commonv1.AnyValue_StringValue:
		if e.oneof(value != nil) {
			e.field("stringValue")
			e.stringValue(value.StringValue)
		}
	case *commonv1.AnyValue_BoolValue:
		if e.oneof(value != nil) {
			e.field("boolValue")
			e.buf = strconv.AppendBool(e.buf, value.BoolValue)
		}
	case *commonv1.AnyValue_IntValue:
		if e.oneof(value != nil) {
			e.field("intValue")
			e.buf = append(e.buf, '"')
			e.buf = strconv.AppendInt(e.buf, value.IntValue, 10)
			e.buf = append(e.buf, '"')
		}
	case *commonv1.AnyValue_DoubleValue:
		if e.oneof(value != nil) {
			e.field("doubleValue")
			e.doubleValue(value.DoubleValue)
		}
	case *commonv1.AnyValue_BytesValue:
		if e.oneof(value != nil) {
			e.field("bytesValue")
			e.bytesValue(value.BytesValue)
		}
	case *commonv1.AnyValue_ArrayValue:
		if e.oneof(value != nil) {
			e.field("arrayValue")
			e.arrayValue(value.ArrayValue)
		}
	case *commonv1.AnyValue_KvlistValue:
		if e.oneof(value != nil) {
			e.field("kvlistValue")
			e.keyValueList(value.KvlistValue)
		}
	case *commonv1.AnyValue_StringValueStrindex:
		if e.oneof(value != nil) {
			e.field("stringValueStrindex")
			e.buf = strconv.AppendInt(e.buf, int64(value.StringValueStrindex), 10)
		}
	default:
		e.err = fmt.Errorf("unsupported AnyValue alternative %T", value)
	}
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) arrayValue(v *commonv1.ArrayValue) {
	if !e.message(v != nil) {
		return
	}
	appendJSONArray(e, "values", v.Values, (*traceJSONEncoder).anyValue)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) keyValueList(v *commonv1.KeyValueList) {
	if !e.message(v != nil) {
		return
	}
	appendJSONArray(e, "values", v.Values, (*traceJSONEncoder).keyValue)
	e.buf = append(e.buf, '}')
}

func (e *traceJSONEncoder) bytesValue(v []byte) {
	if v == nil {
		e.buf = append(e.buf, "null"...)
		return
	}
	e.buf = append(e.buf, '"')
	e.buf = base64.StdEncoding.AppendEncode(e.buf, v)
	e.buf = append(e.buf, '"')
}

// doubleValue formats finite values like encoding/json, which jsonpb uses.
func (e *traceJSONEncoder) doubleValue(v float64) {
	switch {
	case math.IsNaN(v):
		e.buf = append(e.buf, `"NaN"`...)
	case math.IsInf(v, 1):
		e.buf = append(e.buf, `"Infinity"`...)
	case math.IsInf(v, -1):
		e.buf = append(e.buf, `"-Infinity"`...)
	default:
		format := byte('f')
		if abs := math.Abs(v); abs != 0 && (abs < 1e-6 || abs >= 1e21) {
			format = 'e'
		}
		e.buf = strconv.AppendFloat(e.buf, v, format, -1, 64)
		if format == 'e' {
			// Shorten a two-digit negative exponent: 1e-07 becomes 1e-7.
			if n := len(e.buf); n >= 4 && e.buf[n-4] == 'e' && e.buf[n-3] == '-' && e.buf[n-2] == '0' {
				e.buf[n-2] = e.buf[n-1]
				e.buf = e.buf[:n-1]
			}
		}
	}
}

// stringValue escapes like encoding/json with HTML escaping enabled.
func (e *traceJSONEncoder) stringValue(s string) {
	e.buf = append(e.buf, '"')
	start := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' && c != '<' && c != '>' && c != '&' {
				i++
				continue
			}
			e.buf = append(e.buf, s[start:i]...)
			e.asciiEscape(c)
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// encoding/json writes the raw U+FFFD bytes, not a \ufffd escape.
			e.buf = append(e.buf, s[start:i]...)
			e.buf = append(e.buf, "\uFFFD"...)
			i++
			start = i
			continue
		}
		if r == '\u2028' || r == '\u2029' {
			e.buf = append(e.buf, s[start:i]...)
			e.buf = append(e.buf, '\\', 'u', '2', '0', '2', byte('8'+r-'\u2028'))
			i += size
			start = i
			continue
		}
		i += size
	}
	e.buf = append(e.buf, s[start:]...)
	e.buf = append(e.buf, '"')
}

func (e *traceJSONEncoder) asciiEscape(c byte) {
	switch c {
	case '"', '\\':
		e.buf = append(e.buf, '\\', c)
	case '\n':
		e.buf = append(e.buf, '\\', 'n')
	case '\r':
		e.buf = append(e.buf, '\\', 'r')
	case '\t':
		e.buf = append(e.buf, '\\', 't')
	case '\b':
		e.buf = append(e.buf, '\\', 'b')
	case '\f':
		e.buf = append(e.buf, '\\', 'f')
	default:
		const hex = "0123456789abcdef"
		e.buf = append(e.buf, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
	}
}
