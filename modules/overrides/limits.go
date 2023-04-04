package overrides

import (
	"flag"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

const (
	// LocalIngestionRateStrategy indicates that this limit can be evaluated in local terms only
	LocalIngestionRateStrategy = "local"
	// GlobalIngestionRateStrategy indicates that an attempt should be made to consider this limit across the entire Tempo cluster
	GlobalIngestionRateStrategy = "global"

	// ErrorPrefixLiveTracesExceeded is used to flag batches from the ingester that were rejected b/c they had too many traces
	ErrorPrefixLiveTracesExceeded = "LIVE_TRACES_EXCEEDED:"
	// ErrorPrefixTraceTooLarge is used to flag batches from the ingester that were rejected b/c they exceeded the single trace limit
	ErrorPrefixTraceTooLarge = "TRACE_TOO_LARGE:"
	// ErrorPrefixRateLimited is used to flag batches that have exceeded the spans/second of the tenant
	ErrorPrefixRateLimited = "RATE_LIMITED:"

	// metrics
	MetricMaxLocalTracesPerUser           = "max_local_traces_per_user"
	MetricMaxGlobalTracesPerUser          = "max_global_traces_per_user"
	MetricMaxBytesPerTrace                = "max_bytes_per_trace"
	MetricMaxBytesPerTagValuesQuery       = "max_bytes_per_tag_values_query"
	MetricIngestionRateLimitBytes         = "ingestion_rate_limit_bytes"
	MetricIngestionBurstSizeBytes         = "ingestion_burst_size_bytes"
	MetricBlockRetention                  = "block_retention"
	MetricMetricsGeneratorMaxActiveSeries = "metrics_generator_max_active_series"
)

var (
	metricLimitsDesc = prometheus.NewDesc(
		"tempo_limits_defaults",
		"Default resource limits",
		[]string{"limit_name"},
		nil,
	)
)

// Limits describe all the limits for users; can be used to describe global default
// limits via flags, or per-user limits via yaml config.
type Limits struct {
	// Distributor enforced limits.
	IngestionRateStrategy   string `yaml:"ingestion_rate_strategy" json:"ingestion_rate_strategy"`
	IngestionRateLimitBytes int    `yaml:"ingestion_rate_limit_bytes" json:"ingestion_rate_limit_bytes"`
	IngestionBurstSizeBytes int    `yaml:"ingestion_burst_size_bytes" json:"ingestion_burst_size_bytes"`

	// Ingester enforced limits.
	MaxLocalTracesPerUser  int `yaml:"max_traces_per_user" json:"max_traces_per_user"`
	MaxGlobalTracesPerUser int `yaml:"max_global_traces_per_user" json:"max_global_traces_per_user"`

	// Forwarders
	Forwarders []string `yaml:"forwarders" json:"forwarders"`

	// Metrics-generator config
	MetricsGeneratorRingSize                                int             `yaml:"metrics_generator_ring_size" json:"metrics_generator_ring_size"`
	MetricsGeneratorProcessors                              ListToMap       `yaml:"metrics_generator_processors" json:"metrics_generator_processors"`
	MetricsGeneratorMaxActiveSeries                         uint32          `yaml:"metrics_generator_max_active_series" json:"metrics_generator_max_active_series"`
	MetricsGeneratorCollectionInterval                      time.Duration   `yaml:"metrics_generator_collection_interval" json:"metrics_generator_collection_interval"`
	MetricsGeneratorDisableCollection                       bool            `yaml:"metrics_generator_disable_collection" json:"metrics_generator_disable_collection"`
	MetricsGeneratorForwarderQueueSize                      int             `yaml:"metrics_generator_forwarder_queue_size" json:"metrics_generator_forwarder_queue_size"`
	MetricsGeneratorForwarderWorkers                        int             `yaml:"metrics_generator_forwarder_workers" json:"metrics_generator_forwarder_workers"`
	MetricsGeneratorProcessorServiceGraphsHistogramBuckets  []float64       `yaml:"metrics_generator_processor_service_graphs_histogram_buckets" json:"metrics_generator_processor_service_graphs_histogram_buckets"`
	MetricsGeneratorProcessorServiceGraphsDimensions        []string        `yaml:"metrics_generator_processor_service_graphs_dimensions" json:"metrics_generator_processor_service_graphs_dimensions"`
	MetricsGeneratorProcessorSpanMetricsHistogramBuckets    []float64       `yaml:"metrics_generator_processor_span_metrics_histogram_buckets" json:"metrics_generator_processor_span_metrics_histogram_buckets"`
	MetricsGeneratorProcessorSpanMetricsDimensions          []string        `yaml:"metrics_generator_processor_span_metrics_dimensions" json:"metrics_generator_processor_span_metrics_dimensions"`
	MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions map[string]bool `yaml:"metrics_generator_processor_span_metrics_intrinsic_dimensions" json:"metrics_generator_processor_span_metrics_intrinsic_dimensions"`

	// Compactor enforced limits.
	BlockRetention model.Duration `yaml:"block_retention" json:"block_retention"`

	// Querier and Ingester enforced limits.
	MaxBytesPerTagValuesQuery int `yaml:"max_bytes_per_tag_values_query" json:"max_bytes_per_tag_values_query"`

	// QueryFrontend enforced limits
	MaxSearchDuration model.Duration `yaml:"max_search_duration" json:"max_search_duration"`

	// MaxBytesPerTrace is enforced in the Ingester, Compactor, Querier (Search) and Serverless (Search). It
	//  is not used when doing a trace by id lookup.
	MaxBytesPerTrace int `yaml:"max_bytes_per_trace" json:"max_bytes_per_trace"`

	// Configuration for overrides, convenient if it goes here.
	PerTenantOverrideConfig string         `yaml:"per_tenant_override_config" json:"per_tenant_override_config"`
	PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period" json:"per_tenant_override_period"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (l *Limits) RegisterFlags(f *flag.FlagSet) {
	// Distributor Limits
	f.StringVar(&l.IngestionRateStrategy, "distributor.rate-limit-strategy", "local", "Whether the various ingestion rate limits should be applied individually to each distributor instance (local), or evenly shared across the cluster (global).")
	f.IntVar(&l.IngestionRateLimitBytes, "distributor.ingestion-rate-limit-bytes", 15e6, "Per-user ingestion rate limit in bytes per second.")
	f.IntVar(&l.IngestionBurstSizeBytes, "distributor.ingestion-burst-size-bytes", 20e6, "Per-user ingestion burst size in bytes. Should be set to the expected size (in bytes) of a single push request.")

	// Ingester limits
	f.IntVar(&l.MaxLocalTracesPerUser, "ingester.max-traces-per-user", 10e3, "Maximum number of active traces per user, per ingester. 0 to disable.")
	f.IntVar(&l.MaxGlobalTracesPerUser, "ingester.max-global-traces-per-user", 0, "Maximum number of active traces per user, across the cluster. 0 to disable.")
	f.IntVar(&l.MaxBytesPerTrace, "ingester.max-bytes-per-trace", 50e5, "Maximum size of a trace in bytes.  0 to disable.")

	// Querier limits
	f.IntVar(&l.MaxBytesPerTagValuesQuery, "querier.max-bytes-per-tag-values-query", 50e5, "Maximum size of response for a tag-values query. Used mainly to limit large the number of values associated with a particular tag")

	f.StringVar(&l.PerTenantOverrideConfig, "limits.per-user-override-config", "", "File name of per-user overrides.")
	_ = l.PerTenantOverridePeriod.Set("10s")
	f.Var(&l.PerTenantOverridePeriod, "limits.per-user-override-period", "Period with this to reload the overrides.")
}

func (l *Limits) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricLimitsDesc
}

func (l *Limits) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.MaxLocalTracesPerUser), MetricMaxLocalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.MaxGlobalTracesPerUser), MetricMaxGlobalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.MaxBytesPerTrace), MetricMaxBytesPerTrace)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.MaxBytesPerTagValuesQuery), MetricMaxBytesPerTagValuesQuery)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.IngestionRateLimitBytes), MetricIngestionRateLimitBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.IngestionBurstSizeBytes), MetricIngestionBurstSizeBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.BlockRetention), MetricBlockRetention)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.MetricsGeneratorMaxActiveSeries), MetricMetricsGeneratorMaxActiveSeries)
}
