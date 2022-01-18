package util

import (
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
)

func GetServiceName(rs *v1_resource.Resource) string {
	for _, attr := range rs.Attributes {
		if attr.Key == semconv.AttributeServiceName {
			return attr.Value.GetStringValue()
		}
	}

	return ""
}
