package collector

import (
	"errors"
	"fmt"
	"sync"
)

var errDiffNotEnabled = errors.New("diff not enabled")

type DistinctValue[T comparable] struct {
	values           map[T]struct{}
	new              map[T]struct{}
	len              func(T) int
	maxDataSize      int
	currDataSize     int
	currentValuesLen uint32
	maxValues        uint32
	maxCacheHits     uint32
	currentCacheHits uint32
	limExceeded      bool
	diffEnabled      bool
	stopReason       string
	mtx              sync.Mutex
}

// NewDistinctValue with the given maximum data size and values limited.
// maxDataSize is calculated as the total length of the recorded strings.
// staleValueThreshold introduces a stop condition that is triggered when the number of  found cache hits overcomes the limit
// For ease of use, maxDataSize=0 and maxValues are interpreted as unlimited.
// Use NewDistinctValueWithDiff to enable diff support, but that one is slightly slower.
func NewDistinctValue[T comparable](maxDataSize int, maxValues uint32, staleValueThreshold uint32, len func(T) int) *DistinctValue[T] {
	return &DistinctValue[T]{
		values:       make(map[T]struct{}),
		maxDataSize:  maxDataSize,
		diffEnabled:  false, // disable diff to make it faster
		len:          len,
		maxValues:    maxValues,
		maxCacheHits: staleValueThreshold,
	}
}

// NewDistinctValueWithDiff is like NewDistinctValue but with diff support enabled.
func NewDistinctValueWithDiff[T comparable](maxDataSize int, maxValues uint32, staleValueThreshold uint32, len func(T) int) *DistinctValue[T] {
	return &DistinctValue[T]{
		values:       make(map[T]struct{}),
		new:          make(map[T]struct{}),
		maxDataSize:  maxDataSize,
		diffEnabled:  true,
		len:          len,
		maxValues:    maxValues,
		maxCacheHits: staleValueThreshold,
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

	if d.maxDataSize > 0 && d.currDataSize+valueLen >= d.maxDataSize {
		d.stopReason = fmt.Sprintf("Max data exceeded: dataSize %d, maxDataSize %d", d.currDataSize, d.maxDataSize)
		d.limExceeded = true
		return true
	}

	if d.maxValues > 0 && d.currentValuesLen >= d.maxValues {
		d.stopReason = fmt.Sprintf("Max values exceeded: values %d, maxValues %d", d.currentValuesLen, d.maxValues)
		d.limExceeded = true
		return true
	}

	if d.maxCacheHits > 0 && d.currentCacheHits >= d.maxCacheHits {
		d.stopReason = fmt.Sprintf("Max stale values exceeded: cacheHits %d, maxValues %d", d.currentValuesLen, d.maxCacheHits)
		d.limExceeded = true
		return true
	}
	if _, ok := d.values[v]; ok {
		d.currentCacheHits++
		return false // Already present
	}
	d.currentCacheHits = 0 // CacheHits reset to 0 when a new value is found

	if d.diffEnabled {
		d.new[v] = struct{}{}
	}

	d.values[v] = struct{}{}
	d.currDataSize += valueLen
	d.currentValuesLen++

	return false
}

// Values returns the final list of distinct values collected and sorted.
func (d *DistinctValue[T]) Values() []T {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	ss := make([]T, 0, len(d.values))
	for k := range d.values {
		ss = append(ss, k)
	}

	return ss
}

// Exceeded indicates that we have exceeded the limit
// can be used to stop early and to avoid collecting further values
func (d *DistinctValue[T]) Exceeded() bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	return d.limExceeded
}

func (d *DistinctValue[T]) StopReason() string {
	return d.stopReason
}

// Size is the total size of all distinct items collected
func (d *DistinctValue[T]) Size() int {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	return d.currDataSize
}

// Diff returns all new strings collected since the last time diff was called
// returns nil if diff is not enabled
func (d *DistinctValue[T]) Diff() ([]T, error) {
	// can check diffEnabled without lock because it is not modified after creation
	if !d.diffEnabled {
		return nil, errDiffNotEnabled
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	ss := make([]T, 0, len(d.new))
	for k := range d.new {
		ss = append(ss, k)
	}

	clear(d.new)
	return ss, nil
}
