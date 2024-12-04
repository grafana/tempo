package collector

import (
	"sync"
)

const IntrinsicScope = "intrinsic"

type ScopedDistinctString struct {
	cols            map[string]*DistinctString
	newCol          func(int, uint32, uint32) *DistinctString
	maxDataSize     int
	currDataSize    int
	limExceeded     bool
	maxCacheHits    uint32
	diffEnabled     bool
	maxTagsPerScope uint32
	stopReason      string
	mtx             sync.Mutex
}

// NewScopedDistinctString collects the tags per scope
// MaxDataSize is calculated as the total length of the recorded strings.
// MaxTagsPerScope controls how many tags can be added per scope. The intrinsic scope is unbounded.
// For ease of use, maxDataSize=0 and maxTagsPerScope=0 are interpreted as unlimited.
func NewScopedDistinctString(maxDataSize int, maxTagsPerScope uint32, staleValueThreshold uint32) *ScopedDistinctString {
	return &ScopedDistinctString{
		cols:            map[string]*DistinctString{},
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
		cols:            map[string]*DistinctString{},
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
func (d *ScopedDistinctString) Collect(scope string, val string) (exceeded bool) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.limExceeded {
		return true
	}

	valueLen := len(val)
	// can it fit?
	if d.maxDataSize > 0 && d.currDataSize+valueLen > d.maxDataSize {
		// No
		d.limExceeded = true
		return true
	}

	// get or create collector
	col, ok := d.cols[scope]
	if !ok {
		if scope == IntrinsicScope {
			col = d.newCol(0, 0, 0)
		} else {
			col = d.newCol(0, d.maxTagsPerScope, d.maxCacheHits)
		}
		d.cols[scope] = col
	}

	// add valueLen if we successfully added the value
	if col.Collect(val) {
		d.currDataSize += valueLen
	}
	if col.Exceeded() {
		// we stop if one of the scopes exceed the limit
		d.limExceeded = true
		d.stopReason = col.stopReason
		return true
	}
	return false
}

// Strings returns the final list of distinct values collected and sorted.
func (d *ScopedDistinctString) Strings() map[string][]string {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	ss := map[string][]string{}

	for k, v := range d.cols {
		ss[k] = v.Strings()
	}

	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
// Or because one of the scopes max tags was reached.
func (d *ScopedDistinctString) Exceeded() bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.limExceeded {
		return true
	}

	for _, v := range d.cols {
		if v.Exceeded() {
			return true
		}
	}
	return false
}

func (d *ScopedDistinctString) StopReason() string {
	return d.stopReason
}

// Diff returns all new strings collected since the last time Diff was called
func (d *ScopedDistinctString) Diff() (map[string][]string, error) {
	if !d.diffEnabled {
		return nil, errDiffNotEnabled
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	ss := map[string][]string{}

	for k, v := range d.cols {
		diff, err := v.Diff()
		if err != nil {
			return nil, err
		}

		if len(diff) > 0 {
			ss[k] = diff
		}
	}

	return ss, nil
}
