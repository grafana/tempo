// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"context"
	"fmt"
	"time"

	"github.com/gobwas/glob"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor/internal/metadata"
)

// spanInfo pairs a span with its ScopeSpans container for in-place edits.
type spanInfo struct {
	span       ptrace.Span
	scopeSpans ptrace.ScopeSpans
}

// attributePattern caches a compiled glob used for attribute key matching.
type attributePattern struct {
	glob glob.Glob
}

// spanPruningProcessor aggregates similar leaf spans (and eligible parents)
// according to configuration while emitting telemetry about pruning actions.
type spanPruningProcessor struct {
	config            *Config
	logger            *zap.Logger
	attributePatterns []attributePattern
	telemetryBuilder  *metadata.TelemetryBuilder
}

func newSpanPruningProcessor(set processor.Settings, cfg *Config, telemetryBuilder *metadata.TelemetryBuilder) (*spanPruningProcessor, error) {
	// Compile glob patterns for group_by_attributes
	patterns := make([]attributePattern, 0, len(cfg.GroupByAttributes))
	for _, pattern := range cfg.GroupByAttributes {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		patterns = append(patterns, attributePattern{
			glob: g,
		})
	}

	return &spanPruningProcessor{
		config:            cfg,
		logger:            set.Logger,
		attributePatterns: patterns,
		telemetryBuilder:  telemetryBuilder,
	}, nil
}

// shutdown releases processor resources, including telemetry providers.
func (p *spanPruningProcessor) shutdown(_ context.Context) error {
	p.telemetryBuilder.Shutdown()
	return nil
}

// processTraces runs aggregation for each trace batch and records processor
// telemetry about received, pruned, and aggregated spans.
func (p *spanPruningProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	start := time.Now()

	// Count incoming spans
	totalSpans := int64(0)
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		for j := 0; j < td.ResourceSpans().At(i).ScopeSpans().Len(); j++ {
			totalSpans += int64(td.ResourceSpans().At(i).ScopeSpans().At(j).Spans().Len())
		}
	}
	p.telemetryBuilder.ProcessorSpanpruningSpansReceived.Add(ctx, totalSpans)

	// Group spans by TraceID
	traceSpans := p.groupSpansByTraceID(td)

	// Process each trace independently
	tracesProcessed := int64(0)
	for _, spans := range traceSpans {
		p.processTrace(ctx, spans)
		tracesProcessed++
	}

	// Record telemetry only when actual work was done
	if tracesProcessed > 0 {
		p.telemetryBuilder.ProcessorSpanpruningTracesProcessed.Add(ctx, tracesProcessed)
		p.telemetryBuilder.ProcessorSpanpruningProcessingDuration.Record(ctx,
			time.Since(start).Seconds())
	}

	return td, nil
}

// groupSpansByTraceID flattens incoming data into a TraceID-indexed map so
// each trace can be analyzed independently.
func (*spanPruningProcessor) groupSpansByTraceID(td ptrace.Traces) map[pcommon.TraceID][]spanInfo {
	traceSpans := make(map[pcommon.TraceID][]spanInfo)

	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				traceID := span.TraceID()
				traceSpans[traceID] = append(traceSpans[traceID], spanInfo{
					span:       span,
					scopeSpans: ils,
				})
			}
		}
	}

	return traceSpans
}

// processTrace applies the pruning algorithm to a single trace:
// 1) analyze aggregation candidates bottom-up, 2) build a top-down execution
// plan, and 3) create summary spans while removing originals.
func (p *spanPruningProcessor) processTrace(ctx context.Context, spans []spanInfo) {
	// Build trace tree
	tree := p.buildTraceTree(spans)
	if len(tree.nodeByID) == 0 {
		return
	}

	// Phase 1: Analyze aggregations (bottom-up)
	aggregationGroups := p.analyzeAggregationsWithTree(tree)
	if len(aggregationGroups) == 0 {
		return
	}

	// Phase 2: Build aggregation plan (order top-down)
	plan := p.buildAggregationPlan(aggregationGroups)

	// Phase 3: Execute aggregations (top-down) and record pruned spans
	prunedCount := p.executeAggregations(plan, tree)

	// Record telemetry after aggregation is complete
	p.telemetryBuilder.ProcessorSpanpruningSpansPruned.Add(ctx, int64(prunedCount))
	p.telemetryBuilder.ProcessorSpanpruningAggregationsCreated.Add(ctx, int64(len(plan.groups)))
	for i := range plan.groups {
		p.telemetryBuilder.ProcessorSpanpruningAggregationGroupSize.Record(ctx, int64(len(plan.groups[i].nodes)))
	}
}

