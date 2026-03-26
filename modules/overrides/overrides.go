package overrides

import (
	"fmt"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"

	"github.com/grafana/tempo/pkg/util/log"
)

const wildcardTenant = "*"

var tracer = otel.Tracer("modules/overrides")

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
	if cfg.ConfigType == ConfigTypeLegacy {
		// always log a warning for ConfigTypeLegacy
		level.Warn(log.Logger).Log(
			"msg", "DEPRECATED: legacy overrides config format is in use. Legacy overrides are deprecated and will be removed in a future release. "+
				"Please migrate your overrides config to the new overrides format.",
		)

		if !cfg.EnableLegacyOverrides {
			return nil, fmt.Errorf(
				"DEPRECATED: legacy overrides config format detected but legacy overrides are disabled by default. Legacy overrides will be removed in a future release. " +
					"Migrate your overrides config to the new scoped format, or set -config.enable-legacy-overrides=true (or enable_legacy_overrides: true in YAML) to continue using legacy overrides temporarily")
		}

	}

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
