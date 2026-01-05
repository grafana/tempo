package main

import (
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineTraceResults_EmptyResults(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	results := []QueryResult{}
	trace, metadata, err := combiner.CombineTraceResults(results)

	require.NoError(t, err)
	assert.Nil(t, trace)
	assert.Equal(t, 0, metadata.InstancesQueried)
}

func TestCombineTraceResults_AllNotFound(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusNotFound},
			Body:     []byte{},
		},
		{
			Instance: "tempo-2",
			Response: &http.Response{StatusCode: http.StatusNotFound},
			Body:     []byte{},
		},
	}

	trace, metadata, err := combiner.CombineTraceResults(results)

	require.NoError(t, err)
	assert.Nil(t, trace)
	assert.Equal(t, 2, metadata.InstancesQueried)
	assert.Equal(t, 2, metadata.InstancesResponded)
	assert.Equal(t, 2, metadata.InstancesNotFound)
	assert.Equal(t, 0, metadata.InstancesWithTrace)
}

func TestCombineTraceResults_WithErrors(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Error:    assert.AnError,
		},
		{
			Instance: "tempo-2",
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
			Body:     []byte("internal error"),
		},
	}

	trace, metadata, err := combiner.CombineTraceResults(results)

	require.NoError(t, err)
	assert.Nil(t, trace)
	assert.Equal(t, 2, metadata.InstancesQueried)
	assert.Equal(t, 2, metadata.InstancesFailed)
	assert.Equal(t, 2, len(metadata.Errors))
	assert.True(t, metadata.PartialResponse)
}

func TestCombineTraceResults_SingleTrace(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	// Create a simple trace JSON (using batches format that Tempo returns)
	traceJSON := `{
		"batches": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "00000000000000000000000000000001",
					"spanId": "0000000000000001",
					"name": "test-span",
					"startTimeUnixNano": "1000000000",
					"endTimeUnixNano": "2000000000"
				}]
			}]
		}]
	}`

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     []byte(traceJSON),
		},
		{
			Instance: "tempo-2",
			Response: &http.Response{StatusCode: http.StatusNotFound},
			Body:     []byte{},
		},
	}

	trace, metadata, err := combiner.CombineTraceResults(results)

	require.NoError(t, err)
	require.NotNil(t, trace)
	assert.Equal(t, 2, metadata.InstancesQueried)
	assert.Equal(t, 2, metadata.InstancesResponded)
	assert.Equal(t, 1, metadata.InstancesWithTrace)
	assert.Equal(t, 1, metadata.InstancesNotFound)
	assert.Equal(t, 1, len(trace.ResourceSpans))
}

func TestCombineTraceResults_MergesDuplicateSpans(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	// Same trace from two instances (should be deduplicated)
	traceJSON := `{
		"batches": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "00000000000000000000000000000001",
					"spanId": "0000000000000001",
					"name": "test-span",
					"startTimeUnixNano": "1000000000",
					"endTimeUnixNano": "2000000000"
				}]
			}]
		}]
	}`

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     []byte(traceJSON),
		},
		{
			Instance: "tempo-2",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     []byte(traceJSON),
		},
	}

	trace, metadata, err := combiner.CombineTraceResults(results)

	require.NoError(t, err)
	require.NotNil(t, trace)
	assert.Equal(t, 2, metadata.InstancesQueried)
	assert.Equal(t, 2, metadata.InstancesWithTrace)
	// The trace combiner should deduplicate identical spans
	assert.Equal(t, 1, len(trace.ResourceSpans))
}

func TestCombineTraceResults_CombinesDifferentSpans(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	// Different spans from different services for the same trace
	traceJSON1 := `{
		"batches": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "service-a"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "00000000000000000000000000000001",
					"spanId": "0000000000000001",
					"name": "span-from-service-a",
					"startTimeUnixNano": "1000000000",
					"endTimeUnixNano": "2000000000"
				}]
			}]
		}]
	}`

	traceJSON2 := `{
		"batches": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "service-b"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "00000000000000000000000000000001",
					"spanId": "0000000000000002",
					"name": "span-from-service-b",
					"startTimeUnixNano": "1500000000",
					"endTimeUnixNano": "2500000000"
				}]
			}]
		}]
	}`

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     []byte(traceJSON1),
		},
		{
			Instance: "tempo-2",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     []byte(traceJSON2),
		},
	}

	trace, metadata, err := combiner.CombineTraceResults(results)

	require.NoError(t, err)
	require.NotNil(t, trace)
	assert.Equal(t, 2, metadata.InstancesQueried)
	assert.Equal(t, 2, metadata.InstancesWithTrace)
	// Should have 2 resource spans (one from each service)
	assert.Equal(t, 2, len(trace.ResourceSpans))
}

func TestCombineTraceResultsV2_WrappedResponse(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	// V2 API wraps the trace in {"trace": {...}, "metrics": {...}}
	v2JSON := `{
		"trace": {
			"batches": [{
				"resource": {
					"attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
				},
				"scopeSpans": [{
					"spans": [{
						"traceId": "00000000000000000000000000000001",
						"spanId": "0000000000000001",
						"name": "test-span",
						"startTimeUnixNano": "1000000000",
						"endTimeUnixNano": "2000000000"
					}]
				}]
			}]
		},
		"metrics": {}
	}`

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     []byte(v2JSON),
		},
	}

	trace, metadata, err := combiner.CombineTraceResultsV2(results)

	require.NoError(t, err)
	require.NotNil(t, trace)
	assert.Equal(t, 1, metadata.InstancesQueried)
	assert.Equal(t, 1, metadata.InstancesWithTrace)
	assert.Equal(t, 1, len(trace.ResourceSpans))
}

func TestCombineTraceResultsV2_EmptyTrace(t *testing.T) {
	logger := log.NewNopLogger()
	combiner := NewTraceCombiner(50*1024*1024, logger)

	// V2 API with empty trace
	v2JSON := `{
		"trace": {},
		"metrics": {}
	}`

	results := []QueryResult{
		{
			Instance: "tempo-1",
			Response: &http.Response{StatusCode: http.StatusOK},
			Body:     []byte(v2JSON),
		},
	}

	trace, metadata, err := combiner.CombineTraceResultsV2(results)

	require.NoError(t, err)
	assert.Nil(t, trace)
	assert.Equal(t, 1, metadata.InstancesNotFound)
}

// Helper function to create a test trace
func createTestTrace(serviceName, spanName string, spanID []byte) *tempopb.Trace {
	return &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key:   "service.name",
							Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: serviceName}},
						},
					},
				},
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Spans: []*v1_trace.Span{
							{
								TraceId:           []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
								SpanId:            spanID,
								Name:              spanName,
								StartTimeUnixNano: 1000000000,
								EndTimeUnixNano:   2000000000,
							},
						},
					},
				},
			},
		},
	}
}
