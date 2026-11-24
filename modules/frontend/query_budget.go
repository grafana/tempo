package frontend

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/grafana/dskit/tenant"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/v3/modules/frontend/combiner"
	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
	"github.com/grafana/tempo/v3/modules/frontend/pipeline"
	"github.com/grafana/tempo/v3/modules/frontend/querybudget"
	"github.com/grafana/tempo/v3/pkg/api"
)

// queryBudgetAdmitWare rejects queries of tenants whose query budget is in debt. It runs before the multi-tenant
// fan-out, so that a query is admitted or rejected for all its tenants together.
type queryBudgetAdmitWare struct {
	budget             *querybudget.Budget
	op                 jobcost.Op
	multiTenantEnabled bool
	next               pipeline.AsyncRoundTripper[combiner.PipelineResponse]
}

func newQueryBudgetAdmitWare(budget *querybudget.Budget, op jobcost.Op, multiTenantEnabled bool) pipeline.AsyncMiddleware[combiner.PipelineResponse] {
	return pipeline.AsyncMiddlewareFunc[combiner.PipelineResponse](func(next pipeline.AsyncRoundTripper[combiner.PipelineResponse]) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
		return &queryBudgetAdmitWare{budget: budget, op: op, multiTenantEnabled: multiTenantEnabled, next: next}
	})
}

func (w *queryBudgetAdmitWare) RoundTrip(req pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
	ctx := req.Context()
	tenants, err := tenant.TenantIDs(ctx)
	if err != nil || (len(tenants) > 1 && !w.multiTenantEnabled) {
		// the tenant validation and the multi-tenant middleware further down answer these
		return w.next.RoundTrip(req)
	}

	d := pipeline.AdmitQueryBudget(ctx, func() querybudget.Decision {
		return w.budget.Admit(tenants, w.op.String())
	})
	switch {
	case d.Rejected:
		trace.SpanFromContext(ctx).SetAttributes(attribute.Bool("query_budget.rejected", true))
		return pipeline.NewHTTPToAsyncResponse(queryBudgetRejectedResponse(d)), nil
	case d.DryRunRejected:
		trace.SpanFromContext(ctx).SetAttributes(attribute.Bool("query_budget.dry_run_rejected", true))
	}

	return w.next.RoundTrip(req)
}

// queryBudgetChargeWare charges every job sent to the queriers to the query budget of its tenant. It is the
// innermost sync middleware, so cache hits are not charged and every retry is.
type queryBudgetChargeWare struct {
	op   jobcost.Op
	next pipeline.RoundTripper
}

func newQueryBudgetChargeWare(op jobcost.Op) pipeline.Middleware {
	return pipeline.MiddlewareFunc(func(next pipeline.RoundTripper) pipeline.RoundTripper {
		return &queryBudgetChargeWare{op: op, next: next}
	})
}

func (w *queryBudgetChargeWare) RoundTrip(req pipeline.Request) (*http.Response, error) {
	ctx := req.Context()
	cost := req.Cost()
	query := pipeline.QueryBudgetFromContext(ctx)
	if query == nil || cost.Estimate == 0 {
		return w.next.RoundTrip(req)
	}
	tenantID, err := tenant.TenantID(ctx)
	if err != nil {
		return w.next.RoundTrip(req)
	}

	op := w.op.String()
	query.Charge(tenantID, op, cost)

	ctx, dequeued := pipeline.WithDequeueMarker(ctx)
	req.SetContext(ctx)

	// a job that failed before a querier took it, because the queue was full or its query was canceled, did
	// not run. a job that ran keeps its estimate if it failed or its query was canceled, as the bytes it read
	// are not reported
	resp, err := w.next.RoundTrip(req)
	if err != nil && !dequeued.Load() {
		query.Refund(tenantID, op, cost)
	}
	return resp, err
}

// queryBudgetRejectedResponse is the response to a query that the query budget rejected.
func queryBudgetRejectedResponse(d querybudget.Decision) *http.Response {
	retryAfter := strconv.Itoa(int(math.Ceil(d.RetryAfter.Seconds())))
	return &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     http.StatusText(http.StatusTooManyRequests),
		Header:     http.Header{api.HeaderRetryAfter: {retryAfter}},
		Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("query budget used up, retry after %s seconds", retryAfter))),
	}
}
