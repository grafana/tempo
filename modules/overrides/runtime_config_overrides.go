package overrides

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/drone/envsubst"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/runtimeconfig"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

type Validator interface {
	Validate(config *Overrides) (warnings []error, err error)
}

type TenantOverrides map[string]*Overrides

// perTenantOverrides represents the overrides config file
type perTenantOverrides struct {
	TenantLimits TenantOverrides `yaml:"overrides"`

	ConfigType ConfigType `yaml:"-"` // ConfigType is the type of overrides config we are using: legacy or new

	// Merged holds effective overrides per tenant (defaults and tenant overrides combined).
	// Built once at config load/reload time so that runtime lookups
	// via getOverridesForUser are a single map read with no per-request merge overhead.
	// NOTE: merged overrides are ready only and should not be mutated
	merged TenantOverrides `yaml:"-"`
}

func (o *perTenantOverrides) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Note: this implementation relies on callers using yaml.UnmarshalStrict. In non-strict mode
	// unmarshal() will not return an error for legacy configuration and we return immediately.

	// Try to unmarshal it normally
	type rawConfig perTenantOverrides
	if err := unmarshal((*rawConfig)(o)); err == nil {
		o.ConfigType = ConfigTypeNew
		return nil
	}

	var legacyConfig perTenantLegacyOverrides
	if err := unmarshal(&legacyConfig); err != nil {
		return err
	}

	*o = legacyConfig.toNewOverrides()
	o.ConfigType = ConfigTypeLegacy

	return nil
}

// loadPerTenantOverrides is of type runtimeconfig.Loader
// executed when the runtime overrides are reloaded each ReloadPeriod
func loadPerTenantOverrides(validator Validator, typ ConfigType, expandEnv bool, defaults *Overrides) func(r io.Reader) (interface{}, error) {
	return func(r io.Reader) (interface{}, error) {
		overrides := &perTenantOverrides{}

		if expandEnv {
			rr := r.(*bytes.Reader)
			b, err := io.ReadAll(rr)
			if err != nil {
				return nil, err
			}

			s, err := envsubst.EvalEnv(string(b))
			if err != nil {
				return nil, fmt.Errorf("failed to expand env vars: %w", err)
			}
			r = bytes.NewReader([]byte(s))
		}

		decoder := yaml.NewDecoder(r)
		decoder.SetStrict(true)
		if err := decoder.Decode(&overrides); err != nil {
			return nil, err
		}

		if overrides.ConfigType != typ {
			// TODO: Return error?
			level.Warn(log.Logger).Log(
				"msg", "Overrides config type mismatch",
				"err", "per-tenant overrides config type does not match static overrides config type",
				"config_type", overrides.ConfigType,
				"static_config_type", typ,
			)
		}

		// validate the per-tenant overrides on each reload to ensure only valid config is loaded
		if validator != nil {
			for tenant, tenantOverrides := range overrides.TenantLimits {
				warnings, err := validator.Validate(tenantOverrides)
				if err != nil {
					return nil, fmt.Errorf("validating overrides for %s failed: %w", tenant, err)
				}
				for _, warning := range warnings {
					level.Warn(log.Logger).Log("msg", "Overrides validation warning", "tenant", tenant, "warning", warning)
				}
			}
		}

		// Pre-compute merged (defaults + tenant) overrides for fast per-request lookup.
		overrides.merged = buildMergedOverrides(overrides.TenantLimits, defaults)

		return overrides, nil
	}
}

