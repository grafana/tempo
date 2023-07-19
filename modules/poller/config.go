package poller

import (
	"flag"
)

// Config is the Tempo storage configuration
type Config struct {
	// Trace tempodb.Config `yaml:"trace"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(_ string, _ *flag.FlagSet) {
	// cfg.Trace.RegisterFlagsAndApplyDefaults(prefix, f)
}
