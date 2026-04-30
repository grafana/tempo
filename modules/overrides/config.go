package overrides

import (
	"encoding/json"
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/common/config"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/traceql"
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

	TenantShardSize   int            `yaml:"tenant_shard_size,omitempty" json:"tenant_shard_size,omitempty"`
	MaxAttributeBytes int            `yaml:"max_attribute_bytes,omitempty" json:"max_attribute_bytes,omitempty"`
	ArtificialDelay   *time.Duration `yaml:"artificial_delay,omitempty" json:"artificial_delay,omitempty"`
	RetryInfoEnabled  bool           `yaml:"retry_info_enabled,omitempty" json:"retry_info_enabled,omitempty"`
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
	SpanMultiplierKey                     string                      `yaml:"span_multiplier_key,omitempty" json:"span_multiplier_key,omitempty"`
	EnableTraceStateSpanMultiplier        *bool                       `yaml:"enable_tracestate_span_multiplier,omitempty" json:"enable_tracestate_span_multiplier,omitempty"`
}

type SpanMetricsOverrides struct {
	HistogramBuckets               []float64                        `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	Dimensions                     []string                         `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	IntrinsicDimensions            map[string]bool                  `yaml:"intrinsic_dimensions,omitempty" json:"intrinsic_dimensions,omitempty"`
	FilterPolicies                 []filterconfig.FilterPolicy      `yaml:"filter_policies,omitempty" json:"filter_policies,omitempty"`
	DimensionMappings              []sharedconfig.DimensionMappings `yaml:"dimension_mappings,omitempty" json:"dimension_mapings,omitempty"`
	EnableTargetInfo               *bool                            `yaml:"enable_target_info,omitempty" json:"enable_target_info,omitempty"`
	TargetInfoExcludedDimensions   []string                         `yaml:"target_info_excluded_dimensions,omitempty" json:"target_info_excluded_dimensions,omitempty"`
	EnableInstanceLabel            *bool                            `yaml:"enable_instance_label,omitempty" json:"enable_instance_label,omitempty"`
	SpanMultiplierKey              string                           `yaml:"span_multiplier_key,omitempty" json:"span_multiplier_key,omitempty"`
	EnableTraceStateSpanMultiplier *bool                            `yaml:"enable_tracestate_span_multiplier,omitempty" json:"enable_tracestate_span_multiplier,omitempty"`
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
	RingSize                 int                        `yaml:"ring_size,omitempty" json:"ring_size,omitempty"`
	Processors               listtomap.ListToMap        `yaml:"processors,omitempty" json:"processors,omitempty"`
	MaxActiveSeries          uint32                     `yaml:"max_active_series,omitempty" json:"max_active_series,omitempty"`
	MaxActiveEntities        uint32                     `yaml:"max_active_entities,omitempty" json:"max_active_entities,omitempty"`
	CollectionInterval       time.Duration              `yaml:"collection_interval,omitempty" json:"collection_interval,omitempty"`
	DisableCollection        bool                       `yaml:"disable_collection,omitempty" json:"disable_collection,omitempty"`
	GenerateNativeHistograms histograms.HistogramMethod `yaml:"generate_native_histograms,omitempty" json:"generate_native_histograms,omitempty"`
	TraceIDLabelName         string                     `yaml:"trace_id_label_name,omitempty" json:"trace_id_label_name,omitempty"`

	RemoteWriteHeaders RemoteWriteHeaders `yaml:"remote_write_headers,omitempty" json:"remote_write_headers,omitempty"`

	Forwarder      ForwarderOverrides `yaml:"forwarder,omitempty" json:"forwarder,omitempty"`
	Processor      ProcessorOverrides `yaml:"processor,omitempty" json:"processor,omitempty"`
	IngestionSlack time.Duration      `yaml:"ingestion_time_range_slack,omitempty" json:"ingestion_time_range_slack,omitempty"`

	NativeHistogramBucketFactor     float64       `yaml:"native_histogram_bucket_factor,omitempty" json:"native_histogram_bucket_factor,omitempty"`
	NativeHistogramMaxBucketNumber  uint32        `yaml:"native_histogram_max_bucket_number,omitempty" json:"native_histogram_max_bucket_number,omitempty"`
	NativeHistogramMinResetDuration time.Duration `yaml:"native_histogram_min_reset_duration,omitempty" json:"native_histogram_min_reset_duration,omitempty"`
	SpanNameSanitization            string        `yaml:"span_name_sanitization,omitempty" json:"span_name_sanitization,omitempty"`
	MaxCardinalityPerLabel          uint64        `yaml:"max_cardinality_per_label,omitempty" json:"max_cardinality_per_label,omitempty"`
}

