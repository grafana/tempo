package spanmetrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

var metricSpansDiscarded = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "metrics_generator_spans_discarded_total",
	Help:      "The total number of discarded spans received per tenant",
}, []string{"tenant", "reason", "processor"})

func TestSpanMetrics(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	require.Equal(t, p.Name(), "span-metrics")

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true
	cfg.TargetInfoExcludedDimensions = []string{"random.res.attr"}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatchWithAttributes(10, nil, []common_v1.KeyValue{
		{
			Key: "job",
			Value: &common_v1.AnyValue{
				Value: &common_v1.AnyValue_StringValue{
					StringValue: "dummy-job",
				},
			},
		},
		{
			Key: "service.instance.id",
			Value: &common_v1.AnyValue{
				Value: &common_v1.AnyValue_StringValue{
					StringValue: "instance",
				},
			},
		},
		{
			Key: "instance",
			Value: &common_v1.AnyValue{
				Value: &common_v1.AnyValue_StringValue{
					StringValue: "dummy-instance",
				},
			},
		},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"job":         "test-service",
		"instance":    "instance",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))

	targetInfoLabels := labels.FromMap(map[string]string{
		"job":        "test-service",
		"__job":      "dummy-job",
		"instance":   "instance",
		"__instance": "dummy-instance",
	})
	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", targetInfoLabels))
}

func TestSpanMetrics_dimensions(t *testing.T) {
	testCases := []struct {
		name            string
		dimensions      []string
		expectedCount   float64
		expectedBuckets map[float64]float64
		labels          map[string]string
	}{
		{
			name:          "All dimensions present",
			dimensions:    []string{"foo", "bar", "does-not-exist"},
			expectedCount: 10.0,
			expectedBuckets: map[float64]float64{
				0.5:         0.0,
				1.0:         10.0,
				math.Inf(1): 10.0,
			},
			labels: map[string]string{
				"service":        "test-service",
				"span_name":      "test",
				"status_code":    "STATUS_CODE_OK",
				"status_message": "OK",
				"foo":            "foo-value",
				"bar":            "bar-value",
				"does_not_exist": "",
			},
		},
		{
			name: "Valid labels",
			dimensions: []string{
				"valid_label_123", // underscores
				"_underscore_first",
			},
			expectedCount: 10.0,
			expectedBuckets: map[float64]float64{
				0.5:         0.0,
				1.0:         10.0,
				math.Inf(1): 10.0,
			},
			labels: map[string]string{
				"service":           "test-service",
				"span_name":         "test",
				"status_code":       "STATUS_CODE_OK",
				"status_message":    "OK",
				"valid_label_123":   "",
				"_underscore_first": "",
			},
		},
		{
			name: "Invalid labels are sanitized",
			dimensions: []string{
				"label-name",
				"foo.name",
				"foo bar",
			},
			expectedCount: 10.0,
			expectedBuckets: map[float64]float64{
				0.5:         0.0,
				1.0:         10.0,
				math.Inf(1): 10.0,
			},
			labels: map[string]string{
				"service":        "test-service",
				"span_name":      "test",
				"status_code":    "STATUS_CODE_OK",
				"status_message": "OK",
				"label_name":     "",
				"foo_name":       "",
				"foo_bar":        "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testRegistry := registry.NewTestRegistry()

			filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
			invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.HistogramBuckets = []float64{0.5, 1}
			cfg.IntrinsicDimensions.SpanKind = false
			cfg.IntrinsicDimensions.StatusMessage = true
			cfg.Dimensions = tc.dimensions

			p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
			require.NoError(t, err)
			defer p.Shutdown(context.Background())

			batch := test.MakeBatch(10, nil)
			for ri := range batch.ScopeSpans {
				for si := range batch.ScopeSpans[ri].Spans {
					batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
						Key:   "foo",
						Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "foo-value"}},
					})
					batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
						Key:   "bar",
						Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar-value"}},
					})
				}
			}

			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})
			lbls := labels.FromMap(tc.labels)
			fmt.Println(lbls.String())

			assert.Equal(t, tc.expectedCount, testRegistry.Query("traces_spanmetrics_calls_total", lbls))
			assert.Equal(t, tc.expectedCount, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
			assert.Equal(t, tc.expectedCount, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))

			for le, expected := range tc.expectedBuckets {
				assert.Equal(t, expected, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, le)))
			}
		})
	}
}

