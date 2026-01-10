package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/api"
)

// SearchTagsHandler handles search tags requests across all Tempo instances
func (h *Handler) SearchTagsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	req, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	level.Debug(h.logger).Log("msg", "searching tags", "scope", req.Scope, "start", req.Start, "end", req.End)

	// Query all instances in parallel
	results := h.querier.SearchTags(ctx, int64(req.Start), int64(req.End))

	// Combine tag results
	combinedResponse, err := h.combiner.CombineTagsResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tags results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tags results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tags search completed", "tagsFound", len(combinedResponse.TagNames))

	h.writeFormattedContentForRequest(w, r, combinedResponse)
}

// SearchTagsV2Handler handles v2 search tags requests across all Tempo instances
func (h *Handler) SearchTagsV2Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	req, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	level.Debug(h.logger).Log("msg", "searching tags v2", "scope", req.Scope, "start", req.Start, "end", req.End)

	// Query all instances in parallel
	results := h.querier.SearchTagsV2(ctx, int64(req.Start), int64(req.End))

	// Combine tag results
	combinedResponse, err := h.combiner.CombineTagsV2Results(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tags v2 results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tags v2 results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tags v2 search completed", "scopesFound", len(combinedResponse.Scopes))

	h.writeFormattedContentForRequest(w, r, combinedResponse)
}

// SearchTagValuesHandler handles search tag values requests across all Tempo instances
func (h *Handler) SearchTagValuesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	req, err := api.ParseSearchTagValuesRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	level.Debug(h.logger).Log("msg", "searching tag values", "tag", req.TagName)

	// Query all instances in parallel
	results := h.querier.SearchTagValues(ctx, req.TagName)

	// Combine tag values results
	combinedResponse, err := h.combiner.CombineTagValuesResults(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tag values results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tag values results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tag values search completed", "tag", req.TagName, "valuesFound", len(combinedResponse.TagValues))

	h.writeFormattedContentForRequest(w, r, combinedResponse)
}

// SearchTagValuesV2Handler handles v2 search tag values requests across all Tempo instances
func (h *Handler) SearchTagValuesV2Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.QueryTimeout)
	defer cancel()

	req, err := api.ParseSearchTagValuesRequestV2(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	level.Debug(h.logger).Log("msg", "searching tag values v2", "tag", req.TagName, "query", req.Query, "start", req.Start, "end", req.End)

	// Query all instances in parallel
	results := h.querier.SearchTagValuesV2(ctx, req.TagName, req.Query, int64(req.Start), int64(req.End))

	// Combine tag values results
	combinedResponse, err := h.combiner.CombineTagValuesV2Results(results)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to combine tag values v2 results", "err", err)
		http.Error(w, fmt.Sprintf("failed to combine tag values v2 results: %v", err), http.StatusInternalServerError)
		return
	}

	level.Debug(h.logger).Log("msg", "tag values v2 search completed", "tag", req.TagName, "valuesFound", len(combinedResponse.TagValues))

	h.writeFormattedContentForRequest(w, r, combinedResponse)
}
