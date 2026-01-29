package vparquet5

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRebatchTrace(t *testing.T) {
	var (
		// setup distinct resources
		r1 = Resource{
			ServiceName: "service1",
		}
		r2 = Resource{
			ServiceName: "service1",
			Attrs:       []Attribute{{Key: "res.attr.1", Value: []string{"res.val.1"}}},
		}
		r3 = Resource{
			ServiceName: "service2",
		}
		r4 = Resource{
			ServiceName:         "service1",
			DedicatedAttributes: DedicatedAttributes{String01: []string{"string01"}},
		}
		// setup distinct instrumentation scopes
		s1 = InstrumentationScope{
			Name:    "scope1",
			Version: "0.0.1",
		}
		s2 = InstrumentationScope{
			Name:    "scope2",
			Version: "0.0.1",
		}
		s3 = InstrumentationScope{
			Name:    "scope2",
			Version: "0.0.1",
			Attrs:   []Attribute{{Key: "scope.attr.1", Value: []string{"scope.val.1"}}},
		}
		// setup some spans
		spans = makeTestSpans(t, 10)
	)

	tests := []struct {
		name     string
		trace    Trace
		expected Trace
	}{
		{
			name: "same resource same scope",
			trace: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0]}, SpanCount: 1},
						{Scope: s1, Spans: []Span{spans[1]}, SpanCount: 1},
					},
				},
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[2]}, SpanCount: 1},
						{Scope: s1, Spans: []Span{spans[3]}, SpanCount: 1},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0], spans[1], spans[2], spans[3]}, SpanCount: 4},
					},
				},
			}},
		},
		{
			name: "same resource different scope",
			trace: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r2,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0]}, SpanCount: 1},
						{Scope: s1, Spans: []Span{spans[1]}, SpanCount: 1},
						{Scope: s2, Spans: []Span{spans[2]}, SpanCount: 1},
					},
				},
				{
					Resource: r2,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[3]}, SpanCount: 1},
						{Scope: s2, Spans: []Span{spans[4]}, SpanCount: 1},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r2,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0], spans[1], spans[3]}, SpanCount: 3},
						{Scope: s2, Spans: []Span{spans[2], spans[4]}, SpanCount: 2},
					},
				},
			}},
		},
		{
			name: "different resource same scope",
			trace: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s3, Spans: []Span{spans[1]}, SpanCount: 1}, // intentionally out of order
						{Scope: s3, Spans: []Span{spans[0]}, SpanCount: 1},
					},
				},
				{
					Resource: r4,
					ScopeSpans: []ScopeSpans{
						{Scope: s3, Spans: []Span{spans[3]}, SpanCount: 1}, // intentionally out of order
						{Scope: s3, Spans: []Span{spans[2]}, SpanCount: 1},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s3, Spans: []Span{spans[1], spans[0]}, SpanCount: 2},
					},
				},
				{
					Resource: r4,
					ScopeSpans: []ScopeSpans{
						{Scope: s3, Spans: []Span{spans[3], spans[2]}, SpanCount: 2},
					},
				},
			}},
		},
		{
			name: "different resource different scope",
			trace: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0]}, SpanCount: 1},
						{Scope: s2, Spans: []Span{spans[1], spans[2]}, SpanCount: 2},
					},
				},
				{
					Resource: r3,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[3]}, SpanCount: 1},
						{Scope: s2, Spans: []Span{spans[5], spans[4]}, SpanCount: 2},
						{Scope: s3, Spans: []Span{spans[6]}, SpanCount: 1},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{ // unmodified trace
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0]}, SpanCount: 1},
						{Scope: s2, Spans: []Span{spans[1], spans[2]}, SpanCount: 2},
					},
				},
				{
					Resource: r3,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[3]}, SpanCount: 1},
						{Scope: s2, Spans: []Span{spans[5], spans[4]}, SpanCount: 2},
						{Scope: s3, Spans: []Span{spans[6]}, SpanCount: 1},
					},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rebatchTrace(&tt.trace)
			require.Equal(t, tt.expected, tt.trace)
		})
	}
}

var testSpanCount = 0

func makeTestSpans(t testing.TB, n int) []Span {
	spans := make([]Span, 0, n)
	for i := 0; i < n; i++ {
		testSpanCount++

		spanID := make([]byte, 8)
		_, err := rand.Read(spanID)
		require.NoError(t, err)

		spans = append(spans, Span{
			SpanID: spanID,
			Name:   fmt.Sprintf("span-%d", testSpanCount),
		})
	}
	return spans
}

