package overrides

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
)

const wildcardTenant = "*"

var tracer = otel.Tracer("overrides")

var metricOverridesLimitsDesc = prometheus.NewDesc(
	"tempo_limits_overrides",
	"Resource limit overrides applied to tenants",
	[]string{"limit_name", "user"},
	nil,
)

// NewOverrides makes a new Overrides service.
// We store the supplied overrides in a global variable to ensure per-tenant overrides
// are defaulted to those values.  As such, the last call to NewOverrides will
// become the new global defaults.
func NewOverrides(cfg Config, validator Validator, registerer prometheus.Registerer) (Service, error) {
	o, err := newRuntimeConfigOverrides(cfg, validator, registerer)
	if err != nil {
		return nil, err
	}

	if cfg.UserConfigurableOverridesConfig.Enabled {
		// Wrap runtime config with user-config overrides module
		o, err = newUserConfigOverrides(&cfg.UserConfigurableOverridesConfig, o)
	}

	return o, err
}