// runtimeConfigOverridesManager periodically fetch a set of per-user overrides, and provides convenience
// functions for fetching the correct value.
type runtimeConfigOverridesManager struct {
	services.Service

	defaultLimits    *Overrides
	runtimeConfigMgr *runtimeconfig.Manager

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

var _ Interface = (*runtimeConfigOverridesManager)(nil)

func newRuntimeConfigOverrides(cfg Config, validator Validator, registerer prometheus.Registerer) (Service, error) {
	// FIXME: validate cfg.Defaults through the Validator interface. Currently only per-tenant
	// overrides are validated, so invalid defaults set via static YAML config slip through.

	var manager *runtimeconfig.Manager
	subservices := []services.Service(nil)

	if cfg.PerTenantOverrideConfig != "" {
		runtimeCfg := runtimeconfig.Config{
			LoadPath:     []string{cfg.PerTenantOverrideConfig},
			ReloadPeriod: time.Duration(cfg.PerTenantOverridePeriod),
			Loader:       loadPerTenantOverrides(validator, cfg.ConfigType, cfg.ExpandEnv, &cfg.Defaults),
		}
		runtimeCfgMgr, err := runtimeconfig.New(runtimeCfg, "overrides", prometheus.WrapRegistererWithPrefix("tempo_", registerer), log.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime config manager: %w", err)
		}
		manager = runtimeCfgMgr
		subservices = append(subservices, runtimeCfgMgr)
	}

	o := &runtimeConfigOverridesManager{
		runtimeConfigMgr: manager,
		defaultLimits:    &cfg.Defaults,
	}

	if len(subservices) > 0 {
		var err error
		o.subservices, err = services.NewManager(subservices...)
		if err != nil {
			return nil, fmt.Errorf("failed to create subservices: %w", err)
		}
		o.subservicesWatcher = services.NewFailureWatcher()
		o.subservicesWatcher.WatchManager(o.subservices)
	}

	o.Service = services.NewBasicService(o.starting, o.running, o.stopping)

	return o, nil
}

func (o *runtimeConfigOverridesManager) starting(ctx context.Context) error {
	if o.subservices != nil {
		err := services.StartManagerAndAwaitHealthy(ctx, o.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices: %w", err)
		}
	}

	return nil
}

func (o *runtimeConfigOverridesManager) running(ctx context.Context) error {
	if o.subservices != nil {
		select {
		case <-ctx.Done():
			return nil
		case err := <-o.subservicesWatcher.Chan():
			return fmt.Errorf("overrides subservices failed: %w", err)
		}
	}
	<-ctx.Done()
	return nil
}

func (o *runtimeConfigOverridesManager) stopping(_ error) error {
	if o.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), o.subservices)
	}
	return nil
}

func (o *runtimeConfigOverridesManager) tenantOverrides() *perTenantOverrides {
	if o.runtimeConfigMgr == nil {
		return nil
	}
	cfg, ok := o.runtimeConfigMgr.GetConfig().(*perTenantOverrides)
	if !ok || cfg == nil {
		return nil
	}

	return cfg
}

// statusRuntimeConfig is a struct used to print the complete runtime config (defaults + overrides)
type statusRuntimeConfig struct {
	Defaults           *Overrides         `yaml:"defaults"`
	PerTenantOverrides perTenantOverrides `yaml:",inline"`
}

func (o *runtimeConfigOverridesManager) WriteStatusRuntimeConfig(w io.Writer, r *http.Request) error {
	// FIXME: double check if this code is right?? and see if we can and should use o.merged here?
	// Build effective (merged) per-tenant overrides for display.
	tenantOverrides := perTenantOverrides{TenantLimits: TenantOverrides{}}
	if pto := o.tenantOverrides(); pto != nil {
		for tenant, limits := range pto.merged {
			tenantOverrides.TenantLimits[tenant] = limits
		}
	}
	var output interface{}
	cfg := statusRuntimeConfig{
		Defaults:           o.defaultLimits,
		PerTenantOverrides: tenantOverrides,
	}

	mode := r.URL.Query().Get("mode")
	switch mode {
	case "diff":
		// The default runtime config is just an empty struct, but to make the diff work.
		// we set defaultLimits for every tenant that exists in runtime config.
		defaultCfg := perTenantOverrides{TenantLimits: TenantOverrides{}}
		for k, v := range tenantOverrides.TenantLimits {
			if v != nil {
				defaultCfg.TenantLimits[k] = o.defaultLimits
			}
		}

		cfgYaml, err := util.YAMLMarshalUnmarshal(cfg.PerTenantOverrides)
		if err != nil {
			return err
		}

		defaultCfgYaml, err := util.YAMLMarshalUnmarshal(defaultCfg)
		if err != nil {
			return err
		}

		output, err = util.DiffConfig(defaultCfgYaml, cfgYaml)
		if err != nil {
			return err
		}
	default:
		output = cfg
	}

	out, err := yaml.Marshal(output)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte("---\n"))
	if err != nil {
		return err
	}

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (o *runtimeConfigOverridesManager) GetTenantIDs() []string {
	tenantOverrides := o.tenantOverrides()
	if tenantOverrides == nil {
		return nil
	}

	limits := tenantOverrides.TenantLimits
	return slices.AppendSeq(make([]string, 0, len(limits)), maps.Keys(limits))
}