func TestResourceSpanHashNoCollisions(t *testing.T) {
	cases := []ResourceSpans{
		{},
		{Resource: Resource{
			ServiceName: "abc",
			Attrs: []Attribute{
				attr("d", "e"),
			},
		}},
		{Resource: Resource{
			ServiceName: "a",
			Attrs: []Attribute{
				attr("bcd", "e"),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("a", []string{"ab", "c"}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("a", []string{"a", "bc"}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("a", []string{"ab", "c"}),
				attr("b", []string{"x"}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("a", []string{"a", "bc"}),
				attr("b", []string{"x"}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("a", "bcd"),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("a", []string{"bcd"}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("b", []string{"b"}),
				attr("c", []string{"d"}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("b", []int64{1, 2, 3}),
				attr("c", []int64{4}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("b", []int64{1, 2}),
				attr("c", []int64{3, 4}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("b", []float64{1, 2, 3}),
				attr("c", []float64{4}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("b", []float64{1, 2}),
				attr("c", []float64{3, 4}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("b", []bool{true, false, true}),
				attr("c", []bool{false}),
			},
		}},
		{Resource: Resource{
			ServiceName: "svc",
			Attrs: []Attribute{
				attr("b", []bool{true, false}),
				attr("c", []bool{true, false}),
			},
		}},
		{Resource: Resource{
			ServiceName:         "svc",
			DedicatedAttributes: DedicatedAttributes{String01: []string{"a", "b"}},
		}},
		{Resource: Resource{
			ServiceName:         "svc",
			DedicatedAttributes: DedicatedAttributes{String01: []string{"ab"}},
		}},
		{Resource: Resource{
			ServiceName:         "svc",
			DedicatedAttributes: DedicatedAttributes{String01: []string{"a"}, String02: []string{"b"}},
		}},
		{Resource: Resource{
			ServiceName:         "svc",
			DedicatedAttributes: DedicatedAttributes{Int01: []int64{1, 2}},
		}},
		{Resource: Resource{
			ServiceName:         "svc",
			DedicatedAttributes: DedicatedAttributes{Int01: []int64{1}, Int02: []int64{2}},
		}},
	}

	seen := map[uint64]int{}
	for i := range cases {
		h := resourceSpanHash(&cases[i])
		if j, ok := seen[h]; ok {
			t.Errorf("hash collision: resources produced the same hash %d:\n\tcases[%d]=%+v\n\tcases[%d]=%+v", h, j, cases[j], i, cases[i])
		}
		seen[h] = i
	}
}

func TestScopeSpanHashNoCollisions(t *testing.T) {
	cases := []ScopeSpans{
		{},
		{Scope: InstrumentationScope{
			Name:    "a",
			Version: "bcd",
		}},
		{Scope: InstrumentationScope{
			Name:    "abc",
			Version: "d",
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "abc",
			Attrs: []Attribute{
				attr("d", "a"),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "a",
			Attrs: []Attribute{
				attr("bcd", "a"),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("a", []string{"ab", "c"}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("a", []string{"a", "bc"}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("a", []string{"ab", "c"}),
				attr("b", []string{"x"}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("a", []string{"a", "bc"}),
				attr("b", []string{"x"}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("a", "bcd"),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("a", []string{"bcd"}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("a", []string{"b"}),
				attr("c", []string{"d"}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("b", []int64{1, 2, 3}),
				attr("c", []int64{4}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("b", []int64{1, 2}),
				attr("c", []int64{3, 4}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("b", []float64{1, 2, 3}),
				attr("c", []float64{4}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("b", []float64{1, 2}),
				attr("c", []float64{3, 4}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("b", []bool{true, false, true}),
				attr("c", []bool{false}),
			},
		}},
		{Scope: InstrumentationScope{
			Name:    "lib",
			Version: "1.0",
			Attrs: []Attribute{
				attr("b", []bool{true, false}),
				attr("c", []bool{true, false}),
			},
		}},
	}

	seen := map[uint64]int{}
	for i := range cases {
		h := scopeSpanHash(&cases[i])
		if j, ok := seen[h]; ok {
			t.Errorf("hash collision: scopes spans produced the same hash %d:\n\tcases[%d]=%+v\n\tcases[%d]=%+v", h, j, cases[j], i, cases[i])
		}
		seen[h] = i
	}
}
