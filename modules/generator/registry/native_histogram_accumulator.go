package registry

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	promhistogram "github.com/prometheus/prometheus/model/histogram"
)

const (
	nativeAccumulatorSchemaMaximum = 8
	nativeAccumulatorSchemaMinimum = -4
)

var nativeAccumulatorBounds = newNativeAccumulatorBounds()

type nativeHistogramAccumulator struct {
	sparse               bool
	schema               int32
	initialSchema        int32
	zeroThreshold        float64
	initialZeroThreshold float64
	maxBuckets           uint32
	minResetDuration     time.Duration
	lastResetTime        time.Time
	resetScheduled       bool
	count                uint64
	sum                  float64
	zeroCount            uint64

	positiveBucketCounts map[int]int64
	negativeBucketCounts map[int]int64

	positiveKeys   []int
	negativeKeys   []int
	positiveSpans  []promhistogram.Span
	negativeSpans  []promhistogram.Span
	positiveDeltas []int64
	negativeDeltas []int64

	exemplars []nativeHistogramAccumulatorExemplar
}

type nativeHistogramAccumulatorExemplar struct {
	labelName  string
	labelValue string
	value      float64
	timestamp  time.Time
}

func useNativeHistogramAccumulator(hasClassic bool, bucketFactor float64, _ uint32) bool {
	return !hasClassic && !math.IsNaN(bucketFactor)
}

func newNativeHistogramAccumulator(bucketFactor float64, maxBuckets uint32, minResetDuration time.Duration) *nativeHistogramAccumulator {
	a := &nativeHistogramAccumulator{
		maxBuckets:       maxBuckets,
		minResetDuration: minResetDuration,
		lastResetTime:    time.Now(),
		exemplars:        make([]nativeHistogramAccumulatorExemplar, 0, 10),
	}
	if bucketFactor <= 1 {
		return a
	}

	a.sparse = true
	a.initialSchema = pickNativeAccumulatorSchema(bucketFactor)
	a.schema = a.initialSchema
	a.initialZeroThreshold = prometheus.DefNativeHistogramZeroThreshold
	a.zeroThreshold = a.initialZeroThreshold
	a.positiveBucketCounts = make(map[int]int64)
	a.negativeBucketCounts = make(map[int]int64)
	return a
}

func (a *nativeHistogramAccumulator) observe(value float64, traceID string, multiplier float64, traceIDLabelName string) {
	if multiplier == 1 {
		a.observeOne(value, traceID, traceIDLabelName)
		return
	}

	for i := 0.0; i < multiplier; i++ {
		a.observeOne(value, traceID, traceIDLabelName)
	}
}

func (a *nativeHistogramAccumulator) observeOne(value float64, traceID string, traceIDLabelName string) {
	a.maybeRunScheduledReset()

	a.count++
	a.sum += value

	if !a.sparse || math.IsNaN(value) {
		return
	}

	key, isPositive, isZero := a.bucketKey(value)
	switch {
	case isZero:
		a.zeroCount++
	case isPositive:
		a.positiveBucketCounts[key]++
	default:
		a.negativeBucketCounts[key]++
	}
	a.limitBuckets(value)
	a.addExemplar(value, traceID, traceIDLabelName)
}

func (a *nativeHistogramAccumulator) bucketKey(value float64) (int, bool, bool) {
	absValue := math.Abs(value)
	isInf := false
	if math.IsInf(value, 0) {
		absValue = math.MaxFloat64
		isInf = true
	}

	frac, exp := math.Frexp(absValue)
	var key int
	if a.schema > 0 {
		bounds := nativeAccumulatorBounds[a.schema]
		key = sort.SearchFloat64s(bounds, frac) + (exp-1)*len(bounds)
	} else {
		key = exp
		if frac == 0.5 {
			key--
		}
		offset := (1 << -a.schema) - 1
		key = (key + offset) >> -a.schema
	}
	if isInf {
		key++
	}

	switch {
	case value > a.zeroThreshold:
		return key, true, false
	case value < -a.zeroThreshold:
		return key, false, false
	default:
		return 0, false, true
	}
}

