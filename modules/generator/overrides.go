package generator

import (
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/overrides"
)

type metricsGeneratorOverrides interface {
	registry.Overrides

	MetricsGeneratorProcessors(userID string) map[string]struct{}
	MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64
	MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string
	MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID string) []string
	MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID string) []float64
	MetricsGeneratorProcessorSpanMetricsDimensions(userID string) []string
	MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID string) map[string]bool
}

var _ metricsGeneratorOverrides = (*overrides.Overrides)(nil)
