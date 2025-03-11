package backendscheduler

import (
	"flag"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	// ScheduleInterval    time.Duration           `yaml:"schedule_interval"`
	Compactor           tempodb.CompactorConfig `yaml:"compaction"`
	Poll                bool                    `yaml:"-"`
	MaxPendingWorkQueue int                     `yaml:"max_pending_work_queue"`
	MinPendingWorkQueue int                     `yaml:"min_pending_work_queue"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// f.BoolVar(&cfg.Enabled, prefix+"backend-scheduler.enabled", false, "Enable backend scheduler")
	// f.DurationVar(&cfg.ScheduleInterval, prefix+"backend-scheduler.interval", 10*time.Second, "Interval to run the backend scheduler")
	f.IntVar(&cfg.MaxPendingWorkQueue, util.PrefixConfig(prefix, "max_pending_work_queue"), 100, "Maximum number of pending work items in the scheduler")
	f.IntVar(&cfg.MinPendingWorkQueue, util.PrefixConfig(prefix, "min_pending_work_queue"), 10, "Maximum number of pending work items in the scheduler")

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}
