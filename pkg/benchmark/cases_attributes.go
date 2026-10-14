package benchmark

import (
	"cmp"
	"slices"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
)

// attributeCases generates the cases a profile's attributes inform. For now
// those are searches: a search per ranked attribute per band of its own
// profiled values. Each band's value is one the profile proves the block
// holds, so the query matches; the case's ID names the band and attribute,
// and the query holds the value.
func attributeCases(prof *profile.BlockProfile) []benchCase {
	if prof.Attributes == nil {
		return nil
	}

	ranked := pickAttributes(prof.Attributes, rankedAttributesPerGroup, hasValues)
	ids := attributeIDs(ranked)
	var cases []benchCase
	for i, p := range ranked {
		for _, v := range bandValues(p) {
			query := valueFilter(p, v.value.Value)
			cases = append(cases, benchCase{
				id:    "search/ranked/" + v.band + "/" + ids[i],
				api:   apiSearch,
				query: query,
				executions: func(_ *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
					return searchExecutions(query, shards, prof.Block, opts.searchOptions()), nil
				},
			})
		}
	}
	return cases
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
