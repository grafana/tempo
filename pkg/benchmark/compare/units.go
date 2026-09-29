package compare

import "strings"

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
