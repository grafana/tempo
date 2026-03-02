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
	"github.com/prometheus/common/model"
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
func loadPerTenantOverrides(validator Validator, typ ConfigType, expandEnv bool) func(r io.Reader) (interface{}, error) {
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
	var manager *runtimeconfig.Manager
	subservices := []services.Service(nil)

	if cfg.PerTenantOverrideConfig != "" {
		runtimeCfg := runtimeconfig.Config{
			LoadPath:     []string{cfg.PerTenantOverrideConfig},
			ReloadPeriod: time.Duration(cfg.PerTenantOverridePeriod),
			Loader:       loadPerTenantOverrides(validator, cfg.ConfigType, cfg.ExpandEnv),
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

// overridesForUser returns the three levels of the override chain for field-level fallback.
// tenant and wildcard may be nil. defaults is always non-nil.
func (o *runtimeConfigOverridesManager) overridesForUser(userID string) (tenant, wildcard, defaults *Overrides) {
	defaults = o.defaultLimits
	if tenantOverrides := o.tenantOverrides(); tenantOverrides != nil {
		tenant = tenantOverrides.forUser(userID)
		wildcard = tenantOverrides.forUser(wildcardTenant)
	}
	return
}

// getOverridesForUser returns the first matching override struct for display/debug purposes.
// For field-level fallback use overridesForUser instead.
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

// Chain-based fallback helpers. Each walks tenant -> wildcard -> defaults per field.

func firstNonZero[T comparable](tenant, wildcard, defaults *Overrides, get func(*Overrides) T) T {
	var zero T
	if tenant != nil {
		if v := get(tenant); v != zero {
			return v
		}
	}
	if wildcard != nil {
		if v := get(wildcard); v != zero {
			return v
		}
	}
	return get(defaults)
}

func firstNonNilSlice[T any](tenant, wildcard, defaults *Overrides, get func(*Overrides) []T) []T {
	if tenant != nil {
		if v := get(tenant); v != nil {
			return v
		}
	}
	if wildcard != nil {
		if v := get(wildcard); v != nil {
			return v
		}
	}
	return get(defaults)
}

func firstNonNilMap[K comparable, V any](tenant, wildcard, defaults *Overrides, get func(*Overrides) map[K]V) map[K]V {
	if tenant != nil {
		if v := get(tenant); v != nil {
			return v
		}
	}
	if wildcard != nil {
		if v := get(wildcard); v != nil {
			return v
		}
	}
	return get(defaults)
}

func firstNonNilPtr[T any](tenant, wildcard, defaults *Overrides, get func(*Overrides) *T) *T {
	if tenant != nil {
		if v := get(tenant); v != nil {
			return v
		}
	}
	if wildcard != nil {
		if v := get(wildcard); v != nil {
			return v
		}
	}
	return get(defaults)
}

// IngestionRateStrategy returns whether the ingestion rate limit should be individually applied
// to each distributor instance (local) or evenly shared across the cluster (global).
func (o *runtimeConfigOverridesManager) IngestionRateStrategy() string {
	// The ingestion rate strategy can't be overridden on a per-tenant basis,
	// so here we are returning the defaults overrides
	return o.defaultLimits.Ingestion.RateStrategy
}

// MaxLocalTracesPerUser returns the maximum number of traces a user is allowed to store
// in a single ingester.
func (o *runtimeConfigOverridesManager) MaxLocalTracesPerUser(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Ingestion.MaxLocalTracesPerUser })
}

// MaxGlobalTracesPerUser returns the maximum number of traces a user is allowed to store
// across the cluster.
func (o *runtimeConfigOverridesManager) MaxGlobalTracesPerUser(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Ingestion.MaxGlobalTracesPerUser })
}

// MaxCompactionRange returns the maximum compaction window for this tenant.
func (o *runtimeConfigOverridesManager) MaxCompactionRange(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return time.Duration(firstNonZero(t, w, d, func(ov *Overrides) model.Duration { return ov.Compaction.CompactionWindow }))
}

// IngestionRateLimitBytes is the number of spans per second allowed for this tenant.
func (o *runtimeConfigOverridesManager) IngestionRateLimitBytes(userID string) float64 {
	t, w, d := o.overridesForUser(userID)
	return float64(firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Ingestion.RateLimitBytes }))
}

// IngestionBurstSizeBytes is the burst size in spans allowed for this tenant.
func (o *runtimeConfigOverridesManager) IngestionBurstSizeBytes(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Ingestion.BurstSizeBytes })
}

// IngestionTenantShardSize is the shard size.
func (o *runtimeConfigOverridesManager) IngestionTenantShardSize(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Ingestion.TenantShardSize })
}

func (o *runtimeConfigOverridesManager) IngestionMaxAttributeBytes(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Ingestion.MaxAttributeBytes })
}

