package storage

import (
	"flag"

	"github.com/cortexproject/cortex/pkg/chunk/storage"
)

// Config is the loki storage configuration
type Config struct {
	Columnar storage.Config `yaml:"columnar"`
	Trace    TraceConfig    `yaml:"trace"`
}

// RegisterFlags adds the flags required to configure this flag set.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	// jpe : how to adjust the flags "down" a level beneath columnar?  just remove flags?  add trace flags?
	cfg.Columnar.RegisterFlags(f)
}
