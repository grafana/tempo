package backendscheduler

import (
	"flag"
	"time"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	ScheduleInterval       time.Duration           `yaml:"schedule_interval"`
	TenantPriorityInterval time.Duration           `yaml:"tenant_priority_interval"`
	Compactor              tempodb.CompactorConfig `yaml:"compaction"`
	Work                   work.Config             `yaml:"work"`
	Poll                   bool                    `yaml:"-"`
	MaxPendingWorkQueue    int                     `yaml:"max_pending_work_queue"`
	MinPendingWorkQueue    int                     `yaml:"min_pending_work_queue"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// f.BoolVar(&cfg.Enabled, prefix+"backend-scheduler.enabled", false, "Enable backend scheduler")
	f.DurationVar(&cfg.ScheduleInterval, prefix+"backend-scheduler.schedule-interval", 10*time.Second, "Interval to add maintenance work to the work queue")
	f.DurationVar(&cfg.TenantPriorityInterval, prefix+"backend-scheduler.tenant-priority-interval", time.Minute, "Interval at which to reprioritize tenants")
	f.IntVar(&cfg.MaxPendingWorkQueue, util.PrefixConfig(prefix, "max_pending_work_queue"), 100, "Maximum number of pending work items in the scheduler")
	f.IntVar(&cfg.MinPendingWorkQueue, util.PrefixConfig(prefix, "min_pending_work_queue"), 10, "Maximum number of pending work items in the scheduler")

	cfg.Work.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "work"), f)

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}
