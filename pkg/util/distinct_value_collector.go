package util

import (
	"sync"
)

type DistinctValueCollector[T comparable] struct {
	values   map[T]struct{}
	len      func(T) int
	maxLen   int
	currLen  int
	totalLen int
	mtx      sync.RWMutex
}

// NewDistinctValueCollector with the given maximum data size. This is calculated
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
	d.mtx.RLock()
	if _, ok := d.values[v]; ok {
		d.mtx.RUnlock()
		return // Already present
	}
	d.mtx.RUnlock()

	// Calculate length
	valueLen := d.len(v)

	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.values[v]; ok {
		return // Already present
	}

	// Record total inspected length regardless
	d.totalLen += valueLen

	// Can it fit?
	if d.maxLen > 0 && d.currLen+valueLen > d.maxLen {
		// No
		return true
	}

	d.values[v] = struct{}{}
	d.currLen += valueLen
	return false
}

// Values returns the final list of distinct values collected and sorted.
func (d *DistinctValueCollector[T]) Values() []T {
	ss := make([]T, 0, len(d.values))

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	for k := range d.values {
		ss = append(ss, k)
	}

	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
func (d *DistinctValueCollector[T]) Exceeded() bool {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.totalLen > d.currLen
}

// TotalDataSize is the total size of all distinct strings encountered.
func (d *DistinctValueCollector[T]) TotalDataSize() int {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.totalLen
}
