package overrides

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/grafana/dskit/runtimeconfig"
	"github.com/grafana/dskit/services"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
)

const wildcardTenant = "*"

var (
	metricOverridesLimitsDesc = prometheus.NewDesc(
		"tempo_limits_overrides",
		"Resource limit overrides applied to tenants",
		[]string{"limit_name", "user"},
		nil,
	)
)

// perTenantOverrides represents the overrides config file
type perTenantOverrides struct {
	TenantLimits map[string]*Limits `yaml:"overrides"`
}

// forUser returns limits for a given tenant, or nil if there are no tenant-specific limits.
func (o *perTenantOverrides) forUser(userID string) *Limits {
	l, ok := o.TenantLimits[userID]
	if !ok || l == nil {
		return nil
	}
	return l
}

// loadPerTenantOverrides is of type runtimeconfig.Loader
func loadPerTenantOverrides(r io.Reader) (interface{}, error) {
	var overrides = &perTenantOverrides{}

	decoder := yaml.NewDecoder(r)
	decoder.SetStrict(true)
	if err := decoder.Decode(&overrides); err != nil {
		return nil, err
	}

	return overrides, nil
}

// Config is a struct used to print the complete runtime config (defaults + overrides)
type Config struct {
	Defaults           *Limits            `yaml:"defaults"`
	PerTenantOverrides perTenantOverrides `yaml:",inline"`
}

// overrides periodically fetch a set of per-user overrides, and provides convenience
// functions for fetching the correct value.
type overrides struct {
	services.Service

	defaultLimits    *Limits
	runtimeConfigMgr *runtimeconfig.Manager

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// NewOverrides makes a new Overrides.
// We store the supplied limits in a global variable to ensure per-tenant limits
// are defaulted to those values.  As such, the last call to NewOverrides will
// become the new global defaults.
func NewOverrides(defaults Limits) (Service, error) {
	var manager *runtimeconfig.Manager
	subservices := []services.Service(nil)

	if defaults.PerTenantOverrideConfig != "" {
		runtimeCfg := runtimeconfig.Config{
			LoadPath:     []string{defaults.PerTenantOverrideConfig},
			ReloadPeriod: time.Duration(defaults.PerTenantOverridePeriod),
			Loader:       loadPerTenantOverrides,
		}
		runtimeCfgMgr, err := runtimeconfig.New(runtimeCfg, prometheus.WrapRegistererWithPrefix("tempo_", prometheus.DefaultRegisterer), log.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime config manager %w", err)
		}
		manager = runtimeCfgMgr
		subservices = append(subservices, runtimeCfgMgr)
	}

	o := &overrides{
		runtimeConfigMgr: manager,
		defaultLimits:    &defaults,
	}

	if len(subservices) > 0 {
		var err error
		o.subservices, err = services.NewManager(subservices...)
		if err != nil {
			return nil, fmt.Errorf("failed to create subservices %w", err)
		}
		o.subservicesWatcher = services.NewFailureWatcher()
		o.subservicesWatcher.WatchManager(o.subservices)
	}

	o.Service = services.NewBasicService(o.starting, o.running, o.stopping)

	return o, nil
}

func (o *overrides) starting(ctx context.Context) error {
	if o.subservices != nil {
		err := services.StartManagerAndAwaitHealthy(ctx, o.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices %w", err)
		}
	}

	return nil
}

func (o *overrides) running(ctx context.Context) error {
	if o.subservices != nil {
		select {
		case <-ctx.Done():
			return nil
		case err := <-o.subservicesWatcher.Chan():
			return fmt.Errorf("overrides subservices failed %w", err)
		}
	}
	<-ctx.Done()
	return nil
}

func (o *overrides) stopping(_ error) error {
	if o.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), o.subservices)
	}
	return nil
}

func (o *overrides) tenantOverrides() *perTenantOverrides {
	if o.runtimeConfigMgr == nil {
		return nil
	}
	cfg, ok := o.runtimeConfigMgr.GetConfig().(*perTenantOverrides)
	if !ok || cfg == nil {
		return nil
	}

	return cfg
}

