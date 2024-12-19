package util

import (
	"slices"

	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"

	"github.com/prometheus/prometheus/util/strutil"
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

func GetTargetInfoAttributesValues(keys, values *[]string, attributes []*v1_common.KeyValue, exclude, intrinsicLabels []string) {
	// TODO allocate with known length, or take new params for existing buffers
	*keys = (*keys)[:0]
	*values = (*values)[:0]
	for _, attrs := range attributes {
		// ignoring job and instance
		key := attrs.Key
		if key != "service.name" && key != "service.namespace" && key != "service.instance.id" && !Contains(key, exclude) {
			*keys = append(*keys, SanitizeLabelNameWithCollisions(key, intrinsicLabels))
			value := tempo_util.StringifyAnyValue(attrs.Value)
			*values = append(*values, value)
		}
	}
}

func SanitizeLabelNameWithCollisions(name string, dimensions []string) string {
	sanitized := strutil.SanitizeLabelName(name)

	// check if same label as intrinsics
	if slices.Contains(dimensions, sanitized) {
		return "__" + sanitized
	}

	return sanitized
}

func Contains(key string, list []string) bool {
	for _, exclude := range list {
		if key == exclude {
			return true
		}
	}
	return false
}
