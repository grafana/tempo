package backendscheduler

import (
	"flag"
	"time"
)

type Config struct {
	Enabled          bool          `yaml:"enabled"`
	ScheduleInterval time.Duration `yaml:"schedule_interval"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, prefix+"backend-scheduler.enabled", false, "Enable backend scheduler")
	f.DurationVar(&cfg.ScheduleInterval, prefix+"backend-scheduler.interval", 10*time.Second, "Interval to run the backend scheduler")
}
