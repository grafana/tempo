package overrides

import (
	"flag"
	"time"

	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/common/config"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"

	"github.com/prometheus/common/model"
)

type ConfigType string

const (
	ConfigTypeLegacy ConfigType = "legacy"
	ConfigTypeNew    ConfigType = "new"
)

const (
	// LocalIngestionRateStrategy indicates that this limit can be evaluated in local terms only
	LocalIngestionRateStrategy = "local"
	// GlobalIngestionRateStrategy indicates that an attempt should be made to consider this limit across the entire Tempo cluster
	GlobalIngestionRateStrategy = "global"

	// ErrorPrefixLiveTracesExceeded is used to flag batches from the ingester that were rejected b/c they had too many traces
	ErrorPrefixLiveTracesExceeded = "LIVE_TRACES_EXCEEDED"
	// ErrorPrefixTraceTooLarge is used to flag batches from the ingester that were rejected b/c they exceeded the single trace limit
	ErrorPrefixTraceTooLarge = "TRACE_TOO_LARGE"
	// ErrorPrefixRateLimited is used to flag batches that have exceeded the spans/second of the tenant
	ErrorPrefixRateLimited = "RATE_LIMITED"

	// metrics
	MetricMaxLocalTracesPerUser           = "max_local_traces_per_user"
	MetricMaxGlobalTracesPerUser          = "max_global_traces_per_user"
	MetricMaxBytesPerTrace                = "max_bytes_per_trace"
	MetricMaxBytesPerTagValuesQuery       = "max_bytes_per_tag_values_query"
	MetricMaxBlocksPerTagValuesQuery      = "max_blocks_per_tag_values_query"
	MetricIngestionRateLimitBytes         = "ingestion_rate_limit_bytes"
	MetricIngestionBurstSizeBytes         = "ingestion_burst_size_bytes"
	MetricBlockRetention                  = "block_retention"
	MetricCompactionWindow                = "compaction_window"
	MetricMetricsGeneratorMaxActiveSeries = "metrics_generator_max_active_series"
	MetricsGeneratorDryRunEnabled         = "metrics_generator_dry_run_enabled"
)

var metricLimitsDesc = prometheus.NewDesc(
	"tempo_limits_defaults",
	"Default resource limits",
	[]string{"limit_name"},
	nil,
)

type IngestionOverrides struct {
	// Distributor enforced limits.
	RateStrategy   string `yaml:"rate_strategy,omitempty" json:"rate_strategy,omitempty"`
	RateLimitBytes int    `yaml:"rate_limit_bytes,omitempty" json:"rate_limit_bytes,omitempty"`
	BurstSizeBytes int    `yaml:"burst_size_bytes,omitempty" json:"burst_size_bytes,omitempty"`

	// Ingester enforced limits.
	MaxLocalTracesPerUser  int `yaml:"max_traces_per_user,omitempty" json:"max_traces_per_user,omitempty"`
	MaxGlobalTracesPerUser int `yaml:"max_global_traces_per_user,omitempty" json:"max_global_traces_per_user,omitempty"`
}

type ForwarderOverrides struct {
	QueueSize int `yaml:"queue_size,omitempty" json:"queue_size,omitempty"`
	Workers   int `yaml:"workers,omitempty" json:"workers,omitempty"`
}

type ServiceGraphsOverrides struct {
	HistogramBuckets         []float64 `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	Dimensions               []string  `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	PeerAttributes           []string  `yaml:"peer_attributes,omitempty" json:"peer_attributes,omitempty"`
	EnableClientServerPrefix bool      `yaml:"enable_client_server_prefix,omitempty" json:"enable_client_server_prefix,omitempty"`
}

type SpanMetricsOverrides struct {
	HistogramBuckets             []float64                        `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	Dimensions                   []string                         `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	IntrinsicDimensions          map[string]bool                  `yaml:"intrinsic_dimensions,omitempty" json:"intrinsic_dimensions,omitempty"`
	FilterPolicies               []filterconfig.FilterPolicy      `yaml:"filter_policies,omitempty" json:"filter_policies,omitempty"`
	DimensionMappings            []sharedconfig.DimensionMappings `yaml:"dimension_mappings,omitempty" json:"dimension_mapings,omitempty"`
	EnableTargetInfo             bool                             `yaml:"enable_target_info,omitempty" json:"enable_target_info,omitempty"`
	TargetInfoExcludedDimensions []string                         `yaml:"target_info_excluded_dimensions,omitempty" json:"target_info_excluded_dimensions,omitempty"`
}

type LocalBlocksOverrides struct {
	MaxLiveTraces        uint64        `yaml:"max_live_traces,omitempty" json:"max_live_traces,omitempty"`
	MaxBlockDuration     time.Duration `yaml:"max_block_duration,omitempty" json:"max_block_duration,omitempty"`
	MaxBlockBytes        uint64        `yaml:"max_block_bytes,omitempty" json:"max_block_bytes,omitempty"`
	FlushCheckPeriod     time.Duration `yaml:"flush_check_period,omitempty" json:"flush_check_period,omitempty"`
	TraceIdlePeriod      time.Duration `yaml:"trace_idle_period,omitempty" json:"trace_idle_period,omitempty"`
	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout,omitempty" json:"complete_block_timeout,omitempty"`
}

