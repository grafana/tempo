package backendworker

import (
	"flag"
	"fmt"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	BackendSchedulerAddr string                  `yaml:"backend_scheduler_addr"`
	Compactor            tempodb.CompactorConfig `yaml:"compaction"`
	Poll                 bool                    `yaml:"-"`
	Backoff              backoff.Config          `yaml:"backoff"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Backoff.MinBackoff, prefix+".backoff-min-period", 100*time.Millisecond, "Minimum delay when backing off.")
	f.DurationVar(&cfg.Backoff.MaxBackoff, prefix+".backoff-max-period", time.Minute, "Maximum delay when backing off.")
	f.IntVar(&cfg.Backoff.MaxRetries, prefix+".backoff-retries", 0, "Number of times to backoff and retry before failing.")

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}

func ValidateConfig(cfg *Config) error {
	if cfg.BackendSchedulerAddr == "" {
		return fmt.Errorf("backend scheduler address is required")
	}

	if cfg.Backoff.MinBackoff <= 0 {
		return fmt.Errorf("positive backoff min period required")
	}

	if cfg.Backoff.MaxBackoff <= 0 {
		return fmt.Errorf("positive backoff max period required")
	}

	if cfg.Backoff.MaxRetries < 0 {
		return fmt.Errorf("positive backoff retries required")
	}

	return nil
}
