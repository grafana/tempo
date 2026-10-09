package benchmark

import (
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

// attr builds a ranked attribute with one value, carried by the given share of
// spans, worth the given bytes.
func attr(scope, name, typ string, totalBytes uint64, sel float64, value string) attributes.Profile {
	return attributes.Profile{
		Scope: scope, Name: name, Type: typ, TotalBytes: totalBytes,
		Values: []attributes.Value{{Value: value, Selectivity: sel}},
	}
}

func TestPhase2Cases(t *testing.T) {
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
			name: "one case per band, each over the band's most sizeable attribute",
			prof: ranked(
				attr(attributes.ScopeSpan, "key", attributes.TypeString, 100, 0.9, "value"),
				attr(attributes.ScopeResource, "service.name", attributes.TypeString, 50, 0.9, "test-service"),
				attr(attributes.ScopeSpan, "http.status_code", attributes.TypeInt, 40, 0.3, "200"),
				attr(attributes.ScopeSpan, "error", attributes.TypeBool, 10, 0.01, "true"),
			),
			want: []struct{ id, query string }{
				{"search/attr/high", `{ span.key = "value" }`},
				{"search/attr/medium", `{ span.http.status_code = 200 }`},
				{"search/attr/low", `{ span.error = true }`},
			},
		},
		{
			name: "cases are ordered high, medium, low, whatever the byte order",
			prof: ranked(
				attr(attributes.ScopeSpan, "lowbig", attributes.TypeString, 100, 0.01, "x"),
				attr(attributes.ScopeSpan, "highsmall", attributes.TypeString, 10, 0.9, "y"),
			),
			want: []struct{ id, query string }{
				{"search/attr/high", `{ span.highsmall = "y" }`},
				{"search/attr/low", `{ span.lowbig = "x" }`},
			},
		},
		{
			name: "intrinsics are referenced bare, each type in its literal syntax",
			prof: ranked(
				attr(attributes.ScopeIntrinsic, "name", attributes.TypeString, 40, 0.9, "test"),
				attr(attributes.ScopeIntrinsic, "duration", attributes.TypeDuration, 30, 0.3, "1.5s"),
				attr(attributes.ScopeIntrinsic, "status", attributes.TypeStatus, 20, 0.01, "ok"),
			),
			want: []struct{ id, query string }{
				{"search/attr/high", `{ name = "test" }`},
				{"search/attr/medium", `{ duration = 1.5s }`},
				{"search/attr/low", `{ status = ok }`},
			},
		},
		{
			name: "a kind intrinsic is its own literal syntax",
			prof: ranked(
				attr(attributes.ScopeIntrinsic, "kind", attributes.TypeKind, 10, 0.9, "server"),
			),
			want: []struct{ id, query string }{
				{"search/attr/high", `{ kind = server }`},
			},
		},
		{
			name: "a quantile-only attribute generates nothing yet",
			prof: ranked(
				attributes.Profile{
					Scope: attributes.ScopeSpan, Name: "latency", Type: attributes.TypeInt,
					Quantiles: []attributes.Quantile{{Q: 0.99, Value: "100", Selectivity: 0.01}},
				},
			),
		},
		{
			name: "the same attribute in two types lands per its selectivity",
			prof: ranked(
				attr(attributes.ScopeSpan, "foo", attributes.TypeString, 100, 0.9, "bar"),
				attr(attributes.ScopeSpan, "foo", attributes.TypeInt, 50, 0.01, "42"),
			),
			want: []struct{ id, query string }{
				{"search/attr/high", `{ span.foo = "bar" }`},
				{"search/attr/low", `{ span.foo = 42 }`},
			},
		},
		{
			name: "names and values the lexer would split are quoted",
			prof: ranked(
				attr(attributes.ScopeSpan, `foo bar"`, attributes.TypeString, 10, 0.01, `say "hi"\`),
				attr(attributes.ScopeResource, "a=b", attributes.TypeString, 20, 0.9, "x"),
			),
			want: []struct{ id, query string }{
				{`search/attr/high`, `{ resource."a=b" = "x" }`},
				{`search/attr/low`, `{ span."foo bar\"" = "say \"hi\"\\" }`},
			},
		},
		{
			name: "a scope with no TraceQL reference is skipped",
			prof: ranked(
				attr("event", "foo", attributes.TypeString, 10, 0.9, "x"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phase2Cases(tt.prof)
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

// Generated cases shard the way the phase 1 search case does: one execution per
// shard, over the block's own window.
func TestPhase2CasesShard(t *testing.T) {
	prof := ranked(attr(attributes.ScopeSpan, "key", attributes.TypeString, 10, 0.9, "value"))
	prof.Block = &backend.BlockMeta{}

	cases := phase2Cases(prof)
	require.Len(t, cases, 1)

	shards := []Shard{{Index: 0, StartPage: 0, TotalPages: 2}, {Index: 1, StartPage: 2, TotalPages: 2}}
	executions, err := cases[0].executions(prof, shards, RunOptions{})
	require.NoError(t, err)
	require.Len(t, executions, len(shards))
}

func TestSelectivityBand(t *testing.T) {
	for _, tt := range []struct {
		sel  float64
		want string
	}{
		{1.0, "high"},
		{0.5, "high"},
		{0.499, "medium"},
		{0.1, "medium"},
		{0.099, "low"},
		{0, "low"},
	} {
		require.Equal(t, tt.want, selectivityBand(tt.sel), "selectivity %v", tt.sel)
	}
}
