package registry

import (
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/model/exemplar"
	promhistogram "github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type nativeHistogram struct {
	metricName string

	// TODO we can also switch to a HistrogramVec and let prometheus handle the labels. This would remove the series map
	//  and all locking around it.
	//  Downside: you need to list labels at creation time while our interfaces only pass labels at observe time, this
	//  will requires a bigger refactor, maybe something for a second pass?
	//  Might break processors that have variable amount of labels...
	//  promHistogram prometheus.HistogramVec

	seriesMtx sync.Mutex
	series    map[uint64]*nativeHistogramSeries

	onAddSerie    func(count uint32) bool
	onRemoveSerie func(count uint32)

	buckets []float64

	traceIDLabelName string
}

type nativeHistogramSeries struct {
	// labels should not be modified after creation
	labels        LabelPair
	promHistogram prometheus.Histogram
	lastUpdated   *atomic.Int64
}

var (
	_ Histogram = (*nativeHistogram)(nil)
	_ metric    = (*nativeHistogram)(nil)
)

func newNativeHistogram(name string, buckets []float64, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32), traceIDLabelName string) *nativeHistogram {
	if onAddSeries == nil {
		onAddSeries = func(uint32) bool {
			return true
		}
	}
	if onRemoveSeries == nil {
		onRemoveSeries = func(uint32) {}
	}

	if traceIDLabelName == "" {
		traceIDLabelName = "traceID"
	}

	return &nativeHistogram{
		metricName:       name,
		series:           make(map[uint64]*nativeHistogramSeries),
		onAddSerie:       onAddSeries,
		onRemoveSerie:    onRemoveSeries,
		traceIDLabelName: traceIDLabelName,
		buckets:          buckets,
	}
}

func (h *nativeHistogram) ObserveWithExemplar(labelValueCombo *LabelValueCombo, value float64, traceID string, multiplier float64) {
	hash := labelValueCombo.getHash()

	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	s, ok := h.series[hash]
	if ok {
		h.updateSeries(s, value, traceID, multiplier)
		return
	}

	if !h.onAddSerie(h.activeSeriesPerHistogramSerie()) {
		return
	}

	newSeries := h.newSeries(labelValueCombo, value, traceID, multiplier)
	h.series[hash] = newSeries
}

func (h *nativeHistogram) newSeries(labelValueCombo *LabelValueCombo, value float64, traceID string, multiplier float64) *nativeHistogramSeries {
	newSeries := &nativeHistogramSeries{
		// TODO move these labels in HistogramOpts.ConstLabels?
		labels: labelValueCombo.getLabelPair(),
		promHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    h.name(),
			Help:    "Native histogram for metric " + h.name(),
			Buckets: h.buckets,
			// TODO check if these values are sensible and break them out
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: 15 * time.Minute,
		}),
		lastUpdated: atomic.NewInt64(0),
	}

	h.updateSeries(newSeries, value, traceID, multiplier)

	return newSeries
}

func (h *nativeHistogram) updateSeries(s *nativeHistogramSeries, value float64, traceID string, multiplier float64) {
	for i := 0.0; i < multiplier; i++ {
		s.promHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
			value,
			map[string]string{h.traceIDLabelName: traceID},
		)
	}
	s.lastUpdated.Store(time.Now().UnixMilli())
}

func (h *nativeHistogram) name() string {
	return h.metricName
}

