package overrides

import (
	"flag"
	"fmt"

	"github.com/prometheus/common/model"
)

type ConfigType string

const (
	ConfigTypeLegacy ConfigType = "legacy"
	ConfigTypeNew    ConfigType = "new"
)

type Config struct {
	DefaultLimits Limits `yaml:"default_limits" json:"default_limits"`

	PerTenantOverrideConfig string         `yaml:"per_tenant_override_config" json:"per_tenant_override_config"`
	PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period" json:"per_tenant_override_period"`

	ConfigType ConfigType `yaml:"-" json:"-"`
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Note: this implementation relies on callers using yaml.UnmarshalStrict. In non-strict mode
	// unmarshal() will not return an error for legacy configuration and we return immediately.

	// Try to unmarshal it normally
	type rawConfig Config
	err := unmarshal((*rawConfig)(c))
	if err == nil {
		c.ConfigType = ConfigTypeNew
		return nil
	}
	fmt.Println("unmarshal error: ", err)

	// Try to unmarshal inline limits
	type legacyConfig struct {
		DefaultLimits LegacyLimits `yaml:",inline"`

		PerTenantOverrideConfig string         `yaml:"per_tenant_override_config"`
		PerTenantOverridePeriod model.Duration `yaml:"per_tenant_override_period"`
	}
	var legacyCfg legacyConfig
	legacyCfg.DefaultLimits = c.DefaultLimits.toLegacy()
	legacyCfg.PerTenantOverrideConfig = c.PerTenantOverrideConfig
	legacyCfg.PerTenantOverridePeriod = c.PerTenantOverridePeriod

	if err := unmarshal(&legacyCfg); err != nil {
		return err
	}

	c.DefaultLimits = legacyCfg.DefaultLimits.toNewLimits()
	c.PerTenantOverrideConfig = legacyCfg.PerTenantOverrideConfig
	c.PerTenantOverridePeriod = legacyCfg.PerTenantOverridePeriod
	c.ConfigType = ConfigTypeLegacy
	return nil
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (c *Config) RegisterFlags(f *flag.FlagSet) {
	c.DefaultLimits.RegisterFlags(f)

	f.StringVar(&c.PerTenantOverrideConfig, "config.per-user-override-config", "", "File name of per-user Overrides.")
	_ = c.PerTenantOverridePeriod.Set("10s")
	f.Var(&c.PerTenantOverridePeriod, "config.per-user-override-period", "Period with this to reload the Overrides.")
}
