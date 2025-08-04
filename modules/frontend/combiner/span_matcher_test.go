package combiner

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/spanfilter/policymatch"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_r "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeSpanMatcherPolicies(t *testing.T) {
	tc := []struct {
		policy   string
		expected *SpanMatcher
	}{
		{
			policy: "[{`resource.service`=`foo`}]",
			expected: &SpanMatcher{
				Policies: []*FilterPolicy{
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: true,
								resourceFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("service", "foo", config.Strict),
								}),
								intrinsicFilter: nil,
								spanFilter:      nil,
							},
						},
					},
				},
			},
		},
		{
			policy: "[{`span.team`=`foo`}]",
			expected: &SpanMatcher{
				Policies: []*FilterPolicy{
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: true,
								spanFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("team", "foo", config.Strict),
								}),
								intrinsicFilter: nil,
								resourceFilter:  nil,
							},
						},
					},
				},
			},
		},
		{
			policy: "[{`resource.service`=~`foo`}]",
			expected: &SpanMatcher{
				Policies: []*FilterPolicy{
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: true,
								resourceFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("service", "foo", config.Regex),
								}),
								intrinsicFilter: nil,
								spanFilter:      nil,
							},
						},
					},
				},
			},
		},
		{
			policy: "[{`resource.service`!=`foo`}]",
			expected: &SpanMatcher{
				Policies: []*FilterPolicy{
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: false,
								resourceFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("service", "foo", config.Strict),
								}),
								intrinsicFilter: nil,
								spanFilter:      nil,
							},
						},
					},
				},
			},
		},
		{
			policy: "[{`resource.service`!~`foo`}]",
			expected: &SpanMatcher{
				Policies: []*FilterPolicy{
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: false,
								resourceFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("service", "foo", config.Regex),
								}),
								intrinsicFilter: nil,
								spanFilter:      nil,
							},
						},
					},
				},
			},
		},
		{
			policy: "[{`resource.service`=`foo`,`span.team`=`bar`}]",
			expected: &SpanMatcher{
				Policies: []*FilterPolicy{
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: true,
								resourceFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("service", "foo", config.Strict),
								}),
								intrinsicFilter: nil,
								spanFilter:      nil,
							},
							{
								shouldMatch: true,
								spanFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("team", "bar", config.Strict),
								}),
								intrinsicFilter: nil,
								resourceFilter:  nil,
							},
						},
					},
				},
			},
		},
		{
			policy: "[{`resource.service`=`foo`},{`span.team`=`bar`}]",
			expected: &SpanMatcher{
				Policies: []*FilterPolicy{
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: true,
								resourceFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("service", "foo", config.Strict),
								}),
								intrinsicFilter: nil,
								spanFilter:      nil,
							},
						},
					},
					{
						Matchers: []*PolicyMatcher{
							{
								shouldMatch: true,
								spanFilter: policymatch.NewAttributePolicyMatch([]policymatch.AttributeFilter{
									newAttributeFilter("team", "bar", config.Strict),
								}),
								intrinsicFilter: nil,
								resourceFilter:  nil,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tc {
		t.Run(test.policy, func(t *testing.T) {
			sm, err := NewSpanMatcher(test.policy)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, sm)
		})
	}
}

func TestProcessTrace(t *testing.T) {
	tests := []struct {
		policy   string
		expected *tempopb.Trace
	}{
		{
			policy:   "[{`resource.match.all`=`foo`}]",
			expected: makeTestTrace(),
		},
		{
			policy: "[{`resource.service.name`=`foo`}]",
			expected: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{
					makeTestResource(
						[]*v1_common.KeyValue{
							stringKV("match.all", "foo"),
							stringKV("service.name", "foo"),
							stringKV("team", "bar"),
						},
						"test",
						[]*v1.Span{
							makeTestSpan("span1"),
							makeTestSpan("span2"),
						},
					),
					makeTestRedactedResource(
						[]*v1.Span{
							makeTestRedactedSpan("span1"),
							makeTestRedactedSpan("span2"),
						},
					),
				},
			},
		},
		{
			policy: "[{`resource.service.name`=`foo`, `span.name`=`span1`}]",
			expected: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{
					makeTestResource(
						[]*v1_common.KeyValue{
							stringKV("match.all", "foo"),
							stringKV("service.name", "foo"),
							stringKV("team", "bar"),
						},
						"test",
						[]*v1.Span{
							makeTestSpan("span1"),
							makeTestRedactedSpan("span2"),
						},
					),
					makeTestRedactedResource(
						[]*v1.Span{
							makeTestRedactedSpan("span1"),
							makeTestRedactedSpan("span2"),
						},
					),
				},
			},
		},
		{
			// multiple policies are OR operations
			policy:   "[{`resource.match.nothing` = `foo`},{`resource.match.all`=`foo`}]",
			expected: makeTestTrace(),
		},
		{
			// redact span level only
			policy: "[{`span.name`=`span1`}]",
			expected: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{
					makeTestResource(
						[]*v1_common.KeyValue{
							stringKV("match.all", "foo"),
							stringKV("service.name", "foo"),
							stringKV("team", "bar"),
						},
						"test",
						[]*v1.Span{
							makeTestSpan("span1"),
							makeTestRedactedSpan("span2"),
						},
					),
					makeTestResource(
						[]*v1_common.KeyValue{
							stringKV("match.all", "foo"),
							stringKV("service.name", "bar"),
							stringKV("team", "baz"),
						},
						"test",
						[]*v1.Span{
							makeTestSpan("span1"),
							makeTestRedactedSpan("span2"),
						},
					),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.policy, func(t *testing.T) {
			trace := makeTestTrace()
			sm, err := NewSpanMatcher(test.policy)
			require.NoError(t, err)

			sm.ProcessTrace(trace)
			assert.Equal(t, test.expected, trace)
		})
	}

}

