package traceql

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpansetClone(t *testing.T) {
	ss := []*Spanset{
		{
			Spans: []Span{
				&mockSpan{
					id:                 []byte{0x01},
					startTimeUnixNanos: 3,
					durationNanos:      2,
				},
			},
			Scalar:             NewStaticFloat(3.2),
			TraceID:            []byte{0x02},
			RootSpanName:       "a",
			RootServiceName:    "b",
			StartTimeUnixNanos: 1,
			DurationNanos:      5,
			Attributes: map[string]Static{
				"foo": NewStaticString("bar"),
			},
		},
		{
			Spans: []Span{
				&mockSpan{
					id:                 []byte{0x01},
					startTimeUnixNanos: 3,
					durationNanos:      2,
				},
			},
			Scalar:             NewStaticFloat(3.2),
			TraceID:            []byte{0x02},
			RootSpanName:       "a",
			RootServiceName:    "b",
			StartTimeUnixNanos: 1,
			DurationNanos:      5,
		},
	}

	for _, s := range ss {
		require.True(t, reflect.DeepEqual(s, s.clone()))
	}
}
