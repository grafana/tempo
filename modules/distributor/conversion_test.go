package distributor

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/entity"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
)

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
