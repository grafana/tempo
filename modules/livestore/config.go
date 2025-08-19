package livestore

import (
	"flag"
	"time"

	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ring"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

type Config struct {
	Ring          ring.Config                  `yaml:"ring,omitempty"`
	PartitionRing ingester.PartitionRingConfig `yaml:"partition_ring" category:"experimental"`

	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout"`
	ConcurrentBlocks     uint          `yaml:"concurrent_blocks,omitempty"`
	Metrics              MetricsConfig `yaml:"metrics"`

	// This config is dynamically injected because defined outside the ingester config.
	IngestConfig ingest.Config `yaml:"-"`

	// WAL is non-configurable and only uses defaults
	WAL wal.Config `yaml:"-"`
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
	cfg.CompleteBlockTimeout = 3 * tempodb.DefaultBlocklistPoll
	cfg.ConcurrentBlocks = 10
	cfg.Metrics.TimeOverlapCutoff = 0.2

	// Register flags for new fields
	f.DurationVar(&cfg.CompleteBlockTimeout, prefix+".complete-block-timeout", cfg.CompleteBlockTimeout, "Duration to keep blocks in the live store after they have been flushed.")
	f.UintVar(&cfg.ConcurrentBlocks, prefix+".concurrent-blocks", cfg.ConcurrentBlocks, "Number of concurrent blocks to query for metrics.")
	f.Float64Var(&cfg.Metrics.TimeOverlapCutoff, prefix+".metrics.time-overlap-cutoff", cfg.Metrics.TimeOverlapCutoff, "Time overlap cutoff ratio for metrics queries (0.0-1.0).")

	cfg.WAL.RegisterFlags(f) // WAL config has no flags, only defaults
	cfg.WAL.Version = encoding.DefaultEncoding().Version()
	f.StringVar(&cfg.WAL.Filepath, prefix+".wal.path", "/var/tempo/live-store/traces", "Path at which store WAL blocks.")
}

func (cfg *Config) Validate() error {
	return cfg.WAL.Validate()
}
