package compare

import (
	"math"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark"
)

// Summary is one stat of one metric for every case: each run's value, and its
// change from the baseline's.
type Summary struct {
	Metric   string
	Stat     Stat
	Unit     Unit
	Baseline int
	// Rows holds a row per case, in case order.
	Rows []SummaryRow
}

// SummaryRow is one case, with a cell per run in run order, the baseline's
// among them.
type SummaryRow struct {
	// Case indexes Comparison.Cases.
	Case  int
	Cells []SummaryCell
}

// SummaryCell is one run's value of the stat for a case, and its change from
// the baseline's.
type SummaryCell struct {
	// Value is set when HasValue is: not for a run without data, or one that
	// cannot be compared.
	Value    float64
	HasValue bool
	// Delta is the change from the baseline in percent, set when HasDelta is:
	// not for the baseline itself, nor when the baseline has no value or a
	// value of zero.
	Delta    float64
	HasDelta bool
	// Incomparable says why the run cannot be compared with the baseline, when
	// it cannot.
	Incomparable string
}

// Summary lines up one stat of a metric for every case against the baseline.
func (c *Comparison) Summary(metric string, stat Stat, baseline int) Summary {
	sm := Summary{Metric: metric, Stat: stat, Unit: UnitFor(metric), Baseline: baseline}
	for i, cs := range c.Cases {
		s := cs.Series(metric)
		row := SummaryRow{Case: i, Cells: make([]SummaryCell, len(c.Runs))}
		for run := range c.Runs {
			row.Cells[run] = c.summaryCell(cs, s, stat, run, baseline)
		}
		sm.Rows = append(sm.Rows, row)
	}
	return sm
}

func (c *Comparison) summaryCell(cs Case, s Series, stat Stat, run, baseline int) SummaryCell {
	if run != baseline {
		if why := c.Incomparable(cs, run, baseline); why != "" {
			return SummaryCell{Incomparable: why}
		}
	}
	sum := s.Summaries[run]
	if sum == nil {
		return SummaryCell{}
	}
	cell := SummaryCell{Value: stat.Of(sum), HasValue: true}
	if base := s.Summaries[baseline]; run != baseline && base != nil {
		cell.Delta, cell.HasDelta = Delta(stat.Of(base), cell.Value)
	}
	return cell
}

// FamilyOf is the family a case rolls up into, as benchmark.Family has it.
func FamilyOf(id string) string {
	return benchmark.Family(id)
}

// FamilyGlob is a glob over every case of a member's family, like
// search/ranked_generic/*/*, in the syntax --case takes.
func FamilyGlob(id string) string {
	f := FamilyOf(id)
	return f + strings.Repeat("/*", strings.Count(id[len(f):], "/"))
}

// FamilySummary sums up a family's cases at one stat the way benchstat sums up
// a comparison: each run's geometric mean, and its change from the baseline's.
type FamilySummary struct {
	Family string
	Glob   string
	// Cases indexes Comparison.Cases, in case order.
	Cases []int
	// Counted is how many cases the means are over: those every run has a
	// value above zero for and can be compared with the baseline on. A
	// geometric mean takes no zero, and a change between means is only fair
	// over the same cases.
	Counted int
	// Cells hold a cell per run, in run order, the baseline's among them.
	Cells []FamilyCell
}

// FamilyCell is one run's mean over a family, set when the family counted a
// case.
type FamilyCell struct {
	Value float64
	// Delta is the change between the means in percent, set when HasDelta is:
	// not for the baseline itself.
	Delta    float64
	HasDelta bool
	// Least and Most are the smallest and largest change of a counted case.
	Least, Most float64
}

// Families sums up each family of the summary's cases, in the order of its
// first case.
func (c *Comparison) Families(sm Summary) []FamilySummary {
	var out []FamilySummary
	index := map[string]int{}
	var counted [][][]SummaryCell
	for _, r := range sm.Rows {
		f := FamilyOf(c.Cases[r.Case].ID)
		if f == "" {
			continue
		}
		i, ok := index[f]
		if !ok {
			i = len(out)
			index[f] = i
			out = append(out, FamilySummary{Family: f, Glob: FamilyGlob(c.Cases[r.Case].ID)})
			counted = append(counted, nil)
		}
		out[i].Cases = append(out[i].Cases, r.Case)
		if countable(r.Cells) {
			counted[i] = append(counted[i], r.Cells)
		}
	}

	for i := range out {
		out[i].Counted = len(counted[i])
		out[i].Cells = familyCells(counted[i], len(c.Runs), sm.Baseline)
	}
	return out
}

func countable(cells []SummaryCell) bool {
	for _, cell := range cells {
		if cell.Incomparable != "" || !cell.HasValue || cell.Value <= 0 {
			return false
		}
	}
	return true
}

func familyCells(rows [][]SummaryCell, runs, baseline int) []FamilyCell {
	cells := make([]FamilyCell, runs)
	if len(rows) == 0 {
		return cells
	}
	for run := range cells {
		var logSum float64
		cell := FamilyCell{Least: math.Inf(1), Most: math.Inf(-1)}
		for _, r := range rows {
			logSum += math.Log(r[run].Value)
			cell.Least = min(cell.Least, r[run].Delta)
			cell.Most = max(cell.Most, r[run].Delta)
		}
		cell.Value = math.Exp(logSum / float64(len(rows)))
		cells[run] = cell
	}
	for run := range cells {
		if run == baseline {
			cells[run].Least, cells[run].Most = 0, 0
			continue
		}
		cells[run].Delta, cells[run].HasDelta = Delta(cells[baseline].Value, cells[run].Value)
	}
	return cells
}
