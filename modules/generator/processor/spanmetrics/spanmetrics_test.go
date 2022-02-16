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
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_calls_total"}`, Value: 10},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_count"}`, Value: 10},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_sum"}`, Value: 10},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.002"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.004"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.008"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.016"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.032"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.064"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.128"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.256"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.512"}`, Value: 0},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="1.024"}`, Value: 10},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="2.048"}`, Value: 10},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="4.096"}`, Value: 10},
		{Labels: `{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="+Inf"}`, Value: 10},
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

	now := time.Now()
	registry.SetTimeNow(func() time.Time {
		return now
	})

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

	err = registry.Gather(appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_calls_total"}`, Value: 10},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_count"}`, Value: 10},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_sum"}`, Value: 10},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.002"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.004"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.008"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.016"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.032"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.064"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.128"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.256"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="0.512"}`, Value: 0},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="1.024"}`, Value: 10},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="2.048"}`, Value: 10},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="4.096"}`, Value: 10},
		{Labels: `{bar="bar-value", foo="foo-value", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", __name__="traces_spanmetrics_duration_seconds_bucket", le="+Inf"}`, Value: 10},
	}
	appender.ContainsAll(t, expectedMetrics, now)
}
