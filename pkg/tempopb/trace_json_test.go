package tempopb

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

// jsonpb is the reference for the typed encoder: output must match it byte
// for byte.
func jsonpbString(t testing.TB, m proto.Message) string {
	t.Helper()
	s, err := new(jsonpb.Marshaler).MarshalToString(m)
	require.NoError(t, err)
	return s
}

// jsonpbV1 renames the root field, which jsonpb always writes first.
func jsonpbV1(t testing.TB, tr *Trace) string {
	t.Helper()
	return strings.Replace(jsonpbString(t, tr), `"resourceSpans":`, `"batches":`, 1)
}

// requireMatchesJsonpb encodes tr in both formats, compares the bytes with
// jsonpb, and checks that encoding did not modify the input. newTrace must
// return a fresh copy of the same trace on each call.
func requireMatchesJsonpb(t testing.TB, newTrace func() *Trace) {
	t.Helper()
	tr, response := newTrace(), testResponse(newTrace())

	gotV1, err := MarshalToJSONV1(tr)
	require.NoError(t, err)
	require.Equal(t, jsonpbV1(t, tr), string(gotV1))

	gotV2, err := MarshalTraceByIDResponseToJSON(response)
	require.NoError(t, err)
	require.Equal(t, jsonpbString(t, response), string(gotV2))

	if diff := cmp.Diff(newTrace(), tr, cmpopts.EquateNaNs()); diff != "" {
		t.Fatalf("trace mutated (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(testResponse(newTrace()), response, cmpopts.EquateNaNs()); diff != "" {
		t.Fatalf("response mutated (-want +got):\n%s", diff)
	}
}

func TestTraceJSONMatchesJsonpb(t *testing.T) {
	for _, tc := range traceJSONCases() {
		t.Run(tc.name, func(t *testing.T) {
			requireMatchesJsonpb(t, tc.trace)
		})
	}
	for seed := int64(1); seed <= 40; seed++ {
		t.Run(fmt.Sprintf("random/%d", seed), func(t *testing.T) {
			requireMatchesJsonpb(t, func() *Trace { return randomTrace(rand.New(rand.NewSource(seed))) })
		})
	}
}

// TestTraceJSONFormat pins the encoding rules independently of jsonpb.
func TestTraceJSONFormat(t *testing.T) {
	t.Run("values", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			value *commonv1.AnyValue
			want  string
		}{
			{"unset", &commonv1.AnyValue{}, `{}`},
			{"empty string", stringValue(""), `{"stringValue":""}`},
			{"html and line separators", stringValue("<>&\u2028\u2029"), `{"stringValue":"\u003c\u003e\u0026\u2028\u2029"}`},
			{"control characters", stringValue("\"\\\n\r\t\b\f\x00\x1f"), `{"stringValue":"\"\\\n\r\t\b\f\u0000\u001f"}`},
			{"invalid utf-8", stringValue("a\xffb"), "{\"stringValue\":\"a\uFFFDb\"}"},
			{"false", &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{}}, `{"boolValue":false}`},
			{"zero int", intValue(0), `{"intValue":"0"}`},
			{"max int64", intValue(math.MaxInt64), `{"intValue":"9223372036854775807"}`},
			{"nil bytes", &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{}}, `{"bytesValue":null}`},
			{"empty bytes", &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{BytesValue: []byte{}}}, `{"bytesValue":""}`},
			{"bytes", &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{BytesValue: []byte{0, 1, 254, 255}}}, `{"bytesValue":"AAH+/w=="}`},
			{"nil array", &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{}}, `{"arrayValue":null}`},
			{"array with nil", &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{ArrayValue: &commonv1.ArrayValue{Values: []*commonv1.AnyValue{nil, {}}}}}, `{"arrayValue":{"values":[null,{}]}}`},
			{"string index", &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValueStrindex{StringValueStrindex: -42}}, `{"stringValueStrindex":-42}`},
			{"negative zero", doubleValue(math.Copysign(0, -1)), `{"doubleValue":-0}`},
			{"small exponent", doubleValue(1e-7), `{"doubleValue":1e-7}`},
			{"small fixed", doubleValue(1e-6), `{"doubleValue":0.000001}`},
			{"large fixed", doubleValue(1e20), `{"doubleValue":100000000000000000000}`},
			{"large exponent", doubleValue(1e21), `{"doubleValue":1e+21}`},
			{"NaN", doubleValue(math.NaN()), `{"doubleValue":"NaN"}`},
			{"infinity", doubleValue(math.Inf(1)), `{"doubleValue":"Infinity"}`},
			{"negative infinity", doubleValue(math.Inf(-1)), `{"doubleValue":"-Infinity"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				got, err := MarshalToJSONV1(valueTrace(tc.value))
				require.NoError(t, err)
				require.Equal(t, `{"batches":[{"scopeSpans":[{"spans":[{"attributes":[{"key":"value","value":`+tc.want+`}]}]}]}]}`, string(got))
			})
		}
	})

	t.Run("messages", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			v1   *Trace
			v2   *TraceByIDResponse
			want string
		}{
			{name: "nil root", v1: &Trace{}, want: `{}`},
			{name: "empty root", v1: &Trace{ResourceSpans: []*tracev1.ResourceSpans{}}, want: `{"batches":[]}`},
			{name: "nil element", v1: &Trace{ResourceSpans: []*tracev1.ResourceSpans{nil}}, want: `{"batches":[null]}`},
			{name: "empty span", v1: &Trace{ResourceSpans: []*tracev1.ResourceSpans{{ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{
				TraceId: []byte{0, 1}, Kind: tracev1.Span_SPAN_KIND_SERVER, StartTimeUnixNano: math.MaxUint64, Status: &tracev1.Status{Code: 123},
			}}}}}}}, want: `{"batches":[{"scopeSpans":[{"spans":[{"traceId":"AAE=","kind":"SPAN_KIND_SERVER","startTimeUnixNano":"18446744073709551615","status":{"code":123}}]}]}]}`},
			{name: "empty response", v2: &TraceByIDResponse{}, want: `{}`},
			{name: "response", v2: &TraceByIDResponse{
				Trace:   &Trace{},
				Metrics: &TraceByIDMetrics{InspectedBytes: 1, AdditionalMetrics: map[string]int64{"b": -1, "a": 0}},
				Status:  PartialStatus_PARTIAL,
			}, want: `{"trace":{},"metrics":{"inspectedBytes":"1","additionalMetrics":{"a":"0","b":"-1"}},"status":"PARTIAL"}`},
			{name: "unknown status", v2: &TraceByIDResponse{Status: PartialStatus(123)}, want: `{"status":123}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var got []byte
				var err error
				if tc.v1 != nil {
					got, err = MarshalToJSONV1(tc.v1)
				} else {
					got, err = MarshalTraceByIDResponseToJSON(tc.v2)
				}
				require.NoError(t, err)
				require.Equal(t, tc.want, string(got))
			})
		}
	})
}

