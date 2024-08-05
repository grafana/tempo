package registry

import (
	"math"
	"sync"
	"time"

	"github.com/grafana/tempo/modules/overrides"
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

	// Can be "native", classic", "both" to determine which histograms to
	// generate.  A diff in the configured value on the processors will cause a
	// reload of the process, and a new instance of the histogram to be created.
	histogramOverride string
}

type nativeHistogramSeries struct {
	// labels should not be modified after creation
	labels        LabelPair
	promHistogram prometheus.Histogram
	lastUpdated   int64
	histogram     *dto.Histogram
}

var (
	_ Histogram = (*nativeHistogram)(nil)
	_ metric    = (*nativeHistogram)(nil)
)

func newNativeHistogram(name string, buckets []float64, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32), traceIDLabelName string, histogramOverride string) *nativeHistogram {
	if onAddSeries == nil {
		onAddSeries = func(uint32) bool {
			return true
		}
	}
	if onRemoveSeries == nil {
		onRemoveSeries = func(uint32) {}
	}

	if histogramOverride == "" {
		histogramOverride = "native"
	}

	if traceIDLabelName == "" {
		traceIDLabelName = "traceID"
	}

	return &nativeHistogram{
		metricName:        name,
		series:            make(map[uint64]*nativeHistogramSeries),
		onAddSerie:        onAddSeries,
		onRemoveSerie:     onRemoveSeries,
		traceIDLabelName:  traceIDLabelName,
		buckets:           buckets,
		histogramOverride: histogramOverride,
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
		lastUpdated: 0,
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
	s.lastUpdated = time.Now().UnixMilli()
}

func (h *nativeHistogram) name() string {
	return h.metricName
}

func (h *nativeHistogram) collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	lbls := make(labels.Labels, 1+len(externalLabels))
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
		err = s.promHistogram.Write(encodedMetric)
		if err != nil {
			return activeSeries, err
		}

		// NOTE: Store the encoded histogram here so we can keep track of the exemplars
		// that have been sent.  The value is updated here, but the pointers remain
		// the same, and so Reset() call below can be used to clear the exemplars.
		s.histogram = encodedMetric.GetHistogram()

		// If we are in "both" or "classic" mode, also emit classic histograms.
		if overrides.HasClassicHistograms(h.histogramOverride) {
			classicSeries, classicErr := h.classicHistograms(appender, lb, timeMs, s)
			if classicErr != nil {
				return activeSeries, classicErr
			}
			activeSeries += classicSeries
		}

		// If we are in "both" or "native" mode, also emit native histograms.
		if overrides.HasNativeHistograms(h.histogramOverride) {
			nativeErr := h.nativeHistograms(appender, lb, timeMs, s)
			if nativeErr != nil {
				return activeSeries, nativeErr
			}
		}

		// TODO: impact on active series from appending a histogram?
		activeSeries += 0

		if len(s.histogram.Exemplars) > 0 {
			for _, encodedExemplar := range s.histogram.Exemplars {

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
		if s.lastUpdated < staleTimeMs {
			delete(h.series, hash)
			h.onRemoveSerie(h.activeSeriesPerHistogramSerie())
		}
	}
}

func (h *nativeHistogram) activeSeriesPerHistogramSerie() uint32 {
	// TODO can we estimate this?
	return 1
}

func (h *nativeHistogram) nativeHistograms(appender storage.Appender, lb *labels.Builder, timeMs int64, s *nativeHistogramSeries) (err error) {
	// Decode to Prometheus representation
	hist := promhistogram.Histogram{
		Schema:        s.histogram.GetSchema(),
		Count:         s.histogram.GetSampleCount(),
		Sum:           s.histogram.GetSampleSum(),
		ZeroThreshold: s.histogram.GetZeroThreshold(),
		ZeroCount:     s.histogram.GetZeroCount(),
	}
	if len(s.histogram.PositiveSpan) > 0 {
		hist.PositiveSpans = make([]promhistogram.Span, len(s.histogram.PositiveSpan))
		for i, span := range s.histogram.PositiveSpan {
			hist.PositiveSpans[i] = promhistogram.Span{
				Offset: span.GetOffset(),
				Length: span.GetLength(),
			}
		}
	}
	hist.PositiveBuckets = s.histogram.PositiveDelta
	if len(s.histogram.NegativeSpan) > 0 {
		hist.NegativeSpans = make([]promhistogram.Span, len(s.histogram.NegativeSpan))
		for i, span := range s.histogram.NegativeSpan {
			hist.NegativeSpans[i] = promhistogram.Span{
				Offset: span.GetOffset(),
				Length: span.GetLength(),
			}
		}
	}
	hist.NegativeBuckets = s.histogram.NegativeDelta

	lb.Set(labels.MetricName, h.metricName)
	_, err = appender.AppendHistogram(0, lb.Labels(), timeMs, &hist, nil)
	if err != nil {
		return err
	}

	return
}

func (h *nativeHistogram) classicHistograms(appender storage.Appender, lb *labels.Builder, timeMs int64, s *nativeHistogramSeries) (activeSeries int, err error) {
	// sum
	lb.Set(labels.MetricName, h.metricName+"_sum")
	_, err = appender.Append(0, lb.Labels(), timeMs, s.histogram.GetSampleSum())
	if err != nil {
		return activeSeries, err
	}
	activeSeries++

	// count
	lb.Set(labels.MetricName, h.metricName+"_count")
	_, err = appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(s.histogram.GetSampleCountFloat(), s.histogram.GetSampleCount()))
	if err != nil {
		return activeSeries, err
	}
	activeSeries++

	// bucket
	lb.Set(labels.MetricName, h.metricName+"_bucket")

	// the Prometheus histogram will sometimes add the +Inf bucket, it depends on whether there is an exemplar
	// for that bucket or not. To avoid adding it twice, keep track of it with this boolean.
	infBucketWasAdded := false

	for _, bucket := range s.histogram.Bucket {
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

		_, err = appender.Append(0, lb.Labels(), timeMs, getIfGreaterThenZeroOr(s.histogram.GetSampleCountFloat(), s.histogram.GetSampleCount()))
		if err != nil {
			return activeSeries, err
		}
		activeSeries++
	}

	// drop "le" label again
	lb.Del(labels.BucketLabel)

	return
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
