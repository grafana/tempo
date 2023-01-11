package generator

import "time"

type mockOverrides struct {
	processors                     map[string]struct{}
	serviceGraphsHistogramBuckets  []float64
	serviceGraphsDimensions        []string
	spanMetricsHistogramBuckets    []float64
	spanMetricsDimensions          []string
	spanMetricsIntrinsicDimensions map[string]bool
}

var _ metricsGeneratorOverrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(userID string) uint32 {
	return 0
}

func (m *mockOverrides) MetricsGeneratorCollectionInterval(userID string) time.Duration {
	return 15 * time.Second
}

func (m *mockOverrides) MetricsGeneratorProcessors(userID string) map[string]struct{} {
	return m.processors
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(userID string) bool {
	return false
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64 {
	return m.serviceGraphsHistogramBuckets
}

func (m *mockOverrides) MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string {
	return m.serviceGraphsDimensions
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID string) []float64 {
	return m.spanMetricsHistogramBuckets
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsDimensions(userID string) []string {
	return m.spanMetricsDimensions
}

func (m *mockOverrides) MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID string) map[string]bool {
	return m.spanMetricsIntrinsicDimensions
}
