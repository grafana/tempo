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

	// Parse the JSON to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	// Verify top-level structure
	assert.Contains(t, parsed, "trace")
	assert.Contains(t, parsed, "metrics")

	traceData := parsed["trace"].(map[string]interface{})
	assert.Equal(t, "0102030405060708090a0b0c0d0e0f10", traceData["traceId"])
	assert.Contains(t, traceData, "services")

	// Verify service structure
	services := traceData["services"].([]interface{})
	assert.Len(t, services, 1)
	service := services[0].(map[string]interface{})
	assert.Equal(t, "test-service", service["serviceName"])
	assert.Contains(t, service, "resource")
	assert.Contains(t, service, "scopes")

	// Verify resource attributes
	resource := service["resource"].(map[string]interface{})
	assert.Equal(t, "localhost", resource["host.name"])

	// Verify scope structure
	scopes := service["scopes"].([]interface{})
	assert.Len(t, scopes, 1)
	scope := scopes[0].(map[string]interface{})
	assert.Equal(t, "test-scope", scope["name"])
	assert.Equal(t, "1.0.0", scope["version"])
	assert.Contains(t, scope, "spans")

	// Verify span structure
	spans := scope["spans"].([]interface{})
	assert.Len(t, spans, 1)
	span := spans[0].(map[string]interface{})
	assert.Equal(t, "1112131415161718", span["spanId"])
	assert.Equal(t, "2122232425262728", span["parentSpanId"])
	assert.Equal(t, "test-span", span["name"])
	assert.Equal(t, "SPAN_KIND_SERVER", span["kind"])
	assert.Equal(t, "1000000000", span["startTimeUnixNano"])
	assert.Equal(t, "2000000000", span["endTimeUnixNano"])
	assert.Equal(t, 1000.0, span["durationMs"])

	// Verify attributes
	attrs := span["attributes"].(map[string]interface{})
	assert.Equal(t, "GET", attrs["http.method"])
	assert.Equal(t, float64(200), attrs["http.status_code"])
	assert.Equal(t, false, attrs["error"])

	// Verify events
	events := span["events"].([]interface{})
	assert.Len(t, events, 1)
	event := events[0].(map[string]interface{})
	assert.Equal(t, "test-event", event["name"])
	assert.Equal(t, "1500000000", event["timeUnixNano"])

	// Verify links
	links := span["links"].([]interface{})
	assert.Len(t, links, 1)
	link := links[0].(map[string]interface{})
	assert.Equal(t, "a1a2a3a4a5a6a7a8a9aaabacadaeafb0", link["traceId"])
	assert.Equal(t, "b1b2b3b4b5b6b7b8", link["spanId"])

	// Verify status
	status := span["status"].(map[string]interface{})
	assert.Equal(t, "STATUS_CODE_OK", status["code"])
	assert.Equal(t, "success", status["message"])

	// Verify metrics
	metrics := parsed["metrics"].(map[string]interface{})
	assert.Equal(t, float64(12345), metrics["inspectedBytes"])
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

	// Parse the JSON to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	// Verify top-level structure
	assert.Contains(t, parsed, "tagValues")
	assert.Contains(t, parsed, "metrics")

	// Verify tag values grouped by type
	tagValues := parsed["tagValues"].(map[string]interface{})
	assert.Contains(t, tagValues, "string")
	assert.Contains(t, tagValues, "int")
	assert.Contains(t, tagValues, "bool")

	stringValues := tagValues["string"].([]interface{})
	assert.Len(t, stringValues, 2)
	assert.Equal(t, "value1", stringValues[0])
	assert.Equal(t, "value2", stringValues[1])

	intValues := tagValues["int"].([]interface{})
	assert.Len(t, intValues, 2)
	assert.Equal(t, "42", intValues[0])
	assert.Equal(t, "100", intValues[1])

	boolValues := tagValues["bool"].([]interface{})
	assert.Len(t, boolValues, 1)
	assert.Equal(t, "true", boolValues[0])

	// Verify metrics
	metrics := parsed["metrics"].(map[string]interface{})
	assert.Equal(t, float64(5000), metrics["inspectedBytes"])
	assert.Equal(t, float64(10), metrics["totalJobs"])
	assert.Equal(t, float64(10), metrics["completedJobs"])
	assert.Equal(t, float64(5), metrics["totalBlocks"])
	assert.Equal(t, float64(10000), metrics["totalBlockBytes"])
}
