package collector

import "sync"

type ScopedDistinctString struct {
	cols        map[string]*DistinctString
	newCol      func(int) *DistinctString
	maxLen      int
	curLen      int
	limExceeded bool
	diffEnabled bool
	mtx         sync.Mutex
}

func NewScopedDistinctString(maxDataSize int) *ScopedDistinctString {
	return &ScopedDistinctString{
		cols:        map[string]*DistinctString{},
		newCol:      NewDistinctString,
		maxLen:      maxDataSize,
		diffEnabled: false,
	}
}

func NewScopedDistinctStringWithDiff(maxDataSize int) *ScopedDistinctString {
	return &ScopedDistinctString{
		cols:        map[string]*DistinctString{},
		newCol:      NewDistinctStringWithDiff,
		maxLen:      maxDataSize,
		diffEnabled: true,
	}
}

// FIXME: also add a benchmark for this to show it goes faster without diff support

// Collect adds a new value to the distinct string collector.
func (d *ScopedDistinctString) Collect(scope string, val string) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.limExceeded {
		return
	}

	valueLen := len(val)
	// can it fit?
	if d.maxLen > 0 && d.curLen+valueLen > d.maxLen {
		// No
		d.limExceeded = true
		return
	}

	// get or create collector
	col, ok := d.cols[scope]
	if !ok {
		col = d.newCol(0)
		d.cols[scope] = col
	}

	// add valueLen if we successfully added the value
	if col.Collect(val) {
		d.curLen += valueLen
	}
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
func (d *ScopedDistinctString) Exceeded() bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	return d.limExceeded
}

// Diff returns all new strings collected since the last time Diff was called
func (d *ScopedDistinctString) Diff() map[string][]string {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// TODO: return error if Diff is called on diffEnabled=false collector
	if !d.diffEnabled {
		return nil
	}

	ss := map[string][]string{}

	for k, v := range d.cols {
		diff := v.Diff()
		if len(diff) > 0 {
			ss[k] = diff
		}
	}

	return ss
}
