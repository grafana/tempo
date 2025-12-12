package drain

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	FormatLogfmt  = "logfmt"
	FormatJSON    = "json"
	FormatUnknown = "unknown"
	TooFewTokens  = "too_few_tokens"
	TooManyTokens = "too_many_tokens"
	LineTooLong   = "line_too_long"
)

type Metrics struct {
	PatternsEvictedTotal  prometheus.Counter
	PatternsPrunedTotal   prometheus.Counter
	PatternsDetectedTotal prometheus.Counter
	LinesSkipped          *prometheus.CounterVec
	TokensPerLine         prometheus.Observer
	StatePerLine          prometheus.Observer
}
