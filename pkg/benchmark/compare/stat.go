package compare

import (
	"fmt"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// Stat is a percentile a summary shows of each case.
type Stat int

const (
	P50 Stat = iota
	P90
	P99
)

// Stats lists the percentiles a summary can show, in the order a view cycles
// through them.
var Stats = []Stat{P50, P90, P99}

func (s Stat) String() string {
	switch s {
	case P90:
		return "p90"
	case P99:
		return "p99"
	default:
		return "p50"
	}
}

// ParseStat reads a percentile by the name Stat.String gives it.
func ParseStat(name string) (Stat, error) {
	for _, s := range Stats {
		if s.String() == name {
			return s, nil
		}
	}
	return P50, fmt.Errorf("unknown percentile %q, want one of p50, p90, p99", name)
}

func (s Stat) of(sum *metrics.Summary) float64 {
	switch s {
	case P90:
		return sum.P90
	case P99:
		return sum.P99
	default:
		return sum.P50
	}
}

// Changes under MinorChange percent either way are too small to call out, and
// changes of MajorChange percent or more are what a summary is for finding.
// These are for the eye, not a test of significance.
const (
	MinorChange = 2.0
	MajorChange = 10.0
)
