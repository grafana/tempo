package backendscheduler

import (
	"flag"
	"time"
)

type Config struct {
	Enabled      bool          `yaml:"enabled"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, prefix+"backend-scheduler.enabled", false, "Enable backend scheduler")
	f.DurationVar(&cfg.PollInterval, prefix+"backend-scheduler.interval", 30*time.Second, "How often to run scheduling")
}