type ReadOverrides struct {
	// Querier and Ingester enforced overrides.
	MaxBytesPerTagValuesQuery     int `yaml:"max_bytes_per_tag_values_query,omitempty" json:"max_bytes_per_tag_values_query,omitempty"`
	MaxBlocksPerTagValuesQuery    int `yaml:"max_blocks_per_tag_values_query,omitempty" json:"max_blocks_per_tag_values_query,omitempty"`
	MaxConditionGroupsPerTagQuery int `yaml:"max_condition_groups_per_tag_query,omitempty" json:"max_condition_groups_per_tag_query,omitempty"`

	// QueryFrontend enforced overrides
	MaxSearchDuration  model.Duration `yaml:"max_search_duration,omitempty" json:"max_search_duration,omitempty"`
	MaxMetricsDuration model.Duration `yaml:"max_metrics_duration,omitempty" json:"max_metrics_duration,omitempty"`

	UnsafeQueryHints bool `yaml:"unsafe_query_hints,omitempty" json:"unsafe_query_hints,omitempty"`

	// LeftPadTraceIDs left-pads trace IDs in search responses to 32 hex characters with zeros.
	// This produces W3C/OpenTelemetry compliant trace IDs (32-hex-character lowercase strings).
	LeftPadTraceIDs bool `yaml:"left_pad_trace_ids,omitempty" json:"left_pad_trace_ids,omitempty"`

	// MetricsSpanOnlyFetch, when set, enables or disables the new fetch layer by default for TraceQL metrics queries
	// for this tenant.  When not set, then the default behavior is used. Maybe be overridden by query hints.
	MetricsSpanOnlyFetch *bool `yaml:"metrics_spanonly_fetch,omitempty" json:"metrics_spanonly_fetch,omitempty"`
}

type CompactionOverrides struct {
	// Backend-worker/scheduler enforced overrides.
	BlockRetention     model.Duration `yaml:"block_retention,omitempty" json:"block_retention,omitempty"`
	CompactionWindow   model.Duration `yaml:"compaction_window,omitempty" json:"compaction_window,omitempty"`
	CompactionDisabled bool           `yaml:"compaction_disabled,omitempty" json:"compaction_disabled,omitempty"`
}

type GlobalOverrides struct {
	// MaxBytesPerTrace is enforced in the Ingester, Compactor, Querier (Search). It
	//  is not used when doing a trace by id lookup.
	MaxBytesPerTrace int `yaml:"max_bytes_per_trace,omitempty" json:"max_bytes_per_trace,omitempty"`
}

type StorageOverrides struct {
	// tempodb limits
	DedicatedColumns backend.DedicatedColumns `yaml:"parquet_dedicated_columns" json:"parquet_dedicated_columns"`
}

