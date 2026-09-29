package benchmark

import (
	"cmp"
	"math"
	"slices"
)

// histogramBucketsPerDoubling makes each bucket 2^(1/32), about 2.2%, wider
// than the one below it, so a quantile read from the histogram is that close
// to the true value however many spans it has seen.
const histogramBucketsPerDoubling = 32

// logHistogram counts numeric values in logarithmic buckets, keeping memory
// bounded by the range of the values rather than their number. Positive bucket
// i holds (bound(i-1), bound(i)], and negative bucket i holds magnitudes in
// [bound(i-1), bound(i)), so as values every bucket's upper bound is
// inclusive and everything above it lies in later buckets.
type logHistogram struct {
	pos   map[int]uint64
	neg   map[int]uint64
	zero  uint64
	total uint64
}

func newLogHistogram() *logHistogram {
	return &logHistogram{pos: map[int]uint64{}, neg: map[int]uint64{}}
}

// add counts v n times.
func (h *logHistogram) add(v float64, n uint64) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return
	}
	h.total += n
	switch {
	case v > 0:
		h.pos[posBucket(v)] += n
	case v < 0:
		h.neg[negBucket(-v)] += n
	default:
		h.zero += n
	}
}

func bucketBound(i int) float64 {
	return math.Exp2(float64(i) / histogramBucketsPerDoubling)
}

// posBucket returns i with bound(i-1) < m <= bound(i). Log2 can put a value
// that sits on a bound in the neighbouring bucket, so the index is corrected
// against the bounds themselves.
func posBucket(m float64) int {
	i := int(math.Ceil(math.Log2(m) * histogramBucketsPerDoubling))
	for m > bucketBound(i) {
		i++
	}
	for m <= bucketBound(i-1) {
		i--
	}
	return i
}

// negBucket returns i with bound(i-1) <= m < bound(i).
func negBucket(m float64) int {
	i := int(math.Floor(math.Log2(m)*histogramBucketsPerDoubling)) + 1
	for m >= bucketBound(i) {
		i++
	}
	for m < bucketBound(i-1) {
		i--
	}
	return i
}

type histogramBucket struct {
	upper float64
	count uint64
}

// buckets returns the non-empty buckets in ascending value order.
func (h *logHistogram) buckets() []histogramBucket {
	out := make([]histogramBucket, 0, len(h.neg)+len(h.pos)+1)
	for i, n := range h.neg {
		out = append(out, histogramBucket{upper: -bucketBound(i - 1), count: n})
	}
	if h.zero > 0 {
		out = append(out, histogramBucket{upper: 0, count: h.zero})
	}
	for i, n := range h.pos {
		out = append(out, histogramBucket{upper: bucketBound(i), count: n})
	}
	slices.SortFunc(out, func(a, b histogramBucket) int { return cmp.Compare(a.upper, b.upper) })
	return out
}

type histogramQuantile struct {
	q, upper, selectivity float64
}

// quantiles returns, for each q in ascending order, the upper bound of the
// bucket holding that quantile, and the fraction of totalSpans above it. The
// bound is inclusive, so that fraction is a sum of later buckets: exact, not
// an estimate.
func (h *logHistogram) quantiles(qs []float64, totalSpans uint64) []histogramQuantile {
	buckets := h.buckets()
	out := make([]histogramQuantile, 0, len(qs))

	var below uint64 // values in buckets before b
	b := 0
	for _, q := range qs {
		rank := max(uint64(math.Ceil(q*float64(h.total))), 1)
		for below+buckets[b].count < rank {
			below += buckets[b].count
			b++
		}
		above := h.total - below - buckets[b].count
		out = append(out, histogramQuantile{
			q:           q,
			upper:       buckets[b].upper,
			selectivity: float64(above) / float64(totalSpans),
		})
	}
	return out
}
