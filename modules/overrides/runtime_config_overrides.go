package overrides

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/drone/envsubst"
	"github.com/go-kit/log/level"
	"golang.org/x/exp/maps"

	"github.com/grafana/dskit/runtimeconfig"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

// perTenantOverrides represents the overrides config file
type perTenantOverrides struct {
	TenantLimits map[string]*Overrides `yaml:"overrides"`

	ConfigType ConfigType `yaml:"-"` // ConfigType is the type of overrides config we are using: legacy or new
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

// forUser returns limits for a given tenant, or nil if there are no tenant-specific limits.
func (o *perTenantOverrides) forUser(userID string) *Overrides {
	l, ok := o.TenantLimits[userID]
	if !ok || l == nil {
		return nil
	}
	return l
}

// loadPerTenantOverrides is of type runtimeconfig.Loader
func loadPerTenantOverrides(typ ConfigType, expandEnv bool) func(r io.Reader) (interface{}, error) {
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

func newRuntimeConfigOverrides(cfg Config, registerer prometheus.Registerer) (Service, error) {
	var manager *runtimeconfig.Manager
	subservices := []services.Service(nil)

	if cfg.PerTenantOverrideConfig != "" {
		runtimeCfg := runtimeconfig.Config{
			LoadPath:     []string{cfg.PerTenantOverrideConfig},
			ReloadPeriod: time.Duration(cfg.PerTenantOverridePeriod),
			Loader:       loadPerTenantOverrides(cfg.ConfigType, cfg.ExpandEnv),
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
	var tenantOverrides perTenantOverrides
	if o.tenantOverrides() != nil {
		tenantOverrides = *o.tenantOverrides()
	}
	var output interface{}
	cfg := statusRuntimeConfig{
		Defaults:           o.defaultLimits,
		PerTenantOverrides: tenantOverrides,
	}

	mode := r.URL.Query().Get("mode")
	switch mode {
	case "diff":
		// Default runtime config is just empty struct, but to make diff work,
		// we set defaultLimits for every tenant that exists in runtime config.
		defaultCfg := perTenantOverrides{TenantLimits: map[string]*Overrides{}}
		defaultCfg.TenantLimits = map[string]*Overrides{}
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

	return maps.Keys(tenantOverrides.TenantLimits)
}

func (o *runtimeConfigOverridesManager) GetRuntimeOverridesFor(userID string) *Overrides {
	return o.getOverridesForUser(userID)
}

// IngestionRateStrategy returns whether the ingestion rate limit should be individually applied
// to each distributor instance (local) or evenly shared across the cluster (global).
func (o *runtimeConfigOverridesManager) IngestionRateStrategy() string {
	// The ingestion rate strategy can't be overridden on a per-tenant basis,
	// so here we just pick the value for a not-existing user ID (empty string).
	return o.getOverridesForUser("").Ingestion.RateStrategy
}

// MaxLocalTracesPerUser returns the maximum number of traces a user is allowed to store
// in a single ingester.
func (o *runtimeConfigOverridesManager) MaxLocalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).Ingestion.MaxLocalTracesPerUser
}

// MaxGlobalTracesPerUser returns the maximum number of traces a user is allowed to store
// across the cluster.
func (o *runtimeConfigOverridesManager) MaxGlobalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).Ingestion.MaxGlobalTracesPerUser
}

// MaxCompactionRange returns the maximum compaction window for this tenant.
func (o *runtimeConfigOverridesManager) MaxCompactionRange(userID string) time.Duration {
	return time.Duration(o.getOverridesForUser(userID).Compaction.CompactionWindow)
}

// IngestionRateLimitBytes is the number of spans per second allowed for this tenant.
func (o *runtimeConfigOverridesManager) IngestionRateLimitBytes(userID string) float64 {
	return float64(o.getOverridesForUser(userID).Ingestion.RateLimitBytes)
}

// IngestionBurstSizeBytes is the burst size in spans allowed for this tenant.
func (o *runtimeConfigOverridesManager) IngestionBurstSizeBytes(userID string) int {
	return o.getOverridesForUser(userID).Ingestion.BurstSizeBytes
}

// MaxBytesPerTrace returns the maximum size of a single trace in bytes allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBytesPerTrace(userID string) int {
	return o.getOverridesForUser(userID).Global.MaxBytesPerTrace
}

// Forwarders returns the list of forwarder IDs for a user.
func (o *runtimeConfigOverridesManager) Forwarders(userID string) []string {
	return o.getOverridesForUser(userID).Forwarders
}

// MaxBytesPerTagValuesQuery returns the maximum size of a response to a tag-values query allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBytesPerTagValuesQuery(userID string) int {
	return o.getOverridesForUser(userID).Read.MaxBytesPerTagValuesQuery
}

// MaxBlocksPerTagValuesQuery returns the maximum number of blocks to query for a tag-values query allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBlocksPerTagValuesQuery(userID string) int {
	return o.getOverridesForUser(userID).Read.MaxBlocksPerTagValuesQuery
}

