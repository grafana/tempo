package spanmetrics

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

var metricSpansDiscarded = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "metrics_generator_spans_discarded_total",
	Help:      "The total number of discarded spans received per tenant",
}, []string{"tenant", "reason"})

func TestSpanMetrics(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	require.Equal(t, p.Name(), "span-metrics")

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestSpanMetricsTargetInfoEnabled(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"job":         "test-service",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestSpanMetrics_dimensions(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.IntrinsicDimensions.SpanKind = false
	cfg.IntrinsicDimensions.StatusMessage = true
	cfg.Dimensions = []string{"foo", "bar", "does-not-exist"}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO create some spans that are missing the custom dimensions/tags
	batch := test.MakeBatch(10, nil)

	// Add some attributes
	for _, rs := range batch.ScopeSpans {
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
		"status_code":    "STATUS_CODE_OK",
		"status_message": "OK",
		"foo":            "foo-value",
		"bar":            "bar-value",
		"does_not_exist": "",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestSpanMetrics_collisions(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.Dimensions = []string{"span.kind", "span_name"}
	cfg.IntrinsicDimensions.SpanKind = false

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(10, nil)
	for _, rs := range batch.ScopeSpans {
		for _, s := range rs.Spans {
			s.Attributes = append(s.Attributes, &common_v1.KeyValue{
				Key:   "span.kind",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "colliding_kind"}},
			})
			s.Attributes = append(s.Attributes, &common_v1.KeyValue{
				Key:   "span_name",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "colliding_name"}},
			})
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"status_code": "STATUS_CODE_OK",
		"__span_kind": "colliding_kind",
		"__span_name": "colliding_name",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestJobLabelWithNamespaceAndInstanceID(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// add namespace

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.namespace",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-namespace"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"job":         "test-namespace/test-service",
		"instance":    "abc-instance-id-test-def",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestSpanMetrics_applyFilterPolicy(t *testing.T) {
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cases := []struct {
		filterPolicies     []filterconfig.FilterPolicy
		expectedMatches    float64
		expectedRejections float64
	}{
		{
			expectedMatches:    10.0,
			expectedRejections: 0.0,
			filterPolicies: []filterconfig.FilterPolicy{
				{
					Include: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{
								Key:   "span.foo",
								Value: "foo-value",
							},
						},
					},
				},
			},
		},
		{
			expectedMatches:    0.0,
			expectedRejections: 10.0,
			filterPolicies: []filterconfig.FilterPolicy{
				{
					Include: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{
								Key:   "span.nope",
								Value: "nothere",
							},
						},
					},
				},
			},
		},
		{
			expectedMatches:    0.0,
			expectedRejections: 10.0,
			filterPolicies: []filterconfig.FilterPolicy{
				{
					Exclude: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{
								Key:   "status",
								Value: "STATUS_CODE_OK",
							},
						},
					},
				},
			},
		},
		{
			expectedMatches:    10.0,
			expectedRejections: 0.0,
			filterPolicies: []filterconfig.FilterPolicy{
				{
					Include: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Regex,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{
								Key:   "kind",
								Value: "SPAN_KIND_.*",
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range cases {
		testName := fmt.Sprintf("filter_policy_%d", i)
		t.Run(testName, func(t *testing.T) {
			t.Logf("test case: %s", testName)

			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.HistogramBuckets = []float64{0.5, 1}
			cfg.IntrinsicDimensions.SpanKind = false
			cfg.IntrinsicDimensions.StatusMessage = true
			cfg.Dimensions = []string{"foo", "bar", "does-not-exist"}
			cfg.FilterPolicies = tc.filterPolicies

			testRegistry := registry.NewTestRegistry()
			p, err := New(cfg, testRegistry, filteredSpansCounter)
			require.NoError(t, err)
			defer p.Shutdown(context.Background())

			// TODO create some spans that are missing the custom dimensions/tags
			batch := test.MakeBatch(10, nil)

			// Add some attributes
			for _, rs := range batch.ScopeSpans {
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

			t.Logf("batch: %v", batch)

			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

			t.Logf("%s", testRegistry)

			lbls := labels.FromMap(map[string]string{
				"service":        "test-service",
				"span_name":      "test",
				"status_code":    "STATUS_CODE_OK",
				"status_message": "OK",
				"foo":            "foo-value",
				"bar":            "bar-value",
				"does_not_exist": "",
			})

			assert.Equal(t, tc.expectedMatches, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

			assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
			assert.Equal(t, tc.expectedMatches, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
			assert.Equal(t, tc.expectedMatches, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
			assert.Equal(t, tc.expectedMatches, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
			assert.Equal(t, tc.expectedMatches, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
		})
	}
}

func TestJobLabelWithNamespaceAndNoServiceName(t *testing.T) {
	// no service.name = no job label/dimension
	// but service will still be there
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// remove service.name
	serviceNameIndex := 0
	for i, attribute := range batch.Resource.Attributes {
		if attribute.Key == "service.name" {
			serviceNameIndex = i
		}
	}

	copy(batch.Resource.Attributes[serviceNameIndex:], batch.Resource.Attributes[serviceNameIndex+1:])
	batch.Resource.Attributes = batch.Resource.Attributes[:len(batch.Resource.Attributes)-1]

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.namespace",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-namespace"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":     "",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestLabelsWithDifferentBatches(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// first batch will not have name-space nor instance id
	// this will create metrics with job=<service.name> and no instance label

	batch2 := test.MakeBatch(10, nil)

	// batch 2 will have namespace and instance id
	// this will create another set of metrics with job=<service.namespace>/<service.name> and instance=<service.instance.id>
	batch2.Resource.Attributes = append(batch2.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.namespace",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-namespace"}},
	})

	batch2.Resource.Attributes = append(batch2.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	batch3 := test.MakeBatch(10, nil)

	// batch 3 will be exactly like batch 1
	// this will not create new metrics but should increase the values of the first set

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch, batch2, batch3}})

	fmt.Println(testRegistry)

	lbls1 := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"job":         "test-service",
	})

	// first set
	assert.Equal(t, 20.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls1))
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls1, 0.5)))
	assert.Equal(t, 20.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls1, 1)))
	assert.Equal(t, 20.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls1, math.Inf(1))))
	assert.Equal(t, 20.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls1))
	assert.Equal(t, 20.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls1))

	lbs2 := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"job":         "test-namespace/test-service",
		"instance":    "abc-instance-id-test-def",
	})

	// second set
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbs2))
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbs2, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbs2, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbs2, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbs2))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbs2))
}

