// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"fmt"
	"math"
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
	values := make([]indexedValue, n)
	for i, node := range nodes {
		values[i] = indexedValue{
			index: i,
			value: float64(getDuration(node)),
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].value < values[j].value
	})

	// The median and the minimum threshold are both reported and reasoned about
	// in real durations, so take them before any transform is applied.
	median := time.Duration(medianValue(values))
	minThreshold := float64(median) * (1 + cfg.MinOutlierThresholdPercent)

	// Transforms are monotonic, so they preserve both the sort above and the
	// classification below; only the spread the detectors measure changes.
	var zeroIndices []int
	if cfg.DurationTransform == DurationTransformLog {
		// Zero durations sort first and have no logarithm. They are classified as
		// normal and left out of the spread and the median the minimum threshold
		// is relative to, both of which the floor would otherwise set.
		for len(values) > 0 && values[0].value <= 0 {
			zeroIndices = append(zeroIndices, values[0].index)
			values = values[1:]
		}
		if len(zeroIndices) > 0 && len(values) > 0 {
			minThreshold = medianValue(values) * (1 + cfg.MinOutlierThresholdPercent)
		}
		for i := range values {
			values[i].value = logTransform(values[i].value)
		}
		minThreshold = logTransform(minThreshold)
	}

	// Determine method (default to IQR)
	method := cfg.Method
	if method == "" {
		method = OutlierMethodIQR
	}

	var outlierIndices, normalIndices []int

	// Zero durations left out above do not count toward the minimum group size.
	if len(values) >= cfg.MinGroupSize {
		switch method {
		case OutlierMethodMAD:
			outlierIndices, normalIndices = detectOutliersMAD(values, cfg.MADMultiplier, minThreshold)
		default: // IQR
			outlierIndices, normalIndices = detectOutliersIQR(values, cfg.IQRMultiplier, minThreshold)
		}
	} else {
		for _, v := range values {
			normalIndices = append(normalIndices, v.index)
		}
	}
	normalIndices = append(normalIndices, zeroIndices...)

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

// indexedValue pairs an index with the value the detectors compare, which is
// the span duration in nanoseconds, transformed if a transform is configured.
type indexedValue struct {
	index int
	value float64
}

// detectOutliersIQR identifies outliers using Interquartile Range method.
// Returns (outlierIndices, normalIndices).
func detectOutliersIQR(values []indexedValue, multiplier, minThreshold float64) ([]int, []int) {
	n := len(values)
	q1 := values[n/4].value
	q3 := values[3*n/4].value

	upperThreshold := max(q3+(q3-q1)*multiplier, minThreshold)
	return classifyByThreshold(values, upperThreshold)
}

// madScaleFactor converts MAD to a consistent scale with standard deviation.
// For normally distributed data, MAD ≈ 0.6745 * σ, so multiplying by 1.4826
// makes MAD comparable to standard deviation.
const madScaleFactor = 1.4826

// detectOutliersMAD identifies outliers using Median Absolute Deviation method.
// Returns (outlierIndices, normalIndices).
// MAD is more robust to extreme outliers than IQR.
func detectOutliersMAD(values []indexedValue, multiplier, minThreshold float64) ([]int, []int) {
	median := medianValue(values)

	// Absolute deviations from the median, then the median of those.
	deviations := make([]float64, len(values))
	for i, v := range values {
		deviations[i] = math.Abs(v.value - median)
	}
	slices.Sort(deviations)
	mad := medianSorted(deviations)

	upperThreshold := max(median+multiplier*madScaleFactor*mad, minThreshold)
	return classifyByThreshold(values, upperThreshold)
}

// logTransform maps a duration in nanoseconds onto the log scale, where the
// spread the detectors measure is the typical ratio between spans rather than
// the typical difference. The logarithm is undefined at or below zero, so
// values are floored at one nanosecond.
func logTransform(nanos float64) float64 {
	if nanos < 1 {
		nanos = 1
	}
	return math.Log(nanos)
}

// classifyByThreshold splits values into those above the threshold and those at
// or below it, returning the original span indices for each.
func classifyByThreshold(values []indexedValue, upperThreshold float64) ([]int, []int) {
	// Pre-allocate: outliers typically <20%.
	outlierIndices := make([]int, 0, len(values)/5+1)
	normalIndices := make([]int, 0, len(values))
	for _, v := range values {
		if v.value > upperThreshold {
			outlierIndices = append(outlierIndices, v.index)
		} else {
			normalIndices = append(normalIndices, v.index)
		}
	}
	return outlierIndices, normalIndices
}

// medianValue returns the median of a slice already sorted by value.
func medianValue(sorted []indexedValue) float64 {
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2].value
	}
	return (sorted[n/2-1].value + sorted[n/2].value) / 2
}

// medianSorted returns the median of an already sorted slice.
func medianSorted(sorted []float64) float64 {
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
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

// getDuration calculates span duration efficiently. Timestamps are unsigned, so
// a span that has no end timestamp yet underflows the subtraction and the signed
// conversion turns it back into a negative duration. Those are clamped to zero:
// a negative Q1 or median otherwise stretches the measured spread far enough to
// hide every real outlier in the group.
func getDuration(node *spanNode) time.Duration {
	d := time.Duration(node.span.EndTimestamp() - node.span.StartTimestamp())
	if d < 0 {
		return 0
	}
	return d
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
