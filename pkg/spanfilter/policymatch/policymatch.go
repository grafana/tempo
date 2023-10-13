package policymatch

import (
	"fmt"
	"regexp"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

// PolicyMatch is a set of attributes that must match a span for the span to match the policy.
type PolicyMatch struct {
	attributes []MatchPolicyAttribute
}

// NewPolicyMatch returns a new PolicyMatch with the given attributes. If no attributes are given, then the policy matches all spans.
func NewPolicyMatch(attrs ...MatchPolicyAttribute) *PolicyMatch {
	return &PolicyMatch{attributes: attrs}
}

func (p *PolicyMatch) AddAttr(attr MatchPolicyAttribute) {
	p.attributes = append(p.attributes, attr)
}

// MatchAttrs returns true if all attributes in the policy match the attributes in the span.  String, bool, int, and floats are supported.  Regex MatchType may be applied to string span attributes.
func (p *PolicyMatch) MatchAttrs(attrs []*v1_common.KeyValue) bool {
	// If the policy has no attributes, then it matches.
	if len(p.attributes) == 0 {
		return true
	}

	// If the span has no attributes, then it does not match.
	if len(attrs) == 0 {
		return false
	}

	matches := 0
	for _, pa := range p.attributes {
		key := pa.Key()
		for _, attr := range attrs {
			if attr.GetKey() == key {
				v := attr.Value.Value

				// For each type of value, check if the policy attribute value matches the span attribute value.
				switch value := v.(type) {
				case *v1_common.AnyValue_StringValue:
					if !pa.MatchString(value.StringValue) {
						return false
					}
					matches++
				case *v1_common.AnyValue_IntValue:
					if !pa.MatchInt64(value.IntValue) {
						return false
					}
					matches++
				case *v1_common.AnyValue_DoubleValue:
					if !pa.MatchFloat64(value.DoubleValue) {
						return false
					}
					matches++
				case *v1_common.AnyValue_BoolValue:
					if !pa.MatchBool(value.BoolValue) {
						return false
					}
					matches++
				}
			}
		}
	}

	return len(p.attributes) == matches
}

// MatchIntrinsicAttrs returns true when all intrinsic values in the policy match the span.
func (p *PolicyMatch) MatchIntrinsicAttrs(span *v1_trace.Span) bool {
	matches := 0

	for _, pa := range p.attributes {
		switch pa.KeyIntrinsic() {
		// case traceql.IntrinsicDuration:
		// case traceql.IntrinsicChildCount:
		// case traceql.IntrinsicParent:
		case traceql.IntrinsicName:
			if !pa.MatchString(span.GetName()) {
				return false
			}
			matches++
		case traceql.IntrinsicStatus:
			if !pa.MatchStatusCode(span.GetStatus().GetCode()) {
				return false
			}
			matches++
		case traceql.IntrinsicKind:
			if !pa.MatchSpanKind(span.GetKind()) {
				return false
			}
			matches++
		}
	}

	return len(p.attributes) == matches
}

// MatchPolicyAttribute is an attribute that must match a span for the span to match the policy.
type MatchPolicyAttribute interface {
	Key() string
	KeyIntrinsic() traceql.Intrinsic
	MatchString(value string) bool
	MatchInt64(value int64) bool
	MatchSpanKind(value v1_trace.Span_SpanKind) bool
	MatchStatusCode(value v1_trace.Status_StatusCode) bool
	MatchFloat64(value float64) bool
	MatchBool(value bool) bool
}

// NewMatchPolicyAttribute returns a new MatchPolicyAttribute based on the match type.
func NewMatchPolicyAttribute(matchType config.MatchType, key string, value interface{}) (MatchPolicyAttribute, error) {
	if matchType == config.Regex {
		return NewMatchRegexPolicyAttribute(key, value)
	}
	return NewMatchStrictPolicyAttribute(key, value), nil
}

type MatchStrictPolicyAttributeType int

const (
	StringType MatchStrictPolicyAttributeType = iota
	Int64Type
	SpanKindType
	StatusCodeType
	Float64Type
	BoolType
)

type matchStrictPolicyAttribute struct {
	key             string
	keyIntrinsic    traceql.Intrinsic
	typ             MatchStrictPolicyAttributeType
	stringValue     string
	int64Value      int64
	spanKindValue   v1_trace.Span_SpanKind
	statusCodeValue v1_trace.Status_StatusCode
	float64Value    float64
	boolValue       bool
}

// NewMatchStrictPolicyAttribute returns a new MatchPolicyAttribute that matches against the given value.
func NewMatchStrictPolicyAttribute(key string, value interface{}) MatchPolicyAttribute {
	identifier, _ := traceql.ParseIdentifier(key)
	attr := matchStrictPolicyAttribute{
		key:          key,
		keyIntrinsic: identifier.Intrinsic,
	}

	switch v := value.(type) {
	case string:
		attr.typ = StringType
		attr.stringValue = v
	case int:
		attr.typ = Int64Type
		attr.int64Value = int64(v)
	case int64:
		attr.typ = Int64Type
		attr.int64Value = v
	case v1_trace.Span_SpanKind:
		attr.typ = SpanKindType
		attr.spanKindValue = v
	case v1_trace.Status_StatusCode:
		attr.typ = StatusCodeType
		attr.statusCodeValue = v
	case float64:
		attr.typ = Float64Type
		attr.float64Value = v
	case bool:
		attr.typ = BoolType
		attr.boolValue = v
	}

	return attr
}

func (a matchStrictPolicyAttribute) MatchString(value string) bool {
	return a.typ == StringType && a.stringValue == value ||
		a.typ == SpanKindType && a.spanKindValue.String() == value ||
		a.typ == StatusCodeType && a.statusCodeValue.String() == value
}

func (a matchStrictPolicyAttribute) MatchInt64(value int64) bool {
	return a.typ == Int64Type && a.int64Value == value
}

func (a matchStrictPolicyAttribute) MatchSpanKind(value v1_trace.Span_SpanKind) bool {
	return a.typ == SpanKindType && a.spanKindValue == value ||
		a.typ == StringType && a.stringValue == value.String()
}

func (a matchStrictPolicyAttribute) MatchStatusCode(value v1_trace.Status_StatusCode) bool {
	return a.typ == StatusCodeType && a.statusCodeValue == value ||
		a.typ == StringType && a.stringValue == value.String()
}

func (a matchStrictPolicyAttribute) MatchFloat64(value float64) bool {
	return a.typ == Float64Type && a.float64Value == value
}

func (a matchStrictPolicyAttribute) MatchBool(value bool) bool {
	return a.typ == BoolType && a.boolValue == value
}

func (a matchStrictPolicyAttribute) Key() string {
	return a.key
}

func (a matchStrictPolicyAttribute) KeyIntrinsic() traceql.Intrinsic {
	return a.keyIntrinsic
}

type matchRegexPolicyAttribute struct {
	key          string
	value        *regexp.Regexp
	keyIntrinsic traceql.Intrinsic
}

// NewMatchRegexPolicyAttribute returns a new MatchPolicyAttribute that matches against the given regex value.
func NewMatchRegexPolicyAttribute(key string, value interface{}) (MatchPolicyAttribute, error) {
	identifier, _ := traceql.ParseIdentifier(key)
	keyIntrinsic := identifier.Intrinsic

	if stringValue, ok := value.(string); ok {
		return matchRegexPolicyAttribute{
			key:          key,
			keyIntrinsic: keyIntrinsic,
			value:        regexp.MustCompile(stringValue),
		}, nil
	}
	if regexpValue, ok := value.(*regexp.Regexp); ok {
		return matchRegexPolicyAttribute{
			key:          key,
			keyIntrinsic: keyIntrinsic,
			value:        regexpValue,
		}, nil
	}
	return matchRegexPolicyAttribute{}, fmt.Errorf("invalid regex value: %v", value)
}

func (a matchRegexPolicyAttribute) MatchString(value string) bool {
	return a.value.MatchString(value)
}

func (a matchRegexPolicyAttribute) MatchInt64(_ int64) bool {
	return false
}

func (a matchRegexPolicyAttribute) MatchSpanKind(value v1_trace.Span_SpanKind) bool {
	return a.MatchString(value.String())
}

func (a matchRegexPolicyAttribute) MatchStatusCode(value v1_trace.Status_StatusCode) bool {
	return a.MatchString(value.String())
}

func (a matchRegexPolicyAttribute) MatchFloat64(_ float64) bool {
	// Does not support regex matching for float values.
	return false
}

func (a matchRegexPolicyAttribute) MatchBool(_ bool) bool {
	// Does not support regex matching for bool values.
	return false
}

func (a matchRegexPolicyAttribute) Key() string {
	return a.key
}

func (a matchRegexPolicyAttribute) KeyIntrinsic() traceql.Intrinsic {
	return a.keyIntrinsic
}
