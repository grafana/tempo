package overrides

import (
	"github.com/prometheus/client_golang/prometheus"
)

const wildcardTenant = "*"

var metricOverridesLimitsDesc = prometheus.NewDesc(
	"tempo_limits_overrides",
	"Resource limit overrides applied to tenants",
	[]string{"limit_name", "user"},
	nil,
)

// NewOverrides makes a new Overrides.
// We store the supplied limits in a global variable to ensure per-tenant limits
// are defaulted to those values.  As such, the last call to NewOverrides will
// become the new global defaults.
func NewOverrides(cfg Limits) (Service, error) {
	overrides, err := newRuntimeConfigOverrides(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.UserConfigurableOverrides.Enabled {
		// Wrap runtime config with user-config overrides module
		overrides, err = newUserConfigOverrides(&cfg.UserConfigurableOverrides, overrides)
	}

	return overrides, err
}
