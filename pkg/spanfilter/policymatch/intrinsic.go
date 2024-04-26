package policymatch

import (
	"fmt"
	"regexp"

	tracev1 "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

// IntrinsicPolicyMatch is a set of filters that must match a span for the span to match the policy.
type IntrinsicPolicyMatch struct {
	filters []IntrinsicFilter
}

// NewIntrinsicPolicyMatch returns a new IntrinsicPolicyMatch with the given filters. If no filters are given, then the policy matches all spans.
func NewIntrinsicPolicyMatch(filters []IntrinsicFilter) *IntrinsicPolicyMatch {
	return &IntrinsicPolicyMatch{filters: filters}
}

// Matches returns true if the given span matches the policy.
func (p *IntrinsicPolicyMatch) Matches(span *tracev1.Span) bool {
	if len(p.filters) == 0 {
		return true
	}
	for _, attr := range p.filters {
		if !attr.Matches(span) {
			return false
		}
	}
	return true
}

// IntrinsicFilter is a filter that matches spans based on their intrinsic attributes.
type IntrinsicFilter struct {
	intrinsic  traceql.Intrinsic
	name       string
	statusCode tracev1.Status_StatusCode
	kind       tracev1.Span_SpanKind
	regex      *regexp.Regexp
}

// NewStrictIntrinsicFilter returns a new IntrinsicFilter that matches spans based on the given intrinsic and value.
func NewStrictIntrinsicFilter(intrinsic traceql.Intrinsic, value interface{}) (IntrinsicFilter, error) {
	switch intrinsic {
	case traceql.IntrinsicKind:
		if v, ok := value.(string); ok {
			if kind, ok := tracev1.Span_SpanKind_value[v]; ok {
				return NewKindIntrinsicFilter(tracev1.Span_SpanKind(kind)), nil
			}
			return IntrinsicFilter{}, fmt.Errorf("unsupported kind intrinsic string value: %s", v)
		}
		return IntrinsicFilter{}, fmt.Errorf("invalid kind intrinsic value: %v", value)
	case traceql.IntrinsicStatus:
		if v, ok := value.(string); ok {
			if code, ok := tracev1.Status_StatusCode_value[v]; ok {
				return NewStatusIntrinsicFilter(tracev1.Status_StatusCode(code)), nil
			}
			return IntrinsicFilter{}, fmt.Errorf("unsupported status intrinsic string value: %s", v)
		}
		return IntrinsicFilter{}, fmt.Errorf("unsupported status intrinsic value: %v", value)
	case traceql.IntrinsicName:
		if v, ok := value.(string); ok {
			return NewNameIntrinsicFilter(v), nil
		}
		return IntrinsicFilter{}, fmt.Errorf("unsupported name intrinsic value: %v", value)
	default:
		return IntrinsicFilter{}, fmt.Errorf("unsupported intrinsic: %v", intrinsic)
	}
}

// NewKindIntrinsicFilter returns a new IntrinsicFilter that matches spans with the given kind.
func NewKindIntrinsicFilter(kind tracev1.Span_SpanKind) IntrinsicFilter {
	return IntrinsicFilter{intrinsic: traceql.IntrinsicKind, kind: kind}
}

// NewStatusIntrinsicFilter returns a new IntrinsicFilter that matches spans with the given status code.
func NewStatusIntrinsicFilter(statusCode tracev1.Status_StatusCode) IntrinsicFilter {
	return IntrinsicFilter{intrinsic: traceql.IntrinsicStatus, statusCode: statusCode}
}

// NewNameIntrinsicFilter returns a new IntrinsicFilter that matches spans with the given name.
func NewNameIntrinsicFilter(value string) IntrinsicFilter {
	return IntrinsicFilter{intrinsic: traceql.IntrinsicName, name: value}
}

// NewRegexpIntrinsicFilter returns a new IntrinsicFilter that matches spans based on the given regex and intrinsic.
func NewRegexpIntrinsicFilter(intrinsic traceql.Intrinsic, value interface{}) (IntrinsicFilter, error) {
	var (
		stringValue string
		ok          bool
	)
	if stringValue, ok = value.(string); !ok {
		return IntrinsicFilter{}, fmt.Errorf("unsupported intrinsic filter regex value: %v", value)
	}
	r, err := regexp.Compile(stringValue)
	if err != nil {
		return IntrinsicFilter{}, fmt.Errorf("invalid intrinsic filter regex: %v", err)
	}
	switch intrinsic {
	case traceql.IntrinsicName, traceql.IntrinsicStatus, traceql.IntrinsicKind:
		return IntrinsicFilter{intrinsic: intrinsic, regex: r}, nil
	default:
		return IntrinsicFilter{}, fmt.Errorf("intrinsic not supported %s", intrinsic)
	}
}

// Matches returns true if the given span matches the filter.
func (a *IntrinsicFilter) Matches(span *tracev1.Span) bool {
	switch a.intrinsic {
	case traceql.IntrinsicName:
		if a.regex != nil {
			return a.regex.MatchString(span.Name)
		}
		return a.name == span.Name
	case traceql.IntrinsicStatus:
		if a.regex != nil {
			return a.regex.MatchString(span.GetStatus().GetCode().String())
		}
		return a.statusCode == span.GetStatus().GetCode()
	case traceql.IntrinsicKind:
		if a.regex != nil {
			return a.regex.MatchString(span.Kind.String())
		}
		return a.kind == span.Kind
	default:
		return false
	}
}
