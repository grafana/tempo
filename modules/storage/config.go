package storage

import (
	"flag"

	"github.com/grafana/tempo/tempodb"
)

// Config is the Tempo storage configuration
type Config struct {
	Trace tempodb.Config `yaml:"trace"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {

}
