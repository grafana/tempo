// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/gobwas/glob"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor/internal/metadata"
)

// spanInfo pairs a span with its owning ResourceSpans and ScopeSpans
// containers for in-place edits, hierarchy reconstruction, and context-aware
// evaluation.
type spanInfo struct {
	span          ptrace.Span
	resourceSpans ptrace.ResourceSpans
	scopeSpans    ptrace.ScopeSpans
}

// attributePattern caches a compiled glob used for attribute key matching.
type attributePattern struct {
	glob glob.Glob
}

// spanPruningProcessor aggregates similar leaf spans (and eligible parents)
// according to configuration while emitting telemetry about pruning actions.
type spanPruningProcessor struct {
	config                      *Config
	logger                      *zap.Logger
	attributePatterns           []attributePattern
	telemetryBuilder            *metadata.TelemetryBuilder
	enableAttributeLossAnalysis bool
	conditions                  *ottl.ConditionSequence[*ottlspan.TransformContext]
	enableBytesMetrics          bool
}

func newSpanPruningProcessor(set processor.Settings, cfg *Config, telemetryBuilder *metadata.TelemetryBuilder, conditions *ottl.ConditionSequence[*ottlspan.TransformContext]) (*spanPruningProcessor, error) {
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
		config:                      cfg,
		logger:                      set.Logger,
		attributePatterns:           patterns,
		telemetryBuilder:            telemetryBuilder,
		enableAttributeLossAnalysis: cfg.EnableAttributeLossAnalysis,
		conditions:                  conditions,
		enableBytesMetrics:          cfg.EnableBytesMetrics,
	}, nil
}

// shutdown releases processor resources, including telemetry providers.
func (p *spanPruningProcessor) shutdown(_ context.Context) error {
	p.telemetryBuilder.Shutdown()
	return nil
}

// shouldSampleAttributeLossExemplar decides whether to attach exemplars to
// attribute-loss metrics based on the configured sampling rate.
func (p *spanPruningProcessor) shouldSampleAttributeLossExemplar() bool {
	rate := p.config.AttributeLossExemplarSampleRate
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	return rand.Float64() < rate
}

// createExemplarContext creates a context with span context for exemplar attachment.
// Uses direct type casting since pcommon and trace ID types are identical byte arrays.
func createExemplarContext(ctx context.Context, traceID pcommon.TraceID, spanID pcommon.SpanID) context.Context {
	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID(traceID),
		SpanID:     trace.SpanID(spanID),
		TraceFlags: trace.FlagsSampled,
	}))
}

// processTraces runs aggregation for each trace batch and records processor
// telemetry about received, pruned, and aggregated spans.
func (p *spanPruningProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	start := time.Now()

	// Measure bytes received before processing
	if p.enableBytesMetrics {
		var m ptrace.ProtoMarshaler
		p.telemetryBuilder.ProcessorSpanpruningBytesReceived.Add(ctx, int64(m.TracesSize(td)))
	}

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
	matchedTraces, tracesSkipped := p.filterTracesByConditions(ctx, traceSpans)

	var bytesProcessedInput int64
	if p.enableBytesMetrics {
		// Measure matched traces before pruning so bytes_processed_input reflects
		// pre-pruning size.
		bytesProcessedInput = p.getBytes(matchedTraces, traceSpans)
	}

	// Process each trace independently
	tracesProcessed := int64(0)
	for traceID := range matchedTraces {
		p.processTrace(ctx, traceSpans[traceID])
		tracesProcessed++
	}

	// Record telemetry only when actual work was done
	if tracesProcessed > 0 {
		p.telemetryBuilder.ProcessorSpanpruningTracesProcessed.Add(ctx, tracesProcessed)
		p.telemetryBuilder.ProcessorSpanpruningProcessingDuration.Record(ctx,
			time.Since(start).Seconds())
	}
	if tracesSkipped > 0 {
		p.telemetryBuilder.ProcessorSpanpruningTracesSkipped.Add(ctx, tracesSkipped)
	}

	// Measure bytes emitted after pruning to capture the reduction in trace size.
	if p.enableBytesMetrics {
		var m ptrace.ProtoMarshaler
		if bytesProcessedInput > 0 {
			p.telemetryBuilder.ProcessorSpanpruningBytesProcessedInput.Add(ctx, bytesProcessedInput)
			// Re-group from td so getBytes sees post-prune spans (aggregated summaries, removals).
			// We cannot use m.TracesSize(td) here: that measures the entire batch (matched and
			// unmatched traces), which is what bytes_emitted already captures. bytes_processed_output
			// must reflect only the matched subset after pruning — e.g. if 10 of 100 traces matched,
			// bytes_processed_output covers those 10 post-prune, while bytes_emitted covers all 100.
			// The two are equal only when all traces in the batch match the OTTL conditions.
			postPruneTraceSpans := p.groupSpansByTraceID(td)
			bytesProcessedOutput := p.getBytes(matchedTraces, postPruneTraceSpans)
			p.telemetryBuilder.ProcessorSpanpruningBytesProcessedOutput.Add(ctx, bytesProcessedOutput)
		}
		p.telemetryBuilder.ProcessorSpanpruningBytesEmitted.Add(ctx, int64(m.TracesSize(td)))
	}

	return td, nil
}

