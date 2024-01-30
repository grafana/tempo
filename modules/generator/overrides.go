package generator

import (
	"time"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/tempodb/backend"
)

type metricsGeneratorOverrides interface {
	registry.Overrides
	storage.Overrides

	MetricsGeneratorIngestionSlack(userID string) time.Duration
	MetricsGeneratorProcessors(userID string) map[string]struct{}
	MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64
	MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string
	MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID string) []string
	MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID string) []float64
	MetricsGeneratorProcessorSpanMetricsDimensions(userID string) []string
	MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID string) map[string]bool
	MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID string) []filterconfig.FilterPolicy
	MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(userID string) uint64
	MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(userID string) time.Duration
	MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(userID string) uint64
	MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(userID string) time.Duration
	MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(userID string) time.Duration
	MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(userID string) time.Duration
	MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID string) []sharedconfig.DimensionMappings
	MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID string) bool
	MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID string) bool
	MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID string) []string
	DedicatedColumns(userID string) backend.DedicatedColumns
	MaxBytesPerTrace(userID string) int
}

var _ metricsGeneratorOverrides = (overrides.Interface)(nil)
