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
			Attributes:         []*SpansetAttribute{{Name: "foo", Val: NewStaticString("bar")}},
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

func TestMetaConditionsWithout(t *testing.T) {
	conditionsFor := func(q string) []Condition {
		req, err := ExtractFetchSpansRequest(q)
		require.NoError(t, err)

		return req.Conditions
	}

	tcs := []struct {
		remove []Condition
		expect []Condition
	}{
		{
			remove: []Condition{},
			expect: SearchMetaConditions(),
		},
		{
			remove: conditionsFor("{ duration > 1s}"),
			expect: []Condition{
				{NewIntrinsic(IntrinsicTraceRootService), OpNone, nil},
				{NewIntrinsic(IntrinsicTraceRootSpan), OpNone, nil},
				{NewIntrinsic(IntrinsicTraceDuration), OpNone, nil},
				{NewIntrinsic(IntrinsicTraceID), OpNone, nil},
				{NewIntrinsic(IntrinsicTraceStartTime), OpNone, nil},
				{NewIntrinsic(IntrinsicSpanID), OpNone, nil},
				{NewIntrinsic(IntrinsicSpanStartTime), OpNone, nil},
			},
		},
		{
			remove: conditionsFor("{ rootServiceName = `foo` && rootName = `bar`} | avg(duration) > 1s"),
			expect: []Condition{
				{NewIntrinsic(IntrinsicTraceDuration), OpNone, nil},
				{NewIntrinsic(IntrinsicTraceID), OpNone, nil},
				{NewIntrinsic(IntrinsicTraceStartTime), OpNone, nil},
				{NewIntrinsic(IntrinsicSpanID), OpNone, nil},
				{NewIntrinsic(IntrinsicSpanStartTime), OpNone, nil},
			},
		},
	}

	for _, tc := range tcs {
		require.Equal(t, tc.expect, SearchMetaConditionsWithout(tc.remove))
	}
}
