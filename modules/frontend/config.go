package frontend

import (
	"flag"

	"github.com/cortexproject/cortex/pkg/frontend"
)

type Config struct {
	Config      frontend.CombinedFrontendConfig `yaml:",inline"`
	QueryShards int                             `yaml:"query_shards,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Config.CompressResponses = true
	cfg.Config.DownstreamURL = ""
	cfg.Config.Handler.LogQueriesLongerThan = 0
	cfg.Config.FrontendV1.MaxOutstandingPerTenant = 100
	cfg.QueryShards = 4
}
