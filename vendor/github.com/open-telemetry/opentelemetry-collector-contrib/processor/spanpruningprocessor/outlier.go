// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// outlierAnalysisResult contains outlier analysis and attribute correlations.
type outlierAnalysisResult struct {
	median         time.Duration
	correlations   []attributeCorrelation
	outlierIndices []int // indices of outlier spans (sorted by duration desc)
	normalIndices  []int // indices of normal spans
	hasOutliers    bool  // true if any outliers detected
}

// attributeCorrelation represents an attribute value that distinguishes outliers.
type attributeCorrelation struct {
	key               string
	value             string
	outlierOccurrence float64 // fraction of outliers with this value
	normalOccurrence  float64 // fraction of normal spans with this value
	score             float64 // outlierOccurrence - normalOccurrence
}

// analyzeOutliers performs outlier detection and attribute correlation.
// Returns nil if group is too small or no meaningful correlations found.
func analyzeOutliers(nodes []*spanNode, cfg OutlierAnalysisConfig) *outlierAnalysisResult {
	n := len(nodes)
	if n < cfg.MinGroupSize {
		return nil
	}

	// Collect and sort durations
	durations := make([]indexedDuration, n)
	for i, node := range nodes {
		// Use raw timestamps to avoid time.Time allocations
		durations[i] = indexedDuration{
			index:    i,
			duration: time.Duration(node.span.EndTimestamp() - node.span.StartTimestamp()),
		}
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i].duration < durations[j].duration
	})

	// Determine method (default to IQR)
	method := cfg.Method
	if method == "" {
		method = OutlierMethodIQR
	}

	var outlierIndices, normalIndices []int
	var median time.Duration

	switch method {
	case OutlierMethodMAD:
		outlierIndices, normalIndices, median = detectOutliersMAD(durations, cfg.MADMultiplier, cfg.MinOutlierThresholdPercent)
	default: // IQR
		outlierIndices, normalIndices, median = detectOutliersIQR(durations, cfg.IQRMultiplier, cfg.MinOutlierThresholdPercent)
	}

	hasOutliers := len(outlierIndices) > 0

	// Sort outlier indices by duration descending (most extreme first)
	if hasOutliers {
		sort.Slice(outlierIndices, func(i, j int) bool {
			iDur := getDuration(nodes[outlierIndices[i]])
			jDur := getDuration(nodes[outlierIndices[j]])
			return iDur > jDur
		})
	}

	// Need both outliers and normals for correlation
	if len(outlierIndices) == 0 || len(normalIndices) == 0 {
		return &outlierAnalysisResult{
			median:         median,
			outlierIndices: outlierIndices,
			normalIndices:  normalIndices,
			hasOutliers:    hasOutliers,
		}
	}

	// Analyze attribute correlations
	correlations := findCorrelations(
		nodes,
		outlierIndices,
		normalIndices,
		cfg.CorrelationMinOccurrence,
		cfg.CorrelationMaxNormalOccurrence,
		cfg.MaxCorrelatedAttributes,
	)

	return &outlierAnalysisResult{
		median:         median,
		correlations:   correlations,
		outlierIndices: outlierIndices,
		normalIndices:  normalIndices,
		hasOutliers:    true,
	}
}

// indexedDuration pairs an index with its duration for sorting.
type indexedDuration struct {
	index    int
	duration time.Duration
}

// detectOutliersIQR identifies outliers using Interquartile Range method.
// Returns (outlierIndices, normalIndices, median).
func detectOutliersIQR(durations []indexedDuration, multiplier, minThresholdPercent float64) ([]int, []int, time.Duration) {
	n := len(durations)

	// Calculate median
	var median time.Duration
	if n%2 == 1 {
		median = durations[n/2].duration
	} else {
		median = (durations[n/2-1].duration + durations[n/2].duration) / 2
	}

	// Calculate IQR
	q1 := durations[n/4].duration
	q3 := durations[3*n/4].duration
	iqr := q3 - q1

	// Calculate thresholds
	statisticalThreshold := q3 + time.Duration(float64(iqr)*multiplier)
	minimumThreshold := time.Duration(float64(median) * (1 + minThresholdPercent))
	upperThreshold := max(statisticalThreshold, minimumThreshold)

	// Classify spans (pre-allocate: outliers typically <20%)
	outlierIndices := make([]int, 0, n/5+1)
	normalIndices := make([]int, 0, n)

	for _, d := range durations {
		if d.duration > upperThreshold {
			outlierIndices = append(outlierIndices, d.index)
		} else {
			normalIndices = append(normalIndices, d.index)
		}
	}

	return outlierIndices, normalIndices, median
}

// madScaleFactor converts MAD to a consistent scale with standard deviation.
// For normally distributed data, MAD ≈ 0.6745 * σ, so multiplying by 1.4826
// makes MAD comparable to standard deviation.
const madScaleFactor = 1.4826

