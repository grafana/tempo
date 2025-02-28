package vparquet4

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
			Cluster:     ptr("cluster1"),
		}
		r2 = Resource{
			ServiceName: "service1",
			Cluster:     ptr("cluster1"),
			Attrs:       []Attribute{{Key: "res.attr.1", Value: []string{"res.val.1"}}},
		}
		r3 = Resource{
			ServiceName: "service1",
			Cluster:     ptr("cluster2"),
		}
		r4 = Resource{
			ServiceName:         "service1",
			Cluster:             ptr("cluster1"),
			DedicatedAttributes: DedicatedAttributes{String01: ptr("string01")},
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
						{Scope: s1, Spans: []Span{spans[0]}},
						{Scope: s1, Spans: []Span{spans[1]}},
					},
				},
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[2]}},
						{Scope: s1, Spans: []Span{spans[3]}},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0], spans[1], spans[2], spans[3]}},
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
						{Scope: s1, Spans: []Span{spans[0]}},
						{Scope: s1, Spans: []Span{spans[1]}},
						{Scope: s2, Spans: []Span{spans[2]}},
					},
				},
				{
					Resource: r2,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[3]}},
						{Scope: s2, Spans: []Span{spans[4]}},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r2,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0], spans[1], spans[3]}},
						{Scope: s2, Spans: []Span{spans[2], spans[4]}},
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
						{Scope: s3, Spans: []Span{spans[1]}}, // intentionally out of order
						{Scope: s3, Spans: []Span{spans[0]}},
					},
				},
				{
					Resource: r4,
					ScopeSpans: []ScopeSpans{
						{Scope: s3, Spans: []Span{spans[3]}}, // intentionally out of order
						{Scope: s3, Spans: []Span{spans[2]}},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s3, Spans: []Span{spans[1], spans[0]}},
					},
				},
				{
					Resource: r4,
					ScopeSpans: []ScopeSpans{
						{Scope: s3, Spans: []Span{spans[3], spans[2]}},
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
						{Scope: s1, Spans: []Span{spans[0]}},
						{Scope: s2, Spans: []Span{spans[1], spans[2]}},
					},
				},
				{
					Resource: r3,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[3]}},
						{Scope: s2, Spans: []Span{spans[5], spans[4]}},
						{Scope: s3, Spans: []Span{spans[6]}},
					},
				},
			}},
			expected: Trace{ResourceSpans: []ResourceSpans{ // unmodified trace
				{
					Resource: r1,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[0]}},
						{Scope: s2, Spans: []Span{spans[1], spans[2]}},
					},
				},
				{
					Resource: r3,
					ScopeSpans: []ScopeSpans{
						{Scope: s1, Spans: []Span{spans[3]}},
						{Scope: s2, Spans: []Span{spans[5], spans[4]}},
						{Scope: s3, Spans: []Span{spans[6]}},
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
