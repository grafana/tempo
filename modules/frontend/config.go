package frontend

import (
	"flag"
	"time"

	"net/http"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/frontend/transport"
	v1 "github.com/grafana/tempo/modules/frontend/v1"
	"github.com/grafana/tempo/pkg/usagestats"
)

var (
	statVersion = usagestats.NewString("frontend_version")
)

type Config struct {
	Config               v1.Config       `yaml:",inline"`
	MaxRetries           int             `yaml:"max_retries,omitempty"`
	QueryShards          int             `yaml:"query_shards,omitempty"`
	TolerateFailedBlocks int             `yaml:"tolerate_failed_blocks,omitempty"`
	Search               SearchConfig    `yaml:"search"`
	TraceByID            TraceByIDConfig `yaml:"trace_by_id"`
}

type SearchConfig struct {
	Sharder SearchSharderConfig `yaml:",inline"`
}

type TraceByIDConfig struct {
	HedgeRequestsAt   time.Duration `yaml:"hedge_requests_at,omitempty"`
	HedgeRequestsUpTo int           `yaml:"hedge_requests_up_to,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	cfg.MaxRetries = 2
	cfg.QueryShards = 20
	cfg.TolerateFailedBlocks = 0
	cfg.Search = SearchConfig{
		Sharder: SearchSharderConfig{
			QueryBackendAfter:     15 * time.Minute,
			QueryIngestersUntil:   time.Hour,
			DefaultLimit:          20,
			MaxLimit:              0,
			MaxDuration:           61 * time.Minute,
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
		},
	}
	cfg.TraceByID = TraceByIDConfig{
		HedgeRequestsAt:   3 * time.Second,
		HedgeRequestsUpTo: 3,
	}
}

type CortexNoQuerierLimits struct{}

var _ v1.Limits = (*CortexNoQuerierLimits)(nil)

func (CortexNoQuerierLimits) MaxQueriersPerUser(string) int { return 0 }

// InitFrontend initializes V1 frontend
//
// Returned RoundTripper can be wrapped in more round-tripper middlewares, and then eventually registered
// into HTTP server using the Handler from this package. Returned RoundTripper is always non-nil
// (if there are no errors), and it uses the returned frontend (if any).
func InitFrontend(cfg v1.Config, limits v1.Limits, log log.Logger, reg prometheus.Registerer) (http.RoundTripper, *v1.Frontend, error) {
	statVersion.Set("v1")
	// No scheduler = use original frontend.
	fr, err := v1.New(cfg, limits, log, reg)
	if err != nil {
		return nil, nil, err
	}
	return transport.AdaptGrpcRoundTripperToHTTPRoundTripper(fr), fr, nil
}