func (o *runtimeConfigOverridesManager) IngestionArtificialDelay(userID string) (time.Duration, bool) {
	t, w, d := o.overridesForUser(userID)
	p := firstNonNilPtr(t, w, d, func(ov *Overrides) *time.Duration { return ov.Ingestion.ArtificialDelay })
	if p != nil {
		return *p, true
	}
	return 0, false
}

func (o *runtimeConfigOverridesManager) IngestionRetryInfoEnabled(userID string) bool {
	t, w, d := o.overridesForUser(userID)
	return derefOr(firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.Ingestion.RetryInfoEnabled }), false)
}

// MaxBytesPerTrace returns the maximum size of a single trace in bytes allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBytesPerTrace(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Global.MaxBytesPerTrace })
}

// Forwarders returns the list of forwarder IDs for a user.
func (o *runtimeConfigOverridesManager) Forwarders(userID string) []string {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []string { return ov.Forwarders })
}

// MaxBytesPerTagValuesQuery returns the maximum size of a response to a tag-values query allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBytesPerTagValuesQuery(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Read.MaxBytesPerTagValuesQuery })
}

// MaxBlocksPerTagValuesQuery returns the maximum number of blocks to query for a tag-values query allowed for a user.
func (o *runtimeConfigOverridesManager) MaxBlocksPerTagValuesQuery(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.Read.MaxBlocksPerTagValuesQuery })
}

func (o *runtimeConfigOverridesManager) UnsafeQueryHints(userID string) bool {
	t, w, d := o.overridesForUser(userID)
	return derefOr(firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.Read.UnsafeQueryHints }), false)
}

func (o *runtimeConfigOverridesManager) LeftPadTraceIDs(userID string) bool {
	t, w, d := o.overridesForUser(userID)
	return derefOr(firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.Read.LeftPadTraceIDs }), false)
}

func (o *runtimeConfigOverridesManager) CostAttributionMaxCardinality(userID string) uint64 {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) uint64 { return ov.CostAttribution.MaxCardinality })
}

func (o *runtimeConfigOverridesManager) CostAttributionDimensions(userID string) map[string]string {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilMap(t, w, d, func(ov *Overrides) map[string]string { return ov.CostAttribution.Dimensions })
}

// MaxSearchDuration is the duration of the max search duration for this tenant.
func (o *runtimeConfigOverridesManager) MaxSearchDuration(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return time.Duration(firstNonZero(t, w, d, func(ov *Overrides) model.Duration { return ov.Read.MaxSearchDuration }))
}

func (o *runtimeConfigOverridesManager) MaxMetricsDuration(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return time.Duration(firstNonZero(t, w, d, func(ov *Overrides) model.Duration { return ov.Read.MaxMetricsDuration }))
}

// MetricsGeneratorIngestionSlack is the max amount of time passed since a span's end time
// for the span to be considered in metrics generation
func (o *runtimeConfigOverridesManager) MetricsGeneratorIngestionSlack(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) time.Duration { return ov.MetricsGenerator.IngestionSlack })
}

// MetricsGeneratorRemoteWriteHeaders returns the custom remote write headers for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorRemoteWriteHeaders(userID string) map[string]string {
	t, w, d := o.overridesForUser(userID)
	for _, ov := range []*Overrides{t, w, d} {
		if ov != nil && ov.MetricsGenerator.RemoteWriteHeaders != nil {
			return ov.MetricsGenerator.RemoteWriteHeaders.toStringStringMap()
		}
	}
	return nil
}

// MetricsGeneratorRingSize is the desired size of the metrics-generator ring for this tenant.
// Using shuffle sharding, a tenant can use a smaller ring than the entire ring.
func (o *runtimeConfigOverridesManager) MetricsGeneratorRingSize(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.MetricsGenerator.RingSize })
}

// MetricsGeneratorProcessors returns the metrics-generator processors enabled for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessors(userID string) map[string]struct{} {
	t, w, d := o.overridesForUser(userID)
	// Check the raw ListToMap field directly. GetMap() lazily initializes nil maps to empty,
	// which would prevent fallback.
	m := firstNonNilMap(t, w, d, func(ov *Overrides) map[string]struct{} { return ov.MetricsGenerator.Processors })
	if m == nil {
		return map[string]struct{}{}
	}
	return m
}

// MetricsGeneratorMaxActiveSeries is the maximum amount of active series in the metrics-generator
// registry for this tenant. Note this is a local limit enforced in every instance separately.
// Requires the generator's limiter type to be set to "series".
func (o *runtimeConfigOverridesManager) MetricsGeneratorMaxActiveSeries(userID string) uint32 {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) uint32 { return ov.MetricsGenerator.MaxActiveSeries })
}

