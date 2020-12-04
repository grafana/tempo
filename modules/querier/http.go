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
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
)

const (
	TraceIDVar        = "traceID"
	BlockStartKey     = "blockStart"
	BlockEndKey       = "blockEnd"
	QueryIngestersKey = "queryIngesters"
)

// TraceByIDHandler is a http.HandlerFunc to retrieve traces
func (q *Querier) TraceByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.TraceByIDHandler")
	defer span.Finish()

	vars := mux.Vars(r)
	traceID, ok := vars[TraceIDVar]
	if !ok {
		http.Error(w, "please provide a traceID", http.StatusBadRequest)
		return
	}

	byteID, err := util.HexStringToTraceID(traceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate request
	valid, blockStart, blockEnd, queryIngesters := validateAndSanitizeRequest(r)
	if !valid {
		http.Error(w, "invalid parameters", http.StatusBadRequest)
		return
	}
	span.LogFields(
		ot_log.String("msg", "validated request"),
		ot_log.String("blockStart", blockStart),
		ot_log.String("blockEnd", blockEnd),
		ot_log.Bool("queryIngesters", queryIngesters))

	resp, err := q.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID:        byteID,
		BlockStart:     blockStart,
		BlockEnd:       blockEnd,
		QueryIngesters: queryIngesters,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.Trace == nil || len(resp.Trace.Batches) == 0 {
		http.Error(w, fmt.Sprintf("Unable to find %s", traceID), http.StatusNotFound)
		return
	}

	if r.Header.Get(util.ContentTypeHeaderKey) == util.ProtobufTypeHeaderValue {
		b, err := proto.Marshal(resp.Trace)
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

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp.Trace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// return values are (valid, blockStart, blockEnd, queryIngesters)
func validateAndSanitizeRequest(r *http.Request) (bool, string, string, bool) {
	// get parameter values
	q := r.URL.Query().Get(QueryIngestersKey)
	start := r.URL.Query().Get(BlockStartKey)
	end := r.URL.Query().Get(BlockEndKey)

	// validate queryIngesters. it should either be empty or one of (true|false)
	var queryIngesters bool
	if len(q) == 0 || q == "true" {
		queryIngesters = true
	} else if q == "false" {
		queryIngesters = false
	} else {
		return false, "", "", false
	}

	// validate start. it should either be empty or a valid uuid
	if len(start) == 0 {
		start = tempodb.BlockIDMin
	} else {
		_, err := uuid.Parse(start)
		if err != nil {
			return false, "", "", false
		}
	}

	// validate end. it should either be empty or a valid uuid
	if len(end) == 0 {
		end = tempodb.BlockIDMax
	} else {
		_, err := uuid.Parse(end)
		if err != nil {
			return false, "", "", false
		}
	}

	return true, start, end, queryIngesters
}
