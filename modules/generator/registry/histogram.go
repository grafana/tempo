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

var _ metric = (*histogram)(nil)

type histogram struct {
	metricName     string
	nameCount      string
	nameSum        string
	nameBucket     string
	buckets        []float64
	bucketLabels   []string
	externalLabels map[string]string

	seriesMtx    sync.Mutex
	series       map[uint64]*histogramSeries
	seriesDemand *Cardinality

	lifecycler Limiter

	traceIDLabelName string
}

type histogramSeries struct {
	countLabels  labels.Labels
	sumLabels    labels.Labels
	bucketLabels []labels.Labels

	count *atomic.Float64
	sum   *atomic.Float64
	// buckets includes the +Inf bucket
	buckets []*atomic.Float64
	// exemplar is stored as a single traceID
	exemplars      []*atomic.String
	exemplarValues []*atomic.Float64
	lastUpdated    *atomic.Int64
	// firstSeries is used to track if this series is new to the counter.  This
	// is used to ensure that new counters being with 0, and then are incremented
	// to the desired value.  This avoids Prometheus throwing away the first
	// value in the series, due to the transition from null -> x.
	firstSeries *atomic.Bool
}

func (hs *histogramSeries) isNew() bool {
	return hs.firstSeries.Load()
}

func (hs *histogramSeries) registerSeenSeries() {
	hs.firstSeries.Store(false)
}

var (
	_ Histogram = (*histogram)(nil)
	_ metric    = (*histogram)(nil)
)

func newHistogram(name string, buckets []float64, lifecycler Limiter, traceIDLabelName string, externalLabels map[string]string, staleDuration time.Duration) *histogram {
	if traceIDLabelName == "" {
		traceIDLabelName = "traceID"
	}

	// add +Inf bucket
	buckets = append(buckets, math.Inf(1))

	bucketLabels := make([]string, len(buckets))
	for i, bucket := range buckets {
		bucketLabels[i] = formatFloat(bucket)
	}

	return &histogram{
		metricName:       name,
		nameCount:        fmt.Sprintf("%s_count", name),
		nameSum:          fmt.Sprintf("%s_sum", name),
		nameBucket:       fmt.Sprintf("%s_bucket", name),
		buckets:          buckets,
		bucketLabels:     bucketLabels,
		series:           make(map[uint64]*histogramSeries),
		seriesDemand:     NewCardinality(staleDuration, removeStaleSeriesInterval),
		lifecycler:       lifecycler,
		traceIDLabelName: traceIDLabelName,
		externalLabels:   externalLabels,
	}
}

func (h *histogram) ObserveWithExemplar(lbls labels.Labels, value float64, traceID string, multiplier float64) {
	hash := lbls.Hash()

	h.seriesDemand.Insert(hash)

	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	s, ok := h.series[hash]
	if ok {
		h.updateSeries(hash, s, value, traceID, multiplier)
		return
	}

	if !h.lifecycler.OnAdd(hash, h.activeSeriesPerHistogramSerie()) {
		return
	}

	h.series[hash] = h.newSeries(lbls, value, traceID, multiplier)
}

