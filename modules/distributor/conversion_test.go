package distributor

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/entity"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

func TestDirectTracesSchemaFields(t *testing.T) {
	// New protobuf fields need an explicit conversion decision even when fixtures leave them empty.
	tests := []struct {
		message any
		fields  string
	}{
		{&tempopb.Trace{}, "1:resourceSpans"},
		{&trace.ResourceSpans{}, "1:resource 2:scope_spans 3:schema_url"},
		{&resource.Resource{}, "1:attributes 2:dropped_attributes_count 3:entity_refs"},
		{&trace.ScopeSpans{}, "1:scope 2:spans 3:schema_url"},
		{&common.InstrumentationScope{}, "1:name 2:version 3:attributes 4:dropped_attributes_count"},
		{&trace.Span{}, "1:trace_id 2:span_id 3:trace_state 4:parent_span_id 5:name 6:kind 7:start_time_unix_nano 8:end_time_unix_nano 9:attributes 10:dropped_attributes_count 11:events 12:dropped_events_count 13:links 14:dropped_links_count 15:status 16:flags"},
		{&trace.Span_Event{}, "1:time_unix_nano 2:name 3:attributes 4:dropped_attributes_count"},
		{&trace.Span_Link{}, "1:trace_id 2:span_id 3:trace_state 4:attributes 5:dropped_attributes_count 6:flags"},
		{&trace.Status{}, "2:message 3:code"},
		{&common.KeyValue{}, "1:key 2:value 3:key_strindex"},
		{&common.AnyValue{}, "1:string_value 2:bool_value 3:int_value 4:double_value 5:array_value 6:kvlist_value 7:bytes_value 8:string_value_strindex"},
		{&common.ArrayValue{}, "1:values"},
		{&common.KeyValueList{}, "1:values"},
		{&common.EntityRef{}, "1:schema_url 2:type 3:id_keys 4:description_keys"},
	}
	for _, tt := range tests {
		t.Run(reflect.TypeOf(tt.message).Elem().Name(), func(t *testing.T) {
			got := protobufFields(reflect.TypeOf(tt.message).Elem())
			if m, ok := tt.message.(interface{ XXX_OneofWrappers() []interface{} }); ok {
				for _, wrapper := range m.XXX_OneofWrappers() {
					got = append(got, protobufFields(reflect.TypeOf(wrapper).Elem())...)
				}
			}
			require.ElementsMatch(t, strings.Fields(tt.fields), got)
		})
	}
}

func protobufFields(message reflect.Type) []string {
	var fields []string
	for i := 0; i < message.NumField(); i++ {
		tag := message.Field(i).Tag.Get("protobuf")
		if tag == "" {
			continue
		}
		parts := strings.Split(tag, ",")
		for _, part := range parts[2:] {
			if name, ok := strings.CutPrefix(part, "name="); ok {
				fields = append(fields, parts[1]+":"+name)
				break
			}
		}
	}
	return fields
}

func TestDirectTracesParity(t *testing.T) {
	for _, n := range []int{0, 1, 100} {
		for _, rich := range []bool{false, true} {
			td := conversionFixture(n, rich)
			// Empty resource/scope/span records exercise protobuf message presence.
			td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
			data, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(td)
			require.NoError(t, err)
			want := &tempopb.Trace{}
			require.NoError(t, want.Unmarshal(data))
			got := directTraces(td)
			require.Equal(t, want, got)
			// Destroy the source data after conversion; output must remain valid.
			td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool {
				rs.Resource().Attributes().Clear()
				rss := rs.ScopeSpans()
				for i := 0; i < rss.Len(); i++ {
					ss := rss.At(i)
					ss.Spans().RemoveIf(func(s ptrace.Span) bool {
						s.Attributes().Clear()
						s.SetName("changed")
						s.Events().RemoveIf(func(ptrace.SpanEvent) bool { return true })
						s.Links().RemoveIf(func(ptrace.SpanLink) bool { return true })
						return true
					})
				}
				return true
			})
			require.Equal(t, want, got)
		}
	}
}

