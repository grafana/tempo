package policymatch

import (
	"fmt"
	"regexp"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
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
		case StringAttributeFilter, SpanKindAttributeFilter, StatusCodeAttributeFilter:
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
	SpanKindAttributeFilter
	StatusCodeAttributeFilter
	Float64AttributeFilter
	BoolAttributeFilter
	RegexAttributeFilter
)

// AttributeFilter is a filter that matches spans based on their attributes.
type AttributeFilter struct {
	key             string
	typ             AttributeFilterType
	stringValue     string
	int64Value      int64
	spanKindValue   tracev1.Span_SpanKind
	statusCodeValue tracev1.Status_StatusCode
	float64Value    float64
	boolValue       bool
	regex           *regexp.Regexp
}

// NewAttributeFilter returns a new AttributeFilter based on the match type.
func NewAttributeFilter(matchType config.MatchType, key string, value interface{}) (AttributeFilter, error) {
	if matchType == config.Regex {
		return NewRegexpAttributeFilter(key, value)
	}
	return NewStrictAttributeFilter(key, value), nil
}

// NewStrictAttributeFilter returns a new AttributeFilter that matches against the given value.
func NewStrictAttributeFilter(key string, value interface{}) AttributeFilter {
	attr := AttributeFilter{
		key: key,
	}

	switch v := value.(type) {
	case string:
		attr.stringValue = v
		if code, exists := tracev1.Status_StatusCode_value[v]; exists {
			attr.typ = StatusCodeAttributeFilter
			attr.statusCodeValue = tracev1.Status_StatusCode(code)
			break
		}
		if kind, exists := tracev1.Span_SpanKind_value[v]; exists {
			attr.typ = SpanKindAttributeFilter
			attr.spanKindValue = tracev1.Span_SpanKind(kind)
			break
		}
		attr.typ = StringAttributeFilter
	case int:
		attr.typ = Int64AttributeFilter
		attr.int64Value = int64(v)
	case int64:
		attr.typ = Int64AttributeFilter
		attr.int64Value = v
	case tracev1.Span_SpanKind:
		attr.typ = SpanKindAttributeFilter
		attr.spanKindValue = v
		attr.stringValue = v.String()
	case tracev1.Status_StatusCode:
		attr.typ = StatusCodeAttributeFilter
		attr.statusCodeValue = v
		attr.stringValue = v.String()
	case float64:
		attr.typ = Float64AttributeFilter
		attr.float64Value = v
	case bool:
		attr.typ = BoolAttributeFilter
		attr.boolValue = v
	}

	return attr
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
			return filter, fmt.Errorf("invalid regexp value: %v", regex)
		}
		filter.regex = compiled
	}
	if regexpValue, ok := regex.(*regexp.Regexp); ok {
		filter.regex = regexpValue
	}
	if filter.regex == nil {
		return filter, fmt.Errorf("invalid regex value: %v", regex)
	}
	return filter, nil
}
