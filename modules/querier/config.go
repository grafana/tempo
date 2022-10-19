package querier

import (
	"flag"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/tempo/modules/querier/worker"
)

// Config for a querier.
type Config struct {
	Search SearchConfig `yaml:"search"`

	TraceLookupQueryTimeout time.Duration `yaml:"query_timeout,omitempty"`
	ExtraQueryDelay         time.Duration `yaml:"extra_query_delay,omitempty"`
	MaxConcurrentQueries    int           `yaml:"max_concurrent_queries,omitempty"`
	Worker                  worker.Config `yaml:"frontend_worker,omitempty"`
	QueryRelevantIngesters  bool          `yaml:"query_relevant_ingesters,omitempty"`
}

type SearchConfig struct {
	QueryTimeout      time.Duration `yaml:"query_timeout,omitempty"`
	PreferSelf        int           `yaml:"prefer_self,omitempty"`
	ExternalEndpoints []string      `yaml:"external_endpoints,omitempty"`
	HedgeRequestsAt   time.Duration `yaml:"external_hedge_requests_at,omitempty"`
	HedgeRequestsUpTo int           `yaml:"external_hedge_requests_up_to,omitempty"`
}

// RegisterFlagsAndApplyDefaults register flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.TraceLookupQueryTimeout = 10 * time.Second
	cfg.QueryRelevantIngesters = false
	cfg.ExtraQueryDelay = 0
	cfg.MaxConcurrentQueries = 5
	cfg.Search.PreferSelf = 2
	cfg.Search.HedgeRequestsAt = 8 * time.Second
	cfg.Search.HedgeRequestsUpTo = 2
	cfg.Search.QueryTimeout = 30 * time.Second
	cfg.Worker = worker.Config{
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
