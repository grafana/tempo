package frontend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/dskit/tenant"

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

	d := pipeline.AdmitQueryBudgetOnce(ctx, func() querybudget.Decision {
		return w.budget.Admit(tenants, w.op)
	})
	if d.Outcome == querybudget.OutcomeRejected {
		return pipeline.NewHTTPToAsyncResponse(queryBudgetRejectedResponse(d.RetryAfter)), nil
	}
	return w.next.RoundTrip(req)
}

// queryBudgetChargeWare charges every job sent to the queriers to the query budget of its tenant. It is the
// innermost sync middleware, so cache hits are not charged and every retry is.
type queryBudgetChargeWare struct {
	next pipeline.RoundTripper
}

func newQueryBudgetChargeWare() pipeline.Middleware {
	return pipeline.MiddlewareFunc(func(next pipeline.RoundTripper) pipeline.RoundTripper {
		return &queryBudgetChargeWare{next: next}
	})
}

func (w *queryBudgetChargeWare) RoundTrip(req pipeline.Request) (*http.Response, error) {
	ctx := req.Context()
	cost := req.Cost()
	budgetQuery := pipeline.BudgetQueryFromContext(ctx)
	if budgetQuery == nil || cost.Estimate == 0 {
		return w.next.RoundTrip(req)
	}
	tenantID, err := tenant.TenantID(ctx)
	if err != nil {
		return w.next.RoundTrip(req)
	}

	budgetQuery.Charge(tenantID, cost)

	ctx, dequeued := pipeline.WithDequeueMarker(ctx)
	req.SetContext(ctx)

	// a job that failed before a querier took it, because the queue was full or its query was canceled, did
	// not run. a job that ran keeps its estimate if it failed or its query was canceled, as the bytes it read
	// are not reported
	resp, err := w.next.RoundTrip(req)
	if err != nil && !dequeued.Load() {
		budgetQuery.Refund(tenantID, cost)
	}
	return resp, err
}

// queryBudgetSettleOption returns a combiner option that settles the completed jobs of the query whose cell is on
// ctx in its query budget, and does nothing else.
func queryBudgetSettleOption(ctx context.Context) combiner.Option {
	return combiner.WithJobInspectedBytesObserver(func(jobTenant string, inspectedBytes uint64, cost jobcost.Cost) {
		// looked up for every job, because the query is admitted after the combiner is created
		pipeline.BudgetQueryFromContext(ctx).Settle(jobTenant, cost, inspectedBytes)
	})
}

// queryBudgetRejectedResponse is the response to a query that the query budget rejected.
func queryBudgetRejectedResponse(retryAfter time.Duration) *http.Response {
	seconds := retryAfterSeconds(retryAfter)
	return &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     http.StatusText(http.StatusTooManyRequests),
		Header:     http.Header{api.HeaderRetryAfter: {seconds}},
		Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("query budget used up, retry after %s seconds", seconds))),
	}
}

// retryAfterSeconds returns the value of a Retry-After header for d, in whole seconds rounded up.
func retryAfterSeconds(d time.Duration) string {
	return strconv.FormatInt(int64(ceilSeconds(d)/time.Second), 10)
}
