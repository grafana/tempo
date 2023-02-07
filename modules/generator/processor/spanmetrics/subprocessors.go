package spanmetrics

import (
    "strings"
)

type Subprocessor int

const (
	Latency Subprocessor = iota
	Count
	Size
)

var SupportedSubprocessors = []Subprocessor{
    Latency,
    Count,
    Size,
}

func (s Subprocessor) String() string {
	switch s {
	case Latency:
		return "span-metrics-latency"
	case Count:
		return "span-metrics-count"
	case Size:
		return "span-metrics-size"
	default:
		return "unsupported"
	}
}

func ParseSubprocessor(s string) (bool) {
	for _, p := range SupportedSubprocessors {
		if strings.EqualFold(p.String(), s) {
			return true
		}
	}
	return false
}
