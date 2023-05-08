package generator

import (
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/overrides"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

type metricsGeneratorOverrides interface {
	registry.Overrides

	MetricsGeneratorProcessors(userID string) map[string]struct{}
	MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64
	MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string
	MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID string) []float64
	MetricsGeneratorProcessorSpanMetricsDimensions(userID string) []string
	MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID string) map[string]bool
	MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID string) []filterconfig.FilterPolicy
	MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(userID string) uint64
}

var _ metricsGeneratorOverrides = (*overrides.Overrides)(nil)
