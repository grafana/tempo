package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/client"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/combiner"
)

// FederatedQuerier defines the interface for coordinating queries across multiple Tempo instances
type FederatedQuerier interface {
	QueryAllInstances(ctx context.Context, queryFn func(ctx context.Context, c client.TempoClient) (*http.Response, error)) []combiner.QueryResult
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
