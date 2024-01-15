package traceql

import "fmt"

type AggregateOp int

const (
	aggregateCount AggregateOp = iota
	aggregateMax
	aggregateMin
	aggregateSum
	aggregateAvg
)

func (a AggregateOp) String() string {
	switch a {
	case aggregateCount:
		return "count"
	case aggregateMax:
		return "max"
	case aggregateMin:
		return "min"
	case aggregateSum:
		return "sum"
	case aggregateAvg:
		return "avg"
	}

	return fmt.Sprintf("aggregate(%d)", a)
}

type MetricsAggregateOp int

const (
	metricsAggregateRate MetricsAggregateOp = iota
	metricsAggregateCountOverTime
)

func (a MetricsAggregateOp) String() string {
	switch a {
	case metricsAggregateRate:
		return "rate"
	case metricsAggregateCountOverTime:
		return "count_over_time"
	}

	return fmt.Sprintf("aggregate(%d)", a)
}