func TestSpanMetrics_collisions(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.Dimensions = []string{"span.kind", "span_name"}
	cfg.IntrinsicDimensions.SpanKind = false

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(10, nil)
	for ri := range batch.ScopeSpans {
		for si := range batch.ScopeSpans[ri].Spans {
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "span.kind",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "colliding_kind"}},
			})
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "span_name",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "colliding_name"}},
			})
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// add namespace

	batch.Resource.Attributes = append(batch.Resource.Attributes, common_v1.KeyValue{
		Key:   "service.namespace",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-namespace"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

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
			p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
			require.NoError(t, err)
			defer p.Shutdown(context.Background())

			// TODO create some spans that are missing the custom dimensions/tags
			batch := test.MakeBatch(10, nil)

			// Add some attributes
			for ri := range batch.ScopeSpans {
				for si := range batch.ScopeSpans[ri].Spans {
					batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
						Key:   "foo",
						Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "foo-value"}},
					})

					batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
						Key:   "bar",
						Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar-value"}},
					})
				}
			}

			t.Logf("batch: %v", batch)

			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
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

	batch.Resource.Attributes = append(batch.Resource.Attributes, common_v1.KeyValue{
		Key:   "service.namespace",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-namespace"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	// first batch will not have name-space nor instance id
	// this will create metrics with job=<service.name> and no instance label

	batch2 := test.MakeBatch(10, nil)

	// batch 2 will have namespace and instance id
	// this will create another set of metrics with job=<service.namespace>/<service.name> and instance=<service.instance.id>
	batch2.Resource.Attributes = append(batch2.Resource.Attributes, common_v1.KeyValue{
		Key:   "service.namespace",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-namespace"}},
	})

	batch2.Resource.Attributes = append(batch2.Resource.Attributes, common_v1.KeyValue{
		Key:   "service.instance.id",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
	})

	batch3 := test.MakeBatch(10, nil)

	// batch 3 will be exactly like batch 1
	// this will not create new metrics but should increase the values of the first set

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch, *batch2, *batch3}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		// add additional source attributes
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
		{
			Key:   "ip",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = false
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		// add additional source attributes
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
		{
			Key:   "ip",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	fmt.Println(testRegistry)

	registry := fmt.Sprint(testRegistry)
	targetInfoExist := strings.Contains(registry, "traces_target_info")

	assert.Equal(t, false, targetInfoExist)
}

func TestTargetInfoWithEmptyKey(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		{
			Key:   "", // add empty key attribute (should be skipped)
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "should-be-skipped"}},
		},
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		{
			Key:   "cluster", // At least one extra attribute is required to get target_info
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"job":      "test-service",
		"instance": "abc-instance-id-test-def",
		"cluster":  "eu-west-0",
	})

	// Verify target info exists with correct labels
	require.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls))
}

func TestTargetInfoWithEmptyValue(t *testing.T) {
	// Regression test: resource attributes with empty string values (e.g. host.id="")
	// caused target_info to be silently skipped. The Prometheus label builder treats
	// Set("x","") as Del("x"), which reduced the built label count below the raw
	// resourceAttributesCount, causing the guard condition to fail.
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		{
			Key:   "host.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: ""}}, // empty value
		},
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	lbls := labels.FromMap(map[string]string{
		"job":     "test-service",
		"cluster": "eu-west-0",
	})

	// target_info must be generated even when a resource attribute has an empty value
	require.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls))
}

func TestTargetInfoWithAllEmptyValues(t *testing.T) {
	// When ALL extra resource attributes have empty values, target_info should NOT
	// be emitted — there's no meaningful resource data beyond job/instance.
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id"}},
		},
		{
			Key:   "host.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: ""}},
		},
		{
			Key:   "container.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: ""}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	lbls := labels.FromMap(map[string]string{
		"job":      "test-service",
		"instance": "abc-instance-id",
	})

	// target_info should NOT exist — no real resource attributes survived
	require.Equal(t, 0.0, testRegistry.Query("traces_target_info", lbls))
}

