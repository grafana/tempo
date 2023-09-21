package querier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/golang/protobuf/proto"  //nolint:all //ProtoReflect
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
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
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.TraceByID.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.TraceByIDHandler")
	defer span.Finish()

	byteID, err := api.ParseTraceID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate request
	blockStart, blockEnd, queryMode, timeStart, timeEnd, err := api.ValidateAndSanitizeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	span.LogFields(
		ot_log.String("msg", "validated request"),
		ot_log.String("blockStart", blockStart),
		ot_log.String("blockEnd", blockEnd),
		ot_log.String("queryMode", queryMode),
		ot_log.String("timeStart", fmt.Sprint(timeStart)),
		ot_log.String("timeEnd", fmt.Sprint(timeEnd)))

	resp, err := q.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID:    byteID,
		BlockStart: blockStart,
		BlockEnd:   blockEnd,
		QueryMode:  queryMode,
	}, timeStart, timeEnd)
	if err != nil {
		handleError(w, err)
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
		w.Header().Set(api.HeaderContentType, api.HeaderAcceptProtobuf)
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
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}

func (q *Querier) SearchHandler(w http.ResponseWriter, r *http.Request) {
	isSearchBlock := api.IsSearchBlock(r)

	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchHandler")
	defer span.Finish()

	span.SetTag("requestURI", r.RequestURI)
	span.SetTag("isSearchBlock", isSearchBlock)

	var resp *tempopb.SearchResponse
	if !isSearchBlock {
		req, err := api.ParseSearchRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		span.SetTag("SearchRequest", req.String())

		resp, err = q.SearchRecent(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	} else {
		req, err := api.ParseSearchBlockRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		span.SetTag("SearchRequestBlock", req.String())

		resp, err = q.SearchBlock(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	}

	marshaller := &jsonpb.Marshaler{}
	err := marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}

func (q *Querier) SearchTagsHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchTagsHandler")
	defer span.Finish()

	req, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := q.SearchTags(ctx, req)
	if err != nil {
		handleError(w, err)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}

func (q *Querier) SearchTagsV2Handler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchTagsHandler")
	defer span.Finish()

	req, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := q.SearchTagsV2(ctx, req)
	if err != nil {
		handleError(w, err)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}

func (q *Querier) SearchTagValuesHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchTagValuesHandler")
	defer span.Finish()

	req, err := api.ParseSearchTagValuesRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := q.SearchTagValues(ctx, req)
	if err != nil {
		handleError(w, err)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}

func (q *Querier) SearchTagValuesV2Handler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SearchTagValuesHandler")
	defer span.Finish()

	req, err := api.ParseSearchTagValuesRequestV2(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := q.SearchTagValuesV2(ctx, req)
	if err != nil {
		handleError(w, err)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}

func (q *Querier) SpanMetricsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.SpanMetricsSummaryHandler")
	defer span.Finish()

	req, err := api.ParseSpanMetricsSummaryRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := q.SpanMetricsSummary(ctx, req)
	if err != nil {
		handleError(w, err)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}

func handleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	if errors.Is(err, context.Canceled) {
		// todo: context is also canceled when we hit the query timeout. research what the behavior is
		// ignore this error. we regularly cancel context once queries are complete
		return
	}

	// todo: better understand all errors returned from queriers and categorize more as 4XX
	if errors.Is(err, trace.ErrTraceTooLarge) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Error(w, err.Error(), http.StatusInternalServerError)
}
