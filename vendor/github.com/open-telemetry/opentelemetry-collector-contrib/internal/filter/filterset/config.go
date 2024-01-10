// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterset // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset/regexp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset/strict"
)

// MatchType describes the type of pattern matching a FilterSet uses to filter strings.
type MatchType string

const (
	// Regexp is the FilterType for filtering by regexp string matches.
	Regexp MatchType = "regexp"
	// Strict is the FilterType for filtering by exact string matches.
	Strict MatchType = "strict"
	// MatchTypeFieldName is the mapstructure field name for MatchType field.
	MatchTypeFieldName = "match_type"
)

var (
	validMatchTypes = []MatchType{Regexp, Strict}
)

// Config configures the matching behavior of a FilterSet.
type Config struct {
	MatchType    MatchType      `mapstructure:"match_type"`
	RegexpConfig *regexp.Config `mapstructure:"regexp"`
}

func NewUnrecognizedMatchTypeError(matchType MatchType) error {
	return fmt.Errorf("unrecognized %v: '%v', valid types are: %v", MatchTypeFieldName, matchType, validMatchTypes)
}

// CreateFilterSet creates a FilterSet from yaml config.
func CreateFilterSet(filters []string, cfg *Config) (FilterSet, error) {
	switch cfg.MatchType {
	case Regexp:
		return regexp.NewFilterSet(filters, cfg.RegexpConfig)
	case Strict:
		// Strict FilterSets do not have any extra configuration options, so call the constructor directly.
		return strict.NewFilterSet(filters), nil
	default:
		return nil, NewUnrecognizedMatchTypeError(cfg.MatchType)
	}
}
