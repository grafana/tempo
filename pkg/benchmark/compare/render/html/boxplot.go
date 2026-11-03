package html

import (
	"math"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

const (
	// targetTicks is about how many intervals an axis is divided into.
	targetTicks = 5
	// maxReach is how far past the highest p99 the axis stretches to take in the
	// highest max. Past it the max is clipped, since one slow execution would
	// otherwise squash every box to make room for it.
	maxReach = 1.25
)

// boxPlot is a series as a box per run on a shared axis, in the series' own
// values: where each run's marks fall, for boxPlotSVG to draw. A box spans the
// p25 to the p75 with a mark at the median, and its whisker runs from the min
// to the p99 with a tick at the p90.
type boxPlot struct {
	Unit  compare.Unit
	Axis  axis
	Ticks []tick
	Rows  []boxPlotRow
}

// boxPlotRow is one run's box.
type boxPlotRow struct {
	// Name is the run's name, shown as the run, as its whole row is.
	Name render.Text
	// Summary is the run's, or nil for a run without data.
	Summary *metrics.Summary
	// Clipped says the max is past the axis, so it is marked at the axis's
	// edge and written out, as Max.
	Clipped bool
	Max     string
}

// newBoxPlot plots every run of a series on one axis.
func newBoxPlot(names []string, s compare.Series, baseline int) boxPlot {
	a := newAxis(s)
	v := boxPlot{Unit: s.Unit, Axis: a, Ticks: a.ticks(s.Unit)}
	for i, sum := range s.Summaries {
		row := boxPlotRow{Name: render.RunText(names[i], i, baseline), Summary: sum}
		if sum != nil && sum.Max > a.Max {
			row.Clipped, row.Max = true, render.Format(s.Unit, sum.Max)
		}
		v.Rows = append(v.Rows, row)
	}
	return v
}

// axis spans a series' values in round steps.
type axis struct {
	Min, Max float64
	// Step is the distance between ticks.
	Step float64
}

// tick is a value an axis marks, and how it is written.
type tick struct {
	Value float64
	Label string
}

// newAxis fits an axis to every run of the series. It spans the lowest min to
// the highest p99, rounded out to whole ticks, and reaches the highest max
// only when that is close.
func newAxis(s compare.Series) axis {
	lo, hi, top := math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, sum := range s.Summaries {
		if sum == nil {
			continue
		}
		lo = min(lo, sum.Min)
		hi = max(hi, sum.P99)
		top = max(top, sum.Max)
	}
	if math.IsInf(lo, 1) {
		lo, hi, top = 0, 1, 1
	}
	if top <= hi*maxReach {
		hi = top
	}
	if hi <= lo {
		// Every value is the same. Pad around it so it lands mid-axis.
		pad := math.Abs(lo) * 0.1
		if pad == 0 {
			pad = 1
		}
		if lo >= 0 {
			lo = max(0, lo-pad)
		} else {
			lo -= pad
		}
		hi += pad
	}

	step := niceStep((hi - lo) / targetTicks)
	return axis{Min: math.Floor(lo/step) * step, Max: math.Ceil(hi/step) * step, Step: step}
}

// ticks are the values the axis marks, a step apart from its min to its max.
func (a axis) ticks(u compare.Unit) []tick {
	n := int(math.Round((a.Max-a.Min)/a.Step)) + 1
	ticks := make([]tick, n)
	for i := range ticks {
		v := a.Min + float64(i)*a.Step
		ticks[i] = tick{Value: v, Label: render.Format(u, v)}
	}
	return ticks
}

// niceStep rounds a tick distance to 1, 2 or 5 times a power of ten, so tick
// labels read as round numbers.
func niceStep(raw float64) float64 {
	if raw <= 0 || math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	switch n := raw / mag; {
	case n < 1.5:
		return mag
	case n < 3:
		return 2 * mag
	case n < 7:
		return 5 * mag
	default:
		return 10 * mag
	}
}
