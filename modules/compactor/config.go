package compactor

import (
	"flag"
	"time"

	cortex_compactor "github.com/cortexproject/cortex/pkg/compactor"
	"github.com/cortexproject/cortex/pkg/ring/kv"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	WaitOnStartup time.Duration                `yaml:"-"`
	ShardingRing  *cortex_compactor.RingConfig `yaml:"sharding_ring,omitempty"`
	Compactor     tempodb.CompactorConfig      `yaml:"compaction"`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.ShardingRing = &cortex_compactor.RingConfig{
		KVStore: kv.Config{
			Store: "memberlist",
		},
	}

	cfg.Compactor = tempodb.CompactorConfig{
		ChunkSizeBytes:          10485760, // 10 MiB
		CompactedBlockRetention: time.Hour,
	}

	f.DurationVar(&cfg.Compactor.BlockRetention, util.PrefixConfig(prefix, "compaction.block-retention"), 14*24*time.Hour, "Duration to keep blocks/traces.")
	f.DurationVar(&cfg.Compactor.BlockRetention, util.PrefixConfig(prefix, "compaction.max-objects-per-block"), 10000000, "Maximum number of traces in a compacted block.")
	f.DurationVar(&cfg.Compactor.BlockRetention, util.PrefixConfig(prefix, "compaction.compaction_window"), 4*time.Hour, "Maximum time window across which to compact blocks.")
}
