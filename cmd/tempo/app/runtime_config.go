package app

import (
	"github.com/cortexproject/cortex/pkg/util/runtimeconfig"
	"github.com/grafana/tempo/pkg/validation"
)

func tenantLimitsFromRuntimeConfig(c *runtimeconfig.Manager) validation.TenantLimits {
	if c == nil {
		return nil
	}
	return func(userID string) *validation.Limits {
		cfg, ok := c.GetConfig().(*validation.OverridesConfig)
		if !ok || cfg == nil {
			return nil
		}

		return cfg.TenantLimits[userID]
	}
}
