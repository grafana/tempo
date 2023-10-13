package policymatch

import (
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

func TestPolicyMatch_MatchIntrinsicAttrs(t *testing.T) {
	cases := []struct {
		policy *PolicyMatch
		span   *trace_v1.Span
		expect bool
		name   string
	}{
		{
			name:   "match on name, kind and status",
			expect: true,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "kind",
						value: trace_v1.Span_SPAN_KIND_SERVER,
					},
					matchStrictPolicyAttribute{
						key:   "status",
						value: trace_v1.Status_STATUS_CODE_OK,
					},
					matchStrictPolicyAttribute{
						key:   "name",
						value: "test",
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched name",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "kind",
						value: trace_v1.Span_SPAN_KIND_SERVER,
					},
					matchStrictPolicyAttribute{
						key:   "status",
						value: trace_v1.Status_STATUS_CODE_OK,
					},
					matchStrictPolicyAttribute{
						key:   "name",
						value: "test",
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test2",
			},
		},
		{
			name:   "unmatched status",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "kind",
						value: trace_v1.Span_SPAN_KIND_CLIENT,
					},
					matchStrictPolicyAttribute{
						key:   "status",
						value: trace_v1.Status_STATUS_CODE_OK,
					},
					matchStrictPolicyAttribute{
						key:   "name",
						value: "test",
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_CLIENT,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_ERROR,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched kind",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "kind",
						value: trace_v1.Span_SPAN_KIND_SERVER,
					},
					matchStrictPolicyAttribute{
						key:   "status",
						value: trace_v1.Status_STATUS_CODE_OK,
					},
					matchStrictPolicyAttribute{
						key:   "name",
						value: "test",
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_CLIENT,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "matched regex kind and status",
			expect: true,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchRegexPolicyAttribute{
						key:   "kind",
						value: regexp.MustCompile(".*_KIND_.*"),
					},
					matchRegexPolicyAttribute{
						key:   "status",
						value: regexp.MustCompile(".*_CODE_.*"),
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched regex kind",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchRegexPolicyAttribute{
						key:   "kind",
						value: regexp.MustCompile(".*_CLIENT"),
					},
					matchRegexPolicyAttribute{
						key:   "status",
						value: regexp.MustCompile(".*_OK"),
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched regex status",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchRegexPolicyAttribute{
						key:   "kind",
						value: regexp.MustCompile(".*_SERVER"),
					},
					matchRegexPolicyAttribute{
						key:   "status",
						value: regexp.MustCompile(".*_ERROR"),
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.policy.MatchIntrinsicAttrs(tc.span)
			require.Equal(t, tc.expect, r)
		})
	}
}

func TestSpanFilter_policyMatchAttrs(t *testing.T) {
	cases := []struct {
		policy *PolicyMatch
		attrs  []*common_v1.KeyValue
		expect bool
		name   string
	}{
		{
			name:   "single string match",
			expect: true,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "foo",
						value: "bar",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "foo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
			},
		},
		{
			name:   "multiple string match",
			expect: true,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "foo",
						value: "bar",
					},
					matchStrictPolicyAttribute{
						key:   "otherfoo",
						value: "notbar",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "foo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
				{
					Key: "otherfoo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "notbar",
						},
					},
				},
			},
		},
		{
			name:   "multiple string non match",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "foo",
						value: "bar",
					},
					matchStrictPolicyAttribute{
						key:   "otherfoo",
						value: "nope",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "foo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
				{
					Key: "otherfoo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "notbar",
						},
					},
				},
			},
		},
		{
			name:   "combination match",
			expect: true,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "one",
						value: "1",
					},
					matchStrictPolicyAttribute{
						key:   "oneone",
						value: 11,
					},
					matchStrictPolicyAttribute{
						key:   "oneonepointone",
						value: 11.1,
					},
					matchStrictPolicyAttribute{
						key:   "matching",
						value: true,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "one",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "1",
						},
					},
				},
				{
					Key: "oneone",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_IntValue{
							IntValue: 11,
						},
					},
				},
				{
					Key: "oneonepointone",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_DoubleValue{
							DoubleValue: 11.1,
						},
					},
				},
				{
					Key: "matching",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_BoolValue{
							BoolValue: true,
						},
					},
				},
			},
		},
		{
			name:   "regex basic match",
			expect: true,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchRegexPolicyAttribute{
						key:   "dd",
						value: regexp.MustCompile(`\d\d\w{5}`),
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "dd",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "11xxxxx",
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "dd",
						value: true,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "dd",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "11xxxxx",
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string/int",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "dd",
						value: "value",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "dd",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_IntValue{
							IntValue: 11,
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string/float",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "11",
						value: "eleven",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "11",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_DoubleValue{
							DoubleValue: 11.1,
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string/bool",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "11",
						value: "eleven",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "11",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_BoolValue{
							BoolValue: false,
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch int/string",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "11",
						value: 11,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "11",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "11",
						},
					},
				},
			},
		},
		{
			name:   "value mismatch int",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "11",
						value: 11,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "11",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_IntValue{
							IntValue: 12,
						},
					},
				},
			},
		},
		{
			name:   "value mismatch bool",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "11",
						value: true,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "11",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_BoolValue{
							BoolValue: false,
						},
					},
				},
			},
		},
		{
			name:   "value mismatch float",
			expect: false,
			policy: &PolicyMatch{
				Attributes: []MatchPolicyAttribute{
					matchStrictPolicyAttribute{
						key:   "11",
						value: 11.0,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "11",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_DoubleValue{
							DoubleValue: 11.1,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.policy.MatchAttrs(tc.attrs)
			require.Equal(t, tc.expect, r)
		})
	}
}

func TestSpanFilter_stringMatch(t *testing.T) {

}

func Test_matchStrictPolicyAttribute_Match(t *testing.T) {
	cases := []struct {
		s       string
		pattern string
		expect  bool
	}{
		{
			s:       "foo",
			pattern: "foo",
			expect:  true,
		},
		{
			s:       "foo",
			pattern: "bar",
			expect:  false,
		},
	}

	for _, tc := range cases {
		a := matchStrictPolicyAttribute{key: "server", value: tc.pattern}
		r := a.Match(tc.s)
		require.Equal(t, tc.expect, r)
	}
}
