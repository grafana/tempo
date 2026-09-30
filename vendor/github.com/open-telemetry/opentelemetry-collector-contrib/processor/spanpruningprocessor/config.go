// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"go.opentelemetry.io/collector/component"
)

// Config defines the configuration options for the span pruning processor
// and the rules used to identify and aggregate similar spans.
type Config struct {
	// GroupByAttributes lists attribute patterns used to decide which leaf spans
	// belong in the same aggregation group. Spans must share the span name and
	// have identical values for every matched attribute to be grouped. Patterns
	// accept glob syntax, for example:
	//   - "db.*" matches db.operation, db.name, db.statement, etc.
	//   - "http.request.*" matches http.request.method, http.request.header, etc.
	//   - "service" matches only the exact key "service"
	// Examples: ["db.*", "http.method"], ["rpc.*"].
	GroupByAttributes []string `mapstructure:"group_by_attributes"`

	// MinSpansToAggregate is the minimum number of similar spans required before
	// aggregation occurs. Groups smaller than this threshold are preserved.
	// Default: 5
	MinSpansToAggregate int `mapstructure:"min_spans_to_aggregate"`

	// MaxParentDepth bounds how many ancestor levels above the aggregated leaves
	// can also be aggregated. Use 0 to aggregate only leaves, -1 for unlimited
	// depth, or a positive integer to cap traversal.
	// Default: 1
	MaxParentDepth int `mapstructure:"max_parent_depth"`

	// AggregationAttributePrefix prefixes all aggregation-related attributes that
	// are added to summary spans.
	// Default: "aggregation."
	AggregationAttributePrefix string `mapstructure:"aggregation_attribute_prefix"`

	// AggregationHistogramBuckets lists cumulative histogram bucket upper bounds
	// for latency tracking on aggregated spans. Empty slice disables histograms.
	AggregationHistogramBuckets []time.Duration `mapstructure:"aggregation_histogram_buckets"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.MinSpansToAggregate < 2 {
		return errors.New("min_spans_to_aggregate must be at least 2")
	}

	if cfg.MaxParentDepth < -1 {
		return errors.New("max_parent_depth must be -1 (unlimited) or >= 0")
	}

	// Validate AggregationAttributePrefix
	prefix := strings.TrimSpace(cfg.AggregationAttributePrefix)
	if prefix == "" {
		return errors.New("aggregation_attribute_prefix cannot be empty")
	}
	if strings.ContainsAny(prefix, " \t\n\r") {
		return errors.New("aggregation_attribute_prefix cannot contain whitespace")
	}

	// Validate GroupByAttributes glob patterns
	for i, pattern := range cfg.GroupByAttributes {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("group_by_attributes[%d] cannot be empty", i)
		}
		// Try to compile the same way processor.go does to catch invalid syntax early
		_, err := glob.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern at group_by_attributes[%d]: %q: %w", i, pattern, err)
		}
	}

	// Validate histogram buckets
	for i, bucket := range cfg.AggregationHistogramBuckets {
		if bucket <= 0 {
			return errors.New("histogram bucket values must be positive")
		}
		if i > 0 && bucket <= cfg.AggregationHistogramBuckets[i-1] {
			return errors.New("histogram buckets must be sorted in ascending order")
		}
	}

	return nil
}
