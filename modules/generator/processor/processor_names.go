package processor

// Span Metrics Subprocessor Names
// these are used when only specific metrics are wanted
const (
	SpanMetricsLatencyName = "span-metrics-latency"
	SpanMetricsCountName   = "span-metrics-count"
	SpanMetricsSizeName    = "span-metrics-size"
)

// Service Graphs Subprocessor Names
// these are used when only specific metrics are wanted
const (
	ServiceGraphsRequestName        = "service-graphs-request"
	ServiceGraphsLatencyName        = "service-graphs-latency"
	ServiceGraphsConnectionInfoName = "service-graphs-connection-info"
)

const (
	SpanMetricsName   = "span-metrics"
	ServiceGraphsName = "service-graphs"
	HostInfoName      = "host-info"
)
