package benchmark

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

// ranked builds a profile holding exactly the given ranked attributes, in the
// order given, for tests that check what queries they turn into.
func ranked(attrs ...attributes.Profile) *profile.BlockProfile {
	return &profile.BlockProfile{
		Attributes: &attributes.Profiles{Spans: 1, Ranked: attrs},
	}
}

func TestAttributeQueries(t *testing.T) {
	tests := []struct {
		name   string
		ranked []attributes.Profile
		want   map[string][]string
	}{
		{
			name: "no attributes gives no queries",
			want: map[string][]string{},
		},
		{
			name: "each attribute contributes its band's value to the band's queries",
			ranked: []attributes.Profile{
				profileAttr(attributes.ScopeSpan, "key", attributes.TypeString, false, "v1", "v2", "v3", "v4"),
				profileAttr(attributes.ScopeResource, "service.name", attributes.TypeString, false, "svc"),
			},
			want: map[string][]string{
				"high":   {`{ span.key = "v1" }`, `{ resource.service.name = "svc" }`},
				"medium": {`{ span.key = "v2" }`},
				"low":    {`{ span.key = "v4" }`},
			},
		},
		{
			name: "intrinsics are referenced bare, each type in its literal syntax",
			ranked: []attributes.Profile{
				profileAttr(attributes.ScopeIntrinsic, "name", attributes.TypeString, true, "test"),
				profileAttr(attributes.ScopeIntrinsic, "duration", attributes.TypeDuration, true, "1.5s"),
				profileAttr(attributes.ScopeIntrinsic, "status", attributes.TypeStatus, true, "ok"),
			},
			want: map[string][]string{
				"high": {`{ name = "test" }`, `{ duration = 1.5s }`, `{ status = ok }`},
			},
		},
		{
			name: "the same attribute in two types contributes a query each",
			ranked: []attributes.Profile{
				profileAttr(attributes.ScopeSpan, "foo", attributes.TypeString, false, "bar"),
				profileAttr(attributes.ScopeSpan, "foo", attributes.TypeInt, false, "42"),
			},
			want: map[string][]string{
				"high": {`{ span.foo = "bar" }`, `{ span.foo = 42 }`},
			},
		},
		{
			name: "names and values the lexer would split are quoted",
			ranked: []attributes.Profile{
				profileAttr(attributes.ScopeSpan, `foo bar"`, attributes.TypeString, false, `say "hi"\`),
				profileAttr(attributes.ScopeResource, "a=b", attributes.TypeString, false, "x"),
			},
			want: map[string][]string{
				"high": {`{ span."foo bar\"" = "say \"hi\"\\" }`, `{ resource."a=b" = "x" }`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attributeQueries(tt.ranked)
			require.Equal(t, tt.want, got)

			// Every generated query has to parse: one that does not fails
			// every execution it runs in.
			for band, queries := range got {
				for _, q := range queries {
					_, err := traceql.Parse(q)
					require.NoError(t, err, "band %s: %q", band, q)
				}
			}
		})
	}
}

func TestAttributeCases(t *testing.T) {
	noValues := profileAttr(attributes.ScopeSpan, "latency", attributes.TypeInt, false)

	tests := []struct {
		name string
		prof *profile.BlockProfile
		want []string
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
			name: "one case per band the picked attributes have values for",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, "key", attributes.TypeString, false, "v1", "v2", "v3"),
			),
			want: []string{"search/ranked/high", "search/ranked/medium", "search/ranked/low"},
		},
		{
			name: "single-value attributes generate only the high band",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, "key", attributes.TypeString, false, "value"),
			),
			want: []string{"search/ranked/high"},
		},
		{
			name: "quantile-only attributes generate nothing yet",
			prof: ranked(noValues),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attributeCases(tt.prof)
			require.Len(t, got, len(tt.want))
			for i, c := range got {
				require.Equal(t, tt.want[i], c.id)
				require.Equal(t, apiSearch, c.api)
			}
		})
	}
}

// A band case runs each picked attribute's query per shard, the way
// traceid/present runs one execution per ID.
func TestAttributeCasesExecutions(t *testing.T) {
	prof := ranked(
		profileAttr(attributes.ScopeSpan, "a", attributes.TypeString, false, "v1", "v2", "v3"),
		profileAttr(attributes.ScopeSpan, "b", attributes.TypeString, false, "w1", "w2", "w3"),
		profileAttr(attributes.ScopeSpan, "c", attributes.TypeString, false, "x"),
	)
	prof.Block = &backend.BlockMeta{}

	cases := attributeCases(prof)
	require.Len(t, cases, 3)

	shards := []Shard{{Index: 0, StartPage: 0, TotalPages: 2}, {Index: 1, StartPage: 2, TotalPages: 2}}
	for _, c := range cases {
		executions, err := c.executions(prof, shards, RunOptions{})
		require.NoError(t, err, c.id)
		queries := attributeQueries(pickAttributes(prof.Attributes, rankedAttributesPerGroup, hasValues))[c.id[len("search/ranked/"):]]
		require.Len(t, executions, len(shards)*len(queries), c.id)
	}
}

// Each group of attributes — a dedicated column or not — contributes at most
// rankedAttributesPerGroup attributes' queries to a band's case.
func TestAttributeQueriesCapEachGroup(t *testing.T) {
	var attrs []attributes.Profile
	for i := range rankedAttributesPerGroup + 2 {
		attrs = append(attrs,
			profileAttr(attributes.ScopeSpan, fmt.Sprintf("dedicated%d", i), attributes.TypeString, true, "v"),
			profileAttr(attributes.ScopeSpan, fmt.Sprintf("generic%d", i), attributes.TypeString, false, "v"),
		)
	}

	queries := attributeQueries(pickAttributes(ranked(attrs...).Attributes, rankedAttributesPerGroup, hasValues))
	require.Len(t, queries, 1, "single-value attributes give only the high band")
	require.Len(t, queries["high"], 2*rankedAttributesPerGroup)
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
		"search/nopredicate", "search/ranked/high", "search/ranked/medium", "search/ranked/low",
		"metrics/rate", "metrics/rate-by-service",
		"metadata/tagnames/none", "metadata/tagnames/resource", "metadata/tagnames/span",
		"metadata/tagnames/event", "metadata/tagnames/link", "metadata/tagnames/instrumentation",
	}, got)
}