// filterTracesByConditions evaluates conditions against every trace in traceSpans and
// returns the set of matching TraceIDs alongside a count of skipped traces.
// When no conditions are configured, every trace matches.
func (p *spanPruningProcessor) filterTracesByConditions(ctx context.Context, traceSpans map[pcommon.TraceID][]spanInfo) (map[pcommon.TraceID]struct{}, int64) {
	matched := make(map[pcommon.TraceID]struct{})
	var skipped int64
	for traceID, spans := range traceSpans {
		if !p.traceMatchesConditions(ctx, spans) {
			skipped++
			continue
		}
		matched[traceID] = struct{}{}
	}
	return matched, skipped
}

// getBytes returns the serialized size of the subset of traces identified
// by matchedTraces, preserving the original ResourceSpans/ScopeSpans hierarchy.
func (*spanPruningProcessor) getBytes(matchedTraces map[pcommon.TraceID]struct{}, traceSpans map[pcommon.TraceID][]spanInfo) int64 {
	filtered := ptrace.NewTraces()
	// Track already-added ResourceSpans and ScopeSpans by their original object
	// identity to preserve the original hierarchy (same RS/SS grouping).
	// pdata structs hold a pointer to the underlying proto, so struct equality
	// gives pointer identity for free.
	rsMap := make(map[ptrace.ResourceSpans]ptrace.ResourceSpans)
	ssMap := make(map[ptrace.ScopeSpans]ptrace.ScopeSpans)
	for traceID := range matchedTraces {
		for _, si := range traceSpans[traceID] {
			filtRS, ok := rsMap[si.resourceSpans]
			if !ok {
				filtRS = filtered.ResourceSpans().AppendEmpty()
				si.resourceSpans.Resource().CopyTo(filtRS.Resource())
				filtRS.SetSchemaUrl(si.resourceSpans.SchemaUrl())
				rsMap[si.resourceSpans] = filtRS
			}
			filtSS, ok := ssMap[si.scopeSpans]
			if !ok {
				filtSS = filtRS.ScopeSpans().AppendEmpty()
				si.scopeSpans.Scope().CopyTo(filtSS.Scope())
				filtSS.SetSchemaUrl(si.scopeSpans.SchemaUrl())
				ssMap[si.scopeSpans] = filtSS
			}
			si.span.CopyTo(filtSS.Spans().AppendEmpty())
		}
	}
	var m ptrace.ProtoMarshaler
	return int64(m.TracesSize(filtered))
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
					span:          span,
					resourceSpans: rs,
					scopeSpans:    ils,
				})
			}
		}
	}

	return traceSpans
}