func TestTargetInfoEnabled(t *testing.T) {
	// no service.name = no job label/dimension
	// if the only labels are job and instance then target_info should not exist
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// add instance
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	// add additional source attributes
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "cluster",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "ip",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"job":      "test-service",
		"instance": "abc-instance-id-test-def",
		"cluster":  "eu-west-0",
		"ip":       "1.1.1.1",
	})

	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls))
}

func TestTargetInfoDisabled(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = false
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// add instance
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	// add additional source attributes
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "cluster",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "ip",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	registry := fmt.Sprint(testRegistry)
	targetInfoExist := strings.Contains(registry, "traces_target_info")

	assert.Equal(t, false, targetInfoExist)
}

func TestTargetInfoWithExclusion(t *testing.T) {
	// no service.name = no job label/dimension
	// if the only labels are job and instance then target_info should not exist
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.TargetInfoExcludedDimensions = []string{"container", "container.id"}
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// add instance
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	// add additional source attributes
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "cluster",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "ip",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
	})

	// add attribute for labels that we want to drop
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "container",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "container.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "xyz123"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"job":      "test-service",
		"instance": "abc-instance-id-test-def",
		"cluster":  "eu-west-0",
		"ip":       "1.1.1.1",
	})

	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls))
}

func TestTargetInfoSanitizeLabelName(t *testing.T) {
	// no service.name = no job label/dimension
	// if the only labels are job and instance then target_info should not exist
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// add instance
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	// add additional source attributes
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "cluster-id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "target.ip",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"job":        "test-service",
		"instance":   "abc-instance-id-test-def",
		"cluster_id": "eu-west-0",
		"target_ip":  "1.1.1.1",
	})

	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls))
}

