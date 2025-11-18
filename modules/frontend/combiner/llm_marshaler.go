package combiner

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

type llmMarshaler struct{}

func (m *llmMarshaler) marshalToString(t proto.Message) (string, error) {
	// unsupported: *tempopb.Trace, *tempopb.SearchTagsResponse, *tempopb.SearchTagValuesResponse, *tempopb.QueryRangeResponse

	switch v := t.(type) {
	case *tempopb.TraceByIDResponse:
		return traceByIDResponseToSimplifiedJSON(v)
	case *tempopb.SearchTagValuesV2Response:
		return searchTagValuesV2ResponseToSimplifiedJSON(v)
	}

	return "", util.ErrUnsupported
}

func traceByIDResponseToSimplifiedJSON(t *tempopb.TraceByIDResponse) (string, error) {
	if t.Trace == nil || len(t.Trace.ResourceSpans) == 0 {
		return "{}", nil
	}

	// Extract traceId from first span (all spans share the same traceId)
	var traceID string
	if len(t.Trace.ResourceSpans) > 0 && len(t.Trace.ResourceSpans[0].ScopeSpans) > 0 && len(t.Trace.ResourceSpans[0].ScopeSpans[0].Spans) > 0 {
		traceID = bytesToHex(t.Trace.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId)
	}

	// Build services array
	services := make([]map[string]interface{}, 0, len(t.Trace.ResourceSpans))
	for _, rs := range t.Trace.ResourceSpans {
		service := make(map[string]interface{})

		// Extract service name and flatten resource attributes
		resourceAttrs := flattenAttributes(rs.Resource.GetAttributes())
		if serviceName, ok := resourceAttrs["service.name"]; ok {
			service["serviceName"] = serviceName
			delete(resourceAttrs, "service.name")
		}
		if len(resourceAttrs) > 0 {
			service["resource"] = resourceAttrs
		}

		// Build scopes array
		scopes := make([]map[string]interface{}, 0, len(rs.ScopeSpans))
		for _, ss := range rs.ScopeSpans {
			scope := make(map[string]interface{})
			if ss.Scope != nil {
				if ss.Scope.Name != "" {
					scope["name"] = ss.Scope.Name
				}
				if ss.Scope.Version != "" {
					scope["version"] = ss.Scope.Version
				}
			}

			// Build spans array
			spans := make([]map[string]interface{}, 0, len(ss.Spans))
			for _, span := range ss.Spans {
				simplifiedSpan := simplifySpan(span)
				spans = append(spans, simplifiedSpan)
			}

			if len(spans) > 0 {
				scope["spans"] = spans
			}
			scopes = append(scopes, scope)
		}

		if len(scopes) > 0 {
			service["scopes"] = scopes
		}
		services = append(services, service)
	}

	// Build final structure
	result := map[string]interface{}{
		"trace": map[string]interface{}{
			"traceId":  traceID,
			"services": services,
		},
	}

	// Add metrics if present
	if t.Metrics != nil {
		metrics := make(map[string]interface{})
		if t.Metrics.InspectedBytes > 0 {
			metrics["inspectedBytes"] = t.Metrics.InspectedBytes
		}
		if len(metrics) > 0 {
			result["metrics"] = metrics
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func simplifySpan(span *tracev1.Span) map[string]interface{} {
	result := map[string]interface{}{
		"spanId": bytesToHex(span.SpanId),
		"name":   span.Name,
	}

	if len(span.ParentSpanId) > 0 {
		result["parentSpanId"] = bytesToHex(span.ParentSpanId)
	}

	// Add kind if not unspecified
	if span.Kind != tracev1.Span_SPAN_KIND_UNSPECIFIED {
		result["kind"] = span.Kind.String()
	}

	// Add timestamps
	if span.StartTimeUnixNano > 0 {
		result["startTimeUnixNano"] = fmt.Sprintf("%d", span.StartTimeUnixNano)
	}
	if span.EndTimeUnixNano > 0 {
		result["endTimeUnixNano"] = fmt.Sprintf("%d", span.EndTimeUnixNano)
	}

	// Calculate duration in milliseconds
	if span.EndTimeUnixNano > 0 && span.StartTimeUnixNano > 0 {
		durationNano := span.EndTimeUnixNano - span.StartTimeUnixNano
		durationMs := float64(durationNano) / 1_000_000.0
		result["durationMs"] = durationMs
	}

	// Flatten attributes
	if len(span.Attributes) > 0 {
		result["attributes"] = flattenAttributes(span.Attributes)
	}

	// Add events if present
	if len(span.Events) > 0 {
		events := make([]map[string]interface{}, 0, len(span.Events))
		for _, event := range span.Events {
			e := map[string]interface{}{
				"name": event.Name,
			}
			if event.TimeUnixNano > 0 {
				e["timeUnixNano"] = fmt.Sprintf("%d", event.TimeUnixNano)
			}
			if len(event.Attributes) > 0 {
				e["attributes"] = flattenAttributes(event.Attributes)
			}
			events = append(events, e)
		}
		result["events"] = events
	}

	// Add links if present
	if len(span.Links) > 0 {
		links := make([]map[string]interface{}, 0, len(span.Links))
		for _, link := range span.Links {
			l := map[string]interface{}{
				"traceId": bytesToHex(link.TraceId),
				"spanId":  bytesToHex(link.SpanId),
			}
			if len(link.Attributes) > 0 {
				l["attributes"] = flattenAttributes(link.Attributes)
			}
			links = append(links, l)
		}
		result["links"] = links
	}

	// Add status
	status := make(map[string]interface{})
	if span.Status != nil {
		if span.Status.Code != tracev1.Status_STATUS_CODE_UNSET {
			status["code"] = span.Status.Code.String()
		}
		if span.Status.Message != "" {
			status["message"] = span.Status.Message
		}
	}
	if len(status) == 0 {
		// Default empty status
		status["code"] = "STATUS_CODE_UNSET"
		status["message"] = ""
	}
	result["status"] = status

	return result
}

// flattenAttributes converts []*KeyValue to a flat map[string]interface{}
func flattenAttributes(attrs []*commonv1.KeyValue) map[string]interface{} {
	result := make(map[string]interface{})
	for _, attr := range attrs {
		if attr.Value != nil {
			result[attr.Key] = extractAnyValue(attr.Value)
		}
	}
	return result
}

// extractAnyValue extracts the actual value from an AnyValue wrapper
func extractAnyValue(av *commonv1.AnyValue) interface{} {
	switch v := av.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return v.StringValue
	case *commonv1.AnyValue_BoolValue:
		return v.BoolValue
	case *commonv1.AnyValue_IntValue:
		return v.IntValue
	case *commonv1.AnyValue_DoubleValue:
		return v.DoubleValue
	case *commonv1.AnyValue_BytesValue:
		return hex.EncodeToString(v.BytesValue)
	case *commonv1.AnyValue_ArrayValue:
		if v.ArrayValue == nil {
			return []interface{}{}
		}
		arr := make([]interface{}, 0, len(v.ArrayValue.Values))
		for _, item := range v.ArrayValue.Values {
			arr = append(arr, extractAnyValue(item))
		}
		return arr
	case *commonv1.AnyValue_KvlistValue:
		if v.KvlistValue == nil {
			return map[string]interface{}{}
		}
		return flattenAttributes(v.KvlistValue.Values)
	default:
		return nil
	}
}

// bytesToHex converts a byte array (trace ID or span ID) to hex string
func bytesToHex(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hex.EncodeToString(b)
}

func searchTagValuesV2ResponseToSimplifiedJSON(t *tempopb.SearchTagValuesV2Response) (string, error) {
	// Group values by type
	valuesByType := make(map[string][]string)
	for _, tv := range t.TagValues {
		valuesByType[tv.Type] = append(valuesByType[tv.Type], tv.Value)
	}

	// Build simplified structure
	simplified := map[string]interface{}{
		"tagValues": valuesByType,
	}

	// Add metrics if present
	if t.Metrics != nil {
		metrics := make(map[string]interface{})
		if t.Metrics.InspectedBytes > 0 {
			metrics["inspectedBytes"] = t.Metrics.InspectedBytes
		}
		if t.Metrics.TotalJobs > 0 {
			metrics["totalJobs"] = t.Metrics.TotalJobs
		}
		if t.Metrics.CompletedJobs > 0 {
			metrics["completedJobs"] = t.Metrics.CompletedJobs
		}
		if t.Metrics.TotalBlocks > 0 {
			metrics["totalBlocks"] = t.Metrics.TotalBlocks
		}
		if t.Metrics.TotalBlockBytes > 0 {
			metrics["totalBlockBytes"] = t.Metrics.TotalBlockBytes
		}
		if len(metrics) > 0 {
			simplified["metrics"] = metrics
		}
	}

	data, err := json.Marshal(simplified)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
