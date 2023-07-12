package overrides

import (
	"flag"

	"github.com/prometheus/common/model"
)

type Config struct {
	DefaultLimits Limits `yaml:"default_limits" json:"default_limits"`

	PerTenantOverrideConfig string         `yaml:"per_tenant_override_config" json:"per_tenant_override_config"`
	PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period" json:"per_tenant_override_period"`
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Note: this implementation relies on callers using yaml.UnmarshalStrict. In non-strict mode
	// unmarshal() will not return an error for legacy configuration and we return immediately.

	// Try to unmarshal it normally
	type rawConfig Config
	if err := unmarshal((*rawConfig)(c)); err == nil {
		return nil
	}

	// Try to unmarshal inline limits
	type legacyConfig struct {
		DefaultLimits Limits `yaml:",inline"`

		PerTenantOverrideConfig string         `yaml:"per_tenant_override_config"`
		PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period"`
	}
	var legacyCfg legacyConfig
	legacyCfg.DefaultLimits = c.DefaultLimits
	legacyCfg.PerTenantOverrideConfig = c.PerTenantOverrideConfig
	legacyCfg.PerTenantOverridePeriod = c.PerTenantOverridePeriod

	if err := unmarshal(&legacyCfg); err != nil {
		return err
	}

	c.DefaultLimits = legacyCfg.DefaultLimits
	c.PerTenantOverrideConfig = legacyCfg.PerTenantOverrideConfig
	c.PerTenantOverridePeriod = legacyCfg.PerTenantOverridePeriod
	return nil
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (c *Config) RegisterFlags(f *flag.FlagSet) {
	c.DefaultLimits.RegisterFlags(f)

	f.StringVar(&c.PerTenantOverrideConfig, "config.per-user-override-config", "", "File name of per-user overrides.")
	_ = c.PerTenantOverridePeriod.Set("10s")
	f.Var(&c.PerTenantOverridePeriod, "config.per-user-override-period", "Period with this to reload the overrides.")
}
