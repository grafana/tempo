package vparquet3

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
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNone, nil},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("bar")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("bar")}},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicKind), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicKind), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpGreater, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNone, nil},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpGreater, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNone, nil},
				},
			},
		},
		{
			f: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("foo")}},
					{traceql.NewIntrinsic(traceql.IntrinsicChildCount), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("bar")}},
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNotEqual, []traceql.Static{traceql.NewStaticString("foo")}},
				},
			},
			expected: &traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{
					{traceql.NewIntrinsic(traceql.IntrinsicTraceRootService), traceql.OpNone, nil},
					{traceql.NewIntrinsic(traceql.IntrinsicChildCount), traceql.OpEqual, []traceql.Static{traceql.NewStaticString("bar")}},
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
