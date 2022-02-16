package processor

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/generator/processor/util/test"
)

func TestRegistry(t *testing.T) {
	now := time.Now()
	theTime := &now

	registry := NewRegistry(nil)
	registry.SetTimeNow(func() time.Time {
		return *theTime
	})

	// Register some Prometheus metrics
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "my_counter",
		Help:      "This is a test counter",
	})
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "my_counter_vec",
		Help:      "This is a test counter vec",
	}, []string{"label1", "label2"})
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "test",
		Name:      "my_histogram",
		Help:      "This is a test histogram",
		Buckets:   prometheus.LinearBuckets(1, 1, 3),
	})
	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "test",
		Name:      "my_histogram_vec",
		Help:      "This is a test histogram vec",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 3),
	}, []string{"label1"})

	registry.MustRegister(counter, counterVec, histogram, histogramVec)

	// Collect a first time
	testAppender := &test.Appender{}
	err := registry.Gather(testAppender)
	assert.NoError(t, err)

	expectedMetrics := []test.Metric{
		{Labels: `{__name__="test_my_counter"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_count"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_sum"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_bucket", le="1"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_bucket", le="2"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_bucket", le="3"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_bucket", le="+Inf"}`, Value: 0},
	}
	testAppender.ContainsAll(t, expectedMetrics, *theTime)

	*theTime = (*theTime).Add(5 * time.Second)

	// Modify the metrics
	counter.Inc()
	counterVec.WithLabelValues("value1", "value2").Inc()
	counterVec.WithLabelValues("value1", "anotherValue2").Add(2)
	histogram.Observe(2)
	histogram.Observe(3)
	histogram.Observe(4)
	histogramVec.WithLabelValues("value1").Observe(1)
	histogramVec.WithLabelValues("value2").Observe(2)

	// Collect a second time
	testAppender = &test.Appender{}
	err = registry.Gather(testAppender)
	assert.NoError(t, err)

	expectedMetrics = []test.Metric{
		{Labels: `{__name__="test_my_counter"}`, Value: 1},
		{Labels: `{label1="value1", label2="value2", __name__="test_my_counter_vec"}`, Value: 1},
		{Labels: `{label1="value1", label2="anotherValue2", __name__="test_my_counter_vec"}`, Value: 2},
		{Labels: `{__name__="test_my_histogram_count"}`, Value: 3},
		{Labels: `{__name__="test_my_histogram_sum"}`, Value: 9},
		{Labels: `{__name__="test_my_histogram_bucket", le="1"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_bucket", le="2"}`, Value: 1},
		{Labels: `{__name__="test_my_histogram_bucket", le="3"}`, Value: 2},
		{Labels: `{__name__="test_my_histogram_bucket", le="+Inf"}`, Value: 3},
		{Labels: `{label1="value1", __name__="test_my_histogram_vec_count"}`, Value: 1},
		{Labels: `{label1="value1", __name__="test_my_histogram_vec_sum"}`, Value: 1},
		{Labels: `{label1="value1", __name__="test_my_histogram_vec_bucket", le="1"}`, Value: 1},
		{Labels: `{label1="value1", __name__="test_my_histogram_vec_bucket", le="2"}`, Value: 1},
		{Labels: `{label1="value1", __name__="test_my_histogram_vec_bucket", le="4"}`, Value: 1},
		{Labels: `{label1="value1", __name__="test_my_histogram_vec_bucket", le="+Inf"}`, Value: 1},
		{Labels: `{label1="value2", __name__="test_my_histogram_vec_count"}`, Value: 1},
		{Labels: `{label1="value2", __name__="test_my_histogram_vec_sum"}`, Value: 2},
		{Labels: `{label1="value2", __name__="test_my_histogram_vec_bucket", le="1"}`, Value: 0},
		{Labels: `{label1="value2", __name__="test_my_histogram_vec_bucket", le="2"}`, Value: 1},
		{Labels: `{label1="value2", __name__="test_my_histogram_vec_bucket", le="4"}`, Value: 1},
		{Labels: `{label1="value2", __name__="test_my_histogram_vec_bucket", le="+Inf"}`, Value: 1},
	}
	testAppender.ContainsAll(t, expectedMetrics, *theTime)
}