func TestTargetInfoWithExclusion(t *testing.T) {
	// no service.name = no job label/dimension
	// if the only labels are job and instance then target_info should not exist
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.TargetInfoExcludedDimensions = []string{"container", "container.id"}
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		// add additional source attributes
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
		{
			Key:   "ip",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
		},
		// add attribute for labels that we want to drop
		{
			Key:   "container",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		{
			Key:   "container.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "xyz123"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		// add additional source attributes
		{
			Key:   "cluster-id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
		{
			Key:   "target.ip",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)

	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	fmt.Println(testRegistry)

	registry := fmt.Sprint(testRegistry)
	targetInfoExist := strings.Contains(registry, "traces_target_info")

	assert.Equal(t, false, targetInfoExist)
}

func TestTargetInfoNoJobAndNoInstance(t *testing.T) {
	// no service.name = no job label/dimension
	// if both job and instance are missing, target info should not exist
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
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
	batch.Resource.Attributes = append(batch.Resource.Attributes, common_v1.KeyValue{
		Key:   "cluster",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
	})

	batch.Resource.Attributes = append(batch.Resource.Attributes, common_v1.KeyValue{
		Key:   "ip",
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
	})

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	fmt.Println(testRegistry)

	registry := fmt.Sprint(testRegistry)
	targetInfoExist := strings.Contains(registry, "traces_target_info")

	assert.Equal(t, false, targetInfoExist)
}

func TestTargetInfoWithDifferentBatches(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// first batch will not have name-space nor instance id
	// this will NOT create target_info metrics since it only has job but no other attributes

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.
	batch := test.MakeBatch(10, nil)
	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
	}

	// batch 2 will have instance id & cluster
	// this will create a target_info metric with job, instance, and cluster
	batch2 := test.MakeBatch(10, nil)
	batch2.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		// add cluster
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
	}

	// batch 3 will have  ip
	// this will create a target_info metric with job and ip only // no cluster no instance
	batch3 := test.MakeBatch(10, nil)

	batch3.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add ip
		{
			Key:   "ip",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "1.1.1.1"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch, *batch2, *batch3}})

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

func TestEnableInstanceLabelFalse(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.Dimensions = []string{"http.method", "foo"}
	cfg.EnableInstanceLabel = false

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// first batch will not have name-space nor instance id
	// this will NOT create target_info metrics since it only has job but no other attributes

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.

	batchCount := 10
	batch := test.MakeBatch(batchCount, nil)
	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-abc"}},
		},
		// add cluster
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
	}
	for ri := range batch.ScopeSpans {
		for si := range batch.ScopeSpans[ri].Spans {
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "http.method",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "GET"}},
			})
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "foo",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar"}},
			})
		}
	}

	// batch 2 will have instance id & cluster
	// this will create a target_info metric with job, instance, and cluster
	batch2 := test.MakeBatch(batchCount, nil)
	batch2.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		// add cluster
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
	}
	for ri := range batch2.ScopeSpans {
		for si := range batch2.ScopeSpans[ri].Spans {
			batch2.ScopeSpans[ri].Spans[si].Attributes = append(batch2.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "http.method",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "GET"}},
			})
			batch2.ScopeSpans[ri].Spans[si].Attributes = append(batch2.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "foo",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar"}},
			})
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch, *batch2}})

	fmt.Println(testRegistry)

	targetInfoLabels := labels.FromMap(map[string]string{
		"job":     "test-service",
		"cluster": "eu-west-0",
	})

	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", targetInfoLabels))

	spanMetricsLabels := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"http_method": "GET",
		"foo":         "bar",
		"job":         "test-service",
	})
	assert.Equal(t, float64(batchCount*2), testRegistry.Query("traces_spanmetrics_calls_total", spanMetricsLabels))
	assert.Equal(t, float64(batchCount*2), testRegistry.Query("traces_spanmetrics_latency_count", spanMetricsLabels))
	assert.Equal(t, float64(batchCount*2), testRegistry.Query("traces_spanmetrics_latency_sum", spanMetricsLabels))

	expectedBuckets := map[float64]float64{
		0.5:         0.0,
		1.0:         20.0,
		math.Inf(1): 20.0,
	}
	for le, expected := range expectedBuckets {
		assert.Equal(t, expected, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(spanMetricsLabels, le)))
	}
}

