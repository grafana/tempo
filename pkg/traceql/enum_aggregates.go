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
