package spanmetrics

import (
	"context"
	"testing"
	"time"

	test_util "github.com/grafana/tempo/modules/generator/processor/util/test"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestSpanMetrics(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	processor := New(cfg, "test")

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	req := test.MakeBatch(10, nil)
	err := processor.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{req}})
	assert.NoError(t, err)

	appender := &test_util.Appender{}

	collectTime := time.Now()
	err = processor.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_calls_total"}`, 10},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_count"}`, 10},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_sum"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="0.1"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="0.2"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="0.4"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="0.8"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="1.6"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="3.2"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="6.4"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_latency_bucket", le="12.8"}`, 0},
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
	appender := &test_util.Appender{}
	err = processor.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	appender.Contains(t, test_util.Metric{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_calls_total"}`, 1})

	// time skip another 5 minutes
	*theTime = theTime.Add(5 * time.Minute)

	// assert the metric has been removed
	appender = &test_util.Appender{}
	err = processor.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	appender.NotContains(t, `{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", __name__="tempo_calls_total"}`)
}

func TestSpanMetrics_dimensions(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.Dimensions = []string{"foo", "bar"}

	processor := New(cfg, "test")

	batch := test.MakeBatch(10, nil)
	for _, rs := range batch.InstrumentationLibrarySpans {
		for _, s := range rs.Spans {
			s.Attributes = append(s.Attributes, &common_v1.KeyValue{
				Key:   "foo",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "foo-value"}},
			})
			s.Attributes = append(s.Attributes, &common_v1.KeyValue{
				Key:   "bar",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar-value"}},
			})
		}
	}
	err := processor.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})
	assert.NoError(t, err)

	appender := &test_util.Appender{}

	collectTime := time.Now()
	err = processor.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_calls_total"}`, 10},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_count"}`, 10},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_sum"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="0.1"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="0.2"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="0.4"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="0.8"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="1.6"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="3.2"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="6.4"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="12.8"}`, 0},
		{`{service="test-service", span_name="test", span_kind="SPAN_KIND_CLIENT", span_status="STATUS_CODE_OK", foo="foo-value", bar="bar-value", __name__="tempo_latency_bucket", le="+Inf"}`, 0},
	}
	appender.ContainsAll(t, expectedMetrics, collectTime)
}
