package metrics

import (
	"math"
	"slices"
)

// Kind says how a measurement may be combined.
type Kind string

const (
	// Counter means Total is a sum over the case, so two windows add.
	Counter Kind = "counter"
	// Gauge means Total is the value left behind when the case finished.
	// Adding two gauges is meaningless.
	Gauge Kind = "gauge"
)

// Measurement is one named number over a case.
//
// Every source reports this shape, whatever it came from, so a reader does not
// need a rule per source.
type Measurement struct {
	Kind  Kind    `json:"kind"`
	Total float64 `json:"total"`
	// Summary describes the per-execution values. A total on its own hides the
	// tail: the same 80ms over five executions is either an even 16ms each or
	// one 60ms outlier, and on the read path the outlier is the whole story.
	Summary Summary `json:"summary"`
}

// Set is every measurement for one case, keyed by "source.name".
type Set map[string]Measurement

// Summary describes a distribution. The quantiles are here so a box plot can be
// drawn from it without keeping every sample.
type Summary struct {
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	P25    float64 `json:"p25"`
	P50    float64 `json:"p50"`
	P75    float64 `json:"p75"`
	P90    float64 `json:"p90"`
	P99    float64 `json:"p99"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stdDev"`
}

// summarize reduces samples. Percentiles are nearest rank, so every reported
// value is one that was actually measured.
func summarize(samples []float64) Summary {
	if len(samples) == 0 {
		return Summary{}
	}

	sorted := slices.Clone(samples)
	slices.Sort(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	mean := sum / float64(len(sorted))

	var variance float64
	for _, v := range sorted {
		d := v - mean
		variance += d * d
	}

	return Summary{
		Count:  len(sorted),
		Min:    sorted[0],
		P25:    percentile(sorted, 0.25),
		P50:    percentile(sorted, 0.50),
		P75:    percentile(sorted, 0.75),
		P90:    percentile(sorted, 0.90),
		P99:    percentile(sorted, 0.99),
		Max:    sorted[len(sorted)-1],
		Mean:   mean,
		StdDev: math.Sqrt(variance / float64(len(sorted))),
	}
}

func percentile(sorted []float64, q float64) float64 {
	i := int(math.Ceil(q*float64(len(sorted)))) - 1
	return sorted[min(max(i, 0), len(sorted)-1)]
}
