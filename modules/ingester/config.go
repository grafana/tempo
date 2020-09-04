package ingester

import (
	"flag"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util/flagext"
)

// Config for an ingester.
type Config struct {
	LifecyclerConfig ring.LifecyclerConfig `yaml:"lifecycler,omitempty"`

	ConcurrentFlushes    int           `yaml:"concurrent_flushes"`
	FlushCheckPeriod     time.Duration `yaml:"flush_check_period"`
	FlushOpTimeout       time.Duration `yaml:"flush_op_timeout"`
	MaxTraceIdle         time.Duration `yaml:"trace_idle_period"`
	MaxTracesPerBlock    int           `yaml:"traces_per_block"`
	MaxBlockDuration     time.Duration `yaml:"max_block_duration"`
	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// apply generic defaults and then overlay tempo default
	flagext.DefaultValues(&cfg.LifecyclerConfig)
	cfg.LifecyclerConfig.RingConfig.KVStore.Store = "memberlist"
	cfg.LifecyclerConfig.RingConfig.ReplicationFactor = 1

	cfg.ConcurrentFlushes = 16
	cfg.FlushCheckPeriod = 30 * time.Second
	cfg.FlushOpTimeout = 10 * time.Second

	f.DurationVar(&cfg.MaxTraceIdle, "ingester.trace-idle-period", 30*time.Second, "")
	f.IntVar(&cfg.MaxTracesPerBlock, "ingester.traces-per-block", 10000, "")
	f.DurationVar(&cfg.MaxBlockDuration, "ingester.max-block-duration", 4*time.Hour, "")
}
