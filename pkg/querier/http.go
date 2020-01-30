package querier

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/util"
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

	resp, err := q.FindTraceByID(ctx, &friggpb.TraceByIDRequest{
		TraceID: byteID,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(resp.Trace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
