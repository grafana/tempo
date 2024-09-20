package weights

import (
	"testing"

	"github.com/grafana/tempo/pkg/traceql"
)

func TestFetchSpans(t *testing.T) {
	cases := []struct {
		req      *traceql.FetchSpansRequest
		expected int
	}{
		{
			req:      nil,
			expected: 1,
		},
		{
			req:      &traceql.FetchSpansRequest{},
			expected: 2,
		},
		{
			req: &traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: 1,
		},
		{
			req: &traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: 2,
		},
		{
			req: &traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpRegex, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: 3,
		},
	}

	for _, c := range cases {
		actual := FetchSpans(c.req)
		if actual != c.expected {
			t.Errorf("expected %d, got %d", c.expected, actual)
		}
	}
}
