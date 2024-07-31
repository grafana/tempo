package util

import (
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	v1_common "github.com/grafana/tempo/v2/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/v2/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/v2/pkg/util"
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

func GetSpanMultiplier(ratioKey string, span *v1.Span) float64 {
	spanMultiplier := 1.0
	if ratioKey != "" {
		for _, kv := range span.Attributes {
			if kv.Key == ratioKey {
				v := kv.Value.GetDoubleValue()
				if v > 0 {
					spanMultiplier = 1.0 / v
				}
			}
		}
	}
	return spanMultiplier
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

func GetTargetInfoAttributesValues(attributes []*v1_common.KeyValue, exclude []string) ([]string, []string) {
	keys := make([]string, 0)
	values := make([]string, 0)
	for _, attrs := range attributes {
		// ignoring job and instance
		key := attrs.Key
		value := tempo_util.StringifyAnyValue(attrs.Value)
		if key != "service.name" && key != "service.namespace" && key != "service.instance.id" && !Contains(key, exclude) {
			keys = append(keys, key)
			values = append(values, value)
		}
	}

	return keys, values
}

func Contains(key string, list []string) bool {
	for _, exclude := range list {
		if key == exclude {
			return true
		}
	}
	return false
}
