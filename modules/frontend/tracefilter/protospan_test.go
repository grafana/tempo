package tracefilter

import (
	"testing"

	"github.com/stretchr/testify/require"

	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

func TestStaticFromArray(t *testing.T) {
	str := func(s string) *commonv1.AnyValue {
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: s}}
	}
	i := func(n int64) *commonv1.AnyValue {
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: n}}
	}
	d := func(f float64) *commonv1.AnyValue {
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: f}}
	}
	b := func(v bool) *commonv1.AnyValue {
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: v}}
	}

	cases := []struct {
		name string
		vals []*commonv1.AnyValue
		want traceql.Static
	}{
		{"nil array", nil, traceql.NewStaticNil()},
		{"string array", []*commonv1.AnyValue{str("a"), str("b")}, traceql.NewStaticStringArray([]string{"a", "b"})},
		{"int array", []*commonv1.AnyValue{i(1), i(2)}, traceql.NewStaticIntArray([]int{1, 2})},
		{"float array", []*commonv1.AnyValue{d(1.5), d(2.5)}, traceql.NewStaticFloatArray([]float64{1.5, 2.5})},
		{"bool array", []*commonv1.AnyValue{b(true), b(false)}, traceql.NewStaticBooleanArray([]bool{true, false})},
		// single-element arrays collapse to a scalar, matching vp5's read-side attributeCollector.
		{"single string collapses", []*commonv1.AnyValue{str("only")}, traceql.NewStaticString("only")},
		{"single int collapses", []*commonv1.AnyValue{i(7)}, traceql.NewStaticInt(7)},
		// a mixed-type array is unsupported in vp5 storage, we surface nil.
		{"mixed type is nil", []*commonv1.AnyValue{str("a"), i(1)}, traceql.NewStaticNil()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, staticFromArray(&commonv1.ArrayValue{Values: tc.vals}))
		})
	}
}

func TestExpandSpanBindingsCapsFanout(t *testing.T) {
	// events x links beyond maxBindingsPerSpan must truncate, not allocate unbounded or panic.
	span := &tracev1.Span{SpanId: []byte{1}}
	for range 1000 {
		span.Events = append(span.Events, &tracev1.Span_Event{})
	}
	for range 101 {
		span.Links = append(span.Links, &tracev1.Span_Link{})
	}

	got, truncated := expandSpanBindings(nil, span, nil, nil, true)
	require.Len(t, got, maxBindingsPerSpan)
	require.True(t, truncated, "fan-out beyond the cap must report truncation")
}

func TestExpandSpanBindingsSkipsFanoutWithoutElementFilter(t *testing.T) {
	// a filter that reads no event/link scope gets one binding per span, not the event x link product.
	span := &tracev1.Span{SpanId: []byte{1}}
	span.Events = append(span.Events, &tracev1.Span_Event{}, &tracev1.Span_Event{})
	span.Links = append(span.Links, &tracev1.Span_Link{})

	got, truncated := expandSpanBindings(nil, span, nil, nil, false)
	require.Len(t, got, 1)
	require.False(t, truncated)
}