func (a *nativeHistogramAccumulator) addExemplar(value float64, traceID string, traceIDLabelName string) {
	ex := nativeHistogramAccumulatorExemplar{
		labelName:  traceIDLabelName,
		labelValue: traceID,
		value:      value,
		timestamp:  time.Now(),
	}

	if len(a.exemplars) < cap(a.exemplars) {
		insertAt := sort.Search(len(a.exemplars), func(i int) bool {
			return value < a.exemplars[i].value
		})
		a.exemplars = append(a.exemplars, nativeHistogramAccumulatorExemplar{})
		copy(a.exemplars[insertAt+1:], a.exemplars[insertAt:])
		a.exemplars[insertAt] = ex
		return
	}

	if len(a.exemplars) == 1 {
		a.exemplars[0] = ex
		return
	}

	a.replaceExemplar(ex)
}

func (a *nativeHistogramAccumulator) replaceExemplar(ex nativeHistogramAccumulatorExemplar) {
	var (
		oldestTime  time.Time
		oldestIndex = -1
		minDelta    = -1.0
		newIndex    = -1
		replaceIdx  = -1
		currentLog  float64
		previousLog float64
	)

	for i, exemplar := range a.exemplars {
		if oldestIndex == -1 || exemplar.timestamp.Before(oldestTime) {
			oldestTime = exemplar.timestamp
			oldestIndex = i
		}
		if newIndex == -1 && ex.value <= exemplar.value {
			newIndex = i
		}

		previousLog = currentLog
		currentLog = math.Log(exemplar.value)
		if i == 0 {
			continue
		}
		delta := math.Abs(currentLog - previousLog)
		if minDelta == -1 || delta < minDelta {
			minDelta = delta
			if a.exemplars[i].timestamp.Before(a.exemplars[i-1].timestamp) {
				replaceIdx = i
			} else {
				replaceIdx = i - 1
			}
		}
	}

	if newIndex == -1 {
		newIndex = len(a.exemplars)
	}
	if oldestIndex != -1 && ex.timestamp.Sub(oldestTime) > 5*time.Minute {
		replaceIdx = oldestIndex
	} else {
		exemplarLog := math.Log(ex.value)
		if newIndex > 0 {
			delta := math.Abs(exemplarLog - math.Log(a.exemplars[newIndex-1].value))
			if delta < minDelta {
				minDelta = delta
				replaceIdx = newIndex - 1
			}
		}
		if newIndex < len(a.exemplars) {
			delta := math.Abs(math.Log(a.exemplars[newIndex].value) - exemplarLog)
			if delta < minDelta {
				replaceIdx = newIndex
			}
		}
	}

	switch {
	case replaceIdx == newIndex:
		a.exemplars[newIndex] = ex
	case replaceIdx < newIndex:
		copy(a.exemplars[replaceIdx:], a.exemplars[replaceIdx+1:newIndex])
		a.exemplars[newIndex-1] = ex
	case replaceIdx > newIndex:
		copy(a.exemplars[newIndex+1:], a.exemplars[newIndex:replaceIdx])
		a.exemplars[newIndex] = ex
	}
}

func (a *nativeHistogramAccumulator) histogram() *promhistogram.Histogram {
	a.maybeRunScheduledReset()

	a.negativeSpans, a.negativeDeltas, a.negativeKeys = nativeAccumulatorBuckets(
		a.negativeBucketCounts,
		a.negativeKeys,
		a.negativeSpans,
		a.negativeDeltas,
	)
	a.positiveSpans, a.positiveDeltas, a.positiveKeys = nativeAccumulatorBuckets(
		a.positiveBucketCounts,
		a.positiveKeys,
		a.positiveSpans,
		a.positiveDeltas,
	)

	return &promhistogram.Histogram{
		Schema:          a.schema,
		Count:           a.count,
		Sum:             a.sum,
		ZeroThreshold:   a.zeroThreshold,
		ZeroCount:       a.zeroCount,
		NegativeSpans:   a.negativeSpans,
		NegativeBuckets: a.negativeDeltas,
		PositiveSpans:   a.positiveSpans,
		PositiveBuckets: a.positiveDeltas,
	}
}

func (a *nativeHistogramAccumulator) limitBuckets(value float64) {
	if a.maxBuckets == 0 || uint32(a.populatedBuckets()) <= a.maxBuckets {
		return
	}

	now := time.Now()
	if a.minResetDuration > 0 && !a.resetScheduled && now.Sub(a.lastResetTime) >= a.minResetDuration {
		a.reset()
		a.count = 1
		a.sum = value
		if !math.IsNaN(value) {
			key, isPositive, isZero := a.bucketKey(value)
			switch {
			case isZero:
				a.zeroCount = 1
			case isPositive:
				a.positiveBucketCounts[key] = 1
			default:
				a.negativeBucketCounts[key] = 1
			}
		}
		a.lastResetTime = now
		a.resetScheduled = false
		return
	}

	if a.minResetDuration > 0 && !a.resetScheduled {
		a.resetScheduled = true
	}
	a.doubleBucketWidth()
}

