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

// LLM-optimized response structures with JSON tags

type LLMTraceByIDResponse struct {
	Trace   LLMTrace    `json:"trace"`
	Metrics *LLMMetrics `json:"metrics,omitempty"`
}

type LLMTrace struct {
	TraceID  string       `json:"traceId"`
	Services []LLMService `json:"services"`
}

type LLMService struct {
	ServiceName string                 `json:"serviceName,omitempty"`
	Resource    map[string]interface{} `json:"resource,omitempty"`
	Scopes      []LLMScope             `json:"scopes,omitempty"`
}

type LLMScope struct {
	Name    string    `json:"name,omitempty"`
	Version string    `json:"version,omitempty"`
	Spans   []LLMSpan `json:"spans,omitempty"`
}

type LLMSpan struct {
	SpanID            string                 `json:"spanId"`
	Name              string                 `json:"name"`
	ParentSpanID      string                 `json:"parentSpanId,omitempty"`
	Kind              string                 `json:"kind,omitempty"`
	StartTimeUnixNano string                 `json:"startTimeUnixNano,omitempty"`
	EndTimeUnixNano   string                 `json:"endTimeUnixNano,omitempty"`
	DurationMs        float64                `json:"durationMs,omitempty"`
	Attributes        map[string]interface{} `json:"attributes,omitempty"`
	Events            []LLMEvent             `json:"events,omitempty"`
	Links             []LLMLink              `json:"links,omitempty"`
	Status            LLMStatus              `json:"status"`
}

type LLMEvent struct {
	Name         string                 `json:"name"`
	TimeUnixNano string                 `json:"timeUnixNano,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

type LLMLink struct {
	TraceID    string                 `json:"traceId"`
	SpanID     string                 `json:"spanId"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type LLMStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LLMMetrics struct {
	InspectedBytes  uint64 `json:"inspectedBytes,omitempty"`
	TotalJobs       uint32 `json:"totalJobs,omitempty"`
	CompletedJobs   uint32 `json:"completedJobs,omitempty"`
	TotalBlocks     uint32 `json:"totalBlocks,omitempty"`
	TotalBlockBytes uint64 `json:"totalBlockBytes,omitempty"`
}

type LLMSearchTagValuesV2Response struct {
	TagValues map[string][]string `json:"tagValues"`
	Metrics   *LLMMetrics         `json:"metrics,omitempty"`
}

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
	services := make([]LLMService, 0, len(t.Trace.ResourceSpans))
	for _, rs := range t.Trace.ResourceSpans {
		service := LLMService{}

		// Extract service name and flatten resource attributes
		resourceAttrs := flattenAttributes(rs.Resource.GetAttributes())
		if serviceName, ok := resourceAttrs["service.name"]; ok {
			if name, ok := serviceName.(string); ok {
				service.ServiceName = name
			}
			delete(resourceAttrs, "service.name")
		}
		if len(resourceAttrs) > 0 {
			service.Resource = resourceAttrs
		}

		// Build scopes array
		scopes := make([]LLMScope, 0, len(rs.ScopeSpans))
		for _, ss := range rs.ScopeSpans {
			scope := LLMScope{}
			if ss.Scope != nil {
				scope.Name = ss.Scope.Name
				scope.Version = ss.Scope.Version
			}

			// Build spans array
			spans := make([]LLMSpan, 0, len(ss.Spans))
			for _, span := range ss.Spans {
				simplifiedSpan := simplifySpan(span)
				spans = append(spans, simplifiedSpan)
			}

			if len(spans) > 0 {
				scope.Spans = spans
			}
			scopes = append(scopes, scope)
		}

		if len(scopes) > 0 {
			service.Scopes = scopes
		}
		services = append(services, service)
	}

	// Build final structure
	result := LLMTraceByIDResponse{
		Trace: LLMTrace{
			TraceID:  traceID,
			Services: services,
		},
	}

	// Add metrics if present
	if t.Metrics != nil && t.Metrics.InspectedBytes > 0 {
		result.Metrics = &LLMMetrics{
			InspectedBytes: t.Metrics.InspectedBytes,
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func simplifySpan(span *tracev1.Span) LLMSpan {
	result := LLMSpan{
		SpanID: bytesToHex(span.SpanId),
		Name:   span.Name,
	}

	if len(span.ParentSpanId) > 0 {
		result.ParentSpanID = bytesToHex(span.ParentSpanId)
	}

	// Add kind if not unspecified
	if span.Kind != tracev1.Span_SPAN_KIND_UNSPECIFIED {
		result.Kind = span.Kind.String()
	}

	// Add timestamps
	if span.StartTimeUnixNano > 0 {
		result.StartTimeUnixNano = fmt.Sprintf("%d", span.StartTimeUnixNano)
	}
	if span.EndTimeUnixNano > 0 {
		result.EndTimeUnixNano = fmt.Sprintf("%d", span.EndTimeUnixNano)
	}

	// Calculate duration in milliseconds
	if span.EndTimeUnixNano > 0 && span.StartTimeUnixNano > 0 {
		durationNano := span.EndTimeUnixNano - span.StartTimeUnixNano
		durationMs := float64(durationNano) / 1_000_000.0
		result.DurationMs = durationMs
	}

	// Flatten attributes
	if len(span.Attributes) > 0 {
		result.Attributes = flattenAttributes(span.Attributes)
	}

	// Add events if present
	if len(span.Events) > 0 {
		events := make([]LLMEvent, 0, len(span.Events))
		for _, event := range span.Events {
			e := LLMEvent{
				Name: event.Name,
			}
			if event.TimeUnixNano > 0 {
				e.TimeUnixNano = fmt.Sprintf("%d", event.TimeUnixNano)
			}
			if len(event.Attributes) > 0 {
				e.Attributes = flattenAttributes(event.Attributes)
			}
			events = append(events, e)
		}
		result.Events = events
	}

	// Add links if present
	if len(span.Links) > 0 {
		links := make([]LLMLink, 0, len(span.Links))
		for _, link := range span.Links {
			l := LLMLink{
				TraceID: bytesToHex(link.TraceId),
				SpanID:  bytesToHex(link.SpanId),
			}
			if len(link.Attributes) > 0 {
				l.Attributes = flattenAttributes(link.Attributes)
			}
			links = append(links, l)
		}
		result.Links = links
	}

	// Add status
	if span.Status != nil && span.Status.Code != tracev1.Status_STATUS_CODE_UNSET {
		result.Status = LLMStatus{
			Code:    span.Status.Code.String(),
			Message: span.Status.Message,
		}
	} else {
		// Default empty status
		result.Status = LLMStatus{
			Code:    "STATUS_CODE_UNSET",
			Message: "",
		}
	}

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
	result := LLMSearchTagValuesV2Response{
		TagValues: valuesByType,
	}

	// Add metrics if present
	if t.Metrics != nil {
		hasMetrics := t.Metrics.InspectedBytes > 0 || t.Metrics.TotalJobs > 0 ||
			t.Metrics.CompletedJobs > 0 || t.Metrics.TotalBlocks > 0 || t.Metrics.TotalBlockBytes > 0

		if hasMetrics {
			result.Metrics = &LLMMetrics{
				InspectedBytes:  t.Metrics.InspectedBytes,
				TotalJobs:       t.Metrics.TotalJobs,
				CompletedJobs:   t.Metrics.CompletedJobs,
				TotalBlocks:     t.Metrics.TotalBlocks,
				TotalBlockBytes: t.Metrics.TotalBlockBytes,
			}
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
