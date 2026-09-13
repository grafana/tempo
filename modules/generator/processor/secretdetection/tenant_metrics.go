package secretdetection

import (
	"github.com/prometheus/prometheus/model/labels"

	"github.com/grafana/tempo/modules/generator/registry"
)

const (
	tenantMetricDetections = "traces_secret_detections_total"
	tenantMetricPushes     = "traces_secret_detection_pushes_total"
)

// TenantMetrics writes bounded detection telemetry through the tenant's metrics-generator registry.
type TenantMetrics struct {
	detections registry.Counter
	pushes     registry.Counter

	detectionLabels map[string]labels.Labels
	pushLabels      labels.Labels
}

// NewTenantMetrics registers tenant-isolated detection metrics once per metrics-generator instance.
func NewTenantMetrics(reg registry.Registry, sourceStream string) *TenantMetrics {
	if sourceStream == "" {
		sourceStream = "tempo-ingest"
	}
	m := &TenantMetrics{
		detections:      reg.NewCounter(tenantMetricDetections),
		pushes:          reg.NewCounter(tenantMetricPushes),
		detectionLabels: make(map[string]labels.Labels, len(allScopes)),
		pushLabels:      labels.FromStrings("source_stream", sourceStream),
	}
	for _, scope := range allScopes {
		m.detectionLabels[scope] = labels.FromStrings("attribute_scope", scope, "source_stream", sourceStream)
	}
	return m
}

func (m *TenantMetrics) recordPush() {
	m.pushes.Inc(m.pushLabels, 1)
}

func (m *TenantMetrics) recordDetection(scope string) {
	m.detections.Inc(m.detectionLabels[scope], 1)
}
