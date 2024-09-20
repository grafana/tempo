package collector

import (
	"sort"
	"strings"
)

type DistinctString struct {
	values   map[string]struct{}
	new      map[string]struct{}
	maxLen   int
	currLen  int
	totalLen int
}

// NewDistinctString with the given maximum data size. This is calculated
// as the total length of the recorded strings. For ease of use, maximum=0
// is interpreted as unlimited.
func NewDistinctString(maxDataSize int) *DistinctString {
	return &DistinctString{
		values: make(map[string]struct{}),
		new:    make(map[string]struct{}),
		maxLen: maxDataSize,
	}
}

// Collect adds a new value to the distinct string collector.
// return indicates if the value was added or not.
func (d *DistinctString) Collect(s string) bool {
	if _, ok := d.values[s]; ok {
		// Already present
		return false
	}

	// New entry
	d.totalLen += len(s)

	// Can it fit?
	if d.maxLen > 0 && d.currLen+len(s) > d.maxLen {
		// No
		return false
	}

	// Clone instead of referencing original
	s = strings.Clone(s)

	d.new[s] = struct{}{}
	d.values[s] = struct{}{}
	d.currLen += len(s)

	return true
}

// Strings returns the final list of distinct values collected and sorted.
func (d *DistinctString) Strings() []string {
	ss := make([]string, 0, len(d.values))

	for k := range d.values {
		ss = append(ss, k)
	}

	sort.Strings(ss)
	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
func (d *DistinctString) Exceeded() bool {
	return d.totalLen > d.currLen
}

// TotalDataSize is the total size of all distinct strings encountered.
func (d *DistinctString) TotalDataSize() int {
	return d.totalLen
}

// Diff returns all new strings collected since the last time diff was called
func (d *DistinctString) Diff() []string {
	ss := make([]string, 0, len(d.new))

	for k := range d.new {
		ss = append(ss, k)
	}

	clear(d.new)
	sort.Strings(ss)
	return ss
}
