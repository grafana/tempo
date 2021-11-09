package overrides

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/grafana/dskit/runtimeconfig"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/yaml.v2"
)

const wildcardTenant = "*"

var (
	metricOverridesLimits = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "limits_overrides",
		Help:      "Per-Tenant usage limits",
	}, []string{"limit_name", "user"})
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

	for tenant, tenantLimits := range overrides.TenantLimits {
		metricOverridesLimits.WithLabelValues("max_local_traces_per_user", tenant).Set(float64(tenantLimits.MaxLocalTracesPerUser))
		metricOverridesLimits.WithLabelValues("max_global_traces_per_user", tenant).Set(float64(tenantLimits.MaxGlobalTracesPerUser))
		metricOverridesLimits.WithLabelValues("max_bytes_per_trace", tenant).Set(float64(tenantLimits.MaxBytesPerTrace))
		metricOverridesLimits.WithLabelValues("max_search_bytes_per_trace", tenant).Set(float64(tenantLimits.MaxSearchBytesPerTrace))
		metricOverridesLimits.WithLabelValues("ingestion_rate_limit_bytes", tenant).Set(float64(tenantLimits.IngestionRateLimitBytes))
		metricOverridesLimits.WithLabelValues("ingestion_burst_size_bytes", tenant).Set(float64(tenantLimits.IngestionBurstSizeBytes))
		metricOverridesLimits.WithLabelValues("block_retention", tenant).Set(float64(tenantLimits.BlockRetention))
	}

	return overrides, nil
}

// Config is a struct used to print the complete runtime config (defaults + overrides)
type Config struct {
	Defaults           *Limits            `yaml:"defaults"`
	PerTenantOverrides perTenantOverrides `yaml:",inline"`
}

// Overrides periodically fetch a set of per-user overrides, and provides convenience
// functions for fetching the correct value.
type Overrides struct {
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
func NewOverrides(defaults Limits) (*Overrides, error) {
	var manager *runtimeconfig.Manager
	subservices := []services.Service(nil)

	if defaults.PerTenantOverrideConfig != "" {
		runtimeCfg := runtimeconfig.Config{
			LoadPath:     defaults.PerTenantOverrideConfig,
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

	o := &Overrides{
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

func (o *Overrides) starting(ctx context.Context) error {
	if o.subservices != nil {
		err := services.StartManagerAndAwaitHealthy(ctx, o.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices %w", err)
		}
	}

	return nil
}

func (o *Overrides) running(ctx context.Context) error {
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

func (o *Overrides) stopping(_ error) error {
	if o.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), o.subservices)
	}
	return nil
}

func (o *Overrides) tenantOverrides() *perTenantOverrides {
	if o.runtimeConfigMgr == nil {
		return nil
	}
	cfg, ok := o.runtimeConfigMgr.GetConfig().(*perTenantOverrides)
	if !ok || cfg == nil {
		return nil
	}

	return cfg
}

func (o *Overrides) WriteStatusRuntimeConfig(w io.Writer, r *http.Request) error {
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
func (o *Overrides) IngestionRateStrategy() string {
	// The ingestion rate strategy can't be overridden on a per-tenant basis,
	// so here we just pick the value for a not-existing user ID (empty string).
	return o.getOverridesForUser("").IngestionRateStrategy
}

// MaxLocalTracesPerUser returns the maximum number of traces a user is allowed to store
// in a single ingester.
func (o *Overrides) MaxLocalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxLocalTracesPerUser
}

// MaxGlobalTracesPerUser returns the maximum number of traces a user is allowed to store
// across the cluster.
func (o *Overrides) MaxGlobalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxGlobalTracesPerUser
}

// MaxBytesPerTrace returns the maximum size of a single trace in bytes allowed for a user.
func (o *Overrides) MaxBytesPerTrace(userID string) int {
	return o.getOverridesForUser(userID).MaxBytesPerTrace
}

// MaxSearchBytesPerTrace returns the maximum size of search data for trace (in bytes) allowed for a user.
func (o *Overrides) MaxSearchBytesPerTrace(userID string) int {
	return o.getOverridesForUser(userID).MaxSearchBytesPerTrace
}

// IngestionRateLimitBytes is the number of spans per second allowed for this tenant.
func (o *Overrides) IngestionRateLimitBytes(userID string) float64 {
	return float64(o.getOverridesForUser(userID).IngestionRateLimitBytes)
}

// IngestionBurstSizeBytes is the burst size in spans allowed for this tenant.
func (o *Overrides) IngestionBurstSizeBytes(userID string) int {
	return o.getOverridesForUser(userID).IngestionBurstSizeBytes
}

// SearchTagsAllowList is the list of tags to be extracted for search, for this tenant.
func (o *Overrides) SearchTagsAllowList(userID string) map[string]struct{} {
	return o.getOverridesForUser(userID).SearchTagsAllowList.GetMap()
}

// BlockRetention is the duration of the block retention for this tenant.
func (o *Overrides) BlockRetention(userID string) time.Duration {
	return time.Duration(o.getOverridesForUser(userID).BlockRetention)
}

func (o *Overrides) getOverridesForUser(userID string) *Limits {
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