func TestEnableInstanceLabelUnset(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.Dimensions = []string{"http.method", "foo"}
	// cfg.EnableInstanceLabel = true // by default it is true

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// first batch will not have name-space nor instance id
	// this will NOT create target_info metrics since it only has job but no other attributes

	// TODO give these spans some duration so we can verify latencies are recorded correctly, in fact we should also test with various span names etc.

	batchCount := 10
	batch := test.MakeBatch(batchCount, nil)
	batch.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-abc"}},
		},
		// add cluster
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
	}
	for ri := range batch.ScopeSpans {
		for si := range batch.ScopeSpans[ri].Spans {
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "http.method",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "GET"}},
			})
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "foo",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar"}},
			})
		}
	}

	// batch 2 will have instance id & cluster
	// this will create a target_info metric with job, instance, and cluster
	batch2 := test.MakeBatch(batchCount, nil)
	batch2.Resource.Attributes = []common_v1.KeyValue{
		// add service name
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		// add instance
		{
			Key:   "service.instance.id",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "abc-instance-id-test-def"}},
		},
		// add cluster
		{
			Key:   "cluster",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "eu-west-0"}},
		},
	}
	for ri := range batch2.ScopeSpans {
		for si := range batch2.ScopeSpans[ri].Spans {
			batch2.ScopeSpans[ri].Spans[si].Attributes = append(batch2.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "http.method",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "GET"}},
			})
			batch2.ScopeSpans[ri].Spans[si].Attributes = append(batch2.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "foo",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar"}},
			})
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch, *batch2}})

	fmt.Println(testRegistry)

	targetInfoLabels1 := labels.FromMap(map[string]string{
		"job":      "test-service",
		"cluster":  "eu-west-0",
		"instance": "abc-instance-id-test-abc",
	})

	targetInfoLabels2 := labels.FromMap(map[string]string{
		"job":      "test-service",
		"cluster":  "eu-west-0",
		"instance": "abc-instance-id-test-def",
	})

	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", targetInfoLabels1))
	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", targetInfoLabels2))

	spanMetricsLabels1 := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"http_method": "GET",
		"foo":         "bar",
		"job":         "test-service",
		"instance":    "abc-instance-id-test-abc",
	})

	spanMetricsLabels2 := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"http_method": "GET",
		"foo":         "bar",
		"job":         "test-service",
		"instance":    "abc-instance-id-test-def",
	})
	assert.Equal(t, float64(batchCount), testRegistry.Query("traces_spanmetrics_calls_total", spanMetricsLabels1))
	assert.Equal(t, float64(batchCount), testRegistry.Query("traces_spanmetrics_latency_count", spanMetricsLabels1))
	assert.Equal(t, float64(batchCount), testRegistry.Query("traces_spanmetrics_latency_sum", spanMetricsLabels1))

	assert.Equal(t, float64(batchCount), testRegistry.Query("traces_spanmetrics_calls_total", spanMetricsLabels2))
	assert.Equal(t, float64(batchCount), testRegistry.Query("traces_spanmetrics_latency_count", spanMetricsLabels2))
	assert.Equal(t, float64(batchCount), testRegistry.Query("traces_spanmetrics_latency_sum", spanMetricsLabels2))

	expectedBuckets := map[float64]float64{
		0.5:         0.0,
		1.0:         10.0,
		math.Inf(1): 10.0,
	}
	for le, expected := range expectedBuckets {
		assert.Equal(t, expected, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(spanMetricsLabels1, le)))
		assert.Equal(t, expected, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(spanMetricsLabels2, le)))
	}
}

