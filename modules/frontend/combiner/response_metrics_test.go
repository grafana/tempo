package combiner

import (
	"net/http"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

// fakePipelineResponse implements PipelineResponse for tests. The cache hit
// header is what `IsCacheHit` keys on.
type fakePipelineResponse struct {
	cacheHit bool
}

func (f *fakePipelineResponse) HTTPResponse() *http.Response {
	h := http.Header{}
	if f.cacheHit {
		h.Set(TempoCacheHeader, "HIT")
	}
	return &http.Response{Header: h}
}

func (f *fakePipelineResponse) RequestData() any { return nil }
func (f *fakePipelineResponse) IsMetadata() bool { return false }

func TestSearchMetricsCombiner_SumsNewFields(t *testing.T) {
	mc := NewSearchMetricsCombiner()
	resp := &fakePipelineResponse{}

	mc.Combine(&tempopb.SearchMetrics{
		InspectedBytes:    100,
		BackendReads:      2,
		BackendBytes:      4096,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 5},
	}, resp)
	mc.Combine(&tempopb.SearchMetrics{
		InspectedBytes:    200,
		BackendReads:      3,
		BackendBytes:      8192,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 7, tempopb.AdditionalMetricCacheMisses: 1},
	}, resp)

	assert.Equal(t, uint64(300), mc.Metrics.InspectedBytes)
	assert.Equal(t, uint64(5), mc.Metrics.BackendReads)
	assert.Equal(t, uint64(12288), mc.Metrics.BackendBytes)
	assert.Equal(t, int64(12), mc.Metrics.AdditionalMetrics[tempopb.AdditionalMetricCacheHits])
	assert.Equal(t, int64(1), mc.Metrics.AdditionalMetrics[tempopb.AdditionalMetricCacheMisses])
}

func TestSearchMetricsCombiner_CacheHitSuppressesNewFields(t *testing.T) {
	mc := NewSearchMetricsCombiner()
	cacheHit := &fakePipelineResponse{cacheHit: true}
	mc.Combine(&tempopb.SearchMetrics{
		InspectedBytes:    100,
		BackendReads:      2,
		BackendBytes:      4096,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 5},
	}, cacheHit)
	// CompletedJobs still increments on cache hit (matches existing semantics).
	assert.Equal(t, uint32(1), mc.Metrics.CompletedJobs)
	assert.Equal(t, uint64(0), mc.Metrics.InspectedBytes)
	assert.Equal(t, uint64(0), mc.Metrics.BackendReads)
	assert.Equal(t, uint64(0), mc.Metrics.BackendBytes)
	assert.Empty(t, mc.Metrics.AdditionalMetrics)
}

func TestSearchMetricsCombiner_CombineMetadataDoesNotPullNewFields(t *testing.T) {
	// Regression guard: CombineMetadata is for sharder-emitted totals; it must
	// not accumulate the new backend/cache fields.
	mc := NewSearchMetricsCombiner()
	mc.CombineMetadata(&tempopb.SearchMetrics{
		TotalBlocks:       10,
		TotalJobs:         5,
		TotalBlockBytes:   1024,
		BackendReads:      99,
		BackendBytes:      99999,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 99},
	}, nil)
	assert.Equal(t, uint32(10), mc.Metrics.TotalBlocks)
	assert.Equal(t, uint64(0), mc.Metrics.BackendReads)
	assert.Equal(t, uint64(0), mc.Metrics.BackendBytes)
	assert.Empty(t, mc.Metrics.AdditionalMetrics)
}

func TestSearchMetricsCombiner_NilInputIsNoOp(t *testing.T) {
	mc := NewSearchMetricsCombiner()
	mc.Combine(nil, &fakePipelineResponse{})
	assert.Equal(t, uint32(0), mc.Metrics.CompletedJobs)
}

func TestTraceByIDMetricsCombiner_SumsNewFields(t *testing.T) {
	mc := NewTraceByIDMetricsCombiner()
	resp := &fakePipelineResponse{}
	mc.Combine(&tempopb.TraceByIDMetrics{
		InspectedBytes:    50,
		BackendReads:      1,
		BackendBytes:      2048,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 3},
	}, resp)
	mc.Combine(&tempopb.TraceByIDMetrics{
		BackendReads:      2,
		BackendBytes:      1024,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 4},
	}, resp)
	assert.Equal(t, uint64(50), mc.Metrics.InspectedBytes)
	assert.Equal(t, uint64(3), mc.Metrics.BackendReads)
	assert.Equal(t, uint64(3072), mc.Metrics.BackendBytes)
	assert.Equal(t, int64(7), mc.Metrics.AdditionalMetrics[tempopb.AdditionalMetricCacheHits])
}

func TestMetadataMetricsCombiner_SumsNewFields(t *testing.T) {
	mc := NewMetadataMetricsCombiner()
	resp := &fakePipelineResponse{}
	mc.Combine(&tempopb.MetadataMetrics{
		InspectedBytes:    100,
		BackendReads:      4,
		BackendBytes:      4096,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 8},
	}, resp)
	assert.Equal(t, uint64(100), mc.Metrics.InspectedBytes)
	assert.Equal(t, uint64(4), mc.Metrics.BackendReads)
	assert.Equal(t, uint64(4096), mc.Metrics.BackendBytes)
	assert.Equal(t, int64(8), mc.Metrics.AdditionalMetrics[tempopb.AdditionalMetricCacheHits])
}

func TestQueryRangeMetricsCombiner_SumsNewFields(t *testing.T) {
	mc := NewQueryRangeMetricsCombiner()
	resp := &fakePipelineResponse{}
	// Job response (non-metadata).
	mc.Combine(&tempopb.SearchMetrics{
		InspectedBytes:    200,
		BackendReads:      5,
		BackendBytes:      8192,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 9},
	}, resp)
	assert.Equal(t, uint64(200), mc.Metrics.InspectedBytes)
	assert.Equal(t, uint64(5), mc.Metrics.BackendReads)
	assert.Equal(t, uint64(8192), mc.Metrics.BackendBytes)
	assert.Equal(t, int64(9), mc.Metrics.AdditionalMetrics[tempopb.AdditionalMetricCacheHits])
}

func TestMergeAdditionalMetrics(t *testing.T) {
	// nil/empty source is no-op.
	assert.Nil(t, mergeAdditionalMetrics(nil, nil))
	assert.Nil(t, mergeAdditionalMetrics(nil, map[string]int64{}))

	// Allocates dst lazily and copies entries.
	dst := mergeAdditionalMetrics(nil, map[string]int64{"a": 1, "b": 2})
	assert.Equal(t, int64(1), dst["a"])
	assert.Equal(t, int64(2), dst["b"])

	// Sums into existing dst keys.
	dst = mergeAdditionalMetrics(dst, map[string]int64{"a": 5, "c": 7})
	assert.Equal(t, int64(6), dst["a"])
	assert.Equal(t, int64(2), dst["b"])
	assert.Equal(t, int64(7), dst["c"])
}
