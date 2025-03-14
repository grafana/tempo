package backendworker

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	BackendSchedulerAddr string                  `yaml:"backend_scheduler_addr"`
	Backoff              backoff.Config          `yaml:"backoff"`
	Compactor            tempodb.CompactorConfig `yaml:"compaction"`
	OverrideRingKey      string                  `yaml:"override_ring_key"`
	Poll                 bool                    `yaml:"-"`
	Ring                 RingConfig              `yaml:"ring,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// Backoff
	f.DurationVar(&cfg.Backoff.MinBackoff, prefix+".backoff-min-period", 100*time.Millisecond, "Minimum delay when backing off.")
	f.DurationVar(&cfg.Backoff.MaxBackoff, prefix+".backoff-max-period", time.Minute, "Maximum delay when backing off.")
	f.IntVar(&cfg.Backoff.MaxRetries, prefix+".backoff-retries", 0, "Number of times to backoff and retry before failing.")

	// Compactor
	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)

	// Ring
	flagext.DefaultValues(&cfg.Ring)
	cfg.Ring.KVStore.Store = "" // by default worker is not sharded
	cfg.OverrideRingKey = backendWorkerRingKey
}

func ValidateConfig(cfg *Config) error {
	if cfg.BackendSchedulerAddr == "" {
		return fmt.Errorf("backend scheduler address is required")
	}

	if cfg.Backoff.MinBackoff <= 0 {
		return fmt.Errorf("positive backoff min period required")
	}

	if cfg.Backoff.MaxBackoff <= 0 {
		return fmt.Errorf("positive backoff max period required")
	}

	if cfg.Backoff.MaxRetries < 0 {
		return fmt.Errorf("positive backoff retries required")
	}

	return nil
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
