package spanfilter

import (
	"fmt"
	"os"
	"testing"

	"github.com/grafana/tempo/pkg/spanfilter/policymatch"

	"github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

func TestSpanFilter_NewSpanFilter(t *testing.T) {
	cases := []struct {
		name   string
		cfg    []config.FilterPolicy
		filter *SpanFilter
		err    error
	}{
		{
			name:   "empty config",
			cfg:    []config.FilterPolicy{},
			filter: &SpanFilter{},
			err:    nil,
		},
		{
			name:   "nil config",
			cfg:    nil,
			filter: &SpanFilter{},
			err:    nil,
		},
		{
			name:   "nil config",
			cfg:    nil,
			filter: &SpanFilter{},
			err:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSpanFilter(tc.cfg)
			require.NoError(t, err)
		})
	}
}

func Test_splitPolicy_Match(t *testing.T) {
	cases := []struct {
		policy   *config.PolicyMatch
		resource *v1.Resource
		span     *trace_v1.Span
		expect   bool
		testName string
	}{
		{
			testName: "most basic span kind matching",
			expect:   true,
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "span.kind",
						Value: "SPAN_KIND_CLIENT",
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{},
			},
			span: &trace_v1.Span{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "kind",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "SPAN_KIND_CLIENT",
							},
						},
					},
				},
			},
		},
		{
			testName: "most basic intrinsic kind matching",
			expect:   true,
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: "SPAN_KIND_CLIENT",
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{},
			},
			span: &trace_v1.Span{
				Kind: trace_v1.Span_SPAN_KIND_CLIENT,
			},
		},
		{
			testName: "simple matching",
			expect:   true,
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: "SPAN_KIND_CLIENT",
					},
					{
						Key:   "span.status.code",
						Value: "STATUS_CODE_OK",
					},
					{
						Key:   "resource.location",
						Value: "earth",
					},
					{
						Key:   "resource.name",
						Value: "test",
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
								StringValue: "test",
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
				Kind: trace_v1.Span_SPAN_KIND_CLIENT,
				Attributes: []*common_v1.KeyValue{
					{
						Key: "status.code",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "STATUS_CODE_OK",
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
						Key:   "kind",
						Value: "SPAN_KIND_CLIENT",
					},
					{
						Key:   "status",
						Value: "STATUS_CODE_OK",
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{},
			},
			span: &trace_v1.Span{
				Kind:       trace_v1.Span_SPAN_KIND_CLIENT,
				Status:     &trace_v1.Status{Message: "OK", Code: trace_v1.Status_STATUS_CODE_OK},
				Attributes: []*common_v1.KeyValue{},
			},
		},
		{
			testName: "resource matching",
			expect:   true,
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
				Attributes: []*common_v1.KeyValue{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(t *testing.T) {
			policy, err := getSplitPolicy(tc.policy)
			require.NoError(t, err)
			require.NotNil(t, policy)
			r := policy.Match(tc.resource, tc.span)
			require.Equal(t, tc.expect, r)
		})
	}
}

