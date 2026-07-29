package util

import (
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	"github.com/grafana/tempo/pkg/sampling"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

func FindServiceName(attributes []v1_common.KeyValue) (string, bool) {
	return FindAttributeValue(string(semconv.ServiceNameKey), attributes)
}

// FindServiceLabels extracts the service.name, job, and service.instance.id
// label values from resource attributes in a single pass. The job value is
// "<service.namespace>/<service.name>" when the namespace is present and just
// the service name otherwise; it is empty when service.name is absent. The
// first occurrence of each attribute wins.
func FindServiceLabels(attributes []v1_common.KeyValue) (svcName, jobName, instanceID string) {
	var (
		namespace       string
		foundSvcName    bool
		foundNamespace  bool
		foundInstanceID bool
	)

	for i := range attributes {
		kv := &attributes[i]
		switch kv.Key {
		case string(semconv.ServiceNameKey):
			if !foundSvcName {
				svcName = tempo_util.StringifyAnyValue(kv.Value)
				foundSvcName = true
			}
		case string(semconv.ServiceNamespaceKey):
			if !foundNamespace {
				namespace = tempo_util.StringifyAnyValue(kv.Value)
				foundNamespace = true
			}
		case string(semconv.ServiceInstanceIDKey):
			if !foundInstanceID {
				instanceID = tempo_util.StringifyAnyValue(kv.Value)
				foundInstanceID = true
			}
		}
	}

	if svcName == "" {
		return svcName, "", instanceID
	}
	if namespace == "" {
		return svcName, svcName, instanceID
	}
	return svcName, namespace + "/" + svcName, instanceID
}

func FindAttributeValue(key string, attributes ...[]v1_common.KeyValue) (string, bool) {
	for _, attrs := range attributes {
		for i := range attrs {
			if key == attrs[i].Key {
				return tempo_util.StringifyAnyValue(attrs[i].Value), true
			}
		}
	}
	return "", false
}

func GetSpanMultiplier(ratioKey string, span *v1.Span, rs *v1_resource.Resource, enableTraceState bool) float64 {
	if enableTraceState {
		if m := sampling.MultiplierFromTraceState(span.GetTraceState()); m > 0 {
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
