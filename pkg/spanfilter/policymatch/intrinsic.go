package policymatch

import (
	"fmt"
	"regexp"

	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
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
	kindMask   uint8
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
	case traceql.IntrinsicName, traceql.IntrinsicStatus:
		return IntrinsicFilter{intrinsic: intrinsic, regex: r}, nil
	case traceql.IntrinsicKind:
		// Keep the compiled regex even when the mask is set: Span_SpanKind is a
		// proto enum and unknown integer values stringify to decimal (e.g. "6").
		// Such values bypass spanKindMask (which returns 0 for them) and need
		// the regex engine to preserve main's matching behavior.
		return IntrinsicFilter{intrinsic: intrinsic, kindMask: kindRegexMask(r), regex: r}, nil
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
		if a.kindMask != 0 {
			if mask := spanKindMask(span.Kind); mask != 0 {
				return a.kindMask&mask != 0
			}
			// Kind is outside the four canonical values — fall through to the
			// regex engine if we kept one, so unknown enum values match the
			// same set of regexes they would on main.
		}
		if a.regex != nil {
			return a.regex.MatchString(span.Kind.String())
		}
		return a.kind == span.Kind
	default:
		return false
	}
}

const (
	spanKindServerBit uint8 = 1 << iota
	spanKindClientBit
	spanKindProducerBit
	spanKindConsumerBit
)

// kindRegexMask returns a bitmask for any regex whose match set is a non-empty
// subset of {SPAN_KIND_SERVER, SPAN_KIND_CLIENT, SPAN_KIND_PRODUCER,
// SPAN_KIND_CONSUMER}. The mask is determined by trial-matching the regex
// against each kind name and accepting only regexes that match no other
// strings (including the INTERNAL/UNSPECIFIED kinds and the empty string).
// This accelerates the default service-graph filter and any equivalent
// rewrite (different alternation order, anchors, whitespace) without changing
// behavior — the mask path is equivalent to running the regex by construction.
func kindRegexMask(re *regexp.Regexp) uint8 {
	var mask uint8
	if re.MatchString("SPAN_KIND_SERVER") {
		mask |= spanKindServerBit
	}
	if re.MatchString("SPAN_KIND_CLIENT") {
		mask |= spanKindClientBit
	}
	if re.MatchString("SPAN_KIND_PRODUCER") {
		mask |= spanKindProducerBit
	}
	if re.MatchString("SPAN_KIND_CONSUMER") {
		mask |= spanKindConsumerBit
	}
	if mask == 0 {
		return 0
	}
	// Reject regexes whose match set extends beyond the four kinds above.
	if re.MatchString("SPAN_KIND_INTERNAL") || re.MatchString("SPAN_KIND_UNSPECIFIED") {
		return 0
	}
	if re.MatchString("") {
		return 0
	}
	return mask
}

func spanKindMask(kind tracev1.Span_SpanKind) uint8 {
	switch kind {
	case tracev1.Span_SPAN_KIND_SERVER:
		return spanKindServerBit
	case tracev1.Span_SPAN_KIND_CLIENT:
		return spanKindClientBit
	case tracev1.Span_SPAN_KIND_PRODUCER:
		return spanKindProducerBit
	case tracev1.Span_SPAN_KIND_CONSUMER:
		return spanKindConsumerBit
	default:
		return 0
	}
}
