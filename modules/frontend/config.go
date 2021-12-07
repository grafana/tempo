package frontend

import (
	"flag"
	"time"

	"github.com/cortexproject/cortex/pkg/frontend"
	v1 "github.com/cortexproject/cortex/pkg/frontend/v1"
)

type Config struct {
	Config                      frontend.CombinedFrontendConfig `yaml:",inline"`
	MaxRetries                  int                             `yaml:"max_retries,omitempty"`
	QueryShards                 int                             `yaml:"query_shards,omitempty"`
	TolerateFailedBlocks        int                             `yaml:"tolerate_failed_blocks,omitempty"`
	SearchConcurrentRequests    int                             `yaml:"search_concurrent_jobs,omitempty"`
	SearchTargetBytesPerRequest int                             `yaml:"search_target_bytes_per_job,omitempty"`
	QueryIngestersWithinMin     time.Duration                   `yaml:"query_ingesters_within_min,omitempty"`
	QueryIngestersWithinMax     time.Duration                   `yaml:"query_ingesters_within_max,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Config.DownstreamURL = ""
	cfg.Config.Handler.LogQueriesLongerThan = 0
	cfg.Config.FrontendV1.MaxOutstandingPerTenant = 100
	cfg.MaxRetries = 2
	cfg.QueryShards = 20
	cfg.TolerateFailedBlocks = 0
	cfg.QueryIngestersWithinMin = 15 * time.Minute
	cfg.QueryIngestersWithinMax = time.Hour

	cfg.SearchConcurrentRequests = defaultConcurrentRequests
	cfg.SearchTargetBytesPerRequest = defaultTargetBytesPerRequest
}

type CortexNoQuerierLimits struct{}

var _ v1.Limits = (*CortexNoQuerierLimits)(nil)

func (CortexNoQuerierLimits) MaxQueriersPerUser(user string) int { return 0 }
