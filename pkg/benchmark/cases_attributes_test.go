package benchmark

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
)

func caseIDs(cases []benchCase) []string {
	ids := make([]string, 0, len(cases))
	for _, c := range cases {
		ids = append(ids, c.id)
	}
	return ids
}

func TestAttributeCasesMetricsAndTagValues(t *testing.T) {
	prof := &profile.BlockProfile{Attributes: &attributes.Profiles{Spans: 10, Ranked: []attributes.Profile{
		profileAttr(attributes.ScopeResource, "service.name", attributes.TypeString, true, "svc"),
		profileAttr(attributes.ScopeSpan, "http.method", attributes.TypeString, false, "GET", "POST", "PUT", "DELETE"),
		profileAttr(attributes.ScopeSpan, "http.status_code", attributes.TypeInt, false, "200"),
		profileAttr(attributes.ScopeIntrinsic, "duration", attributes.TypeDuration, true),
		profileAttr(attributes.ScopeIntrinsic, "status", attributes.TypeStatus, true, "unset"),
	}}}

	cases := slices.Concat(casesOf(attributeCases(prof), apiMetrics), casesOf(attributeCases(prof), apiMetadata))
	require.Equal(t, []string{
		"metrics/ranked_dedicated/rate/high/resource.service.name",
		"metrics/ranked_dedicated/rate/high/intrinsic.status",
		"metrics/ranked_generic/rate/high/span.http.method",
		"metrics/ranked_generic/rate/medium/span.http.method",
		"metrics/ranked_generic/rate/low/span.http.method",
		"metrics/ranked_generic/rate/high/span.http.status_code",
		"metrics/ranked_dedicated/quantile/high/resource.service.name",
		"metrics/ranked_dedicated/quantile/high/intrinsic.status",
		"metrics/ranked_generic/quantile/high/span.http.method",
		"metrics/ranked_generic/quantile/medium/span.http.method",
		"metrics/ranked_generic/quantile/low/span.http.method",
		"metrics/ranked_generic/quantile/high/span.http.status_code",
		"metadata/ranked_dedicated/tagvalues/resource.service.name",
		"metadata/ranked_generic/tagvalues/span.http.method",
	}, caseIDs(cases))

	byID := map[string]benchCase{}
	for _, c := range cases {
		byID[c.id] = c
	}

	rate := byID["metrics/ranked_dedicated/rate/high/resource.service.name"]
	require.Equal(t, apiMetrics, rate.api)
	require.Equal(t, `{ resource.service.name = "svc" } | rate()`, rate.query)
	require.Equal(t, `{ status = unset } | quantile_over_time(duration, 0.99, 0.95)`, byID["metrics/ranked_dedicated/quantile/high/intrinsic.status"].query)

	require.Equal(t, `{ span.http.method = "DELETE" } | rate()`, byID["metrics/ranked_generic/rate/low/span.http.method"].query)
	require.Equal(t, `{ span.http.method = "POST" } | rate()`, byID["metrics/ranked_generic/rate/medium/span.http.method"].query)

	tags := byID["metadata/ranked_generic/tagvalues/span.http.method"]
	require.Equal(t, apiMetadata, tags.api)
	require.Empty(t, tags.query)
}

func TestExistenceCases(t *testing.T) {
	prof := &profile.BlockProfile{Attributes: &attributes.Profiles{Spans: 10, Ranked: []attributes.Profile{
		profileAttr(attributes.ScopeSpan, "http.status_code", attributes.TypeInt, true, "200", "500"),
		// " = " in the name must survive the filtered case's != operator.
		profileAttr(attributes.ScopeResource, "a = b", attributes.TypeString, false, "x", "y"),
		profileAttr(attributes.ScopeIntrinsic, "status", attributes.TypeStatus, true, "unset"),
	}}}

	queries := map[string]string{}
	for _, c := range existenceCases(prof) {
		queries[c.id] = c.query
	}
	require.Equal(t, map[string]string{
		"search/ranked_dedicated/exists/span.http.status_code":            "{ span.http.status_code != nil }",
		"search/ranked_dedicated/select/span.http.status_code":            "{ } | select(span.http.status_code)",
		"metrics/ranked_dedicated/rate-by/span.http.status_code":          "{ } | rate() by (span.http.status_code)",
		"metrics/ranked_dedicated/filtered-rate-by/span.http.status_code": "{ span.http.status_code != 200 } | rate() by (span.http.status_code)",
		"search/ranked_generic/exists/resource.a = b":                     `{ resource."a = b" != nil }`,
		"search/ranked_generic/select/resource.a = b":                     `{ } | select(resource."a = b")`,
		"metrics/ranked_generic/rate-by/resource.a = b":                   `{ } | rate() by (resource."a = b")`,
		"metrics/ranked_generic/filtered-rate-by/resource.a = b":          `{ resource."a = b" != "x" } | rate() by (resource."a = b")`,
	}, queries)

	require.Empty(t, existenceCases(&profile.BlockProfile{}))
}

func TestAttributeCasesWithoutAttributes(t *testing.T) {
	require.Empty(t, attributeCases(&profile.BlockProfile{}))
}

func TestSelectCases(t *testing.T) {
	cases := []benchCase{{id: "search/nopredicate"}, {id: "metrics/ranked_generic/rate/span.a"}, {id: "metrics/rate"}}

	all, err := selectCases(cases, nil)
	require.NoError(t, err)
	require.Equal(t, caseIDs(cases), caseIDs(all))

	ranked, err := selectCases(cases, []string{"metrics/ranked_*/*/*", "search/*"})
	require.NoError(t, err)
	require.Equal(t, []string{"search/nopredicate", "metrics/ranked_generic/rate/span.a"}, caseIDs(ranked))

	_, err = selectCases(cases, []string{"["})
	require.ErrorContains(t, err, "invalid case pattern")

	_, err = selectCases(cases, []string{"traceid/*"})
	require.ErrorContains(t, err, "no case matches traceid/*")
}
