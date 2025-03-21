package backendscheduler

import (
	"flag"
	"fmt"
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
	MaintenanceInterval       time.Duration           `yaml:"maintenance_interval"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.TenantMeasurementInterval, prefix+"backend-scheduler.tenant-measurement-interval", time.Minute, "Interval at which to measure outstanding blocks")
	f.IntVar(&cfg.MaxJobsPerTenant, prefix+"backend-scheduler.max-jobs-per-tenant", 1000, "Maximum number of jobs to run per tenant before moving on to the next tenant")
	f.DurationVar(&cfg.MaintenanceInterval, prefix+"backend-scheduler.maintenance-interval", time.Minute, "Interval at which to perform maintenance tasks")

	cfg.Work.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "work"), f)

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}

func ValidateConfig(cfg *Config) error {
	if cfg.TenantMeasurementInterval <= 0 {
		return fmt.Errorf("tenant_measurement_interval must be greater than 0")
	}

	if cfg.MaxJobsPerTenant <= 0 {
		return fmt.Errorf("max_jobs_per_tenant must be greater than 0")
	}

	if err := work.ValidateConfig(&cfg.Work); err != nil {
		return err
	}

	return nil
}
