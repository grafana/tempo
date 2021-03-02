package frontend

import (
	"flag"

	"github.com/cortexproject/cortex/pkg/frontend"
	v1 "github.com/cortexproject/cortex/pkg/frontend/v1"
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
	cfg.QueryShards = 2
}

type CortexNoQuerierLimits struct{}

var _ v1.Limits = (*CortexNoQuerierLimits)(nil)

func (CortexNoQuerierLimits) MaxQueriersPerUser(user string) int { return 0 }
