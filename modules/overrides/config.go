package overrides

import (
	"flag"

	"github.com/prometheus/common/model"
)

type Config struct {
	DefaultLimits Limits `yaml:",inline" json:",inline"`

	PerTenantOverrideConfig string         `yaml:"per_tenant_override_config" json:"per_tenant_override_config"`
	PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period" json:"per_tenant_override_period"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (c *Config) RegisterFlags(f *flag.FlagSet) {
	c.DefaultLimits.RegisterFlags(f)

	f.StringVar(&c.PerTenantOverrideConfig, "config.per-user-override-config", "", "File name of per-user overrides.")
	_ = c.PerTenantOverridePeriod.Set("10s")
	f.Var(&c.PerTenantOverridePeriod, "config.per-user-override-period", "Period with this to reload the overrides.")
}