func TestSpanMetricsDimensionMapping(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.IntrinsicDimensions.SpanKind = false
	cfg.IntrinsicDimensions.StatusMessage = true
	cfg.DimensionMappings = []sharedconfig.DimensionMappings{
		{
			Name:        "foo.bar",
			SourceLabel: []string{"foo", "bar"},
			Join:        "/",
		},
	}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO create some spans that are missing the custom dimensions/tags
	batch := test.MakeBatch(10, nil)

	// Add some attributes
	for ri := range batch.ScopeSpans {
		for si := range batch.ScopeSpans[ri].Spans {
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "foo",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "foo-value"}},
			})
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "bar",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "bar-value"}},
			})
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":        "test-service",
		"span_name":      "test",
		"status_code":    "STATUS_CODE_OK",
		"status_message": "OK",
		"foo_bar":        "foo-value/bar-value",
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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

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

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// TODO create some spans that are missing the custom dimensions/tags
	batch := test.MakeBatch(10, nil)

	// Add some attributes
	for ri := range batch.ScopeSpans {
		for si := range batch.ScopeSpans[ri].Spans {
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "first",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "first-value"}},
			})
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "world",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "world-value"}},
			})
			batch.ScopeSpans[ri].Spans[si].Attributes = append(batch.ScopeSpans[ri].Spans[si].Attributes, common_v1.KeyValue{
				Key:   "last",
				Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "last-value"}},
			})
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	fmt.Println(testRegistry)

	lbls := labels.FromMap(map[string]string{
		"service":        "test-service",
		"span_name":      "test",
		"status_code":    "STATUS_CODE_OK",
		"status_message": "OK",
		"first_only":     "first-value",
		"world_only":     "world-value",
		"first_last":     "first-value->last-value",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
}

func TestSpanMetricsNegativeLatency(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{
		Batches: []trace_v1.ResourceSpans{{
			Resource: &resource_v1.Resource{},
			ScopeSpans: []trace_v1.ScopeSpans{{
				Spans: []trace_v1.Span{
					{
						StartTimeUnixNano: uint64(1),
						EndTimeUnixNano:   uint64(0),
					},
				},
			}},
		}},
	})

	lbls := labels.FromMap(map[string]string{
		"service":     "",
		"span_name":   "",
		"span_kind":   "SPAN_KIND_UNSPECIFIED",
		"status_code": "STATUS_CODE_UNSET",
	})

	require.Equal(t, 1.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls), "calls_total")
	require.Equal(t, 1.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)), "bucket_0.5")
	require.Equal(t, 1.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)), "bucket_1")
	require.Equal(t, 1.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))), "bucket_Inf")
	require.Equal(t, 1.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls), "count")
	require.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls), "sum")
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
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	testRegistry := registry.NewTestRegistry()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.FilterPolicies = policies
	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(b, err)
	defer p.Shutdown(context.Background())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})
	}
}

func TestTargetInfoSkipsLabelsStartingWithNumber(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.HistogramBuckets = []float64{0.5, 1}

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(1, nil)
	batch.Resource.Attributes = []common_v1.KeyValue{
		{
			Key:   "service.name",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test-service"}},
		},
		{
			Key:   "5badlabel",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "should-be-ignored"}},
		},
		{
			Key:   "good_label",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "should-appear"}},
		},
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})
	// The produced target_info metric should not contain the bad label
	lbls := labels.FromMap(map[string]string{
		"job":        "test-service",
		"good_label": "should-appear",
	})

	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", lbls))
}

