package querier

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/tempodb/backend"
)

// BenchmarkMetricsRowGroupCacheKey measures the cost of building a per-row-
// group cache key. This runs once per row group on every metric query, so it's
// on the hot path.
func BenchmarkMetricsRowGroupCacheKey(b *testing.B) {
	blockID := backend.MustParse("00000000-0000-0000-0000-000000000123")
	b.ResetTimer()
	for b.Loop() {
		s := metricsRowGroupCacheKey("tenant-a", blockID, 4, 0xdeadbeef)
		if len(s) == 0 {
			b.Fatal("empty key")
		}
	}
}

// BenchmarkRowGroupQueryRangeHash measures the cost of the canonical hash.
// Called once per queryBlock invocation.
func BenchmarkRowGroupQueryRangeHash(b *testing.B) {
	req := &tempopb.QueryRangeRequest{
		Query:                  "{ resource.service.name = `foo` } | rate() by (span.http.url)",
		Step:                   15_000_000_000,
		MaxSeries:              1000,
		Exemplars:              50,
		SkipASTTransformations: []string{"x", "y"},
	}
	dc := backend.DedicatedColumns{
		{Scope: backend.DedicatedColumnScopeSpan, Name: "http.url", Type: backend.DedicatedColumnTypeString},
		{Scope: backend.DedicatedColumnScopeResource, Name: "service.name", Type: backend.DedicatedColumnTypeString},
	}
	b.ResetTimer()
	for b.Loop() {
		h := rowGroupQueryRangeHash(req, dc, 0.2, true)
		if h == 0 {
			b.Fatal("zero hash")
		}
	}
}

// BenchmarkRowGroupCacheValueRoundTrip measures the proto marshal+unmarshal
// cost for a representative QueryRangeResponse. The cache hot path does one
// Unmarshal per hit row group; misses do one Marshal each.
func BenchmarkRowGroupCacheValueRoundTrip(b *testing.B) {
	// Build a mid-sized response: 50 series with 100 samples each. Not
	// representative of every query, but a reasonable sanity check.
	const numSeries = 50
	const numSamples = 100
	series := make([]*tempopb.TimeSeries, numSeries)
	for i := 0; i < numSeries; i++ {
		samples := make([]tempopb.Sample, numSamples)
		for j := 0; j < numSamples; j++ {
			samples[j] = tempopb.Sample{TimestampMs: int64(j) * 1000, Value: float64(j)}
		}
		series[i] = &tempopb.TimeSeries{
			Labels: []commonv1.KeyValue{
				{Key: "__name__", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "rate"}}},
				{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "svc"}}},
			},
			Samples: samples,
		}
	}
	resp := &tempopb.QueryRangeResponse{
		Series: series,
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: 1 << 20,
			InspectedSpans: 100_000,
		},
	}

	b.ResetTimer()
	b.Run("marshal", func(b *testing.B) {
		for b.Loop() {
			buf, err := marshalRowGroupCacheValue(resp)
			if err != nil {
				b.Fatal(err)
			}
			if len(buf) == 0 {
				b.Fatal("empty buf")
			}
		}
	})

	buf, err := marshalRowGroupCacheValue(resp)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("unmarshal", func(b *testing.B) {
		for b.Loop() {
			got, err := unmarshalRowGroupCacheValue(buf)
			if err != nil {
				b.Fatal(err)
			}
			if got == nil || len(got.Series) != numSeries {
				b.Fatalf("unexpected unmarshal result")
			}
		}
	})
}
