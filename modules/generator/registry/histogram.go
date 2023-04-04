package registry

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

type histogram struct {
	metricName   string
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
	// buckets includes the +Inf bucket
	buckets []*atomic.Float64
	// exemplar is stored as a single traceID
	exemplars      []*atomic.String
	exemplarValues []*atomic.Float64
	lastUpdated    *atomic.Int64
}

var _ Histogram = (*histogram)(nil)
var _ metric = (*histogram)(nil)

func newHistogram(name string, labels []string, buckets []float64, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32)) *histogram {
	if onAddSeries == nil {
		onAddSeries = func(uint32) bool {
			return true
		}
	}
	if onRemoveSeries == nil {
		onRemoveSeries = func(uint32) {}
	}

	// add +Inf bucket
	buckets = append(buckets, math.Inf(1))

	bucketLabels := make([]string, len(buckets))
	for i, bucket := range buckets {
		bucketLabels[i] = formatFloat(bucket)
	}

	return &histogram{
		metricName:    name,
		nameCount:     fmt.Sprintf("%s_count", name),
		nameSum:       fmt.Sprintf("%s_sum", name),
		nameBucket:    fmt.Sprintf("%s_bucket", name),
		labels:        labels,
		buckets:       buckets,
		bucketLabels:  bucketLabels,
		series:        make(map[uint64]*histogramSeries),
		onAddSerie:    onAddSeries,
		onRemoveSerie: onRemoveSeries,
	}
}

func (h *histogram) ObserveWithExemplar(labelValues *LabelValues, value float64, traceID string, multiplier float64) {
	if len(h.labels) != len(labelValues.getValues()) {
		panic(fmt.Sprintf("length of given label values does not match with labels, labels: %v, label values: %v", h.labels, labelValues))
	}

	hash := labelValues.getHash()

	h.seriesMtx.RLock()
	s, ok := h.series[hash]
	h.seriesMtx.RUnlock()

	if ok {
		h.updateSeries(s, value, traceID, multiplier)
		return
	}

	if !h.onAddSerie(h.activeSeriesPerHistogramSerie()) {
		return
	}

	newSeries := h.newSeries(labelValues, value, traceID, multiplier)

	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	s, ok = h.series[hash]
	if ok {
		h.updateSeries(s, value, traceID, multiplier)
		return
	}
	h.series[hash] = newSeries
}

func (h *histogram) newSeries(labelValues *LabelValues, value float64, traceID string, multiplier float64) *histogramSeries {
	newSeries := &histogramSeries{
		labelValues: labelValues.getValuesCopy(),
		count:       atomic.NewFloat64(0),
		sum:         atomic.NewFloat64(0),
		buckets:     nil,
		exemplars:   nil,
		lastUpdated: atomic.NewInt64(0),
	}
	for i := 0; i < len(h.buckets); i++ {
		newSeries.buckets = append(newSeries.buckets, atomic.NewFloat64(0))
		newSeries.exemplars = append(newSeries.exemplars, atomic.NewString(""))
		newSeries.exemplarValues = append(newSeries.exemplarValues, atomic.NewFloat64(0))
	}

	h.updateSeries(newSeries, value, traceID, multiplier)

	return newSeries
}

func (h *histogram) updateSeries(s *histogramSeries, value float64, traceID string, multiplier float64) {
	s.count.Add(1 * multiplier)
	s.sum.Add(value * multiplier)

	for i, bucket := range h.buckets {
		if value <= bucket {
			s.buckets[i].Add(1 * multiplier)
		}
	}

	bucket := sort.SearchFloat64s(h.buckets, value)
	s.exemplars[bucket].Store(traceID)
	s.exemplarValues[bucket].Store(value)

	s.lastUpdated.Store(time.Now().UnixMilli())
}

func (h *histogram) name() string {
	return h.metricName
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
		_, err = appender.Append(0, lb.Labels(nil), timeMs, s.sum.Load())
		if err != nil {
			return
		}

		// count
		lb.Set(labels.MetricName, h.nameCount)
		_, err = appender.Append(0, lb.Labels(nil), timeMs, s.count.Load())
		if err != nil {
			return
		}

		// bucket
		lb.Set(labels.MetricName, h.nameBucket)

		for i, bucketLabel := range h.bucketLabels {
			lb.Set(labels.BucketLabel, bucketLabel)
			ref, err := appender.Append(0, lb.Labels(nil), timeMs, s.buckets[i].Load())
			if err != nil {
				return activeSeries, err
			}

			ex := s.exemplars[i].Load()
			if ex != "" {
				_, err = appender.AppendExemplar(ref, lb.Labels(nil), exemplar.Exemplar{
					Labels: []labels.Label{{
						Name:  "traceID",
						Value: ex,
					}},
					Value: s.exemplarValues[i].Load(),
					Ts:    timeMs,
				})
				if err != nil {
					return activeSeries, err
				}
			}
			// clear the exemplar so we don't emit it again
			s.exemplars[i].Store("")
		}

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
	// sum + count + #buckets
	return uint32(2 + len(h.buckets))
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
