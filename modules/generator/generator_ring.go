package generator

import (
	"flag"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"

	"github.com/grafana/tempo/v2/pkg/util/log"
)

type RingConfig struct {
	KVStore          kv.Config     `yaml:"kvstore"`
	HeartbeatPeriod  time.Duration `yaml:"heartbeat_period"`
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout"`

	InstanceID             string   `yaml:"instance_id"`
	InstanceInterfaceNames []string `yaml:"instance_interface_names"`
	InstanceAddr           string   `yaml:"instance_addr"`
	InstancePort           int      `yaml:"instance_port"`
	EnableInet6            bool     `yaml:"enable_inet6"`

	// Injected internally
	ListenPort int `yaml:"-"`
}

func (cfg *RingConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.KVStore.RegisterFlagsWithPrefix(prefix, "collectors/", f)
	cfg.KVStore.Store = "memberlist"

	cfg.HeartbeatPeriod = 5 * time.Second
	cfg.HeartbeatTimeout = 1 * time.Minute

	hostname, err := os.Hostname()
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get hostname", "err", err)
		os.Exit(1)
	}
	cfg.InstanceID = hostname
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
	instanceAddr, err := ring.GetInstanceAddr(cfg.InstanceAddr, cfg.InstanceInterfaceNames, log.Logger, cfg.EnableInet6)
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get instance address", "err", err)
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