// detectOutliersMAD identifies outliers using Median Absolute Deviation method.
// Returns (outlierIndices, normalIndices, median).
// MAD is more robust to extreme outliers than IQR.
func detectOutliersMAD(durations []indexedDuration, multiplier, minThresholdPercent float64) ([]int, []int, time.Duration) {
	n := len(durations)

	// Calculate median (durations are already sorted)
	var median time.Duration
	if n%2 == 1 {
		median = durations[n/2].duration
	} else {
		median = (durations[n/2-1].duration + durations[n/2].duration) / 2
	}

	// Calculate absolute deviations from median
	deviations := make([]time.Duration, n)
	for i, d := range durations {
		dev := d.duration - median
		if dev < 0 {
			dev = -dev
		}
		deviations[i] = dev
	}

	// Sort deviations to find MAD (median of absolute deviations)
	slices.SortFunc(deviations, cmp.Compare)

	var mad time.Duration
	if n%2 == 1 {
		mad = deviations[n/2]
	} else {
		mad = (deviations[n/2-1] + deviations[n/2]) / 2
	}

	// Calculate thresholds
	statisticalThreshold := median + time.Duration(multiplier*madScaleFactor*float64(mad))
	minimumThreshold := time.Duration(float64(median) * (1 + minThresholdPercent))
	upperThreshold := max(statisticalThreshold, minimumThreshold)

	// Classify spans (pre-allocate: outliers typically <20%)
	outlierIndices := make([]int, 0, n/5+1)
	normalIndices := make([]int, 0, n)
	for _, d := range durations {
		if d.duration > upperThreshold {
			outlierIndices = append(outlierIndices, d.index)
		} else {
			normalIndices = append(normalIndices, d.index)
		}
	}

	return outlierIndices, normalIndices, median
}

// findCorrelations identifies attributes that distinguish outliers from normal spans.
func findCorrelations(
	nodes []*spanNode,
	outlierIndices []int,
	normalIndices []int,
	minOccurrence float64,
	maxNormalOccurrence float64,
	maxAttributes int,
) []attributeCorrelation {
	outlierCounts := countAttributeValues(nodes, outlierIndices)
	normalCounts := countAttributeValues(nodes, normalIndices)

	numOutliers := float64(len(outlierIndices))
	numNormals := float64(len(normalIndices))

	var correlations []attributeCorrelation

	for key, valueCounts := range outlierCounts {
		for value, outlierCount := range valueCounts {
			outlierOcc := float64(outlierCount) / numOutliers
			if outlierOcc < minOccurrence {
				continue
			}

			normalCount := 0
			if normalVals, exists := normalCounts[key]; exists {
				normalCount = normalVals[value]
			}
			normalOcc := float64(normalCount) / numNormals

			if normalOcc > maxNormalOccurrence {
				continue
			}

			correlations = append(correlations, attributeCorrelation{
				key:               key,
				value:             value,
				outlierOccurrence: outlierOcc,
				normalOccurrence:  normalOcc,
				score:             outlierOcc - normalOcc,
			})
		}
	}

	if len(correlations) == 0 {
		return nil
	}

	// Sort by score descending, then key ascending for stability
	sort.Slice(correlations, func(i, j int) bool {
		if correlations[i].score != correlations[j].score {
			return correlations[i].score > correlations[j].score
		}
		return correlations[i].key < correlations[j].key
	})

	if len(correlations) > maxAttributes {
		correlations = correlations[:maxAttributes]
	}

	return correlations
}

// countAttributeValues counts key-value occurrences for given node indices.
func countAttributeValues(nodes []*spanNode, indices []int) map[string]map[string]int {
	result := make(map[string]map[string]int)
	for _, idx := range indices {
		nodes[idx].span.Attributes().Range(func(k string, v pcommon.Value) bool {
			if result[k] == nil {
				result[k] = make(map[string]int)
			}
			result[k][v.AsString()]++
			return true
		})
	}
	return result
}

// formatCorrelations produces "key=value(outlier%/normal%), ..." string.
func formatCorrelations(correlations []attributeCorrelation) string {
	if len(correlations) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, c := range correlations {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%s=%s(%.0f%%/%.0f%%)",
			c.key, c.value,
			c.outlierOccurrence*100,
			c.normalOccurrence*100)
	}
	return sb.String()
}

// getDuration calculates span duration efficiently.
func getDuration(node *spanNode) time.Duration {
	return time.Duration(node.span.EndTimestamp() - node.span.StartTimestamp())
}

// filterOutlierNodes returns (normalNodes, outlierNodes) based on analysis.
// outlierNodes are sorted by duration descending (most extreme first).
func filterOutlierNodes(
	nodes []*spanNode,
	analysis *outlierAnalysisResult,
	cfg OutlierAnalysisConfig,
) ([]*spanNode, []*spanNode) {
	if analysis == nil || !cfg.PreserveOutliers || !analysis.hasOutliers {
		return nodes, nil // No filtering
	}

	// Skip preservation if no correlation found and that's required
	if cfg.PreserveOnlyWithCorrelation && len(analysis.correlations) == 0 {
		return nodes, nil
	}

	// Limit preserved outliers if configured
	preservedIndices := analysis.outlierIndices
	if cfg.MaxPreservedOutliers > 0 && len(analysis.outlierIndices) > cfg.MaxPreservedOutliers {
		preservedIndices = analysis.outlierIndices[:cfg.MaxPreservedOutliers]
	}

	// Build set for O(1) lookup
	outlierSet := make(map[int]struct{}, len(preservedIndices))
	for _, idx := range preservedIndices {
		outlierSet[idx] = struct{}{}
	}

	normalNodes := make([]*spanNode, 0, len(nodes)-len(preservedIndices))
	outlierNodes := make([]*spanNode, 0, len(preservedIndices))

	for i, node := range nodes {
		if _, isOutlier := outlierSet[i]; isOutlier {
			outlierNodes = append(outlierNodes, node)
		} else {
			normalNodes = append(normalNodes, node)
		}
	}

	// Sort outlierNodes by duration descending to match preservedIndices order
	sort.Slice(outlierNodes, func(i, j int) bool {
		return getDuration(outlierNodes[i]) > getDuration(outlierNodes[j])
	})

	return normalNodes, outlierNodes
}
