package bufferer

import (
	"flag"
	"os"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/util/log"
)

type Config struct {
	LifecyclerConfig ring.LifecyclerConfig        `yaml:"lifecycler,omitempty"`
	PartitionRing    ingester.PartitionRingConfig `yaml:"partition_ring" category:"experimental"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.LifecyclerConfig.RegisterFlagsWithPrefix(prefix, f, log.Logger)
	cfg.PartitionRing.RegisterFlags(prefix, f)

	hostname, err := os.Hostname()
	if err != nil {
		_ = level.Error(log.Logger).Log("msg", "failed to get hostname", "err", err)
		os.Exit(1)
	}
	f.StringVar(&cfg.LifecyclerConfig.ID, prefix+".lifecycler.ID", hostname, "ID to register in the ring.")
}
