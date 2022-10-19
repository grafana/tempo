package usagestats

import (
	"flag"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	Enabled bool            `yaml:"reporting_enabled,omitempty"`
	Leader  bool            `yaml:"-"`
	Backoff *backoff.Config `yaml:"backoff,omitempty"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, util.PrefixConfig(prefix, "enabled"), true, "Enable anonymous usage reporting.")
	if cfg.Backoff == nil {
		cfg.Backoff = &backoff.Config{}
	}
	cfg.Backoff.RegisterFlagsWithPrefix(prefix, f)
	cfg.Backoff.MaxRetries = 0
}