func TestValidationErrors(t *testing.T) {
	testCases := []struct {
		name   string
		cfg    Config
		expErr error
	}{
		{
			name: "default ok",
			cfg: func() Config {
				cfg := Config{}
				cfg.RegisterFlagsAndApplyDefaults("", nil)
				return cfg
			}(),
			expErr: nil,
		},
		{
			// this may be a valid use case if tenant is using diffferent SDK for instrumentation, so we allow it
			name: "dimension collision ignored",
			cfg: func() Config {
				cfg := Config{}
				cfg.RegisterFlagsAndApplyDefaults("", nil)
				cfg.Dimensions = []string{"deployment_environment", "deployment.environment"}
				return cfg
			}(),
			expErr: nil,
		},
		{
			name: "dimension collision after remapping",
			cfg: func() Config {
				cfg := Config{}
				cfg.RegisterFlagsAndApplyDefaults("", nil)
				cfg.Dimensions = []string{"foo_bar"}
				cfg.DimensionMappings = []sharedconfig.DimensionMappings{
					{
						Name:        "foo_bar",
						SourceLabel: []string{"foo.bar"},
					},
				}
				return cfg
			}(),
			expErr: errors.New(`dimension_mapping "foo_bar" produces label "foo_bar" which collides with dimension "foo_bar"`),
		},
	}

	var (
		testRegistry                 = registry.NewTestRegistry()
		filteredSpansCounter         = metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
		invalidUTF8SpanLabelsCounter = metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")
	)

	for _, tc := range testCases {
		p, err := New(tc.cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
		defer func() {
			if p != nil {
				p.Shutdown(t.Context())
			}
		}()
		require.Equal(t, tc.expErr, err)
	}
}

func TestSpanMetricsTraceStateMultiplier(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics-tracestate")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics-tracestate")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTraceStateSpanMultiplier = true

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// Create a batch with a span that has tracestate th:8 (50% sampling → multiplier 2)
	batch := test.MakeBatch(1, nil)
	for ri := range batch.ScopeSpans {
		for si := range batch.ScopeSpans[ri].Spans {
			batch.ScopeSpans[ri].Spans[si].TraceState = "ot=th:8"
		}
	}

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []trace_v1.ResourceSpans{*batch}})

	lbls := labels.FromMap(map[string]string{
		"service":     "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
	})

	// With 50% sampling (th:8), span multiplier is 2, so 1 span → count of 2
	assert.Equal(t, 2.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))
}

// recordingRegistry wraps a TestRegistry and records the order of every metric
// series update. The production registry limiter grants active-series and
// entity capacity in call order, so tests use this to pin the relative
// ordering of span metric and target_info updates.
type recordingRegistry struct {
	*registry.TestRegistry
	calls []string
}

func newRecordingRegistry() *recordingRegistry {
	return &recordingRegistry{TestRegistry: registry.NewTestRegistry()}
}

func (r *recordingRegistry) NewCounter(name string) registry.Counter {
	return &recordingCounter{Counter: r.TestRegistry.NewCounter(name), name: name, calls: &r.calls}
}

func (r *recordingRegistry) NewGauge(name string) registry.Gauge {
	return &recordingGauge{Gauge: r.TestRegistry.NewGauge(name), name: name, calls: &r.calls}
}

func (r *recordingRegistry) NewHistogram(name string, buckets []float64, mode registry.HistogramMode) registry.Histogram {
	return &recordingHistogram{Histogram: r.TestRegistry.NewHistogram(name, buckets, mode), name: name, calls: &r.calls}
}

type recordingCounter struct {
	registry.Counter
	name  string
	calls *[]string
}

func (c *recordingCounter) Inc(lbls labels.Labels, value float64) {
	*c.calls = append(*c.calls, c.name)
	c.Counter.Inc(lbls, value)
}

func (c *recordingCounter) IncBorrowed(lbls *registry.BorrowedLabels, value float64, timeMs int64) {
	*c.calls = append(*c.calls, c.name)
	c.Counter.IncBorrowed(lbls, value, timeMs)
}

type recordingGauge struct {
	registry.Gauge
	name  string
	calls *[]string
}

func (g *recordingGauge) Set(lbls labels.Labels, value float64) {
	*g.calls = append(*g.calls, g.name)
	g.Gauge.Set(lbls, value)
}

func (g *recordingGauge) Inc(lbls labels.Labels, value float64) {
	*g.calls = append(*g.calls, g.name)
	g.Gauge.Inc(lbls, value)
}

func (g *recordingGauge) SetForTargetInfo(lbls labels.Labels, value float64) {
	*g.calls = append(*g.calls, g.name)
	g.Gauge.SetForTargetInfo(lbls, value)
}

func (g *recordingGauge) SetForTargetInfoBorrowed(lbls *registry.BorrowedLabels, value float64, timeMs int64) {
	*g.calls = append(*g.calls, g.name)
	g.Gauge.SetForTargetInfoBorrowed(lbls, value, timeMs)
}

type recordingHistogram struct {
	registry.Histogram
	name  string
	calls *[]string
}

func (h *recordingHistogram) ObserveWithExemplar(lbls labels.Labels, value float64, traceID string, multiplier float64) {
	*h.calls = append(*h.calls, h.name)
	h.Histogram.ObserveWithExemplar(lbls, value, traceID, multiplier)
}

