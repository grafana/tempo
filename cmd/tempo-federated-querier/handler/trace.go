package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/client"
	"github.com/grafana/tempo/pkg/tempopb"
)

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
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, c client.TempoClient) (*http.Response, error) {
		return c.GetTraceByID(ctx, traceID)
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

	// Create the proper response wrapper that Grafana expects
	resp := &tempopb.TraceByIDResponse{
		Trace:   combinedTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	h.writeProtoResponse(w, r, resp)
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
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, c client.TempoClient) (*http.Response, error) {
		return c.GetTraceByIDV2(ctx, traceID)
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

	// Determine partial status
	status := tempopb.PartialStatus_COMPLETE
	message := ""
	if metadata.PartialResponse {
		status = tempopb.PartialStatus_PARTIAL
		message = fmt.Sprintf("partial response: %d of %d instances responded with trace", metadata.InstancesWithTrace, metadata.InstancesQueried)
	}

	// Create the proper response wrapper that Grafana expects
	resp := &tempopb.TraceByIDResponse{
		Trace:   combinedTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
		Status:  status,
		Message: message,
	}

	h.writeProtoResponse(w, r, resp)
}
