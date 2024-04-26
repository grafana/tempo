package spanfilter

import (
	"fmt"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/spanfilter/policymatch"
	v1 "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

// splitPolicy is the result of parsing a policy from the config file to be
// specific about the area the given policy is applied to.
type splitPolicy struct {
	// ResourceMatch is a set of resource attributes that must match a span for the span to match the policy.
	ResourceMatch *policymatch.AttributePolicyMatch
	// SpanMatch is a set of span attributes that must match a span for the span to match the policy.
	SpanMatch *policymatch.AttributePolicyMatch
	// IntrinsicMatch is a set of intrinsic attributes that must match a span for the span to match the policy.
	IntrinsicMatch *policymatch.IntrinsicPolicyMatch
}

func newSplitPolicy(policy *config.PolicyMatch) (*splitPolicy, error) {
	var resourceAttributeFilters []policymatch.AttributeFilter
	var spanAttributeFilters []policymatch.AttributeFilter
	var intrinsicFilters []policymatch.IntrinsicFilter

	for _, pa := range policy.Attributes {
		attr, err := traceql.ParseIdentifier(pa.Key)
		if err != nil {
			return nil, fmt.Errorf("invalid policy match attribute: %v", err)
		}

		if attr.Intrinsic > 0 {
			var filter policymatch.IntrinsicFilter
			if policy.MatchType == config.Strict {
				filter, err = policymatch.NewStrictIntrinsicFilter(attr.Intrinsic, pa.Value)
				if err != nil {
					return nil, err
				}
			} else {
				filter, err = policymatch.NewRegexpIntrinsicFilter(attr.Intrinsic, pa.Value)
				if err != nil {
					return nil, err
				}
			}
			intrinsicFilters = append(intrinsicFilters, filter)
		} else {
			switch attr.Scope {
			case traceql.AttributeScopeSpan:
				filter, err := policymatch.NewAttributeFilter(policy.MatchType, attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
				spanAttributeFilters = append(spanAttributeFilters, filter)
			case traceql.AttributeScopeResource:
				filter, err := policymatch.NewAttributeFilter(policy.MatchType, attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
				resourceAttributeFilters = append(resourceAttributeFilters, filter)
			default:
				return nil, fmt.Errorf("invalid or unsupported attribute scope: %v", attr.Scope)
			}
		}
	}

	sp := splitPolicy{}
	if len(resourceAttributeFilters) > 0 {
		sp.ResourceMatch = policymatch.NewAttributePolicyMatch(resourceAttributeFilters)
	}

	if len(intrinsicFilters) > 0 {
		sp.IntrinsicMatch = policymatch.NewIntrinsicPolicyMatch(intrinsicFilters)
	}

	if len(spanAttributeFilters) > 0 {
		sp.SpanMatch = policymatch.NewAttributePolicyMatch(spanAttributeFilters)
	}

	return &sp, nil
}

// Match returns true when the resource attributes and span attributes match the policy.
func (p *splitPolicy) Match(rs *v1.Resource, span *tracev1.Span) bool {
	return (p.ResourceMatch == nil || p.ResourceMatch.Matches(rs.Attributes)) &&
		(p.SpanMatch == nil || p.SpanMatch.Matches(span.Attributes)) &&
		(p.IntrinsicMatch == nil || p.IntrinsicMatch.Matches(span))
}
