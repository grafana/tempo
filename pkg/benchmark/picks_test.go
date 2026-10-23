package benchmark

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

// profileAttr without values has only quantiles, like a high-cardinality numeric.
func profileAttr(scope, name, typ string, dedicated bool, values ...string) attributes.Profile {
	p := attributes.Profile{Scope: scope, Name: name, Type: typ, Dedicated: dedicated, Density: 1}
	for i, v := range values {
		p.Values = append(p.Values, attributes.Value{Value: v, Selectivity: 0.5 / float64(i+1)})
	}
	if len(values) == 0 {
		p.Quantiles = []attributes.Quantile{{Q: 0.5, Value: "1", Selectivity: 0.5}}
	}
	return p
}

func pickNames(ps []attributes.Profile) []string {
	names := make([]string, 0, len(ps))
	for _, p := range ps {
		names = append(names, p.Name)
	}
	return names
}

func TestPickAttributes(t *testing.T) {
	attrs := &attributes.Profiles{Spans: 10, Ranked: []attributes.Profile{
		profileAttr(attributes.ScopeResource, "a", attributes.TypeString, true, "x"),
		profileAttr(attributes.ScopeSpan, "b", attributes.TypeString, false, "x"),
		profileAttr(attributes.ScopeSpan, "q", attributes.TypeInt, true),
		profileAttr(attributes.ScopeIntrinsic, "status", attributes.TypeStatus, true, "unset"),
		profileAttr(attributes.ScopeSpan, "d", attributes.TypeInt, false, "200"),
	}}

	require.Equal(t, []string{"a", "status", "b", "d"}, pickNames(pickAttributes(attrs, 10, hasValues)))
	require.Equal(t, []string{"a", "b"}, pickNames(pickAttributes(attrs, 1, hasValues)))
	require.Equal(t, []string{"a", "b"}, pickNames(pickAttributes(attrs, 10, isStringAttribute)))
}

func TestValueFilter(t *testing.T) {
	for _, tc := range []struct {
		p    attributes.Profile
		want string
	}{
		{profileAttr(attributes.ScopeResource, "k8s.cluster.name", attributes.TypeString, true, "dev-us-central-0"), `{ resource.k8s.cluster.name = "dev-us-central-0" }`},
		{profileAttr(attributes.ScopeSpan, "http.status_code", attributes.TypeInt, false, "200"), `{ span.http.status_code = 200 }`},
		{profileAttr(attributes.ScopeSpan, "sampler.param", attributes.TypeFloat, false, "1.0"), `{ span.sampler.param = 1.0 }`},
		{profileAttr(attributes.ScopeSpan, "flag", attributes.TypeBool, false, "true"), `{ span.flag = true }`},
		{profileAttr(attributes.ScopeIntrinsic, "status", attributes.TypeStatus, true, "unset"), `{ status = unset }`},
		{profileAttr(attributes.ScopeIntrinsic, "kind", attributes.TypeKind, true, "client"), `{ kind = client }`},
		{profileAttr(attributes.ScopeIntrinsic, "duration", attributes.TypeDuration, true, "1.5s"), `{ duration = 1.5s }`},
		{profileAttr(attributes.ScopeIntrinsic, "name", attributes.TypeString, true, "ingest receive"), `{ name = "ingest receive" }`},
		{profileAttr(attributes.ScopeSpan, "my attr", attributes.TypeString, false, "x"), `{ span."my attr" = "x" }`},
		{profileAttr(attributes.ScopeSpan, "sql", attributes.TypeString, false, "SELECT \"a\" FROM `t`\n\\"), "{ span.sql = \"SELECT \\\"a\\\" FROM `t`\\n\\\\\" }"},
	} {
		got := valueFilter(tc.p, tc.p.Values[0].Value)
		require.Equal(t, tc.want, got)

		// The filter must parse back to the profiled attribute and value.
		_, _, _, req, err := traceql.Compile(got)
		require.NoError(t, err, got)
		require.Len(t, req.Conditions, 1, got)
		attr, ok := traceqlAttribute(tc.p)
		require.True(t, ok)
		require.Equal(t, attr, req.Conditions[0].Attribute, got)
		require.Equal(t, tc.p.Values[0].Value, req.Conditions[0].Operands[0].EncodeToString(false), got)
	}
}

// A newer profile may rank attributes this run cannot express in TraceQL.
func TestPickAttributesSkipsUnqueryable(t *testing.T) {
	attrs := &attributes.Profiles{Spans: 10, Ranked: []attributes.Profile{
		profileAttr(attributes.ScopeIntrinsic, "nestedSetParent", attributes.TypeInt, true, "-1"),
		profileAttr("event", "exception.type", attributes.TypeString, false, "x"),
		profileAttr(attributes.ScopeSpan, "b", attributes.TypeString, false, "x"),
	}}
	require.Equal(t, []string{"b"}, pickNames(pickAttributes(attrs, 10, hasValues)))
}

func TestBandValues(t *testing.T) {
	bands := func(values ...string) []string {
		var out []string
		for _, v := range bandValues(profileAttr(attributes.ScopeSpan, "x", attributes.TypeString, false, values...)) {
			out = append(out, v.band+"="+v.value.Value)
		}
		return out
	}

	require.Empty(t, bandValues(profileAttr(attributes.ScopeSpan, "n", attributes.TypeInt, false)), "quantiles only")
	require.Equal(t, []string{"high=a"}, bands("a"))
	require.Equal(t, []string{"high=a", "low=b"}, bands("a", "b"))
	require.Equal(t, []string{"high=a", "medium=b", "low=c"}, bands("a", "b", "c"))
	require.Equal(t, []string{"high=a", "medium=b", "low=d"}, bands("a", "b", "c", "d"))
	require.Equal(t, []string{"high=a", "medium=c", "low=e"}, bands("a", "b", "c", "d", "e"))
}

func TestAttributeIDs(t *testing.T) {
	ps := []attributes.Profile{
		profileAttr(attributes.ScopeSpan, "x", attributes.TypeString, false, "a"),
		profileAttr(attributes.ScopeSpan, "x", attributes.TypeInt, false, "1"),
		profileAttr(attributes.ScopeResource, "x", attributes.TypeString, true, "a"),
	}
	require.Equal(t, []string{"span.x.string", "span.x.int", "resource.x"}, attributeIDs(ps))
}

// benchmark compare rolls up only generated cases, so the fixed set must never
// look generated.
func TestFamily(t *testing.T) {
	for _, c := range fixedCases() {
		require.Empty(t, Family(c.id), c.id)
	}
	prof := ranked(
		profileAttr(attributes.ScopeResource, "service.name", attributes.TypeString, true, "a", "b", "c"),
		profileAttr(attributes.ScopeSpan, "key", attributes.TypeString, false, "x", "y"),
	)
	families := map[string]bool{}
	for _, c := range attributeCases(prof) {
		families[Family(c.id)] = true
	}
	require.Equal(t, map[string]bool{
		"search/ranked_dedicated":             true,
		"search/ranked_generic":               true,
		"metrics/ranked_dedicated/rate":       true,
		"metrics/ranked_generic/rate":         true,
		"metrics/ranked_dedicated/quantile":   true,
		"metrics/ranked_generic/quantile":     true,
		"metadata/ranked_dedicated/tagvalues": true,
		"metadata/ranked_generic/tagvalues":   true,
	}, families, "bands collapse into their query's family")
}
