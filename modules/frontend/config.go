package frontend

import (
	"flag"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/frontend/pipeline"
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
	ResponseConsumers         int             `yaml:"response_consumers"`

	// the maximum time limit that tempo will work on an api request. this includes both
	// grpc and http requests and applies to all "api" frontend query endpoints such as
	// traceql, tag search, tag value search, trace by id and all streaming gRPC endpoints.
	// 0 disables
	APITimeout time.Duration `yaml:"api_timeout,omitempty"`

	// A list of regexes for black listing requests, these will apply for every request regardless the endpoint
	URLDenyList []string `yaml:"url_deny_list,omitempty"`
}

type SearchConfig struct {
	Timeout time.Duration       `yaml:"timeout,omitempty"`
	Sharder SearchSharderConfig `yaml:",inline"`
	SLO     SLOConfig           `yaml:",inline"`
}

type TraceByIDConfig struct {
	QueryShards      int       `yaml:"query_shards,omitempty"`
	ConcurrentShards int       `yaml:"concurrent_shards,omitempty"`
	SLO              SLOConfig `yaml:",inline"`
}

type MetricsConfig struct {
	Sharder QueryRangeSharderConfig `yaml:",inline"`
	SLO     SLOConfig               `yaml:",inline"`
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
	cfg.ResponseConsumers = 10
	cfg.Search = SearchConfig{
		Sharder: SearchSharderConfig{
			QueryBackendAfter:     15 * time.Minute,
			QueryIngestersUntil:   30 * time.Minute,
			DefaultLimit:          20,
			MaxLimit:              0,
			MaxDuration:           168 * time.Hour, // 1 week
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
			IngesterShards:        1,
		},
		SLO: slo,
	}
	cfg.TraceByID = TraceByIDConfig{
		QueryShards: 50,
		SLO:         slo,
	}
	cfg.Metrics = MetricsConfig{
		Sharder: QueryRangeSharderConfig{
			MaxDuration:           3 * time.Hour,
			QueryBackendAfter:     30 * time.Minute,
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
			Interval:              5 * time.Minute,
			Exemplars:             false, // TODO: Remove?
			MaxExemplars:          100,
		},
		SLO: slo,
	}

	// enable multi tenant queries by default
	cfg.MultiTenantQueriesEnabled = true
}

type CortexNoQuerierLimits struct{}

// InitFrontend initializes V1 frontend
//
// Returned RoundTripper can be wrapped in more round-tripper middlewares, and then eventually registered
// into HTTP server using the Handler from this package. Returned RoundTripper is always non-nil
// (if there are no errors), and it uses the returned frontend (if any).
func InitFrontend(cfg v1.Config, log log.Logger, reg prometheus.Registerer) (pipeline.RoundTripper, *v1.Frontend, error) {
	statVersion.Set("v1")
	// No scheduler = use original frontend.
	fr, err := v1.New(cfg, log, reg)
	if err != nil {
		return nil, nil, err
	}
	return fr, fr, nil
}
