package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineSearchResults_EmptyResults(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	results := []QueryResult{}
	resp, metadata, err := combiner.CombineSearchResults(results)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Traces)
	assert.Equal(t, 0, metadata.InstancesQueried)
}

func TestCombineSearchResults_SingleInstance(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	searchResp := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "00000000000000000000000000000001",
				RootServiceName:   "test-service",
				RootTraceName:     "test-operation",
				StartTimeUnixNano: 1000000000,
				DurationMs:        100,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 100,
			InspectedBytes:  1024,
			TotalBlocks:     10,
		},
	}

	body, _ := json.Marshal(searchResp)
	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     body,
		},
	}

	resp, metadata, err := combiner.CombineSearchResults(results)

	require.NoError(t, err)
	assert.Equal(t, 1, len(resp.Traces))
	assert.Equal(t, "00000000000000000000000000000001", resp.Traces[0].TraceID)
	assert.Equal(t, uint32(100), resp.Metrics.InspectedTraces)
	assert.Equal(t, 1, metadata.InstancesQueried)
	assert.Equal(t, 1, metadata.InstancesResponded)
}

func TestCombineSearchResults_DeduplicatesTraces(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	// Same trace from two instances
	searchResp1 := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "00000000000000000000000000000001",
				RootServiceName:   "test-service",
				RootTraceName:     "test-operation",
				StartTimeUnixNano: 1000000000,
				DurationMs:        100,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 50,
			InspectedBytes:  512,
		},
	}

	searchResp2 := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "00000000000000000000000000000001",
				RootServiceName:   "test-service",
				RootTraceName:     "test-operation",
				StartTimeUnixNano: 1000000000,
				DurationMs:        150, // Longer duration from this instance
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 75,
			InspectedBytes:  768,
		},
	}

	body1, _ := json.Marshal(searchResp1)
	body2, _ := json.Marshal(searchResp2)

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     body1,
		},
		{
			Instance: "tempo-2",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     body2,
		},
	}

	resp, metadata, err := combiner.CombineSearchResults(results)

	require.NoError(t, err)
	// Should deduplicate to 1 trace
	assert.Equal(t, 1, len(resp.Traces))
	// Should take the longest duration
	assert.Equal(t, uint32(150), resp.Traces[0].DurationMs)
	// Metrics should be summed
	assert.Equal(t, uint32(125), resp.Metrics.InspectedTraces)
	assert.Equal(t, uint64(1280), resp.Metrics.InspectedBytes)
	assert.Equal(t, 2, metadata.InstancesResponded)
}

func TestCombineSearchResults_CombinesDifferentTraces(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	// Different traces from different instances
	searchResp1 := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "00000000000000000000000000000001",
				RootServiceName:   "service-a",
				StartTimeUnixNano: 2000000000, // More recent
				DurationMs:        100,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 50,
		},
	}

	searchResp2 := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "00000000000000000000000000000002",
				RootServiceName:   "service-b",
				StartTimeUnixNano: 1000000000, // Older
				DurationMs:        200,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 75,
		},
	}

	body1, _ := json.Marshal(searchResp1)
	body2, _ := json.Marshal(searchResp2)

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     body1,
		},
		{
			Instance: "tempo-2",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     body2,
		},
	}

	resp, metadata, err := combiner.CombineSearchResults(results)

	require.NoError(t, err)
	// Should have 2 different traces
	assert.Equal(t, 2, len(resp.Traces))
	// Should be sorted by start time descending (most recent first)
	assert.Equal(t, "00000000000000000000000000000001", resp.Traces[0].TraceID)
	assert.Equal(t, "00000000000000000000000000000002", resp.Traces[1].TraceID)
	// Metrics should be summed
	assert.Equal(t, uint32(125), resp.Metrics.InspectedTraces)
	assert.Equal(t, 2, metadata.InstancesResponded)
}

