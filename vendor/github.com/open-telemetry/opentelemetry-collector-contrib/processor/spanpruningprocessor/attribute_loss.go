// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"sort"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// attributeCardinality records how many distinct values an attribute key has
// across spans being aggregated.
type attributeCardinality struct {
	key          string
	uniqueValues int
}

// attributeLossSummary separates two types of attribute loss during aggregation:
// - diverse: attributes present in ALL spans but with different values
// - missing: attributes absent from SOME spans (won't be in summary)
type attributeLossSummary struct {
	diverse []attributeCardinality // present in all spans, multiple values
	missing []attributeCardinality // absent from some spans
}

// isEmpty reports whether any attribute loss was detected.
func (s attributeLossSummary) isEmpty() bool {
	return len(s.diverse) == 0 && len(s.missing) == 0
}

// analyzeAttributeLoss examines spans being aggregated to determine which
// attributes will lose information in the summary span. The template node's
// attributes are preserved on the summary. Results include:
// - diverse: present in all spans but with multiple unique values (loss = unique - 1)
// - missing: absent from some spans (loss depends on template presence)
// Both slices are sorted by uniqueValues descending.
func analyzeAttributeLoss(nodes []*spanNode, template *spanNode) attributeLossSummary {
	if len(nodes) < 2 || template == nil {
		return attributeLossSummary{}
	}

	numSpans := len(nodes)
	templateSpan := template.span

	// Track unique values and presence count per attribute key
	// map[attributeKey]map[attributeValue]struct{}
	attributeValues := make(map[string]map[string]struct{})
	// map[attributeKey]count of spans that have this attribute
	attributePresence := make(map[string]int)

	for _, node := range nodes {
		node.span.Attributes().Range(func(k string, v pcommon.Value) bool {
			if attributeValues[k] == nil {
				attributeValues[k] = make(map[string]struct{})
			}
			attributeValues[k][v.AsString()] = struct{}{}
			attributePresence[k]++
			return true
		})
	}

	var result attributeLossSummary

	for key, values := range attributeValues {
		presence := attributePresence[key]
		uniqueCount := len(values)

		if presence < numSpans {
			// Attribute missing from some spans - presence loss
			// Loss depends on whether template has this attribute
			_, templateHasAttr := templateSpan.Attributes().Get(key)
			var lostCount int
			if templateHasAttr {
				// Template's value is preserved, lose the rest
				lostCount = uniqueCount - 1
			} else {
				// Template lacks it, summary lacks it, lose all values
				lostCount = uniqueCount
			}
			if lostCount > 0 {
				result.missing = append(result.missing, attributeCardinality{
					key:          key,
					uniqueValues: lostCount,
				})
			}
		} else if uniqueCount > 1 {
			// Present in all spans but with different values - diversity loss
			// Summary span keeps one value (from template), so loss = uniqueCount - 1
			result.diverse = append(result.diverse, attributeCardinality{
				key:          key,
				uniqueValues: uniqueCount - 1,
			})
		}
	}

	// Sort both slices by uniqueValues descending, then key ascending
	sortFunc := func(slice []attributeCardinality) {
		sort.Slice(slice, func(i, j int) bool {
			if slice[i].uniqueValues != slice[j].uniqueValues {
				return slice[i].uniqueValues > slice[j].uniqueValues
			}
			return slice[i].key < slice[j].key
		})
	}
	sortFunc(result.diverse)
	sortFunc(result.missing)

	return result
}

// maxLostAttributesEntries bounds how many attribute keys are serialized into
// the loss strings to prevent excessively long attribute values.
const maxLostAttributesEntries = 10

// formatAttributeCardinality formats attribute cardinality as "key(count),..."
// truncated to maxLostAttributesEntries with an ellipsis when needed.
func formatAttributeCardinality(attrs []attributeCardinality) string {
	if len(attrs) == 0 {
		return ""
	}

	truncated := len(attrs) > maxLostAttributesEntries
	count := len(attrs)
	if truncated {
		count = maxLostAttributesEntries
	}

	var sb strings.Builder
	for i, attr := range attrs[:count] {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(attr.key)
		sb.WriteByte('(')
		sb.WriteString(strconv.Itoa(attr.uniqueValues))
		sb.WriteByte(')')
	}

	if truncated {
		sb.WriteString(",...")
	}

	return sb.String()
}
