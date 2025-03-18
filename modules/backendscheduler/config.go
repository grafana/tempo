package backendscheduler

import (
	"flag"
	"time"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	TenantMeasurementInterval time.Duration           `yaml:"tenant_measurement_interval"`
	Compactor                 tempodb.CompactorConfig `yaml:"compaction"`
	Work                      work.Config             `yaml:"work"`
	Poll                      bool                    `yaml:"-"`
	MaxJobsPerTenant          int                     `yaml:"max_jobs_per_tenant"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// f.BoolVar(&cfg.Enabled, prefix+"backend-scheduler.enabled", false, "Enable backend scheduler")
	f.DurationVar(&cfg.TenantMeasurementInterval, prefix+"backend-scheduler.tenant-measurement-interval", time.Minute, "Interval at which to measure outstanding blocks")
	f.IntVar(&cfg.MaxJobsPerTenant, prefix+"backend-scheduler.max-jobs-per-tenant", 1000, "Maximum number of jobs to run per tenant before moving on to the next tenant")

	cfg.Work.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "work"), f)

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}

func ValidateConfig(cfg *Config) error {
	if err := work.ValidateConfig(&cfg.Work); err != nil {
		return err
	}

	return nil
}
