package registry

// Registry is a metrics store.
type Registry interface {
	NewLabelValueCombo(labels []string, values []string) *LabelValueCombo
	NewCounter(name string) Counter
	NewHistogram(name string, buckets []float64) Histogram
	NewGauge(name string) Gauge
}

// Counter
// https://prometheus.io/docs/concepts/metric_types/#counter
type Counter interface {
	Inc(labelValueCombo *LabelValueCombo, value float64)
}

// Histogram
// https://prometheus.io/docs/concepts/metric_types/#histogram
type Histogram interface {
	// ObserveWithExemplar observes a datapoint with the given values. traceID will be added as exemplar.
	ObserveWithExemplar(labelValueCombo *LabelValueCombo, value float64, traceID string, multiplier float64)
}

// Gauge
// https://prometheus.io/docs/concepts/metric_types/#gauge
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Gauge
type Gauge interface {
	// Set sets the Gauge to an arbitrary value.
	Set(labelValueCombo *LabelValueCombo, value float64)
	Inc(labelValueCombo *LabelValueCombo, value float64)
	SetForTargetInfo(labelValueCombo *LabelValueCombo, value float64)
}

// LabelValueCombo is a wrapper around a slice of label values. It has the ability to cache the hash of
// the label values. When passing the same label values to multiple metrics, create LabelValueCombo once
// and pass it to all of them.
type LabelValueCombo struct {
	labels LabelPair
	hash   uint64
}

type LabelPair struct {
	names  []string
	values []string
}

func newLabelPair(labels []string, values []string) LabelPair {
	return LabelPair{
		names:  labels,
		values: values,
	}
}

func newLabelValueCombo(labels []string, values []string) *LabelValueCombo {
	labelPair := newLabelPair(labels, values)
	return &LabelValueCombo{
		labels: labelPair,
		hash:   0,
	}
}

func newLabelValueComboWithMax(labels []string, values []string, maxLabelLength int, maxLengthLabelValue int) *LabelValueCombo {
	truncateLength(labels, maxLabelLength)
	truncateLength(values, maxLengthLabelValue)
	return newLabelValueCombo(labels, values)
}

func (l *LabelValueCombo) getValues() []string {
	if l == nil {
		return nil
	}
	return l.labels.values
}

func (l *LabelValueCombo) getNames() []string {
	if l == nil {
		return nil
	}
	return l.labels.names
}

func (l *LabelValueCombo) getValuesCopy() []string {
	values := l.getValues()
	valuesCopy := make([]string, len(values))
	copy(valuesCopy, values)
	return valuesCopy
}

func (l *LabelValueCombo) getNamesCopy() []string {
	names := l.getNames()
	labelsCopy := make([]string, len(names))
	copy(labelsCopy, names)
	return labelsCopy
}

func (l *LabelValueCombo) getHash() uint64 {
	if l == nil {
		return 0
	}
	if l.hash != 0 {
		return l.hash
	}
	l.hash = hashLabelValues(l.labels)
	return l.hash
}

func (l *LabelValueCombo) getLabelPair() LabelPair {
	if l == nil {
		return LabelPair{}
	}
	return LabelPair{
		names:  l.getNamesCopy(),
		values: l.getValuesCopy(),
	}
}
