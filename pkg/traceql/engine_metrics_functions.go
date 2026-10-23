package traceql

import "math"

// accumulator folds the values that land in one time series cell. The zero
// value is empty, and value() is NaN until something is added.
//
// add folds v with the given weight. That is a single span value with its
// sampling extrapolation, or a partial result from another querier with its
// weight (1 for sum/min/max, the span count for avg). NaN values are missing
// attributes and are ignored.
type accumulator interface {
	add(v, weight float64)
	value() float64
}

var (
	_ accumulator = (*sumAcc)(nil)
	_ accumulator = (*minAcc)(nil)
	_ accumulator = (*maxAcc)(nil)
	_ accumulator = (*avgAcc)(nil)
)

// newAccumulators returns n empty accumulators for op, backed by a single allocation.
func newAccumulators(op SimpleAggregationOp, n int) []accumulator {
	switch op {
	case minOverTimeAggregation:
		return makeAccumulators[minAcc](n)
	case maxOverTimeAggregation:
		return makeAccumulators[maxAcc](n)
	case sumOverTimeAggregation:
		return makeAccumulators[sumAcc](n)
	}
	return nil
}

func makeAccumulators[T any, P interface {
	*T
	accumulator
}](n int) []accumulator {
	backing := make([]T, n)
	cells := make([]accumulator, n)
	for i := range backing {
		cells[i] = P(&backing[i])
	}
	return cells
}

// sumAcc is a Kahan-Neumaier compensated sum of v*weight.
type sumAcc struct {
	sum  float64
	comp float64 // lost low-order bits, added back in value()
	ok   bool
}

func (a *sumAcc) add(v, weight float64) {
	if math.IsNaN(v) {
		return
	}
	v *= weight
	if !a.ok {
		a.sum, a.comp, a.ok = v, 0, true
		return
	}
	a.sum, a.comp = kahanSumInc(v, a.sum, a.comp)
}

func (a *sumAcc) value() float64 {
	if !a.ok {
		return nan
	}
	return a.sum + a.comp
}

// minAcc ignores weights, since sampling doesn't scale extremes.
type minAcc struct {
	v  float64
	ok bool
}

func (a *minAcc) add(v, _ float64) {
	if math.IsNaN(v) {
		return
	}
	if !a.ok || v < a.v {
		a.v, a.ok = v, true
	}
}

func (a *minAcc) value() float64 {
	if !a.ok {
		return nan
	}
	return a.v
}

// maxAcc ignores weights, since sampling doesn't scale extremes.
type maxAcc struct {
	v  float64
	ok bool
}

func (a *maxAcc) add(v, _ float64) {
	if math.IsNaN(v) {
		return
	}
	if !a.ok || v > a.v {
		a.v, a.ok = v, true
	}
}

func (a *maxAcc) value() float64 {
	if !a.ok {
		return nan
	}
	return a.v
}

// avgAcc is a weighted mean kept as a compensated sum and a total weight, so
// the mean is divided once at the end. Partial results are folded in as
// add(mean, count).
type avgAcc struct {
	sum    sumAcc
	weight float64
}

func (a *avgAcc) add(v, weight float64) {
	if math.IsNaN(v) {
		return
	}
	a.sum.add(v, weight)
	a.weight += weight
}

func (a *avgAcc) value() float64 {
	if !a.sum.ok {
		return nan
	}
	return a.sum.value() / a.weight
}

// count is the total weight, NaN if empty.
func (a *avgAcc) count() float64 {
	if !a.sum.ok {
		return nan
	}
	return a.weight
}
