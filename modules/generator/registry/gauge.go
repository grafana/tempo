package registry

import (
	"sync"
	"time"

	"github.com/grafana/tempo/modules/generator/cardinality"
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
	seriesMtx    sync.RWMutex
	series       map[uint64]*gaugeSeries
	seriesDemand *cardinality.Cardinality

	lifecycler Limiter

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

func newGauge(name string, lifecycler Limiter, externalLabels map[string]string, staleDuration time.Duration) *gauge {
	return &gauge{
		metricName:     name,
		series:         make(map[uint64]*gaugeSeries),
		seriesDemand:   cardinality.NewCardinality(staleDuration, removeStaleSeriesInterval),
		lifecycler:     lifecycler,
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

	g.seriesDemand.Insert(hash)

	if ok {
		// target_info will always be 1 so if the series exists, we don't need to go through this loop
		if !updateIfAlreadyExist {
			return
		}
		g.updateSeriesValue(s, value, operation)
		return
	}

	g.seriesMtx.Lock()
	defer g.seriesMtx.Unlock()

	if existing, ok := g.series[hash]; ok {
		if !updateIfAlreadyExist {
			return
		}
		g.updateSeriesValue(existing, value, operation)
		return
	}

	if !g.lifecycler.OnAdd(hash, 1) {
		return
	}

	g.series[hash] = g.newSeries(labelValueCombo, value)
}

func (g *gauge) newSeries(labelValueCombo *LabelValueCombo, value float64) *gaugeSeries {
	return &gaugeSeries{
		labels:      getSeriesLabels(g.metricName, labelValueCombo, g.externalLabels),
		value:       atomic.NewFloat64(value),
		lastUpdated: atomic.NewInt64(time.Now().UnixMilli()),
	}
}

func (g *gauge) updateSeriesValue(s *gaugeSeries, value float64, operation string) {
	if operation == add {
		s.value.Add(value)
	} else {
		s.value.Store(value)
	}
	s.lastUpdated.Store(time.Now().UnixMilli())
}

func (g *gauge) name() string {
	return g.metricName
}

func (g *gauge) collectMetrics(appender storage.Appender, timeMs int64) error {
	g.seriesMtx.RLock()
	defer g.seriesMtx.RUnlock()

	for _, s := range g.series {
		_, err := appender.Append(0, s.labels, timeMs, s.value.Load())
		if err != nil {
			return err
		}
		// TODO: support exemplars
	}

	return nil
}

func (g *gauge) countActiveSeries() int {
	g.seriesMtx.RLock()
	defer g.seriesMtx.RUnlock()

	return len(g.series)
}

func (g *gauge) countSeriesDemand() int {
	g.seriesMtx.RLock()
	defer g.seriesMtx.RUnlock()

	return int(g.seriesDemand.Estimate())
}

func (g *gauge) removeStaleSeries(staleTimeMs int64) {
	g.seriesMtx.Lock()
	defer g.seriesMtx.Unlock()

	for hash, s := range g.series {
		if s.lastUpdated.Load() < staleTimeMs {
			delete(g.series, hash)
			g.lifecycler.OnDelete(hash, 1)
		}
	}
	g.seriesDemand.Advance()
}

func (g *gauge) deleteByHash(hash uint64) {
	g.seriesMtx.Lock()
	delete(g.series, hash)
	g.lifecycler.OnDelete(hash, 1)
	g.seriesMtx.Unlock()
}
