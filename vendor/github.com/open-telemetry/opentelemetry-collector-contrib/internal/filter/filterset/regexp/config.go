// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package regexp // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset/regexp"

// Config represents the options for a NewFilterSet.
type Config struct {
	// CacheEnabled determines whether match results are LRU cached to make subsequent matches faster.
	// Cache size is unlimited unless CacheMaxNumEntries is also specified.
	CacheEnabled bool `mapstructure:"cacheenabled"`
	// CacheMaxNumEntries is the max number of entries of the LRU cache that stores match results.
	// CacheMaxNumEntries is ignored if CacheEnabled is false.
	CacheMaxNumEntries int `mapstructure:"cachemaxnumentries"`
}
