package policymatch

import (
	"fmt"
	"github.com/grafana/tempo/pkg/spanfilter/config"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"regexp"
)

type PolicyMatch struct {
	Attributes []MatchPolicyAttribute
}

// MatchAttrs returns true if all attributes in the policy match the attributes in the span.  String, bool, int, and floats are supported.  Regex MatchType may be applied to string span attributes.
func (p *PolicyMatch) MatchAttrs(attrs []*v1_common.KeyValue) bool {
	matches := 0
	for _, pa := range p.Attributes {

		for _, attr := range attrs {
			if attr.GetKey() == pa.Key() {
				v := attr.GetValue()

				// For each type of value, check if the policy attribute value matches the span attribute value.
				switch v.Value.(type) {
				case *v1_common.AnyValue_StringValue:
					if !pa.Match(v.GetStringValue()) {
						return false
					}
					matches++
				case *v1_common.AnyValue_IntValue:
					if !pa.Match(v.GetIntValue()) {
						return false
					}
					matches++
				case *v1_common.AnyValue_DoubleValue:
					if !pa.Match(v.GetDoubleValue()) {
						return false
					}

					matches++
				case *v1_common.AnyValue_BoolValue:
					if !pa.Match(v.GetBoolValue()) {
						return false
					}
					matches++
				}
			}
		}
	}

	return len(p.Attributes) == matches
}

// MatchIntrinsicAttrs returns true when all intrinsic values in the policy match the span.
func (p *PolicyMatch) MatchIntrinsicAttrs(span *v1_trace.Span) bool {
	matches := 0

	for _, pa := range p.Attributes {
		attr := traceql.MustParseIdentifier(pa.Key())
		switch attr.Intrinsic {
		// case traceql.IntrinsicDuration:
		// case traceql.IntrinsicChildCount:
		// case traceql.IntrinsicParent:
		case traceql.IntrinsicName:
			if !pa.Match(span.GetName()) {
				return false
			}
			matches++
		case traceql.IntrinsicStatus:
			if !pa.Match(span.GetStatus().GetCode()) {
				return false
			}
			matches++
		case traceql.IntrinsicKind:
			if !pa.Match(span.GetKind()) {
				return false
			}
			matches++
		}
	}

	return len(p.Attributes) == matches
}

type MatchPolicyAttribute interface {
	Key() string
	Match(value interface{}) bool
}

func NewMatchPolicyAttribute(matchType config.MatchType, key string, value interface{}) (MatchPolicyAttribute, error) {
	if matchType == config.Regex {
		return NewMatchRegexPolicyAttribute(key, value)
	}
	return NewMatchStrictPolicyAttribute(key, value), nil
}

type matchStrictPolicyAttribute struct {
	key   string
	value interface{}
}

func NewMatchStrictPolicyAttribute(key string, value interface{}) MatchPolicyAttribute {
	return matchStrictPolicyAttribute{key: key, value: value}
}

// Match returns true when the resource attributes and span attributes match the policy.
func (a matchStrictPolicyAttribute) Match(value interface{}) bool {
	switch val := value.(type) {
	case nil:
		return false
	case v1_trace.Span_SpanKind:
		switch attrValue := a.value.(type) {
		case v1_trace.Span_SpanKind:
			return attrValue == val
		case string:
			return attrValue == val.String()
		default:
			return false
		}
	case v1_trace.Status_StatusCode:
		switch attrValue := a.value.(type) {
		case v1_trace.Status_StatusCode:
			return attrValue == val
		case string:
			return attrValue == val.String()
		default:
			return false
		}
	case string:
		switch attrValue := a.value.(type) {
		case fmt.Stringer:
			return attrValue.String() == val
		case string:
			return attrValue == val
		default:
			return false
		}
	case int:
		switch attrValue := a.value.(type) {
		case int64:
			return attrValue == int64(val)
		case int:
			return attrValue == val
		default:
			return false
		}
	case int64:
		switch attrValue := a.value.(type) {
		case int64:
			return attrValue == val
		case int:
			return int64(attrValue) == val
		default:
			return false
		}
	case float64:
		switch attrValue := a.value.(type) {
		case float64:
			return attrValue == val
		default:
			return false
		}
	case bool:
		switch attrValue := a.value.(type) {
		case bool:
			return attrValue == val
		default:
			return false
		}
	default:
		return false
	}
}

func (a matchStrictPolicyAttribute) Key() string {
	return a.key
}

type matchRegexPolicyAttribute struct {
	key   string
	value *regexp.Regexp
}

func NewMatchRegexPolicyAttribute(key string, value interface{}) (MatchPolicyAttribute, error) {
	if stringValue, ok := value.(string); ok {
		return matchRegexPolicyAttribute{key: key, value: regexp.MustCompile(stringValue)}, nil
	}
	if regexpValue, ok := value.(*regexp.Regexp); ok {
		return matchRegexPolicyAttribute{key: key, value: regexpValue}, nil

	}
	return matchRegexPolicyAttribute{}, fmt.Errorf("invalid regex value: %v", value)
}

func (a matchRegexPolicyAttribute) Match(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return a.value.MatchString(v)
	case fmt.Stringer:
		return a.value.MatchString(v.String())
	default:
		return false
	}
}

func (a matchRegexPolicyAttribute) Key() string {
	return a.key
}
