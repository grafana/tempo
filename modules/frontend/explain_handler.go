package frontend

import (
	"encoding/json"
	"net/http"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/traceql"
)

// newExplainHTTPHandler returns an HTTP handler for GET /api/explain.
// It parses the TraceQL query, builds the logical plan, and returns it as JSON.
// No query execution occurs — this is a pure plan-building operation.
func newExplainHTTPHandler(logger log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q, _, _, err := api.ParseExplainRequest(r)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		expr, err := traceql.Parse(q)
		if err != nil {
			level.Warn(logger).Log("msg", "failed to parse TraceQL query for explain", "err", err)
			http.Error(w, `{"error":"failed to parse query: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		var plan traceql.PlanNode
		if expr.MetricsPipeline != nil {
			plan, err = traceql.BuildPlan(expr, nil)
		} else {
			plan, err = traceql.BuildSearchTracePlan(expr)
		}
		if err != nil {
			level.Error(logger).Log("msg", "failed to build explain plan", "err", err)
			http.Error(w, `{"error":"failed to build plan: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		resp := api.ExplainResponse{
			Plan: api.PlanNodeToExplainNode(plan),
		}

		w.Header().Set("Content-Type", "application/json")
		if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
			level.Error(logger).Log("msg", "failed to encode explain response", "err", encErr)
		}
	})
}