func TestTargetInfoWithJobAndInstanceOnly(t *testing.T) {
	// no service.name = no job label/dimension
	// if the only labels are job and instance then target_info should not exist
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// add instance
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	registry := fmt.Sprint(testRegistry)
	targetInfoExist := strings.Contains(registry, "traces_target_info")

	assert.Equal(t, false, targetInfoExist)
}

func TestTargetInfoNoJobAndNoInstance(t *testing.T) {
	// no service.name = no job label/dimension
	// if both job and instance are missing, target info should not exist
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// remove service.name
	serviceNameIndex := 0
	for i, attribute := range batch.Resource.Attributes {
		if attribute.Key == "service.name" {
			serviceNameIndex = i
		}
	}

	copy(batch.Resource.Attributes[serviceNameIndex:], batch.Resource.Attributes[serviceNameIndex+1:])
	batch.Resource.Attributes = batch.Resource.Attributes[:len(batch.Resource.Attributes)-1]

	// add additional source attributes
	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "cluster",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, &common_v1.KeyValue{
		Key:   "ip",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	registry := fmt.Sprint(testRegistry)
	targetInfoExist := strings.Contains(registry, "traces_target_info")

	assert.Equal(t, false, targetInfoExist)
}

func TestTargetInfoWithDifferentBatches(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// first batch will not have name-space nor instance id
	// this will NOT create target_info metrics since it only has job but no other attributes

	batch2 := test.MakeBatch(10, nil)

	// batch 2 will have instance id & cluster
	// this will create a target_info metric with job, instance, and cluster

	// add instance
	batch2.Resource.Attributes = append(batch2.Resource.Attributes, &common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	// add additional source attributes
	batch2.Resource.Attributes = append(batch2.Resource.Attributes, &common_v1.KeyValue{
		Key:   "cluster",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
	})

	batch3 := test.MakeBatch(10, nil)
	// batch 3 will have  ip
	// this will create a target_info metric with job and ip only // no cluster no instance

	// add additional source attributes
	batch3.Resource.Attributes = append(batch3.Resource.Attributes, &common_v1.KeyValue{
		Key:   "ip",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch, batch2, batch3}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"job": "test-service",
	})

	lbls2 := labels.FromMap(map[string]string{
		"job":      "test-service",
		"instance": "abc-instance-id-test-def",
		"cluster":  "eu-west-0",
	})

	lbls3 := labels.FromMap(map[string]string{
		"job": "test-service",
		"ip":  "1.1.1.1",
	})

	assert.Equal(t, 0.0, testRegistry.Query("traces_target_info", lbls))
	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls2))
	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls3))
}

func TestSpanMetricsDimensionMapping(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.IntrinsicDimensions.SpanKind = false
	cfg.IntrinsicDimensions.StatusMessage = true
	cfg.DimensionMappings = []sharedconfig.DimensionMappings{
		{
			Name:        "foobar",
			SourceLabel: []string{"foo", "bar"},
			Join:        "/",
		},
	}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO create some spans that are missing the custom dimensions/tags
	batch := test.MakeBatch(10, nil)

	// Add some attributes
	for _, rs := range batch.ScopeSpans {
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
		"status_code":    "STATUS_CODE_OK",
		"status_message": "OK",
		"foobar":         "foo-value/bar-value",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestSpanMetricsDimensionMappingMissingLabels(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.IntrinsicDimensions.SpanKind = false
	cfg.IntrinsicDimensions.StatusMessage = true
	cfg.DimensionMappings = []sharedconfig.DimensionMappings{
		// label "second" missing in attributes, correct label = "first"
		{
			Name:        "first_only",
			SourceLabel: []string{"first", "second"},
			Join:        "/",
		},
		// label "hello" missin in attributes, correct label = "world"
		{
			Name:        "world_only",
			SourceLabel: []string{"hello", "world"},
			Join:        "/",
		},
		// label "middle" missing in attributes, correct label = "first->last"
		{
			Name:        "first/last",
			SourceLabel: []string{"first", "middle", "last"},
			Join:        "->",
		},
	}

	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO create some spans that are missing the custom dimensions/tags
	batch := test.MakeBatch(10, nil)

	// Add some attributes
	for _, rs := range batch.ScopeSpans {
		for _, s := range rs.Spans {
			s.Attributes = append(s.Attributes, &common_v1.KeyValue{
				Key:   "first",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "first-value"}},
			})
			s.Attributes = append(s.Attributes, &common_v1.KeyValue{
				Key:   "world",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "world-value"}},
			})
			s.Attributes = append(s.Attributes, &common_v1.KeyValue{
				Key:   "last",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "last-value"}},
			})
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":        "test-service",
		"span_name":      "test",
		"status_code":    "STATUS_CODE_OK",
		"status_message": "OK",
		"first_only":     "first-value",
		"world_only":     "world-value",
		"first/last":     "first-value->last-value",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func withLe(lbls labels.Labels, le float64) labels.Labels {
	lb := labels.NewBuilder(lbls)
	lb = lb.Set(labels.BucketLabel, strconv.FormatFloat(le, 'f', -1, 64))
	return lb.Labels()
}

