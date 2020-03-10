package storage

import (
	"flag"

	"github.com/grafana/tempo/tempodb"
)

// Config is the Frigg storage configuration
type Config struct {
	Trace tempodb.Config `yaml:"trace"`
}

// RegisterFlags adds the flags required to configure this flag set.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	// todo:  figure out if i want cli
}
