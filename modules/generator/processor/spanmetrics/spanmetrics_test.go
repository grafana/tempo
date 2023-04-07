package spanmetrics

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
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

func TestSpanMetrics_dimensions(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.IntrinsicDimensions.SpanKind = false
	cfg.IntrinsicDimensions.StatusMessage = true
	cfg.Dimensions = []string{"foo", "bar", "does-not-exist"}

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

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.Dimensions = []string{"span.kind", "span_name"}
	cfg.IntrinsicDimensions.SpanKind = false

	p := New(cfg, testRegistry)
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

func TestSpanMetrics_applyFilterPolicy(t *testing.T) {
	cases := []struct {
		filterPolicies     []FilterPolicy
		expectedMatches    float64
		expectedRejections float64
	}{
		{
			expectedMatches:    10.0,
			expectedRejections: 0.0,
			filterPolicies: []FilterPolicy{
				{

					Include: &PolicyMatch{
						MatchType: Strict,
						Attributes: []MatchPolicyAttribute{
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
			filterPolicies: []FilterPolicy{
				{

					Include: &PolicyMatch{
						MatchType: Strict,
						Attributes: []MatchPolicyAttribute{
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
			filterPolicies: []FilterPolicy{
				{
					Exclude: &PolicyMatch{
						MatchType: Strict,
						Attributes: []MatchPolicyAttribute{
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
			filterPolicies: []FilterPolicy{
				{
					Include: &PolicyMatch{
						MatchType: Regex,
						Attributes: []MatchPolicyAttribute{
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
			assert.Equal(t, tc.expectedRejections, testRegistry.Query(metricFilterDropsTotal, nil))

		})
	}

}

func TestSpanMetrics_policyMatch(t *testing.T) {
	cases := []struct {
		policy   *PolicyMatch
		resource *v1.Resource
		span     *trace_v1.Span
		expect   bool
	}{
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "span.kind",
						Value: "client",
					},
					{
						Key:   "resource.location",
						Value: "earth",
					},
					{
						Key:   "resource.name",
						Value: "goodiegoodie",
					},
					{
						Key:   "resource.othervalue",
						Value: "somethinginteresting",
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "name",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "goodiegoodie",
							},
						},
					},
					{
						Key: "location",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "earth",
							},
						},
					},
					{
						Key: "othervalue",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "somethinginteresting",
							},
						},
					},
				},
			},
			span: &trace_v1.Span{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "kind",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "client",
							},
						},
					},
				},
			},
		},
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "span.kind",
						Value: "client",
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "location",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "earth",
							},
						},
					},
					{
						Key: "othervalue",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "somethinginteresting",
							},
						},
					},
				},
			},
			span: &trace_v1.Span{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "kind",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "client",
							},
						},
					},
				},
			},
		},
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "resource.location",
						Value: "earth",
					},
					{
						Key:   "resource.othervalue",
						Value: "somethinginteresting",
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "location",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "earth",
							},
						},
					},
					{
						Key: "othervalue",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "somethinginteresting",
							},
						},
					},
				},
			},
			span: &trace_v1.Span{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "kind",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "client",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		r := policyMatch(getSplitPolicy(tc.policy), tc.resource, tc.span)
		require.Equal(t, tc.expect, r)
	}
}

func TestSpanMetrics_policyMatchIntrinsicAttrs(t *testing.T) {
	cases := []struct {
		policy *PolicyMatch
		span   *trace_v1.Span
		expect bool
	}{
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: "SPAN_KIND_SERVER",
					},
					{
						Key:   "status",
						Value: "STATUS_CODE_OK",
					},
					{
						Key:   "name",
						Value: "goodiegoodie",
					},
				},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "goodiegoodie",
			},
		},
	}

	for _, tc := range cases {
		r := policyMatchIntrinsicAttrs(tc.policy, tc.span)
		require.Equal(t, tc.expect, r)
	}

}

func TestSpanMetrics_policyMatchAttrs(t *testing.T) {
	cases := []struct {
		policy *PolicyMatch
		attrs  []*common_v1.KeyValue
		expect bool
	}{
		// Single string match
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "foo",
						Value: "bar",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "foo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
			},
		},
		// Multiple string match
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "foo",
						Value: "bar",
					},
					{
						Key:   "otherfoo",
						Value: "notbar",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "foo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
				{
					Key: "otherfoo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "notbar",
						},
					},
				},
			},
		},
		// Multiple string non match
		{
			expect: false,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "foo",
						Value: "bar",
					},
					{
						Key:   "otherfoo",
						Value: "nope",
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "foo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "bar",
						},
					},
				},
				{
					Key: "otherfoo",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "notbar",
						},
					},
				},
			},
		},
		// Combination match
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "one",
						Value: "1",
					},
					{
						Key:   "oneone",
						Value: 11,
					},
					{
						Key:   "oneonepointone",
						Value: 11.1,
					},
					{
						Key:   "matching",
						Value: true,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "one",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "1",
						},
					},
				},
				{
					Key: "oneone",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_IntValue{
							IntValue: 11,
						},
					},
				},
				{
					Key: "oneonepointone",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_DoubleValue{
							DoubleValue: 11.1,
						},
					},
				},
				{
					Key: "matching",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_BoolValue{
							BoolValue: true,
						},
					},
				},
			},
		},
		// Regex basic match
		{
			expect: true,
			policy: &PolicyMatch{
				MatchType: Regex,
				Attributes: []MatchPolicyAttribute{
					{
						Key:   "dd",
						Value: `\d\d\w{5}`,
					},
				},
			},
			attrs: []*common_v1.KeyValue{
				{
					Key: "dd",
					Value: &common_v1.AnyValue{
						Value: &common_v1.AnyValue_StringValue{
							StringValue: "11xxxxx",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		r := policyMatchAttrs(tc.policy, tc.attrs)
		require.Equal(t, tc.expect, r)
	}

}

func TestSpanMetrics_stringMatch(t *testing.T) {
	cases := []struct {
		matchType MatchType
		s         string
		pattern   string
		expect    bool
	}{
		{
			matchType: Strict,
			s:         "foo",
			pattern:   "foo",
			expect:    true,
		},
		{
			matchType: Strict,
			s:         "foo",
			pattern:   "bar",
			expect:    false,
		},
	}

	for _, tc := range cases {
		r := stringMatch(tc.matchType, tc.s, tc.pattern)
		require.Equal(t, tc.expect, r)
	}

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

	policies := []FilterPolicy{}

	benchmarkFilterPolicy(b, policies, batch)
}

func BenchmarkSpanMetrics_applyFilterPolicySmall(b *testing.B) {
	// Read the file generated above
	data, err := os.ReadFile("testbatch100k")
	require.NoError(b, err)
	batch := &trace_v1.ResourceSpans{}
	err = batch.Unmarshal(data)
	require.NoError(b, err)

	policies := []FilterPolicy{
		{
			Include: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
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

	policies := []FilterPolicy{
		{
			Include: &PolicyMatch{
				MatchType: Strict,
				Attributes: []MatchPolicyAttribute{
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

func benchmarkFilterPolicy(b *testing.B, policies []FilterPolicy, batch *trace_v1.ResourceSpans) {
	testRegistry := registry.NewTestRegistry()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.FilterPolicies = policies
	p := New(cfg, testRegistry)
	defer p.Shutdown(context.Background())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})
	}
}
