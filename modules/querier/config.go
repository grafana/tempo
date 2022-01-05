package querier

import (
	"flag"
	"time"

	cortex_worker "github.com/cortexproject/cortex/pkg/querier/worker"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/grpcclient"
)

// Config for a querier.
type Config struct {
	TraceLookupQueryTimeout time.Duration        `yaml:"query_timeout"`
	SearchQueryTimeout      time.Duration        `yaml:"search_query_timeout"`
	SearchExternalEndpoints []string             `yaml:"search_external_endpoints"`
	ExtraQueryDelay         time.Duration        `yaml:"extra_query_delay,omitempty"`
	MaxConcurrentQueries    int                  `yaml:"max_concurrent_queries"`
	Worker                  cortex_worker.Config `yaml:"frontend_worker"`
}

// RegisterFlagsAndApplyDefaults register flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.TraceLookupQueryTimeout = 10 * time.Second
	cfg.SearchQueryTimeout = 30 * time.Second
	cfg.ExtraQueryDelay = 0
	cfg.MaxConcurrentQueries = 5
	cfg.Worker = cortex_worker.Config{
		MatchMaxConcurrency:   true,
		MaxConcurrentRequests: cfg.MaxConcurrentQueries,
		Parallelism:           2,
		GRPCClientConfig: grpcclient.Config{
			MaxRecvMsgSize:  100 << 20,
			MaxSendMsgSize:  16 << 20,
			GRPCCompression: "gzip",
			BackoffConfig: backoff.Config{ // the max possible backoff should be lesser than QueryTimeout, with room for actual query response time
				MinBackoff: 100 * time.Millisecond,
				MaxBackoff: 1 * time.Second,
				MaxRetries: 5,
			},
		},
		DNSLookupPeriod: 10 * time.Second,
	}

	f.StringVar(&cfg.Worker.FrontendAddress, prefix+".frontend-address", "", "Address of query frontend service, in host:port format.")
}
