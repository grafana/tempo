package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

// this is mostly copied from counter

type gauge struct {
	metricName string
	labels     []string

	// seriesMtx is used to sync modifications to the map, not to the data in series
	seriesMtx sync.RWMutex
	series    map[uint64]*gaugeSeries

	onAddSeries    func(count uint32) bool
	onRemoveSeries func(count uint32)
}

type gaugeSeries struct {
	// labelValues should not be modified after creation
	labelValues []string
	value       *atomic.Float64
	lastUpdated *atomic.Int64
	// firstSeries is used to track if this series is new to the gauge.  This
	// is used to ensure that new gauges being with 0, and then are incremented
	// to the desired value.  This avoids Prometheus throwing away the first
	// value in the series, due to the transition from null -> x.
	firstSeries *atomic.Bool
}

var _ Gauge = (*gauge)(nil)
var _ metric = (*gauge)(nil)

const add = "add"
const set = "set"

func (gs *gaugeSeries) isNew() bool {
	return gs.firstSeries.Load()
}

func (gs *gaugeSeries) registerSeenSeries() {
	gs.firstSeries.Store(false)
}

func newGauge(name string, labels []string, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32)) *gauge {
	if onAddSeries == nil {
		onAddSeries = func(uint32) bool {
			return true
		}
	}
	if onRemoveSeries == nil {
		onRemoveSeries = func(uint32) {}
	}

	return &gauge{
		metricName:     name,
		labels:         labels,
		series:         make(map[uint64]*gaugeSeries),
		onAddSeries:    onAddSeries,
		onRemoveSeries: onRemoveSeries,
	}
}

func (g *gauge) UpdateLabels(labels []string) {
	g.labels = labels
}

func (g *gauge) Set(labelValues *LabelValues, value float64) {
	g.change(labelValues, value, set)
}

func (g *gauge) Inc(labelValues *LabelValues, value float64) {
	g.change(labelValues, value, add)
}

func (g *gauge) change(labelValues *LabelValues, value float64, operation string) {
	if len(g.labels) != len(labelValues.getValues()) {
		panic(fmt.Sprintf("length of given label values does not match with labels, labels: %v, label values: %v", g.labels, labelValues))
	}

	hash := labelValues.getHash()

	g.seriesMtx.RLock()
	s, ok := g.series[hash]
	g.seriesMtx.RUnlock()

	if ok {
		g.updateSeries(s, value, operation)
		return
	}

	if !g.onAddSeries(1) {
		return
	}

	newSeries := g.newSeries(labelValues, value)

	g.seriesMtx.Lock()
	defer g.seriesMtx.Unlock()

	s, ok = g.series[hash]
	if ok {
		g.updateSeries(s, value, operation)
		return
	}
	g.series[hash] = newSeries
}

func (g *gauge) newSeries(labelValues *LabelValues, value float64) *gaugeSeries {
	return &gaugeSeries{
		labelValues: labelValues.getValuesCopy(),
		value:       atomic.NewFloat64(value),
		lastUpdated: atomic.NewInt64(time.Now().UnixMilli()),
		firstSeries: atomic.NewBool(true),
	}
}

func (g *gauge) updateSeries(s *gaugeSeries, value float64, operation string) {
	if operation == add {
		s.value.Add(value)
	} else {
		s.value = atomic.NewFloat64(value)
	}
	s.lastUpdated.Store(time.Now().UnixMilli())
}

func (g *gauge) name() string {
	return g.metricName
}

func (g *gauge) collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	g.seriesMtx.RLock()
	defer g.seriesMtx.RUnlock()

	activeSeries = len(g.series)

	lbls := make(labels.Labels, 1+len(externalLabels)+len(g.labels))
	lb := labels.NewBuilder(lbls)

	// set metric name
	lb.Set(labels.MetricName, g.metricName)
	// set external labels
	for name, value := range externalLabels {
		lb.Set(name, value)
	}

	for _, s := range g.series {
		t := time.UnixMilli(timeMs)
		// set series-specific labels
		for i, name := range g.labels {
			lb.Set(name, s.labelValues[i])
		}

		// If we are about to call Append for the first time on a series, we need
		// to first insert a 0 value to allow Prometheus to start from a non-null
		// value.
		if s.isNew() {
			_, err = appender.Append(0, lb.Labels(nil), timeMs, 0)
			if err != nil {
				return
			}
			// Increment timeMs to ensure that the next value is not at the same time.
			t = t.Add(insertOffsetDuration)
			s.registerSeenSeries()
		}

		_, err = appender.Append(0, lb.Labels(nil), t.UnixMilli(), s.value.Load())
		if err != nil {
			return
		}

		// TODO support exemplars
	}

	return
}

func (g *gauge) removeStaleSeries(staleTimeMs int64) {
	g.seriesMtx.Lock()
	defer g.seriesMtx.Unlock()

	for hash, s := range g.series {
		if s.lastUpdated.Load() < staleTimeMs {
			delete(g.series, hash)
			g.onRemoveSeries(1)
		}
	}
}
