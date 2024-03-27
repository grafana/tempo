// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package regexp // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset/regexp"

import (
	"math"
	"regexp"

	lru "github.com/hashicorp/golang-lru/v2"
)

// FilterSet encapsulates a set of filters and caches match results.
// Filters are re2 regex strings.
// FilterSet is exported for convenience, but has unexported fields and should be constructed through NewFilterSet.
//
// FilterSet satisfies the FilterSet interface from
// "go.opentelemetry.io/collector/internal/processor/filterset"
type FilterSet struct {
	regexes []*regexp.Regexp
	cache   *lru.Cache[string, bool]
}

// NewFilterSet constructs a FilterSet of re2 regex strings.
// If any of the given filters fail to compile into re2, an error is returned.
func NewFilterSet(filters []string, cfg *Config) (*FilterSet, error) {
	fs := &FilterSet{
		regexes: make([]*regexp.Regexp, 0, len(filters)),
	}

	if err := fs.addFilters(filters); err != nil {
		return nil, err
	}

	if cfg != nil && cfg.CacheEnabled {
		// Because of legacy behavior, CacheMaxNumEntries == 0 means unbounded cache.
		numEntries := cfg.CacheMaxNumEntries
		if numEntries == 0 {
			numEntries = math.MaxInt
		}
		var err error
		fs.cache, err = lru.New[string, bool](numEntries)
		if err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// Matches returns true if the given string matches any of the FilterSet's filters.
// The given string must be fully matched by at least one filter's re2 regex.
func (rfs *FilterSet) Matches(toMatch string) bool {
	if rfs.cache != nil {
		if v, ok := rfs.cache.Get(toMatch); ok {
			return v
		}
	}

	for _, r := range rfs.regexes {
		if r.MatchString(toMatch) {
			if rfs.cache != nil {
				rfs.cache.Add(toMatch, true)
			}
			return true
		}
	}

	if rfs.cache != nil {
		rfs.cache.Add(toMatch, false)
	}
	return false
}

// addFilters compiles all the given filters and stores them as regexes.
func (rfs *FilterSet) addFilters(filters []string) error {
	dedup := make(map[string]struct{}, len(filters))
	for _, f := range filters {
		if _, ok := dedup[f]; ok {
			continue
		}

		re, err := regexp.Compile(f)
		if err != nil {
			return err
		}
		rfs.regexes = append(rfs.regexes, re)
		dedup[f] = struct{}{}
	}

	return nil
}
