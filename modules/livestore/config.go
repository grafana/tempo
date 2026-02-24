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

	WAL wal.Config `yaml:"wal"  doc:"Configuration for the write ahead log."`

	QueryBlockConcurrency    uint          `yaml:"query_block_concurrency,omitempty"`
	CompleteBlockTimeout     time.Duration `yaml:"complete_block_timeout"`
	CompleteBlockConcurrency int           `yaml:"complete_block_concurrency,omitempty"`

	// ShutdownMarkerDir is the path to the shutdown marker directory
	ShutdownMarkerDir string `yaml:"shutdown_marker_dir"`

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

	// GlobalBlockConfig is the main storage trace block config (storage.trace.block). Used as fallback
	// when block_config.version and wal.version are not set. This config is injected by the application when creating the LiveStore.
	GlobalBlockConfig *common.BlockConfig `yaml:"-"`

	// ReadinessTargetLag is the target consumer lag threshold before the live-store
	// is considered ready to serve queries. The live-store will wait until lag drops
	// below this value. Set to 0 to disable readiness waiting (default, backward compatible).
	ReadinessTargetLag time.Duration `yaml:"readiness_target_lag"`

	// ReadinessMaxWait is the maximum time to wait for catching up a˛t startup.
	// If this timeout is exceeded, the live-store becomes ready anyway.
	// Only used if ReadinessTargetLag > 0. Default: 30m.
	ReadinessMaxWait time.Duration `yaml:"readiness_max_wait"`

	// FailOnHighLag makes the live-store fail on search and metrics requests if lag is high
	// and live-store cannot guarantee completeness of results.
	FailOnHighLag bool `yaml:"fail_on_high_lag"`

	// RemoveOwnerOnShutdown controls whether the partition owner is removed from
	// the partition ring during normal shutdown. When true (default), the owner
	// entry is cleaned up so stale entries don't persist. Set to false to preserve
	// the owner registration across restarts.
	RemoveOwnerOnShutdown bool `yaml:"remove_owner_on_shutdown"`

	// testing config
	holdAllBackgroundProcesses bool `yaml:"-"` // if this is set to true, the live store will never release its background processes

	initialBackoff time.Duration `yaml:"-"` // default initial backoff for complete operations
	maxBackoff     time.Duration `yaml:"-"` // default max backoff for complete operations
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
	cfg.CompleteBlockConcurrency = 2
	cfg.Metrics.TimeOverlapCutoff = 0.2

	// Set defaults for timing configuration (based on ingester defaults)
	cfg.InstanceFlushPeriod = 10 * time.Second
	cfg.InstanceCleanupPeriod = 5 * time.Minute
	cfg.MaxTraceLive = 30 * time.Second
	cfg.MaxTraceIdle = 5 * time.Second
	cfg.MaxLiveTracesBytes = 250_000_000 // 250MB
	cfg.MaxBlockDuration = 30 * time.Minute
	cfg.MaxBlockBytes = 100 * 1024 * 1024

	cfg.CommitInterval = 5 * time.Second

	// Readiness config - default to disabled (backward compatible)
	cfg.ReadinessTargetLag = 0
	cfg.ReadinessMaxWait = 30 * time.Minute

	cfg.RemoveOwnerOnShutdown = true

	cfg.initialBackoff = defaultInitialBackoff
	cfg.maxBackoff = defaultMaxBackoff

	// Initialize block config with defaults
	cfg.BlockConfig.RegisterFlagsAndApplyDefaults(prefix+".block", f)

	// Register flags for existing fields
	f.DurationVar(&cfg.CompleteBlockTimeout, prefix+".complete-block-timeout", cfg.CompleteBlockTimeout, "Duration to keep blocks in the live store after they have been flushed.")
	f.UintVar(&cfg.QueryBlockConcurrency, prefix+".concurrent-blocks", cfg.QueryBlockConcurrency, "Number of concurrent blocks to query for metrics.")
	f.Float64Var(&cfg.Metrics.TimeOverlapCutoff, prefix+".metrics.time-overlap-cutoff", cfg.Metrics.TimeOverlapCutoff, "Time overlap cutoff ratio for metrics queries (0.0-1.0).")
	f.DurationVar(&cfg.ReadinessTargetLag, prefix+".readiness-target-lag", cfg.ReadinessTargetLag, "Target lag threshold before live-store is ready. 0 disables waiting (backward compatible).")
	f.DurationVar(&cfg.ReadinessMaxWait, prefix+".readiness-max-wait", cfg.ReadinessMaxWait, "Maximum time to wait for catching up at startup. Only used if readiness-target-lag > 0.")
	f.BoolVar(&cfg.RemoveOwnerOnShutdown, prefix+".remove-owner-on-shutdown", cfg.RemoveOwnerOnShutdown, "Remove partition owner from the ring on shutdown.")

	cfg.WAL.RegisterFlags(f) // WAL config has no flags, only defaults
	f.StringVar(&cfg.WAL.Filepath, prefix+".wal.path", "/var/tempo/live-store/traces", "Path at which store WAL blocks.")
	f.StringVar(&cfg.ShutdownMarkerDir, prefix+".shutdown_marker_dir", "/var/tempo/live-store/shutdown-marker", "Path to the shutdown marker directory.")
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

	if _, _, err := coalesceBlockVersions(cfg); err != nil {
		return err
	}

	if err := common.ValidateConfig(&cfg.BlockConfig); err != nil {
		return fmt.Errorf("block_config validation failed: %w", err)
	}

	if err := cfg.WAL.Validate(); err != nil {
		return err
	}

	return nil
}

// coalesceBlockVersions resolves complete block and WAL encodings from configs
// using the shared encoding.CoalesceVersion helper.
// Block priority: default < storage.trace.block < live_store.block_config.
// WAL priority:   default < storage.trace.block < live_store.block_config < live_store.wal.version.
// Returns an error if any resolved version isn't writable.
func coalesceBlockVersions(cfg *Config) (completeBlockEncoding, walEncoding encoding.VersionedEncoding, err error) {
	globalVer := ""
	if cfg.GlobalBlockConfig != nil {
		globalVer = cfg.GlobalBlockConfig.Version
	}

	completeBlockEncoding, err = encoding.CoalesceVersion(globalVer, cfg.BlockConfig.Version)
	if err != nil {
		return nil, nil, fmt.Errorf("complete block version: %w", err)
	}

	walEncoding, err = encoding.CoalesceVersion(globalVer, cfg.BlockConfig.Version, cfg.WAL.Version)
	if err != nil {
		return nil, nil, fmt.Errorf("wal version: %w", err)
	}

	cfg.BlockConfig.Version = completeBlockEncoding.Version()
	cfg.WAL.Version = walEncoding.Version()

	return completeBlockEncoding, walEncoding, nil
}
