package render

import "github.com/grafana/tempo/v3/pkg/benchmark/compare"

// CaseView is one case of one metric: every run's box plot, over a table of
// their numbers. Summaries are per execution: one trace lookup, or one shard
// of a search.
type CaseView struct {
	Title string
	Query string
	// Problems say why runs cannot be compared with the baseline on the case.
	Problems []string
	Plot     BoxPlotView
	Table    TableView
}

// NewCase shows one metric of a case, the one Comparison.Cases holds at i.
func NewCase(c *compare.Comparison, i int, metric string, baseline int) CaseView {
	cs := c.Cases[i]
	names := c.Names()
	s := cs.Series(metric)
	return CaseView{
		Title:    cs.ID + " · " + metric + " · per execution",
		Query:    cs.Query,
		Problems: Problems(c, cs, baseline),
		Plot:     NewBoxPlot(names, s, baseline),
		Table:    NewTable(names, s, baseline),
	}
}