// traceMatchesConditions evaluates whether any span in the trace matches the configured
// OTTL conditions. Returns true when no conditions are configured (prune all traces).
// When conditions are set, returns true if at least one span matches any condition.
func (p *spanPruningProcessor) traceMatchesConditions(ctx context.Context, spans []spanInfo) bool {
	if p.conditions == nil {
		return true
	}

	for _, si := range spans {
		tCtx := ottlspan.NewTransformContext(si.resourceSpans, si.scopeSpans, si.span)
		matches, err := p.conditions.Eval(ctx, tCtx)
		tCtx.Close()
		if err != nil {
			p.logger.Error("OTTL condition evaluation error",
				zap.Error(err),
				zap.String("span.name", si.span.Name()),
				zap.String("trace.id", si.span.TraceID().String()))
			continue
		}
		if matches {
			return true
		}
	}

	return false
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
	aggregationGroups := p.analyzeAggregationsWithTree(ctx, tree)
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

// analyzeAggregationsWithTree plans the candidate aggregation groups, then
// detects and protects outliers within them, then aggregates the non-protected
// spans. Planning runs once and feeds both detection and execution.
func (p *spanPruningProcessor) analyzeAggregationsWithTree(ctx context.Context, tree *traceTree) map[string]aggregationGroup {
	// Step 1: plan the groups that would aggregate.
	groups := p.planCandidateGroups(tree)
	if len(groups) == 0 {
		return nil
	}

	// Step 2: detect outliers at every level and sample exemplars at the top of
	// each aggregation tree, protecting both as whole subtrees before any
	// aggregation runs.
	outlierResultByKey, protectedRootsByKey, exemplarByKey := p.detectAndProtect(ctx, groups)

	// Step 3: aggregate. planCandidateGroups returns groups bottom-up (leaves
	// first, then parents by increasing depth), so processing them in order
	// marks a group's children before the parent group is evaluated.
	aggregationGroups := make(map[string]aggregationGroup, len(groups))

	for _, g := range groups {
		nodes := g.nodes
		if g.depth == 0 {
			// Leaf group: drop protected (preserved outlier or exemplar) subtrees,
			// then re-check the floor.
			nodes = excludeProtectedNodes(nodes)
			if len(nodes) < p.config.MinSpansToAggregate {
				continue
			}
		} else {
			// Parent group: protection, or a child group that fell below the
			// floor, can leave a planned parent with an un-aggregated child, so
			// re-check eligibility against the marks set by the deeper groups.
			nodes = p.eligibleParents(nodes)
			if len(nodes) < 2 {
				continue
			}
		}

		templateNode := findLongestDurationNode(nodes)
		lossInfo := p.recordAttributeLoss(ctx, g.depth == 0, nodes, templateNode)

		preserved := protectedRootsByKey[g.key]
		sel := exemplarByKey[g.key]
		aggregationGroups[g.key] = aggregationGroup{
			nodes:                  nodes,
			depth:                  g.depth,
			lossInfo:               lossInfo,
			templateNode:           templateNode,
			outlierAnalysis:        outlierResultByKey[g.key],
			preservedOutliers:      preserved,
			exemplars:              sel.roots,
			exemplarPopulationSize: sel.populationSize,
			exemplarSampleSize:     sel.sampleSize,
		}
		for _, n := range nodes {
			n.markedForRemoval = true
		}
		p.recordPreserved(ctx, preserved)
		p.recordExemplars(ctx, sel.roots)
	}

	return aggregationGroups
}

// eligibleParents returns the parents of a planned parent group that are still
// eligible to aggregate, in a stable order so the summary's parent is chosen
// deterministically. Protection, or a child group that fell below
// MinSpansToAggregate, can make a planned parent ineligible by the time it is
// executed.
func (p *spanPruningProcessor) eligibleParents(nodes []*spanNode) []*spanNode {
	out := make([]*spanNode, 0, len(nodes))
	for _, n := range nodes {
		if p.isEligibleForParentAggregation(n) {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return nodeOrderLess(out[i], out[j])
	})
	return out
}

// recordAttributeLoss computes attribute loss for a group and records the
// matching leaf or parent telemetry, returning the loss summary.
func (p *spanPruningProcessor) recordAttributeLoss(ctx context.Context, isLeaf bool, nodes []*spanNode, templateNode *spanNode) attributeLossSummary {
	if !p.enableAttributeLossAnalysis {
		return attributeLossSummary{}
	}
	lossInfo := analyzeAttributeLoss(nodes, templateNode)
	if lossInfo.isEmpty() {
		return lossInfo
	}
	recordCtx := ctx
	if p.shouldSampleAttributeLossExemplar() {
		recordCtx = createExemplarContext(ctx, templateNode.span.TraceID(), templateNode.span.SpanID())
	}
	if isLeaf {
		p.telemetryBuilder.ProcessorSpanpruningLeafAttributeDiversityLoss.Record(recordCtx, int64(len(lossInfo.diverse)))
		p.telemetryBuilder.ProcessorSpanpruningLeafAttributeLoss.Record(recordCtx, int64(len(lossInfo.missing)))
	} else {
		p.telemetryBuilder.ProcessorSpanpruningParentAttributeDiversityLoss.Record(recordCtx, int64(len(lossInfo.diverse)))
		p.telemetryBuilder.ProcessorSpanpruningParentAttributeLoss.Record(recordCtx, int64(len(lossInfo.missing)))
	}
	return lossInfo
}

// detectAndProtect runs outlier detection at every level and exemplar sampling
// at the top of each aggregation tree, protecting both as whole subtrees before
// aggregation. It returns per-group outlier detection results (to annotate the
// matching summary), the preserved-outlier roots grouped by key, and the
// exemplar selection grouped by key (both linked to their summary during
// execution).
//
// Groups are processed shallowest-first so a higher group's protected subtrees
// land before any deeper group is examined. Top-level groups (closest to the
// root) sort first, so each exemplar's whole subtree is protected before deeper
// outlier detection runs and skips it. Within a top-level group, that group's
// outliers are protected before exemplars are drawn, so exemplars come from the
// remaining (non-outlier) spans and the two never overlap.
func (p *spanPruningProcessor) detectAndProtect(ctx context.Context, groups []candidateGroup) (map[string]*outlierAnalysisResult, map[string][]*spanNode, map[string]exemplarSelection) {
	outliersEnabled := p.config.EnableOutlierAnalysis
	exemplarsEnabled := p.config.EnableExemplarSampling
	if !outliersEnabled && !exemplarsEnabled {
		return nil, nil, nil
	}

	var inGroup map[*spanNode]struct{}
	if exemplarsEnabled {
		inGroup = make(map[*spanNode]struct{})
		for _, g := range groups {
			for _, n := range g.nodes {
				inGroup[n] = struct{}{}
			}
		}
	}

	outlierResultByKey := make(map[string]*outlierAnalysisResult)
	protectedRootsByKey := make(map[string][]*spanNode)
	exemplarByKey := make(map[string]exemplarSelection)
	preserve := outliersEnabled && p.config.OutlierAnalysis.PreserveOutliers

	ordered := append([]candidateGroup(nil), groups...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].nodes[0].depth() < ordered[j].nodes[0].depth()
	})

	for _, group := range ordered {
		// Groups whose spans are all inside an already-protected subtree have
		// nothing left to aggregate or preserve; skip them so detection metrics
		// don't count spans that are already kept.
		if !anyUnprotected(group.nodes) {
			continue
		}

		if outliersEnabled {
			if res := analyzeOutliers(group.nodes, p.config.OutlierAnalysis); res != nil {
				outlierResultByKey[group.key] = res
				if res.hasOutliers {
					p.telemetryBuilder.ProcessorSpanpruningOutliersDetected.Add(ctx, int64(len(res.outlierIndices)))
					if len(res.correlations) > 0 {
						p.telemetryBuilder.ProcessorSpanpruningOutliersCorrelationsDetected.Add(ctx, 1)
					}
				}
				if preserve {
					p.protectOutliers(group, res, protectedRootsByKey)
				}
			}
		}

		// Exemplar sampling runs only at the top of each aggregation tree, after
		// that group's outliers are protected. The draw is taken over the full
		// group (outliers included) but the kept exemplar roots exclude any draw
		// that is already a protected outlier. Doing it here, before any deeper
		// group is examined, protects each exemplar's whole subtree so deeper
		// detection skips it.
		if exemplarsEnabled && isTopLevelGroup(group, inGroup) {
			if sel, ok := p.sampleExemplarsFromGroup(group.nodes); ok {
				exemplarByKey[group.key] = sel
			}
		}
	}
	return outlierResultByKey, protectedRootsByKey, exemplarByKey
}