type CostAttributionOverrides struct {
	MaxCardinality uint64            `yaml:"max_cardinality,omitempty" json:"max_cardinality,omitempty"`
	Dimensions     map[string]string `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
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
	Storage         StorageOverrides         `yaml:"storage,omitempty" json:"storage,omitempty"`
	CostAttribution CostAttributionOverrides `yaml:"cost_attribution,omitempty" json:"cost_attribution,omitempty"`
	// Extensions holds per-tenant overrides added by vendoring applications via RegisterExtension.
	// Values are typed Extension instances after unmarshal.
	Extensions map[string]any `yaml:",inline" json:"-"`
}

// knownOverridesJSONFields returns the JSON key names declared on Overrides
var knownOverridesJSONFields = sync.OnceValue(func() map[string]struct{} {
	return fieldNamesFor(Overrides{}, "json")
})

func (o *Overrides) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Overrides
	if err := unmarshal((*plain)(o)); err != nil {
		return err
	}
	return processExtensions(o)
}

func (o *Overrides) UnmarshalJSON(data []byte) error {
	type plain Overrides
	if err := json.Unmarshal(data, (*plain)(o)); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for key := range knownOverridesJSONFields() {
		delete(raw, key)
	}
	if len(raw) == 0 {
		// No extension keys in this payload; clear any stale Extensions from a prior decode.
		o.Extensions = nil
		return nil
	}

	o.Extensions = make(map[string]any, len(raw))
	for k, v := range raw {
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			return err
		}
		o.Extensions[k] = val
	}
	return processExtensions(o)
}

func (o Overrides) MarshalJSON() ([]byte, error) {
	type plain Overrides
	data, err := json.Marshal(plain(o))
	if err != nil {
		return nil, err
	}
	if len(o.Extensions) == 0 {
		return data, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for k, v := range o.Extensions {
		if _, exists := m[k]; exists {
			continue // known fields take precedence
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = b
	}
	return json.Marshal(m)
}

type Config struct {
	Defaults Overrides `yaml:"defaults,omitempty" json:"defaults,omitempty"`

	// Configuration for overrides module
	PerTenantOverrideConfig string         `yaml:"per_tenant_override_config" json:"per_tenant_override_config"`
	PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period" json:"per_tenant_override_period"`

	UserConfigurableOverridesConfig UserConfigurableOverridesConfig `yaml:"user_configurable_overrides" json:"user_configurable_overrides"`

	// EnableLegacyOverrides allows using the deprecated legacy (flat, unscoped) overrides format.
	// Legacy overrides are disabled by default. Set this to true to opt back in while migrating
	// to the new format. This option will be removed in a future release.
	EnableLegacyOverrides bool `yaml:"enable_legacy_overrides" json:"enable_legacy_overrides"`

	ConfigType ConfigType `yaml:"-" json:"-"`
	ExpandEnv  bool       `yaml:"-" json:"-"`
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Note: this implementation relies on callers using yaml.UnmarshalStrict. In non-strict mode
	// unmarshal() will not return an error for legacy configuration and we return immediately.

	// Try to unmarshal it normally. Overrides.UnmarshalYAML calls processExtensions for c.Defaults.
	// If the error is an extensionError, the config is in the new format but misconfigured.
	type rawConfig Config
	err := unmarshal((*rawConfig)(c))
	if err == nil {
		c.ConfigType = ConfigTypeNew
		return nil
	}

	// Fail only on extension-specific errors, otherwise fallback to legacy mode
	if isExtensionError(err) || isExtensionKeyError(err) {
		return err
	}

	// Try to unmarshal inline limits
	type legacyConfig struct {
		DefaultOverrides LegacyOverrides `yaml:",inline"`

		PerTenantOverrideConfig string         `yaml:"per_tenant_override_config"`
		PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period"`

		UserConfigurableOverridesConfig UserConfigurableOverridesConfig `yaml:"user_configurable_overrides"`

		EnableLegacyOverrides bool `yaml:"enable_legacy_overrides"`
	}
	var legacyCfg legacyConfig
	legacyCfg.DefaultOverrides = c.Defaults.toLegacy()
	legacyCfg.PerTenantOverrideConfig = c.PerTenantOverrideConfig
	legacyCfg.PerTenantOverridePeriod = c.PerTenantOverridePeriod
	legacyCfg.UserConfigurableOverridesConfig = c.UserConfigurableOverridesConfig
	legacyCfg.EnableLegacyOverrides = c.EnableLegacyOverrides

	if legacyErr := unmarshal(&legacyCfg); legacyErr != nil {
		return fmt.Errorf("failed to unmarshal config: %w; also failed in legacy format: %w", err, legacyErr)
	}

	// Ensure legacy extension flat keys are converted to typed instances before toNewLimits.
	// processLegacyExtensions may not be triggered automatically for inline struct fields.
	if err := processLegacyExtensions(&legacyCfg.DefaultOverrides); err != nil {
		return fmt.Errorf("defaults: %w", err)
	}

	c.Defaults = *legacyCfg.DefaultOverrides.toNewLimits()
	c.PerTenantOverrideConfig = legacyCfg.PerTenantOverrideConfig
	c.PerTenantOverridePeriod = legacyCfg.PerTenantOverridePeriod
	c.UserConfigurableOverridesConfig = legacyCfg.UserConfigurableOverridesConfig
	c.EnableLegacyOverrides = legacyCfg.EnableLegacyOverrides
	c.ConfigType = ConfigTypeLegacy
	return nil
}

