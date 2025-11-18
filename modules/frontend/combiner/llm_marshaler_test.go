package combiner

import (
	"encoding/json"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceByIDResponseToSimplifiedJSON(t *testing.T) {
	// Create a comprehensive trace with various patterns
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "test-service"}}},
						{Key: "host.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "localhost"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Scope: &commonv1.InstrumentationScope{
							Name:    "test-scope",
							Version: "1.0.0",
						},
						Spans: []*tracev1.Span{
							{
								TraceId:           []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
								SpanId:            []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
								ParentSpanId:      []byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
								Name:              "test-span",
								Kind:              tracev1.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: 1000000000,
								EndTimeUnixNano:   2000000000,
								Attributes: []*commonv1.KeyValue{
									{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "GET"}}},
									{Key: "http.status_code", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 200}}},
									{Key: "error", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: false}}},
								},
								Events: []*tracev1.Span_Event{
									{
										Name:         "test-event",
										TimeUnixNano: 1500000000,
										Attributes: []*commonv1.KeyValue{
											{Key: "event.type", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "log"}}},
										},
									},
								},
								Links: []*tracev1.Span_Link{
									{
										TraceId: []byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf, 0xb0},
										SpanId:  []byte{0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8},
										Attributes: []*commonv1.KeyValue{
											{Key: "link.type", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "parent"}}},
										},
									},
								},
								Status: &tracev1.Status{
									Code:    tracev1.Status_STATUS_CODE_OK,
									Message: "success",
								},
							},
						},
					},
				},
			},
		},
	}

	traceResponse := &tempopb.TraceByIDResponse{
		Trace: trace,
		Metrics: &tempopb.TraceByIDMetrics{
			InspectedBytes: 12345,
		},
	}

	result, err := traceByIDResponseToSimplifiedJSON(traceResponse)
	require.NoError(t, err)

	// Parse the JSON
	var actual LLMTraceByIDResponse
	err = json.Unmarshal([]byte(result), &actual)
	require.NoError(t, err)

	// Build expected structure
	expected := &LLMTraceByIDResponse{
		Trace: LLMTrace{
			TraceID: "0102030405060708090a0b0c0d0e0f10",
			Services: []LLMService{
				{
					ServiceName: "test-service",
					Resource: map[string]interface{}{
						"host.name": "localhost",
					},
					Scopes: []LLMScope{
						{
							Name:    "test-scope",
							Version: "1.0.0",
							Spans: []LLMSpan{
								{
									SpanID:            "1112131415161718",
									ParentSpanID:      "2122232425262728",
									Name:              "test-span",
									Kind:              "SPAN_KIND_SERVER",
									StartTimeUnixNano: "1000000000",
									EndTimeUnixNano:   "2000000000",
									DurationMs:        1000.0,
									Attributes: map[string]interface{}{
										"http.method":      "GET",
										"http.status_code": float64(200),
										"error":            false,
									},
									Events: []LLMEvent{
										{
											Name:         "test-event",
											TimeUnixNano: "1500000000",
											Attributes: map[string]interface{}{
												"event.type": "log",
											},
										},
									},
									Links: []LLMLink{
										{
											TraceID: "a1a2a3a4a5a6a7a8a9aaabacadaeafb0",
											SpanID:  "b1b2b3b4b5b6b7b8",
											Attributes: map[string]interface{}{
												"link.type": "parent",
											},
										},
									},
									Status: LLMStatus{
										Code:    "STATUS_CODE_OK",
										Message: "success",
									},
								},
							},
						},
					},
				},
			},
		},
		Metrics: &LLMMetrics{
			InspectedBytes: 12345,
		},
	}

	assert.Equal(t, expected, &actual)
}

func TestSearchTagValuesV2ResponseToSimplifiedJSON(t *testing.T) {
	response := &tempopb.SearchTagValuesV2Response{
		TagValues: []*tempopb.TagValue{
			{Type: "string", Value: "value1"},
			{Type: "string", Value: "value2"},
			{Type: "int", Value: "42"},
			{Type: "int", Value: "100"},
			{Type: "bool", Value: "true"},
		},
		Metrics: &tempopb.MetadataMetrics{
			InspectedBytes:  5000,
			TotalJobs:       10,
			CompletedJobs:   10,
			TotalBlocks:     5,
			TotalBlockBytes: 10000,
		},
	}

	result, err := searchTagValuesV2ResponseToSimplifiedJSON(response)
	require.NoError(t, err)

	// Parse the JSON
	var actual LLMSearchTagValuesV2Response
	err = json.Unmarshal([]byte(result), &actual)
	require.NoError(t, err)

	// Build expected structure
	expected := &LLMSearchTagValuesV2Response{
		TagValues: map[string][]string{
			"string": {"value1", "value2"},
			"int":    {"42", "100"},
			"bool":   {"true"},
		},
		Metrics: &LLMMetrics{
			InspectedBytes:  5000,
			TotalJobs:       10,
			CompletedJobs:   10,
			TotalBlocks:     5,
			TotalBlockBytes: 10000,
		},
	}

	assert.Equal(t, expected, &actual)
}
