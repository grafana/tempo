package querier

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
)

const (
	BlockStartKey = "blockStart"
	BlockEndKey   = "blockEnd"
	QueryModeKey  = "mode"

	QueryModeIngesters = "ingesters"
	QueryModeBlocks    = "blocks"
	QueryModeAll       = "all"
)

// TraceByIDHandler is a http.HandlerFunc to retrieve traces
func (q *Querier) TraceByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.TraceLookupQueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.TraceByIDHandler")
	defer span.Finish()

	byteID, err := api.ParseTraceID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate request
	blockStart, blockEnd, queryMode, err := validateAndSanitizeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	span.LogFields(
		ot_log.String("msg", "validated request"),
		ot_log.String("blockStart", blockStart),
		ot_log.String("blockEnd", blockEnd),
		ot_log.String("queryMode", queryMode))

	resp, err := q.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID:    byteID,
		BlockStart: blockStart,
		BlockEnd:   blockEnd,
		QueryMode:  queryMode,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// record not found here, but continue on so we can marshal metrics
	// to the body
	if resp.Trace == nil || len(resp.Trace.Batches) == 0 {
		w.WriteHeader(http.StatusNotFound)
	}

	if r.Header.Get(api.HeaderAccept) == api.HeaderAcceptProtobuf {
		span.SetTag("contentType", api.HeaderAcceptProtobuf)
		b, err := proto.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	span.SetTag("contentType", api.HeaderAcceptJSON)
	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// return values are (blockStart, blockEnd, queryMode, error)
func validateAndSanitizeRequest(r *http.Request) (string, string, string, error) {
	q := r.URL.Query().Get(QueryModeKey)

	// validate queryMode. it should either be empty or one of (QueryModeIngesters|QueryModeBlocks|QueryModeAll)
	var queryMode string
	if len(q) == 0 || q == QueryModeAll {
		queryMode = QueryModeAll
	} else if q == QueryModeIngesters {
		queryMode = QueryModeIngesters
	} else if q == QueryModeBlocks {
		queryMode = QueryModeBlocks
	} else {
		return "", "", "", fmt.Errorf("invalid value for mode %s", q)
	}

	// no need to validate/sanitize other parameters if queryMode == QueryModeIngesters
	if queryMode == QueryModeIngesters {
		return "", "", queryMode, nil
	}

	start := r.URL.Query().Get(BlockStartKey)
	end := r.URL.Query().Get(BlockEndKey)

	// validate start. it should either be empty or a valid uuid
	if len(start) == 0 {
		start = tempodb.BlockIDMin
	} else {
		_, err := uuid.Parse(start)
		if err != nil {
			return "", "", "", errors.Wrap(err, "invalid value for blockStart")
		}
	}

	// validate end. it should either be empty or a valid uuid
	if len(end) == 0 {
		end = tempodb.BlockIDMax
	} else {
		_, err := uuid.Parse(end)
		if err != nil {
			return "", "", "", errors.Wrap(err, "invalid value for blockEnd")
		}
	}

	return start, end, queryMode, nil
}

func (q *Querier) SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.SearchQueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchHandler")
	defer span.Finish()

	span.SetTag("requestURI", r.RequestURI)

	req, err := q.parseSearchRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	span.SetTag("SearchRequest", req.String())

	resp, err := q.Search(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (q *Querier) SearchTagsHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.SearchQueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchTagsHandler")
	defer span.Finish()

	req := &tempopb.SearchTagsRequest{}

	resp, err := q.SearchTags(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (q *Querier) SearchTagValuesHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.SearchQueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchTagValuesHandler")
	defer span.Finish()

	vars := mux.Vars(r)
	tagName, ok := vars["tagName"]
	if !ok {
		http.Error(w, "please provide a tagName", http.StatusBadRequest)
		return
	}
	req := &tempopb.SearchTagValuesRequest{
		TagName: tagName,
	}

	resp, err := q.SearchTagValues(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// todo(search): consolidate
func (q *Querier) BackendSearchHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.SearchQueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.BackendSearch")
	defer span.Finish()

	searchReq, err := q.parseSearchRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// parseSearchRequest doesn't respect these as "reserved" tags. let's remove them here.
	// this will all be cleaned up when search paths are consolidated.
	delete(searchReq.Tags, api.URLParamBlockID)
	delete(searchReq.Tags, api.URLParamStartPage)
	delete(searchReq.Tags, api.URLParamTotalPages)
	delete(searchReq.Tags, api.URLParamStart)
	delete(searchReq.Tags, api.URLParamEnd)

	start, end, _, err := api.ParseBackendSearch(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	startPage, totalPages, blockID, err := api.ParseBackendSearchQuerier(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// extract params and populate
	req := &tempopb.BackendSearchRequest{
		Search:     searchReq,
		Start:      uint32(start),
		End:        uint32(end),
		StartPage:  startPage,
		TotalPages: totalPages,
		BlockID:    blockID[:],
	}

	resp, err := q.BackendSearch(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
