package collector

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type DistinctString struct {
	values           sync.Map // map[string]struct{}, written once per distinct value, read many times
	new              sync.Map // map[string]struct{}, only used when diffEnabled
	maxDataSize      int
	currDataSize     atomic.Int64
	currentValuesLen atomic.Uint32
	maxValues        uint32
	maxCacheHits     uint32
	currentCacheHits atomic.Uint32
	diffEnabled      bool
	limExceeded      atomic.Bool
	stopReason       atomic.Pointer[string]
}

// NewDistinctString with the given maximum data size and max items.
// MaxDataSize is calculated as the total length of the recorded strings.
// For ease of use, maxDataSize=0 and maxItems=0 are interpreted as unlimited.
func NewDistinctString(maxDataSize int, maxValues uint32, staleValueThreshold uint32) *DistinctString {
	return &DistinctString{
		maxDataSize:  maxDataSize,
		diffEnabled:  false, // disable diff to make it faster
		maxValues:    maxValues,
		maxCacheHits: staleValueThreshold,
	}
}

// NewDistinctStringWithDiff is like NewDistinctString but with diff support enabled.
// MaxDataSize is calculated as the total length of the recorded strings.
// For ease of use, maxDataSize=0 and maxItems=0 are interpreted as unlimited.
func NewDistinctStringWithDiff(maxDataSize int, maxValues uint32, staleValueThreshold uint32) *DistinctString {
	return &DistinctString{
		maxDataSize:  maxDataSize,
		diffEnabled:  true,
		maxValues:    maxValues,
		maxCacheHits: staleValueThreshold,
	}
}

// Collect adds a new value to the distinct string collector
// and returns a boolean indicating whether the value was successfully added or not.
// To check if the limit has been reached, you must call the Exceeded method separately.
//
// The hot path is an already-seen value (tag keys repeat across every row group
// and block), which is a lock-free sync.Map read with no shared writes. Limits
// are soft: under concurrency a few values may slip past before limExceeded is observed.
func (d *DistinctString) Collect(s string) (added bool) {
	if d.limExceeded.Load() {
		return false
	}

	valueLen := len(s)

	if d.maxDataSize > 0 && int(d.currDataSize.Load())+valueLen >= d.maxDataSize {
		d.setExceeded(fmt.Sprintf("Max data exceeded: dataSize %d, maxDataSize %d", d.currDataSize.Load(), d.maxDataSize))
		return false
	}

	if d.maxValues > 0 && d.currentValuesLen.Load() >= d.maxValues {
		d.setExceeded(fmt.Sprintf("Max values exceeded: values %d, maxValues %d", d.currentValuesLen.Load(), d.maxValues))
		return false
	}

	if d.maxCacheHits > 0 && d.currentCacheHits.Load() >= d.maxCacheHits {
		d.setExceeded(fmt.Sprintf("Max stale values exceeded: cacheHits %d, maxValues %d", d.currentCacheHits.Load(), d.maxCacheHits))
		return false
	}

	// Fast path: already present, lock-free read.
	if _, ok := d.values.Load(s); ok {
		if d.maxCacheHits > 0 {
			d.currentCacheHits.Add(1)
		}
		return false
	}

	// Clone instead of referencing original
	s = strings.Clone(s)

	// LoadOrStore keeps the insert atomic: a concurrent insert of the same value
	// is counted once, treated here as a cache hit.
	if _, loaded := d.values.LoadOrStore(s, struct{}{}); loaded {
		if d.maxCacheHits > 0 {
			d.currentCacheHits.Add(1)
		}
		return false
	}
	d.currentCacheHits.Store(0) // CacheHits reset to 0 when a new value is found

	if d.diffEnabled {
		d.new.Store(s, struct{}{})
	}
	d.currDataSize.Add(int64(valueLen))
	d.currentValuesLen.Add(1)

	return true
}

// setExceeded records the first stop reason and marks the collector as full.
func (d *DistinctString) setExceeded(reason string) {
	if d.limExceeded.Swap(true) {
		return // already marked by another goroutine
	}
	d.stopReason.Store(&reason)
}

// Strings returns the final list of distinct values collected and sorted.
func (d *DistinctString) Strings() []string {
	ss := make([]string, 0, d.currentValuesLen.Load())

	d.values.Range(func(k, _ any) bool {
		ss = append(ss, k.(string))
		return true
	})

	sort.Strings(ss)
	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
func (d *DistinctString) Exceeded() bool {
	return d.limExceeded.Load()
}

func (d *DistinctString) StopReason() string {
	if r := d.stopReason.Load(); r != nil {
		return *r
	}
	return ""
}

// Size is the total size of all distinct strings encountered.
func (d *DistinctString) Size() int {
	return int(d.currDataSize.Load())
}

// Diff returns all new strings collected since the last time diff was called
func (d *DistinctString) Diff() ([]string, error) {
	// can check diffEnabled without lock because it is not modified after creation
	if !d.diffEnabled {
		return nil, errDiffNotEnabled
	}

	ss := make([]string, 0)
	d.new.Range(func(k, _ any) bool {
		ss = append(ss, k.(string))
		d.new.Delete(k)
		return true
	})

	sort.Strings(ss)
	return ss, nil
}
