package compactor

import (
	"flag"
	"net"
	"strconv"
	"time"

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

	// Shceduler config
	UseScheduler         bool          `yaml:"use_scheduler"`
	PollingInterval      time.Duration `yaml:"polling_interval"`
	BackendSchedulerAddr string        `yaml:"backend_scheduler_addr"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Compactor = tempodb.CompactorConfig{
		ChunkSizeBytes:          tempodb.DefaultChunkSizeBytes, // 5 MiB
		FlushSizeBytes:          tempodb.DefaultFlushSizeBytes,
		CompactedBlockRetention: time.Hour,
		RetentionConcurrency:    tempodb.DefaultRetentionConcurrency,
		IteratorBufferSize:      tempodb.DefaultIteratorBufferSize,
		MaxTimePerTenant:        tempodb.DefaultMaxTimePerTenant,
		CompactionCycle:         tempodb.DefaultCompactionCycle,
	}

	flagext.DefaultValues(&cfg.ShardingRing)
	cfg.ShardingRing.KVStore.Store = "" // by default compactor is not sharded

	f.DurationVar(&cfg.Compactor.BlockRetention, util.PrefixConfig(prefix, "compaction.block-retention"), 14*24*time.Hour, "Duration to keep blocks/traces.")
	f.IntVar(&cfg.Compactor.MaxCompactionObjects, util.PrefixConfig(prefix, "compaction.max-objects-per-block"), 6000000, "Maximum number of traces in a compacted block.")
	f.Uint64Var(&cfg.Compactor.MaxBlockBytes, util.PrefixConfig(prefix, "compaction.max-block-bytes"), 100*1024*1024*1024 /* 100GB */, "Maximum size of a compacted block.")
	f.DurationVar(&cfg.Compactor.MaxCompactionRange, util.PrefixConfig(prefix, "compaction.compaction-window"), time.Hour, "Maximum time window across which to compact blocks.")
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
