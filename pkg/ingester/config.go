package ingester

import (
	"flag"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
)

// Config for an ingester.
type Config struct {
	LifecyclerConfig ring.LifecyclerConfig `yaml:"lifecycler,omitempty"`

	// Config for transferring chunks.
	MaxTransferRetries int `yaml:"max_transfer_retries,omitempty"`

	ConcurrentFlushes    int           `yaml:"concurrent_flushes"`
	FlushCheckPeriod     time.Duration `yaml:"flush_check_period"`
	FlushOpTimeout       time.Duration `yaml:"flush_op_timeout"`
	MaxTraceIdle         time.Duration `yaml:"trace_idle_period"`
	MaxTracesPerBlock    int           `yaml:"traces_per_block"`
	MaxBlockDuration     time.Duration `yaml:"max_block_duration"`
	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout"`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.LifecyclerConfig.RegisterFlags(f)

	f.IntVar(&cfg.MaxTransferRetries, "ingester.max-transfer-retries", 10, "Number of times to try and transfer chunks before falling back to flushing. If set to 0 or negative value, transfers are disabled.")
	f.IntVar(&cfg.ConcurrentFlushes, "ingester.concurrent-flushed", 16, "")
	f.DurationVar(&cfg.FlushCheckPeriod, "ingester.flush-check-period", 30*time.Second, "")
	f.DurationVar(&cfg.FlushOpTimeout, "ingester.flush-op-timeout", 10*time.Second, "")
	f.DurationVar(&cfg.MaxTraceIdle, "ingester.trace-idle-period", 30*time.Second, "")
	f.IntVar(&cfg.MaxTracesPerBlock, "ingester.traces-per-block", 10000, "")
	f.DurationVar(&cfg.MaxBlockDuration, "ingester.max-block-duration", 4*time.Hour, "")
}
