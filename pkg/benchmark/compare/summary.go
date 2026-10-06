package compare

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
