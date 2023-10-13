package spanfilter

import (
	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/spanfilter/policymatch"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

// splitPolicy is the result of parsing a policy from the config file to be
// specific about the area the given policy is applied to.
type splitPolicy struct {
	ResourceMatch  *policymatch.PolicyMatch
	SpanMatch      *policymatch.PolicyMatch
	IntrinsicMatch *policymatch.PolicyMatch
}

func newSplitPolicy(policy *config.PolicyMatch) (*splitPolicy, error) {
	// A policy to match against the resource attributes
	resourcePolicy := policymatch.NewPolicyMatch()

	// A policy to match against the span attributes
	spanPolicy := policymatch.NewPolicyMatch()

	// A policy to match against the span intrinsic attributes
	intrinsicPolicy := policymatch.NewPolicyMatch()

	for _, pa := range policy.Attributes {
		attr := traceql.MustParseIdentifier(pa.Key)

		if attr.Intrinsic > 0 {
			var attribute policymatch.MatchPolicyAttribute
			if policy.MatchType == config.Strict {
				switch attr.Intrinsic {
				case traceql.IntrinsicStatus:
					value := v1_trace.Status_StatusCode(v1_trace.Status_StatusCode_value[pa.Value.(string)])
					attribute = policymatch.NewMatchStrictPolicyAttribute(attr.Name, value)
				case traceql.IntrinsicKind:
					value := v1_trace.Span_SpanKind(v1_trace.Span_SpanKind_value[pa.Value.(string)])
					attribute = policymatch.NewMatchStrictPolicyAttribute(attr.Name, value)
				default:
					attribute = policymatch.NewMatchStrictPolicyAttribute(attr.Name, pa.Value)
				}
			} else {
				var err error
				attribute, err = policymatch.NewMatchRegexPolicyAttribute(attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
			}
			intrinsicPolicy.AddAttr(attribute)
		} else {
			switch attr.Scope {
			case traceql.AttributeScopeSpan:
				attribute, err := policymatch.NewMatchPolicyAttribute(policy.MatchType, attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
				spanPolicy.AddAttr(attribute)
			case traceql.AttributeScopeResource:
				attribute, err := policymatch.NewMatchPolicyAttribute(policy.MatchType, attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
				resourcePolicy.AddAttr(attribute)
			}
		}
	}

	return &splitPolicy{
		ResourceMatch:  resourcePolicy,
		SpanMatch:      spanPolicy,
		IntrinsicMatch: intrinsicPolicy,
	}, nil
}

// Match returns true when the resource attributes and span attributes match the policy.
func (p *splitPolicy) Match(rs *v1.Resource, span *v1_trace.Span) bool {
	return p.IntrinsicMatch.MatchIntrinsicAttrs(span) &&
		p.ResourceMatch.MatchAttrs(rs.Attributes) &&
		p.SpanMatch.MatchAttrs(span.Attributes)
}
