package generator

import (
	"time"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/tempodb/backend"
)

type mockOverrides struct {
	processors                                         map[string]struct{}
	serviceGraphsHistogramBuckets                      []float64
	serviceGraphsDimensions                            []string
	serviceGraphsPeerAttributes                        []string
	serviceGraphsEnableClientServerPrefix              bool
	serviceGraphsEnableMessagingSystemLatencyHistogram *bool
	serviceGraphsEnableVirtualNodeLabel                *bool
	spanMetricsHistogramBuckets                        []float64
	spanMetricsDimensions                              []string
	spanMetricsIntrinsicDimensions                     map[string]bool
	spanMetricsFilterPolicies                          []filterconfig.FilterPolicy
	spanMetricsDimensionMappings                       []sharedconfig.DimensionMappings
	spanMetricsEnableTargetInfo                        *bool
	spanMetricsTargetInfoExcludedDimensions            []string
	localBlocksMaxLiveTraces                           uint64
	localBlocksMaxBlockDuration                        time.Duration
	localBlocksMaxBlockBytes                           uint64
	localBlocksFlushCheckPeriod                        time.Duration
	localBlocksTraceIdlePeriod                         time.Duration
	localBlocksCompleteBlockTimeout                    time.Duration
	dedicatedColumns                                   backend.DedicatedColumns
	maxLocalTraces                                     int
	maxBytesPerTrace                                   int
	unsafeQueryHints                                   bool
	nativeHistograms                                   overrides.HistogramMethod
	hostInfoHostIdentifiers                            []string
	hostInfoMetricName                                 string
}

var _ metricsGeneratorOverrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorIngestionSlack(string) time.Duration {
	return 30 * time.Second
}

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

func (m *mockOverrides) MetricsGeneratorGenerateNativeHistograms(string) overrides.HistogramMethod {
	return m.nativeHistograms
}

func (m *mockOverrides) MetricsGenerationTraceIDLabelName(string) string {
	return ""
}

func (m *mockOverrides) MetricsGeneratorRemoteWriteHeaders(string) map[string]string {
	return nil
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
func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(string) (bool, bool) {
	spanMetricsEnableTargetInfo := m.spanMetricsEnableTargetInfo
	if spanMetricsEnableTargetInfo != nil {
		return *spanMetricsEnableTargetInfo, true
	}
	return false, false
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(string) bool {
	return m.serviceGraphsEnableClientServerPrefix
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram(string) (bool, bool) {
	serviceGraphsEnableMessagingSystemLatencyHistogram := m.serviceGraphsEnableMessagingSystemLatencyHistogram
	if serviceGraphsEnableMessagingSystemLatencyHistogram != nil {
		return *serviceGraphsEnableMessagingSystemLatencyHistogram, true
	}
	return false, false
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel(string) (bool, bool) {
	serviceGraphsEnableVirtualNodeLabel := m.serviceGraphsEnableVirtualNodeLabel
	if serviceGraphsEnableVirtualNodeLabel != nil {
		return *serviceGraphsEnableVirtualNodeLabel, true
	}
	return false, false
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(string) []string {
	return m.spanMetricsTargetInfoExcludedDimensions
}

func (m *mockOverrides) DedicatedColumns(string) backend.DedicatedColumns {
	return m.dedicatedColumns
}

func (m *mockOverrides) MaxLocalTracesPerUser(string) int {
	return m.maxLocalTraces
}

func (m *mockOverrides) MaxBytesPerTrace(string) int {
	return m.maxBytesPerTrace
}

func (m *mockOverrides) UnsafeQueryHints(string) bool {
	return m.unsafeQueryHints
}

func (m *mockOverrides) MetricsGeneratorProcessorHostInfoHostIdentifiers(string) []string {
	return m.hostInfoHostIdentifiers
}

func (m *mockOverrides) MetricsGeneratorProcessorHostInfoMetricName(string) string {
	return m.hostInfoMetricName
}
