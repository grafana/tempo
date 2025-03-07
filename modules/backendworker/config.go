package backendworker

import (
	"flag"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	BackendSchedulerAddr string                  `yaml:"backend_scheduler_addr"`
	Interval             time.Duration           `yaml:"interval"`
	Compactor            tempodb.CompactorConfig `yaml:"compaction"`
	Poll                 bool                    `yaml:"-"`
	Backoff              backoff.Config          `yaml:"backoff"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Interval, util.PrefixConfig(prefix, "backend-worker.interval"), 10*time.Second, "Interval at which to request the next job from the backend scheduler")

	f.DurationVar(&cfg.Backoff.MinBackoff, prefix+".backoff-min-period", 100*time.Millisecond, "Minimum delay when backing off.")
	f.DurationVar(&cfg.Backoff.MaxBackoff, prefix+".backoff-max-period", 30*time.Second, "Maximum delay when backing off.")
	f.IntVar(&cfg.Backoff.MaxRetries, prefix+".backoff-retries", 0, "Number of times to backoff and retry before failing.")

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}
