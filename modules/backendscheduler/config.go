package backendscheduler

import (
	"flag"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	ScheduleInterval time.Duration           `yaml:"schedule_interval"`
	Compactor        tempodb.CompactorConfig `yaml:"compaction"`
	Poll             bool                    `yaml:"-"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// f.BoolVar(&cfg.Enabled, prefix+"backend-scheduler.enabled", false, "Enable backend scheduler")
	f.DurationVar(&cfg.ScheduleInterval, prefix+"backend-scheduler.interval", 10*time.Second, "Interval to run the backend scheduler")

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)

	spew.Dump("scheduler after", cfg)
}