// protectOutliers selects and protects the outlier subtrees for a single group,
// appending the preserved roots to protectedRootsByKey under the group key.
func (p *spanPruningProcessor) protectOutliers(group candidateGroup, res *outlierAnalysisResult, protectedRootsByKey map[string][]*spanNode) {
	// filterOutlierNodes applies correlation gating and orders outliers most
	// extreme first. Ask it for all of them (cap 0) and apply
	// MaxPreservedOutliers here, after dropping outliers already kept by an
	// enclosing protected subtree. Letting filterOutlierNodes cap first would
	// spend a budget slot on an already-covered outlier and starve a
	// not-yet-preserved one further down the order.
	unlimited := p.config.OutlierAnalysis
	unlimited.MaxPreservedOutliers = 0
	_, outliers := filterOutlierNodes(group.nodes, res, unlimited)
	limit := p.config.OutlierAnalysis.MaxPreservedOutliers
	for _, o := range outliers {
		// Skip outliers already kept by an enclosing protected subtree.
		if o.protected {
			continue
		}
		if limit > 0 && len(protectedRootsByKey[group.key]) >= limit {
			break
		}
		markOutlierSubtree(o)
		protectedRootsByKey[group.key] = append(protectedRootsByKey[group.key], o)
	}
}

// exemplarSelection is the result of sampling exemplars from one top-level
// group: the sampled subtree roots plus the population size (N) and draw size
// (D) the executor uses to compose each exemplar's CPS threshold (D/N).
type exemplarSelection struct {
	roots          []*spanNode
	populationSize int
	sampleSize     int
}

