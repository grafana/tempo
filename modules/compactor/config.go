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
	OverrideRingKey string                      `yaml:override_ring_key`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Compactor = tempodb.CompactorConfig{
		ChunkSizeBytes:          10485760, // 10 MiB
		CompactedBlockRetention: time.Hour,
	}

	flagext.DefaultValues(&cfg.ShardingRing)

	f.DurationVar(&cfg.Compactor.BlockRetention, util.PrefixConfig(prefix, "compaction.block-retention"), 14*24*time.Hour, "Duration to keep blocks/traces.")
	f.IntVar(&cfg.Compactor.MaxCompactionObjects, util.PrefixConfig(prefix, "compaction.max-objects-per-block"), 10000000, "Maximum number of traces in a compacted block.")
	f.DurationVar(&cfg.Compactor.MaxCompactionRange, util.PrefixConfig(prefix, "compaction.compaction-window"), 4*time.Hour, "Maximum time window across which to compact blocks.")
	f.StringVar(&cfg.OverrideRingKey, util.PrefixConfig(prefix, "compaction.override-ring-key"), ring.CompactorRingKey, "Override key to ignore previous ring state.")
}
