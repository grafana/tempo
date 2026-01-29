package spanmetrics

import (
	"strings"

	"github.com/grafana/tempo/modules/generator/processor"
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
		return processor.SpanMetricsLatencyName
	case Count:
		return processor.SpanMetricsCountName
	case Size:
		return processor.SpanMetricsSizeName
	default:
		return "unsupported"
	}
}

func ParseSubprocessor(s string) bool {
	for _, p := range SupportedSubprocessors {
		if strings.EqualFold(p.String(), s) {
			return true
		}
	}
	return false
}