func BenchmarkSpanMetrics_applyFilterPolicyNone(b *testing.B) {
	// Generate a batch of 100k spans
	// r, done := test.NewRandomBatcher()
	// defer done()
	// batch := r.GenerateBatch(1e6)
	// data, _ := batch.Marshal()
	// _ = ioutil.WriteFile("testbatch100k", data, 0600)

	// Read the file generated above
	data, err := os.ReadFile("testbatch100k")
	require.NoError(b, err)
	batch := &trace_v1.ResourceSpans{}
	err = batch.Unmarshal(data)
	require.NoError(b, err)

	// b.Logf("size: %s", humanize.Bytes(uint64(batch.Size())))
	// b.Logf("span count: %d", len(batch.ScopeSpans))

	policies := []filterconfig.FilterPolicy{}

	benchmarkFilterPolicy(b, policies, batch)
}

func BenchmarkSpanMetrics_applyFilterPolicySmall(b *testing.B) {
	// Read the file generated above
	data, err := os.ReadFile("testbatch100k")
	require.NoError(b, err)
	batch := &trace_v1.ResourceSpans{}
	err = batch.Unmarshal(data)
	require.NoError(b, err)

	policies := []filterconfig.FilterPolicy{
		{
			Include: &filterconfig.PolicyMatch{
				MatchType: filterconfig.Strict,
				Attributes: []filterconfig.MatchPolicyAttribute{
					{
						Key:   "span.foo",
						Value: "foo-value",
					},
				},
			},
		},
	}

	benchmarkFilterPolicy(b, policies, batch)
}

func BenchmarkSpanMetrics_applyFilterPolicyMedium(b *testing.B) {
	// Read the file generated above
	data, err := os.ReadFile("testbatch100k")
	require.NoError(b, err)
	batch := &trace_v1.ResourceSpans{}
	err = batch.Unmarshal(data)
	require.NoError(b, err)

	policies := []filterconfig.FilterPolicy{
		{
			Include: &filterconfig.PolicyMatch{
				MatchType: filterconfig.Strict,
				Attributes: []filterconfig.MatchPolicyAttribute{
					{
						Key:   "span.foo",
						Value: "foo-value",
					},
					{
						Key:   "span.x",
						Value: "foo-value",
					},
					{
						Key:   "span.y",
						Value: "foo-value",
					},
					{
						Key:   "span.z",
						Value: "foo-value",
					},
				},
			},
		},
	}

	benchmarkFilterPolicy(b, policies, batch)
}

func benchmarkFilterPolicy(b *testing.B, policies []filterconfig.FilterPolicy, batch *trace_v1.ResourceSpans) {
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered")

	testRegistry := registry.NewTestRegistry()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.FilterPolicies = policies
	p, err := New(cfg, testRegistry, filteredSpansCounter)
	require.NoError(b, err)
	defer p.Shutdown(context.Background())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})
	}
}
