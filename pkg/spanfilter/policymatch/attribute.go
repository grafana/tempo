package policymatch

import (
	"fmt"
	"regexp"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/common/v1"
)

// AttributePolicyMatch is a set of attribute filters that must match a span for the span to match the policy.
type AttributePolicyMatch struct {
	filters []AttributeFilter
}

// NewAttributePolicyMatch returns a new AttributePolicyMatch with the given filters. If no filters are given, then the policy matches all spans.
func NewAttributePolicyMatch(filters []AttributeFilter) *AttributePolicyMatch {
	return &AttributePolicyMatch{filters: filters}
}

// Matches returns true if the given span matches the policy.
func (p *AttributePolicyMatch) Matches(attrs []*commonv1.KeyValue) bool {
	// If there are no filters, then the span matches.
	if len(p.filters) == 0 {
		return true
	}

	// If there are no attributes, then the span does not match.
	if len(attrs) == 0 {
		return false
	}

	for _, pa := range p.filters {
		if !matchesAnyFilter(pa, attrs) {
			return false
		}
	}

	return true
}

func matchesAnyFilter(pa AttributeFilter, attrs []*commonv1.KeyValue) bool {
	for _, attr := range attrs {
		// If the attribute key does not match, then it cannot match the policy.
		if pa.key != attr.Key {
			continue
		}
		switch pa.typ {
		case StringAttributeFilter:
			return pa.stringValue == attr.Value.GetStringValue()
		case Int64AttributeFilter:
			return pa.int64Value == attr.Value.GetIntValue()
		case Float64AttributeFilter:
			return pa.float64Value == attr.Value.GetDoubleValue()
		case BoolAttributeFilter:
			return pa.boolValue == attr.Value.GetBoolValue()
		case RegexAttributeFilter:
			return pa.regex.MatchString(attr.Value.GetStringValue())
		}
	}
	return false
}

type AttributeFilterType int

const (
	StringAttributeFilter AttributeFilterType = iota
	Int64AttributeFilter
	Float64AttributeFilter
	BoolAttributeFilter
	RegexAttributeFilter
)

// AttributeFilter is a filter that matches spans based on their attributes.
type AttributeFilter struct {
	key          string
	typ          AttributeFilterType
	stringValue  string
	int64Value   int64
	float64Value float64
	boolValue    bool
	regex        *regexp.Regexp
}

// NewAttributeFilter returns a new AttributeFilter based on the match type.
func NewAttributeFilter(matchType config.MatchType, key string, value interface{}) (AttributeFilter, error) {
	if matchType == config.Regex {
		return NewRegexpAttributeFilter(key, value)
	}
	return NewStrictAttributeFilter(key, value)
}

// NewStrictAttributeFilter returns a new AttributeFilter that matches against the given value.
func NewStrictAttributeFilter(key string, value interface{}) (AttributeFilter, error) {
	switch v := value.(type) {
	case string:
		return AttributeFilter{
			key:         key,
			typ:         StringAttributeFilter,
			stringValue: v,
		}, nil
	case int:
		return AttributeFilter{
			key:        key,
			typ:        Int64AttributeFilter,
			int64Value: int64(v),
		}, nil
	case int64:
		return AttributeFilter{
			key:        key,
			typ:        Int64AttributeFilter,
			int64Value: v,
		}, nil
	case float64:
		return AttributeFilter{
			key:          key,
			typ:          Float64AttributeFilter,
			float64Value: v,
		}, nil
	case bool:
		return AttributeFilter{
			key:       key,
			typ:       BoolAttributeFilter,
			boolValue: v,
		}, nil
	default:
		return AttributeFilter{}, fmt.Errorf("value type not supported: %T", value)
	}
}

// NewRegexpAttributeFilter returns a new AttributeFilter that matches against the given regex value.
func NewRegexpAttributeFilter(key string, regex interface{}) (AttributeFilter, error) {
	filter := AttributeFilter{
		key: key,
		typ: RegexAttributeFilter,
	}
	if stringValue, ok := regex.(string); ok {
		compiled, err := regexp.Compile(stringValue)
		if err != nil {
			return filter, fmt.Errorf("invalid attribute filter regexp: %v", err)
		}
		filter.regex = compiled
	}
	if regexpValue, ok := regex.(*regexp.Regexp); ok {
		filter.regex = regexpValue
	}
	if filter.regex == nil {
		return filter, fmt.Errorf("invalid attribute filter regexp value: %v", regex)
	}
	return filter, nil
}
