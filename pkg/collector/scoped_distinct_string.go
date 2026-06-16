package collector

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const IntrinsicScope = "intrinsic"

type ScopedDistinctString struct {
	cols            sync.Map // map[string]*DistinctString, one collector per scope
	newCol          func(int, uint32, uint32) *DistinctString
	maxDataSize     int
	currDataSize    atomic.Int64
	limExceeded     atomic.Bool
	maxCacheHits    uint32
	diffEnabled     bool
	maxTagsPerScope uint32
	stopReason      atomic.Pointer[string]
}

// NewScopedDistinctString collects the tags per scope
// MaxDataSize is calculated as the total length of the recorded strings.
// MaxTagsPerScope controls how many tags can be added per scope. The intrinsic scope is unbounded.
// For ease of use, maxDataSize=0 and maxTagsPerScope=0 are interpreted as unlimited.
func NewScopedDistinctString(maxDataSize int, maxTagsPerScope uint32, staleValueThreshold uint32) *ScopedDistinctString {
	return &ScopedDistinctString{
		newCol:          NewDistinctString,
		maxDataSize:     maxDataSize,
		diffEnabled:     false,
		maxTagsPerScope: maxTagsPerScope,
		maxCacheHits:    staleValueThreshold,
	}
}

// NewScopedDistinctStringWithDiff collects the tags per scope with diff
// MaxDataSize is calculated as the total length of the recorded strings.
// MaxTagsPerScope controls how many tags can be added per scope. The intrinsic scope is unbounded.
// For ease of use, maxDataSize=0 and maxTagsPerScope=0 are interpreted as unlimited.
func NewScopedDistinctStringWithDiff(maxDataSize int, maxTagsPerScope uint32, staleValueThreshold uint32) *ScopedDistinctString {
	return &ScopedDistinctString{
		newCol:          NewDistinctStringWithDiff,
		maxDataSize:     maxDataSize,
		diffEnabled:     true,
		maxTagsPerScope: maxTagsPerScope,
		maxCacheHits:    staleValueThreshold,
	}
}

// Collect adds a new value to the distinct string collector.
// returns true when it reaches the limits and can't fit more values.
// can be used to stop early during Collect without calling Exceeded.
//
// Lock-free: per-scope collectors live in a sync.Map and hold their own
// lock-free state, so concurrent callers no longer serialize on one mutex.
func (d *ScopedDistinctString) Collect(scope string, val string) (exceeded bool) {
	if d.limExceeded.Load() {
		return true
	}

	valueLen := len(val)
	// can it fit?
	if d.maxDataSize > 0 && int(d.currDataSize.Load())+valueLen > d.maxDataSize {
		// No
		d.setExceeded(fmt.Sprintf("Max data exceeded: dataSize %d, maxDataSize %d", d.currDataSize.Load(), d.maxDataSize))
		return true
	}

	// get or create collector for this scope
	col := d.getOrCreateCollector(scope)

	// add valueLen if we successfully added the value
	if col.Collect(val) {
		d.currDataSize.Add(int64(valueLen))
	}
	if col.Exceeded() {
		// we stop if one of the scopes exceed the limit
		d.setExceeded(col.StopReason())
		return true
	}
	return false
}

// setExceeded records the first stop reason and marks the collector as full.
func (d *ScopedDistinctString) setExceeded(reason string) {
	if d.limExceeded.Swap(true) {
		return // already marked by another goroutine
	}
	d.stopReason.Store(&reason)
}

func (d *ScopedDistinctString) getOrCreateCollector(scope string) *DistinctString {
	if col, ok := d.cols.Load(scope); ok {
		return col.(*DistinctString)
	}

	var newCol *DistinctString
	if scope == IntrinsicScope {
		newCol = d.newCol(0, 0, 0)
	} else {
		newCol = d.newCol(0, d.maxTagsPerScope, d.maxCacheHits)
	}
	// LoadOrStore so a racing creation for the same scope keeps one collector.
	col, _ := d.cols.LoadOrStore(scope, newCol)
	return col.(*DistinctString)
}

// Strings returns the final list of distinct values collected and sorted.
func (d *ScopedDistinctString) Strings() map[string][]string {
	ss := map[string][]string{}

	d.cols.Range(func(k, v any) bool {
		ss[k.(string)] = v.(*DistinctString).Strings()
		return true
	})

	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
// Or because one of the scopes max tags was reached.
func (d *ScopedDistinctString) Exceeded() bool {
	if d.limExceeded.Load() {
		return true
	}

	exceeded := false
	d.cols.Range(func(_, v any) bool {
		if v.(*DistinctString).Exceeded() {
			exceeded = true
			return false // stop ranging
		}
		return true
	})
	return exceeded
}

func (d *ScopedDistinctString) StopReason() string {
	if r := d.stopReason.Load(); r != nil {
		return *r
	}
	return ""
}

// Diff returns all new strings collected since the last time Diff was called
func (d *ScopedDistinctString) Diff() (map[string][]string, error) {
	if !d.diffEnabled {
		return nil, errDiffNotEnabled
	}

	ss := map[string][]string{}

	var rangeErr error
	d.cols.Range(func(k, v any) bool {
		diff, err := v.(*DistinctString).Diff()
		if err != nil {
			rangeErr = err
			return false
		}
		if len(diff) > 0 {
			ss[k.(string)] = diff
		}
		return true
	})
	if rangeErr != nil {
		return nil, rangeErr
	}

	return ss, nil
}
