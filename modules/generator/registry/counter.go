package registry

import (
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

var _ metric = (*counter)(nil)

type counter struct {
	metric
	metricName string

	// seriesMtx is used to sync modifications to the map, not to the data in series
	seriesMtx sync.RWMutex
	series    map[uint64]*counterSeries

	onAddSeries    func(count uint32) bool
	onRemoveSeries func(count uint32)
}

type counterSeries struct {
	labels      LabelPair
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

const insertOffsetDuration = 1 * time.Second

func (co *counterSeries) isNew() bool {
	return co.firstSeries.Load()
}

func (co *counterSeries) registerSeenSeries() {
	co.firstSeries.Store(false)
}

func newCounter(name string, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32)) *counter {
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
		onAddSeries:    onAddSeries,
		onRemoveSeries: onRemoveSeries,
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

	if ok {
		c.updateSeries(s, value)
		return
	}

	if !c.onAddSeries(1) {
		return
	}

	newSeries := c.newSeries(labelValueCombo, value)

	c.seriesMtx.Lock()
	defer c.seriesMtx.Unlock()

	s, ok = c.series[hash]
	if ok {
		c.updateSeries(s, value)
		return
	}
	c.series[hash] = newSeries
}

func (c *counter) newSeries(labelValueCombo *LabelValueCombo, value float64) *counterSeries {
	return &counterSeries{
		labels:      labelValueCombo.getLabelPair(),
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

func (c *counter) collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	c.seriesMtx.RLock()
	defer c.seriesMtx.RUnlock()

	activeSeries = len(c.series)

	labelsCount := 0
	if activeSeries > 0 && c.series[0] != nil {
		labelsCount = len(c.series[0].labels.names)
	}

	// base labels
	baseLabels := make(labels.Labels, 0, 1+len(externalLabels)+labelsCount)

	// add external labels
	for name, value := range externalLabels {
		baseLabels = append(baseLabels, labels.Label{Name: name, Value: value})
	}

	// add metric name
	baseLabels = append(baseLabels, labels.Label{Name: labels.MetricName, Value: c.metricName})

	lb := labels.NewBuilder(baseLabels)

	for _, s := range c.series {
		t := time.UnixMilli(timeMs)

		// reset labels for every series
		lb.Reset(baseLabels)

		// set series-specific labels
		for i, name := range s.labels.names {
			lb.Set(name, s.labels.values[i])
		}

		// If we are about to call Append for the first time on a series, we need
		// to first insert a 0 value to allow Prometheus to start from a non-null
		// value.
		if s.isNew() {
			_, err = appender.Append(0, lb.Labels(), timeMs, 0)
			if err != nil {
				return
			}
			// Increment timeMs to ensure that the next value is not at the same time.
			t = t.Add(insertOffsetDuration)
			s.registerSeenSeries()
		}

		_, err = appender.Append(0, lb.Labels(), t.UnixMilli(), s.value.Load())
		if err != nil {
			return
		}

		// TODO support exemplars
	}

	return
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
}
