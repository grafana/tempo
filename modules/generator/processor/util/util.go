package util

import (
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

func GetSpanMultiplier(ratioKey string, span *v1.Span, rs *v1_resource.Resource) float64 {
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

func GetJobValue(attributes []*v1_common.KeyValue) string {
	svName, _ := FindServiceName(attributes)
	namespace, _ := FindServiceNamespace(attributes)

	// if service name is not present, consider job value empty
	if svName == "" {
		return ""
	} else if namespace != "" {
		namespace += "/"
	}

	return namespace + svName
}
