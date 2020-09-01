package compactor

import (
	"flag"
	"time"

	cortex_compactor "github.com/cortexproject/cortex/pkg/compactor"
	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	ShardingEnabled bool                        `yaml:"sharding_enabled"`
	WaitOnStartup   time.Duration               `yaml:"-"`
	ShardingRing    cortex_compactor.RingConfig `yaml:"sharding_ring,omitempty"`

	Compactor *tempodb.CompactorConfig `yaml:"compaction"`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.ShardingRing.RegisterFlags(f)
}