func TestRegistry_exemplars(t *testing.T) {
	now := time.Now()
	theTime := &now

	registry := NewRegistry(nil)
	registry.SetTimeNow(func() time.Time {
		return *theTime
	})

	// Register a Prometheus histogram
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "test",
		Name:      "my_histogram",
		Help:      "This is a test histogram",
		Buckets:   prometheus.LinearBuckets(1, 1, 3),
	})

	registry.MustRegister(histogram)

	// Observe some values with exemplars
	histogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
		2, prometheus.Labels{"traceID": "1112"},
	)
	histogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
		3, prometheus.Labels{"traceID": "1113"},
	)
	histogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
		4, prometheus.Labels{"traceID": "1114"},
	)

	// Collect metrics
	testAppender := &test.Appender{}
	err := registry.Gather(testAppender)
	assert.NoError(t, err)

	expectedMetrics := []test.Metric{
		{Labels: `{__name__="test_my_histogram_count"}`, Value: 3},
		{Labels: `{__name__="test_my_histogram_sum"}`, Value: 9},
		{Labels: `{__name__="test_my_histogram_bucket", le="1"}`, Value: 0},
		{Labels: `{__name__="test_my_histogram_bucket", le="2"}`, Value: 1},
		{Labels: `{__name__="test_my_histogram_bucket", le="3"}`, Value: 2},
		{Labels: `{__name__="test_my_histogram_bucket", le="+Inf"}`, Value: 3},
	}
	testAppender.ContainsAll(t, expectedMetrics, *theTime)

	expectedLabels := []string{
		`{__name__="test_my_histogram_bucket", le="2"}`,
		`{__name__="test_my_histogram_bucket", le="3"}`,
		`{__name__="test_my_histogram_bucket", le="+Inf"}`,
	}
	expectedExemplars := []exemplar.Exemplar{
		{Labels: []labels.Label{{Name: "traceID", Value: "1112"}}, Value: 2, Ts: theTime.UnixMilli(), HasTs: true},
		{Labels: []labels.Label{{Name: "traceID", Value: "1113"}}, Value: 3, Ts: theTime.UnixMilli(), HasTs: true},
		{Labels: []labels.Label{{Name: "traceID", Value: "1114"}}, Value: 4, Ts: theTime.UnixMilli(), HasTs: true},
	}
	testAppender.ContainsAllExemplars(t, expectedLabels, expectedExemplars)
}

func TestRegisterer_externalLabels(t *testing.T) {
	now := time.Now()
	theTime := &now

	registry := NewRegistry(map[string]string{
		"external_label": "constant_value",
	})
	registry.SetTimeNow(func() time.Time {
		return *theTime
	})

	// Register some Prometheus metrics
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "my_counter",
		Help:      "This is a test counter",
	})
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "test",
		Name:      "my_histogram",
		Help:      "This is a test histogram",
		Buckets:   prometheus.LinearBuckets(1, 1, 3),
	})

	registry.MustRegister(counter, histogram)

	// Collect the metrics
	testAppender := &test.Appender{}
	err := registry.Gather(testAppender)
	assert.NoError(t, err)

	expectedMetrics := []test.Metric{
		{Labels: `{external_label="constant_value", __name__="test_my_counter"}`, Value: 0},
		{Labels: `{external_label="constant_value", __name__="test_my_histogram_count"}`, Value: 0},
		{Labels: `{external_label="constant_value", __name__="test_my_histogram_sum"}`, Value: 0},
		{Labels: `{external_label="constant_value", __name__="test_my_histogram_bucket", le="1"}`, Value: 0},
		{Labels: `{external_label="constant_value", __name__="test_my_histogram_bucket", le="2"}`, Value: 0},
		{Labels: `{external_label="constant_value", __name__="test_my_histogram_bucket", le="3"}`, Value: 0},
		{Labels: `{external_label="constant_value", __name__="test_my_histogram_bucket", le="+Inf"}`, Value: 0},
	}
	testAppender.ContainsAll(t, expectedMetrics, *theTime)
}
