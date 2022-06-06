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

var stringerAggs = map[AggregateOp]string{
	aggregateCount: "count",
	aggregateMax:   "max",
	aggregateMin:   "min",
	aggregateSum:   "sum",
	aggregateAvg:   "avg",
}

func (a AggregateOp) String() string {
	s, ok := stringerAggs[a]
	if ok {
		return s
	}
	return fmt.Sprintf("aggregate(%d)", a)
}
