package benchmark

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

// ranked builds a profile holding exactly the given ranked attributes, in the
// order given, for tests that check what queries they turn into.
func ranked(attrs ...attributes.Profile) *profile.BlockProfile {
	return &profile.BlockProfile{
		Attributes: &attributes.Profiles{Spans: 1, Ranked: attrs},
	}
}

func TestAttributeCases(t *testing.T) {
	noValues := profileAttr(attributes.ScopeSpan, "latency", attributes.TypeInt, false)

	tests := []struct {
		name    string
		prof    *profile.BlockProfile
		want    []string
		queries []string
	}{
		{
			name: "profile without attributes generates nothing",
			prof: &profile.BlockProfile{},
		},
		{
			name: "profile with no ranked attributes generates nothing",
			prof: ranked(),
		},
		{
			name: "one case per attribute and band it has a value for",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, "key", attributes.TypeString, false, "v1", "v2", "v3"),
				profileAttr(attributes.ScopeResource, "service.name", attributes.TypeString, false, "svc"),
			),
			want:    []string{"search/ranked_generic/high/span.key", "search/ranked_generic/medium/span.key", "search/ranked_generic/low/span.key", "search/ranked_generic/high/resource.service.name"},
			queries: []string{`{ span.key = "v1" }`, `{ span.key = "v2" }`, `{ span.key = "v3" }`, `{ resource.service.name = "svc" }`},
		},
		{
			name: "the same attribute in two types gets a case each",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, "foo", attributes.TypeString, false, "bar"),
				profileAttr(attributes.ScopeSpan, "foo", attributes.TypeInt, false, "42"),
			),
			want:    []string{"search/ranked_generic/high/span.foo.string", "search/ranked_generic/high/span.foo.int"},
			queries: []string{`{ span.foo = "bar" }`, `{ span.foo = 42 }`},
		},
		{
			name: "quantile-only attributes generate nothing yet",
			prof: ranked(noValues),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := casesOf(attributeCases(tt.prof), apiSearch)
			require.Len(t, got, len(tt.want))
			for i, c := range got {
				require.Equal(t, tt.want[i], c.id)
				require.Equal(t, apiSearch, c.api)
				require.Equal(t, tt.queries[i], c.query)
			}
		})
	}
}

// A generated search runs its query once per shard, like the fixed search.
func TestAttributeCasesExecutions(t *testing.T) {
	prof := ranked(
		profileAttr(attributes.ScopeSpan, "a", attributes.TypeString, false, "v1", "v2", "v3"),
		profileAttr(attributes.ScopeSpan, "c", attributes.TypeString, false, "x"),
	)
	prof.Block = &backend.BlockMeta{}

	cases := casesOf(attributeCases(prof), apiSearch)
	require.Len(t, cases, 4)

	shards := []Shard{{Index: 0, StartPage: 0, TotalPages: 2}, {Index: 1, StartPage: 2, TotalPages: 2}}
	for _, c := range cases {
		executions, err := c.executions(prof, shards, RunOptions{})
		require.NoError(t, err, c.id)
		require.Len(t, executions, len(shards), c.id)
	}
}

// Each group of attributes — a dedicated column or not — contributes at most
// rankedAttributesPerGroup attributes' searches.
func TestAttributeCasesCapEachGroup(t *testing.T) {
	var attrs []attributes.Profile
	for i := range rankedAttributesPerGroup + 2 {
		attrs = append(attrs,
			profileAttr(attributes.ScopeSpan, fmt.Sprintf("dedicated%d", i), attributes.TypeString, true, "v"),
			profileAttr(attributes.ScopeSpan, fmt.Sprintf("generic%d", i), attributes.TypeString, false, "v"),
		)
	}

	require.Len(t, casesOf(attributeCases(ranked(attrs...)), apiSearch), 2*rankedAttributesPerGroup)
}

// allCases keeps the cases of an API together, so the summary of a comparison
// shows one group per API: the generated searches join the fixed search case,
// not a group of their own after the metadata cases.
func TestAllCasesKeepsAPIsTogether(t *testing.T) {
	prof := ranked(
		profileAttr(attributes.ScopeSpan, "most", attributes.TypeString, false, "a", "b", "c"),
		profileAttr(attributes.ScopeSpan, "some", attributes.TypeString, false, "x"),
	)

	var got []string
	for _, c := range allCases(prof) {
		got = append(got, c.id)
	}
	require.Equal(t, []string{
		"traceid/present", "traceid/absent",
		"search/nopredicate",
		"search/ranked_generic/high/span.most", "search/ranked_generic/medium/span.most", "search/ranked_generic/low/span.most",
		"search/ranked_generic/high/span.some",
		"metrics/rate", "metrics/rate-by-service",
		"metrics/drilldown/rate", "metrics/drilldown/errors", "metrics/drilldown/duration", "metrics/drilldown/by-service",
		"metrics/ranked_generic/rate/high/span.most", "metrics/ranked_generic/rate/medium/span.most", "metrics/ranked_generic/rate/low/span.most",
		"metrics/ranked_generic/rate/high/span.some",
		"metrics/ranked_generic/quantile/high/span.most", "metrics/ranked_generic/quantile/medium/span.most", "metrics/ranked_generic/quantile/low/span.most",
		"metrics/ranked_generic/quantile/high/span.some",
		"metadata/tagnames/none", "metadata/tagnames/resource", "metadata/tagnames/span",
		"metadata/tagnames/event", "metadata/tagnames/link", "metadata/tagnames/instrumentation",
		"metadata/ranked_generic/tagvalues/span.most", "metadata/ranked_generic/tagvalues/span.some",
	}, got)
}

func casesOf(cases []benchCase, api string) []benchCase {
	var out []benchCase
	for _, c := range cases {
		if c.api == api {
			out = append(out, c)
		}
	}
	return out
}
