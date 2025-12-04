package registry

import (
	"github.com/prometheus/prometheus/model/labels"
)

// Registry is a metrics store.
type Registry interface {
	NewLabelBuilder() LabelBuilder
	NewCounter(name string) Counter
	NewHistogram(name string, buckets []float64, histogramOverride HistogramMode) Histogram
	NewGauge(name string) Gauge
}

type LabelBuilder interface {
	Add(name, value string)
	CloseAndBuildLabels() (labels.Labels, bool)
}

// Counter
// https://prometheus.io/docs/concepts/metric_types/#counter
type Counter interface {
	metric

	Inc(lbls labels.Labels, value float64)
}

// Histogram
// https://prometheus.io/docs/concepts/metric_types/#histogram
type Histogram interface {
	metric

	// ObserveWithExemplar observes a datapoint with the given values. traceID will be added as exemplar.
	ObserveWithExemplar(lbls labels.Labels, value float64, traceID string, multiplier float64)
}

// Gauge
// https://prometheus.io/docs/concepts/metric_types/#gauge
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Gauge
type Gauge interface {
	metric

	// Set sets the Gauge to an arbitrary value.
	Set(lbls labels.Labels, value float64)
	Inc(lbls labels.Labels, value float64)
	SetForTargetInfo(lbls labels.Labels, value float64)
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
