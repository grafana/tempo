package registry

import (
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

type counter struct {
	//nolint unused
	metric
	metricName string

	// seriesMtx is used to sync modifications to the map, not to the data in series
	seriesMtx    sync.RWMutex
	series       map[uint64]*counterSeries
	seriesDemand *Cardinality

	onAddSeries    func(count uint32) bool
	onRemoveSeries func(count uint32)

	externalLabels map[string]string
}

type counterSeries struct {
	labels      labels.Labels
	value       *atomic.Float64
	lastUpdated *atomic.Int64
	// firstSeries is used to track if this series is new to the counter.  This
	// is used to ensure that new counters being with 0, and then are incremented
	// to the desired value.  This avoids Prometheus throwing away the first
	// value in the series, due to the transition from null -> x.
	firstSeries *atomic.Bool
}

var (
	_ Counter = (*counter)(nil)
	_ metric  = (*counter)(nil)
)

func (co *counterSeries) isNew() bool {
	return co.firstSeries.Load()
}

func (co *counterSeries) registerSeenSeries() {
	co.firstSeries.Store(false)
}

func newCounter(name string, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32), externalLabels map[string]string, staleDuration time.Duration) *counter {
	if onAddSeries == nil {
		onAddSeries = func(uint32) bool {
			return true
		}
	}
	if onRemoveSeries == nil {
		onRemoveSeries = func(uint32) {}
	}

	return &counter{
		metricName:     name,
		series:         make(map[uint64]*counterSeries),
		seriesDemand:   NewCardinality(staleDuration, removeStaleSeriesInterval),
		onAddSeries:    onAddSeries,
		onRemoveSeries: onRemoveSeries,
		externalLabels: externalLabels,
	}
}

func (c *counter) Inc(labelValueCombo *LabelValueCombo, value float64) {
	if value < 0 {
		panic("counter can only increase")
	}

	hash := labelValueCombo.getHash()

	c.seriesMtx.RLock()
	s, ok := c.series[hash]
	c.seriesMtx.RUnlock()

	c.seriesDemand.Insert(hash)
	if ok {
		c.updateSeries(s, value)
		return
	}

	c.seriesMtx.Lock()
	defer c.seriesMtx.Unlock()

	if existing, ok := c.series[hash]; ok {
		c.updateSeries(existing, value)
		return
	}

	if !c.onAddSeries(1) {
		return
	}

	c.series[hash] = c.newSeries(labelValueCombo, value)
}

func (c *counter) newSeries(labelValueCombo *LabelValueCombo, value float64) *counterSeries {
	return &counterSeries{
		labels:      getSeriesLabels(c.metricName, labelValueCombo, c.externalLabels),
		value:       atomic.NewFloat64(value),
		lastUpdated: atomic.NewInt64(time.Now().UnixMilli()),
		firstSeries: atomic.NewBool(true),
	}
}

func (c *counter) updateSeries(s *counterSeries, value float64) {
	s.value.Add(value)
	s.lastUpdated.Store(time.Now().UnixMilli())
}

func (c *counter) name() string {
	return c.metricName
}

func (c *counter) collectMetrics(appender storage.Appender, timeMs int64) error {
	c.seriesMtx.RLock()
	defer c.seriesMtx.RUnlock()

	for _, s := range c.series {
		// If we are about to call Append for the first time on a series, we need
		// to first insert a 0 value to allow Prometheus to start from a non-null
		// value.
		if s.isNew() {
			// We set the timestamp of the init serie at the end of the previous minute, that way we ensure it ends in a
			// different aggregation interval to avoid be downsampled.
			endOfLastMinuteMs := getEndOfLastMinuteMs(timeMs)
			_, err := appender.Append(0, s.labels, endOfLastMinuteMs, 0)
			if err != nil {
				return err
			}
			s.registerSeenSeries()
		}

		_, err := appender.Append(0, s.labels, timeMs, s.value.Load())
		if err != nil {
			return err
		}

		// TODO: support exemplars
	}

	return nil
}

func (c *counter) countActiveSeries() int {
	c.seriesMtx.RLock()
	defer c.seriesMtx.RUnlock()

	return len(c.series)
}

func (c *counter) countSeriesDemand() int {
	c.seriesMtx.RLock()
	defer c.seriesMtx.RUnlock()

	return int(c.seriesDemand.Estimate())
}

func (c *counter) removeStaleSeries(staleTimeMs int64) {
	c.seriesMtx.Lock()
	defer c.seriesMtx.Unlock()

	for hash, s := range c.series {
		if s.lastUpdated.Load() < staleTimeMs {
			delete(c.series, hash)
			c.onRemoveSeries(1)
		}
	}
	c.seriesDemand.Advance()
}
