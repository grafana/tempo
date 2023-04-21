package localblocks

import (
	"flag"
	"time"

	"github.com/grafana/tempo/tempodb/wal"
)

const (
	Name = "local-blocks"
)

type Config struct {
	WAL                  *wal.Config   `yaml:"wal"`
	FlushCheckPeriod     time.Duration `yaml:"flush_check_period"`
	MaxTraceIdle         time.Duration `yaml:"trace_idle_period"`
	MaxBlockDuration     time.Duration `yaml:"max_block_duration"`
	MaxBlockBytes        uint64        `yaml:"max_block_bytes"`
	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.FlushCheckPeriod = 5 * time.Second
	cfg.MaxTraceIdle = 5 * time.Second
	cfg.MaxBlockDuration = 15 * time.Minute
	cfg.MaxBlockBytes = 500_000_000
	cfg.CompleteBlockTimeout = time.Hour
}
