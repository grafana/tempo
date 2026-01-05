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

	// Search endpoint
	r.HandleFunc("/api/search", h.SearchHandler).Methods(http.MethodGet)

	// Tags endpoints
	r.HandleFunc("/api/search/tags", h.SearchTagsHandler).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/search/tags", h.SearchTagsV2Handler).Methods(http.MethodGet)
	r.HandleFunc("/api/search/tag/{tagName}/values", h.SearchTagValuesHandler).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/search/tag/{tagName}/values", h.SearchTagValuesV2Handler).Methods(http.MethodGet)

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

// SearchHandler handles search requests across all Tempo instances
func (h *Handler) SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Pass through all query parameters to each instance
	queryParams := r.URL.RawQuery

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Info(h.logger).Log("msg", "searching traces", "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, client *TempoClient) (*http.Response, error) {
		return client.Search(ctx, queryParams)
	})

	// Combine search results
	combinedResponse, metadata, err := h.combiner.CombineSearchResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine search results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine search results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Info(h.logger).Log(
		"msg", "search completed",
		"instancesQueried", metadata.InstancesQueried,
		"instancesResponded", metadata.InstancesResponded,
		"instancesFailed", metadata.InstancesFailed,
		"tracesFound", len(combinedResponse.Traces),
	)

	// Write response
	h.writeSearchResponse(w, r, combinedResponse)
}

// SearchTagsHandler handles search tags requests across all Tempo instances
func (h *Handler) SearchTagsHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.RawQuery

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Debug(h.logger).Log("msg", "searching tags", "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, client *TempoClient) (*http.Response, error) {
		return client.SearchTags(ctx, queryParams)
	})

	// Combine tag results
	combinedResponse, err := h.combiner.CombineTagsResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tags results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tags results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tags search completed", "tagsFound", len(combinedResponse.TagNames))

	// Write response
	h.writeTagsResponse(w, r, combinedResponse)
}

// SearchTagsV2Handler handles v2 search tags requests across all Tempo instances
func (h *Handler) SearchTagsV2Handler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.RawQuery

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Debug(h.logger).Log("msg", "searching tags v2", "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, client *TempoClient) (*http.Response, error) {
		return client.SearchTagsV2(ctx, queryParams)
	})

	// Combine tag results
	combinedResponse, err := h.combiner.CombineTagsV2Results(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tags v2 results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tags v2 results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tags v2 search completed", "scopesFound", len(combinedResponse.Scopes))

	// Write response
	h.writeTagsV2Response(w, r, combinedResponse)
}

// SearchTagValuesHandler handles search tag values requests across all Tempo instances
func (h *Handler) SearchTagValuesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tagName := vars["tagName"]
	queryParams := r.URL.RawQuery

	if tagName == "" {
		http.Error(w, "tagName is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Debug(h.logger).Log("msg", "searching tag values", "tag", tagName, "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, client *TempoClient) (*http.Response, error) {
		return client.SearchTagValues(ctx, tagName, queryParams)
	})

	// Combine tag values results
	combinedResponse, err := h.combiner.CombineTagValuesResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tag values results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tag values results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tag values search completed", "tag", tagName, "valuesFound", len(combinedResponse.TagValues))

	// Write response
	h.writeTagValuesResponse(w, r, combinedResponse)
}

// SearchTagValuesV2Handler handles v2 search tag values requests across all Tempo instances
func (h *Handler) SearchTagValuesV2Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tagName := vars["tagName"]
	queryParams := r.URL.RawQuery

	if tagName == "" {
		http.Error(w, "tagName is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Debug(h.logger).Log("msg", "searching tag values v2", "tag", tagName, "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, client *TempoClient) (*http.Response, error) {
		return client.SearchTagValuesV2(ctx, tagName, queryParams)
	})

	// Combine tag values results
	combinedResponse, err := h.combiner.CombineTagValuesV2Results(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tag values v2 results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tag values v2 results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tag values v2 search completed", "tag", tagName, "valuesFound", len(combinedResponse.TagValues))

	// Write response
	h.writeTagValuesV2Response(w, r, combinedResponse)
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

// writeSearchResponse writes the search response in the appropriate format
func (h *Handler) writeSearchResponse(w http.ResponseWriter, r *http.Request, resp *tempopb.SearchResponse) {
	accept := r.Header.Get("Accept")

	// Check if protobuf is requested
	if strings.Contains(accept, "application/protobuf") {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(resp)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal search response to protobuf", "err", err)
			http.Error(w, fmt.Sprintf("failed to marshal search response: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON - use jsonpb for proper protobuf JSON format
	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, resp); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal search response to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal search response: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeTagsResponse writes the tags response in the appropriate format
func (h *Handler) writeTagsResponse(w http.ResponseWriter, r *http.Request, resp *tempopb.SearchTagsResponse) {
	accept := r.Header.Get("Accept")

	// Check if protobuf is requested
	if strings.Contains(accept, "application/protobuf") {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(resp)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal tags response to protobuf", "err", err)
			http.Error(w, fmt.Sprintf("failed to marshal tags response: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON - use jsonpb for proper protobuf JSON format
	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, resp); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal tags response to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal tags response: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeTagsV2Response writes the v2 tags response in the appropriate format
func (h *Handler) writeTagsV2Response(w http.ResponseWriter, r *http.Request, resp *tempopb.SearchTagsV2Response) {
	accept := r.Header.Get("Accept")

	// Check if protobuf is requested
	if strings.Contains(accept, "application/protobuf") {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(resp)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal tags v2 response to protobuf", "err", err)
			http.Error(w, fmt.Sprintf("failed to marshal tags v2 response: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON - use jsonpb for proper protobuf JSON format
	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, resp); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal tags v2 response to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal tags v2 response: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeTagValuesResponse writes the tag values response in the appropriate format
func (h *Handler) writeTagValuesResponse(w http.ResponseWriter, r *http.Request, resp *tempopb.SearchTagValuesResponse) {
	accept := r.Header.Get("Accept")

	// Check if protobuf is requested
	if strings.Contains(accept, "application/protobuf") {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(resp)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal tag values response to protobuf", "err", err)
			http.Error(w, fmt.Sprintf("failed to marshal tag values response: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON - use jsonpb for proper protobuf JSON format
	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, resp); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal tag values response to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal tag values response: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeTagValuesV2Response writes the v2 tag values response in the appropriate format
func (h *Handler) writeTagValuesV2Response(w http.ResponseWriter, r *http.Request, resp *tempopb.SearchTagValuesV2Response) {
	accept := r.Header.Get("Accept")

	// Check if protobuf is requested
	if strings.Contains(accept, "application/protobuf") {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(resp)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal tag values v2 response to protobuf", "err", err)
			http.Error(w, fmt.Sprintf("failed to marshal tag values v2 response: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON - use jsonpb for proper protobuf JSON format
	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, resp); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal tag values v2 response to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal tag values v2 response: %v", err), http.StatusInternalServerError)
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