func (h *recordingHistogram) ObserveBorrowed(lbls *registry.BorrowedLabels, value float64, traceID []byte, multiplier float64, timeMs int64) {
	*h.calls = append(*h.calls, h.name)
	h.Histogram.ObserveBorrowed(lbls, value, traceID, multiplier, timeMs)
}

// TestSpanMetricsTargetInfoRegisteredAfterSpanMetrics pins the order in which
// series are updated when target_info is enabled. The registry limiter grants
// active-series and entity capacity in call order, so near the limit the span
// metrics must claim the remaining capacity before target_info — not the
// other way around. It also pins that target_info is registered exactly once
// per resource batch.
func TestSpanMetricsTargetInfoRegisteredAfterSpanMetrics(t *testing.T) {
	recReg := newRecordingRegistry()
	filteredSpansCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "filtered", "span-metrics")
	invalidUTF8SpanLabelsCounter := metricSpansDiscarded.WithLabelValues("test-tenant", "invalid_utf8", "span-metrics")

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.EnableTargetInfo = true

	p, err := New(cfg, recReg, filteredSpansCounter, invalidUTF8SpanLabelsCounter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(2, nil)
	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	expected := []string{
		// First accepted span updates its own series before target_info.
		"traces_spanmetrics_calls_total",
		"traces_spanmetrics_latency",
		"traces_spanmetrics_size_total",
		"traces_target_info",
		// Subsequent spans of the same resource do not re-register target_info.
		"traces_spanmetrics_calls_total",
		"traces_spanmetrics_latency",
		"traces_spanmetrics_size_total",
	}
	assert.Equal(t, expected, recReg.calls)
}

// TestSpanMetricsTargetInfoInvalidUTF8Spans verifies that target_info follows
// the pre-optimization contract: it is registered only once a span's primary
// labels validate, so a resource whose accepted spans all carry invalid UTF-8
// labels emits no target_info at all.
func TestSpanMetricsTargetInfoInvalidUTF8Spans(t *testing.T) {
	newProcessor := func(t *testing.T) (*registry.TestRegistry, prometheus.Counter, gen.Processor) {
		t.Helper()
		testRegistry := registry.NewTestRegistry()
		filteredSpansCounter := prometheus.NewCounter(prometheus.CounterOpts{Name: "test_filtered_spans_total"})
		invalidUTF8Counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "test_invalid_utf8_spans_total"})

		cfg := Config{}
		cfg.RegisterFlagsAndApplyDefaults("", nil)
		cfg.HistogramBuckets = []float64{0.5, 1}
		cfg.EnableTargetInfo = true

		p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8Counter)
		require.NoError(t, err)
		return testRegistry, invalidUTF8Counter, p
	}

	t.Run("all spans invalid emits no target_info", func(t *testing.T) {
		testRegistry, invalidUTF8Counter, p := newProcessor(t)
		defer p.Shutdown(context.Background())

		batch := test.MakeBatch(2, nil)
		for _, ss := range batch.ScopeSpans {
			for _, span := range ss.Spans {
				span.Name = "invalid-\xff-utf8"
			}
		}

		p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

		assert.NotContains(t, testRegistry.String(), "traces_target_info")
		assert.NotContains(t, testRegistry.String(), "traces_spanmetrics_calls_total")
		assert.Equal(t, 2.0, testutil.ToFloat64(invalidUTF8Counter))
	})

	t.Run("target_info registered after first valid span", func(t *testing.T) {
		testRegistry, invalidUTF8Counter, p := newProcessor(t)
		defer p.Shutdown(context.Background())

		batch := test.MakeBatch(2, nil)
		batch.ScopeSpans[0].Spans[0].Name = "invalid-\xff-utf8"

		p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

		assert.Contains(t, testRegistry.String(), "traces_target_info")
		lbls := labels.FromMap(map[string]string{
			"service":     "test-service",
			"span_name":   "test",
			"span_kind":   "SPAN_KIND_CLIENT",
			"status_code": "STATUS_CODE_OK",
			"job":         "test-service",
		})
		assert.Equal(t, 1.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))
		assert.Equal(t, 1.0, testutil.ToFloat64(invalidUTF8Counter))
	})

	t.Run("invalid target_info labels still emit span metrics", func(t *testing.T) {
		testRegistry, invalidUTF8Counter, p := newProcessor(t)
		defer p.Shutdown(context.Background())

		// Valid spans, but a resource attribute with an invalid UTF-8 value
		// poisons the target_info label set only.
		batch := test.MakeBatchWithAttributes(2, nil, []*common_v1.KeyValue{
			{
				Key: "res.attr",
				Value: &common_v1.AnyValue{
					Value: &common_v1.AnyValue_StringValue{StringValue: "invalid-\xff-utf8"},
				},
			},
		})

		p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

		// Span metrics still emit for both spans; target_info is rejected and
		// counted once per accepted span (pre-optimization contract).
		lbls := labels.FromMap(map[string]string{
			"service":     "test-service",
			"span_name":   "test",
			"span_kind":   "SPAN_KIND_CLIENT",
			"status_code": "STATUS_CODE_OK",
			"job":         "test-service",
		})
		assert.Equal(t, 2.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))
		assert.NotContains(t, testRegistry.String(), "traces_target_info")
		assert.Equal(t, 2.0, testutil.ToFloat64(invalidUTF8Counter))
	})
}

