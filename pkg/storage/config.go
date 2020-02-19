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
	// todo:  figure out if i want cli
}
