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
	"gopkg.in/yaml.v2"
)

// TenantLimits is a function that returns limits for given tenant, or
// nil, if there are no tenant-specific limits.
type TenantLimits func(userID string) *Limits

const wildcardTenant = "*"

// perTenantOverrides represents the overrides config file
type perTenantOverrides struct {
	TenantLimits map[string]*Limits `yaml:"overrides"`
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

type Config struct {
	Defaults           *Limits             `yaml:"defaults"`
	PerTenantOverrides *perTenantOverrides `yaml:"overrides,omitempty"`
}

// Overrides periodically fetch a set of per-user overrides, and provides convenience
// functions for fetching the correct value.
type Overrides struct {
	services.Service

	config       *Config
	tenantLimits TenantLimits

	// Manager for subservices
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// NewOverrides makes a new Overrides.
// We store the supplied limits in a global variable to ensure per-tenant limits
// are defaulted to those values.  As such, the last call to NewOverrides will
// become the new global defaults.
func NewOverrides(defaults Limits) (*Overrides, error) {
	var (
		tenantLimits TenantLimits
		config       = &Config{
			Defaults:           &defaults,
			PerTenantOverrides: &perTenantOverrides{TenantLimits: make(map[string]*Limits)},
		}
	)
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
		perTenantOverrides, ok := runtimeCfgMgr.GetConfig().(*perTenantOverrides)
		if ok && perTenantOverrides != nil {
			config.PerTenantOverrides = perTenantOverrides
		}
		tenantLimits = tenantLimitsFromRuntimeConfig(runtimeCfgMgr)
		subservices = append(subservices, runtimeCfgMgr)
	}

	o := &Overrides{
		tenantLimits: tenantLimits,
		config:       config,
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

func (o *Overrides) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var output interface{}
		cfg := o.config
		switch r.URL.Query().Get("mode") {
		case "diff":
			// Default runtime config is just empty struct, but to make diff work,
			// we set defaultLimits for every tenant that exists in runtime config.
			defaultCfg := perTenantOverrides{}
			defaultCfg.TenantLimits = map[string]*Limits{}
			for k, v := range o.config.PerTenantOverrides.TenantLimits {
				if v != nil {
					defaultCfg.TenantLimits[k] = o.config.Defaults
				}
			}

			cfgYaml, err := util.YAMLMarshalUnmarshal(cfg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defaultCfgYaml, err := util.YAMLMarshalUnmarshal(cfg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			output, err = util.DiffConfig(defaultCfgYaml, cfgYaml)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			output = cfg
		}
		util.WriteYAMLResponse(w, output)
	}
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

// IngestionRateSpans is the number of spans per second allowed for this tenant
func (o *Overrides) IngestionRateLimitBytes(userID string) float64 {
	return float64(o.getOverridesForUser(userID).IngestionRateLimitBytes)
}

// IngestionBurstSize is the burst size in spans allowed for this tenant
func (o *Overrides) IngestionBurstSizeBytes(userID string) int {
	return o.getOverridesForUser(userID).IngestionBurstSizeBytes
}

func (o *Overrides) BlockRetention(userID string) time.Duration {
	return time.Duration(o.getOverridesForUser(userID).BlockRetention)
}

func (o *Overrides) getOverridesForUser(userID string) *Limits {
	if o.tenantLimits != nil {
		l := o.tenantLimits(userID)
		if l != nil {
			return l
		}

		l = o.tenantLimits(wildcardTenant)
		if l != nil {
			return l
		}
	}
	return o.config.Defaults
}

func tenantLimitsFromRuntimeConfig(c *runtimeconfig.Manager) TenantLimits {
	if c == nil {
		return nil
	}
	return func(userID string) *Limits {
		cfg, ok := c.GetConfig().(*perTenantOverrides)
		if !ok || cfg == nil {
			return nil
		}

		return cfg.TenantLimits[userID]
	}
}