// MetricsGeneratorMaxActiveEntities is the maximum number of entities in the metrics-generator registry
// for this tenant. Note this is a local limit enforced in every instance separately.
// Requires the generator's limiter type to be set to "entity".
func (o *runtimeConfigOverridesManager) MetricsGeneratorMaxActiveEntities(userID string) uint32 {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) uint32 { return ov.MetricsGenerator.MaxActiveEntities })
}

// MetricsGeneratorCollectionInterval is the collection interval of the metrics-generator registry
// for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorCollectionInterval(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) time.Duration { return ov.MetricsGenerator.CollectionInterval })
}

// MetricsGeneratorDisableCollection controls whether metrics are remote written for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorDisableCollection(userID string) bool {
	t, w, d := o.overridesForUser(userID)
	return derefOr(firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.MetricsGenerator.DisableCollection }), false)
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorGenerateNativeHistograms(userID string) histograms.HistogramMethod {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) histograms.HistogramMethod { return ov.MetricsGenerator.GenerateNativeHistograms })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorNativeHistogramBucketFactor(userID string) float64 {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) float64 { return ov.MetricsGenerator.NativeHistogramBucketFactor })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorNativeHistogramMaxBucketNumber(userID string) uint32 {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) uint32 { return ov.MetricsGenerator.NativeHistogramMaxBucketNumber })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorNativeHistogramMinResetDuration(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) time.Duration { return ov.MetricsGenerator.NativeHistogramMinResetDuration })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorSpanNameSanitization(userID string) string {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) string { return ov.MetricsGenerator.SpanNameSanitization })
}

// MetricsGeneratorMaxCardinalityPerLabel is the maximum number of distinct values any single
// label can have before values are replaced with __cardinality_overflow__.
func (o *runtimeConfigOverridesManager) MetricsGeneratorMaxCardinalityPerLabel(userID string) uint64 {
	t, w, d := o.overridesForUser(userID)
	return derefOr(firstNonNilPtr(t, w, d, func(ov *Overrides) *uint64 { return ov.MetricsGenerator.MaxCardinalityPerLabel }), 0)
}

// MetricsGeneratorTraceIDLabelName is the label name used for the trace ID in metrics.
// "TraceID" is used if no value is provided.
func (o *runtimeConfigOverridesManager) MetricsGeneratorTraceIDLabelName(userID string) string {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) string { return ov.MetricsGenerator.TraceIDLabelName })
}

// MetricsGeneratorForwarderQueueSize is the size of the buffer of requests to send to the metrics-generator
// from the distributor for this tenant.
func (o *runtimeConfigOverridesManager) MetricsGeneratorForwarderQueueSize(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.MetricsGenerator.Forwarder.QueueSize })
}

// MetricsGeneratorForwarderWorkers is the number of workers to send metrics to the metrics-generator
func (o *runtimeConfigOverridesManager) MetricsGeneratorForwarderWorkers(userID string) int {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) int { return ov.MetricsGenerator.Forwarder.Workers })
}

// MetricsGeneratorProcessorServiceGraphsHistogramBuckets controls the histogram buckets to be used
// by the service graphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID string) []float64 {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []float64 { return ov.MetricsGenerator.Processor.ServiceGraphs.HistogramBuckets })
}

// MetricsGeneratorProcessorServiceGraphsDimensions controls the dimensions that are added to the
// service graphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsDimensions(userID string) []string {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []string { return ov.MetricsGenerator.Processor.ServiceGraphs.Dimensions })
}

// MetricsGeneratorProcessorServiceGraphsPeerAttributes controls the attributes that are used to build virtual nodes
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID string) []string {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []string { return ov.MetricsGenerator.Processor.ServiceGraphs.PeerAttributes })
}

// MetricsGeneratorProcessorServiceGraphsFilterPolicies controls the filter policies that are added to the servicegraphs processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsFilterPolicies(userID string) []filterconfig.FilterPolicy {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []filterconfig.FilterPolicy { return ov.MetricsGenerator.Processor.ServiceGraphs.FilterPolicies })
}

// MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix enables "client" and "server" prefix
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID string) bool {
	t, w, d := o.overridesForUser(userID)
	p := firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.MetricsGenerator.Processor.ServiceGraphs.EnableClientServerPrefix })
	if p != nil {
		return *p
	}
	return false
}

// MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram enables this metric
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram(userID string) (bool, bool) {
	t, w, d := o.overridesForUser(userID)
	p := firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.MetricsGenerator.Processor.ServiceGraphs.EnableMessagingSystemLatencyHistogram })
	if p != nil {
		return *p, true
	}
	return false, false
}

// MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel adds the "virtual_node" label
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel(userID string) (bool, bool) {
	t, w, d := o.overridesForUser(userID)
	p := firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.MetricsGenerator.Processor.ServiceGraphs.EnableVirtualNodeLabel })
	if p != nil {
		return *p, true
	}
	return false, false
}

