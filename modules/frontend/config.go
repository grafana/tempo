package frontend

import (
	"flag"

	"github.com/cortexproject/cortex/pkg/querier/frontend"
)

type FrontendConfig struct {
	frontend.Config `yaml:",inline"`
	ShardNum int    `yaml:"shard_num,omitempty"`
}

func (cfg *FrontendConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Config.CompressResponses = false
	cfg.Config.DownstreamURL = ""
	cfg.Config.LogQueriesLongerThan = 0
	cfg.Config.MaxOutstandingPerTenant = 100
	cfg.ShardNum = 4
}