func TestTraceJSONErrors(t *testing.T) {
	_, err := MarshalToJSONV1(nil)
	require.EqualError(t, err, "Marshal called with nil")
	_, err = MarshalTraceByIDResponseToJSON(nil)
	require.EqualError(t, err, "Marshal called with nil")

	for _, value := range []*commonv1.AnyValue{
		{Value: (*commonv1.AnyValue_StringValue)(nil)},
		{Value: (*commonv1.AnyValue_BoolValue)(nil)},
		{Value: (*commonv1.AnyValue_IntValue)(nil)},
		{Value: (*commonv1.AnyValue_DoubleValue)(nil)},
		{Value: (*commonv1.AnyValue_BytesValue)(nil)},
		{Value: (*commonv1.AnyValue_ArrayValue)(nil)},
		{Value: (*commonv1.AnyValue_KvlistValue)(nil)},
		{Value: (*commonv1.AnyValue_StringValueStrindex)(nil)},
	} {
		t.Run(fmt.Sprintf("typed-nil %T", value.Value), func(t *testing.T) {
			tr := valueTrace(value)
			got, err := MarshalToJSONV1(tr)
			require.ErrorContains(t, err, "typed-nil")
			require.Nil(t, got)
			got, err = MarshalTraceByIDResponseToJSON(&TraceByIDResponse{Trace: tr})
			require.ErrorContains(t, err, "typed-nil")
			require.Nil(t, got)
		})
	}
}

