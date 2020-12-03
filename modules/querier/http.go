package querier

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
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
		TraceID:        byteID,
		BlockStart:     []byte(r.URL.Query().Get(BlockStartKey)),
		BlockEnd:       []byte(r.URL.Query().Get(BlockEndKey)),
		QueryIngesters: r.URL.Query().Get(QueryIngestersKey) == "true",
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
		level.Info(cortex_util.Logger).Log("msg", "received content type application/protobuf")
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
