package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

type counter struct {
	name   string
	labels []string

	// seriesMtx is used to sync modifications to the map, not to the data in series
	seriesMtx sync.RWMutex
	series    map[uint64]*counterSeries

	onAddSeries    func(count uint32) bool
	onRemoveSeries func(count uint32)
}

type counterSeries struct {
	// labelValues should not be modified after creation
	labelValues []string
	value       *atomic.Float64
	lastUpdated *atomic.Int64
}

var _ Counter = (*counter)(nil)
var _ metric = (*counter)(nil)

func newCounter(name string, labels []string, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32)) *counter {
	if onAddSeries == nil {
		onAddSeries = func(uint32) bool {
			return true
		}
	}
	if onRemoveSeries == nil {
		onRemoveSeries = func(uint32) {}
	}

	return &counter{
		name:           name,
		labels:         labels,
		series:         make(map[uint64]*counterSeries),
		onAddSeries:    onAddSeries,
		onRemoveSeries: onRemoveSeries,
	}
}

func (c *counter) Inc(labelValues *LabelValues, value float64) {
	if value < 0 {
		panic("counter can only increase")
	}
	if len(c.labels) != len(labelValues.getValues()) {
		panic(fmt.Sprintf("length of given label values does not match with labels, labels: %v, label values: %v", c.labels, labelValues))
	}

	hash := labelValues.getHash()

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

	newSeries := c.newSeries(labelValues, value)

	c.seriesMtx.Lock()
	defer c.seriesMtx.Unlock()

	s, ok = c.series[hash]
	if ok {
		c.updateSeries(s, value)
		return
	}
	c.series[hash] = newSeries
}

func (c *counter) newSeries(labelValues *LabelValues, value float64) *counterSeries {
	return &counterSeries{
		labelValues: labelValues.getValuesCopy(),
		value:       atomic.NewFloat64(value),
		lastUpdated: atomic.NewInt64(time.Now().UnixMilli()),
	}
}

func (c *counter) updateSeries(s *counterSeries, value float64) {
	s.value.Add(value)
	s.lastUpdated.Store(time.Now().UnixMilli())
}

func (c *counter) collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	c.seriesMtx.RLock()
	defer c.seriesMtx.RUnlock()

	activeSeries = len(c.series)

	lbls := make(labels.Labels, 1+len(externalLabels)+len(c.labels))
	lb := labels.NewBuilder(lbls)

	// set metric name
	lb.Set(labels.MetricName, c.name)
	// set external labels
	for name, value := range externalLabels {
		lb.Set(name, value)
	}

	for _, s := range c.series {
		// set series-specific labels
		for i, name := range c.labels {
			lb.Set(name, s.labelValues[i])
		}

		_, err = appender.Append(0, lb.Labels(), timeMs, s.value.Load())
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
