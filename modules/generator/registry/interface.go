package registry

import "github.com/prometheus/prometheus/model/labels"

// Registry is a metrics store.
type Registry interface {
	NewLabelValueCombo(labels []string, values []string) *LabelValueCombo
	NewCounter(name string) Counter
	NewHistogram(name string, buckets []float64, histogramOverride HistogramMode) Histogram
	NewGauge(name string) Gauge
}

// Counter
// https://prometheus.io/docs/concepts/metric_types/#counter
type Counter interface {
	metric

	Inc(labelValueCombo *LabelValueCombo, value float64)
}

// Histogram
// https://prometheus.io/docs/concepts/metric_types/#histogram
type Histogram interface {
	metric

	// ObserveWithExemplar observes a datapoint with the given values. traceID will be added as exemplar.
	ObserveWithExemplar(labelValueCombo *LabelValueCombo, value float64, traceID string, multiplier float64)
}

// Gauge
// https://prometheus.io/docs/concepts/metric_types/#gauge
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Gauge
type Gauge interface {
	metric

	// Set sets the Gauge to an arbitrary value.
	Set(labelValueCombo *LabelValueCombo, value float64)
	Inc(labelValueCombo *LabelValueCombo, value float64)
	SetForTargetInfo(labelValueCombo *LabelValueCombo, value float64)
}

type HistogramMode int

const (
	HistogramModeClassic HistogramMode = iota
	HistogramModeNative
	HistogramModeBoth
)

var HistogramModeToString = map[HistogramMode]string{
	HistogramModeClassic: "classic",
	HistogramModeNative:  "native",
	HistogramModeBoth:    "both",
}

var HistogramModeToValue = map[string]HistogramMode{
	"classic": HistogramModeClassic,
	"native":  HistogramModeNative,
	"both":    HistogramModeBoth,
}

// LabelValueCombo is a wrapper around a slice of label values. It has the ability to cache the hash of
// the label values. When passing the same label values to multiple metrics, create LabelValueCombo once
// and pass it to all of them.
type LabelValueCombo struct {
	labels labels.Labels
}

func newLabelPair(lbls []string, values []string) labels.Labels {
	builder := labels.NewBuilder(labels.New())
	for i := range lbls {
		builder.Set(lbls[i], values[i])
	}
	return builder.Labels()
}

func newLabelValueCombo(labels []string, values []string) *LabelValueCombo {
	labelPair := newLabelPair(labels, values)
	return &LabelValueCombo{
		labels: labelPair,
	}
}

func newLabelValueComboWithMax(labels []string, values []string, maxLabelLength int, maxLengthLabelValue int) *LabelValueCombo {
	truncateLength(labels, maxLabelLength)
	truncateLength(values, maxLengthLabelValue)
	return newLabelValueCombo(labels, values)
}

func (l *LabelValueCombo) getHash() uint64 {
	if l == nil {
		return 0
	}
	return l.labels.Hash()
}
