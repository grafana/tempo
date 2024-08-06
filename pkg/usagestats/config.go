package usagestats

import (
	"flag"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/tempo/v2/pkg/util"
)

type Config struct {
	Enabled bool           `yaml:"reporting_enabled"`
	Leader  bool           `yaml:"-"`
	Backoff backoff.Config `yaml:"backoff"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, util.PrefixConfig(prefix, "enabled"), true, "Enable anonymous usage reporting.")
	cfg.Backoff.RegisterFlagsWithPrefix(prefix, f)
	cfg.Backoff.MaxRetries = 0
}
