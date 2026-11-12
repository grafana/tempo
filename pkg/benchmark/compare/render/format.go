package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

const (
	// NoValue stands for a run without data.
	NoValue = "–"
	// NotApplicable stands for a change from a baseline of zero, which no
	// percentage can say.
	NotApplicable = "n/a"
)

// Changes under MinorChange percent either way are too small to call out, and
// changes of MajorChange percent or more are what a summary is for finding.
// These are for the eye, not a test of significance.
const (
	MinorChange = 2.0
	MajorChange = 10.0
)

var (
	durationSuffixes = []string{"ns", "µs", "ms", "s"}
	byteSuffixes     = []string{"B", "kB", "MB", "GB", "TB"}
	countSuffixes    = []string{"", "k", "M", "G", "T"}
)

// Format writes v in its unit to three significant digits, so a column of
// values stays narrow however large they get.
func Format(u compare.Unit, v float64) string {
	switch u {
	case compare.Nanoseconds:
		return scaled(v, durationSuffixes)
	case compare.Seconds:
		return scaled(v*1e9, durationSuffixes)
	case compare.Bytes:
		return scaled(v, byteSuffixes)
	default:
		return scaled(v, countSuffixes)
	}
}

// scaled divides v by 1000 until it reads in the largest suffix that keeps it
// at least one, then rounds it to three significant digits.
func scaled(v float64, suffixes []string) string {
	if v == 0 {
		return "0" + suffixes[0]
	}
	i := 0
	for i < len(suffixes)-1 && math.Abs(roundSig(v, 3)) >= 1000 {
		v /= 1000
		i++
	}
	v = roundSig(v, 3)

	decimals := max(0, 3-int(math.Ceil(math.Log10(math.Abs(v)))))
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s + suffixes[i]
}

func roundSig(v float64, digits int) float64 {
	if v == 0 {
		return 0
	}
	scale := math.Pow(10, float64(digits)-math.Ceil(math.Log10(math.Abs(v))))
	return math.Round(v*scale) / scale
}

// FormatChange writes a change to a tenth of a percent, and no change as 0%
// rather than a signed zero.
func FormatChange(pct float64) string {
	if pct == 0 {
		return "0%"
	}
	return fmt.Sprintf("%+.1f%%", pct)
}
