package generator

import (
	"time"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/tempodb/backend"
)

type mockOverrides struct {
	processors                                         map[string]struct{}
	nativeHistogramMaxBucketNumber                     uint32
	nativeHistogramBucketFactor                        float64
	nativeHistogramMinResetDuration                    time.Duration
	serviceGraphsHistogramBuckets                      []float64
	serviceGraphsDimensions                            []string
	serviceGraphsPeerAttributes                        []string
	serviceGraphsFilterPolicies                        []filterconfig.FilterPolicy
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
	spanMetricsEnableInstanceLabel                     *bool
	dedicatedColumns                                   backend.DedicatedColumns
	maxLocalTraces                                     int
	maxBytesPerTrace                                   int
	unsafeQueryHints                                   bool
	nativeHistograms                                   histograms.HistogramMethod
	hostInfoHostIdentifiers                            []string
	hostInfoMetricName                                 string
	serviceGraphsSpanMultiplierKey                     string
	serviceGraphsEnableTraceStateSpanMultiplier        *bool
	spanMetricsSpanMultiplierKey                       string
	spanMetricsEnableTraceStateSpanMultiplier          *bool
	ingestionSlack                                     time.Duration
	collectionInterval                                 time.Duration
}

var _ metricsGeneratorOverrides = (*mockOverrides)(nil)

// MetricsGeneratorIngestionSlack mirrors the production overrides: it returns
// the raw value and leaves the zero-value fallback to the instance, which
// falls back to Config.MetricsIngestionSlack.
func (m *mockOverrides) MetricsGeneratorIngestionSlack(string) time.Duration {
	return m.ingestionSlack
}

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(string) uint32 {
	return 0
}

func (m *mockOverrides) MetricsGeneratorMaxActiveEntities(string) uint32 {
	return 0
}

// MetricsGeneratorCollectionInterval mirrors the production overrides: it
// returns the raw value and leaves the zero-value fallback to the registry,
// which falls back to registry.Config.CollectionInterval.
func (m *mockOverrides) MetricsGeneratorCollectionInterval(string) time.Duration {
	return m.collectionInterval
}

func (m *mockOverrides) MetricsGeneratorProcessors(string) map[string]struct{} {
	return m.processors
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(string) bool {
	return false
}

func (m *mockOverrides) MetricsGeneratorGenerateNativeHistograms(string) histograms.HistogramMethod {
	return m.nativeHistograms
}

func (m *mockOverrides) MetricsGeneratorTraceIDLabelName(string) string {
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

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsFilterPolicies(string) []filterconfig.FilterPolicy {
	return m.serviceGraphsFilterPolicies
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

// MetricsGeneratorProcessorSpanMetricsDimensionMappings controls custom dimension mapping
func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsDimensionMappings(string) []sharedconfig.DimensionMappings {
	return m.spanMetricsDimensionMappings
}

// The native histogram accessors fall back to the registered production
// defaults (modules/overrides/config.go) when unset. The registry feeds these
// values to prometheus without further defaulting, so returning a zero bucket
// factor would silently disable native (sparse) bucket observation.
func (m *mockOverrides) MetricsGeneratorNativeHistogramBucketFactor(string) float64 {
	if m.nativeHistogramBucketFactor != 0 {
		return m.nativeHistogramBucketFactor
	}
	return 1.1
}

func (m *mockOverrides) MetricsGeneratorNativeHistogramMaxBucketNumber(string) uint32 {
	if m.nativeHistogramMaxBucketNumber != 0 {
		return m.nativeHistogramMaxBucketNumber
	}
	return 100
}

func (m *mockOverrides) MetricsGeneratorNativeHistogramMinResetDuration(string) time.Duration {
	if m.nativeHistogramMinResetDuration != 0 {
		return m.nativeHistogramMinResetDuration
	}
	return 15 * time.Minute
}

func (m *mockOverrides) MetricsGeneratorSpanNameSanitization(string) string {
	return ""
}

func (m *mockOverrides) MetricsGeneratorMaxCardinalityPerLabel(string) uint64 {
	return 0
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

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(string) (bool, bool) {
	EnableInstanceLabel := m.spanMetricsEnableInstanceLabel
	if EnableInstanceLabel != nil {
		return *EnableInstanceLabel, true
	}
	return true, false // default to true if not set
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

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey(string) string {
	return m.serviceGraphsSpanMultiplierKey
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsEnableTraceStateSpanMultiplier(string) (bool, bool) {
	if m.serviceGraphsEnableTraceStateSpanMultiplier != nil {
		return *m.serviceGraphsEnableTraceStateSpanMultiplier, true
	}
	return false, false
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey(string) string {
	return m.spanMetricsSpanMultiplierKey
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsEnableTraceStateSpanMultiplier(string) (bool, bool) {
	if m.spanMetricsEnableTraceStateSpanMultiplier != nil {
		return *m.spanMetricsEnableTraceStateSpanMultiplier, true
	}
	return false, false
}