// TestTraceJSONAllFields fills every field reachable from the encoded messages,
// including every oneof alternative, and compares the output with jsonpb. A
// field, message, or oneof alternative added upstream fails this test until
// the typed encoder supports it.
func TestTraceJSONAllFields(t *testing.T) {
	f := &fieldFiller{t: t, onPath: map[reflect.Type]int{}, seen: map[reflect.Type]bool{}}
	tr, response := &Trace{}, &TraceByIDResponse{}
	f.fill(reflect.ValueOf(tr).Elem())
	f.fill(reflect.ValueOf(response).Elem())

	const msg = "the typed encoder does not match jsonpb; update trace_json.go for new or changed fields"
	got, err := MarshalToJSONV1(tr)
	require.NoError(t, err, msg)
	require.Equal(t, jsonpbV1(t, tr), string(got), msg)

	got, err = MarshalTraceByIDResponseToJSON(response)
	require.NoError(t, err, msg)
	require.Equal(t, jsonpbString(t, response), string(got), msg)

	for _, wrapper := range (&commonv1.AnyValue{}).XXX_OneofWrappers() {
		require.True(t, f.seen[reflect.TypeOf(wrapper)], "oneof alternative %T not filled", wrapper)
	}
}

// fieldFiller sets every field of a generated message to a nonzero value.
// Recursive message types are expanded at most twice on any path.
type fieldFiller struct {
	t      testing.TB
	onPath map[reflect.Type]int
	seen   map[reflect.Type]bool
	next   int
}

func (f *fieldFiller) fill(v reflect.Value) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			switch {
			case strings.HasPrefix(field.Name, "XXX_"):
			case field.Tag.Get("protobuf_oneof") != "":
				f.fillOneof(v, v.Field(i))
			default:
				f.fill(v.Field(i))
			}
		}
	case reflect.Pointer:
		elem := v.Type().Elem()
		if f.onPath[elem] >= 2 {
			return
		}
		f.onPath[elem]++
		p := reflect.New(elem)
		f.fill(p.Elem())
		f.onPath[elem]--
		v.Set(p)
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes([]byte{0, 1, 254, 255})
			return
		}
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := 0; i < s.Len(); i++ {
			f.fill(s.Index(i))
		}
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		key, value := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()
		f.fill(key)
		f.fill(value)
		m.SetMapIndex(key, value)
		v.Set(m)
	case reflect.String:
		v.SetString("value <>&")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int32, reflect.Int64:
		v.SetInt(1) // 1 is a named value of every enum.
	case reflect.Uint32, reflect.Uint64:
		v.SetUint(7)
	case reflect.Float64:
		v.SetFloat(1.5)
	default:
		f.t.Fatalf("fieldFiller: unsupported kind %s for %s", v.Kind(), v.Type())
	}
}

// fillOneof cycles through the alternatives so that repeated values cover all
// of them.
func (f *fieldFiller) fillOneof(message, field reflect.Value) {
	wrappers := message.Addr().Interface().(interface{ XXX_OneofWrappers() []any }).XXX_OneofWrappers()
	wrapper := reflect.TypeOf(wrappers[f.next%len(wrappers)])
	f.next++
	f.seen[wrapper] = true
	w := reflect.New(wrapper.Elem())
	f.fill(w.Elem())
	field.Set(w)
}

// FuzzTraceJSON compares the typed encoder with jsonpb on fuzzed traces. Run with:
//
//	go test ./pkg/tempopb -run '^$' -fuzz '^FuzzTraceJSON$' -fuzztime 30s
func FuzzTraceJSON(f *testing.F) {
	for _, seed := range []string{"", "batches", `{"resourceSpans":[]}`, "\xff\x00\x7f\x80", "<>&\\\"\u2028\u2029"} {
		f.Add([]byte(seed), int64(1))
	}
	f.Fuzz(func(t *testing.T, data []byte, seed int64) {
		requireMatchesJsonpb(t, func() *Trace {
			tr := randomTrace(rand.New(rand.NewSource(seed)))
			s := string(data)
			tr.ResourceSpans = append(tr.ResourceSpans, &tracev1.ResourceSpans{ScopeSpans: []*tracev1.ScopeSpans{{
				Scope: &commonv1.InstrumentationScope{Name: s},
				Spans: []*tracev1.Span{{
					Name: s, TraceId: data,
					Attributes: []*commonv1.KeyValue{
						{Key: s, Value: stringValue(s)},
						{Key: "bytes", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{BytesValue: data}}},
					},
				}},
			}}})
			return tr
		})
	})
}

type traceJSONCase struct {
	name  string
	trace func() *Trace
}