func (o *runtimeConfigOverridesManager) GetRuntimeOverridesFor(userID string) *Overrides {
	return o.getOverridesForUser(userID)
}

// getOverridesForUser returns the effective overrides for the given tenant.
// Lookup: per-tenant OR wildcard OR static defaults (first match wins).
// Each entry is pre-merged with defaults at load time, so this is a simple map lookup.
// The returned pointer must not be modified - it is shared across concurrent readers.
func (o *runtimeConfigOverridesManager) getOverridesForUser(userID string) *Overrides {
	if tenantOverrides := o.tenantOverrides(); tenantOverrides != nil {
		// check if a per-tenant override exists
		if m, ok := tenantOverrides.merged[userID]; ok {
			return m
		}
		// check if a wildcardTenant override exists
		if m, ok := tenantOverrides.merged[wildcardTenant]; ok {
			return m
		}
	}

	// return default overrides if no per tenant overrides exist
	return o.defaultLimits
}

// IngestionRateStrategy returns whether the ingestion rate limit should be individually applied
// to each distributor instance (local) or evenly shared across the cluster (global).
func (o *runtimeConfigOverridesManager) IngestionRateStrategy() IngestionRateStrategy {
	// The ingestion rate strategy can't be overridden on a per-tenant basis,
	// so here we are returning the defaults overrides
	return o.defaultLimits.Ingestion.RateStrategy
}

// MaxLocalTracesPerUser returns the maximum number of traces a user is allowed to store
// in a single ingester.
func (o *runtimeConfigOverridesManager) MaxLocalTracesPerUser(userID string) int {
	return *o.getOverridesForUser(userID).Ingestion.MaxLocalTracesPerUser
}

// MaxGlobalTracesPerUser returns the maximum number of traces a user is allowed to store
// across the cluster.
func (o *runtimeConfigOverridesManager) MaxGlobalTracesPerUser(userID string) int {
	return *o.getOverridesForUser(userID).Ingestion.MaxGlobalTracesPerUser
}

// MaxCompactionRange returns the maximum compaction window for this tenant.
func (o *runtimeConfigOverridesManager) MaxCompactionRange(userID string) time.Duration {
	return time.Duration(*o.getOverridesForUser(userID).Compaction.CompactionWindow)
}

// IngestionRateLimitBytes is the number of spans per second allowed for this tenant.
func (o *runtimeConfigOverridesManager) IngestionRateLimitBytes(userID string) float64 {
	return float64(*o.getOverridesForUser(userID).Ingestion.RateLimitBytes)
}

// IngestionBurstSizeBytes is the burst size in spans allowed for this tenant.
func (o *runtimeConfigOverridesManager) IngestionBurstSizeBytes(userID string) int {
	return *o.getOverridesForUser(userID).Ingestion.BurstSizeBytes
}

// IngestionTenantShardSize is the shard size.
func (o *runtimeConfigOverridesManager) IngestionTenantShardSize(userID string) int {
	return *o.getOverridesForUser(userID).Ingestion.TenantShardSize
}

func (o *runtimeConfigOverridesManager) IngestionMaxAttributeBytes(userID string) int {
	return *o.getOverridesForUser(userID).Ingestion.MaxAttributeBytes
}

func (o *runtimeConfigOverridesManager) IngestionArtificialDelay(userID string) (time.Duration, bool) {
	artificialDelay := o.getOverridesForUser(userID).Ingestion.ArtificialDelay
	if artificialDelay != nil {
		return *artificialDelay, true
	}
	// FIXME: check if we need to do the both bool and time.Duration??
	return 0, false
}

func (o *runtimeConfigOverridesManager) IngestionRetryInfoEnabled(userID string) bool {
	return *o.getOverridesForUser(userID).Ingestion.RetryInfoEnabled
}

