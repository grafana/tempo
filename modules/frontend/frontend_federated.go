package frontend

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/dskit/middleware"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
)

// NewFederated creates a new QueryFrontend configured for federation mode.
func NewFederated(cfg Config, next pipeline.RoundTripper, o overrides.Interface, reader tempodb.Reader, cacheProvider cache.Provider, apiPrefix string, authMiddleware middleware.Interface, dataAccessController DataAccessController, logger log.Logger, registerer prometheus.Registerer) (*QueryFrontend, error) {
	level.Info(logger).Log("msg", "creating federated query frontend", "instances", len(cfg.Federation.Instances))

	if !cfg.Federation.Enabled || len(cfg.Federation.Instances) == 0 {
		return nil, fmt.Errorf("federation must be enabled with at least one instance configured")
	}

	// Build federation config
	federationCfg := pipeline.FederationConfig{
		Enabled:            true,
		ConcurrentRequests: cfg.Federation.ConcurrentRequests,
	}
	maxTimeout := 30 * time.Second
	for _, inst := range cfg.Federation.Instances {
		federationCfg.Instances = append(federationCfg.Instances, pipeline.FederationInstance{
			Name:     inst.Name,
			Endpoint: inst.Endpoint,
			OrgID:    inst.OrgID,
			Timeout:  inst.Timeout,
			Headers:  inst.Headers,
		})
		if inst.Timeout > maxTimeout {
			maxTimeout = inst.Timeout
		}
	}

	// Create middlewares
	federationSharder := pipeline.NewAsyncFederationSharder(federationCfg, logger)
	retryWare := pipeline.NewRetryWare(cfg.MaxRetries, cfg.Weights.RetryWithWeights, registerer)
	traceIDStatusCodeWare := pipeline.NewStatusCodeAdjustWareWithAllowedCode(http.StatusNotFound)
	urlDenyListWare := pipeline.NewURLDenyListWare(cfg.URLDenyList)
	headerStripWare := pipeline.NewStripHeadersWare(cfg.AllowedHeaders)
	tenantValidatorWare := pipeline.NewTenantValidatorMiddleware()

	// Build federated trace pipeline - no local sharding, no caching
	tracePipeline := pipeline.Build(
		[]pipeline.AsyncMiddleware[combiner.PipelineResponse]{
			headerStripWare,
			urlDenyListWare,
			tenantValidatorWare,
			federationSharder,
		},
		[]pipeline.Middleware{traceIDStatusCodeWare, retryWare},
		next)

	// Create handlers
	traces := newTraceIDHandler(cfg, tracePipeline, o, combiner.NewTypedTraceByID, logger, dataAccessController)
	tracesV2 := newTraceIDV2Handler(cfg, tracePipeline, o, combiner.NewTypedTraceByIDV2, logger, dataAccessController)

	// Create not supported handler for streaming gRPC endpoints
	notSupportedErr := fmt.Errorf("federation mode: streaming gRPC endpoints are not supported")

	f := &QueryFrontend{
		// http/discrete
		TraceByIDHandler:           newHandler(cfg.Config.LogQueryRequestHeaders, traces, logger),
		TraceByIDHandlerV2:         newHandler(cfg.Config.LogQueryRequestHeaders, tracesV2, logger),
		SearchHandler:              http.NotFoundHandler(),
		SearchTagsHandler:          http.NotFoundHandler(),
		SearchTagsV2Handler:        http.NotFoundHandler(),
		SearchTagsValuesHandler:    http.NotFoundHandler(),
		SearchTagsValuesV2Handler:  http.NotFoundHandler(),
		MetricsSummaryHandler:      http.NotFoundHandler(),
		MetricsQueryInstantHandler: http.NotFoundHandler(),
		MetricsQueryRangeHandler:   http.NotFoundHandler(),

		// grpc/streaming - federation mode does not support streaming endpoints
		streamingSearch: func(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
			return notSupportedErr
		},
		streamingTags: func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsServer) error {
			return notSupportedErr
		},
		streamingTagsV2: func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsV2Server) error {
			return notSupportedErr
		},
		streamingTagValues: func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesServer) error {
			return notSupportedErr
		},
		streamingTagValuesV2: func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
			return notSupportedErr
		},
		streamingQueryRange: func(req *tempopb.QueryRangeRequest, srv tempopb.StreamingQuerier_MetricsQueryRangeServer) error {
			return notSupportedErr
		},
		streamingQueryInstant: func(req *tempopb.QueryInstantRequest, srv tempopb.StreamingQuerier_MetricsQueryInstantServer) error {
			return notSupportedErr
		},

		cacheProvider: nil,
		logger:        logger,
	}

	// MCP is not supported in federation mode
	f.MCPHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	return f, nil
}
