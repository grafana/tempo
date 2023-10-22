package policymatch

import (
	"testing"

	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

func Test_strictAttributeFilter_Matches(t *testing.T) {
	cases := []struct {
		policy *AttributePolicyMatch
		attrs  []*commonv1.KeyValue
		expect bool
		name   string
	}{
		{
			name:   "single string match",
			expect: true,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("foo", "bar"),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "foo",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
			},
		},
		{
			name:   "multiple string match",
			expect: true,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("foo", "bar"),
					NewStrictAttributeFilter("otherfoo", "notbar"),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "foo",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
				{
					Key: "otherfoo",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "notbar",
						},
					},
				},
			},
		},
		{
			name:   "multiple string non match",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("foo", "bar"),
					NewStrictAttributeFilter("otherfoo", "nope"),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "foo",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
				{
					Key: "otherfoo",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "notbar",
						},
					},
				},
			},
		},
		{
			name:   "combination match",
			expect: true,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("one", "1"),
					NewStrictAttributeFilter("oneone", 11),
					NewStrictAttributeFilter("oneonepointone", 11.1),
					NewStrictAttributeFilter("matching", true),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "one",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "1",
						},
					},
				},
				{
					Key: "oneone",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_IntValue{
							IntValue: 11,
						},
					},
				},
				{
					Key: "oneonepointone",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_DoubleValue{
							DoubleValue: 11.1,
						},
					},
				},
				{
					Key: "matching",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_BoolValue{
							BoolValue: true,
						},
					},
				},
			},
		},
		{
			name:   "regex basic match",
			expect: true,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					must(NewRegexpAttributeFilter("dd", `\d\d\w{5}`)),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "dd",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "11xxxxx",
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("dd", true),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "dd",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "11xxxxx",
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string/int",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("dd", "value"),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "dd",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_IntValue{
							IntValue: 11,
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string/float",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("11", "eleven"),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "11",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_DoubleValue{
							DoubleValue: 11.1,
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch string/bool",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("11", "eleven"),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "11",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_BoolValue{
							BoolValue: false,
						},
					},
				},
			},
		},
		{
			name:   "value type mismatch int/string",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("11", 11),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "11",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_StringValue{
							StringValue: "11",
						},
					},
				},
			},
		},
		{
			name:   "value mismatch int",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("11", 11),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "11",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_IntValue{
							IntValue: 12,
						},
					},
				},
			},
		},
		{
			name:   "value mismatch bool",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("11", true),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "11",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_BoolValue{
							BoolValue: false,
						},
					},
				},
			},
		},
		{
			name:   "value mismatch float",
			expect: false,
			policy: &AttributePolicyMatch{
				filters: []AttributeFilter{
					NewStrictAttributeFilter("11", 11.0),
				},
			},
			attrs: []*commonv1.KeyValue{
				{
					Key: "11",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_DoubleValue{
							DoubleValue: 11.1,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.policy.Matches(tc.attrs)
			require.Equal(t, tc.expect, r)
		})
	}
}

func Test_regexpAttributeFilter_Matches(t *testing.T) {
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
		a := NewAttributePolicyMatch([]AttributeFilter{must(NewRegexpAttributeFilter("server", tc.pattern))})
		r := a.Matches([]*commonv1.KeyValue{
			{
				Key: "server",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: tc.s,
					},
				},
			},
		})
		require.Equal(t, tc.expect, r)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
