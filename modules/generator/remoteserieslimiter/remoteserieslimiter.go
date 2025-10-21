package remoteserieslimiter

import (
	"flag"
	"os"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/generator/remoteserieslimiter/usagetrackerclient"
)

type Config struct {
	Enabled                   bool                                   `yaml:"enabled"`
	UsageTrackerRing          usagetrackerclient.InstanceRingConfig  `yaml:"usage_tracker_ring"`
	UsageTrackerPartitionRing usagetrackerclient.PartitionRingConfig `yaml:"usage_tracker_partition_ring"`
	UsageTrackerClientCfg     usagetrackerclient.Config              `yaml:"usage_tracker_client"`
}

func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, "remote-series-limiter.enabled", false, "Enable remote series limiter.")
	cfg.UsageTrackerRing.RegisterFlags(f, log.NewLogfmtLogger(os.Stderr))
	cfg.UsageTrackerPartitionRing.RegisterFlags(f)
	cfg.UsageTrackerClientCfg.RegisterFlags(f)
}
