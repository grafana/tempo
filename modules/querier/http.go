package querier

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

const (
	TraceIDVar = "traceID"
)

// TraceByIDHandler is a http.HandlerFunc to retrieve traces
func (q *Querier) TraceByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Enforce the query timeout while querying backends
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(q.cfg.QueryTimeout))
	defer cancel()

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

	resp, err := q.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID: byteID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.Trace == nil || len(resp.Trace.Batches) == 0 {
		http.Error(w, fmt.Sprintf("Unable to find %s", traceID), http.StatusNotFound)
		return
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp.Trace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
