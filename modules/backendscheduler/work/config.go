package work

import (
	"flag"
	"fmt"
	"time"
)

type Config struct {
	PruneAge time.Duration `yaml:"prune_age"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.PruneAge, prefix+"prune-age", time.Hour, "Age at which to prune completed jobs")
}

func ValidateConfig(cfg *Config) error {
	if cfg.PruneAge <= 0 {
		return fmt.Errorf("positive prune age required")
	}

	return nil
}
