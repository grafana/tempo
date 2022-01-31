package spanmetrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	gen "github.com/grafana/tempo/modules/generator/processor"
	test_util "github.com/grafana/tempo/modules/generator/processor/util/test"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestSpanMetrics(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	p := New(cfg, "test")

	registry := gen.NewRegistry(nil)
	err := p.RegisterMetrics(registry)
	assert.NoError(t, err)

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	req := test.MakeBatch(10, nil)

	err = p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{req}})
	assert.NoError(t, err)

	appender := &test_util.Appender{}

	collectTime := time.Now()
	err = registry.Gather(appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_calls_total"}`, 10},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_count"}`, 10},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_sum"}`, 10},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.002"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.004"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.008"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.016"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.032"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.064"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.128"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.256"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.512"}`, 0},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="1.024"}`, 10},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="2.048"}`, 10},
		{`{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="4.096"}`, 10},
	}
	appender.ContainsAll(t, expectedMetrics, collectTime)
}

func TestSpanMetrics_dimensions(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.Dimensions = []string{"foo", "bar"}
	p := New(cfg, "test")

	registry := gen.NewRegistry(nil)
	err := p.RegisterMetrics(registry)
	assert.NoError(t, err)

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
	err = p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})
	assert.NoError(t, err)

	appender := &test_util.Appender{}

	collectTime := time.Now()
	err = registry.Gather(appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_calls_total"}`, 10},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_count"}`, 10},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_sum"}`, 10},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.002"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.004"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.008"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.016"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.032"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.064"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.128"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.256"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="0.512"}`, 0},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="1.024"}`, 10},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="2.048"}`, 10},
		{`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="4.096"}`, 10},
		// {`{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_span_metrics_duration_seconds_bucket", le="+Inf"}`, 10},
	}
	appender.ContainsAll(t, expectedMetrics, collectTime)
}
