package pipeline

import (
	"context"

	"github.com/grafana/tempo/v3/modules/frontend/querybudget"
)

// AdmitQueryBudgetOnce returns the decision of admit for the query whose cell is installed on ctx. admit only
// runs for the first call, later calls of the same query get the same decision, for example the second fetch of
// a trace diff. Without a cell admit runs every time and the jobs of the query are not charged.
func AdmitQueryBudgetOnce(ctx context.Context, admit func() querybudget.Decision) querybudget.Decision {
	cell := queryShapeCellFromContext(ctx)
	if cell == nil {
		return admit()
	}

	cell.budgetOnce.Do(func() {
		d := admit()
		cell.budgetDecision.Store(&d)
	})
	return *cell.budgetDecision.Load()
}

// BudgetQueryFromContext returns the Query that charges the jobs of the query whose cell is installed on ctx. It
// is nil if the query budget didn't admit the query or none of its tenants has a query budget.
func BudgetQueryFromContext(ctx context.Context) *querybudget.Query {
	return QueryBudgetDecisionFromContext(ctx).Query
}

// QueryBudgetDecisionFromContext returns the decision of the query budget for the query whose cell is installed
// on ctx. It is the zero Decision if there is no cell or the query budget didn't decide yet.
func QueryBudgetDecisionFromContext(ctx context.Context) querybudget.Decision {
	if cell := queryShapeCellFromContext(ctx); cell != nil {
		if d := cell.budgetDecision.Load(); d != nil {
			return *d
		}
	}
	return querybudget.Decision{}
}