func assertConversionParity(t testing.TB, td ptrace.Traces) *tempopb.Trace {
	t.Helper()
	data, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(td)
	require.NoError(t, err)
	want := &tempopb.Trace{}
	require.NoError(t, want.Unmarshal(data))
	got := directTraces(td)
	wantWire, err := want.Marshal()
	require.NoError(t, err)
	gotWire, err := got.Marshal()
	require.NoError(t, err)
	require.Equal(t, wantWire, gotWire)
	return got
}

func TestDirectTracesEntitiesAndValues(t *testing.T) {
	td := conversionFixture(1, true)
	rs := td.ResourceSpans().At(0)
	refs := entity.ResourceEntityRefs(rs.Resource())
	for range 2 {
		ref := refs.AppendEmpty()
		ref.SetType("service")
		ref.SetSchemaUrl("https://example.com/schema")
		ref.IdKeys().FromRaw([]string{"service.name", "service.instance.id"})
		ref.DescriptionKeys().FromRaw([]string{"region", "zone"})
	}
	refs.AppendEmpty()
	ss := rs.ScopeSpans().At(0)
	span := ss.Spans().At(0)
	maps := []pcommon.Map{rs.Resource().Attributes(), ss.Scope().Attributes(), span.Attributes(), span.Events().At(0).Attributes(), span.Links().At(0).Attributes()}
	for _, m := range maps {
		m.PutEmptyBytes("empty-bytes")
		m.PutEmptyMap("empty-map")
		m.PutEmptySlice("empty-slice")
		m.PutStr("empty-string", "")
		m.PutBool("false", false)
		m.PutInt("zero", 0)
		m.PutEmptyBytes("bytes").FromRaw([]byte{1, 2, 3})
		array := m.PutEmptySlice("nested")
		array.AppendEmpty().SetEmptyBytes()
		array.AppendEmpty().SetEmptyMap().PutEmptyBytes("bytes")
	}
	got := assertConversionParity(t, td)
	before, err := got.Marshal()
	require.NoError(t, err)
	refs.At(0).IdKeys().SetAt(0, "changed")
	refs.At(0).DescriptionKeys().SetAt(0, "changed")
	refs.At(0).SetType("changed")
	for _, m := range maps {
		v, ok := m.Get("bytes")
		require.True(t, ok)
		v.Bytes().SetAt(0, 99)
	}
	span.SetTraceID(pcommon.TraceID{99})
	span.SetSpanID(pcommon.SpanID{99})
	span.Links().At(0).SetTraceID(pcommon.TraceID{99})
	after, err := got.Marshal()
	require.NoError(t, err)
	require.Equal(t, before, after, "converted records must own mutable slices")
}

func TestDirectTracesProfileOnlyFields(t *testing.T) {
	// OTLP specifies empty semantics for profiling string indexes in trace data.
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			td := conversionFixture(1, false)
			data, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(td)
			require.NoError(t, err)
			input := &tempopb.Trace{}
			require.NoError(t, input.Unmarshal(data))
			attr := &common.KeyValue{KeyStrindex: 12, Value: &common.AnyValue{Value: &common.AnyValue_StringValueStrindex{StringValueStrindex: 34}}}
			attrs := input.ResourceSpans[0].Resource.Attributes
			if nested {
				attrs = append(attrs, &common.KeyValue{Key: "nested", Value: &common.AnyValue{Value: &common.AnyValue_KvlistValue{KvlistValue: &common.KeyValueList{Values: []*common.KeyValue{attr}}}}})
			} else {
				attrs = append(attrs, attr)
			}
			input.ResourceSpans[0].Resource.Attributes = attrs
			data, err = input.Marshal()
			require.NoError(t, err)
			td, err = (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(data)
			require.NoError(t, err)
			attr.KeyStrindex = 0
			attr.Value.Value = nil
			require.Equal(t, input, directTraces(td))
		})
	}
}

