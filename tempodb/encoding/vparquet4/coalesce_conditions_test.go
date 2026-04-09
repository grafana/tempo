package vparquet4

import (
	"fmt"
	"testing"

	"github.com/grafana/tempo/v2/pkg/traceql"
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
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			coalesceConditions(tc.f)
			require.Equal(t, tc.expected, tc.f)
		})
	}
}
