package frontend

import (
	"flag"
	"time"

	"net/http"

	"github.com/go-kit/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/frontend/transport"
	v1 "github.com/grafana/tempo/modules/frontend/v1"
	v2 "github.com/grafana/tempo/modules/frontend/v2"
	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	Config               CombinedFrontendConfig `yaml:",inline"`
	MaxRetries           int                    `yaml:"max_retries,omitempty"`
	QueryShards          int                    `yaml:"query_shards,omitempty"`
	TolerateFailedBlocks int                    `yaml:"tolerate_failed_blocks,omitempty"`
	Search               SearchSharderConfig    `yaml:"search"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Config.DownstreamURL = ""
	cfg.Config.Handler.LogQueriesLongerThan = 0
	cfg.Config.FrontendV1.MaxOutstandingPerTenant = 100
	cfg.MaxRetries = 2
	cfg.QueryShards = 20
	cfg.TolerateFailedBlocks = 0
	cfg.Search = SearchSharderConfig{
		QueryBackendAfter:     15 * time.Minute,
		QueryIngestersUntil:   time.Hour,
		DefaultLimit:          20,
		MaxLimit:              0,
		MaxDuration:           61 * time.Minute,
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
	}
}

type CortexNoQuerierLimits struct{}

var _ v1.Limits = (*CortexNoQuerierLimits)(nil)

func (CortexNoQuerierLimits) MaxQueriersPerUser(user string) int { return 0 }

// This struct combines several configuration options together to preserve backwards compatibility.
type CombinedFrontendConfig struct {
	Handler    transport.HandlerConfig `yaml:",inline"`
	FrontendV1 v1.Config               `yaml:",inline"`
	FrontendV2 v2.Config               `yaml:",inline"`

	DownstreamURL string `yaml:"downstream_url"`
}

func (cfg *CombinedFrontendConfig) RegisterFlags(f *flag.FlagSet) {
	cfg.Handler.RegisterFlags(f)
	cfg.FrontendV1.RegisterFlags(f)
	cfg.FrontendV2.RegisterFlags(f)

	f.StringVar(&cfg.DownstreamURL, "frontend.downstream-url", "", "URL of downstream Prometheus.")
}

// InitFrontend initializes frontend (either V1 -- without scheduler, or V2 -- with scheduler) or no frontend at
// all if downstream Prometheus URL is used instead.
//
// Returned RoundTripper can be wrapped in more round-tripper middlewares, and then eventually registered
// into HTTP server using the Handler from this package. Returned RoundTripper is always non-nil
// (if there are no errors), and it uses the returned frontend (if any).
func InitFrontend(cfg CombinedFrontendConfig, limits v1.Limits, grpcListenPort int, log log.Logger, reg prometheus.Registerer) (http.RoundTripper, *v1.Frontend, *v2.Frontend, error) {
	switch {
	case cfg.DownstreamURL != "":
		// If the user has specified a downstream Prometheus, then we should use that.
		rt, err := NewDownstreamRoundTripper(cfg.DownstreamURL, http.DefaultTransport)
		return rt, nil, nil, err

	case cfg.FrontendV2.SchedulerAddress != "":
		// If query-scheduler address is configured, use Frontend.
		if cfg.FrontendV2.Addr == "" {
			addr, err := util.GetFirstAddressOf(cfg.FrontendV2.InfNames)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "failed to get frontend address")
			}

			cfg.FrontendV2.Addr = addr
		}

		if cfg.FrontendV2.Port == 0 {
			cfg.FrontendV2.Port = grpcListenPort
		}

		fr, err := v2.NewFrontend(cfg.FrontendV2, log, reg)
		return transport.AdaptGrpcRoundTripperToHTTPRoundTripper(fr), nil, fr, err

	default:
		// No scheduler = use original frontend.
		fr, err := v1.New(cfg.FrontendV1, limits, log, reg)
		if err != nil {
			return nil, nil, nil, err
		}
		return transport.AdaptGrpcRoundTripperToHTTPRoundTripper(fr), fr, nil, nil
	}
}
