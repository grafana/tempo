package collector

import (
	"sync"
)

type DistinctValue[T comparable] struct {
	values      map[T]struct{}
	new         map[T]struct{}
	len         func(T) int
	maxLen      int
	currLen     int
	limExceeded bool
	diffEnabled bool
	mtx         sync.Mutex
}

// NewDistinctValue with the given maximum data size. This is calculated
// as the total length of the recorded strings. For ease of use, maximum=0
// is interpreted as unlimited.
// Use NewDistinctValueWithDiff to enable diff support, but that one is slightly slower.
func NewDistinctValue[T comparable](maxDataSize int, len func(T) int) *DistinctValue[T] {
	return &DistinctValue[T]{
		values:      make(map[T]struct{}),
		new:         make(map[T]struct{}),
		maxLen:      maxDataSize,
		diffEnabled: false, // disable diff to make it faster
		len:         len,
	}
}

// NewDistinctValueWithDiff is like NewDistinctValue but with diff support enabled.
func NewDistinctValueWithDiff[T comparable](maxDataSize int, len func(T) int) *DistinctValue[T] {
	return &DistinctValue[T]{
		values:      make(map[T]struct{}),
		new:         make(map[T]struct{}),
		maxLen:      maxDataSize,
		diffEnabled: true,
		len:         len,
	}
}

// Collect adds a new value to the distinct value collector.
// return true when it reaches the limits and can't fit more values.
// callers of return of Collect or call Exceeded to stop early.
func (d *DistinctValue[T]) Collect(v T) (exceeded bool) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.limExceeded {
		return true
	}

	// Calculate length
	valueLen := d.len(v)

	// Can it fit?
	// note: we will stop adding values slightly before the limit is reached
	if d.maxLen > 0 && d.currLen+valueLen >= d.maxLen {
		// No, it can't fit
		exceeded = true
		return true
	}

	if _, ok := d.values[v]; ok {
		return // Already present
	}

	if d.diffEnabled {
		d.new[v] = struct{}{}
	}

	d.values[v] = struct{}{}
	d.currLen += valueLen

	return false
}

// Values returns the final list of distinct values collected and sorted.
func (d *DistinctValue[T]) Values() []T {
	ss := make([]T, 0, len(d.values))

	d.mtx.Lock()
	defer d.mtx.Unlock()

	for k := range d.values {
		ss = append(ss, k)
	}

	return ss
}

// Exceeded indicates that
// if we get rid of totalLen, then Exceeded won't work as expected
func (d *DistinctValue[T]) Exceeded() bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.limExceeded
}

// Size is the total size of all distinct items collected
func (d *DistinctValue[T]) Size() int {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.currLen
}

// Diff returns all new strings collected since the last time diff was called
// returns nil if diff is not enabled
func (d *DistinctValue[T]) Diff() []T {
	if !d.diffEnabled {
		return nil
	}

	ss := make([]T, 0, len(d.new))

	d.mtx.Lock()
	defer d.mtx.Unlock()

	for k := range d.new {
		ss = append(ss, k)
	}

	clear(d.new)
	return ss
}
