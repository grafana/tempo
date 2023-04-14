package spanfilter

import (
	"reflect"
	"regexp"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
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
	ResourceMatch  *config.PolicyMatch
	SpanMatch      *config.PolicyMatch
	IntrinsicMatch *config.PolicyMatch
}

func NewSpanFilter(filterPolicies []config.FilterPolicy) (*SpanFilter, error) {
	var policies []*filterPolicy

	var err error
	for _, policy := range filterPolicies {
		err = config.ValidateFilterPolicy(policy)
		if err != nil {
			return nil, err
		}

		p := &filterPolicy{
			Include: getSplitPolicy(policy.Include),
			Exclude: getSplitPolicy(policy.Exclude),
		}

		if p.Include != nil || p.Exclude != nil {
			policies = append(policies, p)
		}
	}

	return &SpanFilter{
		filterPolicies: policies,
	}, nil
}

// applyFilterPolicy returns true if the span should be included in the metrics.
func (f *SpanFilter) ApplyFilterPolicy(rs *v1.Resource, span *v1_trace.Span) bool {
	// With no filter policies specified, all spans are included.
	if len(f.filterPolicies) == 0 {
		return true
	}

	for _, policy := range f.filterPolicies {
		if policy.Include != nil {
			if !policyMatch(policy.Include, rs, span) {
				return false
			}
		}

		if policy.Exclude != nil {
			if policyMatch(policy.Exclude, rs, span) {
				return false
			}
		}
	}

	return true
}

func stringMatch(matchType config.MatchType, s, pattern string) bool {
	switch matchType {
	case config.Strict:
		return s == pattern
	case config.Regex:
		re := regexp.MustCompile(pattern)
		return re.MatchString(s)
	default:
		return false
	}
}

// policyMatch returns true when the resource attribtues and span attributes match the policy.
func policyMatch(policy *splitPolicy, rs *v1.Resource, span *v1_trace.Span) bool {
	return policyMatchAttrs(policy.ResourceMatch, rs.Attributes) &&
		policyMatchAttrs(policy.SpanMatch, span.Attributes) &&
		policyMatchIntrinsicAttrs(policy.IntrinsicMatch, span)
}

// policyMatchIntrinsicAttrs returns true when all intrinsic values in the policy match the span.
func policyMatchIntrinsicAttrs(policy *config.PolicyMatch, span *v1_trace.Span) bool {
	matches := 0
	for _, pa := range policy.Attributes {
		attr := traceql.MustParseIdentifier(pa.Key)
		switch attr.Intrinsic {
		// case traceql.IntrinsicDuration:
		// case traceql.IntrinsicChildCount:
		// case traceql.IntrinsicParent:
		case traceql.IntrinsicName:
			if stringMatch(policy.MatchType, span.GetName(), pa.Value.(string)) {
				matches++
			}
		case traceql.IntrinsicStatus:
			if stringMatch(policy.MatchType, span.GetStatus().GetCode().String(), pa.Value.(string)) {
				matches++
			}
		case traceql.IntrinsicKind:
			if stringMatch(policy.MatchType, span.GetKind().String(), pa.Value.(string)) {
				matches++
			}
		}
	}

	return len(policy.Attributes) == matches
}

// policyMatchAttrs returns true if all attributes in the policy match the attributes in the span.  String, bool, int, and floats are supported.  Regex MatchType may be applied to string span attributes.
func policyMatchAttrs(policy *config.PolicyMatch, attrs []*v1_common.KeyValue) bool {

	matches := 0
	var v *v1_common.AnyValue

	for _, pa := range policy.Attributes {
		pAttrValueType := reflect.TypeOf(pa.Value).String()

		for _, attr := range attrs {
			if attr.GetKey() == pa.Key {
				v = attr.GetValue()

				switch v.Value.(type) {
				case *v1_common.AnyValue_StringValue:
					if pAttrValueType != "string" {
						continue
					}

					if stringMatch(policy.MatchType, v.GetStringValue(), pa.Value.(string)) {
						matches++
					}
				case *v1_common.AnyValue_IntValue:
					if pAttrValueType != "int" {
						continue
					}

					if v.GetIntValue() == int64(pa.Value.(int)) {
						matches++
					}
				case *v1_common.AnyValue_DoubleValue:
					if pAttrValueType != "float64" {
						continue
					}

					if v.GetDoubleValue() == pa.Value.(float64) {
						matches++
					}
				case *v1_common.AnyValue_BoolValue:
					if pAttrValueType != "bool" {
						continue
					}

					if v.GetBoolValue() == pa.Value.(bool) {
						matches++
					}
				}
			}
		}
	}

	return len(policy.Attributes) == matches
}

func getSplitPolicy(policy *config.PolicyMatch) *splitPolicy {
	if policy == nil {
		return nil
	}

	// A policy to match against the resource attributes
	resourcePolicy := &config.PolicyMatch{
		MatchType:  policy.MatchType,
		Attributes: make([]config.MatchPolicyAttribute, 0),
	}

	// A policy to match against the span attributes
	spanPolicy := &config.PolicyMatch{
		MatchType:  policy.MatchType,
		Attributes: make([]config.MatchPolicyAttribute, 0),
	}

	intrinsicPolicy := &config.PolicyMatch{
		MatchType:  policy.MatchType,
		Attributes: make([]config.MatchPolicyAttribute, 0),
	}

	for _, pa := range policy.Attributes {
		attr := traceql.MustParseIdentifier(pa.Key)

		attribute := config.MatchPolicyAttribute{
			Key:   attr.Name,
			Value: pa.Value,
		}

		if attr.Intrinsic > 0 {
			intrinsicPolicy.Attributes = append(intrinsicPolicy.Attributes, attribute)
		} else {
			switch attr.Scope {
			case traceql.AttributeScopeSpan:
				spanPolicy.Attributes = append(spanPolicy.Attributes, attribute)
			case traceql.AttributeScopeResource:
				resourcePolicy.Attributes = append(resourcePolicy.Attributes, attribute)
			}
		}
	}

	return &splitPolicy{
		ResourceMatch:  resourcePolicy,
		SpanMatch:      spanPolicy,
		IntrinsicMatch: intrinsicPolicy,
	}
}
