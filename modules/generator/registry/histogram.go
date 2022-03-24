package registry

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

type histogram struct {
	nameCount    string
	nameSum      string
	nameBucket   string
	labels       []string
	buckets      []float64
	bucketLabels []string

	// seriesMtx is used to sync modifications to the map, not to the data in series
	seriesMtx sync.RWMutex
	series    map[uint64]*histogramSeries

	onAddSerie    func(count uint32) bool
	onRemoveSerie func(count uint32)
}

type histogramSeries struct {
	// labelValues should not be modified after creation
	labelValues []string
	count       *atomic.Float64
	sum         *atomic.Float64
	buckets     []*atomic.Float64
	bucketInf   *atomic.Float64
	lastUpdated *atomic.Int64
}

var _ Histogram = (*histogram)(nil)
var _ metric = (*histogram)(nil)

func newHistogram(name string, labels []string, buckets []float64) *histogram {
	bucketLabels := make([]string, len(buckets))
	for i, bucket := range buckets {
		bucketLabels[i] = formatFloat(bucket)
	}

	return &histogram{
		nameCount:    fmt.Sprintf("%s_count", name),
		nameSum:      fmt.Sprintf("%s_sum", name),
		nameBucket:   fmt.Sprintf("%s_bucket", name),
		labels:       labels,
		buckets:      buckets,
		bucketLabels: bucketLabels,
		series:       make(map[uint64]*histogramSeries),
	}
}

func (h *histogram) setCallbacks(onAddSeries func(count uint32) bool, onRemoveSeries func(count uint32)) {
	h.onAddSerie = onAddSeries
	h.onRemoveSerie = onRemoveSeries
}

func (h *histogram) Observe(labelValues *LabelValues, value float64) {
	if len(h.labels) != len(labelValues.getValues()) {
		panic(fmt.Sprintf("length of given label values does not match with labels, labels: %v, label values: %v", h.labels, labelValues))
	}

	hash := labelValues.getHash()

	h.seriesMtx.RLock()
	s, ok := h.series[hash]
	h.seriesMtx.RUnlock()

	if ok {
		h.updateSeries(s, value)
		return
	}

	if !h.onAddSerie(h.activeSeriesPerHistogramSerie()) {
		return
	}

	newSeries := h.newSeries(labelValues, value)

	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	s, ok = h.series[hash]
	if ok {
		h.updateSeries(s, value)
		return
	}
	h.series[hash] = newSeries
}

func (h *histogram) newSeries(labelValues *LabelValues, value float64) *histogramSeries {
	newSeries := &histogramSeries{
		labelValues: labelValues.getValuesCopy(),
		count:       atomic.NewFloat64(1),
		sum:         atomic.NewFloat64(value),
		buckets:     nil,
		bucketInf:   atomic.NewFloat64(1),
		lastUpdated: atomic.NewInt64(time.Now().UnixMilli()),
	}
	for _, bucket := range h.buckets {
		if value <= bucket {
			newSeries.buckets = append(newSeries.buckets, atomic.NewFloat64(1))
		} else {
			newSeries.buckets = append(newSeries.buckets, atomic.NewFloat64(0))
		}
	}
	return newSeries
}

func (h *histogram) updateSeries(s *histogramSeries, value float64) {
	s.count.Add(1)
	s.sum.Add(value)
	for i, bucket := range h.buckets {
		if value <= bucket {
			s.buckets[i].Add(1)
		}
	}
	s.bucketInf.Add(1)
	s.lastUpdated.Store(time.Now().UnixMilli())
}

func (h *histogram) collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	h.seriesMtx.RLock()
	defer h.seriesMtx.RUnlock()

	activeSeries = len(h.series) * int(h.activeSeriesPerHistogramSerie())

	lbls := make(labels.Labels, 1+len(externalLabels)+len(h.labels)+1)
	lb := labels.NewBuilder(lbls)

	// set external labels
	for name, value := range externalLabels {
		lb.Set(name, value)
	}

	for _, s := range h.series {
		// set series-specific labels
		for i, name := range h.labels {
			lb.Set(name, s.labelValues[i])
		}

		// sum
		lb.Set(labels.MetricName, h.nameSum)
		_, err = appender.Append(0, lb.Labels(), timeMs, s.sum.Load())
		if err != nil {
			return
		}

		// count
		lb.Set(labels.MetricName, h.nameCount)
		_, err = appender.Append(0, lb.Labels(), timeMs, s.count.Load())
		if err != nil {
			return
		}

		// bucket
		lb.Set(labels.MetricName, h.nameBucket)

		for i, bucketLabel := range h.bucketLabels {
			lb.Set(labels.BucketLabel, bucketLabel)
			_, err = appender.Append(0, lb.Labels(), timeMs, s.buckets[i].Load())
			if err != nil {
				return
			}
		}

		lb.Set(labels.BucketLabel, "+Inf")
		_, err = appender.Append(0, lb.Labels(), timeMs, s.bucketInf.Load())
		if err != nil {
			return
		}

		// TODO support exemplars

		lb.Del(labels.BucketLabel)
	}

	return
}

func (h *histogram) removeStaleSeries(staleTimeMs int64) {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	for hash, s := range h.series {
		if s.lastUpdated.Load() < staleTimeMs {
			delete(h.series, hash)
			h.onRemoveSerie(h.activeSeriesPerHistogramSerie())
		}
	}
}

func (h *histogram) activeSeriesPerHistogramSerie() uint32 {
	// sum + count + +Inf + #buckets
	return uint32(3 + len(h.buckets))
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
