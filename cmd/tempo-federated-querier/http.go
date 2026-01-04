package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/tempopb"
)

// Handler handles HTTP requests for the federated querier
type Handler struct {
	querier  *FederatedQuerier
	combiner *TraceCombiner
	cfg      Config
	logger   log.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(querier *FederatedQuerier, cfg Config, logger log.Logger) *Handler {
	return &Handler{
		querier:  querier,
		combiner: NewTraceCombiner(cfg.MaxBytesPerTrace, logger),
		cfg:      cfg,
		logger:   logger,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Trace by ID endpoints
	r.HandleFunc("/api/traces/{traceID}", h.TraceByIDHandler).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/traces/{traceID}", h.TraceByIDV2Handler).Methods(http.MethodGet)

	// Health and info endpoints
	r.HandleFunc("/ready", h.ReadyHandler).Methods(http.MethodGet)
	r.HandleFunc("/api/echo", h.EchoHandler).Methods(http.MethodGet)
	r.HandleFunc("/api/status/buildinfo", h.BuildInfoHandler).Methods(http.MethodGet)
	r.HandleFunc("/api/status/instances", h.InstancesHandler).Methods(http.MethodGet)
}

// TraceByIDHandler handles trace by ID requests (v1 API)
func (h *Handler) TraceByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	traceID := vars["traceID"]

	if traceID == "" {
		http.Error(w, "traceID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Info(h.logger).Log("msg", "querying trace by ID", "traceID", traceID)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, client *TempoClient) (*http.Response, error) {
		return client.GetTraceByID(ctx, traceID)
	})

	// Combine results
	combinedTrace, metadata, err := h.combiner.CombineTraceResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine traces", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine traces: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if we found any trace
	if combinedTrace == nil || len(combinedTrace.ResourceSpans) == 0 {
		level.Info(h.logger).Log("msg", "trace not found", "traceID", traceID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	level.Info(h.logger).Log(
		"msg", "trace found",
		"traceID", traceID,
		"instancesQueried", metadata.InstancesQueried,
		"instancesWithTrace", metadata.InstancesWithTrace,
		"instancesNotFound", metadata.InstancesNotFound,
		"instancesFailed", metadata.InstancesFailed,
		"spanCount", metadata.TotalSpans,
	)

	// Encode response
	h.writeTraceResponse(w, r, combinedTrace, metadata)
}

// TraceByIDV2Handler handles trace by ID requests (v2 API with metrics)
func (h *Handler) TraceByIDV2Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	traceID := vars["traceID"]

	if traceID == "" {
		http.Error(w, "traceID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Info(h.logger).Log("msg", "querying trace by ID v2", "traceID", traceID)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, client *TempoClient) (*http.Response, error) {
		return client.GetTraceByIDV2(ctx, traceID)
	})

	// Combine results - use V2 combiner for wrapped response format
	combinedTrace, metadata, err := h.combiner.CombineTraceResultsV2(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine traces", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine traces: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if we found any trace
	if combinedTrace == nil || len(combinedTrace.ResourceSpans) == 0 {
		level.Info(h.logger).Log("msg", "trace not found", "traceID", traceID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	level.Info(h.logger).Log(
		"msg", "trace found (v2)",
		"traceID", traceID,
		"instancesQueried", metadata.InstancesQueried,
		"instancesWithTrace", metadata.InstancesWithTrace,
		"instancesNotFound", metadata.InstancesNotFound,
		"instancesFailed", metadata.InstancesFailed,
		"spanCount", metadata.TotalSpans,
	)

	// Encode response with metadata
	h.writeTraceResponseV2(w, r, combinedTrace, metadata)
}

// ReadyHandler returns ready status
func (h *Handler) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ready")
}

// EchoHandler echoes the request
func (h *Handler) EchoHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "echo")
}

// BuildInfoHandler returns build information
func (h *Handler) BuildInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version":   Version,
		"revision":  Revision,
		"branch":    Branch,
		"buildDate": BuildDate,
		"goVersion": GoVersion,
	}
	h.writeJSONResponse(w, info)
}

// InstancesHandler returns the list of configured Tempo instances
func (h *Handler) InstancesHandler(w http.ResponseWriter, r *http.Request) {
	instances := make([]map[string]interface{}, len(h.cfg.Instances))
	for i, inst := range h.cfg.Instances {
		instances[i] = map[string]interface{}{
			"name":     inst.Name,
			"endpoint": inst.Endpoint,
		}
	}
	h.writeJSONResponse(w, map[string]interface{}{
		"instances": instances,
	})
}

// writeTraceResponse writes the trace response in the appropriate format
// This wraps the trace in tempopb.TraceByIDResponse for compatibility with Grafana
func (h *Handler) writeTraceResponse(w http.ResponseWriter, r *http.Request, trace *tempopb.Trace, metadata *CombineMetadata) {
	// Create the proper response wrapper that Grafana expects
	resp := &tempopb.TraceByIDResponse{
		Trace:   trace,
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	accept := r.Header.Get("Accept")

	// Check if protobuf is requested
	if strings.Contains(accept, "application/protobuf") {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(resp)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal trace to protobuf", "err", err)
			http.Error(w, fmt.Sprintf("failed to marshal trace: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON - use jsonpb for proper protobuf JSON format
	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, resp); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal trace to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal trace: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeTraceResponseV2 writes the v2 trace response with metrics
// This wraps the trace in tempopb.TraceByIDResponse for compatibility with Grafana
func (h *Handler) writeTraceResponseV2(w http.ResponseWriter, r *http.Request, trace *tempopb.Trace, metadata *CombineMetadata) {
	// Determine partial status
	status := tempopb.PartialStatus_COMPLETE
	message := ""
	if metadata.PartialResponse {
		status = tempopb.PartialStatus_PARTIAL
		message = fmt.Sprintf("partial response: %d of %d instances responded with trace", metadata.InstancesWithTrace, metadata.InstancesQueried)
	}

	// Create the proper response wrapper that Grafana expects
	resp := &tempopb.TraceByIDResponse{
		Trace:   trace,
		Metrics: &tempopb.TraceByIDMetrics{},
		Status:  status,
		Message: message,
	}

	accept := r.Header.Get("Accept")

	// Check if protobuf is requested
	if strings.Contains(accept, "application/protobuf") {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(resp)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal trace to protobuf", "err", err)
			http.Error(w, fmt.Sprintf("failed to marshal trace: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON - use jsonpb for proper protobuf JSON format
	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, resp); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal trace to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal trace: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeJSONResponse writes a JSON response
func (h *Handler) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode JSON response", "err", err)
	}
}
