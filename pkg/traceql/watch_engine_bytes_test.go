package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

func spanWithAttr(key string, val Static) *mockSpan {
	s := newMockSpan(nil)
	s.attributes[NewScopedAttribute(AttributeScopeSpan, false, key)] = val
	return s
}

func TestEngineBytesWatcher(t *testing.T) {
	tests := []struct {
		name string
		span Span
		want int64
	}{
		{
			name: "start time 1 is 2 bytes",
			span: newMockSpan(nil).WithStartTime(1),
			want: 2,
		},
		{
			// type byte only
			name: "nil attribute is 1 byte",
			span: spanWithAttr("missing", NewStaticNil()),
			want: 1,
		},
		{
			name: "http.status_code=200 is 3 bytes",
			span: newMockSpan(nil).WithSpanInt("http.status_code", 200),
			want: 3,
		},
		{
			name: `service.name="test" is 6 bytes`,
			span: newMockSpan(nil).WithSpanString("service.name", "test"),
			want: 6,
		},
		{
			// type(1) + len(1) + varint(1)=1 + varint(200)=2 → 5
			name: "int array [1, 200] is 5 bytes",
			span: spanWithAttr("codes", NewStaticIntArray([]int{1, 200})),
			want: 5,
		},
		{
			// type(1) + len(1) + varint(len("test"))=1 + 4 → 7
			name: `string array ["test"] is 7 bytes`,
			span: spanWithAttr("tags", NewStaticStringArray([]string{"test"})),
			want: 7,
		},
		{
			// type(1) + len(1) + 8 → 10
			name: "float array [1.5] is 10 bytes",
			span: spanWithAttr("values", NewStaticFloatArray([]float64{1.5})),
			want: 10,
		},
		{
			// type(1) + len(1) + bool + bool → 4
			name: "bool array [true, false] is 4 bytes",
			span: spanWithAttr("flags", NewStaticBooleanArray([]bool{true, false})),
			want: 4,
		},
		{
			// type(1) + len(0)=1 → 2
			name: "empty int array is 2 bytes",
			span: spanWithAttr("empty", NewStaticIntArray([]int{})),
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewEngineBytesWatcher()
			require.True(t, o.WatchSpan(tt.span))
			require.Equal(t, tt.want, o.Stats()[tempopb.AdditionalMetricEngineBytes])
		})
	}
}
