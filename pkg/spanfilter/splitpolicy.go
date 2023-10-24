package spanfilter

import (
	"fmt"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/spanfilter/policymatch"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"golang.org/x/exp/maps"
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
			return nil, err
		}

		if attr.Intrinsic > 0 {
			var filter policymatch.IntrinsicFilter
			if policy.MatchType == config.Strict {
				switch attr.Intrinsic {
				case traceql.IntrinsicKind:
					switch v := pa.Value.(type) {
					case tracev1.Span_SpanKind:
						filter = policymatch.NewKindIntrinsicFilter(v)
					case string:
						if kind, ok := tracev1.Span_SpanKind_value[v]; ok {
							filter = policymatch.NewKindIntrinsicFilter(tracev1.Span_SpanKind(kind))
						} else {
							return nil, fmt.Errorf("currently unsupported kind intrinsic string value: %s; supported values: %v", v, maps.Keys(tracev1.Span_SpanKind_value))
						}
					default:
						return nil, fmt.Errorf("invalid kind intrinsic value: %v", v)
					}
				case traceql.IntrinsicStatus:
					switch v := pa.Value.(type) {
					case tracev1.Status_StatusCode:
						filter = policymatch.NewStatusIntrinsicFilter(v)
					case string:
						if code, ok := tracev1.Status_StatusCode_value[v]; ok {
							filter = policymatch.NewStatusIntrinsicFilter(tracev1.Status_StatusCode(code))
						} else {
							return nil, fmt.Errorf("currently unsupported status intrinsic string value: %s; supported values: %v", v, maps.Keys(tracev1.Status_StatusCode_value))
						}
					default:
						return nil, fmt.Errorf("currently unsupported intrinsic: %v", v)
					}
				case traceql.IntrinsicName:
					filter = policymatch.NewNameIntrinsicFilter(pa.Value.(string))
				}
			} else {
				filter, err = policymatch.NewRegexpIntrinsicFilter(attr.Intrinsic, pa.Value.(string))
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
