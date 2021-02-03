package compactor

import (
	"flag"
	"time"

	cortex_compactor "github.com/cortexproject/cortex/pkg/compactor"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	ShardingRing    cortex_compactor.RingConfig `yaml:"ring,omitempty"`
	Compactor       tempodb.CompactorConfig     `yaml:"compaction"`
	OverrideRingKey string                      `yaml:"override_ring_key"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Compactor = tempodb.CompactorConfig{
		ChunkSizeBytes:          10 * 1024 * 1024, // 10 MiB
		FlushSizeBytes:          30 * 1024 * 1024, // 30 MiB
		CompactedBlockRetention: time.Hour,
		RetentionConcurrency:    tempodb.DefaultRetentionConcurrency,
	}

	flagext.DefaultValues(&cfg.ShardingRing)
	cfg.ShardingRing.KVStore.Store = "" // by default compactor is not sharded

	f.DurationVar(&cfg.Compactor.BlockRetention, util.PrefixConfig(prefix, "compaction.block-retention"), 14*24*time.Hour, "Duration to keep blocks/traces.")
	f.IntVar(&cfg.Compactor.MaxCompactionObjects, util.PrefixConfig(prefix, "compaction.max-objects-per-block"), 6000000, "Maximum number of traces in a compacted block.")
	f.DurationVar(&cfg.Compactor.MaxCompactionRange, util.PrefixConfig(prefix, "compaction.compaction-window"), 4*time.Hour, "Maximum time window across which to compact blocks.")
	cfg.OverrideRingKey = ring.CompactorRingKey
}
