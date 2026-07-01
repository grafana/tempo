package distributor

import (
	"encoding/binary"
	"fmt"
	"testing"

	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// makeRebatchRequest builds a single push request containing numTraces distinct
// trace IDs, each with spansPerTrace spans, all under one resource+scope. It
// returns the request batches and the total span count.
func makeRebatchRequest(numTraces, spansPerTrace int) ([]v1.ResourceSpans, int) {
	spanCount := numTraces * spansPerTrace
	spans := make([]v1.Span, 0, spanCount)
	var seq uint64
	for t := 0; t < numTraces; t++ {
		traceID := make([]byte, 16)
		binary.BigEndian.PutUint64(traceID[8:], uint64(t+1))
		for s := 0; s < spansPerTrace; s++ {
			seq++
			spanID := make([]byte, 8)
			binary.BigEndian.PutUint64(spanID, seq)
			spans = append(spans, v1.Span{TraceId: traceID, SpanId: spanID})
		}
	}
	return []v1.ResourceSpans{
		{
			Resource: &v1_resource.Resource{
				Attributes: []v1_common.KeyValue{
					{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "svc"}}},
				},
			},
			ScopeSpans: []v1.ScopeSpans{
				{
					Scope: &v1_common.InstrumentationScope{Name: "lib", Version: "1.0.0"},
					Spans: spans,
				},
			},
		},
	}, spanCount
}

// BenchmarkRequestsByTraceIDScenarios covers normal prod-like traffic, a few
// legitimately large traces, and high-cardinality requests (the case that, before
// bounding preallocation, preallocated too much memory).
func BenchmarkRequestsByTraceIDScenarios(b *testing.B) {
	cases := []struct {
		name          string
		numTraces     int
		spansPerTrace int
	}{
		{"normal_prod_like", 37, 6},        // prod: ~37 traces/batch, ~6 spans/trace
		{"few_large_traces", 7, 5000},      // legitimate large traces
		{"high_cardinality_5k", 5000, 1},   // many unique traces, 1 span each
		{"high_cardinality_20k", 20000, 1}, // many unique traces, 1 span each
	}

	for _, tc := range cases {
		batches, spanCount := makeRebatchRequest(tc.numTraces, tc.spansPerTrace)
		b.Run(fmt.Sprintf("%s/traces=%d/spansPer=%d", tc.name, tc.numTraces, tc.spansPerTrace), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, traces, _, _, err := requestsByTraceID(batches, "test", spanCount, 0)
				if err != nil {
					b.Fatal(err)
				}
				if len(traces) != tc.numTraces {
					b.Fatalf("expected %d traces, got %d", tc.numTraces, len(traces))
				}
			}
		})
	}
}
