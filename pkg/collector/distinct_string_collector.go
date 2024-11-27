package collector

import (
	"sort"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
)

type DistinctString struct {
	values           map[string]struct{}
	new              map[string]struct{}
	maxDataSize      int
	currDataSize     int
	currentValuesLen uint32
	maxValues        uint32
	maxCacheHits     uint32
	currentCacheHits uint32
	diffEnabled      bool
	limExceeded      bool
	mtx              sync.Mutex
	logger           log.Logger
}

// NewDistinctString with the given maximum data size and max items.
// MaxDataSize is calculated as the total length of the recorded strings.
// For ease of use, maxDataSize=0 and maxItems=0 are interpreted as unlimited.
func NewDistinctString(maxDataSize int, maxValues uint32, staleValueThreshold uint32, logger log.Logger) *DistinctString {
	return &DistinctString{
		values:       make(map[string]struct{}),
		maxDataSize:  maxDataSize,
		diffEnabled:  false, // disable diff to make it faster
		maxValues:    maxValues,
		maxCacheHits: staleValueThreshold,
		logger:       logger,
	}
}

// NewDistinctStringWithDiff is like NewDistinctString but with diff support enabled.
// MaxDataSize is calculated as the total length of the recorded strings.
// For ease of use, maxDataSize=0 and maxItems=0 are interpreted as unlimited.
func NewDistinctStringWithDiff(maxDataSize int, maxValues uint32, staleValueThreshold uint32, logger log.Logger) *DistinctString {
	return &DistinctString{
		values:       make(map[string]struct{}),
		new:          make(map[string]struct{}),
		maxDataSize:  maxDataSize,
		diffEnabled:  true,
		maxValues:    maxValues,
		maxCacheHits: staleValueThreshold,
		logger:       logger,
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
	valueLen := len(s)

	if d.maxDataSize > 0 && d.currDataSize+valueLen >= d.maxDataSize {
		level.Warn(d.logger).Log("msg", "Max data exceeded", "dataSize", d.currDataSize, "maxDataSize", d.maxDataSize)
		d.limExceeded = true
		return false
	}

	if d.maxValues > 0 && d.currentValuesLen >= d.maxValues {
		level.Warn(d.logger).Log("msg", "Max values exceeded", "values", d.currentValuesLen, "maxValues", d.maxValues)
		d.limExceeded = true
		return false
	}

	if d.maxCacheHits > 0 && d.currentCacheHits >= d.maxCacheHits {
		level.Warn(d.logger).Log("msg", "Max stale values exceeded", "cacheHits", d.currentCacheHits, "maxValues", d.maxCacheHits)
		d.limExceeded = true
		return false
	}

	if _, ok := d.values[s]; ok {
		// Already present
		d.currentCacheHits++
		return false
	}
	d.currentCacheHits = 0 // CacheHits reset to 0 when a new value is found

	// Clone instead of referencing original
	s = strings.Clone(s)

	if d.diffEnabled {
		d.new[s] = struct{}{}
	}
	d.values[s] = struct{}{}
	d.currDataSize += valueLen
	d.currentValuesLen++

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

	return d.currDataSize
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
