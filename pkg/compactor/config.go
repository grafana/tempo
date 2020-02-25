package compactor

import (
	"flag"

	"github.com/grafana/frigg/friggdb"
)

type Config struct {
	Compactor *friggdb.CompactorConfig `yaml:"compactor"`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {

}