// MaxSearchDuration is the duration of the max search duration for this tenant.
func (o *runtimeConfigOverridesManager) MaxSearchDuration(userID string) time.Duration {
	return time.Duration(o.getOverridesForUser(userID).Read.MaxSearchDuration)
}

// MetricsGeneratorIngestionSlack is the max amount of time passed since a span's end time
// for the span to be considered in metrics generation
func (o *runtimeConfigOverridesManager) MetricsGeneratorIngestionSlack(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGenerator.IngestionSlack
}

// MetricsGeneratorRemoteWriteHeaders returns the custom remote write headers for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorRemoteWriteHeaders(userID string) map[string]string {
	return o.getOverridesForUser(userID).MetricsGenerator.RemoteWriteHeaders.toStringStringMap()
}

// MetricsGeneratorRingSize is the desired size of the metrics-generator ring for this tenant.
// Using shuffle sharding, a tenant can use a smaller ring than the entire ring.
func (o *runtimeConfigOverridesManager) MetricsGeneratorRingSize(userID string) int {
	return o.getOverridesForUser(userID).MetricsGenerator.RingSize
}

// MetricsGeneratorProcessors returns the metrics-generator processors enabled for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessors(userID string) map[string]struct{} {
	return o.getOverridesForUser(userID).MetricsGenerator.Processors.GetMap()
}

// MetricsGeneratorMaxActiveSeries is the maximum amount of active series in the metrics-generator
// registry for this tenant. Note this is a local limit enforced in every instance separately.
func (o *runtimeConfigOverridesManager) MetricsGeneratorMaxActiveSeries(userID string) uint32 {
	return o.getOverridesForUser(userID).MetricsGenerator.MaxActiveSeries
}

// MetricsGeneratorCollectionInterval is the collection interval of the metrics-generator registry
// for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorCollectionInterval(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGenerator.CollectionInterval
}

// MetricsGeneratorDisableCollection controls whether metrics are remote written for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorDisableCollection(userID string) bool {
	return o.getOverridesForUser(userID).MetricsGenerator.DisableCollection
}

// MetricsGenerationTraceIDLabelName is the label name used for the trace ID in metrics.
// "TraceID" is used if no value is provided.
func (o *runtimeConfigOverridesManager) MetricsGenerationTraceIDLabelName(userID string) string {
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
// by the service graphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64 {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.HistogramBuckets
}

// MetricsGeneratorProcessorServiceGraphsDimensions controls the dimensions that are added to the
// service graphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.Dimensions
}

// MetricsGeneratorProcessorServiceGraphsPeerAttributes controls the attributes that are used to build virtual nodes
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.PeerAttributes
}

// MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix enables "client" and "server" prefix
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID string) bool {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.ServiceGraphs.EnableClientServerPrefix
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
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID string) bool {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.EnableTargetInfo
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(userID string) uint64 {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.LocalBlocks.MaxLiveTraces
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.LocalBlocks.MaxBlockDuration
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(userID string) uint64 {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.LocalBlocks.MaxBlockBytes
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.LocalBlocks.TraceIdlePeriod
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.LocalBlocks.FlushCheckPeriod
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.LocalBlocks.CompleteBlockTimeout
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGenerator.Processor.SpanMetrics.TargetInfoExcludedDimensions
}

// BlockRetention is the duration of the block retention for this tenant.
func (o *runtimeConfigOverridesManager) BlockRetention(userID string) time.Duration {
	return time.Duration(o.getOverridesForUser(userID).Compaction.BlockRetention)
}

func (o *runtimeConfigOverridesManager) DedicatedColumns(userID string) backend.DedicatedColumns {
	return o.getOverridesForUser(userID).Storage.DedicatedColumns
}

func (o *runtimeConfigOverridesManager) getOverridesForUser(userID string) *Overrides {
	if tenantOverrides := o.tenantOverrides(); tenantOverrides != nil {
		l := tenantOverrides.forUser(userID)
		if l != nil {
			return l
		}

		l = tenantOverrides.forUser(wildcardTenant)
		if l != nil {
			return l
		}
	}

	return o.defaultLimits
}

func (o *runtimeConfigOverridesManager) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricOverridesLimitsDesc
}

func (o *runtimeConfigOverridesManager) Collect(ch chan<- prometheus.Metric) {
	overrides := o.tenantOverrides()
	if overrides == nil {
		return
	}

	for tenant, limits := range overrides.TenantLimits {
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.Ingestion.MaxLocalTracesPerUser), MetricMaxLocalTracesPerUser, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.Ingestion.MaxGlobalTracesPerUser), MetricMaxGlobalTracesPerUser, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.Ingestion.RateLimitBytes), MetricIngestionRateLimitBytes, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.Ingestion.BurstSizeBytes), MetricIngestionBurstSizeBytes, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.Global.MaxBytesPerTrace), MetricMaxBytesPerTrace, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.Compaction.BlockRetention), MetricBlockRetention, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.MetricsGenerator.MaxActiveSeries), MetricMetricsGeneratorMaxActiveSeries, tenant)

		if limits.MetricsGenerator.DisableCollection {
			ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(1), MetricsGeneratorDryRunEnabled, tenant)
		}
	}
}
