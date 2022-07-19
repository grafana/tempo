package usagestats

import (
	"flag"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	Enabled bool           `yaml:"reporting_enabled"`
	Leader  bool           `yaml:"-"`
	Backoff backoff.Config `yaml:"backoff"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {

	f.BoolVar(&cfg.Enabled, util.PrefixConfig(prefix, "enabled"), true, "Enable anonymous usage reporting.")
	f.DurationVar(&cfg.Backoff.MaxBackoff, util.PrefixConfig(prefix, "backoff.max_backoff"), time.Minute, "maximum time to back off retry")
	f.DurationVar(&cfg.Backoff.MinBackoff, util.PrefixConfig(prefix, "backoff.min_backoff"), time.Second, "minimum time to back off retry")
	f.IntVar(&cfg.Backoff.MaxRetries, util.PrefixConfig(prefix, "backoff.max_retries"), 0, "maximum number of times to retry")
}
