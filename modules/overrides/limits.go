package overrides

import (
	"flag"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
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
	MetricMaxBlocksPerTagValuesQuery      = "max_blocks_per_tag_values_query"
	MetricIngestionRateLimitBytes         = "ingestion_rate_limit_bytes"
	MetricIngestionBurstSizeBytes         = "ingestion_burst_size_bytes"
	MetricBlockRetention                  = "block_retention"
	MetricMetricsGeneratorMaxActiveSeries = "metrics_generator_max_active_series"
)

var metricLimitsDesc = prometheus.NewDesc(
	"tempo_limits_defaults",
	"Default resource limits",
	[]string{"limit_name"},
	nil,
)

type IngestionConfig struct {
	// Distributor enforced limits.
	RateStrategy   string `yaml:"rate_strategy,omitempty" json:"rate_strategy,omitempty"`
	RateLimitBytes int    `yaml:"rate_limit_bytes,omitempty" json:"rate_limit_bytes,omitempty"`
	BurstSizeBytes int    `yaml:"burst_size_bytes,omitempty" json:"burst_size_bytes,omitempty"`

	// Ingester enforced limits.
	MaxLocalTracesPerUser  int `yaml:"max_traces_per_user,omitempty" json:"max_traces_per_user,omitempty"`
	MaxGlobalTracesPerUser int `yaml:"max_global_traces_per_user,omitempty" json:"max_global_traces_per_user,omitempty"`
}

type ForwarderConfig struct {
	QueueSize int `yaml:"queue_size,omitempty" json:"queue_size,omitempty"`
	Workers   int `yaml:"workers,omitempty" json:"workers,omitempty"`
}

type ServiceGraphsConfig struct {
	HistogramBuckets         []float64 `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	Dimensions               []string  `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	PeerAttributes           []string  `yaml:"peer_attributes,omitempty" json:"peer_attributes,omitempty"`
	EnableClientServerPrefix bool      `yaml:"enable_client_server_prefix,omitempty" json:"enable_client_server_prefix,omitempty"`
}

type SpanMetricsConfig struct {
	HistogramBuckets    []float64                        `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	Dimensions          []string                         `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	IntrinsicDimensions map[string]bool                  `yaml:"intrinsic_dimensions,omitempty" json:"intrinsic_dimensions,omitempty"`
	FilterPolicies      []filterconfig.FilterPolicy      `yaml:"filter_policies,omitempty" json:"filter_policies,omitempty"`
	DimensionMappings   []sharedconfig.DimensionMappings `yaml:"dimension_mappings,omitempty" json:"dimension_mapings,omitempty"`
	EnableTargetInfo    bool                             `yaml:"enable_target_info,omitempty" json:"enable_target_info,omitempty"`
}

type LocalBlocksConfig struct {
	MaxLiveTraces        uint64        `yaml:"max_live_traces,omitempty" json:"max_live_traces,omitempty"`
	MaxBlockDuration     time.Duration `yaml:"max_block_duration,omitempty" json:"max_block_duration,omitempty"`
	MaxBlockBytes        uint64        `yaml:"max_block_bytes,omitempty" json:"max_block_bytes,omitempty"`
	FlushCheckPeriod     time.Duration `yaml:"flush_check_period,omitempty" json:"flush_check_period,omitempty"`
	TraceIdlePeriod      time.Duration `yaml:"trace_idle_period,omitempty" json:"trace_idle_period,omitempty"`
	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout,omitempty" json:"complete_block_timeout,omitempty"`
}

type ProcessorConfig struct {
	ServiceGraphs ServiceGraphsConfig `yaml:"service_graphs,omitempty" json:"service_graphs,omitempty"`

	SpanMetrics SpanMetricsConfig `yaml:"span_metrics,omitempty" json:"span_metrics,omitempty"`

	LocalBlocks LocalBlocksConfig `yaml:"local_blocks,omitempty" json:"local_blocks,omitempty"`
}

type MetricsGeneratorConfig struct {
	RingSize           int           `yaml:"ring_size,omitempty" json:"ring_size,omitempty"`
	Processors         ListToMap     `yaml:"processors,omitempty" json:"processors,omitempty"`
	MaxActiveSeries    uint32        `yaml:"max_active_series,omitempty" json:"max_active_series,omitempty"`
	CollectionInterval time.Duration `yaml:"collection_interval,omitempty" json:"collection_interval,omitempty"`
	DisableCollection  bool          `yaml:"disable_collection,omitempty" json:"disable_collection,omitempty"`

	Forwarder ForwarderConfig `yaml:"forwarder,omitempty" json:"forwarder,omitempty"`

	Processor ProcessorConfig `yaml:"processor,omitempty" json:"processor,omitempty"`
}

