package benchmark

import (
	"cmp"
	"container/heap"
	"slices"
	"strings"
)

// valueCount is a value and the number of spans carrying it.
type valueCount struct {
	value string
	count uint64
}

// spaceSaving counts a stream's most frequent values in bounded memory
// (Metwally et al., "Efficient Computation of Frequent and Top-k Elements in
// Data Streams"), weighted by the spans each occurrence covers. Counts are
// exact until more than capacity distinct values arrive. After that a new value
// takes over the least frequent counter and its count, so a count is
// over-stated by at most total/capacity, and any value carried by more than
// that many is guaranteed to be kept.
type spaceSaving struct {
	capacity int
	entries  map[string]*ssEntry
	heap     ssHeap
	evicted  bool
}

type ssEntry struct {
	valueCount
	index int
}

func newSpaceSaving(capacity int) *spaceSaving {
	return &spaceSaving{capacity: capacity, entries: map[string]*ssEntry{}}
}

func (s *spaceSaving) add(v string, n uint64) {
	if e, ok := s.entries[v]; ok {
		e.count += n
		heap.Fix(&s.heap, e.index)
		return
	}

	// A kept value is copied so it does not pin the trace it came from.
	v = strings.Clone(v)
	if len(s.heap) < s.capacity {
		e := &ssEntry{valueCount: valueCount{value: v, count: n}}
		s.entries[v] = e
		heap.Push(&s.heap, e)
		return
	}

	s.evicted = true
	e := s.heap[0]
	delete(s.entries, e.value)
	e.value = v
	e.count += n
	s.entries[v] = e
	heap.Fix(&s.heap, 0)
}

// distinct is the number of distinct values seen, exact only if none was
// evicted.
func (s *spaceSaving) distinct() int {
	return len(s.entries)
}

func (s *spaceSaving) top(n int) []valueCount {
	out := make([]valueCount, 0, len(s.heap))
	for _, e := range s.heap {
		out = append(out, e.valueCount)
	}
	sortValueCounts(out)
	return out[:min(n, len(out))]
}

// sortValueCounts orders by count descending, and by value to keep ties
// deterministic.
func sortValueCounts(vs []valueCount) {
	slices.SortFunc(vs, func(a, b valueCount) int {
		return cmp.Or(cmp.Compare(b.count, a.count), cmp.Compare(a.value, b.value))
	})
}

// ssHeap is a min-heap of counters, so the least frequent is the one replaced.
type ssHeap []*ssEntry

func (h ssHeap) Len() int { return len(h) }

func (h ssHeap) Less(i, j int) bool {
	return cmp.Or(cmp.Compare(h[i].count, h[j].count), cmp.Compare(h[j].value, h[i].value)) < 0
}

func (h ssHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h *ssHeap) Push(x any) {
	e := x.(*ssEntry)
	e.index = len(*h)
	*h = append(*h, e)
}

func (h *ssHeap) Pop() any {
	old := *h
	e := old[len(old)-1]
	*h = old[:len(old)-1]
	return e
}
