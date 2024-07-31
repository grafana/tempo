package spanfilter

import (
	"errors"
	"testing"

	"github.com/grafana/tempo/v2/pkg/traceql"

	"github.com/grafana/tempo/v2/pkg/spanfilter/policymatch"
	tracev1 "github.com/grafana/tempo/v2/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v2/pkg/spanfilter/config"
)

func Test_newSplitPolicy(t *testing.T) {
	cases := []struct {
		policy *config.PolicyMatch
		split  *splitPolicy
		name   string
		err    error
	}{
		{
			name: "strict kind matching",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: "SPAN_KIND_CLIENT",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewIntrinsicPolicyMatch(
					[]policymatch.IntrinsicFilter{
						policymatch.NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_CLIENT),
					}),
			},
		},
		{
			name: "strict status matching",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "status",
						Value: "STATUS_CODE_OK",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewIntrinsicPolicyMatch(
					[]policymatch.IntrinsicFilter{
						policymatch.NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					}),
			},
		},
		{
			name: "strict name matching",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "name",
						Value: "foo",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewIntrinsicPolicyMatch(
					[]policymatch.IntrinsicFilter{
						policymatch.NewNameIntrinsicFilter("foo"),
					},
				),
			},
		},
		{
			name: "regex name matching",
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "name",
						Value: "foo.*",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewIntrinsicPolicyMatch(
					[]policymatch.IntrinsicFilter{
						must(policymatch.NewRegexpIntrinsicFilter(traceql.IntrinsicName, "foo.*")),
					},
				),
			},
		},
		{
			name: "regex kind matching",
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: ".*_CLIENT",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewIntrinsicPolicyMatch(
					[]policymatch.IntrinsicFilter{
						must(policymatch.NewRegexpIntrinsicFilter(traceql.IntrinsicKind, ".*_CLIENT")),
					},
				),
			},
		},
		{
			name: "regex status matching",
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "status",
						Value: ".*_OK",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewIntrinsicPolicyMatch(
					[]policymatch.IntrinsicFilter{
						must(policymatch.NewRegexpIntrinsicFilter(traceql.IntrinsicStatus, ".*_OK")),
					},
				),
			},
		},
		{
			name: "strict resource attribute matching",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "resource.foo",
						Value: "bar",
					},
				},
			},
			split: &splitPolicy{
				ResourceMatch: policymatch.NewAttributePolicyMatch(
					[]policymatch.AttributeFilter{
						must(policymatch.NewStrictAttributeFilter("foo", "bar")),
					},
				),
			},
		},
		{
			name: "regex resource attribute matching",
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "resource.foo",
						Value: ".*",
					},
				},
			},
			split: &splitPolicy{
				ResourceMatch: policymatch.NewAttributePolicyMatch(
					[]policymatch.AttributeFilter{
						must(policymatch.NewRegexpAttributeFilter("foo", ".*")),
					},
				),
			},
		},
		{
			name: "strict span attribute matching",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "span.foo",
						Value: "bar",
					},
				},
			},
			split: &splitPolicy{
				SpanMatch: policymatch.NewAttributePolicyMatch(
					[]policymatch.AttributeFilter{
						must(policymatch.NewStrictAttributeFilter("foo", "bar")),
					},
				),
			},
		},
		{
			name: "regex span attribute matching",
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "span.foo",
						Value: ".*",
					},
				},
			},
			split: &splitPolicy{
				SpanMatch: policymatch.NewAttributePolicyMatch(
					[]policymatch.AttributeFilter{
						must(policymatch.NewRegexpAttributeFilter("foo", ".*")),
					},
				),
			},
		},
		{
			name: "invalid regex span attribute matching",
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "span.foo",
						Value: ".*(",
					},
				},
			},
			err: errors.New("invalid attribute filter regexp: error parsing regexp: missing closing ): `.*(`"),
		},
		{
			name: "invalid regex kind intrinsic matching",
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: ".*(",
					},
				},
			},
			err: errors.New("invalid intrinsic filter regex: error parsing regexp: missing closing ): `.*(`"),
		},
		{
			name: "invalid intrinsic",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "unsupported",
						Value: "foo",
					},
				},
			},
			err: errors.New("invalid policy match attribute: tag name is not valid intrinsic or scoped attribute: unsupported"),
		},
		{
			name: "unsupported kind intrinsic string value",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: "foo",
					},
				},
			},
			err: errors.New("unsupported kind intrinsic string value: foo"),
		},
		{
			name: "unsupported status intrinsic string value",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "status",
						Value: "foo",
					},
				},
			},
			err: errors.New("unsupported status intrinsic string value: foo"),
		},
		{
			name: "unsupported status intrinsic value",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "status",
						Value: true,
					},
				},
			},
			err: errors.New("unsupported status intrinsic value: true"),
		},
		{
			name: "unsupported kind intrinsic value",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: true,
					},
				},
			},
			err: errors.New("invalid kind intrinsic value: true"),
		},
		{
			name: "unsupported name intrinsic value",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "name",
						Value: true,
					},
				},
			},
			err: errors.New("unsupported name intrinsic value: true"),
		},
		{
			name: "unsupported intrinsic",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "childCount",
						Value: "foo",
					},
				},
			},
			err: errors.New("unsupported intrinsic: childCount"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := newSplitPolicy(tc.policy)
			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, s)

			if tc.split.IntrinsicMatch != nil {
				require.Equal(t, tc.split.IntrinsicMatch, s.IntrinsicMatch)
			}
			if tc.split.SpanMatch != nil {
				require.Equal(t, tc.split.SpanMatch, s.SpanMatch)
			}
			if tc.split.ResourceMatch != nil {
				require.Equal(t, tc.split.ResourceMatch, s.ResourceMatch)
			}
		})
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