// analyzeAggregationsWithTree performs Phase 1 using tree structure
// Uses markedForRemoval field on nodes instead of separate map for better performance
// Optimized to walk up from marked nodes instead of scanning all nodes
func (p *spanPruningProcessor) analyzeAggregationsWithTree(tree *traceTree) map[string]aggregationGroup {
	// Step 1: Get pre-computed leaf nodes
	leafNodes := tree.getLeaves()
	if len(leafNodes) == 0 {
		return nil
	}

	// Step 2: Group similar leaf nodes
	leafGroups := p.groupLeafNodesByKey(leafNodes)

	// Step 3: Filter groups meeting minimum threshold and mark nodes
	// Pre-size based on expected number of groups
	aggregationGroups := make(map[string]aggregationGroup, len(leafGroups)/2)

	// Track nodes marked in this round for candidate collection
	var markedNodes []*spanNode

	for groupKey, nodes := range leafGroups {
		if len(nodes) < p.config.MinSpansToAggregate {
			continue
		}

		// Find template from nodes
		templateNode := findLongestDurationNode(nodes)

		aggregationGroups[groupKey] = aggregationGroup{
			nodes:        nodes,
			depth:        0,
			templateNode: templateNode,
		}

		// Mark spans for removal
		for _, node := range nodes {
			node.markedForRemoval = true
		}
		markedNodes = append(markedNodes, nodes...)
	}

	if len(aggregationGroups) == 0 {
		return nil
	}

	// Step 4: Walk up the tree to find eligible parent spans recursively
	// Respect MaxParentDepth: 0 = no parent aggregation, -1 = unlimited, >0 = limit
	if p.config.MaxParentDepth == 0 {
		return aggregationGroups
	}

	// Collect initial parent candidates from marked leaf nodes
	candidates := collectParentCandidates(markedNodes)

	depth := 1
	for len(candidates) > 0 {
		// Check if we've reached the maximum parent depth limit
		if p.config.MaxParentDepth > 0 && depth > p.config.MaxParentDepth {
			break
		}

		// Find eligible parents from candidates (walks up from marked nodes)
		eligibleParents := p.findEligibleParentNodesFromCandidates(candidates)
		if len(eligibleParents) == 0 {
			break
		}

		// Group parent candidates by name + status
		parentGroups := make(map[string][]*spanNode)
		for _, node := range eligibleParents {
			parentKey := p.buildParentGroupKey(node.span, depth)
			parentGroups[parentKey] = append(parentGroups[parentKey], node)
		}

		// Add parent groups (at least 2 parents to aggregate)
		markedNodes = markedNodes[:0] // reset for this round
		for parentKey, nodes := range parentGroups {
			if len(nodes) < 2 {
				continue
			}

			// Find the template node (longest duration) for this group
			templateNode := findLongestDurationNode(nodes)

			aggregationGroups[parentKey] = aggregationGroup{
				nodes:        nodes,
				depth:        depth,
				templateNode: templateNode,
			}
			// Mark parent nodes for removal
			for _, node := range nodes {
				node.markedForRemoval = true
			}
			markedNodes = append(markedNodes, nodes...)
		}

		if len(markedNodes) == 0 {
			break
		}

		// Collect next round of candidates from newly marked nodes
		candidates = collectParentCandidates(markedNodes)
		depth++
	}

	return aggregationGroups
}
