package validation

import (
	"fmt"
	"io"

	"github.com/cortexproject/cortex/pkg/util/runtimeconfig"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
)

// OverridesConfig represents the overrides config file
type OverridesConfig struct {
	TenantLimits map[string]*Limits `yaml:"overrides"`
}

// loadOverridesConfig is of type runtimeconfig.Loader
func loadOverridesConfig(r io.Reader) (interface{}, error) {
	var overrides = &OverridesConfig{}

	decoder := yaml.NewDecoder(r)
	decoder.SetStrict(true)
	if err := decoder.Decode(&overrides); err != nil {
		return nil, err
	}

	return overrides, nil
}

// Overrides periodically fetch a set of per-user overrides, and provides convenience
// functions for fetching the correct value.
type Overrides struct {
	defaultLimits *Limits
	tenantLimits  TenantLimits
}

// NewOverrides makes a new Overrides.
// We store the supplied limits in a global variable to ensure per-tenant limits
// are defaulted to those values.  As such, the last call to NewOverrides will
// become the new global defaults.
func NewOverrides(defaults Limits) (*Overrides, services.Service, error) {
	var srv services.Service
	var tenantLimits TenantLimits

	if defaults.PerTenantOverrideConfig != "" {
		runtimeCfg := runtimeconfig.ManagerConfig{
			LoadPath:     defaults.PerTenantOverrideConfig,
			ReloadPeriod: defaults.PerTenantOverridePeriod,
			Loader:       loadOverridesConfig,
		}
		runtimeCfgManager, err := runtimeconfig.NewRuntimeConfigManager(runtimeCfg, prometheus.DefaultRegisterer)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create runtime config manager %w", err)
		}
		tenantLimits = tenantLimitsFromRuntimeConfig(runtimeCfgManager)
		srv = runtimeCfgManager
	}

	defaultLimits = &defaults
	return &Overrides{
		tenantLimits:  tenantLimits,
		defaultLimits: &defaults,
	}, srv, nil
}

// IngestionRateStrategy returns whether the ingestion rate limit should be individually applied
// to each distributor instance (local) or evenly shared across the cluster (global).
func (o *Overrides) IngestionRateStrategy() string {
	// The ingestion rate strategy can't be overridden on a per-tenant basis,
	// so here we just pick the value for a not-existing user ID (empty string).
	return o.getOverridesForUser("").IngestionRateStrategy
}

// MaxLocalTracesPerUser returns the maximum number of streams a user is allowed to store
// in a single ingester.
func (o *Overrides) MaxLocalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxLocalTracesPerUser
}

// MaxGlobalTracesPerUser returns the maximum number of streams a user is allowed to store
// across the cluster.
func (o *Overrides) MaxGlobalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxGlobalTracesPerUser
}

// IngestionRateSpans is the number of spans per second allowed for this tenant
func (o *Overrides) IngestionRateSpans(userID string) float64 { //jpe change?
	return float64(o.getOverridesForUser(userID).IngestionRate)
}

// IngestionBurstSpans is the burst size in spans allowed for this tenant
func (o *Overrides) IngestionBurstSpans(userID string) int { // jpe change
	return o.getOverridesForUser(userID).IngestionMaxBatchSize
}

func (o *Overrides) getOverridesForUser(userID string) *Limits {
	if o.tenantLimits != nil {
		l := o.tenantLimits(userID)
		if l != nil {
			return l
		}
	}
	return o.defaultLimits
}

func tenantLimitsFromRuntimeConfig(c *runtimeconfig.Manager) TenantLimits {
	if c == nil {
		return nil
	}
	return func(userID string) *Limits {
		cfg, ok := c.GetConfig().(*OverridesConfig)
		if !ok || cfg == nil {
			return nil
		}

		return cfg.TenantLimits[userID]
	}
}
