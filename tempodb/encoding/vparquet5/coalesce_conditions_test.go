package vparquet5

import (
	"fmt"
	"testing"

	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/stretchr/testify/require"
)

func TestCoalesceConditions(t *testing.T) {
	tcs := []struct {
		f        *traceql.FetchSpansRequest
		expected *traceql.FetchSpansRequest
	}{
		{
			f:        &traceql.FetchSpansRequest{},
			expected: &traceql.FetchSpansRequest{},
		},
		{
			f: &traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNone, Operands: nil},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("bar")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("bar")}},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicKind), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicKind), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpGreater, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNone, Operands: nil},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpGreater, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNone, Operands: nil},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicChildCount), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("bar")}},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNotEqual, Operands: []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), Op: traceql.OpNone, Operands: nil},
					{Attribute: traceql.NewIntrinsic(traceql.IntrinsicChildCount), Op: traceql.OpEqual, Operands: []traceql.Static{traceql.NewStaticString("bar")}},
				},
			},
		},
		{
			// OpNone only pulls spans that have the attribute, so it can't replace OpNotExists
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone},
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNotExists},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone},
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNotExists},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNotExists},
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNotExists},
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone},
				},
			},
		},
		{
			// OpNone conditions of the same type fetch the same columns
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone, Operands: []traceql.Static{traceql.NewStaticString("a")}},
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone, Operands: []traceql.Static{traceql.NewStaticString("b")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone, Operands: []traceql.Static{traceql.NewStaticString("a")}},
				},
			},
		},
		{
			// OpNone conditions of different types need the columns of both, so every type is fetched
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone, Operands: []traceql.Static{traceql.NewStaticString("a")}},
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone, Operands: []traceql.Static{traceql.NewStaticInt(1)}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone},
				},
			},
		},
		{
			// OpNone without operands fetches every type
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone, Operands: []traceql.Static{traceql.NewStaticString("a")}},
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{Attribute: traceql.NewAttribute("foo"), Op: traceql.OpNone},
				},
			},
		},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			coalesceConditions(tc.f)
			require.Equal(t, tc.expected, tc.f)
		})
	}
}