// traceJSONCases returns constructors rather than values so each test can
// build an identical copy to detect input mutation.
func traceJSONCases() []traceJSONCase {
	cases := []traceJSONCase{
		{"empty", func() *Trace { return &Trace{} }},
		{"empty resources", func() *Trace { return &Trace{ResourceSpans: []*tracev1.ResourceSpans{}} }},
		{"nil messages", func() *Trace {
			return &Trace{ResourceSpans: []*tracev1.ResourceSpans{nil, {
				Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{nil}, EntityRefs: []*commonv1.EntityRef{nil}},
				ScopeSpans: []*tracev1.ScopeSpans{nil, {
					Scope: &commonv1.InstrumentationScope{Attributes: []*commonv1.KeyValue{}},
					Spans: []*tracev1.Span{nil, {Events: []*tracev1.Span_Event{nil}, Links: []*tracev1.Span_Link{nil}}},
				}},
			}}}
		}},
		{"empty fields", func() *Trace {
			return &Trace{ResourceSpans: []*tracev1.ResourceSpans{{
				Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{}, EntityRefs: []*commonv1.EntityRef{{IdKeys: []string{}, DescriptionKeys: []string{}}}},
				ScopeSpans: []*tracev1.ScopeSpans{{Scope: &commonv1.InstrumentationScope{}, Spans: []*tracev1.Span{{
					TraceId: []byte{}, SpanId: []byte{}, ParentSpanId: []byte{}, Attributes: []*commonv1.KeyValue{},
					Events: []*tracev1.Span_Event{}, Links: []*tracev1.Span_Link{}, Status: &tracev1.Status{},
				}}}},
			}}}
		}},
		{"all fields", func() *Trace { return fullTrace(1) }},
		{"256 spans", func() *Trace { return fullTrace(256) }},
		{"unknown enums", func() *Trace {
			tr := fullTrace(1)
			span := tr.ResourceSpans[0].ScopeSpans[0].Spans[0]
			span.Kind = tracev1.Span_SpanKind(-123)
			span.Status.Code = tracev1.Status_StatusCode(123)
			return tr
		}},
	}

	var controls strings.Builder
	for i := 0; i < 256; i++ {
		controls.WriteByte(byte(i))
	}
	values := map[string]func() *commonv1.AnyValue{
		"unset":        func() *commonv1.AnyValue { return &commonv1.AnyValue{} },
		"empty string": func() *commonv1.AnyValue { return stringValue("") },
		"all bytes":    func() *commonv1.AnyValue { return stringValue(controls.String()) },
		"unicode":      func() *commonv1.AnyValue { return stringValue("<>&\"\\ /\u2028\u2029 ☃ 😀 \xed\xa0\x80\xc0") },
		"false":        func() *commonv1.AnyValue { return &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{}} },
		"min int64":    func() *commonv1.AnyValue { return intValue(math.MinInt64) },
		"nil bytes":    func() *commonv1.AnyValue { return &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{}} },
		"nil array":    func() *commonv1.AnyValue { return &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{}} },
		"nil kvlist":   func() *commonv1.AnyValue { return &commonv1.AnyValue{Value: &commonv1.AnyValue_KvlistValue{}} },
		"empty kvlist": func() *commonv1.AnyValue {
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_KvlistValue{KvlistValue: &commonv1.KeyValueList{Values: []*commonv1.KeyValue{}}}}
		},
		"zero string index": func() *commonv1.AnyValue {
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValueStrindex{}}
		},
		"nested": nestedValue,
	}
	for name, value := range values {
		cases = append(cases, traceJSONCase{"value/" + name, func() *Trace { return valueTrace(value()) }})
	}
	for _, f := range []float64{0, math.Copysign(0, -1), 1.25, -123.456, 1e-7, 1e-6, 1e20, 1e21, math.SmallestNonzeroFloat64, math.MaxFloat64, math.NaN(), math.Inf(1), math.Inf(-1)} {
		cases = append(cases, traceJSONCase{fmt.Sprintf("float/%g", f), func() *Trace { return valueTrace(doubleValue(f)) }})
	}
	// User data spelled like either root field must not be rewritten.
	for i, payload := range []string{`{"resourceSpans":[]}`, `"batches":`, "batches", "resourceSpans"} {
		cases = append(cases, traceJSONCase{fmt.Sprintf("root field in user data/%d", i), func() *Trace {
			tr := fullTrace(1)
			tr.ResourceSpans[0].ScopeSpans[0].Scope.Name = payload
			span := tr.ResourceSpans[0].ScopeSpans[0].Spans[0]
			span.Name = payload
			span.Attributes = append(span.Attributes, &commonv1.KeyValue{Key: payload, Value: stringValue(payload)})
			return tr
		}})
	}
	return cases
}

