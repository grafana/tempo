package local

import (
	"flag"

	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	Path string `yaml:"path"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Path, util.PrefixConfig(prefix, "local.path"), "", "path to store traces at.")
}

func (cfg *Config) PathMatches(other *Config) bool {
	return cfg.Path == other.Path
}
