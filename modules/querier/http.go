package querier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/golang/protobuf/proto"  //nolint:all //ProtoReflect
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	BlockStartKey = "blockStart"
	BlockEndKey   = "blockEnd"
	QueryModeKey  = "mode"

	QueryModeIngesters = "ingesters"
	QueryModeBlocks    = "blocks"
	QueryModeAll       = "all"
	QueryModeRecent    = "recent"
)

// TraceByIDHandler is a http.HandlerFunc to retrieve traces
func (q *Querier) TraceByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.TraceByID.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.TraceByIDHandler")
	defer span.End()

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
	span.AddEvent("validated request", oteltrace.WithAttributes(
		attribute.String("blockStart", blockStart),
		attribute.String("blockEnd", blockEnd),
		attribute.String("queryMode", queryMode),
		attribute.String("timeStart", fmt.Sprint(timeStart)),
		attribute.String("timeEnd", fmt.Sprint(timeEnd)),
		attribute.String("apiVersion", "v1"),
	))

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
	if resp.Trace == nil || len(resp.Trace.ResourceSpans) == 0 {
		w.WriteHeader(http.StatusNotFound)
	}

	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) TraceByIDHandlerV2(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.TraceByID.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.TraceByIDHandlerV2")
	defer span.End()

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
	span.AddEvent("validated request", oteltrace.WithAttributes(
		attribute.String("blockStart", blockStart),
		attribute.String("blockEnd", blockEnd),
		attribute.String("queryMode", queryMode),
		attribute.String("timeStart", fmt.Sprint(timeStart)),
		attribute.String("timeEnd", fmt.Sprint(timeEnd)),
		attribute.String("apiVersion", "v2"),
	))

	resp, err := q.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID:           byteID,
		BlockStart:        blockStart,
		BlockEnd:          blockEnd,
		QueryMode:         queryMode,
		AllowPartialTrace: true,
	}, timeStart, timeEnd)
	if err != nil {
		handleError(w, err)
		return
	}
	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) SearchHandler(w http.ResponseWriter, r *http.Request) {
	isSearchBlock := api.IsSearchBlock(r)

	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.SearchHandler")
	defer span.End()

	span.SetAttributes(attribute.String("requestURI", r.RequestURI))
	span.SetAttributes(attribute.Bool("isSearchBlock", isSearchBlock))

	var resp *tempopb.SearchResponse
	if !isSearchBlock {
		req, err := api.ParseSearchRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		span.SetAttributes(attribute.String("SearchRequest", req.String()))

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

		span.SetAttributes(attribute.String("SearchRequestBlock", req.String()))

		resp, err = q.SearchBlock(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	}

	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) SearchTagsHandler(w http.ResponseWriter, r *http.Request) {
	isSearchBlock := api.IsSearchBlock(r)

	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.SearchTagsHandler")
	defer span.End()

	var resp *tempopb.SearchTagsResponse
	if !isSearchBlock {
		req, err := api.ParseSearchTagsRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err = q.SearchTags(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	} else {
		req, err := api.ParseSearchTagsBlockRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err = q.SearchTagsBlocks(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	}
	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) SearchTagsV2Handler(w http.ResponseWriter, r *http.Request) {
	isSearchBlock := api.IsSearchBlock(r)

	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.SearchTagsHandler")
	defer span.End()

	var resp *tempopb.SearchTagsV2Response
	if !isSearchBlock {
		req, err := api.ParseSearchTagsRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err = q.SearchTagsV2(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	} else {
		req, err := api.ParseSearchTagsBlockRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err = q.SearchTagsBlocksV2(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	}

	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) SearchTagValuesHandler(w http.ResponseWriter, r *http.Request) {
	isSearchBlock := api.IsSearchBlock(r)

	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.SearchTagValuesHandler")
	defer span.End()

	var resp *tempopb.SearchTagValuesResponse
	if !isSearchBlock {
		req, err := api.ParseSearchTagValuesRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err = q.SearchTagValues(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	} else {
		req, err := api.ParseSearchTagValuesBlockRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err = q.SearchTagValuesBlocks(ctx, req)
		if err != nil {
			handleError(w, err)
			return
		}
	}

	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) SearchTagValuesV2Handler(w http.ResponseWriter, r *http.Request) {
	isSearchBlock := api.IsSearchBlock(r)

	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.SearchTagValuesHandler")
	defer span.End()

	var resp *tempopb.SearchTagValuesV2Response
	var err error

	if !isSearchBlock {
		var req *tempopb.SearchTagValuesRequest
		req, err = api.ParseSearchTagValuesRequestV2(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err = q.SearchTagValuesV2(ctx, req)
	} else {
		var req *tempopb.SearchTagValuesBlockRequest
		req, err = api.ParseSearchTagValuesBlockRequestV2(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err = q.SearchTagValuesBlocksV2(ctx, req)
	}

	if err != nil {
		handleError(w, err)
		return
	}

	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) SpanMetricsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.SpanMetricsSummaryHandler")
	defer span.End()

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

	writeFormattedContentForRequest(w, r, resp, span)
}

func (q *Querier) QueryRangeHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		resp *tempopb.QueryRangeResponse
	)

	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.Search.QueryTimeout))
	defer cancel()

	ctx, span := tracer.Start(ctx, "Querier.QueryRangeHandler")
	defer span.End()

	errHandler := func(ctx context.Context, span oteltrace.Span, err error) {
		if errors.Is(err, context.Canceled) {
			// todo: context is also canceled when we hit the query timeout. research what the behavior is
			// ignore this error. we regularly cancel context once queries are complete
			span.RecordError(err)
			return
		}

		if ctx.Err() != nil {
			span.RecordError(ctx.Err())
			return
		}

		if err != nil {
			span.RecordError(err)
		}
	}

	defer func() {
		errHandler(ctx, span, err)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeFormattedContentForRequest(w, r, resp, span)
	}()

	req, err := api.ParseQueryRangeRequest(r)
	if err != nil {
		errHandler(ctx, span, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.String("query", req.Query))
	span.SetAttributes(attribute.Int64("step", time.Duration(req.Step).Nanoseconds()))
	span.SetAttributes(attribute.Int64("interval", time.Unix(0, int64(req.End)).Sub(time.Unix(0, int64(req.Start))).Nanoseconds()))

	resp, err = q.QueryRange(ctx, req)
	if err != nil {
		errHandler(ctx, span, err)
		return
	}

	if resp != nil && resp.Metrics != nil {
		span.SetAttributes(attribute.Int64("inspectedBytes", int64(resp.Metrics.InspectedBytes)))
		span.SetAttributes(attribute.Int64("inspectedSpans", int64(resp.Metrics.InspectedSpans)))
	}
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

func writeFormattedContentForRequest(w http.ResponseWriter, req *http.Request, m proto.Message, span oteltrace.Span) {
	switch req.Header.Get(api.HeaderAccept) {
	case api.HeaderAcceptProtobuf:
		b, err := proto.Marshal(m)
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
		if span != nil {
			span.SetAttributes(attribute.String("contentType", api.HeaderAcceptProtobuf))
		}

	default:
		w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
		err := new(jsonpb.Marshaler).Marshal(w, m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if span != nil {
			span.SetAttributes(attribute.String("contentType", api.HeaderAcceptJSON))
		}

	}
}