func (o *overrides) WriteStatusRuntimeConfig(w io.Writer, r *http.Request) error {
	var tenantOverrides perTenantOverrides
	if o.tenantOverrides() != nil {
		tenantOverrides = *o.tenantOverrides()
	}
	var output interface{}
	cfg := Config{
		Defaults:           o.defaultLimits,
		PerTenantOverrides: tenantOverrides,
	}

	mode := r.URL.Query().Get("mode")
	switch mode {
	case "diff":
		// Default runtime config is just empty struct, but to make diff work,
		// we set defaultLimits for every tenant that exists in runtime config.
		defaultCfg := perTenantOverrides{TenantLimits: map[string]*Limits{}}
		defaultCfg.TenantLimits = map[string]*Limits{}
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

// IngestionRateStrategy returns whether the ingestion rate limit should be individually applied
// to each distributor instance (local) or evenly shared across the cluster (global).
func (o *overrides) IngestionRateStrategy() string {
	// The ingestion rate strategy can't be overridden on a per-tenant basis,
	// so here we just pick the value for a not-existing user ID (empty string).
	return o.getOverridesForUser("").IngestionRateStrategy
}

// MaxLocalTracesPerUser returns the maximum number of traces a user is allowed to store
// in a single ingester.
func (o *overrides) MaxLocalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxLocalTracesPerUser
}

// MaxGlobalTracesPerUser returns the maximum number of traces a user is allowed to store
// across the cluster.
func (o *overrides) MaxGlobalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxGlobalTracesPerUser
}

// MaxBytesPerTrace returns the maximum size of a single trace in bytes allowed for a user.
func (o *overrides) MaxBytesPerTrace(userID string) int {
	return o.getOverridesForUser(userID).MaxBytesPerTrace
}

// Forwarders returns the list of forwarder IDs for a user.
func (o *overrides) Forwarders(userID string) []string {
	return o.getOverridesForUser(userID).Forwarders
}

// MaxBytesPerTagValuesQuery returns the maximum size of a response to a tag-values query allowed for a user.
func (o *overrides) MaxBytesPerTagValuesQuery(userID string) int {
	return o.getOverridesForUser(userID).MaxBytesPerTagValuesQuery
}

// MaxBlocksPerTagValuesQuery returns the maximum number of blocks to query for a tag-values query allowed for a user.
func (o *overrides) MaxBlocksPerTagValuesQuery(userID string) int {
	return o.getOverridesForUser(userID).MaxBlocksPerTagValuesQuery
}

// IngestionRateLimitBytes is the number of spans per second allowed for this tenant.
func (o *overrides) IngestionRateLimitBytes(userID string) float64 {
	return float64(o.getOverridesForUser(userID).IngestionRateLimitBytes)
}

// IngestionBurstSizeBytes is the burst size in spans allowed for this tenant.
func (o *overrides) IngestionBurstSizeBytes(userID string) int {
	return o.getOverridesForUser(userID).IngestionBurstSizeBytes
}

// MetricsGeneratorRingSize is the desired size of the metrics-generator ring for this tenant.
// Using shuffle sharding, a tenant can use a smaller ring than the entire ring.
func (o *overrides) MetricsGeneratorRingSize(userID string) int {
	return o.getOverridesForUser(userID).MetricsGeneratorRingSize
}

// MetricsGeneratorProcessors returns the metrics-generator processors enabled for this tenant.
func (o *overrides) MetricsGeneratorProcessors(userID string) map[string]struct{} {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessors.GetMap()
}

// MetricsGeneratorMaxActiveSeries is the maximum amount of active series in the metrics-generator
// registry for this tenant. Note this is a local limit enforced in every instance separately.
func (o *overrides) MetricsGeneratorMaxActiveSeries(userID string) uint32 {
	return o.getOverridesForUser(userID).MetricsGeneratorMaxActiveSeries
}

// MetricsGeneratorCollectionInterval is the collection interval of the metrics-generator registry
// for this tenant.
func (o *overrides) MetricsGeneratorCollectionInterval(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGeneratorCollectionInterval
}

// MetricsGeneratorDisableCollection controls whether metrics are remote written for this tenant.
func (o *overrides) MetricsGeneratorDisableCollection(userID string) bool {
	return o.getOverridesForUser(userID).MetricsGeneratorDisableCollection
}

// MetricsGeneratorForwarderQueueSize is the size of the buffer of requests to send to the metrics-generator
// from the distributor for this tenant.
func (o *overrides) MetricsGeneratorForwarderQueueSize(userID string) int {
	return o.getOverridesForUser(userID).MetricsGeneratorForwarderQueueSize
}

