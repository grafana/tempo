package compactor

import (
	"flag"

	"github.com/grafana/tempo/tempodb"
)

type Config struct {
	Compactor *tempodb.CompactorConfig `yaml:"compaction"`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {

}
