package spanmetrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempopb"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestSpanMetrics(t *testing.T) {
	processor := New(Config{}, "test")

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
	appender.ContainsAll(t, expectedMetrics, collectTime)
}

func TestSpanMetrics_staleMetrics(t *testing.T) {
	now := time.Now()
	theTime := &now

	cfg := Config{
		DeleteAfterLastUpdate: 5 * time.Minute,
	}
	processor := New(cfg, "test").(*processor)
	processor.now = func() time.Time {
		return *theTime
	}

	// push some spans
	req := test.MakeBatch(1, nil)
	err := processor.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{req}})
	assert.NoError(t, err)

	// time skip by 2 minutes
	*theTime = theTime.Add(2 * time.Minute)

	// assert the metric has been created
	appender := &testAppender{}
	err = processor.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	appender.Contains(t, testMetric{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_calls_total"}`, 1})

	// time skip another 5 minutes
	*theTime = theTime.Add(5 * time.Minute)

	// assert the metric has been removed
	appender = &testAppender{}
	err = processor.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	appender.NotContains(t, `{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_calls_total"}`)
}
