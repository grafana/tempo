package generator

import (
	"flag"
	"os"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/ring"
	"github.com/prometheus/prometheus/config"
)

// Config for a generator.
type Config struct {
	LifecyclerConfig ring.LifecyclerConfig `yaml:"lifecycler,omitempty"`
	OverrideRingKey  string                `yaml:"override_ring_key"`

	RemoteWrite RemoteWriteConfig `yaml:"remote_write,omitempty"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// apply generic defaults and then overlay tempo default
	flagext.DefaultValues(&cfg.LifecyclerConfig)
	cfg.LifecyclerConfig.RingConfig.KVStore.Store = "memberlist"
	cfg.LifecyclerConfig.RingConfig.ReplicationFactor = 1
	cfg.LifecyclerConfig.RingConfig.HeartbeatTimeout = 5 * time.Minute
	// TODO a generator that is terminated doesn't leave in the correct way yet, the generator stays in status LEAVING until manually forgotten
	cfg.LifecyclerConfig.UnregisterOnShutdown = true
	cfg.LifecyclerConfig.ReadinessCheckRingHealth = false

	hostname, err := os.Hostname()
	if err != nil {
		level.Error(cortex_util.Logger).Log("msg", "failed to get hostname", "err", err)
		os.Exit(1)
	}
	f.StringVar(&cfg.LifecyclerConfig.ID, prefix+".lifecycler.ID", hostname, "ID to register in the ring.")

	// TODO other components have constants in dskit/ring/ring.go, does this value actually matter?
	cfg.OverrideRingKey = "generator"

	cfg.RemoteWrite.RegisterFlagsAndApplyDefaults(prefix, f)
}

type RemoteWriteConfig struct {
	Client  config.RemoteWriteConfig `yaml:"client"`
	Enabled bool                     `yaml:"enabled"`
}

func (c *RemoteWriteConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
}