type ProcessorOverrides struct {
	ServiceGraphs ServiceGraphsOverrides `yaml:"service_graphs,omitempty" json:"service_graphs,omitempty"`

	SpanMetrics SpanMetricsOverrides `yaml:"span_metrics,omitempty" json:"span_metrics,omitempty"`

	LocalBlocks LocalBlocksOverrides `yaml:"local_blocks,omitempty" json:"local_blocks,omitempty"`
}

type RemoteWriteHeaders map[string]config.Secret

func (h *RemoteWriteHeaders) toStringStringMap() map[string]string {
	if h == nil {
		return nil
	}

	headers := make(map[string]string)
	for k, v := range *h {
		headers[k] = string(v)
	}
	return headers
}

type MetricsGeneratorOverrides struct {
	RingSize           int                 `yaml:"ring_size,omitempty" json:"ring_size,omitempty"`
	Processors         listtomap.ListToMap `yaml:"processors,omitempty" json:"processors,omitempty"`
	MaxActiveSeries    uint32              `yaml:"max_active_series,omitempty" json:"max_active_series,omitempty"`
	CollectionInterval time.Duration       `yaml:"collection_interval,omitempty" json:"collection_interval,omitempty"`
	DisableCollection  bool                `yaml:"disable_collection,omitempty" json:"disable_collection,omitempty"`
	TraceIDLabelName   string              `yaml:"trace_id_label_name,omitempty" json:"trace_id_label_name,omitempty"`
	RemoteWriteHeaders RemoteWriteHeaders  `yaml:"remote_write_headers,omitempty" json:"remote_write_headers,omitempty"`

	Forwarder ForwarderOverrides `yaml:"forwarder,omitempty" json:"forwarder,omitempty"`

	Processor      ProcessorOverrides `yaml:"processor,omitempty" json:"processor,omitempty"`
	IngestionSlack time.Duration      `yaml:"ingestion_time_range_slack" json:"ingestion_time_range_slack"`
}

type ReadOverrides struct {
	// Querier and Ingester enforced overrides.
	MaxBytesPerTagValuesQuery  int `yaml:"max_bytes_per_tag_values_query,omitempty" json:"max_bytes_per_tag_values_query,omitempty"`
	MaxBlocksPerTagValuesQuery int `yaml:"max_blocks_per_tag_values_query,omitempty" json:"max_blocks_per_tag_values_query,omitempty"`

	// QueryFrontend enforced overrides
	MaxSearchDuration model.Duration `yaml:"max_search_duration,omitempty" json:"max_search_duration,omitempty"`
}

type CompactionOverrides struct {
	// Compactor enforced overrides.
	BlockRetention   model.Duration `yaml:"block_retention,omitempty" json:"block_retention,omitempty"`
	CompactionWindow model.Duration `yaml:"compaction_window,omitempty" json:"compaction_window,omitempty"`
}

type GlobalOverrides struct {
	// MaxBytesPerTrace is enforced in the Ingester, Compactor, Querier (Search) and Serverless (Search). It
	//  is not used when doing a trace by id lookup.
	MaxBytesPerTrace int `yaml:"max_bytes_per_trace,omitempty" json:"max_bytes_per_trace,omitempty"`
}

type StorageOverrides struct {
	// tempodb limits
	DedicatedColumns backend.DedicatedColumns `yaml:"parquet_dedicated_columns" json:"parquet_dedicated_columns"`
}

type Overrides struct {
	// Ingestion enforced overrides.
	Ingestion IngestionOverrides `yaml:"ingestion,omitempty" json:"ingestion,omitempty"`
	// Read enforced overrides.
	Read ReadOverrides `yaml:"read,omitempty" json:"read,omitempty"`
	// Compaction enforced overrides.
	Compaction CompactionOverrides `yaml:"compaction,omitempty" json:"compaction,omitempty"`
	// MetricsGenerator enforced overrides.
	MetricsGenerator MetricsGeneratorOverrides `yaml:"metrics_generator,omitempty" json:"metrics_generator,omitempty"`
	// Forwarders
	Forwarders []string `yaml:"forwarders,omitempty" json:"forwarders,omitempty"`
	// Global enforced overrides.
	Global GlobalOverrides `yaml:"global,omitempty" json:"global,omitempty"`
	// Storage enforced overrides.
	Storage StorageOverrides `yaml:"storage,omitempty" json:"storage,omitempty"`
}

