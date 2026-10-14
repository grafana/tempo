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

func TestSearchCases(t *testing.T) {
	tests := []struct {
		name string
		prof *profile.BlockProfile
		want []struct{ id, query string }
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
			name: "one case per attribute per band of its own values",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, "key", attributes.TypeString, false, "v1", "v2", "v3", "v4"),
				profileAttr(attributes.ScopeResource, "service.name", attributes.TypeString, false, "svc"),
			),
			want: []struct{ id, query string }{
				{"search/ranked/high/span.key", `{ span.key = "v1" }`},
				{"search/ranked/medium/span.key", `{ span.key = "v2" }`},
				{"search/ranked/low/span.key", `{ span.key = "v4" }`},
				{"search/ranked/high/resource.service.name", `{ resource.service.name = "svc" }`},
			},
		},
		{
			name: "intrinsics are referenced bare, each type in its literal syntax",
			prof: ranked(
				profileAttr(attributes.ScopeIntrinsic, "name", attributes.TypeString, true, "test"),
				profileAttr(attributes.ScopeIntrinsic, "duration", attributes.TypeDuration, true, "1.5s"),
				profileAttr(attributes.ScopeIntrinsic, "status", attributes.TypeStatus, true, "ok"),
			),
			want: []struct{ id, query string }{
				{"search/ranked/high/intrinsic.name", `{ name = "test" }`},
				{"search/ranked/high/intrinsic.duration", `{ duration = 1.5s }`},
				{"search/ranked/high/intrinsic.status", `{ status = ok }`},
			},
		},
		{
			name: "the same attribute in two types lands a case each",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, "foo", attributes.TypeString, false, "bar"),
				profileAttr(attributes.ScopeSpan, "foo", attributes.TypeInt, false, "42"),
			),
			want: []struct{ id, query string }{
				{"search/ranked/high/span.foo.string", `{ span.foo = "bar" }`},
				{"search/ranked/high/span.foo.int", `{ span.foo = 42 }`},
			},
		},
		{
			name: "names and values the lexer would split are quoted",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, `foo bar"`, attributes.TypeString, false, `say "hi"\`),
				profileAttr(attributes.ScopeResource, "a=b", attributes.TypeString, false, "x"),
			),
			want: []struct{ id, query string }{
				{`search/ranked/high/span.foo bar"`, `{ span."foo bar\"" = "say \"hi\"\\" }`},
				{`search/ranked/high/resource.a=b`, `{ resource."a=b" = "x" }`},
			},
		},
		{
			name: "a quantile-only attribute generates nothing yet",
			prof: ranked(
				profileAttr(attributes.ScopeSpan, "latency", attributes.TypeInt, false),
			),
		},
		{
			name: "a scope with no TraceQL reference is skipped",
			prof: ranked(
				profileAttr("event", "foo", attributes.TypeString, false, "x"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attributeCases(tt.prof)
			require.Len(t, got, len(tt.want))

			for i, c := range got {
				require.Equal(t, tt.want[i].id, c.id)
				require.Equal(t, tt.want[i].query, c.query)
				require.Equal(t, apiSearch, c.api)

				// Every generated query has to parse: one that does not
				// fails every execution of its case.
				_, err := traceql.Parse(c.query)
				require.NoError(t, err, "case %s: %q", c.id, c.query)
			}
		})
	}
}

// Each group of attributes — a dedicated column or not — contributes at most
// rankedAttributesPerGroup cases' worth of attributes, dedicated first.
func TestSearchCasesCapsEachGroup(t *testing.T) {
	var attrs []attributes.Profile
	for i := range rankedAttributesPerGroup + 2 {
		attrs = append(attrs,
			profileAttr(attributes.ScopeSpan, fmt.Sprintf("dedicated%d", i), attributes.TypeString, true, "v"),
			profileAttr(attributes.ScopeSpan, fmt.Sprintf("generic%d", i), attributes.TypeString, false, "v"),
		)
	}

	cases := attributeCases(ranked(attrs...))
	require.Len(t, cases, 2*rankedAttributesPerGroup)
	for i, c := range cases {
		if i < rankedAttributesPerGroup {
			require.Equal(t, fmt.Sprintf("search/ranked/high/span.dedicated%d", i), c.id)
		} else {
			require.Equal(t, fmt.Sprintf("search/ranked/high/span.generic%d", i-rankedAttributesPerGroup), c.id)
		}
	}
}

// Generated cases shard the way the fixed search case does: one execution per
// shard, over the block's own window.
func TestSearchCasesShard(t *testing.T) {
	prof := ranked(profileAttr(attributes.ScopeSpan, "key", attributes.TypeString, false, "value"))
	prof.Block = &backend.BlockMeta{}

	cases := attributeCases(prof)
	require.Len(t, cases, 1)

	shards := []Shard{{Index: 0, StartPage: 0, TotalPages: 2}, {Index: 1, StartPage: 2, TotalPages: 2}}
	executions, err := cases[0].executions(prof, shards, RunOptions{})
	require.NoError(t, err)
	require.Len(t, executions, len(shards))
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
		"search/nopredicate", "search/ranked/high/span.most", "search/ranked/medium/span.most",
		"search/ranked/low/span.most", "search/ranked/high/span.some",
		"metrics/rate", "metrics/rate-by-service",
		"metadata/tagnames/none", "metadata/tagnames/resource", "metadata/tagnames/span",
		"metadata/tagnames/event", "metadata/tagnames/link", "metadata/tagnames/instrumentation",
	}, got)
}
