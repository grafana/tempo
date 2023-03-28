package spanmetrics

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/registry"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

var (
	metricSpansDiscarded = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_spans_discarded_total",
		Help:      "The total number of discarded spans received per tenant",
	}, []string{"tenant", "reason"})
)

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
		"job":         "test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"instance":    "",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_target_info", lbls))
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
		"job":            "test-service",
		"span_name":      "test",
		"status_code":    "STATUS_CODE_OK",
		"status_message": "OK",
		"instance":       "",
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
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_target_info", lbls))
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
		"job":         "test-service",
		"span_name":   "test",
		"status_code": "STATUS_CODE_OK",
		"instance":    "",
		"__span_kind": "colliding_kind",
		"__span_name": "colliding_name",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_target_info", lbls))
}

func TestJobLabelWithNamespaceAndInstanceID(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}

	p := New(cfg, testRegistry)
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
		"job":         "test-namespace/test-service",
		"span_name":   "test",
		"span_kind":   "SPAN_KIND_CLIENT",
		"status_code": "STATUS_CODE_OK",
		"instance":    "abc-instance-id-test-def",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_target_info", lbls))
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

func TestSpanMetricsDimensionMapping(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.IntrinsicDimensions.SpanKind = false
	cfg.IntrinsicDimensions.StatusMessage = true
	cfg.Dimensions = []string{"foo", "bar"}
	cfg.DimensionMappings = []DimensionMappings{
		{
			Label: "foo",
			Replacement: "cat",
		},
	}

	p := New(cfg, testRegistry)
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
		"job":            "test-service",
		"span_name":      "test",
		"status_code":    "STATUS_CODE_OK",
		"status_message": "OK",
		"instance":       "",
		"cat":            "foo-value",
		"bar":            "bar-value",
	})

	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_calls_total", lbls))

	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 10.0, testRegistry.Query("traces_spanmetrics_latency_sum", lbls))
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_target_info", lbls))
}

func withLe(lbls labels.Labels, le float64) labels.Labels {
	lb := labels.NewBuilder(lbls)
	lb = lb.Set(labels.BucketLabel, strconv.FormatFloat(le, 'f', -1, 64))
	return lb.Labels(nil)
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
