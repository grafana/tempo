package overrides

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/common/config"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/grafana/tempo/tempodb/backend"

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

// IngestionRateStrategy defines how the ingestion rate limit is applied.
type IngestionRateStrategy string

const (
	// LocalIngestionRateStrategy indicates that this limit can be evaluated in local terms only
	LocalIngestionRateStrategy IngestionRateStrategy = "local"
	// GlobalIngestionRateStrategy indicates that an attempt should be made to consider this limit across the entire Tempo cluster
	GlobalIngestionRateStrategy IngestionRateStrategy = "global"
)

func (s *IngestionRateStrategy) Set(val string) error {
	switch v := IngestionRateStrategy(val); v {
	case LocalIngestionRateStrategy, GlobalIngestionRateStrategy:
		*s = v
		return nil
	default:
		return fmt.Errorf("invalid ingestion rate strategy %q, must be one of: %s, %s", val, LocalIngestionRateStrategy, GlobalIngestionRateStrategy)
	}
}

func (s *IngestionRateStrategy) String() string {
	if s == nil {
		return string(LocalIngestionRateStrategy)
	}
	return string(*s)
}

const (
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
	MetricCompactionDisabled              = "compaction_disabled"
	MetricMetricsGeneratorMaxActiveSeries = "metrics_generator_max_active_series"
	MetricsGeneratorDryRunEnabled         = "metrics_generator_dry_run_enabled"
)

var metricLimitsDesc = prometheus.NewDesc(
	"tempo_limits_defaults",
	"Default resource limits",
	[]string{"limit_name"},
	nil,
)

// ptrTo returns a pointer to the given value.
func ptrTo[T any](v T) *T { return &v }

type IngestionOverrides struct {
	// Distributor enforced limits.

	// RateStrategy is local or global; only applies to RateLimitBytes (default: local)
	RateStrategy IngestionRateStrategy `yaml:"rate_strategy,omitempty" json:"rate_strategy,omitempty"`
	// RateLimitBytes is the per-user ingestion rate limit in bytes (default: 15MB)
	RateLimitBytes *int `yaml:"rate_limit_bytes,omitempty" json:"rate_limit_bytes,omitempty"`
	// BurstSizeBytes is the burst size in bytes, always local (default: 20MB)
	BurstSizeBytes *int `yaml:"burst_size_bytes,omitempty" json:"burst_size_bytes,omitempty"`

	// Ingester enforced limits.
	// MaxLocalTracesPerUser is the max active traces per user per ingester; 0 disables (default: 10000)
	MaxLocalTracesPerUser *int `yaml:"max_traces_per_user,omitempty" json:"max_traces_per_user,omitempty"`
	// MaxGlobalTracesPerUser is the max active traces per user across the cluster; 0 disables (default: 0)
	MaxGlobalTracesPerUser *int `yaml:"max_global_traces_per_user,omitempty" json:"max_global_traces_per_user,omitempty"`

	// TenantShardSize is the shuffle sharding shard count; 0 uses all ingesters (default: 0)
	TenantShardSize *int `yaml:"tenant_shard_size,omitempty" json:"tenant_shard_size,omitempty"`
	// MaxAttributeBytes is the max bytes for attribute keys and values (default: 0)
	MaxAttributeBytes *int `yaml:"max_attribute_bytes,omitempty" json:"max_attribute_bytes,omitempty"`
	// ArtificialDelay pads push requests to ensure min avg latency; nil disables (default: nil)
	ArtificialDelay *time.Duration `yaml:"artificial_delay,omitempty" json:"artificial_delay,omitempty"`
	// RetryInfoEnabled toggles retry-after header in rate-limit responses.
	// Only effective when cluster-level distributor.retry_after_on_resource_exhausted > 0. (default: true)
	RetryInfoEnabled *bool `yaml:"retry_info_enabled,omitempty" json:"retry_info_enabled,omitempty"`
}

type ForwarderOverrides struct {
	QueueSize int `yaml:"queue_size,omitempty" json:"queue_size,omitempty"`
	Workers   int `yaml:"workers,omitempty" json:"workers,omitempty"`
}

