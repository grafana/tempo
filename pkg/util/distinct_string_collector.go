package util

import (
	"sort"
)

type DistinctStringCollector struct {
	c *DistinctValueCollector[string]
}

// NewDistinctStringCollector with the given maximum data size. This is calculated
// as the total length of the recorded strings. For ease of use, maximum=0
// is interpreted as unlimited.
func NewDistinctStringCollector(maxDataSize int) *DistinctStringCollector {
	return &DistinctStringCollector{c: NewDistinctValueCollector[string](maxDataSize, func(s string) int {
		return len(s)
	})}
}

func (d *DistinctStringCollector) Collect(s string) bool {
	return d.c.Collect(s)
}

// Strings returns the final list of distinct values collected and sorted.
func (d *DistinctStringCollector) Strings() []string {
	values := d.c.Values()

	ss := make([]string, 0, len(values))
	ss = append(ss, values...)

	sort.Strings(ss)
	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
func (d *DistinctStringCollector) Exceeded() bool {
	return d.c.Exceeded()
}

// TotalDataSize is the total size of all distinct strings encountered.
func (d *DistinctStringCollector) TotalDataSize() int {
	return d.c.TotalDataSize()
}
