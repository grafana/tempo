package policymatch

import (
	"testing"

	tracev1 "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

func TestIntrinsicPolicyMatch_Matches(t *testing.T) {
	cases := []struct {
		policy *IntrinsicPolicyMatch
		span   *tracev1.Span
		expect bool
		name   string
	}{
		{
			name:   "match on name, kind and status",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched name",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test2",
			},
		},
		{
			name:   "unmatched status",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_CLIENT,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_ERROR,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched kind",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_CLIENT,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "matched regex kind and status",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, ".*_KIND_.*")),
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicStatus, ".*_CODE_.*")),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched regex kind",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, ".*_CLIENT")),
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicStatus, ".*_OK")),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched regex status",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, ".*_SERVER")),
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicStatus, ".*_ERROR")),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.policy.Matches(tc.span)
			require.Equal(t, tc.expect, r)
		})
	}
}
