package registry

import (
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

var _ metric = (*gauge)(nil)

// this is mostly copied from counter
type gauge struct {
	//nolint unused
	metric

	metricName string

	// seriesMtx is used to sync modifications to the map, not to the data in series
	seriesMtx sync.RWMutex
	series    map[uint64]*gaugeSeries

	onAddSeries    func(count uint32) bool
	onRemoveSeries func(count uint32)

	externalLabels map[string]string
}

type gaugeSeries struct {
	labels      labels.Labels
	value       *atomic.Float64
	lastUpdated *atomic.Int64
}

var (
	_ Gauge  = (*gauge)(nil)
	_ metric = (*gauge)(nil)
)

const (
	add = "add"
	set = "set"
)

func newGauge(name string, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32), externalLabels map[string]string) *gauge {
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
		series:         make(map[uint64]*gaugeSeries),
		onAddSeries:    onAddSeries,
		onRemoveSeries: onRemoveSeries,
		externalLabels: externalLabels,
	}
}

func (g *gauge) Set(labelValueCombo *LabelValueCombo, value float64) {
	g.updateSeries(labelValueCombo, value, set, true)
}

func (g *gauge) Inc(labelValueCombo *LabelValueCombo, value float64) {
	g.updateSeries(labelValueCombo, value, add, true)
}

func (g *gauge) SetForTargetInfo(labelValueCombo *LabelValueCombo, value float64) {
	g.updateSeries(labelValueCombo, value, set, false)
}

func (g *gauge) updateSeries(labelValueCombo *LabelValueCombo, value float64, operation string, updateIfAlreadyExist bool) {
	hash := labelValueCombo.getHash()

	g.seriesMtx.RLock()
	s, ok := g.series[hash]
	g.seriesMtx.RUnlock()

	if ok {
		// target_info will always be 1 so if the series exists, we don't need to go through this loop
		if !updateIfAlreadyExist {
			return
		}
		g.updateSeriesValue(s, value, operation)
		return
	}

	if !g.onAddSeries(1) {
		return
	}

	newSeries := g.newSeries(labelValueCombo, value)

	g.seriesMtx.Lock()
	defer g.seriesMtx.Unlock()

	s, ok = g.series[hash]
	if ok {
		g.updateSeriesValue(s, value, operation)
		return
	}
	g.series[hash] = newSeries
}

func (g *gauge) newSeries(labelValueCombo *LabelValueCombo, value float64) *gaugeSeries {
	lbls := labelValueCombo.getLabelPair()
	lb := labels.NewBuilder(make(labels.Labels, 1+len(lbls.names)+len(g.externalLabels)))

	for i, name := range lbls.names {
		lb.Set(name, lbls.values[i])
	}

	for name, value := range g.externalLabels {
		lb.Set(name, value)
	}

	lb.Set(labels.MetricName, g.metricName)

	return &gaugeSeries{
		labels:      lb.Labels(),
		value:       atomic.NewFloat64(value),
		lastUpdated: atomic.NewInt64(time.Now().UnixMilli()),
	}
}

func (g *gauge) updateSeriesValue(s *gaugeSeries, value float64, operation string) {
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

func (g *gauge) collectMetrics(appender storage.Appender, timeMs int64) (activeSeries int, err error) {
	g.seriesMtx.RLock()
	defer g.seriesMtx.RUnlock()

	activeSeries = len(g.series)

	for _, s := range g.series {
		t := time.UnixMilli(timeMs)
		_, err = appender.Append(0, s.labels, t.UnixMilli(), s.value.Load())
		if err != nil {
			return
		}
		// TODO: support exemplars
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
