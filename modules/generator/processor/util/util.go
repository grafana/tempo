package util

import (
	"slices"

	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
	semconv "go.opentelemetry.io/collector/semconv/v1.9.0"
)

func FindServiceName(attributes []*v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(semconv.AttributeServiceName, attributes)
}

func FindServiceNamespace(attributes []*v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(semconv.AttributeServiceNamespace, attributes)
}

func FindInstanceID(attributes []*v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(semconv.AttributeServiceInstanceID, attributes)
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
		if key != "service.name" && key != "service.namespace" && key != "service.instance.id" && !slices.Contains(exclude, key) {
			keys = append(keys, key)
			values = append(values, value)
		}
	}

	return keys, values
}
