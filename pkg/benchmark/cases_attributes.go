package benchmark

import (
	"cmp"
	"slices"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
)

// attributeCases generates the cases a profile's attributes inform: one
// search case per selectivity band. A band's case runs every picked
// attribute's equality for its band value — the attributes of a band are
// summarized as one case, the way traceid/present is one case over every
// profiled ID — since the band is what says something about a predicate's
// cost. Every value is one the profile proves the block holds, so each
// query matches.
func attributeCases(prof *profile.BlockProfile) []benchCase {
	if prof.Attributes == nil {
		return nil
	}

	byBand := attributeQueries(pickAttributes(prof.Attributes, rankedAttributesPerGroup, hasValues))

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