type ReadConfig struct {
	// Querier and Ingester enforced limits.
	MaxBytesPerTagValuesQuery  int `yaml:"max_bytes_per_tag_values_query,omitempty" json:"max_bytes_per_tag_values_query,omitempty"`
	MaxBlocksPerTagValuesQuery int `yaml:"max_blocks_per_tag_values_query,omitempty" json:"max_blocks_per_tag_values_query,omitempty"`

	// QueryFrontend enforced limits
	MaxSearchDuration model.Duration `yaml:"max_search_duration,omitempty" json:"max_search_duration,omitempty"`
}

type CompactionConfig struct {
	// Compactor enforced limits.
	BlockRetention model.Duration `yaml:"block_retention,omitempty" json:"block_retention,omitempty"`
}

// TODO: Ingestion limit instead?
type GlobalLimitsConfig struct {
	// MaxBytesPerTrace is enforced in the Ingester, Compactor, Querier (Search) and Serverless (Search). It
	//  is not used when doing a trace by id lookup.
	MaxBytesPerTrace int `yaml:"max_bytes_per_trace,omitempty" json:"max_bytes_per_trace,omitempty"`
}

type Limits struct {
	// Ingestion enforced limits.
	Ingestion IngestionConfig `yaml:"ingestion,omitempty" json:"ingestion,omitempty"`
	// Read enforced limits.
	Read ReadConfig `yaml:"read,omitempty" json:"read,omitempty"`
	// Compaction enforced limits.
	Compaction CompactionConfig `yaml:"compaction,omitempty" json:"compaction,omitempty"`
	// MetricsGenerator enforced limits.
	MetricsGenerator MetricsGeneratorConfig `yaml:"metrics_generator,omitempty" json:"metrics_generator,omitempty"`
	// Forwarders
	Forwarders []string `yaml:"forwarders,omitempty" json:"forwarders,omitempty"`
	// Global enforced limits.
	Global GlobalLimitsConfig `yaml:"global,omitempty" json:"global,omitempty"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (l *Limits) RegisterFlags(f *flag.FlagSet) {
	// Distributor LegacyLimits
	f.StringVar(&l.Ingestion.RateStrategy, "distributor.rate-limit-strategy", "local", "Whether the various ingestion rate limits should be applied individually to each distributor instance (local), or evenly shared across the cluster (global).")
	f.IntVar(&l.Ingestion.RateLimitBytes, "distributor.ingestion-rate-limit-bytes", 15e6, "Per-user ingestion rate limit in bytes per second.")
	f.IntVar(&l.Ingestion.BurstSizeBytes, "distributor.ingestion-burst-size-bytes", 20e6, "Per-user ingestion burst size in bytes. Should be set to the expected size (in bytes) of a single push request.")

	// Ingester limits
	f.IntVar(&l.Ingestion.MaxLocalTracesPerUser, "ingester.max-traces-per-user", 10e3, "Maximum number of active traces per user, per ingester. 0 to disable.")
	f.IntVar(&l.Ingestion.MaxGlobalTracesPerUser, "ingester.max-global-traces-per-user", 0, "Maximum number of active traces per user, across the cluster. 0 to disable.")
	f.IntVar(&l.Global.MaxBytesPerTrace, "ingester.max-bytes-per-trace", 50e5, "Maximum size of a trace in bytes.  0 to disable.")

	// Querier limits
	f.IntVar(&l.Read.MaxBytesPerTagValuesQuery, "querier.max-bytes-per-tag-values-query", 50e5, "Maximum size of response for a tag-values query. Used mainly to limit large the number of values associated with a particular tag")
	f.IntVar(&l.Read.MaxBlocksPerTagValuesQuery, "querier.max-blocks-per-tag-values-query", 0, "Maximum number of blocks to query for a tag-values query. 0 to disable.")
}

func (l *Limits) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricLimitsDesc
}

func (l *Limits) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Ingestion.MaxLocalTracesPerUser), MetricMaxLocalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Ingestion.MaxGlobalTracesPerUser), MetricMaxGlobalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Ingestion.RateLimitBytes), MetricIngestionRateLimitBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Ingestion.BurstSizeBytes), MetricIngestionBurstSizeBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Read.MaxBytesPerTagValuesQuery), MetricMaxBytesPerTagValuesQuery)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Read.MaxBlocksPerTagValuesQuery), MetricMaxBlocksPerTagValuesQuery)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Global.MaxBytesPerTrace), MetricMaxBytesPerTrace)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.Compaction.BlockRetention), MetricBlockRetention)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(l.MetricsGenerator.MaxActiveSeries), MetricMetricsGeneratorMaxActiveSeries)
}
