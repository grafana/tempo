package combiner

import (
	"errors"
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/spanfilter/policymatch"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

const SpanMatcherHeader = "X-Span-Matcher"

type SpanMatcher struct {
	Policies []*FilterPolicy
}

type FilterPolicy struct {
	Matchers []*PolicyMatcher
}

type PolicyMatcher struct {
	shouldMatch     bool
	spanFilter      *policymatch.AttributePolicyMatch
	resourceFilter  *policymatch.AttributePolicyMatch
	intrinsicFilter *policymatch.IntrinsicPolicyMatch
}

func (fp *FilterPolicy) MatchIntrinsic(span *tracev1.Span) bool {
	// these are AND operators between matchers so exit early if any matcher does not match
	for _, matcher := range fp.Matchers {
		if matcher.intrinsicFilter == nil {
			return true
		}
		if matcher.shouldMatch != matcher.intrinsicFilter.Matches(span) {
			return false
		}
	}
	return true
}

func (fp *FilterPolicy) MatchSpan(span *tracev1.Span) bool {
	// these are AND operators between matchers so exit early if any matcher does not match
	for _, matcher := range fp.Matchers {
		if matcher.spanFilter == nil {
			return true
		}
		if matcher.shouldMatch != matcher.spanFilter.Matches(span.Attributes) {
			return false
		}
	}
	return true
}

func (fp *FilterPolicy) MatchResource(rsAttrs []*commonv1.KeyValue) bool {
	// these are AND operators between matchers so exit early if any matcher does not match
	for _, matcher := range fp.Matchers {
		if matcher.resourceFilter == nil {
			return true
		}
		if matcher.shouldMatch != matcher.resourceFilter.Matches(rsAttrs) {
			return false
		}
	}
	return true
}

func NewSpanMatcher(matcherValue string) (*SpanMatcher, error) {
	if matcherValue == "" {
		return nil, errors.New("span matcher header value is empty")
	}
	matcherValue = strings.TrimSpace(matcherValue)
	matcherValue = strings.ReplaceAll(matcherValue, "[", "")
	matcherValue = strings.ReplaceAll(matcherValue, "]", "")

	if matcherValue == "" {
		return nil, errors.New("span matcher header value is empty after trimming brackets")
	}

	stringPolicies := strings.Split(matcherValue, "},")

	policies := make([]*FilterPolicy, 0, len(stringPolicies))

	for _, policyStr := range stringPolicies {
		if !strings.HasSuffix(policyStr, "}") {
			policyStr += "}"
		}
		matchers, err := parser.ParseMetricSelector(policyStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing policy selector '%s': %v", policyStr, err)
		}
		policyMatchers := make([]*PolicyMatcher, 0, len(matchers))
		for _, matcher := range matchers {
			shouldMatch := true
			matchType := config.Strict
			attr, err := traceql.ParseIdentifier(matcher.Name)
			if err != nil {
				return nil, fmt.Errorf("invalid policy match attribute: %v", err)
			}
			switch matcher.Type {
			case labels.MatchEqual:
				matchType = config.Strict
				shouldMatch = true
			case labels.MatchNotEqual:
				matchType = config.Strict
				shouldMatch = false
			case labels.MatchRegexp:
				matchType = config.Regex
				shouldMatch = true
			case labels.MatchNotRegexp:
				matchType = config.Regex
				shouldMatch = false
			default:
				return nil, fmt.Errorf("unsupported match type: %v", matcher.Type)
			}
			policyMatcher := &PolicyMatcher{
				shouldMatch: shouldMatch,
			}
			if attr.Intrinsic > 0 {
				intrinsictMatcher, err := makeIntrinsicMatcher(matcher.Value, matchType)
				if err != nil {
					return nil, fmt.Errorf("error creating intrinsic matcher: %w", err)
				}
				policyMatcher.intrinsicFilter = policymatch.NewIntrinsicPolicyMatch([]policymatch.IntrinsicFilter{intrinsictMatcher})
			} else {
				if attr.Scope == traceql.AttributeScopeSpan {
					spanFilter, err := policymatch.NewAttributeFilter(matchType, attr.Name, matcher.Value)
					if err != nil {
						return nil, fmt.Errorf("error creating span attribute filter: %w", err)
					}
					policyMatcher.spanFilter = policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{spanFilter})
				} else if attr.Scope == traceql.AttributeScopeResource {
					resourceFilter, err := policymatch.NewAttributeFilter(matchType, attr.Name, matcher.Value)
					if err != nil {
						return nil, fmt.Errorf("error creating resource attribute filter: %w", err)
					}
					fmt.Printf("resourceFilter: %v\n", resourceFilter)
					policyMatcher.resourceFilter = policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{resourceFilter})
				} else {
					return nil, fmt.Errorf("invalid or unsupported attribute scope: %v", attr.Scope)
				}
			}
			policyMatchers = append(policyMatchers, policyMatcher)
			fmt.Println("policyMatcher:", policyMatcher)
		}
		policies = append(policies, &FilterPolicy{
			Matchers: policyMatchers,
		})
		fmt.Println("FilterPolicy:", policies)
	}
	return &SpanMatcher{
		Policies: policies,
	}, nil
}