func newAttributeFilter(key, value string, typ config.MatchType) policymatch.AttributeFilter {
	af, _ := policymatch.NewAttributeFilter(typ, key, value)
	return af
}

func stringKV(k, v string) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   k,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}},
	}
}

func makeTestSpan(name string) *v1.Span {
	return &v1.Span{
		Name:   name,
		SpanId: []byte(name),
		Kind:   v1.Span_SPAN_KIND_CLIENT,
		Links: []*v1.Span_Link{
			{
				TraceId:                []byte{0},
				SpanId:                 []byte{0},
				TraceState:             "state",
				DroppedAttributesCount: 3,
				Attributes: []*v1_common.KeyValue{
					stringKV("opentracing.ref_type", "child-of"),
				},
			},
		},
		Events: []*v1.Span_Event{
			{
				TimeUnixNano: uint64(1000*time.Second) + uint64(500*time.Millisecond),
				Name:         "event name",
				Attributes: []*v1_common.KeyValue{
					stringKV("exception.message", "random error"),
				},
			},
		},
		Attributes: []*v1_common.KeyValue{
			stringKV("service.name", "foo"),
			stringKV("team", "bar"),
			stringKV("name", name),
		},
	}
}

func makeTestResource(attributes []*v1_common.KeyValue, scopeName string, spans []*v1.Span) *v1.ResourceSpans {
	return &v1.ResourceSpans{
		Resource: &v1_r.Resource{
			Attributes: attributes,
		},
		ScopeSpans: []*v1.ScopeSpans{
			{
				Scope: &v1_common.InstrumentationScope{
					Name:    scopeName,
					Version: "1.0.0",
				},
				Spans: spans,
			},
		},
	}
}

func makeTestRedactedResource(spans []*v1.Span) *v1.ResourceSpans {
	return &v1.ResourceSpans{
		Resource: &v1_r.Resource{
			Attributes: []*v1_common.KeyValue{
				stringKV("service.name", "redacted"),
			},
			DroppedAttributesCount: 0,
		},
		ScopeSpans: []*v1.ScopeSpans{
			{
				Scope: makeTestRedactedScope(),
				Spans: spans,
			},
		},
	}
}

func makeTestRedactedSpan(name string) *v1.Span {
	return &v1.Span{
		Name:                   "redacted",
		SpanId:                 []byte(name),
		Kind:                   v1.Span_SPAN_KIND_CLIENT,
		Links:                  []*v1.Span_Link{},
		Events:                 []*v1.Span_Event{},
		Attributes:             []*v1_common.KeyValue{},
		DroppedAttributesCount: 0,
		DroppedEventsCount:     0,
		DroppedLinksCount:      0,
	}
}

func makeTestRedactedScope() *v1_common.InstrumentationScope {
	return &v1_common.InstrumentationScope{
		Name:                   "redacted",
		Version:                "redacted",
		Attributes:             []*v1_common.KeyValue{},
		DroppedAttributesCount: 0,
	}
}

func makeTestTrace() *tempopb.Trace {
	return &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			makeTestResource(
				[]*v1_common.KeyValue{
					stringKV("match.all", "foo"),
					stringKV("service.name", "foo"),
					stringKV("team", "bar"),
				},
				"test",
				[]*v1.Span{
					makeTestSpan("span1"),
					makeTestSpan("span2"),
				},
			),
			makeTestResource(
				[]*v1_common.KeyValue{
					stringKV("match.all", "foo"),
					stringKV("service.name", "bar"),
					stringKV("team", "baz"),
				},
				"test",
				[]*v1.Span{
					makeTestSpan("span1"),
					makeTestSpan("span2"),
				},
			),
		},
	}
}
