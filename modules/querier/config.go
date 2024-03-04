package querier

import (
	"flag"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/tempo/modules/querier/external"
	"github.com/grafana/tempo/modules/querier/worker"
)

// Config for a querier.
type Config struct {
	Search    SearchConfig    `yaml:"search"`
	TraceByID TraceByIDConfig `yaml:"trace_by_id"`
	Metrics   MetricsConfig   `yaml:"metrics"`

	ExtraQueryDelay        time.Duration `yaml:"extra_query_delay,omitempty"`
	MaxConcurrentQueries   int           `yaml:"max_concurrent_queries"`
	Worker                 worker.Config `yaml:"frontend_worker"`
	QueryRelevantIngesters bool          `yaml:"query_relevant_ingesters"`
	SecondaryIngesterRing  string        `yaml:"secondary_ingester_ring,omitempty"`

	AutocompleteFilteringEnabled bool `yaml:"-"`
}

type SearchConfig struct {
	QueryTimeout      time.Duration `yaml:"query_timeout"`
	PreferSelf        int           `yaml:"prefer_self"`
	HedgeRequestsAt   time.Duration `yaml:"external_hedge_requests_at"`
	HedgeRequestsUpTo int           `yaml:"external_hedge_requests_up_to"`

	// backends
	ExternalBackend string                   `yaml:"external_backend"`
	CloudRun        *external.CloudRunConfig `yaml:"google_cloud_run"`

	// Ideally this config should be under the external.HTTPConfig struct, but
	// it will require a breaking change.
	ExternalEndpoints []string `yaml:"external_endpoints"`
}

type TraceByIDConfig struct {
	QueryTimeout time.Duration `yaml:"query_timeout"`
}

type MetricsConfig struct {
	BlockConcurrency int `yaml:"block_concurrency,omitempty"`
}

// RegisterFlagsAndApplyDefaults register flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.TraceByID.QueryTimeout = 10 * time.Second
	cfg.QueryRelevantIngesters = false
	cfg.ExtraQueryDelay = 0
	cfg.MaxConcurrentQueries = 20
	cfg.Search.PreferSelf = 10
	cfg.Search.HedgeRequestsAt = 8 * time.Second
	cfg.Search.HedgeRequestsUpTo = 2
	cfg.Search.QueryTimeout = 30 * time.Second
	cfg.Metrics.BlockConcurrency = 2
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