func TestCombineSearchResults_WithErrors(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	searchResp := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         "00000000000000000000000000000001",
				RootServiceName: "test-service",
			},
		},
	}
	body, _ := json.Marshal(searchResp)

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     body,
		},
		{
			Instance: "tempo-2",
			Error:    assert.AnError,
		},
		{
			Instance: "tempo-3",
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
			Body:     []byte("error"),
		},
	}

	resp, metadata, err := combiner.CombineSearchResults(results)

	require.NoError(t, err)
	assert.Equal(t, 1, len(resp.Traces))
	assert.Equal(t, 3, metadata.InstancesQueried)
	assert.Equal(t, 1, metadata.InstancesResponded)
	assert.Equal(t, 2, metadata.InstancesFailed)
	assert.Equal(t, 2, len(metadata.Errors))
}

func TestCombineSearchResultMetadata_TakesEarliestStartTime(t *testing.T) {
	existing := &tempopb.TraceSearchMetadata{
		TraceID:           "trace1",
		StartTimeUnixNano: 2000000000,
	}
	incoming := &tempopb.TraceSearchMetadata{
		TraceID:           "trace1",
		StartTimeUnixNano: 1000000000, // Earlier
	}

	combineSearchResultMetadata(existing, incoming)

	assert.Equal(t, uint64(1000000000), existing.StartTimeUnixNano)
}

func TestCombineSearchResultMetadata_TakesLongestDuration(t *testing.T) {
	existing := &tempopb.TraceSearchMetadata{
		TraceID:    "trace1",
		DurationMs: 100,
	}
	incoming := &tempopb.TraceSearchMetadata{
		TraceID:    "trace1",
		DurationMs: 200, // Longer
	}

	combineSearchResultMetadata(existing, incoming)

	assert.Equal(t, uint32(200), existing.DurationMs)
}

func TestCombineSearchResultMetadata_FillsMissingFields(t *testing.T) {
	existing := &tempopb.TraceSearchMetadata{
		TraceID: "trace1",
		// Missing root service name and trace name
	}
	incoming := &tempopb.TraceSearchMetadata{
		TraceID:         "trace1",
		RootServiceName: "my-service",
		RootTraceName:   "my-operation",
	}

	combineSearchResultMetadata(existing, incoming)

	assert.Equal(t, "my-service", existing.RootServiceName)
	assert.Equal(t, "my-operation", existing.RootTraceName)
}

func TestCombineSearchResultMetadata_MergesServiceStats(t *testing.T) {
	existing := &tempopb.TraceSearchMetadata{
		TraceID: "trace1",
		ServiceStats: map[string]*tempopb.ServiceStats{
			"service-a": {SpanCount: 5, ErrorCount: 1},
		},
	}
	incoming := &tempopb.TraceSearchMetadata{
		TraceID: "trace1",
		ServiceStats: map[string]*tempopb.ServiceStats{
			"service-a": {SpanCount: 10, ErrorCount: 0}, // Higher span count
			"service-b": {SpanCount: 3, ErrorCount: 2},  // New service
		},
	}

	combineSearchResultMetadata(existing, incoming)

	assert.Equal(t, uint32(10), existing.ServiceStats["service-a"].SpanCount)
	assert.Equal(t, uint32(1), existing.ServiceStats["service-a"].ErrorCount) // Keep higher error count
	assert.Equal(t, uint32(3), existing.ServiceStats["service-b"].SpanCount)
}

func TestSortTracesByStartTime(t *testing.T) {
	traces := []*tempopb.TraceSearchMetadata{
		{TraceID: "oldest", StartTimeUnixNano: 1000000000},
		{TraceID: "newest", StartTimeUnixNano: 3000000000},
		{TraceID: "middle", StartTimeUnixNano: 2000000000},
	}

	sortTracesByStartTime(traces)

	// Should be sorted descending (most recent first)
	assert.Equal(t, "newest", traces[0].TraceID)
	assert.Equal(t, "middle", traces[1].TraceID)
	assert.Equal(t, "oldest", traces[2].TraceID)
}

func TestSpansetKey(t *testing.T) {
	// Nil spanset
	assert.Equal(t, "", spansetKey(nil))

	// Spanset with no spans
	ss1 := &tempopb.SpanSet{Matched: 5}
	assert.Equal(t, "matched:5", spansetKey(ss1))

	// Spanset with spans
	ss2 := &tempopb.SpanSet{
		Matched: 1,
		Spans: []*tempopb.Span{
			{SpanID: "abc123"},
		},
	}
	assert.Contains(t, spansetKey(ss2), "616263313233") // hex of "abc123"
}
