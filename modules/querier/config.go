package querier

import (
	"flag"
	"time"

	cortex_worker "github.com/cortexproject/cortex/pkg/querier/worker"
	"github.com/cortexproject/cortex/pkg/util/grpcclient"
)

// Config for a querier.
type Config struct {
	QueryTimeout         time.Duration        `yaml:"query_timeout"`
	ExtraQueryDelay      time.Duration        `yaml:"extra_query_delay,omitempty"`
	MaxConcurrentQueries int                  `yaml:"max_concurrent_queries"`
	Worker               cortex_worker.Config `yaml:"frontend_worker"`
}

// RegisterFlagsAndApplyDefaults register flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.QueryTimeout = 10 * time.Second
	cfg.ExtraQueryDelay = 0
	cfg.MaxConcurrentQueries = 5
	cfg.Worker = cortex_worker.Config{
		MatchMaxConcurrency:   true,
		MaxConcurrentRequests: cfg.MaxConcurrentQueries,
		Parallelism:           2,
		GRPCClientConfig: grpcclient.ConfigWithTLS{
			GRPC: grpcclient.Config{
				MaxRecvMsgSize:     100 << 20,
				MaxSendMsgSize:     16 << 20,
				UseGzipCompression: false,
				GRPCCompression:    "gzip",
			},
		},
	}

	f.StringVar(&cfg.Worker.FrontendAddress, prefix+".frontend-address", "", "Address of query frontend service, in host:port format.")
}
