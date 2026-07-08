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

	tempo_util "github.com/grafana/tempo/pkg/util"
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

// histogramSeries state is serialized by histogram.seriesMtx: the observe,
// collect, and stale-removal paths all take the full mutex, so all fields are
// plain values.
type histogramSeries struct {
	countLabels  labels.Labels
	sumLabels    labels.Labels
	bucketLabels []labels.Labels

	count float64
	sum   float64
	// buckets includes the +Inf bucket
	buckets []float64
	// exemplars stores either a hex-encoded string traceID or a raw <=16 byte
	// traceID per bucket.
	exemplars      []histogramExemplar
	exemplarValues []float64
	lastUpdated    int64
	// firstSeries is used to track if this series is new to the counter.  This
	// is used to ensure that new counters being with 0, and then are incremented
	// to the desired value.  This avoids Prometheus throwing away the first
	// value in the series, due to the transition from null -> x.
	firstSeries bool
}

type histogramExemplar struct {
	traceID      string
	traceIDBytes [16]byte
	traceIDLen   int
}

func newHistogramStringExemplar(traceID string) histogramExemplar {
	return histogramExemplar{traceID: traceID}
}

func newHistogramTraceIDBytesExemplar(traceID []byte) histogramExemplar {
	if len(traceID) == 0 {
		return histogramExemplar{}
	}
	if len(traceID) > 16 {
		return histogramExemplar{traceID: tempo_util.TraceIDToHexString(traceID)}
	}

	ex := histogramExemplar{traceIDLen: len(traceID)}
	copy(ex.traceIDBytes[:], traceID)
	return ex
}

func (ex histogramExemplar) string() string {
	if ex.traceIDLen == 0 {
		return ex.traceID
	}
	return tempo_util.TraceIDToHexString(ex.traceIDBytes[:ex.traceIDLen])
}

func (hs *histogramSeries) isNew() bool {
	return hs.firstSeries
}

func (hs *histogramSeries) registerSeenSeries() {
	hs.firstSeries = false
}

var (
	_ Histogram = (*histogram)(nil)
	_ metric    = (*histogram)(nil)
)

func newHistogram(name string, buckets []float64, lifecycler Limiter, traceIDLabelName string, externalLabels map[string]string, staleDuration time.Duration) *histogram {
	if traceIDLabelName == "" {
		traceIDLabelName = "traceID"
	}

	// Defensively copy and sort the buckets in ascending order. updateSeries
	// counts buckets with sort.SearchFloat64s, which requires sorted input, and
	// classic histograms require monotonic le buckets anyway. The override paths
	// validate this, but a static histogram_buckets in the generator config is
	// not validated, so sorting here keeps bucket counts correct on every path.
	// Copying avoids mutating the caller's slice.
	sorted := make([]float64, len(buckets), len(buckets)+1)
	copy(sorted, buckets)
	sort.Float64s(sorted)

	// add +Inf bucket (always the largest value, so it stays last after sorting)
	sorted = append(sorted, math.Inf(1))
	buckets = sorted

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
	h.observeWithExemplarWithHashAt(lbls, lbls.Hash(), value, newHistogramStringExemplar(traceID), multiplier, time.Now().UnixMilli())
}

func (h *histogram) ObserveWithExemplarTraceIDBytesWithHashAt(lbls labels.Labels, hash uint64, value float64, traceID []byte, multiplier float64, timeMs int64) {
	h.observeWithExemplarWithHashAt(lbls, hash, value, newHistogramTraceIDBytesExemplar(traceID), multiplier, timeMs)
}

func (h *histogram) observeWithExemplarWithHashAt(lbls labels.Labels, hash uint64, value float64, ex histogramExemplar, multiplier float64, timeMs int64) {
	h.seriesDemand.Insert(hash)

	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	s, lbls, hash := resolveSeries(h.series, hash, lbls, h.lifecycler, h.activeSeriesPerHistogramSerie())
	if s != nil {
		h.updateSeries(hash, s, value, ex, multiplier, timeMs)
		return
	}
	h.series[hash] = h.newSeries(lbls, hash, value, ex, multiplier, timeMs)
}

func (h *histogram) newSeries(lbls labels.Labels, hash uint64, value float64, ex histogramExemplar, multiplier float64, timeMs int64) *histogramSeries {
	newSeries := &histogramSeries{
		buckets:        make([]float64, len(h.buckets)),
		exemplars:      make([]histogramExemplar, len(h.buckets)),
		exemplarValues: make([]float64, len(h.buckets)),
		firstSeries:    true,
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

	h.updateSeries(hash, newSeries, value, ex, multiplier, timeMs)

	return newSeries
}

func (h *histogram) updateSeries(hash uint64, s *histogramSeries, value float64, ex histogramExemplar, multiplier float64, timeMs int64) {
	s.count += multiplier
	s.sum += value * multiplier

	bucket := sort.SearchFloat64s(h.buckets, value)
	for i := bucket; i < len(h.buckets); i++ {
		s.buckets[i] += multiplier
	}

	s.exemplars[bucket] = ex
	s.exemplarValues[bucket] = value

	s.lastUpdated = timeMs
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
			if err != nil && !isOutOfOrderError(err) {
				return err
			}
		}

		// sum
		_, err := appender.Append(0, s.sumLabels, timeMs, s.sum)
		if err != nil {
			return err
		}

		// count
		_, err = appender.Append(0, s.countLabels, timeMs, s.count)
		if err != nil {
			return err
		}

		// bucket
		for i := range h.bucketLabels {
			if s.isNew() {
				endOfLastMinuteMs := getEndOfLastMinuteMs(timeMs)
				_, err = appender.Append(0, s.bucketLabels[i], endOfLastMinuteMs, 0)
				if err != nil && !isOutOfOrderError(err) {
					return err
				}
			}
			ref, err := appender.Append(0, s.bucketLabels[i], timeMs, s.buckets[i])
			if err != nil {
				return err
			}

			ex := s.exemplars[i]
			traceID := ex.string()
			if traceID != "" {

				lbls := []labels.Label{{
					Name:  h.traceIDLabelName,
					Value: traceID,
				}}

				_, err = appender.AppendExemplar(ref, s.bucketLabels[i], exemplar.Exemplar{
					Labels: labels.New(lbls...),
					Value:  s.exemplarValues[i],
					Ts:     timeMs,
				})
				if err != nil {
					return err
				}
			}
			// clear the exemplar so we don't emit it again
			s.exemplars[i] = histogramExemplar{}
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
		if s.lastUpdated < staleTimeMs {
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