// TestSpanMetricsTargetInfoWithDisabledSubprocessors pins that target_info
// registration only depends on the span's primary labels being valid UTF-8,
// not on any span metric series actually being updated.
func TestSpanMetricsTargetInfoWithDisabledSubprocessors(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	filteredSpansCounter := prometheus.NewCounter(prometheus.CounterOpts{Name: "test_filtered_spans_total"})
	invalidUTF8Counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "test_invalid_utf8_spans_total"})

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableTargetInfo = true
	cfg.Subprocessors[Count] = false
	cfg.Subprocessors[Latency] = false
	cfg.Subprocessors[Size] = false

	p, err := New(cfg, testRegistry, filteredSpansCounter, invalidUTF8Counter)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(1, nil)
	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	var resAttr string
	for _, kv := range batch.Resource.Attributes {
		if kv.Key == "random.res.attr" {
			resAttr = kv.Value.GetStringValue()
		}
	}
	targetInfoLabels := labels.FromMap(map[string]string{
		"job":             "test-service",
		"random_res_attr": resAttr,
	})
	assert.Equal(t, 1.0, testRegistry.Query("traces_target_info", targetInfoLabels))
	assert.NotContains(t, testRegistry.String(), "traces_spanmetrics_calls_total")
	assert.Equal(t, 0.0, testutil.ToFloat64(invalidUTF8Counter))
}

// TestEnumStringFastPathsMatchProto guards the hand-rolled switches in
// spanKindString / statusCodeString against enum drift: for every known value
// they must return exactly what the generated proto String() returns, and
// unknown values must fall back to the generated String(). Correctness cannot
// drift for an added enum value — the switch's default returns String() — but
// the new value would silently take the slow map-lookup path, so the Len pins
// fail on any enum change to force the switches to be extended in step.
func TestEnumStringFastPathsMatchProto(t *testing.T) {
	require.Len(t, trace_v1.Span_SpanKind_name, 6, "Span_SpanKind enum changed: extend spanKindString and update this count")
	require.Len(t, trace_v1.Status_StatusCode_name, 3, "Status_StatusCode enum changed: extend statusCodeString and update this count")

	for value := range trace_v1.Span_SpanKind_name {
		kind := trace_v1.Span_SpanKind(value)
		require.Equal(t, kind.String(), spanKindString(kind))
	}
	for value := range trace_v1.Status_StatusCode_name {
		code := trace_v1.Status_StatusCode(value)
		require.Equal(t, code.String(), statusCodeString(code))
	}

	// Unknown values must fall back to the generated String().
	unknownKind := trace_v1.Span_SpanKind(math.MaxInt32)
	require.Equal(t, unknownKind.String(), spanKindString(unknownKind))
	unknownCode := trace_v1.Status_StatusCode(math.MaxInt32)
	require.Equal(t, unknownCode.String(), statusCodeString(unknownCode))
}
