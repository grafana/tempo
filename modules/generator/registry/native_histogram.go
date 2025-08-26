package registry

import (
	"fmt"
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

	// TODO: we can also switch to a HistrogramVec and let prometheus handle the labels. This would remove the series map
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
	histogramOverride HistogramMode

	externalLabels map[string]string

	// classic
	nameCount  string
	nameSum    string
	nameBucket string
}

type nativeHistogramSeries struct {
	// labels should not be modified after creation
	lb            *labels.Builder
	labels        labels.Labels
	promHistogram prometheus.Histogram
	lastUpdated   int64
	histogram     *dto.Histogram

	// firstSeries is used to track if this series is new to the counter.
	// This is used in classic histograms to ensure that new counters begin with 0.
	// This avoids Prometheus throwing away the first value in the series,
	// due to the transition from null -> x.
	firstSeries *atomic.Bool

	// classic
	countLabels labels.Labels
	sumLabels   labels.Labels
	// bucketLabels []labels.Labels
}

func (hs *nativeHistogramSeries) isNew() bool {
	return hs.firstSeries.Load()
}

func (hs *nativeHistogramSeries) registerSeenSeries() {
	hs.firstSeries.Store(false)
}

var (
	_ Histogram = (*nativeHistogram)(nil)
	_ metric    = (*nativeHistogram)(nil)
)

func newNativeHistogram(name string, buckets []float64, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32), traceIDLabelName string, histogramOverride HistogramMode, externalLabels map[string]string) *nativeHistogram {
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
		metricName:        name,
		series:            make(map[uint64]*nativeHistogramSeries),
		onAddSerie:        onAddSeries,
		onRemoveSerie:     onRemoveSeries,
		traceIDLabelName:  traceIDLabelName,
		buckets:           buckets,
		histogramOverride: histogramOverride,
		externalLabels:    externalLabels,

		// classic
		nameCount:  fmt.Sprintf("%s_count", name),
		nameSum:    fmt.Sprintf("%s_sum", name),
		nameBucket: fmt.Sprintf("%s_bucket", name),
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

	h.series[hash] = h.newSeries(labelValueCombo, value, traceID, multiplier)
}

func (h *nativeHistogram) newSeries(labelValueCombo *LabelValueCombo, value float64, traceID string, multiplier float64) *nativeHistogramSeries {
	// Configure histogram based on mode
	//
	// Native-only mode sets buckets to nil, and uses the histogram.Exemplars slice as the native exemplar format.
	// Hybrid mode uses classic buckets and bucket.Exemplar format for compatibility.

	var buckets []float64

	// The native histogram only uses the static buckets when the classic histograms are enabled.
	hasClassic := hasClassicHistograms(h.histogramOverride)
	if hasClassic {
		// Hybrid "both" mode: include classic buckets for compatibility
		buckets = h.buckets
	}

	// Configure native histogram options based on mode
	nativeOpts := prometheus.HistogramOpts{
		Name:    h.name(),
		Help:    "Native histogram for metric " + h.name(),
		Buckets: buckets, // nil for pure native, h.buckets for hybrid
		// Native histogram parameters
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 15 * time.Minute,
	}

	if hasClassic {
		// Hybrid mode: let Prometheus decide defaults for compatibility
		nativeOpts.NativeHistogramMaxExemplars = -1 // Use default
	}

	newSeries := &nativeHistogramSeries{
		promHistogram: prometheus.NewHistogram(nativeOpts),
		lastUpdated:   0,
		firstSeries:   atomic.NewBool(true),
	}

	h.updateSeries(newSeries, value, traceID, multiplier)

	lbls := labelValueCombo.getLabelPair()
	lb := labels.NewBuilder(make(labels.Labels, 1+len(lbls.names)))

	// set series labels
	for i, name := range lbls.names {
		lb.Set(name, lbls.values[i])
	}
	// set external labels
	for name, value := range h.externalLabels {
		lb.Set(name, value)
	}

	lb.Set(labels.MetricName, h.metricName)

	newSeries.labels = lb.Labels()
	newSeries.lb = lb

	// _count
	lb.Set(labels.MetricName, h.nameCount)
	newSeries.countLabels = lb.Labels()

	// _sum
	lb.Set(labels.MetricName, h.nameSum)
	newSeries.sumLabels = lb.Labels()

	return newSeries
}