// so we don't have to pass the resource attributes every time we check a span
func (sm *SpanMatcher) MatchResource(rsAttrs []*commonv1.KeyValue, rs *v1.Resource) []bool {
	// these are OR operations between policies so exit early if any policy matches
	matchedIndexes := make([]bool, len(sm.Policies))
	for i, policy := range sm.Policies {
		if policy.MatchResource(rsAttrs) {
			matchedIndexes[i] = true
		}
	}
	return matchedIndexes
}

func (sm *SpanMatcher) Match(matchedResourceIndex []bool, span *tracev1.Span) bool {
	// these are OR operations between policies so exit early if any policy matches
	for i, policy := range sm.Policies {
		if matchedResourceIndex[i] && (policy.MatchSpan(span) || policy.MatchIntrinsic(span)) {
			return true
		}
	}
	return false
}


func (sm *SpanMatcher) ProcessTrace(trace *tempopb.Trace) {
	for _, b := range trace.ResourceSpans {
		matchedResourceIndexes := sm.MatchResource(b.Resource.Attributes, b.Resource)
		matchedRsSpans := 0
		for _, ils := range b.ScopeSpans {
			matchedScopeSpans := 0
			for _, span := range ils.Spans {
				match := sm.Match(matchedResourceIndexes, span)
				if match {
					matchedRsSpans++
					matchedScopeSpans++
				} else {
					ProcessUnmatchedSpans(span)
				}
			}
			if matchedScopeSpans == 0 {
				// if no spans matched, remove the scope attributes
				ProcessUnmatchedScope(ils.Scope)
			}
		}
		if matchedRsSpans == 0 {
			// only completely remove resource attributes if no spans matched under this resource
			ProcessUnmatchedResource(b.Resource)
		}
	}
}

func makeIntrinsicMatcher(value string, matchType config.MatchType) (policymatch.IntrinsicFilter, error) {
	if matchType == config.Strict {
		return policymatch.NewNameIntrinsicFilter(value), nil
	}
	return policymatch.NewRegexpIntrinsicFilter(traceql.IntrinsicName, value)
}

func ProcessUnmatchedSpans(span *tracev1.Span) {
	span.Attributes = []*commonv1.KeyValue{}
	span.DroppedAttributesCount = 0
	span.Name = "redacted"
	span.Events = []*tracev1.Span_Event{}
	span.DroppedEventsCount = 0
	span.Links = []*tracev1.Span_Link{}
	span.DroppedLinksCount = 0
}

func ProcessUnmatchedResource(rs *v1.Resource) {
	rs.Attributes = []*commonv1.KeyValue{
		{
			Key:   "service.name",
			Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "redacted"}},
		},
	}
	rs.DroppedAttributesCount = 0
}

func ProcessUnmatchedScope(scope *commonv1.InstrumentationScope) {
	if scope.Name != "" {
		scope.Name = "redacted"
	}
	if scope.Version != "" {
		scope.Version = "redacted"
	}
	scope.Version = ""
	scope.Attributes = []*commonv1.KeyValue{}
	scope.DroppedAttributesCount = 0
}
