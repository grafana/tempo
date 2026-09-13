package secretdetection

import (
	"flag"

	"github.com/grafana/tempo/pkg/secrets"
)

type Config struct {
	Enabled        bool                    `yaml:"-" json:"-"`
	SourceStream   string                  `yaml:"-" json:"-"`
	PolicyCompiler *secrets.PolicyCompiler `yaml:"-" json:"-"`
	CompiledPolicy *secrets.CompiledPolicy `yaml:"-" json:"-"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {}