func (h *nativeHistogram) updateSeries(s *nativeHistogramSeries, value float64, traceID string, multiplier float64) {
	// Use Prometheus native exemplar handling
	exemplarObserver := s.promHistogram.(prometheus.ExemplarObserver)

	labels := prometheus.Labels{h.traceIDLabelName: traceID}

	for i := 0.0; i < multiplier; i++ {
		// Let Prometheus handle exemplars natively
		exemplarObserver.ObserveWithExemplar(value, labels)
	}

	s.lastUpdated = time.Now().UnixMilli()
}

func (h *nativeHistogram) name() string {
	return h.metricName
}

func (h *nativeHistogram) collectMetrics(appender storage.Appender, timeMs int64) (activeSeries int, err error) {
	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	activeSeries = 0

	for _, s := range h.series {
		// Extract histogram
		encodedMetric := &dto.Metric{}

		// Encode to protobuf representation
		err = s.promHistogram.Write(encodedMetric)
		if err != nil {
			return activeSeries, err
		}

		// NOTE: Store the encoded histogram here so we can keep track of the
		// exemplars that have been sent.  The value is updated here, but the
		// pointers remain the same, and so Reset() call below can be used to clear
		// the exemplars.
		s.histogram = encodedMetric.GetHistogram()

		// If we are in "both" or "classic" mode, also emit classic histograms.
		if hasClassicHistograms(h.histogramOverride) {
			classicSeries, classicErr := h.classicHistograms(appender, timeMs, s)
			if classicErr != nil {
				return activeSeries, classicErr
			}
			activeSeries += classicSeries
		}

		// If we are in "both" or "native" mode, also emit native histograms.
		if hasNativeHistograms(h.histogramOverride) {
			nativeErr := h.nativeHistograms(appender, s.labels, timeMs, s)
			if nativeErr != nil {
				return activeSeries, nativeErr
			}
		}

		// TODO: impact on active series from appending a histogram?
		activeSeries += 0
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
	// TODO: can we estimate this?
	return 1
}

func (h *nativeHistogram) nativeHistograms(appender storage.Appender, lbls labels.Labels, timeMs int64, s *nativeHistogramSeries) (err error) {
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

	// Append the native histogram
	ref, err := appender.AppendHistogram(0, lbls, timeMs, &hist, nil)
	if err != nil {
		return err
	}

	// NOTE: two exemplar formats are used:
	// Native exemplars use the histogram.Exemplars slice, which is the native format.
	// Bucket exemplars use the bucket.Exemplar field, which is the classic format.
	//
	// Use native exemplars when available, falling back to bucket exemplars

	for _, ex := range s.histogram.Exemplars {
		if ex != nil && len(ex.Label) > 0 {
			_, err = appender.AppendExemplar(ref, lbls, exemplar.Exemplar{
				Labels: convertLabelPairToLabels(ex.GetLabel()),
				Value:  ex.GetValue(),
				Ts:     timeMs,
			})
			if err != nil {
				return err
			}
		}
	}

	// NOTE: We clear the native exemplar slice to prevent accumulation, but the
	// client_golang package handles the expiration of exemplars internally, and
	// we don't have control over clearing the native histogram exemplars in the
	// same way we do for the class histogram exemplars.
	if len(s.histogram.Exemplars) > 0 {
		clear(s.histogram.Exemplars)
		s.histogram.Exemplars = s.histogram.Exemplars[:0]
	}

	// For pure native mode, never emit bucket exemplars - only native ones
	// For hybrid mode, fallback to bucket exemplars if no native exemplars available
	isHybridMode := hasClassicHistograms(h.histogramOverride)
	if isHybridMode && len(s.histogram.Exemplars) == 0 {
		// Hybrid mode fallback: use bucket exemplars if no native exemplars
		for _, bucket := range s.histogram.Bucket {
			if bucket.Exemplar != nil && len(bucket.Exemplar.Label) > 0 {
				_, err = appender.AppendExemplar(ref, lbls, exemplar.Exemplar{
					Labels: convertLabelPairToLabels(bucket.Exemplar.GetLabel()),
					Value:  bucket.Exemplar.GetValue(),
					Ts:     timeMs,
				})
				if err != nil {
					return err
				}
				// Don't clear bucket exemplars here as they might be needed for classic emission
			}
		}
	}

	return
}

func (h *nativeHistogram) classicHistograms(appender storage.Appender, timeMs int64, s *nativeHistogramSeries) (activeSeries int, err error) {
	if s.isNew() {
		endOfLastMinuteMs := getEndOfLastMinuteMs(timeMs)
		_, err = appender.Append(0, s.countLabels, endOfLastMinuteMs, 0)
		if err != nil {
			return activeSeries, err
		}
	}

	// sum
	_, err = appender.Append(0, s.sumLabels, timeMs, s.histogram.GetSampleSum())
	if err != nil {
		return activeSeries, err
	}
	activeSeries++

	// count
	_, err = appender.Append(0, s.countLabels, timeMs, getIfGreaterThenZeroOr(s.histogram.GetSampleCountFloat(), s.histogram.GetSampleCount()))
	if err != nil {
		return activeSeries, err
	}
	activeSeries++

	// bucket
	s.lb.Set(labels.MetricName, h.metricName+"_bucket")

	// the Prometheus histogram will sometimes add the +Inf bucket, it depends on whether there is an exemplar
	// for that bucket or not. To avoid adding it twice, keep track of it with this boolean.
	infBucketWasAdded := false

	for _, bucket := range s.histogram.Bucket {
		// add "le" label
		s.lb.Set(labels.BucketLabel, formatFloat(bucket.GetUpperBound()))

		if bucket.GetUpperBound() == math.Inf(1) {
			infBucketWasAdded = true
		}
		if s.isNew() {
			endOfLastMinuteMs := getEndOfLastMinuteMs(timeMs)
			_, appendErr := appender.Append(0, s.lb.Labels(), endOfLastMinuteMs, 0)
			if appendErr != nil {
				return activeSeries, appendErr
			}
		}

		ref, appendErr := appender.Append(0, s.lb.Labels(), timeMs, getIfGreaterThenZeroOr(bucket.GetCumulativeCountFloat(), bucket.GetCumulativeCount()))
		if appendErr != nil {
			return activeSeries, appendErr
		}
		activeSeries++

		// Check for exemplars from prometheus histogram
		if bucket.Exemplar != nil && len(bucket.Exemplar.Label) > 0 {
			_, err = appender.AppendExemplar(ref, s.lb.Labels(), exemplar.Exemplar{
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
		s.lb.Set(labels.BucketLabel, "+Inf")
		if s.isNew() {
			endOfLastMinuteMs := getEndOfLastMinuteMs(timeMs)
			_, err = appender.Append(0, s.lb.Labels(), endOfLastMinuteMs, 0)
			if err != nil {
				return activeSeries, err
			}
		}
		_, err := appender.Append(0, s.lb.Labels(), timeMs, getIfGreaterThenZeroOr(s.histogram.GetSampleCountFloat(), s.histogram.GetSampleCount()))
		if err != nil {
			return activeSeries, err
		}
		activeSeries++
	}

	// drop "le" label again
	s.lb.Del(labels.BucketLabel)

	if s.isNew() {
		s.registerSeenSeries()
	}

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
