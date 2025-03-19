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
	stale       *atomic.Bool
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

	// check if the series exists under read lock
	g.seriesMtx.RLock()
	s, ok := g.series[hash]
	g.seriesMtx.RUnlock()

	if ok {
		// update to existing series removes staleness
		s.stale.Store(false)

		// if update is needed, modify the value
		if updateIfAlreadyExist {
			g.updateSeriesValue(s, value, operation)
		}
		return
	}

	// if it doesn't exist, call onAddSeries
	if !g.onAddSeries(1) {
		return
	}

	// acquire full lock and recheck before adding
	g.seriesMtx.Lock()
	defer g.seriesMtx.Unlock()

	// check again in case another goroutine already added the series
	if s, ok := g.series[hash]; ok {
		s.stale.Store(false)
		if updateIfAlreadyExist {
			g.updateSeriesValue(s, value, operation)
		}
		return
	}

	// create and add new series
	g.series[hash] = g.newSeries(labelValueCombo, value)
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
		stale:       atomic.NewBool(false),
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
	activeSeries = len(g.series)
	staleSeries := []uint64{}

	for hash, s := range g.series {
		t := time.UnixMilli(timeMs)

		if s.stale.Load() {
			_, err = appender.Append(0, s.labels, t.UnixMilli(), staleMarker())
			if err != nil {
				g.seriesMtx.RUnlock()
				return
			}
			staleSeries = append(staleSeries, hash)
			continue
		}
		_, err = appender.Append(0, s.labels, t.UnixMilli(), s.value.Load())
		if err != nil {
			g.seriesMtx.RUnlock()
			return
		}
	}
	g.seriesMtx.RUnlock()

	// only acquire write lock if there are stale series to remove
	if len(staleSeries) > 0 {
		g.seriesMtx.Lock()
		defer g.seriesMtx.Unlock()

		for _, hash := range staleSeries {
			if s, ok := g.series[hash]; ok && s.stale.Load() {
				delete(g.series, hash)
			}
		}

		g.onRemoveSeries(uint32(len(staleSeries)))

	}

	return
}

func (g *gauge) removeStaleSeries(staleTimeMs int64) {
	g.seriesMtx.RLock()
	defer g.seriesMtx.RUnlock()

	for _, s := range g.series {
		if s.lastUpdated.Load() < staleTimeMs {
			s.stale.Store(true)
		}
	}
}
