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

	cfg.Backoff.RegisterFlagsWithPrefix(util.PrefixConfig(prefix, "backoff"), f)
	cfg.Backoff.MaxRetries = 0

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}
