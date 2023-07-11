package generator

import (
	"time"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

type mockOverrides struct {
	processors                            map[string]struct{}
	serviceGraphsHistogramBuckets         []float64
	serviceGraphsDimensions               []string
	serviceGraphsPeerAttributes           []string
	serviceGraphsEnableClientServerPrefix bool
	spanMetricsHistogramBuckets           []float64
	spanMetricsDimensions                 []string
	spanMetricsIntrinsicDimensions        map[string]bool
	spanMetricsFilterPolicies             []filterconfig.FilterPolicy
	spanMetricsDimensionMappings          []sharedconfig.DimensionMappings
	spanMetricsEnableTargetInfo           bool
	localBlocksMaxLiveTraces              uint64
	localBlocksMaxBlockDuration           time.Duration
	localBlocksMaxBlockBytes              uint64
	localBlocksFlushCheckPeriod           time.Duration
	localBlocksTraceIdlePeriod            time.Duration
	localBlocksCompleteBlockTimeout       time.Duration
}

var _ metricsGeneratorOverrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(string) uint32 {
	return 0
}

func (m *mockOverrides) MetricsGeneratorCollectionInterval(string) time.Duration {
	return 15 * time.Second
}

func (m *mockOverrides) MetricsGeneratorProcessors(string) map[string]struct{} {
	return m.processors
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(string) bool {
	return false
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsHistogramBuckets(string) []float64 {
	return m.serviceGraphsHistogramBuckets
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsDimensions(string) []string {
	return m.serviceGraphsDimensions
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsPeerAttributes(string) []string {
	return m.serviceGraphsPeerAttributes
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsHistogramBuckets(string) []float64 {
	return m.spanMetricsHistogramBuckets
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsDimensions(string) []string {
	return m.spanMetricsDimensions
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(string) map[string]bool {
	return m.spanMetricsIntrinsicDimensions
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsFilterPolicies(string) []filterconfig.FilterPolicy {
	return m.spanMetricsFilterPolicies
}

func (m *mockOverrides) MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(string) uint64 {
	return m.localBlocksMaxLiveTraces
}

func (m *mockOverrides) MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(string) time.Duration {
	return m.localBlocksMaxBlockDuration
}

func (m *mockOverrides) MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(string) uint64 {
	return m.localBlocksMaxBlockBytes
}

func (m *mockOverrides) MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(string) time.Duration {
	return m.localBlocksTraceIdlePeriod
}

func (m *mockOverrides) MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(string) time.Duration {
	return m.localBlocksFlushCheckPeriod
}

func (m *mockOverrides) MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(string) time.Duration {
	return m.localBlocksCompleteBlockTimeout
}

// MetricsGeneratorProcessorSpanMetricsDimensionMappings controls custom dimension mapping
func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsDimensionMappings(string) []sharedconfig.DimensionMappings {
	return m.spanMetricsDimensionMappings
}

// MetricsGeneratorProcessorSpanMetricsEnableTargetInfo enables target_info metrics
func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(string) bool {
	return m.spanMetricsEnableTargetInfo
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(string) bool {
	return m.serviceGraphsEnableClientServerPrefix
}
