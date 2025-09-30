package livestore

import (
	"flag"
	"fmt"
	"time"

	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ring"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

const defaultCompleteBlockTimeout = time.Hour

type Config struct {
	Ring          ring.Config                  `yaml:"ring,omitempty"`
	PartitionRing ingester.PartitionRingConfig `yaml:"partition_ring" category:"experimental"`
	Metrics       MetricsConfig                `yaml:"metrics"`

	// CommitInterval configures how often the partition reader commits to kafka
	// 0s means synchronous commits
	CommitInterval time.Duration `yaml:"commit_interval"`

	// This config is dynamically injected because defined outside the ingester config.
	IngestConfig ingest.Config `yaml:"-"`

	// WAL is non-configurable and only uses defaults
	WAL wal.Config `yaml:"-"`

	QueryBlockConcurrency    uint          `yaml:"query_block_concurrency,omitempty"`
	CompleteBlockTimeout     time.Duration `yaml:"complete_block_timeout"`
	CompleteBlockConcurrency int           `yaml:"complete_block_concurrency,omitempty"`

	// Timing configuration
	InstanceFlushPeriod   time.Duration `yaml:"flush_check_period"`
	InstanceCleanupPeriod time.Duration `yaml:"flush_op_timeout"`
	MaxTraceLive          time.Duration `yaml:"max_trace_live"`
	MaxTraceIdle          time.Duration `yaml:"max_trace_idle"`
	MaxLiveTracesBytes    uint64        `yaml:"max_live_traces_bytes"`
	MaxBlockDuration      time.Duration `yaml:"max_block_duration"`
	MaxBlockBytes         uint64        `yaml:"max_block_bytes"`

	// Block configuration
	BlockConfig common.BlockConfig `yaml:"block_config"`

	// testing config
	holdAllBackgroundProcesses bool `yaml:"-"` // if this is set to true, the live store will never release its background processes
}

type MetricsConfig struct {
	// TimeOverlapCutoff is a tuning factor that controls whether the trace-level
	// timestamp columns are used in a metrics query.  Loading these columns has a cost,
	// so in some cases it faster to skip these columns entirely, reducing I/O but
	// increasing the number of spans evalulated and thrown away. The value is a ratio
	// between 0.0 and 1.0.  If a block overlaps the time window by less than this value,
	// then we skip the columns. A value of 1.0 will always load the columns, and 0.0 never.
	TimeOverlapCutoff float64 `yaml:"time_overlap_cutoff,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.PartitionRing.RegisterFlags(prefix, f)

	// Set defaults for new fields
	cfg.CompleteBlockTimeout = defaultCompleteBlockTimeout
	cfg.QueryBlockConcurrency = 10
	cfg.CompleteBlockConcurrency = 4
	cfg.Metrics.TimeOverlapCutoff = 0.2

	// Set defaults for timing configuration (based on ingester defaults)
	cfg.InstanceFlushPeriod = 10 * time.Second
	cfg.InstanceCleanupPeriod = 5 * time.Minute
	cfg.MaxTraceLive = 30 * time.Second
	cfg.MaxTraceIdle = 5 * time.Second
	cfg.MaxLiveTracesBytes = 250_000_000 // 250MB
	cfg.MaxBlockDuration = 30 * time.Minute
	cfg.MaxBlockBytes = 500 * 1024 * 1024

	cfg.CommitInterval = 5 * time.Second

	// Initialize block config with defaults
	cfg.BlockConfig.RegisterFlagsAndApplyDefaults(prefix+".block", f)

	// Register flags for existing fields
	f.DurationVar(&cfg.CompleteBlockTimeout, prefix+".complete-block-timeout", cfg.CompleteBlockTimeout, "Duration to keep blocks in the live store after they have been flushed.")
	f.UintVar(&cfg.QueryBlockConcurrency, prefix+".concurrent-blocks", cfg.QueryBlockConcurrency, "Number of concurrent blocks to query for metrics.")
	f.Float64Var(&cfg.Metrics.TimeOverlapCutoff, prefix+".metrics.time-overlap-cutoff", cfg.Metrics.TimeOverlapCutoff, "Time overlap cutoff ratio for metrics queries (0.0-1.0).")

	cfg.WAL.RegisterFlags(f) // WAL config has no flags, only defaults
	cfg.WAL.Version = encoding.DefaultEncoding().Version()
	f.StringVar(&cfg.WAL.Filepath, prefix+".wal.path", "/var/tempo/live-store/traces", "Path at which store WAL blocks.")
}

func (cfg *Config) Validate() error {
	if cfg.CompleteBlockTimeout <= 0 {
		return fmt.Errorf("complete_block_timeout must be greater than 0, got %s", cfg.CompleteBlockTimeout)
	}

	if cfg.QueryBlockConcurrency == 0 {
		return fmt.Errorf("query_blocks must be greater than 0, got %d", cfg.QueryBlockConcurrency)
	}

	if cfg.CompleteBlockConcurrency <= 0 {
		return fmt.Errorf("complete_block_concurrency must be greater than 0, got %d", cfg.CompleteBlockConcurrency)
	}

	if cfg.InstanceFlushPeriod <= 0 {
		return fmt.Errorf("flush_check_period must be greater than 0, got %s", cfg.InstanceFlushPeriod)
	}

	if cfg.InstanceCleanupPeriod <= 0 {
		return fmt.Errorf("flush_op_timeout must be greater than 0, got %s", cfg.InstanceCleanupPeriod)
	}

	if cfg.MaxTraceLive <= 0 {
		return fmt.Errorf("max_trace_live must be greater than 0, got %s", cfg.MaxTraceLive)
	}

	if cfg.MaxTraceIdle <= 0 {
		return fmt.Errorf("max_trace_idle must be greater than 0, got %s", cfg.MaxTraceIdle)
	}

	if cfg.MaxBlockDuration <= 0 {
		return fmt.Errorf("max_block_duration must be greater than 0, got %s", cfg.MaxBlockDuration)
	}

	if cfg.MaxBlockBytes == 0 {
		return fmt.Errorf("max_block_bytes must be greater than 0, got %d", cfg.MaxBlockBytes)
	}

	if cfg.MaxTraceIdle > cfg.MaxTraceLive {
		return fmt.Errorf("max_trace_idle (%s) cannot be greater than max_trace_live (%s)", cfg.MaxTraceIdle, cfg.MaxTraceLive)
	}

	if err := common.ValidateConfig(&cfg.BlockConfig); err != nil {
		return fmt.Errorf("block_config validation failed: %w", err)
	}

	return cfg.WAL.Validate()
}
