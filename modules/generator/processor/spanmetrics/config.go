package spanmetrics

import (
	"flag"
	"time"
)

type Config struct {
	// Duration after which to delete an inactive metric. A metric series inactive if it hasn't been
	// updated anymore.
	// Default: 2m
	DeleteAfterLastUpdate time.Duration `yaml:"delete_after_last_update"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.DeleteAfterLastUpdate = 2 * time.Minute
}