// MetricsGeneratorProcessorSpanMetricsHistogramBuckets controls the histogram buckets to be used
// by the span metrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID string) []float64 {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []float64 { return ov.MetricsGenerator.Processor.SpanMetrics.HistogramBuckets })
}

// MetricsGeneratorProcessorSpanMetricsDimensions controls the dimensions that are added to the
// span metrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsDimensions(userID string) []string {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []string { return ov.MetricsGenerator.Processor.SpanMetrics.Dimensions })
}

// MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions controls the intrinsic dimensions such as service, span_kind, or
// span_name that are activated or deactivated on the span metrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID string) map[string]bool {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilMap(t, w, d, func(ov *Overrides) map[string]bool { return ov.MetricsGenerator.Processor.SpanMetrics.IntrinsicDimensions })
}

// MetricsGeneratorProcessorSpanMetricsFilterPolicies controls the filter policies that are added to the spanmetrics processor.
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID string) []filterconfig.FilterPolicy {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []filterconfig.FilterPolicy { return ov.MetricsGenerator.Processor.SpanMetrics.FilterPolicies })
}

// MetricsGeneratorProcessorSpanMetricsDimensionMappings controls custom dimension mapping
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID string) []sharedconfig.DimensionMappings {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []sharedconfig.DimensionMappings { return ov.MetricsGenerator.Processor.SpanMetrics.DimensionMappings })
}

// MetricsGeneratorProcessorSpanMetricsEnableTargetInfo enables target_info metrics
func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID string) (bool, bool) {
	t, w, d := o.overridesForUser(userID)
	p := firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.MetricsGenerator.Processor.SpanMetrics.EnableTargetInfo })
	if p != nil {
		return *p, true
	}
	return false, false
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(userID string) uint64 {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) uint64 { return ov.MetricsGenerator.Processor.LocalBlocks.MaxLiveTraces })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) time.Duration { return ov.MetricsGenerator.Processor.LocalBlocks.MaxBlockDuration })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(userID string) uint64 {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) uint64 { return ov.MetricsGenerator.Processor.LocalBlocks.MaxBlockBytes })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) time.Duration { return ov.MetricsGenerator.Processor.LocalBlocks.TraceIdlePeriod })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) time.Duration { return ov.MetricsGenerator.Processor.LocalBlocks.FlushCheckPeriod })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) time.Duration { return ov.MetricsGenerator.Processor.LocalBlocks.CompleteBlockTimeout })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID string) []string {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []string { return ov.MetricsGenerator.Processor.SpanMetrics.TargetInfoExcludedDimensions })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(userID string) (bool, bool) {
	t, w, d := o.overridesForUser(userID)
	p := firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.MetricsGenerator.Processor.SpanMetrics.EnableInstanceLabel })
	if p != nil {
		return *p, true
	}
	return true, false // default to true
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorHostInfoHostIdentifiers(userID string) []string {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []string { return ov.MetricsGenerator.Processor.HostInfo.HostIdentifiers })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorHostInfoMetricName(userID string) string {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) string { return ov.MetricsGenerator.Processor.HostInfo.MetricName })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey(userID string) string {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) string { return ov.MetricsGenerator.Processor.ServiceGraphs.SpanMultiplierKey })
}

func (o *runtimeConfigOverridesManager) MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey(userID string) string {
	t, w, d := o.overridesForUser(userID)
	return firstNonZero(t, w, d, func(ov *Overrides) string { return ov.MetricsGenerator.Processor.SpanMetrics.SpanMultiplierKey })
}

// BlockRetention is the duration of the block retention for this tenant.
func (o *runtimeConfigOverridesManager) BlockRetention(userID string) time.Duration {
	t, w, d := o.overridesForUser(userID)
	return time.Duration(firstNonZero(t, w, d, func(ov *Overrides) model.Duration { return ov.Compaction.BlockRetention }))
}

// CompactionDisabled will not compact tenants which have this enabled.
func (o *runtimeConfigOverridesManager) CompactionDisabled(userID string) bool {
	t, w, d := o.overridesForUser(userID)
	return derefOr(firstNonNilPtr(t, w, d, func(ov *Overrides) *bool { return ov.Compaction.CompactionDisabled }), false)
}

func (o *runtimeConfigOverridesManager) DedicatedColumns(userID string) backend.DedicatedColumns {
	t, w, d := o.overridesForUser(userID)
	return firstNonNilSlice(t, w, d, func(ov *Overrides) []backend.DedicatedColumn { return ov.Storage.DedicatedColumns })
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

		if limits.MetricsGenerator.DisableCollection != nil && *limits.MetricsGenerator.DisableCollection {
			ch <- prometheus.MustNewConstMetric(metricOverridesLimitsDesc, prometheus.GaugeValue, float64(1), MetricsGeneratorDryRunEnabled, tenant)
		}
	}
}
