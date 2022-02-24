package remotewrite

import (
	"flag"

	"github.com/prometheus/prometheus/config"
)

type Config struct {
	// Enable remote-write requests. If disabled all generated metrics will be discarded.
	Enabled bool `yaml:"enabled"`

	Client config.RemoteWriteConfig `yaml:"client"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
}
