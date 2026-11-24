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
		cell.budgetDecision = admit()
		cell.budgetQuery.Store(cell.budgetDecision.Query)
	})
	return cell.budgetDecision
}

// BudgetQueryFromContext returns the Query that charges the jobs of the query whose cell is installed on ctx. It
// is nil if the query budget didn't admit the query or none of its tenants has a query budget.
func BudgetQueryFromContext(ctx context.Context) *querybudget.Query {
	if cell := queryShapeCellFromContext(ctx); cell != nil {
		return cell.budgetQuery.Load()
	}
	return nil
}
