package spanfilter

import (
	"os"
	"testing"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

func TestSpanFilter_policyMatch(t *testing.T) {
	cases := []struct {
		policy   *config.PolicyMatch
		resource *v1.Resource
		span     *trace_v1.Span
		expect   bool
	}{
		{
			expect: true,
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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

func TestSpanFilter_policyMatchIntrinsicAttrs(t *testing.T) {
	cases := []struct {
		policy *config.PolicyMatch
		span   *trace_v1.Span
		expect bool
	}{
		{
			expect: true,
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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

func TestSpanFilter_policyMatchAttrs(t *testing.T) {
	cases := []struct {
		policy *config.PolicyMatch
		attrs  []*common_v1.KeyValue
		expect bool
	}{
		// Single string match
		{
			expect: true,
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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
			policy: &config.PolicyMatch{
				MatchType: config.Regex,
				Attributes: []config.MatchPolicyAttribute{
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

func TestSpanFilter_stringMatch(t *testing.T) {
	cases := []struct {
		matchType config.MatchType
		s         string
		pattern   string
		expect    bool
	}{
		{
			matchType: config.Strict,
			s:         "foo",
			pattern:   "foo",
			expect:    true,
		},
		{
			matchType: config.Strict,
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

func BenchmarkSpanFilter_applyFilterPolicyNone(b *testing.B) {
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

	policies := []config.FilterPolicy{}

	benchmarkFilterPolicy(b, policies, batch)
}

func BenchmarkSpanFilter_applyFilterPolicySmall(b *testing.B) {
	// Read the file generated above
	data, err := os.ReadFile("testbatch100k")
	require.NoError(b, err)
	batch := &trace_v1.ResourceSpans{}
	err = batch.Unmarshal(data)
	require.NoError(b, err)

	policies := []config.FilterPolicy{
		{
			Include: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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

func BenchmarkSpanFilter_applyFilterPolicyMedium(b *testing.B) {
	// Read the file generated above
	data, err := os.ReadFile("testbatch100k")
	require.NoError(b, err)
	batch := &trace_v1.ResourceSpans{}
	err = batch.Unmarshal(data)
	require.NoError(b, err)

	policies := []config.FilterPolicy{
		{
			Include: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
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

func benchmarkFilterPolicy(b *testing.B, policies []config.FilterPolicy, batch *trace_v1.ResourceSpans) {
	filter, err := NewSpanFilter(policies)
	require.NoError(b, err)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		pushspans(&tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}}, filter)
	}
}

func pushspans(req *tempopb.PushSpansRequest, filter *SpanFilter) {
	for _, rs := range req.Batches {
		for _, ils := range rs.ScopeSpans {
			for _, span := range ils.Spans {
				filter.ApplyFilterPolicy(rs.Resource, span)
			}
		}
	}
}
