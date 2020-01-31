package storage

import (
	"flag"

	"github.com/grafana/frigg/friggdb"
)

// Config is the Frigg storage configuration
type Config struct {
	Trace friggdb.Config `yaml:"trace"`
}

// RegisterFlags adds the flags required to configure this flag set.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	// todo:  flesh out cli
	f.StringVar(&cfg.Trace.Backend, "tracestore.backend", "local", "The trace storage backend to use.")
	f.Float64Var(&cfg.Trace.BloomFilterFalsePositive, "tracestore.bloom-filter-false-positive", .01, "Target false positive rate for the bloom filters.")
}