// sampleExemplarsFromGroup draws random exemplars over the full top-level group
// (keeping is "outlier OR sampled") and protects each sampled span's whole
// subtree so it is kept intact instead of aggregated. A drawn span that is
// already a preserved outlier is kept via the outlier rule (deterministic, so
// its threshold is left untouched) and is not additionally treated as an
// exemplar. The draw size D is taken over the full group N and reported as the
// threshold numerator, because a non-outlier's inclusion probability is D/N
// regardless of how many draws land on outliers. Lower groups are never
// sampled: an exemplar is a complete representative tree, taken once at the top.
func (p *spanPruningProcessor) sampleExemplarsFromGroup(nodes []*spanNode) (exemplarSelection, bool) {
	if len(nodes) < 2 {
		return exemplarSelection{}, false
	}
	_, sampled := sampleExemplars(nodes, p.config.ExemplarSampling.PrecisionMultiplier)
	if len(sampled) == 0 {
		return exemplarSelection{}, false
	}
	roots := make([]*spanNode, 0, len(sampled))
	for _, s := range sampled {
		// A drawn outlier is already kept deterministically; don't double-tag it.
		if s.protected {
			continue
		}
		markExemplarSubtree(s)
		roots = append(roots, s)
	}
	if len(roots) == 0 {
		return exemplarSelection{}, false
	}
	return exemplarSelection{roots: roots, populationSize: len(nodes), sampleSize: len(sampled)}, true
}

// isTopLevelGroup reports whether a candidate group sits at the top of its
// aggregation tree: none of its members' parents are themselves members of a
// candidate group, so no higher group aggregates above it.
func isTopLevelGroup(g candidateGroup, inGroup map[*spanNode]struct{}) bool {
	for _, n := range g.nodes {
		if n.parent == nil {
			continue
		}
		if _, ok := inGroup[n.parent]; ok {
			return false
		}
	}
	return true
}

// recordExemplars emits exemplar telemetry: one count per sampled exemplar
// subtree kept in the output.
func (p *spanPruningProcessor) recordExemplars(ctx context.Context, roots []*spanNode) {
	if len(roots) == 0 {
		return
	}
	p.telemetryBuilder.ProcessorSpanpruningExemplarsSampled.Add(ctx, int64(len(roots)))
}

