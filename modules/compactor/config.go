package compactor

import (
	"flag"
	"net"
	"strconv"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	Disabled        bool                    `yaml:"disabled,omitempty"`
	ShardingRing    RingConfig              `yaml:"ring,omitempty"`
	Compactor       tempodb.CompactorConfig `yaml:"compaction"`
	OverrideRingKey string                  `yaml:"override_ring_key"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)

	flagext.DefaultValues(&cfg.ShardingRing)
	cfg.ShardingRing.KVStore.Store = "" // by default compactor is not sharded

	f.BoolVar(&cfg.Disabled, util.PrefixConfig(prefix, "disabled"), false, "Disable compaction.")
	cfg.OverrideRingKey = compactorRingKey
}

func toBasicLifecyclerConfig(cfg RingConfig, logger log.Logger) (ring.BasicLifecyclerConfig, error) {
	instanceAddr, err := ring.GetInstanceAddr(cfg.InstanceAddr, cfg.InstanceInterfaceNames, logger, cfg.EnableInet6)
	if err != nil {
		return ring.BasicLifecyclerConfig{}, err
	}

	instancePort := ring.GetInstancePort(cfg.InstancePort, cfg.ListenPort)

	instanceAddrPort := net.JoinHostPort(instanceAddr, strconv.Itoa(instancePort))

	return ring.BasicLifecyclerConfig{
		ID:              cfg.InstanceID,
		Addr:            instanceAddrPort,
		HeartbeatPeriod: cfg.HeartbeatPeriod,
		NumTokens:       ringNumTokens,
	}, nil
}
