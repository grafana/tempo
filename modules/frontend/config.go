package frontend

import (
	"flag"

	"github.com/cortexproject/cortex/pkg/querier/frontend"
)

type Config struct {
	frontend.Config `yaml:",inline"`
	QueryShards     int `yaml:"query_shards,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Config.CompressResponses = true
	cfg.Config.DownstreamURL = ""
	cfg.Config.LogQueriesLongerThan = 0
	cfg.Config.MaxOutstandingPerTenant = 100
	cfg.QueryShards = 4
}