func (h *histogram) newSeries(lbls labels.Labels, value float64, traceID string, multiplier float64) *histogramSeries {
	newSeries := &histogramSeries{
		count:          atomic.NewFloat64(0),
		sum:            atomic.NewFloat64(0),
		buckets:        make([]*atomic.Float64, 0, len(h.buckets)),
		exemplars:      make([]*atomic.String, 0, len(h.buckets)),
		exemplarValues: make([]*atomic.Float64, 0, len(h.buckets)),
		lastUpdated:    atomic.NewInt64(0),
		firstSeries:    atomic.NewBool(true),
	}
	for i := 0; i < len(h.buckets); i++ {
		newSeries.buckets = append(newSeries.buckets, atomic.NewFloat64(0))
		newSeries.exemplars = append(newSeries.exemplars, atomic.NewString(""))
		newSeries.exemplarValues = append(newSeries.exemplarValues, atomic.NewFloat64(0))
	}

	// Precompute all labels for all sub-metrics upfront

	// Create and populate label builder
	lb := newSeriesLabelsBuilder(lbls, h.externalLabels)

	// _count
	lb.Set(labels.MetricName, h.nameCount)
	newSeries.countLabels = lb.Labels()

	// _sum
	lb.Set(labels.MetricName, h.nameSum)
	newSeries.sumLabels = lb.Labels()

	// _bucket
	lb.Set(labels.MetricName, h.nameBucket)
	for _, b := range h.bucketLabels {
		lb.Set(labels.BucketLabel, b)
		newSeries.bucketLabels = append(newSeries.bucketLabels, lb.Labels())
	}

	h.updateSeries(lbls.Hash(), newSeries, value, traceID, multiplier)

	return newSeries
}

func (h *histogram) updateSeries(hash uint64, s *histogramSeries, value float64, traceID string, multiplier float64) {
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
	h.lifecycler.OnUpdate(hash, h.activeSeriesPerHistogramSerie())
}

func (h *histogram) name() string {
	return h.metricName
}

func (h *histogram) collectMetrics(appender storage.Appender, timeMs int64) error {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	for _, s := range h.series {
		// If we are about to call Append for the first time on a series,
		// we need to first insert a 0 value to allow Prometheus to start from a non-null value.
		if s.isNew() {
			// We set the timestamp of the init serie at the end of the previous minute, that way we ensure it ends in a
			// different aggregation interval to avoid be downsampled.
			endOfLastMinuteMs := getEndOfLastMinuteMs(timeMs)
			_, err := appender.Append(0, s.countLabels, endOfLastMinuteMs, 0)
			if err != nil {
				return err
			}
		}

		// sum
		_, err := appender.Append(0, s.sumLabels, timeMs, s.sum.Load())
		if err != nil {
			return err
		}

		// count
		_, err = appender.Append(0, s.countLabels, timeMs, s.count.Load())
		if err != nil {
			return err
		}

		// bucket
		for i := range h.bucketLabels {
			if s.isNew() {
				endOfLastMinuteMs := getEndOfLastMinuteMs(timeMs)
				_, err = appender.Append(0, s.bucketLabels[i], endOfLastMinuteMs, 0)
				if err != nil {
					return err
				}
			}
			ref, err := appender.Append(0, s.bucketLabels[i], timeMs, s.buckets[i].Load())
			if err != nil {
				return err
			}

			ex := s.exemplars[i].Load()
			if ex != "" {

				lbls := []labels.Label{{
					Name:  h.traceIDLabelName,
					Value: ex,
				}}

				_, err = appender.AppendExemplar(ref, s.bucketLabels[i], exemplar.Exemplar{
					Labels: labels.New(lbls...),
					Value:  s.exemplarValues[i].Load(),
					Ts:     timeMs,
				})
				if err != nil {
					return err
				}
			}
			// clear the exemplar so we don't emit it again
			s.exemplars[i].Store("")
		}

		if s.isNew() {
			s.registerSeenSeries()
		}
	}

	return nil
}

func (h *histogram) countActiveSeries() int {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	return len(h.series) * int(h.activeSeriesPerHistogramSerie())
}

func (h *histogram) countSeriesDemand() int {
	est := h.seriesDemand.Estimate()
	return int(est) * int(h.activeSeriesPerHistogramSerie())
}

func (h *histogram) removeStaleSeries(staleTimeMs int64) {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	for hash, s := range h.series {
		if s.lastUpdated.Load() < staleTimeMs {
			delete(h.series, hash)
			h.lifecycler.OnDelete(hash, h.activeSeriesPerHistogramSerie())
		}
	}
	h.seriesDemand.Advance()
}

func (h *histogram) activeSeriesPerHistogramSerie() uint32 {
	// sum + count + #buckets (including +Inf)
	return uint32(2 + len(h.buckets))
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