// candidateGroup is a group of sibling spans that would aggregate, paired with
// the key and aggregation depth the executor will use for it (depth 0 for a leaf
// group, >=1 for a parent group). Carrying these from planning means detection
// and execution agree without recomputing them.
type candidateGroup struct {
	key   string
	depth int
	nodes []*spanNode
}

// planCandidateGroups computes the groups that would aggregate (leaf groups at
// or above MinSpansToAggregate and eligible parent groups), ignoring outlier
// protection. It mutates no node state, tracking would-aggregate membership in a
// local set, so detection runs over the same groups the executor later forms.
func (p *spanPruningProcessor) planCandidateGroups(tree *traceTree) []candidateGroup {
	leafNodes := tree.getLeaves()
	if len(leafNodes) == 0 {
		return nil
	}

	var groups []candidateGroup
	would := make(map[*spanNode]bool)
	var marked []*spanNode

	for key, nodes := range p.groupLeafNodesByKey(leafNodes) {
		if len(nodes) < p.config.MinSpansToAggregate {
			continue
		}
		groups = append(groups, candidateGroup{key: key, depth: 0, nodes: nodes})
		for _, n := range nodes {
			would[n] = true
			marked = append(marked, n)
		}
	}

	if p.config.MaxParentDepth == 0 {
		return groups
	}

	candidates := collectParentCandidates(marked)
	depth := 1
	for len(candidates) > 0 {
		if p.config.MaxParentDepth > 0 && depth > p.config.MaxParentDepth {
			break
		}

		var eligible []*spanNode
		for _, n := range candidates {
			if planEligibleForParentAggregation(n, would) {
				eligible = append(eligible, n)
			}
		}
		if len(eligible) == 0 {
			break
		}

		parentGroups := make(map[string][]*spanNode)
		for _, n := range eligible {
			key := p.buildParentGroupKey(n.span, n.depth())
			parentGroups[key] = append(parentGroups[key], n)
		}

		marked = marked[:0]
		for key, nodes := range parentGroups {
			if len(nodes) < 2 {
				continue
			}
			groups = append(groups, candidateGroup{key: key, depth: depth, nodes: nodes})
			for _, n := range nodes {
				would[n] = true
				marked = append(marked, n)
			}
		}
		if len(marked) == 0 {
			break
		}
		candidates = collectParentCandidates(marked)
		depth++
	}
	return groups
}

// recordPreserved emits preserved-outlier telemetry: every span kept across the
// preserved subtrees rooted at the given outlier roots, not just the roots.
func (p *spanPruningProcessor) recordPreserved(ctx context.Context, roots []*spanNode) {
	if len(roots) == 0 {
		return
	}
	p.telemetryBuilder.ProcessorSpanpruningOutliersPreserved.Add(ctx, int64(preservedSpanCount(roots)))
}

// planEligibleForParentAggregation mirrors isEligibleForParentAggregation for
// the planning phase, using the local would-aggregate set rather than node
// flags and ignoring protection (not yet decided during planning).
func planEligibleForParentAggregation(node *spanNode, would map[*spanNode]bool) bool {
	if node.isLeaf || node.parent == nil || would[node] {
		return false
	}
	for _, child := range node.children {
		if !would[child] {
			return false
		}
	}
	return true
}

// anyUnprotected reports whether at least one node is not protected.
func anyUnprotected(nodes []*spanNode) bool {
	for _, n := range nodes {
		if !n.protected {
			return true
		}
	}
	return false
}

// excludeProtectedNodes returns the subset of nodes that are not protected,
// allocating only when at least one node is protected.
func excludeProtectedNodes(nodes []*spanNode) []*spanNode {
	protectedCount := 0
	for _, n := range nodes {
		if n.protected {
			protectedCount++
		}
	}
	if protectedCount == 0 {
		return nodes
	}
	out := make([]*spanNode, 0, len(nodes)-protectedCount)
	for _, n := range nodes {
		if !n.protected {
			out = append(out, n)
		}
	}
	return out
}
