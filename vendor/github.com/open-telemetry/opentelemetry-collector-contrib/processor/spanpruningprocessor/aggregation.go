// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"encoding/binary"
	"math/rand/v2"
	"sort"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// aggregationGroup captures the spans to aggregate along with execution
// metadata (tree depth, preassigned summary ID).
type aggregationGroup struct {
	nodes         []*spanNode    // nodes to aggregate (replaces []spanInfo for efficiency)
	depth         int            // tree depth (0 = leaf, 1 = parent of leaf, etc.)
	summarySpanID pcommon.SpanID // SpanID of the summary span (assigned before creation)
	templateNode  *spanNode      // node to use as summary template (longest duration)
}

// aggregationPlan orders aggregation groups for top-down execution and
// carries precomputed summary span IDs.
type aggregationPlan struct {
	groups []aggregationGroup
}

// findLongestDurationNode returns the node with the longest duration.
func findLongestDurationNode(nodes []*spanNode) *spanNode {
	if len(nodes) == 0 {
		return nil
	}
	longest := nodes[0]
	// pcommon.Timestamp is uint64 nanoseconds; direct subtraction avoids
	// creating intermediate time.Time objects (2 per span otherwise).
	longestDuration := int64(longest.span.EndTimestamp()) - int64(longest.span.StartTimestamp())
	for _, node := range nodes[1:] {
		duration := int64(node.span.EndTimestamp()) - int64(node.span.StartTimestamp())
		if duration > longestDuration {
			longest = node
			longestDuration = duration
		}
	}
	return longest
}

// generateSpanID produces a non-cryptographic span ID suitable for summary
// spans; uniqueness is sufficient, not randomness strength.
func generateSpanID() pcommon.SpanID {
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], rand.Uint64())
	return pcommon.SpanID(id)
}

// buildAggregationPlan sorts aggregation groups by depth (parents before
// children) and preassigns summary SpanIDs to avoid conflicts during writes.
func (*spanPruningProcessor) buildAggregationPlan(groups map[string]aggregationGroup) aggregationPlan {
	// Convert map to slice with pre-allocation
	groupSlice := make([]aggregationGroup, 0, len(groups))
	for key := range groups {
		groupSlice = append(groupSlice, groups[key])
	}

	// Sort by depth descending (highest depth first = top-down)
	sort.Slice(groupSlice, func(i, j int) bool {
		return groupSlice[i].depth > groupSlice[j].depth
	})

	// Pre-assign SpanIDs for all summary spans
	for i := range groupSlice {
		groupSlice[i].summarySpanID = generateSpanID()
	}

	return aggregationPlan{groups: groupSlice}
}

// executeAggregations performs the top-down creation of summary spans, removes
// originals using the tree's markedForRemoval flags, and returns the number of
// pruned spans.
func (p *spanPruningProcessor) executeAggregations(plan aggregationPlan, tree *traceTree) int {
	prunedCount := 0

	for i := range plan.groups {
		group := &plan.groups[i]
		// Calculate statistics and time range in single pass
		data := p.calculateAggregationData(group.nodes)

		// Determine the parent SpanID for the summary span.
		// Walk the tree: if the parent node was already replaced by a summary
		// span (from a higher-depth group), use that replacement ID.
		summaryParentID := group.nodes[0].span.ParentSpanID()
		if parentNode := group.nodes[0].parent; parentNode != nil && !parentNode.replacementSpanID.IsEmpty() {
			summaryParentID = parentNode.replacementSpanID
		}

		// Create summary span with correct parent
		p.createSummarySpanWithParent(*group, data, summaryParentID)

		// Record replacement span ID on each node so child groups can find it
		for _, node := range group.nodes {
			node.replacementSpanID = group.summarySpanID
		}
		prunedCount += len(group.nodes)
	}

	// Collect unique ScopeSpans that contain marked nodes, then remove in a
	// single pass per ScopeSpans using the tree's flags set during analysis.
	seen := make(map[ptrace.ScopeSpans]struct{})
	for _, node := range tree.nodeByID {
		if node.markedForRemoval {
			seen[node.scopeSpans] = struct{}{}
		}
	}
	for scopeSpans := range seen {
		scopeSpans.Spans().RemoveIf(func(span ptrace.Span) bool {
			n, ok := tree.nodeByID[span.SpanID()]
			return ok && n.markedForRemoval
		})
	}

	return prunedCount
}

// createSummarySpanWithParent builds the summary span for an aggregation
// group, wiring it under the provided parent SpanID and attaching stats.
func (p *spanPruningProcessor) createSummarySpanWithParent(group aggregationGroup, data aggregationData, parentSpanID pcommon.SpanID) ptrace.Span {
	// Use the template node (longest duration span) as a template
	templateNode := group.templateNode
	templateSpan := templateNode.span
	scopeSpans := templateNode.scopeSpans

	// Create new span in the same ScopeSpans as the first span
	newSpan := scopeSpans.Spans().AppendEmpty()

	// Copy basic properties from template
	newSpan.SetName(templateSpan.Name())
	newSpan.SetTraceID(templateSpan.TraceID())
	newSpan.SetSpanID(group.summarySpanID)
	newSpan.SetParentSpanID(parentSpanID)
	newSpan.SetKind(templateSpan.Kind())

	// Set timestamps from aggregation data
	newSpan.SetStartTimestamp(data.earliestStart)
	newSpan.SetEndTimestamp(data.latestEnd)

	// Copy attributes from template
	templateSpan.Attributes().CopyTo(newSpan.Attributes())

	// Copy status from template
	templateSpan.Status().CopyTo(newSpan.Status())

	// Copy TraceState from template for Consistent Probability Sampling compatibility
	newSpan.TraceState().FromRaw(templateSpan.TraceState().AsRaw())

	// Copy events and links from template
	templateSpan.Events().CopyTo(newSpan.Events())
	templateSpan.Links().CopyTo(newSpan.Links())

	// Add aggregation statistics as attributes
	prefix := p.config.AggregationAttributePrefix
	newSpan.Attributes().PutBool(prefix+"is_summary", true)
	newSpan.Attributes().PutInt(prefix+"span_count", data.count)
	newSpan.Attributes().PutInt(prefix+"duration_min_ns", int64(data.minDuration))
	newSpan.Attributes().PutInt(prefix+"duration_max_ns", int64(data.maxDuration))
	newSpan.Attributes().PutInt(prefix+"duration_total_ns", int64(data.sumDuration))
	if data.count > 0 {
		newSpan.Attributes().PutInt(prefix+"duration_avg_ns", int64(data.sumDuration)/data.count)
	}

	// Add histogram attributes if enabled.
	if len(p.config.AggregationHistogramBuckets) > 0 {
		// Add bucket bounds in seconds.
		bucketBoundsSlice := newSpan.Attributes().PutEmptySlice(prefix + "histogram_bucket_bounds_s")
		for _, bucket := range p.config.AggregationHistogramBuckets {
			bucketBoundsSlice.AppendEmpty().SetDouble(float64(bucket) / float64(time.Second))
		}

		// Add cumulative bucket counts.
		bucketCountsSlice := newSpan.Attributes().PutEmptySlice(prefix + "histogram_bucket_counts")
		for _, count := range data.bucketCounts {
			bucketCountsSlice.AppendEmpty().SetInt(count)
		}
	}

	return newSpan
}
