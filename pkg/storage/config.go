package storage

import (
	"flag"

	"github.com/cortexproject/cortex/pkg/chunk/storage"
)

// Config is the loki storage configuration
type Config struct {
	storage.Config `yaml:",inline"`
}

// RegisterFlags adds the flags required to configure this flag set.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.Config.RegisterFlags(f)
}
