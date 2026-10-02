package compare

import (
	"fmt"
	"slices"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// Stat is a point of a case's spread a summary shows: a percentile, or the
// least or greatest value measured.
type Stat int

const (
	Min Stat = iota
	P25
	P50
	P75
	P90
	P99
	Max
)

// Stats lists the stats a summary can show, from least to greatest.
var Stats = []Stat{Min, P25, P50, P75, P90, P99, Max}

var statNames = []string{"min", "p25", "p50", "p75", "p90", "p99", "max"}

func (s Stat) String() string {
	return statNames[s]
}

// ParseStat reads a stat by the name Stat.String gives it.
func ParseStat(name string) (Stat, error) {
	if i := slices.Index(statNames, name); i >= 0 {
		return Stat(i), nil
	}
	return 0, fmt.Errorf("unknown percentile %q, want one of %s", name, strings.Join(statNames, ", "))
}

// Of picks the stat out of a summary.
func (s Stat) Of(sum *metrics.Summary) float64 {
	switch s {
	case Min:
		return sum.Min
	case P25:
		return sum.P25
	case P50:
		return sum.P50
	case P75:
		return sum.P75
	case P90:
		return sum.P90
	case P99:
		return sum.P99
	default:
		return sum.Max
	}
}
