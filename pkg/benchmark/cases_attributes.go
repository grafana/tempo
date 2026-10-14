package benchmark

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

var rankedMetricsPipelines = []struct {
	id       string
	pipeline string
}{
	{rateCaseID, "rate()"},
	{"quantile", "quantile_over_time(duration, 0.99, 0.95)"},
}

// attributeCases generates the cases a profile's attributes inform: one
// search case per selectivity band. A band's case runs every picked
// attribute's equality for its band value — the attributes of a band are
// summarized as one case, the way traceid/present is one case over every
// profiled ID — since the band is what says something about a predicate's
// cost. Every value is one the profile proves the block holds, so each
// query matches. Metrics cases are generated per attribute and band, and tag
// values per string attribute, since there the attribute itself drives the cost.
func attributeCases(prof *profile.BlockProfile) []benchCase {
	if prof.Attributes == nil {
		return nil
	}

	ranked := pickAttributes(prof.Attributes, rankedAttributesPerGroup, hasValues)
	byBand := attributeQueries(ranked)

	var cases []benchCase
	for _, band := range []string{bandHigh, bandMedium, bandLow} {
		queries := byBand[band]
		if len(queries) == 0 {
			continue // none of the picked attributes has a value in this band
		}
		cases = append(cases, benchCase{
			id:  "search/ranked/" + band,
			api: apiSearch,
			executions: func(_ *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
				var executions []execution
				for _, q := range queries {
					executions = append(executions, searchExecutions(q, shards, prof.Block, opts.searchOptions())...)
				}
				return executions, nil
			},
		})
	}

	rankedIDs := attributeIDs(ranked)
	for _, m := range rankedMetricsPipelines {
		for i, p := range ranked {
			for _, v := range bandValues(p) {
				id := "metrics/ranked/" + m.id + "/" + v.band + "/" + rankedIDs[i]
				cases = append(cases, metricsCase(id, valueFilter(p, v.value.Value)+" | "+m.pipeline))
			}
		}
	}

	tagged := pickAttributes(prof.Attributes, tagValueAttributesPerGroup, isStringAttribute)
	for i, id := range attributeIDs(tagged) {
		cases = append(cases, tagValuesCase("metadata/tagvalues/"+id, tagged[i]))
	}
	return cases
}

// attributeQueries groups the picked attributes' equality filters by selectivity
// band, in pick order within a band.
func attributeQueries(ranked []attributes.Profile) map[string][]string {
	byBand := map[string][]string{}
	for _, p := range ranked {
		for _, v := range bandValues(p) {
			byBand[v.band] = append(byBand[v.band], valueFilter(p, v.value.Value))
		}
	}
	return byBand
}

// allCases is every case a run measures: the fixed set with the profile's
// generated cases, the cases of each API kept together, so a comparison
// lists one group per API rather than one per phase. The fixed set fixes
// the order the groups come in; generated cases of an API it lacks come
// after, as a group of their own.
func allCases(prof *profile.BlockProfile) []benchCase {
	all := slices.Concat(fixedCases(), attributeCases(prof))

	// The cases of an API stay together, in the order the APIs first appear
	// in the list above — the fixed set's order — and, within an API, as given.
	first := map[string]int{}
	for i, c := range all {
		if _, ok := first[c.api]; !ok {
			first[c.api] = i
		}
	}
	slices.SortStableFunc(all, func(a, b benchCase) int {
		return cmp.Compare(first[a.api], first[b.api])
	})
	return all
}

func metricsCase(id, query string) benchCase {
	return benchCase{
		id:    id,
		api:   apiMetrics,
		query: query,
		executions: func(prof *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
			if !prof.Block.EndTime.After(prof.Block.StartTime) {
				return nil, fmt.Errorf("a metrics query needs a non-empty time window, but the block's is %s to %s",
					prof.Block.StartTime, prof.Block.EndTime)
			}
			if _, err := traceql.Parse(query); err != nil {
				return nil, fmt.Errorf("parsing %q: %w", query, err)
			}
			return metricsExecutions(query, shards, prof.Block, opts.searchOptions()), nil
		},
	}
}

func tagValuesCase(id string, p attributes.Profile) benchCase {
	attr, _ := traceqlAttribute(p)
	return benchCase{
		id:  id,
		api: apiMetadata,
		executions: func(_ *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
			return tagValuesExecutions(attr, shards, opts.searchOptions()), nil
		},
	}
}

// selectCases takes patterns in benchmark compare's syntax. Matching nothing is
// an error, since a typo would otherwise read as a finished, empty run.
func selectCases(cases []benchCase, patterns []string) ([]benchCase, error) {
	if len(patterns) == 0 {
		return cases, nil
	}
	for _, p := range patterns {
		if _, err := path.Match(p, ""); err != nil {
			return nil, fmt.Errorf("invalid case pattern %q: %w", p, err)
		}
	}

	var kept []benchCase
	for _, c := range cases {
		if slices.ContainsFunc(patterns, func(p string) bool {
			ok, _ := path.Match(p, c.id)
			return ok
		}) {
			kept = append(kept, c)
		}
	}
	if len(kept) == 0 {
		return nil, fmt.Errorf("no case matches %s", strings.Join(patterns, ", "))
	}
	return kept, nil
}
