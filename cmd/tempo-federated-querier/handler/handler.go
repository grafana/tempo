package handler

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/combiner"
)

// FederatedQuerier defines the interface for coordinating queries across multiple Tempo instances
type FederatedQuerier interface {
	QueryTraces(ctx context.Context, traceID string) []combiner.TraceResult
	QueryTracesV2(ctx context.Context, traceID string) []combiner.TraceByIDResult
	Search(ctx context.Context, query string, start, end int64) []combiner.SearchResult
	SearchTags(ctx context.Context, start, end int64) []combiner.SearchTagsResult
	SearchTagsV2(ctx context.Context, start, end int64) []combiner.SearchTagsV2Result
	SearchTagValues(ctx context.Context, tagName string) []combiner.SearchTagValuesResult
	SearchTagValuesV2(ctx context.Context, tagName string, query string, start, end int64) []combiner.SearchTagValuesV2Result
	Instances() []string
}

// Config holds handler-specific configuration
type Config struct {
	QueryTimeout time.Duration
	Instances    []InstanceInfo
}

// InstanceInfo holds basic instance information for status endpoints
type InstanceInfo struct {
	Name     string
	Endpoint string
}

// BuildInfo holds build version information
type BuildInfo struct {
	Version   string
	Revision  string
	Branch    string
	BuildDate string
	GoVersion string
}

// Handler handles HTTP requests for the federated querier
type Handler struct {
	querier   FederatedQuerier
	combiner  *combiner.Combiner
	cfg       Config
	buildInfo BuildInfo
	logger    log.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(querier FederatedQuerier, comb *combiner.Combiner, cfg Config, buildInfo BuildInfo, logger log.Logger) *Handler {
	return &Handler{
		querier:   querier,
		combiner:  comb,
		cfg:       cfg,
		buildInfo: buildInfo,
		logger:    logger,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Trace by ID endpoints
	r.HandleFunc("/api/traces/{traceID}", h.TraceByIDHandler).Methods("GET")
	r.HandleFunc("/api/v2/traces/{traceID}", h.TraceByIDV2Handler).Methods("GET")

	// Search endpoint
	r.HandleFunc("/api/search", h.SearchHandler).Methods("GET")

	// Tags endpoints
	r.HandleFunc("/api/search/tags", h.SearchTagsHandler).Methods("GET")
	r.HandleFunc("/api/v2/search/tags", h.SearchTagsV2Handler).Methods("GET")
	r.HandleFunc("/api/search/tag/{tagName}/values", h.SearchTagValuesHandler).Methods("GET")
	r.HandleFunc("/api/v2/search/tag/{tagName}/values", h.SearchTagValuesV2Handler).Methods("GET")

	// Health and info endpoints
	r.HandleFunc("/ready", h.ReadyHandler).Methods("GET")
	r.HandleFunc("/api/echo", h.EchoHandler).Methods("GET")
	r.HandleFunc("/api/status/buildinfo", h.BuildInfoHandler).Methods("GET")
	r.HandleFunc("/api/status/instances", h.InstancesHandler).Methods("GET")
}
