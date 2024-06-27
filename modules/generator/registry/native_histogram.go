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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type nativeHistogram struct {
	metricName string

	promHistogramInit sync.Once
	promHistogram     *prometheus.HistogramVec

	onAddSerie    func(count uint32) bool
	onRemoveSerie func(count uint32)

	labelNames []string
	buckets    []float64

	traceIDLabelName string
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
		metricName: name,
		// we delay set up until first call to ObserveWithExemplar
		promHistogram:    nil,
		onAddSerie:       onAddSeries,
		onRemoveSerie:    onRemoveSeries,
		traceIDLabelName: traceIDLabelName,
		buckets:          buckets,
	}
}

func (h *nativeHistogram) ObserveWithExemplar(labelValueCombo *LabelValueCombo, value float64, traceID string, multiplier float64) {
	// NOTE: we make the assumption here that all LabelValueCombos have the same label names. This
	// not guaranteed at all.
	h.promHistogramInit.Do(func() {
		h.labelNames = labelValueCombo.getNames()
		h.promHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    h.name(),
			Help:    "Native histogram for metric " + h.name(),
			Buckets: h.buckets,
			// TODO check if these values are sensible and break them out
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: 15 * time.Minute,
		}, h.labelNames)
	})

	// TODO this is not correct anymore, we have now way to check whether this label combo already exists
	if !h.onAddSerie(h.activeSeriesPerHistogramSerie()) {
		return
	}

	histogram, err := h.promHistogram.GetMetricWithLabelValues(labelValueCombo.getValues()...)
	if err != nil {
		// we gambled and we lost - a processor has been setting dynamic values
		panic(err)
	}

	// observe with exemplar
	histogram.(prometheus.ExemplarObserver).ObserveWithExemplar(value, map[string]string{h.traceIDLabelName: traceID})

	// if multiplier is set, observe some more times
	if multiplier > 1 {
		for i := 0.0; i < multiplier-1; i++ {
			histogram.Observe(value)
		}
	}

}

func (h *nativeHistogram) name() string {
	return h.metricName
}

func (h *nativeHistogram) collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	lbls := make(labels.Labels, 1+len(externalLabels)+len(h.labelNames))
	lb := labels.NewBuilder(lbls)

	lb.Set(labels.MetricName, h.metricName)

	// set external labels
	for name, value := range externalLabels {
		lb.Set(name, value)
	}

	metricChan := make(chan prometheus.Metric)
	go func() {
		h.promHistogram.Collect(metricChan)
		close(metricChan)
	}()

	for m := range metricChan {
		// Extract histogram
		encodedMetric := &dto.Metric{}

		// Encode to protobuf representation
		err = m.Write(encodedMetric)
		if err != nil {
			return activeSeries, err
		}
		histogram := encodedMetric.GetHistogram()

		// Set series-specific labels
		for _, pair := range encodedMetric.GetLabel() {
			lb.Set(pair.GetName(), pair.GetValue())
		}

		// *** Classic histogram

		// sum
		lb.Set(labels.MetricName, h.metricName+"_sum")
		_, err = appender.Append(0, lb.Labels(), timeMs, histogram.GetSampleSum())
		if err != nil {
			return activeSeries, err
		}
		activeSeries++

		// count
		lb.Set(labels.MetricName, h.metricName+"_count")
		_, err = appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(histogram.GetSampleCountFloat(), histogram.GetSampleCount()))
		if err != nil {
			return activeSeries, err
		}
		activeSeries++

		// bucket
		lb.Set(labels.MetricName, h.metricName+"_bucket")

		// the Prometheus histogram will sometimes add the +Inf bucket, it depends on whether there is an exemplar
		// for that bucket or not. To avoid adding it twice, keep track of it with this boolean.
		infBucketWasAdded := false

		for _, bucket := range histogram.Bucket {
			// add "le" label
			lb.Set(labels.BucketLabel, formatFloat(bucket.GetUpperBound()))

			if bucket.GetUpperBound() == math.Inf(1) {
				infBucketWasAdded = true
			}

			ref, appendErr := appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(bucket.GetCumulativeCountFloat(), bucket.GetCumulativeCount()))
			if appendErr != nil {
				return activeSeries, appendErr
			}
			activeSeries++

			if bucket.Exemplar != nil && len(bucket.Exemplar.Label) > 0 {
				// TODO are we appending the same exemplar twice?
				_, err = appender.AppendExemplar(ref, lb.Labels(), exemplar.Exemplar{
					Labels: convertLabelPairToLabels(bucket.Exemplar.GetLabel()),
					Value:  bucket.Exemplar.GetValue(),
					Ts:     timeMs,
				})
				if err != nil {
					return activeSeries, err
				}
				bucket.Exemplar.Reset()
			}
		}

		if !infBucketWasAdded {
			// Add +Inf bucket
			lb.Set(labels.BucketLabel, "+Inf")

			_, err = appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(histogram.GetSampleCountFloat(), histogram.GetSampleCount()))
			if err != nil {
				return activeSeries, err
			}
			activeSeries++
		}

		// drop "le" label again
		lb.Del(labels.BucketLabel)

		// *** Native histogram

		// Decode to Prometheus representation
		hist := promhistogram.Histogram{
			Schema:        histogram.GetSchema(),
			Count:         histogram.GetSampleCount(),
			Sum:           histogram.GetSampleSum(),
			ZeroThreshold: histogram.GetZeroThreshold(),
			ZeroCount:     histogram.GetZeroCount(),
		}
		if len(histogram.PositiveSpan) > 0 {
			hist.PositiveSpans = make([]promhistogram.Span, len(histogram.PositiveSpan))
			for i, span := range histogram.PositiveSpan {
				hist.PositiveSpans[i] = promhistogram.Span{
					Offset: span.GetOffset(),
					Length: span.GetLength(),
				}
			}
		}
		hist.PositiveBuckets = histogram.PositiveDelta
		if len(histogram.NegativeSpan) > 0 {
			hist.NegativeSpans = make([]promhistogram.Span, len(histogram.NegativeSpan))
			for i, span := range histogram.NegativeSpan {
				hist.NegativeSpans[i] = promhistogram.Span{
					Offset: span.GetOffset(),
					Length: span.GetLength(),
				}
			}
		}
		hist.NegativeBuckets = histogram.NegativeDelta

		lb.Set(labels.MetricName, h.metricName)
		_, err = appender.AppendHistogram(0, lb.Labels(), timeMs, &hist, nil)
		if err != nil {
			return activeSeries, err
		}

		// TODO: impact on active series from appending a histogram?
		activeSeries += 0

		if len(histogram.Exemplars) > 0 {
			for _, encodedExemplar := range histogram.Exemplars {

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
	// TODO can't really do much here now 🤷
	//  possible solution: we can keep a map of label values + their last updated time and delete them from HistogramVec
	//  con: we need to keep a map again with locking
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
