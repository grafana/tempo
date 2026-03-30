package util

import (
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

func FindServiceName(attributes []*v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(string(semconv.ServiceNameKey), attributes)
}

func FindServiceNamespace(attributes []*v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(string(semconv.ServiceNamespaceKey), attributes)
}

func FindInstanceID(attributes []*v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(string(semconv.ServiceInstanceIDKey), attributes)
}

func FindAttributeValue(key string, attributes ...[]*v1_common.KeyValue) (string, bool) {
	for _, attrs := range attributes {
		for _, kv := range attrs {
			if key == kv.Key {
				return tempo_util.StringifyAnyValue(kv.Value), true
			}
		}
	}
	return "", false
}

func GetSpanMultiplier(ratioKey string, span *v1.Span, rs *v1_resource.Resource, enableTraceState bool) float64 {
	if enableTraceState {
		if m := getSpanMultiplierFromTraceState(span.GetTraceState()); m > 0 {
			return m
		}
	}

	if ratioKey != "" {
		for _, kv := range span.Attributes {
			if kv.Key == ratioKey {
				v := kv.Value.GetDoubleValue()
				if v > 0 {
					return 1.0 / v
				}
			}
		}
		for _, kv := range rs.Attributes {
			if kv.Key == ratioKey {
				v := kv.Value.GetDoubleValue()
				if v > 0 {
					return 1.0 / v
				}
			}
		}
	}
	return 1.0
}

// getSpanMultiplierFromTraceState extracts a span multiplier from the W3C tracestate
// OTel probability sampling threshold.
// Returns 0 if the tracestate is empty, invalid, or has no OTel sampling data.
func getSpanMultiplierFromTraceState(traceState string) float64 {
	// Manual parsing of trace state is about twice as fast
	// sampling.NewW3CTraceState as we only care about the ot key.
	ot := extractOpenTelemetryTraceState(traceState)
	if ot == "" {
		return 0
	}

	otts, err := sampling.NewOpenTelemetryTraceState(ot)
	if err != nil {
		return 0
	}

	return otts.AdjustedCount()
}

// extractOpenTelemetryTraceState parses the tracestate for the ot vendor key
// and returns the value of the key (or empty if it does not exist). It is
// about twice as fast as `sampling.NewW3CTraceState` and does no allocations.
func extractOpenTelemetryTraceState(traceState string) string {
	// tracestate is formatted like vendor1=value1,vendor2=value2. See
	// https://www.w3.org/TR/trace-context/#list.
	for {
		// Trim any optional white space surrounding vendor elements.
		traceState = strings.TrimSpace(traceState)
		nextComma := strings.IndexByte(traceState, ',')
		if strings.HasPrefix(traceState, "ot=") {
			end := len(traceState)
			if nextComma > 0 {
				end = nextComma
			}
			return traceState[3:end]
		}

		if nextComma == -1 {
			return ""
		}
		traceState = strings.TrimSpace(traceState[nextComma+1:])
	}
}

func GetJobValue(attributes []*v1_common.KeyValue) string {
	svName, _ := FindServiceName(attributes)
	// if service name is not present, consider job value empty
	if svName == "" {
		return ""
	}
	namespace, _ := FindServiceNamespace(attributes)
	if namespace == "" {
		return svName
	}
	return namespace + "/" + svName
}