func (a *nativeHistogramAccumulator) maybeRunScheduledReset() {
	if !a.resetScheduled || a.minResetDuration == 0 || time.Since(a.lastResetTime) < a.minResetDuration {
		return
	}
	a.reset()
	a.lastResetTime = time.Now()
	a.resetScheduled = false
}

func (a *nativeHistogramAccumulator) reset() {
	a.schema = a.initialSchema
	a.zeroThreshold = a.initialZeroThreshold
	a.count = 0
	a.sum = 0
	a.zeroCount = 0
	clear(a.positiveBucketCounts)
	clear(a.negativeBucketCounts)
}

func (a *nativeHistogramAccumulator) populatedBuckets() int {
	return len(a.positiveBucketCounts) + len(a.negativeBucketCounts)
}

func (a *nativeHistogramAccumulator) doubleBucketWidth() {
	if a.schema == nativeAccumulatorSchemaMinimum {
		return
	}
	a.schema--
	a.mergeBucketsForLowerSchema(a.positiveBucketCounts)
	a.mergeBucketsForLowerSchema(a.negativeBucketCounts)
}

func (a *nativeHistogramAccumulator) mergeBucketsForLowerSchema(buckets map[int]int64) {
	if len(buckets) == 0 {
		return
	}

	mergedBuckets := make(map[int]int64, len(buckets))
	for key, count := range buckets {
		mergedKey := key
		if mergedKey > 0 {
			mergedKey++
		}
		mergedKey /= 2
		mergedBuckets[mergedKey] += count
		delete(buckets, key)
	}
	for key, count := range mergedBuckets {
		buckets[key] = count
	}
}

func nativeAccumulatorBuckets(
	bucketCounts map[int]int64,
	keys []int,
	spans []promhistogram.Span,
	deltas []int64,
) ([]promhistogram.Span, []int64, []int) {
	keys = keys[:0]
	for key := range bucketCounts {
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return spans[:0], deltas[:0], keys
	}
	sort.Ints(keys)

	spans = spans[:0]
	deltas = deltas[:0]
	var (
		previousCount int64
		nextIndex     int
	)
	for n, key := range keys {
		count := bucketCounts[key]
		indexDelta := int32(key - nextIndex)
		if n == 0 || indexDelta > 2 {
			spans = append(spans, promhistogram.Span{
				Offset: indexDelta,
			})
		} else {
			for j := int32(0); j < indexDelta; j++ {
				spans[len(spans)-1].Length++
				deltas = append(deltas, -previousCount)
				previousCount = 0
			}
		}

		spans[len(spans)-1].Length++
		deltas = append(deltas, count-previousCount)
		previousCount = count
		nextIndex = key + 1
	}
	return spans, deltas, keys
}

func pickNativeAccumulatorSchema(bucketFactor float64) int32 {
	if bucketFactor <= 1 {
		panic(fmt.Errorf("bucketFactor %f is <=1", bucketFactor))
	}
	floor := math.Floor(math.Log2(math.Log2(bucketFactor)))
	switch {
	case floor <= -8:
		return nativeAccumulatorSchemaMaximum
	case floor >= 4:
		return nativeAccumulatorSchemaMinimum
	default:
		return -int32(floor)
	}
}

func newNativeAccumulatorBounds() [][]float64 {
	boundsBySchema := make([][]float64, nativeAccumulatorSchemaMaximum+1)
	numBuckets := 1
	for schema := range boundsBySchema {
		bounds := []float64{0.5}
		factor := math.Exp2(math.Exp2(float64(-schema)))
		for bucket := 0; bucket < numBuckets-1; bucket++ {
			var bound float64
			if (bucket+1)%2 == 0 {
				bound = boundsBySchema[schema-1][bucket/2+1]
			} else {
				bound = bounds[bucket] * factor
			}
			bounds = append(bounds, bound)
		}
		numBuckets *= 2
		boundsBySchema[schema] = bounds
	}
	return boundsBySchema
}
