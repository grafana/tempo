package registry

import (
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
	//promHistogram prometheus.HistogramVec

	seriesMtx sync.Mutex
	series    map[uint64]*nativeHistogramSeries

	onAddSerie    func(count uint32) bool
	onRemoveSerie func(count uint32)

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

func newNativeHistogram(name string, onAddSeries func(uint32) bool, onRemoveSeries func(count uint32), traceIDLabelName string) *nativeHistogram {
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
	s, ok = h.series[hash]
	if ok {
		h.updateSeries(s, value, traceID, multiplier)
		return
	}
	h.series[hash] = newSeries
}

func (h *nativeHistogram) newSeries(labelValueCombo *LabelValueCombo, value float64, traceID string, multiplier float64) *nativeHistogramSeries {
	newSeries := &nativeHistogramSeries{
		// TODO move these labels in HistogramOpts.ConstLabels?
		labels: labelValueCombo.getLabelPair(),
		promHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: h.name(),
			// TODO support help text
			Help: "Native histogram for metric " + h.name(),
			// TODO we can set these to also emit a classic histogram
			Buckets: nil,
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
	s.promHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
		value*multiplier,
		map[string]string{h.traceIDLabelName: traceID},
	)
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

	lb.Set(labels.MetricName, h.metricName)

	// set external labels
	for name, value := range externalLabels {
		lb.Set(name, value)
	}

	for _, s := range h.series {

		// set series-specific labels
		for i, name := range s.labels.names {
			lb.Set(name, s.labels.values[i])
		}

		// Append native histogram
		encodedMetric := &dto.Metric{}

		// Encode to protobuf representation
		err := s.promHistogram.Write(encodedMetric)
		if err != nil {
			return activeSeries, err
		}
		encodedHistogram := encodedMetric.GetHistogram()

		// Decode to Prometheus representation
		h := promhistogram.Histogram{
			Schema:        encodedHistogram.GetSchema(),
			Count:         encodedHistogram.GetSampleCount(),
			Sum:           encodedHistogram.GetSampleSum(),
			ZeroThreshold: encodedHistogram.GetZeroThreshold(),
			ZeroCount:     encodedHistogram.GetZeroCount(),
		}
		if len(encodedHistogram.PositiveSpan) > 0 {
			h.PositiveSpans = make([]promhistogram.Span, len(encodedHistogram.PositiveSpan))
			for i, span := range encodedHistogram.PositiveSpan {
				h.PositiveSpans[i] = promhistogram.Span{
					Offset: span.GetOffset(),
					Length: span.GetLength(),
				}
			}
		}
		h.PositiveBuckets = encodedHistogram.PositiveDelta
		if len(encodedHistogram.NegativeSpan) > 0 {
			h.NegativeSpans = make([]promhistogram.Span, len(encodedHistogram.NegativeSpan))
			for i, span := range encodedHistogram.NegativeSpan {
				h.NegativeSpans[i] = promhistogram.Span{
					Offset: span.GetOffset(),
					Length: span.GetLength(),
				}
			}
		}
		h.NegativeBuckets = encodedHistogram.NegativeDelta

		// TODO update activeSeries

		_, err = appender.AppendHistogram(0, lb.Labels(), timeMs, &h, nil)
		if err != nil {
			return activeSeries, err
		}

		if len(encodedHistogram.Exemplars) > 0 {
			for _, encodedExemplar := range encodedHistogram.Exemplars {

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
