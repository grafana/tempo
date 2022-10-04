package util

import (
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
	semconv "go.opentelemetry.io/collector/semconv/v1.9.0"
)

func FindServiceName(attributes []*v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(semconv.AttributeServiceName, attributes)
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
