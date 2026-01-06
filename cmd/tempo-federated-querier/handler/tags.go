package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/client"
)

// SearchTagsHandler handles search tags requests across all Tempo instances
func (h *Handler) SearchTagsHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.RawQuery

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Debug(h.logger).Log("msg", "searching tags", "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, c client.TempoClient) (*http.Response, error) {
		return c.SearchTags(ctx, queryParams)
	})

	// Combine tag results
	combinedResponse, err := h.combiner.CombineTagsResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tags results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tags results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tags search completed", "tagsFound", len(combinedResponse.TagNames))

	h.writeProtoResponse(w, r, combinedResponse)
}

// SearchTagsV2Handler handles v2 search tags requests across all Tempo instances
func (h *Handler) SearchTagsV2Handler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.RawQuery

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	level.Debug(h.logger).Log("msg", "searching tags v2", "query", queryParams)

	// Query all instances in parallel
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, c client.TempoClient) (*http.Response, error) {
		return c.SearchTagsV2(ctx, queryParams)
	})

	// Combine tag results
	combinedResponse, err := h.combiner.CombineTagsV2Results(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tags v2 results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tags v2 results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tags v2 search completed", "scopesFound", len(combinedResponse.Scopes))

	h.writeProtoResponse(w, r, combinedResponse)
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
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, c client.TempoClient) (*http.Response, error) {
		return c.SearchTagValues(ctx, tagName, queryParams)
	})

	// Combine tag values results
	combinedResponse, err := h.combiner.CombineTagValuesResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tag values results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tag values results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tag values search completed", "tag", tagName, "valuesFound", len(combinedResponse.TagValues))

	h.writeProtoResponse(w, r, combinedResponse)
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
	results := h.querier.QueryAllInstances(ctx, func(ctx context.Context, c client.TempoClient) (*http.Response, error) {
		return c.SearchTagValuesV2(ctx, tagName, queryParams)
	})

	// Combine tag values results
	combinedResponse, err := h.combiner.CombineTagValuesV2Results(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tag values v2 results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tag values v2 results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tag values v2 search completed", "tag", tagName, "valuesFound", len(combinedResponse.TagValues))

	h.writeProtoResponse(w, r, combinedResponse)
}