type ServiceGraphsOverrides struct {
	HistogramBuckets                      []float64                   `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	Dimensions                            []string                    `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	PeerAttributes                        []string                    `yaml:"peer_attributes,omitempty" json:"peer_attributes,omitempty"`
	FilterPolicies                        []filterconfig.FilterPolicy `yaml:"filter_policies,omitempty" json:"filter_policies,omitempty"`
	EnableClientServerPrefix              *bool                       `yaml:"enable_client_server_prefix,omitempty" json:"enable_client_server_prefix,omitempty"`
	EnableMessagingSystemLatencyHistogram *bool                       `yaml:"enable_messaging_system_latency_histogram,omitempty" json:"enable_messaging_system_latency_histogram,omitempty"`
	EnableVirtualNodeLabel                *bool                       `yaml:"enable_virtual_node_label,omitempty" json:"enable_virtual_node_label,omitempty"`
	// SpanMultiplierKey is the attribute key used to multiply span metrics. nil means not set (inherit), "" disables the multiplier.
	SpanMultiplierKey              *string `yaml:"span_multiplier_key,omitempty" json:"span_multiplier_key,omitempty"`
	EnableTraceStateSpanMultiplier *bool   `yaml:"enable_tracestate_span_multiplier,omitempty" json:"enable_tracestate_span_multiplier,omitempty"`
}

type SpanMetricsOverrides struct {
	HistogramBuckets             []float64                        `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	Dimensions                   []string                         `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	IntrinsicDimensions          map[string]bool                  `yaml:"intrinsic_dimensions,omitempty" json:"intrinsic_dimensions,omitempty"`
	FilterPolicies               []filterconfig.FilterPolicy      `yaml:"filter_policies,omitempty" json:"filter_policies,omitempty"`
	DimensionMappings            []sharedconfig.DimensionMappings `yaml:"dimension_mappings,omitempty" json:"dimension_mapings,omitempty"`
	EnableTargetInfo             *bool                            `yaml:"enable_target_info,omitempty" json:"enable_target_info,omitempty"`
	TargetInfoExcludedDimensions []string                         `yaml:"target_info_excluded_dimensions,omitempty" json:"target_info_excluded_dimensions,omitempty"`
	EnableInstanceLabel          *bool                            `yaml:"enable_instance_label,omitempty" json:"enable_instance_label,omitempty"`
	// SpanMultiplierKey is the attribute key used to multiply span metrics. nil means not set (inherit), "" disables the multiplier.
	SpanMultiplierKey              *string `yaml:"span_multiplier_key,omitempty" json:"span_multiplier_key,omitempty"`
	EnableTraceStateSpanMultiplier *bool   `yaml:"enable_tracestate_span_multiplier,omitempty" json:"enable_tracestate_span_multiplier,omitempty"`
}

type HostInfoOverrides struct {
	HostIdentifiers []string `yaml:"host_identifiers,omitempty" json:"host_identifies,omitempty"`
	MetricName      string   `yaml:"metric_name,omitempty" json:"metric_name,omitempty"`
}