type Config struct {
	Defaults Overrides `yaml:"defaults,omitempty" json:"defaults,omitempty"`

	// Configuration for overrides module
	PerTenantOverrideConfig string         `yaml:"per_tenant_override_config" json:"per_tenant_override_config"`
	PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period" json:"per_tenant_override_period"`

	UserConfigurableOverridesConfig UserConfigurableOverridesConfig `yaml:"user_configurable_overrides" json:"user_configurable_overrides"`

	ConfigType ConfigType `yaml:"-" json:"-"`
	ExpandEnv  bool       `yaml:"-" json:"-"`
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Note: this implementation relies on callers using yaml.UnmarshalStrict. In non-strict mode
	// unmarshal() will not return an error for legacy configuration and we return immediately.

	// Try to unmarshal it normally
	type rawConfig Config
	err := unmarshal((*rawConfig)(c))
	if err == nil {
		c.ConfigType = ConfigTypeNew
		return nil
	}

	// Try to unmarshal inline limits
	type legacyConfig struct {
		DefaultOverrides LegacyOverrides `yaml:",inline"`

		PerTenantOverrideConfig string         `yaml:"per_tenant_override_config"`
		PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period"`

		UserConfigurableOverridesConfig UserConfigurableOverridesConfig `yaml:"user_configurable_overrides"`
	}
	var legacyCfg legacyConfig
	legacyCfg.DefaultOverrides = c.Defaults.toLegacy()
	legacyCfg.PerTenantOverrideConfig = c.PerTenantOverrideConfig
	legacyCfg.PerTenantOverridePeriod = c.PerTenantOverridePeriod
	legacyCfg.UserConfigurableOverridesConfig = c.UserConfigurableOverridesConfig

	if err := unmarshal(&legacyCfg); err != nil {
		return err
	}

	c.Defaults = legacyCfg.DefaultOverrides.toNewLimits()
	c.PerTenantOverrideConfig = legacyCfg.PerTenantOverrideConfig
	c.PerTenantOverridePeriod = legacyCfg.PerTenantOverridePeriod
	c.UserConfigurableOverridesConfig = legacyCfg.UserConfigurableOverridesConfig
	c.ConfigType = ConfigTypeLegacy
	return nil
}

// RegisterFlagsAndApplyDefaults adds the flags required to config this to the given FlagSet
func (c *Config) RegisterFlagsAndApplyDefaults(f *flag.FlagSet) {
	// Distributor LegacyOverrides
	f.StringVar(&c.Defaults.Ingestion.RateStrategy, "distributor.rate-limit-strategy", "local", "Whether the various ingestion rate limits should be applied individually to each distributor instance (local), or evenly shared across the cluster (global).")
	f.IntVar(&c.Defaults.Ingestion.RateLimitBytes, "distributor.ingestion-rate-limit-bytes", 15e6, "Per-user ingestion rate limit in bytes per second.")
	f.IntVar(&c.Defaults.Ingestion.BurstSizeBytes, "distributor.ingestion-burst-size-bytes", 20e6, "Per-user ingestion burst size in bytes. Should be set to the expected size (in bytes) of a single push request.")

	// Ingester limits
	f.IntVar(&c.Defaults.Ingestion.MaxLocalTracesPerUser, "ingester.max-traces-per-user", 10e3, "Maximum number of active traces per user, per ingester. 0 to disable.")
	f.IntVar(&c.Defaults.Ingestion.MaxGlobalTracesPerUser, "ingester.max-global-traces-per-user", 0, "Maximum number of active traces per user, across the cluster. 0 to disable.")
	f.IntVar(&c.Defaults.Global.MaxBytesPerTrace, "ingester.max-bytes-per-trace", 50e5, "Maximum size of a trace in bytes.  0 to disable.")

	// Querier limits
	f.IntVar(&c.Defaults.Read.MaxBytesPerTagValuesQuery, "querier.max-bytes-per-tag-values-query", 50e5, "Maximum size of response for a tag-values query. Used mainly to limit large the number of values associated with a particular tag")
	f.IntVar(&c.Defaults.Read.MaxBlocksPerTagValuesQuery, "querier.max-blocks-per-tag-values-query", 0, "Maximum number of blocks to query for a tag-values query. 0 to disable.")

	f.StringVar(&c.PerTenantOverrideConfig, "config.per-user-override-config", "", "File name of per-user Overrides.")
	_ = c.PerTenantOverridePeriod.Set("10s")
	f.Var(&c.PerTenantOverridePeriod, "config.per-user-override-period", "Period with this to reload the Overrides.")

	c.UserConfigurableOverridesConfig.RegisterFlagsAndApplyDefaults(f)
}

func (c *Config) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricLimitsDesc
}

func (c *Config) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Ingestion.MaxLocalTracesPerUser), MetricMaxLocalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Ingestion.MaxGlobalTracesPerUser), MetricMaxGlobalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Ingestion.RateLimitBytes), MetricIngestionRateLimitBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Ingestion.BurstSizeBytes), MetricIngestionBurstSizeBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Read.MaxBytesPerTagValuesQuery), MetricMaxBytesPerTagValuesQuery)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Read.MaxBlocksPerTagValuesQuery), MetricMaxBlocksPerTagValuesQuery)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Global.MaxBytesPerTrace), MetricMaxBytesPerTrace)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.Compaction.BlockRetention), MetricBlockRetention)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(c.Defaults.MetricsGenerator.MaxActiveSeries), MetricMetricsGeneratorMaxActiveSeries)
}