// MaxBytesPerTrace returns the maximum size of a single trace in bytes allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBytesPerTrace(userID string) int {
	return *o.getOverridesForUser(userID).Global.MaxBytesPerTrace
}

// Forwarders returns the list of forwarder IDs for a user.
func (o *runtimeConfigOverridesManager) Forwarders(userID string) []string {
	return o.getOverridesForUser(userID).Forwarders
}

// MaxBytesPerTagValuesQuery returns the maximum size of a response to a tag-values query allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBytesPerTagValuesQuery(userID string) int {
	return *o.getOverridesForUser(userID).Read.MaxBytesPerTagValuesQuery
}

// MaxBlocksPerTagValuesQuery returns the maximum number of blocks to query for a tag-values query allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBlocksPerTagValuesQuery(userID string) int {
	return *o.getOverridesForUser(userID).Read.MaxBlocksPerTagValuesQuery
}

func (o *runtimeConfigOverridesManager) UnsafeQueryHints(userID string) bool {
	return *o.getOverridesForUser(userID).Read.UnsafeQueryHints
}

func (o *runtimeConfigOverridesManager) LeftPadTraceIDs(userID string) bool {
	return *o.getOverridesForUser(userID).Read.LeftPadTraceIDs
}

func (o *runtimeConfigOverridesManager) CostAttributionMaxCardinality(userID string) uint64 {
	return *o.getOverridesForUser(userID).CostAttribution.MaxCardinality
}

func (o *runtimeConfigOverridesManager) CostAttributionDimensions(userID string) map[string]string {
	return o.getOverridesForUser(userID).CostAttribution.Dimensions
}

// MaxSearchDuration is the duration of the max search duration for this tenant.
func (o *runtimeConfigOverridesManager) MaxSearchDuration(userID string) time.Duration {
	return time.Duration(*o.getOverridesForUser(userID).Read.MaxSearchDuration)
}

func (o *runtimeConfigOverridesManager) MaxMetricsDuration(userID string) time.Duration {
	return time.Duration(*o.getOverridesForUser(userID).Read.MaxMetricsDuration)
}

// MetricsGeneratorIngestionSlack is the max amount of time passed since a span's end time
// for the span to be considered in metrics generation
func (o *runtimeConfigOverridesManager) MetricsGeneratorIngestionSlack(userID string) time.Duration {
	return *o.getOverridesForUser(userID).MetricsGenerator.IngestionSlack
}

// MetricsGeneratorRemoteWriteHeaders returns the custom remote write headers for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorRemoteWriteHeaders(userID string) map[string]string {
	return o.getOverridesForUser(userID).MetricsGenerator.RemoteWriteHeaders.toStringStringMap()
}

// MetricsGeneratorRingSize is the desired size of the metrics-generator ring for this tenant.
// Using shuffle sharding, a tenant can use a smaller ring than the entire ring.
func (o *runtimeConfigOverridesManager) MetricsGeneratorRingSize(userID string) int {
	return *o.getOverridesForUser(userID).MetricsGenerator.RingSize
}

// MetricsGeneratorProcessors returns the metrics-generator processors enabled for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessors(userID string) map[string]struct{} {
	return o.getOverridesForUser(userID).MetricsGenerator.Processors.GetMap()
}

// MetricsGeneratorMaxActiveSeries is the maximum amount of active series in the metrics-generator
// registry for this tenant. Note this is a local limit enforced in every instance separately.
// Requires the generator's limiter type to be set to "series".
func (o *runtimeConfigOverridesManager) MetricsGeneratorMaxActiveSeries(userID string) uint32 {
	return *o.getOverridesForUser(userID).MetricsGenerator.MaxActiveSeries
}

// MetricsGeneratorMaxActiveEntities is the maximum number of entities in the metrics-generator registry
// for this tenant. Note this is a local limit enforced in every instance separately.
// Requires the generator's limiter type to be set to "entity".
func (o *runtimeConfigOverridesManager) MetricsGeneratorMaxActiveEntities(userID string) uint32 {
	return *o.getOverridesForUser(userID).MetricsGenerator.MaxActiveEntities
}

// MetricsGeneratorCollectionInterval is the collection interval of the metrics-generator registry
// for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorCollectionInterval(userID string) time.Duration {
	return *o.getOverridesForUser(userID).MetricsGenerator.CollectionInterval
}

