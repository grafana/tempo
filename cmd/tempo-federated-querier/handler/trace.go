package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

// TraceByIDHandler handles trace by ID requests (v1 API)
func (h *Handler) TraceByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	byteID, err := api.ParseTraceID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	traceID := fmt.Sprintf("%x", byteID)

	level.Info(h.logger).Log("msg", "querying trace by ID", "traceID", traceID)

	// Query all instances in parallel
	results := h.querier.QueryTraces(ctx, traceID)

	// Combine results
	combinedTrace, metadata, err := h.combiner.CombineTraceResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine traces", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine traces: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if we found any trace - write 404 but continue to marshal response
	if combinedTrace == nil || len(combinedTrace.ResourceSpans) == 0 {
		level.Info(h.logger).Log("msg", "trace not found", "traceID", traceID)
		w.WriteHeader(http.StatusNotFound)
	}

	level.Info(h.logger).Log(
		"msg", "trace query completed",
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

	h.writeFormattedContentForRequest(w, r, resp)
}

// TraceByIDV2Handler handles trace by ID requests (v2 API with metrics)
func (h *Handler) TraceByIDV2Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	byteID, err := api.ParseTraceID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	traceID := fmt.Sprintf("%x", byteID)

	level.Info(h.logger).Log("msg", "querying trace by ID v2", "traceID", traceID)

	// Query all instances in parallel
	results := h.querier.QueryTracesV2(ctx, traceID)

	// Combine results - use V2 combiner for wrapped response format
	combinedTrace, metadata, err := h.combiner.CombineTraceResultsV2(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine traces", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine traces: %v", err), http.StatusInternalServerError)
		return
	}

	level.Info(h.logger).Log(
		"msg", "trace v2 query completed",
		"traceID", traceID,
		"instancesQueried", metadata.InstancesQueried,
		"instancesWithTrace", metadata.InstancesWithTrace,
		"instancesNotFound", metadata.InstancesNotFound,
		"instancesFailed", metadata.InstancesFailed,
		"spanCount", metadata.TotalSpans,
	)

	// Create the proper response wrapper
	resp := &tempopb.TraceByIDResponse{
		Trace:   combinedTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	h.writeFormattedContentForRequest(w, r, resp)
}