func FuzzDirectTraces(f *testing.F) {
	for _, rich := range []bool{false, true} {
		td := conversionFixture(1, rich)
		td.ResourceSpans().At(0).Resource().Attributes().PutEmptyBytes("empty-bytes")
		ref := entity.ResourceEntityRefs(td.ResourceSpans().At(0).Resource()).AppendEmpty()
		ref.SetType("service")
		ref.IdKeys().Append("service.name")
		data, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(td)
		require.NoError(f, err)
		f.Add(data)
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		td, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(data)
		if err != nil {
			return
		}
		wire, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(td)
		require.NoError(t, err)
		want := &tempopb.Trace{}
		require.NoError(t, want.Unmarshal(wire))
		normalizeProfileFields(want)
		got := directTraces(td)
		expected, err := want.Marshal()
		require.NoError(t, err)
		actual, err := got.Marshal()
		require.NoError(t, err)
		if !bytes.Equal(expected, actual) {
			t.Fatalf("conversion differs: expected %x, got %x", expected, actual)
		}
	})
}

func normalizeProfileFields(td *tempopb.Trace) {
	var attributes func([]*common.KeyValue)
	var value func(*common.AnyValue)
	attributes = func(attrs []*common.KeyValue) {
		for _, a := range attrs {
			a.KeyStrindex = 0
			value(a.Value)
		}
	}
	value = func(v *common.AnyValue) {
		if v == nil {
			return
		}
		switch x := v.Value.(type) {
		case *common.AnyValue_StringValueStrindex:
			v.Value = nil
		case *common.AnyValue_KvlistValue:
			if x.KvlistValue != nil {
				attributes(x.KvlistValue.Values)
			}
		case *common.AnyValue_ArrayValue:
			if x.ArrayValue != nil {
				for _, v := range x.ArrayValue.Values {
					value(v)
				}
			}
		}
	}
	for _, rs := range td.ResourceSpans {
		attributes(rs.Resource.Attributes)
		for _, ss := range rs.ScopeSpans {
			attributes(ss.Scope.Attributes)
			for _, s := range ss.Spans {
				attributes(s.Attributes)
				for _, e := range s.Events {
					attributes(e.Attributes)
				}
				for _, l := range s.Links {
					attributes(l.Attributes)
				}
			}
		}
	}
}

func TestDirectTracesMultipleResourcesAndScopes(t *testing.T) {
	td := ptrace.NewTraces()
	for i := range 3 {
		source := conversionFixture(2, true)
		rs := td.ResourceSpans().AppendEmpty()
		source.ResourceSpans().At(0).CopyTo(rs)
		rs.SetSchemaUrl(fmt.Sprintf("resource-%d", i))
		rs.Resource().Attributes().PutStr("service.name", fmt.Sprintf("service-%d", i))
		for j := range 3 {
			ss := rs.ScopeSpans().AppendEmpty()
			source.ResourceSpans().At(0).ScopeSpans().At(0).CopyTo(ss)
			ss.SetSchemaUrl(fmt.Sprintf("scope-%d", j))
			ss.Scope().Attributes().PutStr("unicode", "東京\x00é")
			ss.Scope().Attributes().PutEmptyBytes("bytes")
			span := ss.Spans().At(0)
			span.SetKind(ptrace.SpanKind(-1))
			span.Status().SetCode(ptrace.StatusCode(-1))
			span.SetFlags(math.MaxUint32)
			span.SetStartTimestamp(pcommon.Timestamp(math.MaxUint64))
			span.Attributes().PutDouble("nan", math.NaN())
			span.Attributes().PutDouble("inf", math.Inf(1))
			span.Attributes().PutDouble("negative-zero", math.Copysign(0, -1))
		}
	}
	assertConversionParity(t, td)
}
