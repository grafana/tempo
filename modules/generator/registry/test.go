package registry

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

// TestRegistry is a simple implementation of Registry intended for tests. It is not concurrent-safe.
type TestRegistry struct {
	// "metric{labels}" -> value
	metrics map[string]float64
}

var _ Registry = (*TestRegistry)(nil)

func NewTestRegistry() *TestRegistry {
	return &TestRegistry{
		metrics: map[string]float64{},
	}
}

func (t *TestRegistry) NewCounter(name string) Counter {
	return &testCounter{
		n:        name,
		registry: t,
	}
}

func (t *TestRegistry) NewGauge(name string) Gauge {
	return &testGauge{
		n:        name,
		registry: t,
	}
}

func (t *TestRegistry) NewLabelValueCombo(labels []string, values []string) *LabelValueCombo {
	return newLabelValueCombo(labels, values)
}

func (t *TestRegistry) NewHistogram(name string, buckets []float64, histogramOverrides HistogramMode) Histogram {
	return &testHistogram{
		nameSum:            name + "_sum",
		nameCount:          name + "_count",
		nameBucket:         name + "_bucket",
		buckets:            buckets,
		registry:           t,
		histogramOverrides: histogramOverrides,
	}
}

func (t *TestRegistry) addToMetric(name string, lbls labels.Labels, value float64) {
	if t == nil || t.metrics == nil {
		return
	}
	t.metrics[name+lbls.String()] += value
}

func (t *TestRegistry) setMetric(name string, lbls labels.Labels, value float64) {
	if t == nil || t.metrics == nil {
		return
	}
	t.metrics[name+lbls.String()] = value
}

// Query returns the value of the given metric. Note this is a rather naive query engine, it's only
// possible to query metrics by using the exact same labels as they were stored with.
func (t *TestRegistry) Query(name string, lbls labels.Labels) float64 {
	return t.metrics[name+lbls.String()]
}

func (t *TestRegistry) String() string {
	var metrics []string

	for metric, value := range t.metrics {
		metrics = append(metrics, fmt.Sprintf("%s %g", metric, value))
	}
	sort.Strings(metrics)

	return strings.Join(metrics, "\n")
}

type testCounter struct {
	n        string
	registry *TestRegistry
}

var _ Counter = (*testCounter)(nil)

func (t *testCounter) Inc(labelValueCombo *LabelValueCombo, value float64) {
	if value < 0 {
		panic("counter can only increase")
	}

	lbls := make(labels.Labels, len(labelValueCombo.labels.names))
	for i, label := range labelValueCombo.labels.names {
		lbls[i] = labels.Label{Name: label, Value: labelValueCombo.labels.values[i]}
	}
	sort.Sort(lbls)

	t.registry.addToMetric(t.n, lbls, value)
}

func (t *testCounter) name() string {
	return t.n
}

func (t *testCounter) collectMetrics(_ storage.Appender, _ int64, _ map[string]string) (activeSeries int, err error) {
	return
}

func (t *testCounter) removeStaleSeries(int64) {
	panic("implement me")
}

type testGauge struct {
	n        string
	registry *TestRegistry
}

var _ Gauge = (*testGauge)(nil)

func (t *testGauge) Inc(labelValueCombo *LabelValueCombo, value float64) {
	if value < 0 {
		panic("counter can only increase")
	}

	lbls := make(labels.Labels, len(labelValueCombo.labels.names))
	for i, label := range labelValueCombo.labels.names {
		lbls[i] = labels.Label{Name: label, Value: labelValueCombo.labels.values[i]}
	}
	sort.Sort(lbls)

	t.registry.addToMetric(t.n, lbls, value)
}

func (t *testGauge) Set(labelValueCombo *LabelValueCombo, value float64) {
	lbls := make(labels.Labels, len(labelValueCombo.labels.names))
	for i, label := range labelValueCombo.labels.names {
		lbls[i] = labels.Label{Name: label, Value: labelValueCombo.labels.values[i]}
	}
	sort.Sort(lbls)

	t.registry.setMetric(t.n, lbls, value)
}

func (t *testGauge) SetForTargetInfo(labelValueCombo *LabelValueCombo, value float64) {
	t.Set(labelValueCombo, value)
}

func (t *testGauge) name() string {
	return t.n
}

func (t *testGauge) collectMetrics(_ storage.Appender, _ int64, _ map[string]string) (activeSeries int, err error) {
	return 0, nil
}

func (t *testGauge) removeStaleSeries(int64) {
	panic("implement me")
}

type testHistogram struct {
	nameSum            string
	nameCount          string
	nameBucket         string
	buckets            []float64
	registry           *TestRegistry
	histogramOverrides HistogramMode
}

var (
	_ Histogram = (*testHistogram)(nil)
	_ metric    = (*testHistogram)(nil)
)

func (t *testHistogram) ObserveWithExemplar(labelValueCombo *LabelValueCombo, value float64, _ string, multiplier float64) {
	lbls := make(labels.Labels, len(labelValueCombo.labels.names))
	for i, label := range labelValueCombo.labels.names {
		lbls[i] = labels.Label{Name: label, Value: labelValueCombo.labels.values[i]}
	}
	sort.Sort(lbls)

	t.registry.addToMetric(t.nameCount, lbls, 1*multiplier)
	t.registry.addToMetric(t.nameSum, lbls, value*multiplier)

	for _, bucket := range t.buckets {
		if value <= bucket {
			t.registry.addToMetric(t.nameBucket, withLe(lbls, bucket), 1*multiplier)
		}
	}
	t.registry.addToMetric(t.nameBucket, withLe(lbls, math.Inf(1)), 1*multiplier)
}

func (t *testHistogram) name() string {
	panic("implement me")
}

func withLe(lbls labels.Labels, le float64) labels.Labels {
	lb := labels.NewBuilder(lbls)
	lb.Set(labels.BucketLabel, formatFloat(le))
	return lb.Labels()
}

func (t *testHistogram) collectMetrics(_ storage.Appender, _ int64, _ map[string]string) (activeSeries int, err error) {
	panic("implement me")
}

func (t *testHistogram) removeStaleSeries(int64) {
	panic("implement me")
}
