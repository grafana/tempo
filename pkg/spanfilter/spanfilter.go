package spanfilter

import (
	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/spanfilter/policymatch"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

type SpanFilter struct {
	filterPolicies []*filterPolicy
}

type filterPolicy struct {
	Include *splitPolicy
	Exclude *splitPolicy
}

// SplitPolicy is the result of parsing a policy from the config file to be
// specific about the area the given policy is applied to.
type splitPolicy struct {
	ResourceMatch  *policymatch.PolicyMatch
	SpanMatch      *policymatch.PolicyMatch
	IntrinsicMatch *policymatch.PolicyMatch
}

// Match returns true when the resource attributes and span attributes match the policy.
func (p *splitPolicy) Match(rs *v1.Resource, span *v1_trace.Span) bool {
	return p.ResourceMatch.MatchAttrs(rs.Attributes) &&
		p.SpanMatch.MatchAttrs(span.Attributes) &&
		p.IntrinsicMatch.MatchIntrinsicAttrs(span)
}

func NewSpanFilter(filterPolicies []config.FilterPolicy) (*SpanFilter, error) {
	var policies []*filterPolicy

	for _, policy := range filterPolicies {
		err := config.ValidateFilterPolicy(policy)
		if err != nil {
			return nil, err
		}

		include, err := getSplitPolicy(policy.Include)
		if err != nil {
			return nil, err
		}
		exclude, err := getSplitPolicy(policy.Exclude)
		if err != nil {
			return nil, err
		}
		p := &filterPolicy{
			Include: include,
			Exclude: exclude,
		}

		if p.Include != nil || p.Exclude != nil {
			policies = append(policies, p)
		}
	}

	return &SpanFilter{
		filterPolicies: policies,
	}, nil
}

// ApplyFilterPolicy returns true if the span should be included in the metrics.
func (f *SpanFilter) ApplyFilterPolicy(rs *v1.Resource, span *v1_trace.Span) bool {
	// With no filter policies specified, all spans are included.
	if len(f.filterPolicies) == 0 {
		return true
	}

	for _, policy := range f.filterPolicies {
		if policy.Include != nil {
			if !policy.Include.Match(rs, span) {
				return false
			}
		}

		if policy.Exclude != nil {
			if policy.Exclude.Match(rs, span) {
				return false
			}
		}
	}

	return true
}

func getSplitPolicy(policy *config.PolicyMatch) (*splitPolicy, error) {
	if policy == nil {
		return nil, nil
	}

	// A policy to match against the resource attributes
	resourcePolicy := &policymatch.PolicyMatch{
		Attributes: make([]policymatch.MatchPolicyAttribute, 0),
	}

	// A policy to match against the span attributes
	spanPolicy := &policymatch.PolicyMatch{
		Attributes: make([]policymatch.MatchPolicyAttribute, 0),
	}

	intrinsicPolicy := &policymatch.PolicyMatch{
		Attributes: make([]policymatch.MatchPolicyAttribute, 0),
	}

	for _, pa := range policy.Attributes {
		attr := traceql.MustParseIdentifier(pa.Key)

		if attr.Intrinsic > 0 {
			var (
				attribute policymatch.MatchPolicyAttribute
				err       error
			)
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
				attribute, err = policymatch.NewMatchRegexPolicyAttribute(attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
			}
			intrinsicPolicy.Attributes = append(intrinsicPolicy.Attributes, attribute)
		} else {
			switch attr.Scope {
			case traceql.AttributeScopeSpan:
				attribute, err := policymatch.NewMatchPolicyAttribute(policy.MatchType, attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
				spanPolicy.Attributes = append(spanPolicy.Attributes, attribute)
			case traceql.AttributeScopeResource:
				attribute, err := policymatch.NewMatchPolicyAttribute(policy.MatchType, attr.Name, pa.Value)
				if err != nil {
					return nil, err
				}
				resourcePolicy.Attributes = append(resourcePolicy.Attributes, attribute)
			}
		}
	}

	return &splitPolicy{
		ResourceMatch:  resourcePolicy,
		SpanMatch:      spanPolicy,
		IntrinsicMatch: intrinsicPolicy,
	}, nil
}