// MetricsGeneratorForwarderWorkers is the number of workers to send metrics to the metrics-generator
func (o *overrides) MetricsGeneratorForwarderWorkers(userID string) int {
	return o.getOverridesForUser(userID).MetricsGeneratorForwarderWorkers
}

// MetricsGeneratorProcessorServiceGraphsHistogramBuckets controls the histogram buckets to be used
// by the service graphs processor.
func (o *overrides) MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64 {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorServiceGraphsHistogramBuckets
}

// MetricsGeneratorProcessorServiceGraphsDimensions controls the dimensions that are added to the
// service graphs processor.
func (o *overrides) MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorServiceGraphsDimensions
}

// MetricsGeneratorProcessorServiceGraphsPeerAttributes controls the attributes that are used to build virtual nodes
func (o *overrides) MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorServiceGraphsPeerAttributes
}

// MetricsGeneratorProcessorSpanMetricsHistogramBuckets controls the histogram buckets to be used
// by the span metrics processor.
func (o *overrides) MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID string) []float64 {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorSpanMetricsHistogramBuckets
}

// MetricsGeneratorProcessorSpanMetricsDimensions controls the dimensions that are added to the
// span metrics processor.
func (o *overrides) MetricsGeneratorProcessorSpanMetricsDimensions(userID string) []string {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorSpanMetricsDimensions
}

// MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions controls the intrinsic dimensions such as service, span_kind, or
// span_name that are activated or deactivated on the span metrics processor.
func (o *overrides) MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID string) map[string]bool {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions
}

// MetricsGeneratorProcessorSpanMetricsFilterPolicies controls the filter policies that are added to the spanmetrics processor.
func (o *overrides) MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID string) []filterconfig.FilterPolicy {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorSpanMetricsFilterPolicies
}

func (o *overrides) MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(userID string) uint64 {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorLocalBlocksMaxLiveTraces
}

func (o *overrides) MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorLocalBlocksMaxBlockDuration
}

func (o *overrides) MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(userID string) uint64 {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorLocalBlocksMaxBlockBytes
}

func (o *overrides) MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod
}

func (o *overrides) MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod
}

func (o *overrides) MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(userID string) time.Duration {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout
}

// MetricsGeneratorProcessorSpanMetricsDimensionMappings controls custom dimension mapping
func (o *overrides) MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID string) []sharedconfig.DimensionMappings {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorSpanMetricsDimensionMappings
}

// MetricsGeneratorProcessorSpanMetricsEnableTargetInfo enables target_info metrics
func (o *overrides) MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID string) bool {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorSpanMetricsEnableTargetInfo
}

// MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix enables "client" and "server" prefix
func (o *overrides) MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID string) bool {
	return o.getOverridesForUser(userID).MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix
}

// BlockRetention is the duration of the block retention for this tenant.
func (o *overrides) BlockRetention(userID string) time.Duration {
	return time.Duration(o.getOverridesForUser(userID).BlockRetention)
}

// MaxSearchDuration is the duration of the max search duration for this tenant.
func (o *overrides) MaxSearchDuration(userID string) time.Duration {
	return time.Duration(o.getOverridesForUser(userID).MaxSearchDuration)
}

func (o *overrides) getOverridesForUser(userID string) *Limits {
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

func (o *overrides) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricOverridesLimitsDesc
}

func (o *overrides) Collect(ch chan<- prometheus.Metric) {
	overrides := o.tenantOverrides()
	if overrides == nil {
		return
	}

	for tenant, limits := range overrides.TenantLimits {
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.MaxLocalTracesPerUser), MetricMaxLocalTracesPerUser, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.MaxGlobalTracesPerUser), MetricMaxGlobalTracesPerUser, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.MaxBytesPerTrace), MetricMaxBytesPerTrace, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.IngestionRateLimitBytes), MetricIngestionRateLimitBytes, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.IngestionBurstSizeBytes), MetricIngestionBurstSizeBytes, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.BlockRetention), MetricBlockRetention, tenant)
		ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(limits.MetricsGeneratorMaxActiveSeries), MetricMetricsGeneratorMaxActiveSeries, tenant)

		if limits.MetricsGeneratorDisableCollection {
			ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(1), MetricsGeneratorDryRunEnabled, tenant)
		}
	}
}
