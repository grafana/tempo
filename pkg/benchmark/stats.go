package benchmark

import (
	"math"
	"slices"
)

// Summary describes a distribution of per-execution latencies in nanoseconds.
type Summary struct {
	Count  int     `json:"count"`
	Min    int64   `json:"min"`
	P50    int64   `json:"p50"`
	P90    int64   `json:"p90"`
	P99    int64   `json:"p99"`
	Max    int64   `json:"max"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stdDev"`
}

// summarize sorts a copy of samples and reduces it. Percentiles are nearest
// rank, so every reported value is one that was actually measured.
func summarize(samples []int64) Summary {
	if len(samples) == 0 {
		return Summary{}
	}

	sorted := slices.Clone(samples)
	slices.Sort(sorted)

	var sum float64
	for _, v := range sorted {
		sum += float64(v)
	}
	mean := sum / float64(len(sorted))

	var variance float64
	for _, v := range sorted {
		d := float64(v) - mean
		variance += d * d
	}

	return Summary{
		Count:  len(sorted),
		Min:    sorted[0],
		P50:    percentile(sorted, 0.50),
		P90:    percentile(sorted, 0.90),
		P99:    percentile(sorted, 0.99),
		Max:    sorted[len(sorted)-1],
		Mean:   mean,
		StdDev: math.Sqrt(variance / float64(len(sorted))),
	}
}

func percentile(sorted []int64, q float64) int64 {
	i := int(math.Ceil(q*float64(len(sorted)))) - 1
	return sorted[min(max(i, 0), len(sorted)-1)]
}