func TestSpanMetrics_applyFilterPolicy(t *testing.T) {
	cases := []struct {
		name           string
		err            error
		filterPolicies []config.FilterPolicy
		expect         bool
		resource       *v1.Resource
		span           *trace_v1.Span
	}{
		{
			name:           "no policies matches",
			err:            nil,
			expect:         true,
			filterPolicies: []config.FilterPolicy{},
		},
		{
			name:           "nil policies matches",
			err:            nil,
			expect:         true,
			filterPolicies: nil,
		},
		{
			name:   "non nil policy with nil include/exclude fails",
			err:    fmt.Errorf("invalid filter policy; policies must have at least an `include` or `exclude`: {<nil> <nil>}"),
			expect: false,
			filterPolicies: []config.FilterPolicy{{
				Include: nil,
				Exclude: nil,
			}},
		},
		{
			name:   "a matching policy",
			err:    nil,
			expect: true,
			filterPolicies: []config.FilterPolicy{
				{
					Include: &config.PolicyMatch{
						MatchType: config.Strict,
						Attributes: []config.MatchPolicyAttribute{
							{
								Key:   "kind",
								Value: "SPAN_KIND_SERVER",
							},
							{
								Key:   "resource.location",
								Value: "earth",
							},
						},
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "name",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "test",
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
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "a non-matching include policy",
			err:    nil,
			expect: false,
			filterPolicies: []config.FilterPolicy{
				{
					Include: &config.PolicyMatch{
						MatchType: config.Strict,
						Attributes: []config.MatchPolicyAttribute{
							{
								Key:   "kind",
								Value: "SPAN_KIND_CLIENT",
							},
							{
								Key:   "resource.location",
								Value: "earth",
							},
						},
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "name",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "test",
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
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "a matching include with rejecting exclude policy",
			err:    nil,
			expect: false,
			filterPolicies: []config.FilterPolicy{
				{
					Include: &config.PolicyMatch{
						MatchType: config.Strict,
						Attributes: []config.MatchPolicyAttribute{
							{
								Key:   "kind",
								Value: "SPAN_KIND_SERVER",
							},
							{
								Key:   "resource.location",
								Value: "earth",
							},
						},
					},
					Exclude: &config.PolicyMatch{
						MatchType: config.Regex,
						Attributes: []config.MatchPolicyAttribute{
							{
								Key:   "resource.othervalue",
								Value: "something.*",
							},
						},
					},
				},
			},
			resource: &v1.Resource{
				Attributes: []*common_v1.KeyValue{
					{
						Key: "name",
						Value: &common_v1.AnyValue{
							Value: &common_v1.AnyValue_StringValue{
								StringValue: "test",
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
				Kind: trace_v1.Span_SPAN_KIND_SERVER,
				Status: &trace_v1.Status{
					Code: trace_v1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sf, err := NewSpanFilter(tc.filterPolicies)
			require.Equal(t, tc.err, err)
			if err != nil {
				return
			}
			x := sf.ApplyFilterPolicy(tc.resource, tc.span)
			require.Equal(t, tc.expect, x)
		})
	}
}

func TestSpanFilter_getSplitPolicy(t *testing.T) {
	cases := []struct {
		policy *config.PolicyMatch
		split  *splitPolicy
		name   string
	}{
		{
			name: "basic kind matching",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "kind",
						Value: "SPAN_KIND_CLIENT",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewPolicyMatch(
					policymatch.NewMatchStrictPolicyAttribute("kind", trace_v1.Span_SPAN_KIND_CLIENT),
				),
			},
		},
		{
			name: "basic status matching",
			policy: &config.PolicyMatch{
				MatchType: config.Strict,
				Attributes: []config.MatchPolicyAttribute{
					{
						Key:   "status",
						Value: "STATUS_CODE_OK",
					},
				},
			},
			split: &splitPolicy{
				IntrinsicMatch: policymatch.NewPolicyMatch(
					policymatch.NewMatchStrictPolicyAttribute("status", trace_v1.Status_STATUS_CODE_OK),
				),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := getSplitPolicy(tc.policy)
			require.NoError(t, err)
			require.NotNil(t, s)
			require.NotNil(t, s.IntrinsicMatch)
			require.NotNil(t, s.SpanMatch)
			require.NotNil(t, s.ResourceMatch)

			if tc.split.IntrinsicMatch != nil {
				require.Equal(t, tc.split.IntrinsicMatch, s.IntrinsicMatch)
			}
			if tc.split.SpanMatch != nil {
				require.Equal(t, tc.split.SpanMatch, s.SpanMatch)
			}
			if tc.split.ResourceMatch != nil {
				require.Equal(t, tc.split.ResourceMatch, s.ResourceMatch)
			}
		})
	}
}

func BenchmarkSpanFilter_applyFilterPolicyNone(b *testing.B) {
	// Generate a batch of 100k spans
	//r, done := test.NewRandomBatcher()
	//defer done()
	//batch := r.GenerateBatch(1e6)
	//data, _ := batch.Marshal()
	//_ = os.WriteFile("testbatch100k", data, 0600)

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
