package livestore

import (
	"flag"
	"time"

	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

type Config struct {
	LifecyclerConfig ring.LifecyclerConfig        `yaml:"lifecycler,omitempty"`
	PartitionRing    ingester.PartitionRingConfig `yaml:"partition_ring" category:"experimental"`

	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout"`
	Metrics              MetricsConfig `yaml:"metrics"`

	// This config is dynamically injected because defined outside the ingester config.
	IngestConfig ingest.Config `yaml:"-"`

	// WAL is non-configurable and only uses defaults
	WAL wal.Config `yaml:"-"`
}

type MetricsConfig struct {
	ConcurrentBlocks  uint    `yaml:"concurrent_blocks,omitempty"`
	TimeOverlapCutoff float64 `yaml:"time_overlap_cutoff,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.LifecyclerConfig.RegisterFlagsWithPrefix(prefix, f, log.Logger)
	cfg.PartitionRing.RegisterFlags(prefix, f)

	// Set defaults for new fields
	cfg.CompleteBlockTimeout = 3 * tempodb.DefaultBlocklistPoll
	cfg.Metrics.ConcurrentBlocks = 10
	cfg.Metrics.TimeOverlapCutoff = 0.2

	// Register flags for new fields
	f.DurationVar(&cfg.CompleteBlockTimeout, prefix+".complete-block-timeout", cfg.CompleteBlockTimeout, "Duration to keep blocks in the live store after they have been flushed.")
	f.UintVar(&cfg.Metrics.ConcurrentBlocks, prefix+".metrics.concurrent-blocks", cfg.Metrics.ConcurrentBlocks, "Number of concurrent blocks to query for metrics.")
	f.Float64Var(&cfg.Metrics.TimeOverlapCutoff, prefix+".metrics.time-overlap-cutoff", cfg.Metrics.TimeOverlapCutoff, "Time overlap cutoff ratio for metrics queries (0.0-1.0).")

	cfg.WAL.RegisterFlags(f) // WAL config has no flags, only defaults
	cfg.WAL.Version = encoding.DefaultEncoding().Version()
	f.StringVar(&cfg.WAL.Filepath, prefix+".wal.path", "/var/tempo/live-store/traces", "Path at which store WAL blocks.")
}

func (cfg *Config) Validate() error {
	err := cfg.LifecyclerConfig.Validate()
	if err != nil {
		return err
	}

	err = cfg.WAL.Validate()
	if err != nil {
		return err
	}

	return nil
}
