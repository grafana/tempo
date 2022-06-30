package usagestats

import "flag"

type Config struct {
	Enabled bool `yaml:"reporting_enabled"`
	Leader  bool `yaml:"-"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (cfg *Config) RegisterFlagsAndApplyDefaults(f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, "reporting.enabled", true, "Enable anonymous usage reporting.")
}
