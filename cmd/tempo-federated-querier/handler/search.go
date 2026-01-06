package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/client"
)

// SearchHandler handles search requests across all Tempo instances
func (h *Handler) SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Pass through all query parameters to each instance
	queryParams := r.URL.RawQuery

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Info(h.logger).Log("msg", "searching traces", "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, c client.TempoClient) (*http.Response, error) {
		return c.Search(ctx, queryParams)
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

	h.writeProtoResponse(w, r, combinedResponse)
}
