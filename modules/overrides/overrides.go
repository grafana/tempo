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

func NewOverrides(cfg Config) (Service, error) {
	overrides, err := newRuntimeConfigOverrides(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.UserConfigurableOverridesConfig.Enabled {
		// Wrap runtime config with user-config overrides module
		overrides, err = newUserConfigOverrides(&cfg.UserConfigurableOverridesConfig, overrides)
	}

	return overrides, err
}
