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

// Collect adds a new value to the distinct string collector
// and returns a boolean indicating whether the value was successfully added or not.
// To check if the limit has been reached, you must call the Exceeded method separately.
func (d *DistinctString) Collect(s string) (added bool) {
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
	// can check diffEnabled without lock because it is not modified after creation
	if !d.diffEnabled {
		return nil, errDiffNotEnabled
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	ss := make([]string, 0, len(d.new))

	for k := range d.new {
		ss = append(ss, k)
	}

	clear(d.new)
	sort.Strings(ss)
	return ss, nil
}