// MetricsGeneratorDisableCollection controls whether metrics are remote written for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorDisableCollection(userID string) bool {
	return *o.getOverridesForUser(userID).MetricsGenerator.DisableCollection
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorGenerateNativeHistograms(userID string) histograms.HistogramMethod {
	return o.getOverridesForUser(userID).MetricsGenerator.GenerateNativeHistograms
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorNativeHistogramBucketFactor(userID string) float64 {
	return o.getOverridesForUser(userID).MetricsGenerator.NativeHistogramBucketFactor
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorNativeHistogramMaxBucketNumber(userID string) uint32 {
	return o.getOverridesForUser(userID).MetricsGenerator.NativeHistogramMaxBucketNumber
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorNativeHistogramMinResetDuration(userID string) time.Duration {
	return *o.getOverridesForUser(userID).MetricsGenerator.NativeHistogramMinResetDuration
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorSpanNameSanitization(userID string) string {
	return *o.getOverridesForUser(userID).MetricsGenerator.SpanNameSanitization
}

// MetricsGeneratorMaxCardinalityPerLabel is the maximum number of distinct values any single
// label can have before values are replaced with __cardinality_overflow__.
// 0 disables the limit.
func (o *runtimeConfigOverridesManager) MetricsGeneratorMaxCardinalityPerLabel(userID string) uint64 {
	return *o.getOverridesForUser(userID).MetricsGenerator.MaxCardinalityPerLabel
}

// MetricsGeneratorTraceIDLabelName is the label name used for the trace ID in metrics.
func (o *runtimeConfigOverridesManager) MetricsGeneratorTraceIDLabelName(userID string) string {
	return o.getOverridesForUser(userID).MetricsGenerator.TraceIDLabelName
}

// MetricsGeneratorForwarderQueueSize is the size of the buffer of requests to send to the metrics-generator
// from the distributor for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorForwarderQueueSize(userID string) int {
	return o.getOverridesForUser(userID).MetricsGenerator.Forwarder.QueueSize
}

// MetricsGeneratorForwarderWorkers is the number of workers to send metrics to the metrics-generator
func (o *runtimeConfigOverridesManager) MetricsGeneratorForwarderWorkers(userID string) int {
	return o.getOverridesForUser(userID).MetricsGenerator.Forwarder.Workers
}

// MetricsGeneratorProcessorServiceGraphsHistogramBuckets controls the histogram buckets to be used
// by the service-graphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64 {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.HistogramBuckets
}

// MetricsGeneratorProcessorServiceGraphsDimensions controls the dimensions that are added to the
// service-graphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.Dimensions
}

// MetricsGeneratorProcessorServiceGraphsPeerAttributes controls the attributes that are used to build virtual nodes
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.PeerAttributes
}

// MetricsGeneratorProcessorServiceGraphsFilterPolicies controls the filter policies that are added to the service-graphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsFilterPolicies(userID string) []filterconfig.FilterPolicy {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.FilterPolicies
}

// MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix enables "client" and "server" prefix
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID string) bool {
	return *o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.EnableClientServerPrefix
}

// MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram enables this metric
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram(userID string) (bool, bool) {
	// FIXME: see if we need to return two bools here??
	enableMessagingSystemLatencyHistogram := o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.EnableMessagingSystemLatencyHistogram
	if enableMessagingSystemLatencyHistogram != nil {
		return *enableMessagingSystemLatencyHistogram, true
	}
	return false, false
}

// MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel adds the "virtual_node" label
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel(userID string) (bool, bool) {
	// FIXME: see if we need to return two bools here??
	enableVirtualNodeLabel := o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.EnableVirtualNodeLabel
	if enableVirtualNodeLabel != nil {
		return *enableVirtualNodeLabel, true
	}
	return false, false
}

// MetricsGeneratorProcessorSpanMetricsHistogramBuckets controls the histogram buckets to be used
// by the span metrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID string) []float64 {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.HistogramBuckets
}

// MetricsGeneratorProcessorSpanMetricsDimensions controls the dimensions that are added to the
// span metrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsDimensions(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.Dimensions
}

// MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions controls the intrinsic dimensions such as service, span_kind, or
// span_name that are activated or deactivated on the span metrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID string) map[string]bool {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.IntrinsicDimensions
}

// MetricsGeneratorProcessorSpanMetricsFilterPolicies controls the filter policies that are added to the spanmetrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID string) []filterconfig.FilterPolicy {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.FilterPolicies
}

// MetricsGeneratorProcessorSpanMetricsDimensionMappings controls custom dimension mapping
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID string) []sharedconfig.DimensionMappings {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.DimensionMappings
}

// MetricsGeneratorProcessorSpanMetricsEnableTargetInfo enables target_info metrics
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID string) (bool, bool) {
	// FIXME: see if we need to return two bools here??
	enableTargetInfo := o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.EnableTargetInfo
	if enableTargetInfo != nil {
		return *enableTargetInfo, true
	}
	return false, false
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.TargetInfoExcludedDimensions
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(userID string) (bool, bool) {
	// FIXME: see if we need to return two bools here??
	EnableInstanceLabel := o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.EnableInstanceLabel
	if EnableInstanceLabel != nil {
		return *EnableInstanceLabel, true
	}
	// should this default be somewhere else like the actual defaults instead of here??
	return true, false // default to true
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorHostInfoHostIdentifiers(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.HostInfo.HostIdentifiers
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorHostInfoMetricName(userID string) string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.HostInfo.MetricName
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey(userID string) string {
	return *o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.SpanMultiplierKey
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableTraceStateSpanMultiplier(userID string) (bool, bool) {
	enableTraceStateSpanMultiplier := o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.EnableTraceStateSpanMultiplier
	if enableTraceStateSpanMultiplier != nil {
		return *enableTraceStateSpanMultiplier, true
	}
	return false, false
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey(userID string) string {
	return *o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.SpanMultiplierKey
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsEnableTraceStateSpanMultiplier(userID string) (bool, bool) {
	enableTraceStateSpanMultiplier := o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.EnableTraceStateSpanMultiplier
	if enableTraceStateSpanMultiplier != nil {
		return *enableTraceStateSpanMultiplier, true
	}
	return false, false
}

// BlockRetention is the duration of the block retention for this tenant.
func (o *runtimeConfigOverridesManager) BlockRetention(userID string) time.Duration {
	return time.Duration(*o.getOverridesForUser(userID).Compaction.BlockRetention)
}

// CompactionDisabled will not compact tenants which have this enabled.
func (o *runtimeConfigOverridesManager) CompactionDisabled(userID string) bool {
	return *o.getOverridesForUser(userID).Compaction.CompactionDisabled
}

func (o *runtimeConfigOverridesManager) DedicatedColumns(userID string) backend.DedicatedColumns {
	return o.getOverridesForUser(userID).Storage.DedicatedColumns
}

func (o *runtimeConfigOverridesManager) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricOverridesLimitsDesc
}

func (o *runtimeConfigOverridesManager) Collect(ch chan<- prometheus.Metric) {
	overrides := o.tenantOverrides()
	if overrides == nil {
		return
	}

	// Use pre-merged views so metrics reflect effective values.
	// Pointer fields dereferenced below are guaranteed non-nil after merge because
	// RegisterFlagsAndApplyDefaults allocates them in the default struct.
	for tenant, limits := range overrides.merged {
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.Ingestion.MaxLocalTracesPerUser), MetricMaxLocalTracesPerUser, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.Ingestion.MaxGlobalTracesPerUser), MetricMaxGlobalTracesPerUser, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.Ingestion.RateLimitBytes), MetricIngestionRateLimitBytes, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.Ingestion.BurstSizeBytes), MetricIngestionBurstSizeBytes, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.Global.MaxBytesPerTrace), MetricMaxBytesPerTrace, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.Compaction.BlockRetention), MetricBlockRetention, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.Compaction.CompactionWindow), MetricCompactionWindow, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, boolToFloat64(*limits.Compaction.CompactionDisabled), MetricCompactionDisabled, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(*limits.MetricsGenerator.MaxActiveSeries), MetricMetricsGeneratorMaxActiveSeries, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, boolToFloat64(*limits.MetricsGenerator.DisableCollection), MetricsGeneratorDryRunEnabled, tenant)
	}
}