func (h *nativeHistogram) collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	labelsCount := 0
	if h.series[0] != nil {
		labelsCount = len(h.series[0].labels.names)
	}
	lbls := make(labels.Labels, 1+len(externalLabels)+labelsCount)
	lb := labels.NewBuilder(lbls)
	activeSeries = 0

	lb.Set(labels.MetricName, h.metricName)

	// set external labels
	for name, value := range externalLabels {
		lb.Set(name, value)
	}

	for _, s := range h.series {

		// Set series-specific labels
		for i, name := range s.labels.names {
			lb.Set(name, s.labels.values[i])
		}

		// Extract histogram
		encodedMetric := &dto.Metric{}

		// Encode to protobuf representation
		err := s.promHistogram.Write(encodedMetric)
		if err != nil {
			return activeSeries, err
		}
		encodedHistogram := encodedMetric.GetHistogram()

		// *** Classic histogram

		// sum
		lb.Set(labels.MetricName, h.metricName+"_sum")
		_, err = appender.Append(0, lb.Labels(), timeMs, encodedHistogram.GetSampleSum())
		if err != nil {
			return activeSeries, err
		}
		activeSeries += 1

		// count
		lb.Set(labels.MetricName, h.metricName+"_count")
		_, err = appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(encodedHistogram.GetSampleCountFloat(), encodedHistogram.GetSampleCount()))
		if err != nil {
			return activeSeries, err
		}
		activeSeries += 1

		// bucket
		lb.Set(labels.MetricName, h.metricName+"_bucket")

		// the Prometheus histogram will sometimes add the +Inf bucket, it depends on whether there is an exemplar
		// for that bucket or not. To avoid adding it twice, keep track of it with this boolean.
		infBucketWasAdded := false

		for _, bucket := range encodedHistogram.Bucket {
			// add "le" label
			lb.Set(labels.BucketLabel, formatFloat(bucket.GetUpperBound()))

			if bucket.GetUpperBound() == math.Inf(1) {
				infBucketWasAdded = true
			}

			ref, err := appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(bucket.GetCumulativeCountFloat(), bucket.GetCumulativeCount()))
			if err != nil {
				return activeSeries, err
			}
			activeSeries += 1

			ex := bucket.Exemplar
			if ex != nil {
				// TODO are we appending the same exemplar twice?
				_, err = appender.AppendExemplar(ref, lb.Labels(), exemplar.Exemplar{
					Labels: convertLabelPairToLabels(ex.GetLabel()),
					Value:  ex.GetValue(),
					Ts:     timeMs,
				})
				if err != nil {
					return activeSeries, err
				}
			}
		}

		if !infBucketWasAdded {
			// Add +Inf bucket
			lb.Set(labels.BucketLabel, "+Inf")

			_, err = appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(encodedHistogram.GetSampleCountFloat(), encodedHistogram.GetSampleCount()))
			if err != nil {
				return activeSeries, err
			}
			activeSeries += 1
		}

		// drop "le" label again
		lb.Del(labels.BucketLabel)

		// *** Native histogram

		// Decode to Prometheus representation
		hist := promhistogram.Histogram{
			Schema:        encodedHistogram.GetSchema(),
			Count:         encodedHistogram.GetSampleCount(),
			Sum:           encodedHistogram.GetSampleSum(),
			ZeroThreshold: encodedHistogram.GetZeroThreshold(),
			ZeroCount:     encodedHistogram.GetZeroCount(),
		}
		if len(encodedHistogram.PositiveSpan) > 0 {
			hist.PositiveSpans = make([]promhistogram.Span, len(encodedHistogram.PositiveSpan))
			for i, span := range encodedHistogram.PositiveSpan {
				hist.PositiveSpans[i] = promhistogram.Span{
					Offset: span.GetOffset(),
					Length: span.GetLength(),
				}
			}
		}
		hist.PositiveBuckets = encodedHistogram.PositiveDelta
		if len(encodedHistogram.NegativeSpan) > 0 {
			hist.NegativeSpans = make([]promhistogram.Span, len(encodedHistogram.NegativeSpan))
			for i, span := range encodedHistogram.NegativeSpan {
				hist.NegativeSpans[i] = promhistogram.Span{
					Offset: span.GetOffset(),
					Length: span.GetLength(),
				}
			}
		}
		hist.NegativeBuckets = encodedHistogram.NegativeDelta

		lb.Set(labels.MetricName, h.metricName)
		_, err = appender.AppendHistogram(0, lb.Labels(), timeMs, &hist, nil)
		if err != nil {
			return activeSeries, err
		}
		// TODO impact on active series from appending a histogram?
		activeSeries += 0

		if len(encodedHistogram.Exemplars) > 0 {
			for _, encodedExemplar := range encodedHistogram.Exemplars {

				// TODO are we appending the same exemplar twice?
				e := exemplar.Exemplar{
					Labels: convertLabelPairToLabels(encodedExemplar.Label),
					Value:  encodedExemplar.GetValue(),
					Ts:     convertTimestampToMs(encodedExemplar.GetTimestamp()),
					HasTs:  true,
				}
				_, err = appender.AppendExemplar(0, lb.Labels(), e)
				if err != nil {
					return activeSeries, err
				}
			}
		}

	}

	return
}

func (h *nativeHistogram) removeStaleSeries(staleTimeMs int64) {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()
	for hash, s := range h.series {
		if s.lastUpdated.Load() < staleTimeMs {
			delete(h.series, hash)
			h.onRemoveSerie(h.activeSeriesPerHistogramSerie())
		}
	}
}

func (h *nativeHistogram) activeSeriesPerHistogramSerie() uint32 {
	// TODO can we estimate this?
	return 1
}

func convertLabelPairToLabels(lbps []*dto.LabelPair) labels.Labels {
	lbs := make([]labels.Label, len(lbps))
	for i, lbp := range lbps {
		lbs[i] = labels.Label{
			Name:  lbp.GetName(),
			Value: lbp.GetValue(),
		}
	}
	return lbs
}

func convertTimestampToMs(ts *timestamppb.Timestamp) int64 {
	if ts == nil {
		return 0
	}
	return ts.GetSeconds()*1000 + int64(ts.GetNanos()/1_000_000)
}

// getIfGreaterThenZeroOr returns v1 is if it's greater than zero, otherwise it returns v2.
func getIfGreaterThenZeroOr(v1 float64, v2 uint64) float64 {
	if v1 > 0 {
		return v1
	}
	return float64(v2)
}