func stringValue(s string) *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: s}}
}

func intValue(v int64) *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: v}}
}

func doubleValue(v float64) *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: v}}
}

func nestedValue() *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{ArrayValue: &commonv1.ArrayValue{Values: []*commonv1.AnyValue{
		nil,
		{},
		intValue(9007199254740993),
		{Value: &commonv1.AnyValue_KvlistValue{KvlistValue: &commonv1.KeyValueList{Values: []*commonv1.KeyValue{
			nil, {Key: "duplicate", Value: intValue(1)}, {Key: "duplicate", Value: intValue(2)},
		}}}},
	}}}}
}

func valueTrace(v *commonv1.AnyValue) *Trace {
	return &Trace{ResourceSpans: []*tracev1.ResourceSpans{{ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{
		Attributes: []*commonv1.KeyValue{{Key: "value", Value: v}},
	}}}}}}}
}

// fullTrace sets every reachable field to a nonzero value.
func fullTrace(spans int) *Trace {
	attrs := func() []*commonv1.KeyValue {
		return []*commonv1.KeyValue{{Key: "resourceSpans", KeyStrindex: 17, Value: nestedValue()}, {Key: "no value"}}
	}
	ss := &tracev1.ScopeSpans{
		Scope:     &commonv1.InstrumentationScope{Name: "library <>&", Version: "1", Attributes: attrs(), DroppedAttributesCount: math.MaxUint32},
		SchemaUrl: "scope/schema",
	}
	for i := range spans {
		ss.Spans = append(ss.Spans, &tracev1.Span{
			TraceId: []byte{0, 1, 254, 255}, SpanId: []byte{2, 3}, ParentSpanId: []byte{4}, TraceState: "state=one",
			Flags: math.MaxUint32, Name: fmt.Sprintf("span-%d", i), Kind: tracev1.Span_SPAN_KIND_SERVER,
			StartTimeUnixNano: math.MaxUint64, EndTimeUnixNano: 9007199254740993, Attributes: attrs(), DroppedAttributesCount: math.MaxUint32,
			Events: []*tracev1.Span_Event{{TimeUnixNano: math.MaxUint64, Name: "event", Attributes: attrs(), DroppedAttributesCount: 1}}, DroppedEventsCount: 2,
			Links: []*tracev1.Span_Link{{TraceId: []byte{5}, SpanId: []byte{6}, TraceState: "other", Attributes: attrs(), DroppedAttributesCount: 3, Flags: math.MaxUint32}}, DroppedLinksCount: 4,
			Status: &tracev1.Status{Message: "failed <>&", Code: tracev1.Status_STATUS_CODE_ERROR},
		})
	}
	return &Trace{ResourceSpans: []*tracev1.ResourceSpans{{
		Resource: &resourcev1.Resource{Attributes: attrs(), DroppedAttributesCount: 5, EntityRefs: []*commonv1.EntityRef{{
			SchemaUrl: "entity/schema", Type: "service", IdKeys: []string{"service.name", ""}, DescriptionKeys: []string{"description"},
		}}}, ScopeSpans: []*tracev1.ScopeSpans{ss}, SchemaUrl: "resource/schema",
	}}}
}

func testResponse(tr *Trace) *TraceByIDResponse {
	return &TraceByIDResponse{
		Trace: tr,
		Metrics: &TraceByIDMetrics{
			InspectedBytes: math.MaxUint64, BackendReads: 9007199254740993, BackendBytes: 1,
			AdditionalMetrics: map[string]int64{"resourceSpans": math.MinInt64, "batches": math.MaxInt64, "zero": 0, "\"<>&\u2028\xff": -42},
		},
		Status: PartialStatus_PARTIAL, Message: "partial \"<>&\u2028\xff",
	}
}

