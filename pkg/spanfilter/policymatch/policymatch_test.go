package policymatch

import (
	"regexp"
	"testing"

	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("kind", trace_v1.Span_SPAN_KIND_SERVER),
					NewMatchStrictPolicyAttribute("status", trace_v1.Status_STATUS_CODE_OK),
					NewMatchStrictPolicyAttribute("name", "test"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("kind", trace_v1.Span_SPAN_KIND_SERVER),
					NewMatchStrictPolicyAttribute("status", trace_v1.Status_STATUS_CODE_OK),
					NewMatchStrictPolicyAttribute("name", "test"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("kind", trace_v1.Span_SPAN_KIND_CLIENT),
					NewMatchStrictPolicyAttribute("status", trace_v1.Status_STATUS_CODE_OK),
					NewMatchStrictPolicyAttribute("name", "test"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("kind", trace_v1.Span_SPAN_KIND_SERVER),
					NewMatchStrictPolicyAttribute("status", trace_v1.Status_STATUS_CODE_OK),
					NewMatchStrictPolicyAttribute("name", "test"),
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
				attributes: []MatchPolicyAttribute{
					must(NewMatchRegexPolicyAttribute("kind", ".*_KIND_.*")),
					must(NewMatchRegexPolicyAttribute("status", ".*_CODE_.*")),
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
				attributes: []MatchPolicyAttribute{
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
				attributes: []MatchPolicyAttribute{
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("foo", "bar"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("foo", "bar"),
					NewMatchStrictPolicyAttribute("otherfoo", "notbar"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("foo", "bar"),
					NewMatchStrictPolicyAttribute("otherfoo", "nope"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("one", "1"),
					NewMatchStrictPolicyAttribute("oneone", 11),
					NewMatchStrictPolicyAttribute("oneonepointone", 11.1),
					NewMatchStrictPolicyAttribute("matching", true),
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
				attributes: []MatchPolicyAttribute{
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("dd", true),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("dd", "value"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("11", "eleven"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("11", "eleven"),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("11", 11),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("11", 11),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("11", true),
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
				attributes: []MatchPolicyAttribute{
					NewMatchStrictPolicyAttribute("11", 11.0),
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
		a := NewMatchStrictPolicyAttribute("server", tc.pattern)
		r := a.MatchString(tc.s)
		require.Equal(t, tc.expect, r)
	}
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
