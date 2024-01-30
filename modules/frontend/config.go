package frontend

import (
	"flag"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/frontend/transport"
	v1 "github.com/grafana/tempo/modules/frontend/v1"
	"github.com/grafana/tempo/pkg/usagestats"
)

var statVersion = usagestats.NewString("frontend_version")

type Config struct {
	Config                    v1.Config       `yaml:",inline"`
	MaxRetries                int             `yaml:"max_retries,omitempty"`
	Search                    SearchConfig    `yaml:"search"`
	TraceByID                 TraceByIDConfig `yaml:"trace_by_id"`
	Metrics                   MetricsConfig   `yaml:"metrics"`
	MultiTenantQueriesEnabled bool            `yaml:"multi_tenant_queries_enabled"`
}

type SearchConfig struct {
	Sharder SearchSharderConfig `yaml:",inline"`
	SLO     SLOConfig           `yaml:",inline"`
}

type TraceByIDConfig struct {
	QueryShards      int           `yaml:"query_shards,omitempty"`
	ConcurrentShards int           `yaml:"concurrent_shards,omitempty"`
	Hedging          HedgingConfig `yaml:",inline"`
	SLO              SLOConfig     `yaml:",inline"`
}

type MetricsConfig struct {
	Sharder QueryRangeSharderConfig `yaml:",inline"`
}

type HedgingConfig struct {
	HedgeRequestsAt   time.Duration `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo int           `yaml:"hedge_requests_up_to"`
}

type SLOConfig struct {
	DurationSLO        time.Duration `yaml:"duration_slo,omitempty"`
	ThroughputBytesSLO float64       `yaml:"throughput_bytes_slo,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	slo := SLOConfig{
		DurationSLO:        0,
		ThroughputBytesSLO: 0,
	}

	cfg.Config.MaxOutstandingPerTenant = 2000
	cfg.Config.MaxBatchSize = 5
	cfg.MaxRetries = 2
	cfg.Search = SearchConfig{
		Sharder: SearchSharderConfig{
			QueryBackendAfter:     15 * time.Minute,
			QueryIngestersUntil:   30 * time.Minute,
			DefaultLimit:          20,
			MaxLimit:              0,
			MaxDuration:           168 * time.Hour, // 1 week
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
		},
		SLO: slo,
	}
	cfg.TraceByID = TraceByIDConfig{
		QueryShards: 50,
		SLO:         slo,
		Hedging: HedgingConfig{
			HedgeRequestsAt:   2 * time.Second,
			HedgeRequestsUpTo: 2,
		},
	}
	cfg.Metrics = MetricsConfig{
		Sharder: QueryRangeSharderConfig{
			QueryBackendAfter:     time.Hour,
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: 100 * 1024 * 1024,
			Interval:              5 * time.Minute,
		},
	}

	// enable multi tenant queries by default
	cfg.MultiTenantQueriesEnabled = true
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
