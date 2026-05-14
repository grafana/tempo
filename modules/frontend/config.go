package frontend

import (
	"flag"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	v1 "github.com/grafana/tempo/modules/frontend/v1"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/pkg/util"
)

var statVersion = usagestats.NewString("frontend_version")

type Config struct {
	Config                    v1.Config              `yaml:",inline"`
	MaxRetries                int                    `yaml:"max_retries,omitempty"`
	Search                    SearchConfig           `yaml:"search"`
	TraceByID                 TraceByIDConfig        `yaml:"trace_by_id"`
	Metrics                   MetricsConfig          `yaml:"metrics"`
	MultiTenantQueriesEnabled bool                   `yaml:"multi_tenant_queries_enabled"`
	ResponseConsumers         int                    `yaml:"response_consumers"`
	Weights                   pipeline.WeightsConfig `yaml:"weights"`
	MCPServer                 MCPServerConfig        `yaml:"mcp_server"`

	// the maximum time limit that tempo will work on an api request. this includes both
	// grpc and http requests and applies to all "api" frontend query endpoints such as
	// traceql, tag search, tag value search, trace by id and all streaming gRPC endpoints.
	// 0 disables
	APITimeout time.Duration `yaml:"api_timeout,omitempty"`

	// A list of regexes for black listing requests, these will apply for every request regardless the endpoint
	URLDenyList []string `yaml:"url_deny_list,omitempty"`

	// The maximum size of a response message returned over gRPC streaming calls.
	// Diffs and final responses will be segmented into packets of this size.
	// This is separate from the max grpc packet server response size for the process overall,
	// because we may need to target smaller responses for downstream clients.
	MaxGRPCStreamingPacketSize int `yaml:"max_grpc_streaming_packet_size,omitempty"`

	// Maximum allowed size of the raw TraceQL Query expression in bytes
	MaxQueryExpressionSizeBytes int `yaml:"max_query_expression_size_bytes,omitempty"`

	// A list of headers allowed through the HTTP pipeline. Everything else will be stripped.
	AllowedHeaders []string `yaml:"allowed_headers,omitempty"`

	// RF1After specifies the time after which RF1 logic is applied.
	RF1After time.Time `yaml:"rf1_after,omitempty" category:"advanced"` // Deprecated: it's ignored

	// QueryEndCutoff prevents querying incomplete recent data.
	QueryEndCutoff time.Duration `yaml:"query_end_cutoff,omitempty"`

	// SkipASTTransformations is a list of AST transformation names to disable globally.
	// Valid names: "or_to_in". Use "all" to disable all transformations.
	SkipASTTransformations []string `yaml:"skip_ast_transformations,omitempty"`
}

type MCPServerConfig struct {
	Enabled bool `yaml:"enabled"`
}

type SearchConfig struct {
	Timeout     time.Duration       `yaml:"timeout,omitempty"`
	Sharder     SearchSharderConfig `yaml:",inline"`
	SLO         SLOConfig           `yaml:",inline"`
	MetadataSLO SLOConfig           `yaml:"metadata_slo,omitempty"`
}

type TraceByIDConfig struct {
	QueryShards      int       `yaml:"query_shards,omitempty"`
	ConcurrentShards int       `yaml:"concurrent_shards,omitempty"`
	BlocksPerShard   uint      `yaml:"blocks_per_shard,omitempty"` // BlocksPerShard is used to dynamically create shards based on the number of blocks instead of the fixed amount in QueryShards. Set to 0 to disable and fall back to QueryShards.
	SLO              SLOConfig `yaml:",inline"`
	ExternalEnabled  bool      `yaml:"external_enabled,omitempty"`
}

type MetricsConfig struct {
	Sharder      QueryRangeSharderConfig `yaml:",inline"`
	SLO          SLOConfig               `yaml:",inline"`
	MaxIntervals uint64                  `yaml:"max_intervals,omitempty"`
}

type SLOConfig struct {
	DurationSLO        time.Duration `yaml:"duration_slo,omitempty"`
	ThroughputBytesSLO float64       `yaml:"throughput_bytes_slo,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	slo := SLOConfig{
		DurationSLO:        0,
		ThroughputBytesSLO: 0,
	}

	cfg.Config.MaxOutstandingPerTenant = 2000
	cfg.Config.MaxBatchSize = 7
	cfg.MaxRetries = 2
	cfg.ResponseConsumers = 10
	cfg.MaxGRPCStreamingPacketSize = 2 * 1024 * 1024 // 2MB
	cfg.Search = SearchConfig{
		Sharder: SearchSharderConfig{
			QueryBackendAfter:      15 * time.Minute,
			DefaultLimit:           20,
			MaxLimit:               256 * 1024,
			MaxDuration:            168 * time.Hour, // 1 week
			ConcurrentRequests:     defaultConcurrentRequests,
			TargetBytesPerRequest:  defaultTargetBytesPerRequest,
			MostRecentShards:       defaultMostRecentShards,
			IngesterShards:         3,
			DefaultSpansPerSpanSet: 3,
			MaxSpansPerSpanSet:     100,
		},
		SLO: slo,
	}
	cfg.TraceByID = TraceByIDConfig{
		QueryShards:    50,
		BlocksPerShard: 30, // This is a good default for most workloads, found by surveying production deployments.
		SLO:            slo,
	}
	cfg.Metrics = MetricsConfig{
		Sharder: QueryRangeSharderConfig{
			MaxDuration:           24 * time.Hour,
			QueryBackendAfter:     15 * time.Minute,
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
			Interval:              5 * time.Minute,
			MaxExemplars:          100,
			MaxResponseSeries:     0,
			StreamingShards:       defaultStreamingShards,
		},
		SLO: slo,
	}
	cfg.Weights = pipeline.WeightsConfig{
		RequestWithWeights:   true,
		RetryWithWeights:     true,
		MaxRegexConditions:   1,
		MaxTraceQLConditions: 4,
	}

	// set default max query size to 128 KiB, queries larger than this will be rejected
	cfg.MaxQueryExpressionSizeBytes = 128 * 1024
	// enable multi tenant queries by default
	cfg.MultiTenantQueriesEnabled = true
	cfg.Metrics.MaxIntervals = 10_000

	// enabling an mcp server opens the door to send tracing data to an LLM. it should require
	// explicit enabling. registers a flag in addition to YAML configuration.
	f.BoolVar(&cfg.MCPServer.Enabled, util.PrefixConfig(prefix, "mcp-server.enabled"), false, "Set to true to enable the MCP server")
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
