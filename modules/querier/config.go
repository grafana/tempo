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

	TraceLookupQueryTimeout time.Duration `yaml:"query_timeout"`
	ExtraQueryDelay         time.Duration `yaml:"extra_query_delay,omitempty"`
	MaxConcurrentQueries    int           `yaml:"max_concurrent_queries"`
	Worker                  worker.Config `yaml:"frontend_worker"`
}

type SearchConfig struct {
	QueryTimeout      time.Duration `yaml:"query_timeout"`
	PreferSelf        int           `yaml:"prefer_self"`
	ExternalEndpoints []string      `yaml:"external_endpoints"`
	HedgeRequestsAt   time.Duration `yaml:"external_hedge_requests_at"`
	HedgeRequestsUpTo int           `yaml:"external_hedge_requests_up_to"`
}

// RegisterFlagsAndApplyDefaults register flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.TraceLookupQueryTimeout = 10 * time.Second
	cfg.ExtraQueryDelay = 0
	cfg.MaxConcurrentQueries = 5
	cfg.Search.PreferSelf = 2
	cfg.Search.HedgeRequestsAt = 4 * time.Second
	cfg.Search.HedgeRequestsUpTo = 3
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
