package spanmetrics

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestSpanMetrics(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}

	p := New(cfg, testRegistry)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"span_status": "STATUS_CODE_OK",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_duration_seconds_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_sum", lbls))
}

func TestSpanMetrics_dimensions(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.Dimensions = []string{"foo", "bar", "does-not-exist"}

	p := New(cfg, testRegistry)
	defer p.Shutdown(context.Background())

	// TODO create some spans that are missing the custom dimensions/tags
	batch := test.MakeBatch(10, nil)

	// Add some attributes
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

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":        "test-service",
		"span_name":      "test",
		"span_kind":      "SPAN_KIND_CLIENT",
		"span_status":    "STATUS_CODE_OK",
		"foo":            "foo-value",
		"bar":            "bar-value",
		"does_not_exist": "",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_duration_seconds_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_duration_seconds_sum", lbls))
}

func withLe(lbls labels.Labels, le float64) labels.Labels {
	lb := labels.NewBuilder(lbls)
	lb = lb.Set(labels.BucketLabel, strconv.FormatFloat(le, 'f', -1, 64))
	return lb.Labels()
}
