package work

import (
	"flag"
	"time"
)

type Config struct {
	PruneAge time.Duration `yaml:"prune_age"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.PruneAge, prefix+"prune-age", time.Hour, "Age at which to prune completed jobs")
}
