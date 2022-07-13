package usagestats

import (
	"flag"
	"time"

	"github.com/grafana/dskit/backoff"
)

type Config struct {
	Enabled bool           `yaml:"reporting_enabled"`
	Leader  bool           `yaml:"-"`
	Backoff backoff.Config `yaml:"backoff"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (cfg *Config) RegisterFlagsAndApplyDefaults(f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, "reporting.enabled", true, "Enable anonymous usage reporting.")
	f.DurationVar(&cfg.Backoff.MaxBackoff, "reporting.backoff.max_backoff", time.Minute, "maximum time to back off retry")
	f.DurationVar(&cfg.Backoff.MinBackoff, "reporting.backoff.min_backoff", time.Second, "minimum time to back off retry")
	f.IntVar(&cfg.Backoff.MaxRetries, "reporting.backoff.max_retries", 0, "maximum number of times to retry")
}
