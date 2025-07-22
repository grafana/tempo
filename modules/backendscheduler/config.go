package backendscheduler

import (
	"flag"
	"fmt"
	"time"

	"github.com/grafana/tempo/modules/backendscheduler/provider"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	Work                 work.Config   `yaml:"work"`
	Poll                 bool          `yaml:"-"`
	MaintenanceInterval  time.Duration `yaml:"maintenance_interval"`
	BackendFlushInterval time.Duration `yaml:"backend_flush_interval"`

	// Provider configs
	ProviderConfig provider.Config `yaml:"provider"`
	JobTimeout     time.Duration   `yaml:"job_timeout"`
	LocalWorkPath  string          `yaml:"local_work_path,omitempty"` // Path to store local work cache
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.MaintenanceInterval, prefix+"backend-scheduler.maintenance-interval", time.Minute, "Interval at which to perform scheduler maintenance tasks")
	f.DurationVar(&cfg.BackendFlushInterval, prefix+"backend-scheduler.backend-flush-interval", time.Minute, "Interval at which to flush the work cache to the backend storage")
	f.DurationVar(&cfg.JobTimeout, prefix+"backend-scheduler.job-timeout", 15*time.Second, "Internal duration to wait for a job before telling the worker to try again")

	f.StringVar(&cfg.LocalWorkPath, prefix+"backend-scheduler.local-work-path", "/var/tempo", "Path to store local work cache.")

	cfg.Work.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "work"), f)

	cfg.ProviderConfig.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "provider"), f)
}

func ValidateConfig(cfg *Config) error {
	if err := work.ValidateConfig(&cfg.Work); err != nil {
		return err
	}

	if err := provider.ValidateConfig(&cfg.ProviderConfig); err != nil {
		return err
	}

	// Validate the measurement interval is twice the speed of the prune interval
	// so that the when newBlockSelector is called it has enough time to delete
	// the temporary entry and know that it has been persisted to the work cache.
	// If the prune age is too short, work could get deleted before the
	// newBlockSelector is able to delete the temporary entry.
	if cfg.ProviderConfig.Compaction.MeasureInterval > cfg.Work.PruneAge/2 {
		return fmt.Errorf("provider.compaction.measure_interval must be no more than half of work.prune_age; tenant measurement should happen at least twice as often as the work prune, got %s and %s", cfg.ProviderConfig.Compaction.MeasureInterval, cfg.Work.PruneAge/2)
	}

	return nil
}
