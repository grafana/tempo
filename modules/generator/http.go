package generator

import (
	"context"
	"net/http"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/opentracing/opentracing-go"
)

func (g *Generator) SpanSummaryHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(g.cfg.SummaryTimeout))
	defer cancel()

	span, ctx := opentracing.StartSpanFromContext(ctx, "Generator.SpanSummaryHandler")
	defer span.Finish()

	span.SetTag("requestURI", r.RequestURI)

	req, err := api.ParseSummaryRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resp *tempopb.SpanSummaryResponse
	resp, err = g.SpanSummary(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
}
