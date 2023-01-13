package util

type DistinctValueCollector[T comparable] struct {
	values   map[T]struct{}
	len      func(T) int
	maxLen   int
	currLen  int
	totalLen int
}

// NewDistinctStringCollector with the given maximum data size. This is calculated
// as the total length of the recorded strings. For ease of use, maximum=0
// is interpreted as unlimited.
func NewDistinctValueCollector[T comparable](maxDataSize int, len func(T) int) *DistinctValueCollector[T] {
	return &DistinctValueCollector[T]{
		values: make(map[T]struct{}),
		maxLen: maxDataSize,
		len:    len,
	}
}

func (d *DistinctValueCollector[T]) Collect(v T) (exceeded bool) {
	if _, ok := d.values[v]; ok {
		// Already present
		return
	}

	len := d.len(v)

	// Record total inspected length regardless
	d.totalLen += len

	// Can it fit?
	if d.maxLen > 0 && d.currLen+len > d.maxLen {
		// No
		return true
	}

	d.values[v] = struct{}{}
	d.currLen += len
	return false
}

// Strings returns the final list of distinct values collected and sorted.
func (d *DistinctValueCollector[T]) Values() []T {
	ss := make([]T, 0, len(d.values))

	for k := range d.values {
		ss = append(ss, k)
	}

	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
func (d *DistinctValueCollector[T]) Exceeded() bool {
	return d.totalLen > d.currLen
}

// TotalDataSize is the total size of all distinct strings encountered.
func (d *DistinctValueCollector[T]) TotalDataSize() int {
	return d.totalLen
}
