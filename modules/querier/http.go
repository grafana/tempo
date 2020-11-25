package querier

import (
	"context"
	"fmt"
	cortex_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/proto"
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

	// todo: this logic needs to move into FindTraceByID
	isSharded := false
	blockStart := r.URL.Query().Get("blockStart")
	blockEnd := r.URL.Query().Get("blockStart")
	if len(blockStart) > 0 && len(blockEnd) > 0 {
		isSharded = true
	}

	byteID, err := util.HexStringToTraceID(traceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tidReq := &tempopb.TraceByIDRequest{
		TraceID: byteID,
	}
	if isSharded {
		tidReq.BlockEnd = []byte(blockEnd)
		tidReq.BlockStart = []byte(blockStart)
	}

	resp, err := q.FindTraceByID(ctx, tidReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.Trace == nil || len(resp.Trace.Batches) == 0 {
		http.Error(w, fmt.Sprintf("Unable to find %s", traceID), http.StatusNotFound)
		return
	}

	if r.Header.Get("Tempo-query-content-type") == "application/grpc" {
		level.Info(cortex_util.Logger).Log("msg", "received content type application/grpc")
		trace := &tempopb.Trace{Batches: resp.Trace.Batches}
		b, err := proto.Marshal(trace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp.Trace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