type ProcessorOverrides struct {
	ServiceGraphs ServiceGraphsOverrides `yaml:"service_graphs,omitempty" json:"service_graphs,omitempty"`
	SpanMetrics   SpanMetricsOverrides   `yaml:"span_metrics,omitempty" json:"span_metrics,omitempty"`
	HostInfo      HostInfoOverrides      `yaml:"host_info,omitempty" json:"host_info,omitempty"`
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
	// RingSize is the per-tenant metrics-generator ring size; 0 uses the full ring (default: 0)
	RingSize *int `yaml:"ring_size,omitempty" json:"ring_size,omitempty"`
	// Processors is the list of enabled metrics-generator processors for this tenant
	Processors listtomap.ListToMap `yaml:"processors,omitempty" json:"processors,omitempty"`
	// MaxActiveSeries is the max active series in the registry per instance; 0 disables (default: 0)
	MaxActiveSeries *uint32 `yaml:"max_active_series,omitempty" json:"max_active_series,omitempty"`
	// MaxActiveEntities is the max active entities in the registry per instance; 0 disables (default: 0)
	MaxActiveEntities *uint32 `yaml:"max_active_entities,omitempty" json:"max_active_entities,omitempty"`
	// CollectionInterval is the per-tenant collection interval; 0 uses the static config default (default: 0)
	CollectionInterval *time.Duration `yaml:"collection_interval,omitempty" json:"collection_interval,omitempty"`
	// DisableCollection disables a registry collection without disabling ingestion (default: false)
	DisableCollection *bool `yaml:"disable_collection,omitempty" json:"disable_collection,omitempty"`
	// GenerateNativeHistograms controls native histogram generation; "" means classic only (default: classic)
	GenerateNativeHistograms histograms.HistogramMethod `yaml:"generate_native_histograms" json:"generate_native_histograms,omitempty"`
	// TraceIDLabelName is the label name for trace ID in exemplars (default: "traceID")
	TraceIDLabelName string `yaml:"trace_id_label_name,omitempty" json:"trace_id_label_name,omitempty"`

	// RemoteWriteHeaders are additional headers to send with remote write requests
	RemoteWriteHeaders RemoteWriteHeaders `yaml:"remote_write_headers,omitempty" json:"remote_write_headers,omitempty"`

	Forwarder ForwarderOverrides `yaml:"forwarder,omitempty" json:"forwarder,omitempty"`
	Processor ProcessorOverrides `yaml:"processor,omitempty" json:"processor,omitempty"`
	// IngestionSlack filters out spans older than this duration; 0 uses the static config default (default: 0)
	IngestionSlack *time.Duration `yaml:"ingestion_time_range_slack" json:"ingestion_time_range_slack,omitempty"`

	// FIXME: check if NativeHistogramBucketFactor and NativeHistogramMaxBucketNumber should be pointers??
	// NativeHistogramBucketFactor is the growth factor between native histogram buckets (default: 1.1)
	NativeHistogramBucketFactor float64 `yaml:"native_histogram_bucket_factor,omitempty" json:"native_histogram_bucket_factor,omitempty"`
	// NativeHistogramMaxBucketNumber is the max number of native histogram buckets (default: 100)
	NativeHistogramMaxBucketNumber uint32 `yaml:"native_histogram_max_bucket_number,omitempty" json:"native_histogram_max_bucket_number,omitempty"`
	// NativeHistogramMinResetDuration is the min duration before a native histogram can be reset; 0 disables (default: 15m)
	NativeHistogramMinResetDuration *time.Duration `yaml:"native_histogram_min_reset_duration,omitempty" json:"native_histogram_min_reset_duration,omitempty"`
	// SpanNameSanitization controls span name sanitization; nil means not set, "" disables (default: "")
	SpanNameSanitization *string `yaml:"span_name_sanitization,omitempty" json:"span_name_sanitization,omitempty"`
	// MaxCardinalityPerLabel is the max distinct values per label; 0 disables (default: 0)
	MaxCardinalityPerLabel *uint64 `yaml:"max_cardinality_per_label,omitempty" json:"max_cardinality_per_label,omitempty"`
}

type ReadOverrides struct {
	// Querier and Ingester enforced overrides.

	// MaxBytesPerTagValuesQuery is the max response size for tag-values queries; 0 disables (default: 1MB)
	MaxBytesPerTagValuesQuery *int `yaml:"max_bytes_per_tag_values_query,omitempty" json:"max_bytes_per_tag_values_query,omitempty"`
	// MaxBlocksPerTagValuesQuery is the max blocks to inspect for tag-values queries; 0 disables (default: 0)
	MaxBlocksPerTagValuesQuery *int `yaml:"max_blocks_per_tag_values_query,omitempty" json:"max_blocks_per_tag_values_query,omitempty"`

	// QueryFrontend enforced overrides.

	// MaxSearchDuration is the per-tenant max search time range; 0 uses the frontend default (default: 0)
	MaxSearchDuration *model.Duration `yaml:"max_search_duration,omitempty" json:"max_search_duration,omitempty"`
	// MaxMetricsDuration is the per-tenant max metrics query time range; 0 uses the frontend default (default: 0)
	MaxMetricsDuration *model.Duration `yaml:"max_metrics_duration,omitempty" json:"max_metrics_duration,omitempty"`

	// UnsafeQueryHints enables query hints that can improve performance but may be unsafe (default: false)
	UnsafeQueryHints *bool `yaml:"unsafe_query_hints,omitempty" json:"unsafe_query_hints,omitempty"`

	// LeftPadTraceIDs left-pads trace IDs to 32 hex characters for W3C/OTel compliance (default: false)
	LeftPadTraceIDs *bool `yaml:"left_pad_trace_ids,omitempty" json:"left_pad_trace_ids,omitempty"`
}

