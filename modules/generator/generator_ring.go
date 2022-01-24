package generator

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
)

type RingConfig struct {
	KVStore          kv.Config     `yaml:"kvstore"`
	HeartbeatPeriod  time.Duration `yaml:"heartbeat_period"`
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout"`

	InstanceID             string   `yaml:"instance_id"`
	InstanceInterfaceNames []string `yaml:"instance_interface_names"`
	InstanceAddr           string   `yaml:"instance_addr"`

	// Injected internally
	ListenPort int `yaml:"-"`
}

func (cfg *RingConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.KVStore.RegisterFlagsWithPrefix(prefix, "collectors/", f)
	cfg.KVStore.Store = "memberlist"

	cfg.HeartbeatPeriod = 5 * time.Second
	cfg.HeartbeatTimeout = 1 * time.Minute

	cfg.InstanceInterfaceNames = []string{"eth0", "en0"}
}

func (cfg *RingConfig) ToRingConfig() ring.Config {
	rc := ring.Config{}
	flagext.DefaultValues(&rc)

	rc.KVStore = cfg.KVStore
	rc.HeartbeatTimeout = cfg.HeartbeatTimeout
	rc.ReplicationFactor = 1
	rc.SubringCacheDisabled = true

	return rc
}

func (cfg *RingConfig) toLifecyclerConfig() (ring.BasicLifecyclerConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get hostname", "err", err)
		return ring.BasicLifecyclerConfig{}, err
	}

	instanceAddr, err := ring.GetInstanceAddr(cfg.InstanceAddr, cfg.InstanceInterfaceNames, log.Logger)
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get instance address", "err", err)
		return ring.BasicLifecyclerConfig{}, err
	}

	instancePort := cfg.ListenPort

	return ring.BasicLifecyclerConfig{
		ID:              hostname,
		Addr:            fmt.Sprintf("%s:%d", instanceAddr, instancePort),
		HeartbeatPeriod: cfg.HeartbeatPeriod,
		NumTokens:       ringNumTokens,
	}, nil
}