// RegisterFlagsAndApplyDefaults adds the flags required to config this to the given FlagSet
func (c *Config) RegisterFlagsAndApplyDefaults(f *flag.FlagSet) {
	// Generator
	c.Defaults.MetricsGenerator.GenerateNativeHistograms = histograms.HistogramMethodClassic

	// Distributor LegacyOverrides
	// enabled in overrides by default, only takes effect when
	// distributor.retry_after_on_resource_exhausted is greater than 0.cluster level default is 5s.
	c.Defaults.Ingestion.RetryInfoEnabled = true
	f.StringVar(&c.Defaults.Ingestion.RateStrategy, "distributor.rate-limit-strategy", "local", "Whether the various ingestion rate limits should be applied individually to each distributor instance (local), or evenly shared across the cluster (global).")
	f.IntVar(&c.Defaults.Ingestion.RateLimitBytes, "distributor.ingestion-rate-limit-bytes", 30e6, "Per-user ingestion rate limit in bytes per second.")
	f.IntVar(&c.Defaults.Ingestion.BurstSizeBytes, "distributor.ingestion-burst-size-bytes", 30e6, "Per-user ingestion burst size in bytes. Should be set to the expected size (in bytes) of a single push request.")

	// Ingester limits
	f.IntVar(&c.Defaults.Ingestion.MaxLocalTracesPerUser, "ingester.max-traces-per-user", 10e3, "Maximum number of active traces per user, per ingester. 0 to disable.")
	f.IntVar(&c.Defaults.Ingestion.MaxGlobalTracesPerUser, "ingester.max-global-traces-per-user", 0, "Maximum number of active traces per user, across the cluster. 0 to disable.")
	f.IntVar(&c.Defaults.Global.MaxBytesPerTrace, "ingester.max-bytes-per-trace", 50e5, "Maximum size of a trace in bytes.  0 to disable.")

	// Querier limits
	f.IntVar(&c.Defaults.Read.MaxBytesPerTagValuesQuery, "querier.max-bytes-per-tag-values-query", 10e5, "Maximum size of response for a tag-values query. Used mainly to limit large the number of values associated with a particular tag")
	f.IntVar(&c.Defaults.Read.MaxBlocksPerTagValuesQuery, "querier.max-blocks-per-tag-values-query", 0, "Maximum number of blocks to query for a tag-values query. 0 to disable.")
	f.IntVar(&c.Defaults.Read.MaxConditionGroupsPerTagQuery, "querier.max-condition-groups-per-tag-query", traceql.DefaultMaxConditionGroupsPerTagQuery, "Maximum number of OR-expanded condition groups allowed in a tag search query. Queries that expand beyond this limit will be rejected.")

	// Generator - NativeHistograms config
	f.Float64Var(&c.Defaults.MetricsGenerator.NativeHistogramBucketFactor, "metrics-generator.native-histogram-bucket-factor", 1.1, "The growth factor between buckets for native histograms.")
	_ = (*Uint32Value)(&c.Defaults.MetricsGenerator.NativeHistogramMaxBucketNumber).Set("100")
	f.Var((*Uint32Value)(&c.Defaults.MetricsGenerator.NativeHistogramMaxBucketNumber), "metrics-generator.native-histogram-max-bucket-number", "The maximum number of buckets for native histograms.")
	f.DurationVar(&c.Defaults.MetricsGenerator.NativeHistogramMinResetDuration, "metrics-generator.native-histogram-min-reset-duration", 15*time.Minute, "The minimum duration before a native histogram can be reset.")

	f.StringVar(&c.PerTenantOverrideConfig, "config.per-user-override-config", "", "File name of per-user Overrides.")
	_ = c.PerTenantOverridePeriod.Set("10s")
	f.Var(&c.PerTenantOverridePeriod, "config.per-user-override-period", "Period with this to reload the Overrides.")
	f.BoolVar(&c.EnableLegacyOverrides, "config.enable-legacy-overrides", false, "Enable the deprecated legacy overrides format. This is disabled by default and will be removed in a future release.")

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

func HasNativeHistograms(s histograms.HistogramMethod) bool {
	return s == histograms.HistogramMethodNative || s == histograms.HistogramMethodBoth
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

func fieldNamesFor(v any, tagKey string) map[string]struct{} {
	t := reflect.TypeOf(v)
	fields := make(map[string]struct{}, t.NumField())
	for field := range t.Fields() {
		tag := field.Tag.Get(tagKey)
		if tag == "-" || tag == ",inline" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" {
			name = field.Name
		}
		fields[name] = struct{}{}
	}
	return fields
}
