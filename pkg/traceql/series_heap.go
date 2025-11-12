package traceql

import "strings"

// seriesValue keeps a value from a time series with its key
type seriesValue struct {
	key   SeriesMapKey
	value float64
}

// compareSeriesMapKey compares two SeriesMapKey values deterministically.
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func compareSeriesMapKey(a, b SeriesMapKey) int {
	for i := range a {
		// labels should be in the same order so we should really only have to compare value strings here.
		// doing the resut of comparisons for safety.
		if cmp := strings.Compare(a[i].Value.str, b[i].Value.str); cmp != 0 {
			return cmp * -1 // reverse the order b/c we want topk to return alphabetically earlier first
		}
		if cmp := strings.Compare(a[i].Name, b[i].Name); cmp != 0 {
			return cmp * -1
		}
		if a[i].Value.typ != b[i].Value.typ {
			if a[i].Value.typ < b[i].Value.typ {
				return -1
			}
			return 1
		}
		if a[i].Value.code != b[i].Value.code {
			if a[i].Value.code < b[i].Value.code {
				return -1
			}
			return 1
		}
	}
	return 0
}

// dataPointGreaterThan returns true if the new value should replace the smallest in a topk heap.
// This happens when the new value is greater, or equal but alphabetically earlier.
func dataPointGreaterThan(a, b seriesValue) bool {
	if a.value == b.value {
		return compareSeriesMapKey(a.key, b.key) > 0
	}

	return a.value > b.value
}

// dataPointLessThan returns true if the new value should replace the largest in a bottomk heap.
// This happens when the new value is smaller, or equal but alphabetically earlier.
func dataPointLessThan(a, b seriesValue) bool {
	if a.value == b.value {
		return compareSeriesMapKey(a.key, b.key) < 0
	}

	return a.value < b.value
}

// seriesHeap implements a min-heap of seriesValue
type seriesHeap []seriesValue

func (h seriesHeap) Len() int { return len(h) }

func (h seriesHeap) Less(i, j int) bool {
	return dataPointLessThan(h[i], h[j])
}

func (h seriesHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *seriesHeap) Push(x interface{}) {
	*h = append(*h, x.(seriesValue))
}

func (h *seriesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// reverseSeriesHeap implements a max-heap of seriesValue
type reverseSeriesHeap []seriesValue

func (h reverseSeriesHeap) Len() int { return len(h) }

func (h reverseSeriesHeap) Less(i, j int) bool {
	return dataPointGreaterThan(h[i], h[j])
}

func (h reverseSeriesHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *reverseSeriesHeap) Push(x interface{}) {
	*h = append(*h, x.(seriesValue))
}

func (h *reverseSeriesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
