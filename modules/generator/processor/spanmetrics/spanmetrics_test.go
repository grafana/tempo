package spanmetrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempopb"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestSpanMetrics(t *testing.T) {
	processor := New()

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	req := test.MakeBatch(10, nil)
	err := processor.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{req}})
	assert.NoError(t, err)

	appender := &testAppender{}

	collectTime := time.Now()
	err = processor.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	assert.False(t, appender.isCommitted)
	assert.False(t, appender.isRolledback)

	expectedMetrics := []testMetric{
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_calls_total"}`, 10},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_count"}`, 10},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_sum"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="1"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="10"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="50"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="100"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="500"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="+Inf"}`, 0},
	}
	appender.AssertContainsAll(t, expectedMetrics, collectTime)
}

type testAppender struct {
	isCommitted, isRolledback bool

	samples []testSample
}

type testMetric struct {
	labels string
	value  float64
}

type testSample struct {
	l labels.Labels
	t int64
	v float64
}

var _ storage.Appender = (*testAppender)(nil)

func (a *testAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	a.samples = append(a.samples, testSample{l, t, v})
	return 0, nil
}

func (a *testAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	panic("TODO add support for AppendExemplar")
}

func (a *testAppender) Commit() error {
	a.isCommitted = true
	return nil
}

func (a *testAppender) Rollback() error {
	a.isRolledback = true
	return nil
}

// AssertContainsAll asserts that testAppender contains all of expectedSamples in the given order.
// All samples should have a timestamp equal to timestamp with 1 millisecond of error margin.
func (a *testAppender) AssertContainsAll(t *testing.T, expectedSamples []testMetric, timestamp time.Time) {
	for i, sample := range a.samples {
		assert.Equal(t, expectedSamples[i].labels, sample.l.String())
		assert.InDelta(t, timestamp.UnixMilli(), sample.t, 1)
		assert.Equal(t, expectedSamples[i].value, sample.v)
	}
}
