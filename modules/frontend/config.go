package frontend

import (
	"flag"
	"time"

	"github.com/cortexproject/cortex/pkg/frontend"
	v1 "github.com/cortexproject/cortex/pkg/frontend/v1"
)

type Config struct {
	Config               frontend.CombinedFrontendConfig `yaml:",inline"`
	MaxRetries           int                             `yaml:"max_retries,omitempty"`
	QueryShards          int                             `yaml:"query_shards,omitempty"`
	TolerateFailedBlocks int                             `yaml:"tolerate_failed_blocks,omitempty"`
	Search               SearchSharderConfig             `yaml:"search"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Config.DownstreamURL = ""
	cfg.Config.Handler.LogQueriesLongerThan = 0
	cfg.Config.FrontendV1.MaxOutstandingPerTenant = 100
	cfg.MaxRetries = 2
	cfg.QueryShards = 20
	cfg.TolerateFailedBlocks = 0
	cfg.Search = SearchSharderConfig{
		QueryIngestersWithinMin: 15 * time.Minute,
		QueryIngestersWithinMax: time.Hour,
		DefaultLimit:            20,
		MaxLimit:                0,
		MaxDuration:             61 * time.Minute,
		ConcurrentRequests:      defaultConcurrentRequests,
		TargetBytesPerRequest:   defaultTargetBytesPerRequest,
	}
}

type CortexNoQuerierLimits struct{}

var _ v1.Limits = (*CortexNoQuerierLimits)(nil)

func (CortexNoQuerierLimits) MaxQueriersPerUser(user string) int { return 0 }
