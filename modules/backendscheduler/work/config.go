package work

import (
	"flag"
	"fmt"
	"time"
)

type Config struct {
	PruneAge       time.Duration `yaml:"prune_age"`
	DeadJobTimeout time.Duration `yaml:"dead_job_timeout"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.PruneAge, prefix+"prune-age", time.Hour, "Age at which to prune completed jobs")
	f.DurationVar(&cfg.DeadJobTimeout, prefix+"dead-job-timeout", 24*time.Hour, "Time after which a job is considered dead and marked as failed")
}

func ValidateConfig(cfg *Config) error {
	if cfg.PruneAge <= 0 {
		return fmt.Errorf("positive prune age required")
	}

	return nil
}