type CompactionOverrides struct {
	// Backend-worker/scheduler enforced overrides.
	// BlockRetention is the per-tenant block retention; 0 uses the compaction config default (default: 0)
	BlockRetention *model.Duration `yaml:"block_retention,omitempty" json:"block_retention,omitempty"`
	// CompactionWindow is the per-tenant compaction window; 0 uses the compaction config default (default: 0)
	CompactionWindow *model.Duration `yaml:"compaction_window,omitempty" json:"compaction_window,omitempty"`
	// CompactionDisabled disables compaction and retention for this tenant (default: false)
	CompactionDisabled *bool `yaml:"compaction_disabled,omitempty" json:"compaction_disabled,omitempty"`
}

type GlobalOverrides struct {
	// MaxBytesPerTrace is the max trace size in bytes, enforced in ingester, compactor, and querier (search).
	// Not enforced for trace-by-ID lookups. 0 disables. (default: 5MB)
	MaxBytesPerTrace *int `yaml:"max_bytes_per_trace,omitempty" json:"max_bytes_per_trace,omitempty"`
}

type StorageOverrides struct {
	// tempodb limits
	DedicatedColumns backend.DedicatedColumns `yaml:"parquet_dedicated_columns" json:"parquet_dedicated_columns"`
}

type CostAttributionOverrides struct {
	// MaxCardinality is the max number of cost attribution series per tenant; 0 disables the limit (default: 10000)
	MaxCardinality *uint64           `yaml:"max_cardinality,omitempty" json:"max_cardinality,omitempty"`
	Dimensions     map[string]string `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
}

type Overrides struct {
	// Global enforced overrides.
	Global GlobalOverrides `yaml:"global,omitempty" json:"global,omitempty"`
	// Ingestion enforced overrides.
	Ingestion IngestionOverrides `yaml:"ingestion,omitempty" json:"ingestion,omitempty"`
	// Read enforced overrides.
	Read ReadOverrides `yaml:"read,omitempty" json:"read,omitempty"`
	// MetricsGenerator enforced overrides.
	MetricsGenerator MetricsGeneratorOverrides `yaml:"metrics_generator,omitempty" json:"metrics_generator,omitempty"`
	// Forwarders
	Forwarders []string `yaml:"forwarders,omitempty" json:"forwarders,omitempty"`
	// Compaction enforced overrides.
	Compaction CompactionOverrides `yaml:"compaction,omitempty" json:"compaction,omitempty"`
	// Storage enforced overrides.
	Storage StorageOverrides `yaml:"storage,omitempty" json:"storage,omitempty"`
	// CostAttribution overrides, used to configure the usage tracker
	CostAttribution CostAttributionOverrides `yaml:"cost_attribution,omitempty" json:"cost_attribution,omitempty"`
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
	// unmarshal() will not return an error for legacy configuration, and we return immediately.

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

	if legacyErr := unmarshal(&legacyCfg); legacyErr != nil {
		return fmt.Errorf("failed to unmarshal config: %w; also failed in legacy format: %w", err, legacyErr)
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
	c.Defaults.Ingestion.RetryInfoEnabled = ptrTo(true) // enabled in overrides by default, but it's disabled with RetryAfterOnResourceExhausted = 0
	c.Defaults.Ingestion.RateStrategy = LocalIngestionRateStrategy
	f.Var(&c.Defaults.Ingestion.RateStrategy, "distributor.rate-limit-strategy", "Whether the various ingestion rate limits should be applied individually to each distributor instance (local), or evenly shared across the cluster (global).")
	c.Defaults.Ingestion.RateLimitBytes = f.Int("distributor.ingestion-rate-limit-bytes", 15_000_000, "Per-user ingestion rate limit in bytes per second.")
	c.Defaults.Ingestion.BurstSizeBytes = f.Int("distributor.ingestion-burst-size-bytes", 20_000_000, "Per-user ingestion burst size in bytes. Should be set to the expected size (in bytes) of a single push request.")

	// Ingester limits - pointer fields need allocation before IntVar can bind to them.
	c.Defaults.Ingestion.MaxLocalTracesPerUser = ptrTo(0)
	f.IntVar(c.Defaults.Ingestion.MaxLocalTracesPerUser, "ingester.max-traces-per-user", 10_000, "Maximum number of active traces per user, per ingester. 0 to disable.")
	c.Defaults.Ingestion.MaxGlobalTracesPerUser = ptrTo(0)
	f.IntVar(c.Defaults.Ingestion.MaxGlobalTracesPerUser, "ingester.max-global-traces-per-user", 0, "Maximum number of active traces per user, across the cluster. 0 to disable.")
	c.Defaults.Global.MaxBytesPerTrace = ptrTo(0)
	f.IntVar(c.Defaults.Global.MaxBytesPerTrace, "ingester.max-bytes-per-trace", 5_000_000, "Maximum size of a trace in bytes.  0 to disable.")

	// Querier limits
	c.Defaults.Read.MaxBytesPerTagValuesQuery = ptrTo(0)
	f.IntVar(c.Defaults.Read.MaxBytesPerTagValuesQuery, "querier.max-bytes-per-tag-values-query", 1_000_000, "Maximum size of response for a tag-values query. Used mainly to limit large the number of values associated with a particular tag")
	c.Defaults.Read.MaxSearchDuration = ptrTo(model.Duration(0))
	c.Defaults.Read.MaxMetricsDuration = ptrTo(model.Duration(0))
	c.Defaults.Read.MaxBlocksPerTagValuesQuery = ptrTo(0)
	f.IntVar(c.Defaults.Read.MaxBlocksPerTagValuesQuery, "querier.max-blocks-per-tag-values-query", 0, "Maximum number of blocks to query for a tag-values query. 0 to disable.")
	c.Defaults.Read.UnsafeQueryHints = ptrTo(false)
	c.Defaults.Read.LeftPadTraceIDs = ptrTo(false)

	// Compaction limits
	c.Defaults.Compaction.BlockRetention = ptrTo(model.Duration(0))
	c.Defaults.Compaction.CompactionWindow = ptrTo(model.Duration(0))
	c.Defaults.Compaction.CompactionDisabled = ptrTo(false)

	// Ingester pointer fields
	c.Defaults.Ingestion.TenantShardSize = ptrTo(0)
	c.Defaults.Ingestion.MaxAttributeBytes = ptrTo(0)
	// c.Defaults.Ingestion.ArtificialDelay is left nil by default. nil means "not set" in the override,
	// so the distributor's own cfg.ArtificialDelay is used as the fallback.

	// Metrics-generator limits
	c.Defaults.MetricsGenerator.RingSize = ptrTo(0)
	c.Defaults.MetricsGenerator.MaxActiveSeries = ptrTo(uint32(0))
	c.Defaults.MetricsGenerator.MaxActiveEntities = ptrTo(uint32(0))
	c.Defaults.MetricsGenerator.CollectionInterval = ptrTo(time.Duration(0))
	c.Defaults.MetricsGenerator.DisableCollection = ptrTo(false)
	c.Defaults.MetricsGenerator.MaxCardinalityPerLabel = ptrTo(uint64(0))
	c.Defaults.MetricsGenerator.Processor.ServiceGraphs.EnableClientServerPrefix = ptrTo(false)

	// Metrics-generator forwarder defaults
	c.Defaults.MetricsGenerator.Forwarder.QueueSize = 100
	c.Defaults.MetricsGenerator.Forwarder.Workers = 2
	c.Defaults.MetricsGenerator.TraceIDLabelName = "traceID"
	c.Defaults.MetricsGenerator.Processor.HostInfo.MetricName = "traces_host_info"

	// Cost attribution
	c.Defaults.CostAttribution.MaxCardinality = ptrTo(uint64(10000))

	// Metrics-generator pointer fields for newly migrated types
	c.Defaults.MetricsGenerator.IngestionSlack = ptrTo(time.Duration(0))
	c.Defaults.MetricsGenerator.NativeHistogramMinResetDuration = ptrTo(time.Duration(0))
	c.Defaults.MetricsGenerator.SpanNameSanitization = ptrTo("")
	c.Defaults.MetricsGenerator.Processor.ServiceGraphs.SpanMultiplierKey = ptrTo("")
	c.Defaults.MetricsGenerator.Processor.SpanMetrics.SpanMultiplierKey = ptrTo("")

	// MetricsGenerator - NativeHistograms config
	c.Defaults.MetricsGenerator.GenerateNativeHistograms = histograms.HistogramMethodClassic
	f.Float64Var(&c.Defaults.MetricsGenerator.NativeHistogramBucketFactor, "metrics-generator.native-histogram-bucket-factor", 1.1, "The growth factor between buckets for native histograms.")
	_ = (*Uint32Value)(&c.Defaults.MetricsGenerator.NativeHistogramMaxBucketNumber).Set("100")
	f.Var((*Uint32Value)(&c.Defaults.MetricsGenerator.NativeHistogramMaxBucketNumber), "metrics-generator.native-histogram-max-bucket-number", "The maximum number of buckets for native histograms.")
	f.DurationVar(c.Defaults.MetricsGenerator.NativeHistogramMinResetDuration, "metrics-generator.native-histogram-min-reset-duration", 15*time.Minute, "The minimum duration before a native histogram can be reset.")

	f.StringVar(&c.PerTenantOverrideConfig, "config.per-user-override-config", "", "File name of per-user Overrides.")
	_ = c.PerTenantOverridePeriod.Set("10s")
	f.Var(&c.PerTenantOverridePeriod, "config.per-user-override-period", "Period with this to reload the Overrides.")

	c.UserConfigurableOverridesConfig.RegisterFlagsAndApplyDefaults(f)
}

func (c *Config) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricLimitsDesc
}

// Collect reports the static default limits as prometheus metrics.
// Pointer fields are guaranteed non-nil after RegisterFlagsAndApplyDefaults.
func (c *Config) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Ingestion.MaxLocalTracesPerUser), MetricMaxLocalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Ingestion.MaxGlobalTracesPerUser), MetricMaxGlobalTracesPerUser)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Ingestion.RateLimitBytes), MetricIngestionRateLimitBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Ingestion.BurstSizeBytes), MetricIngestionBurstSizeBytes)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Read.MaxBytesPerTagValuesQuery), MetricMaxBytesPerTagValuesQuery)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Read.MaxBlocksPerTagValuesQuery), MetricMaxBlocksPerTagValuesQuery)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Global.MaxBytesPerTrace), MetricMaxBytesPerTrace)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Compaction.BlockRetention), MetricBlockRetention)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.Compaction.CompactionWindow), MetricCompactionWindow)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, boolToFloat64(*c.Defaults.Compaction.CompactionDisabled), MetricCompactionDisabled)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, float64(*c.Defaults.MetricsGenerator.MaxActiveSeries), MetricMetricsGeneratorMaxActiveSeries)
	ch <- prometheus.MustNewConstMetric(metricLimitsDesc, prometheus.GaugeValue, boolToFloat64(*c.Defaults.MetricsGenerator.DisableCollection), MetricsGeneratorDryRunEnabled)
}

func HasNativeHistograms(s histograms.HistogramMethod) bool {
	return s == histograms.HistogramMethodNative || s == histograms.HistogramMethodBoth
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

type Uint32Value uint32

func (u *Uint32Value) String() string { return fmt.Sprintf("%d", *u) }
func (u *Uint32Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	*u = Uint32Value(v)
	return nil
}