// randomTrace builds a deterministic trace that mixes nil, empty, and set
// values at every level, including strings that resemble the root field names.
func randomTrace(rnd *rand.Rand) *Trace {
	randomString := func() string {
		return []string{"", "batches", "resourceSpans", `{"resourceSpans":["batches"]}`, "<>&\u2028\u2029☃ \x00\x1f\x7f\xff", fmt.Sprintf("s-%d", rnd.Int63())}[rnd.Intn(6)]
	}
	var value func(depth int) *commonv1.AnyValue
	value = func(depth int) *commonv1.AnyValue {
		switch rnd.Intn(10) {
		case 0:
			return &commonv1.AnyValue{}
		case 1:
			return stringValue(randomString())
		case 2:
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: rnd.Intn(2) == 0}}
		case 3:
			return intValue(int64(rnd.Uint64()))
		case 4:
			return doubleValue(math.Float64frombits(rnd.Uint64()))
		case 5:
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{BytesValue: []byte(randomString())}}
		case 6:
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{}}
		case 7:
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValueStrindex{StringValueStrindex: rnd.Int31() - rnd.Int31()}}
		case 8:
			if depth <= 0 {
				return intValue(1)
			}
			values := make([]*commonv1.AnyValue, rnd.Intn(4))
			for i := range values {
				if rnd.Intn(5) != 0 {
					values[i] = value(depth - 1)
				}
			}
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{ArrayValue: &commonv1.ArrayValue{Values: values}}}
		default:
			if depth <= 0 {
				return intValue(2)
			}
			values := make([]*commonv1.KeyValue, rnd.Intn(3))
			for i := range values {
				values[i] = &commonv1.KeyValue{Key: randomString(), KeyStrindex: rnd.Int31(), Value: value(depth - 1)}
			}
			return &commonv1.AnyValue{Value: &commonv1.AnyValue_KvlistValue{KvlistValue: &commonv1.KeyValueList{Values: values}}}
		}
	}
	attributes := func() []*commonv1.KeyValue {
		if rnd.Intn(4) == 0 {
			return nil
		}
		attrs := make([]*commonv1.KeyValue, rnd.Intn(4))
		for i := range attrs {
			attrs[i] = &commonv1.KeyValue{Key: randomString(), Value: value(3)}
		}
		return attrs
	}
	span := func() *tracev1.Span {
		if rnd.Intn(8) == 0 {
			return nil
		}
		span := &tracev1.Span{
			TraceId: []byte(randomString()), SpanId: []byte(randomString()), ParentSpanId: []byte(randomString()),
			TraceState: randomString(), Name: randomString(), Kind: tracev1.Span_SpanKind(rnd.Intn(8) - 1),
			Flags: rnd.Uint32(), StartTimeUnixNano: rnd.Uint64(), EndTimeUnixNano: rnd.Uint64(),
			Attributes: attributes(), DroppedAttributesCount: rnd.Uint32(),
			DroppedEventsCount: rnd.Uint32(), DroppedLinksCount: rnd.Uint32(),
		}
		if rnd.Intn(3) == 0 {
			span.Status = &tracev1.Status{Message: randomString(), Code: tracev1.Status_StatusCode(rnd.Intn(4) - 1)}
		}
		if rnd.Intn(3) == 0 {
			span.Events = []*tracev1.Span_Event{{TimeUnixNano: rnd.Uint64(), Name: randomString(), Attributes: attributes()}}
		}
		if rnd.Intn(3) == 0 {
			span.Links = []*tracev1.Span_Link{{TraceId: []byte(randomString()), SpanId: []byte(randomString()), Attributes: attributes(), Flags: rnd.Uint32()}}
		}
		return span
	}
	tr := &Trace{}
	for i := rnd.Intn(3); i > 0; i-- {
		resource := &tracev1.ResourceSpans{SchemaUrl: randomString()}
		if rnd.Intn(3) != 0 {
			resource.Resource = &resourcev1.Resource{
				Attributes: attributes(), DroppedAttributesCount: rnd.Uint32(),
				EntityRefs: []*commonv1.EntityRef{{SchemaUrl: randomString(), Type: randomString(), IdKeys: []string{randomString(), ""}}},
			}
		}
		scope := &tracev1.ScopeSpans{SchemaUrl: randomString()}
		if rnd.Intn(3) != 0 {
			scope.Scope = &commonv1.InstrumentationScope{Name: randomString(), Version: randomString(), Attributes: attributes()}
		}
		for j := rnd.Intn(4); j > 0; j-- {
			scope.Spans = append(scope.Spans, span())
		}
		resource.ScopeSpans = []*tracev1.ScopeSpans{scope}
		tr.ResourceSpans = append(tr.ResourceSpans, resource)
	}
	return tr
}
