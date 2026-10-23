package traceql

import "math"

// accumulator folds the values that land in one interval of a time series.
// value() is NaN until something is added.
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

// Empty accumulators. NaN marks an accumulator nothing was added to.
var (
	emptySumAcc = sumAcc{sum: nan}
	emptyMinAcc = minAcc{v: nan}
	emptyMaxAcc = maxAcc{v: nan}
	emptyAvgAcc = avgAcc{sum: emptySumAcc}
)

// newAccumulators returns n empty accumulators for op, backed by a single allocation.
func newAccumulators(op SimpleAggregationOp, n int) []accumulator {
	switch op {
	case minOverTimeAggregation:
		return makeAccumulators(n, emptyMinAcc)
	case maxOverTimeAggregation:
		return makeAccumulators(n, emptyMaxAcc)
	case sumOverTimeAggregation:
		return makeAccumulators(n, emptySumAcc)
	}
	return nil
}

func makeAccumulators[T any, P interface {
	*T
	accumulator
}](n int, empty T) []accumulator {
	backing := make([]T, n)
	accs := make([]accumulator, n)
	for i := range backing {
		backing[i] = empty
		accs[i] = P(&backing[i])
	}
	return accs
}

// sumAcc is a Kahan-Neumaier compensated sum of v*weight. It is empty while
// sum is NaN and comp is 0; a sum that became NaN (Inf + -Inf) has a NaN comp,
// so it stays NaN instead of restarting.
type sumAcc struct {
	sum  float64
	comp float64 // lost low-order bits, added back in value()
}

func (a *sumAcc) empty() bool {
	return math.IsNaN(a.sum) && a.comp == 0
}

func (a *sumAcc) add(v, weight float64) {
	if math.IsNaN(v) {
		return
	}
	v *= weight
	if a.empty() {
		a.sum = v
		return
	}
	a.sum, a.comp = kahanSumInc(v, a.sum, a.comp)
}

func (a *sumAcc) value() float64 {
	return a.sum + a.comp
}

// minAcc ignores weights, since sampling doesn't scale extremes.
type minAcc struct {
	v float64
}

func (a *minAcc) add(v, _ float64) {
	if math.IsNaN(a.v) || v < a.v {
		a.v = v
	}
}

func (a *minAcc) value() float64 {
	return a.v
}

// maxAcc ignores weights, since sampling doesn't scale extremes.
type maxAcc struct {
	v float64
}

func (a *maxAcc) add(v, _ float64) {
	if math.IsNaN(a.v) || v > a.v {
		a.v = v
	}
}

func (a *maxAcc) value() float64 {
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
	return a.sum.value() / a.weight
}

// count is the total weight, NaN if empty.
func (a *avgAcc) count() float64 {
	if a.sum.empty() {
		return nan
	}
	return a.weight
}
