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
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// f.BoolVar(&cfg.Enabled, prefix+"backend-scheduler.enabled", false, "Enable backend scheduler")
	f.DurationVar(&cfg.TenantMeasurementInterval, prefix+"backend-scheduler.tenant-measurement-interval", time.Minute, "Interval at which to measure outstanding blocks")

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
