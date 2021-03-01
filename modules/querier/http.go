package querier

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
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
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.QueryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.TraceByIDHandler")
	defer span.Finish()

	byteID, err := util.ParseTraceID(r)
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

	if resp.Trace == nil || len(resp.Trace.Batches) == 0 {
		http.Error(w, fmt.Sprintf("Unable to find %s", hex.EncodeToString(byteID)), http.StatusNotFound)
		return
	}

	if r.Header.Get(util.AcceptHeaderKey) == util.ProtobufTypeHeaderValue {
		span.SetTag("response marshalling format", util.ProtobufTypeHeaderValue)
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

	span.SetTag("response marshalling format", util.JSONTypeHeaderValue)
	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp.Trace)
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
		return "", "", "", fmt.Errorf("invalid value for queryMode %s", q)
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
