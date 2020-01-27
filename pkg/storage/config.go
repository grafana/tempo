package storage

import (
	"flag"

	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/grafana/frigg/pkg/storage/trace_backend/local"
)

// Config is the Frigg storage configuration
type Config struct {
	Columnar storage.Config `yaml:"columnar"`
	Trace    TraceConfig    `yaml:"trace"`
}

type TraceConfig struct {
	Backend string       `yaml:"backend"`
	Local   local.Config `yaml:"local"`

	BloomFilterFalsePositive float64 `yaml:"bloom-filter-false-positive"`
}

// RegisterFlags adds the flags required to configure this flag set.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	// todo : how to adjust the flags "down" a level beneath columnar?  just remove flags?  add trace flags?
	cfg.Columnar.RegisterFlags(f)

	f.StringVar(&cfg.Trace.Backend, "tracestore.backend", "local", "The trace storage backend to use.")
	f.Float64Var(&cfg.Trace.BloomFilterFalsePositive, "tracestore.bloom-filter-false-positive", .01, "Target false positive rate for the bloom filters.")
}
