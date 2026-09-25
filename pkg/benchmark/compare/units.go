package compare

import (
	"math"
	"strconv"
	"strings"
)

// Unit says how a metric's values read.
type Unit int

const (
	Count Unit = iota
	Nanoseconds
	Seconds
	Bytes
)

// UnitFor infers a metric's unit from its key, since a result does not record
// one. Harness and backend keys end in Ns or Bytes; process keys follow the
// Prometheus convention of a _seconds or _bytes suffix.
func UnitFor(metric string) Unit {
	name := metric
	if _, n, ok := strings.Cut(metric, "."); ok {
		name = n
	}
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(name, "Ns"):
		return Nanoseconds
	case strings.HasSuffix(lower, "_seconds"), strings.HasSuffix(lower, "_seconds_sum"), strings.HasSuffix(lower, "_seconds_total"):
		return Seconds
	case strings.HasSuffix(lower, "bytes"), strings.HasSuffix(lower, "bytes_total"):
		return Bytes
	default:
		return Count
	}
}

var (
	durationSuffixes = []string{"ns", "µs", "ms", "s"}
	byteSuffixes     = []string{"B", "kB", "MB", "GB", "TB"}
	countSuffixes    = []string{"", "k", "M", "G", "T"}
)

// Format writes v in its unit to three significant digits, so a column of
// values stays narrow however large they get.
func Format(u Unit, v float64) string {
	switch u {
	case Nanoseconds:
		return scaled(v, durationSuffixes)
	case Seconds:
		return scaled(v*1e9, durationSuffixes)
	case Bytes:
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
