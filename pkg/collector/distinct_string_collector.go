package collector

import (
	"sort"
	"strings"
	"sync"
)

type DistinctString struct {
	values      map[string]struct{}
	new         map[string]struct{}
	maxLen      int
	currLen     int
	diffEnabled bool
	limExceeded bool
	mtx         sync.Mutex
}

// NewDistinctString with the given maximum data size. This is calculated
// as the total length of the recorded strings. For ease of use, maximum=0
// is interpreted as unlimited.
func NewDistinctString(maxDataSize int) *DistinctString {
	return &DistinctString{
		values:      make(map[string]struct{}),
		new:         make(map[string]struct{}),
		maxLen:      maxDataSize,
		diffEnabled: false, // disable diff to make it faster
	}
}

// NewDistinctStringWithDiff is like NewDistinctString but with diff support enabled.
func NewDistinctStringWithDiff(maxDataSize int) *DistinctString {
	return &DistinctString{
		values:      make(map[string]struct{}),
		new:         make(map[string]struct{}),
		maxLen:      maxDataSize,
		diffEnabled: true,
	}
}

// FIXME: also add a benchmark for this to show it goes faster without diff support

// Collect adds a new value to the distinct string collector.
// return indicates if the value was added or not.
// FIXME: return an exceeded flag to stop early
func (d *DistinctString) Collect(s string) bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.limExceeded {
		return false
	}

	if _, ok := d.values[s]; ok {
		// Already present
		return false
	}

	valueLen := len(s)
	// Can it fit?
	if d.maxLen > 0 && d.currLen+valueLen > d.maxLen {
		// No, it can't fit
		d.limExceeded = true
		return false
	}

	// Clone instead of referencing original
	s = strings.Clone(s)

	if d.diffEnabled {
		d.new[s] = struct{}{}
	}
	d.values[s] = struct{}{}
	d.currLen += valueLen

	return true
}

// Strings returns the final list of distinct values collected and sorted.
func (d *DistinctString) Strings() []string {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	ss := make([]string, 0, len(d.values))

	for k := range d.values {
		ss = append(ss, k)
	}

	sort.Strings(ss)
	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
func (d *DistinctString) Exceeded() bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	return d.limExceeded
}

// Size is the total size of all distinct strings encountered.
func (d *DistinctString) Size() int {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	return d.currLen
}

// Diff returns all new strings collected since the last time diff was called
func (d *DistinctString) Diff() ([]string, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if !d.diffEnabled {
		return nil, errDiffNotEnabled
	}

	ss := make([]string, 0, len(d.new))

	for k := range d.new {
		ss = append(ss, k)
	}

	clear(d.new)
	sort.Strings(ss)
	return ss, nil
}
