// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"bytes"
	"sort"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// spanNode models a span in the trace tree with cached relationships and
// aggregation bookkeeping.
type spanNode struct {
	span               ptrace.Span
	scopeSpans         ptrace.ScopeSpans
	parent             *spanNode
	children           []*spanNode
	groupKey           string         // cached group key for leaf spans
	replacementSpanID  pcommon.SpanID // summary span ID that replaced this node's group
	isLeaf             bool           // true if node has no children
	markedForRemoval   bool           // true if node will be aggregated
	isPreservedOutlier bool           // true if this node is the root of a preserved outlier subtree
	isExemplar         bool           // true if this node is the root of a sampled exemplar subtree
	protected          bool           // true if this node is within a preserved outlier or exemplar subtree (never aggregated)
}

// traceTree holds span nodes indexed by ID plus quick leaf/orphan lists for
// efficient aggregation analysis.
type traceTree struct {
	nodeByID map[pcommon.SpanID]*spanNode
	leaves   []*spanNode // nodes with no children, populated during build
	orphans  []*spanNode // spans whose parent is not in the trace
}

// buildTraceTree constructs parent/child links for a trace and records
// leaves, roots, and orphans so aggregation decisions can account for
// incomplete traces.
func (p *spanPruningProcessor) buildTraceTree(spans []spanInfo) *traceTree {
	tree := &traceTree{
		nodeByID: make(map[pcommon.SpanID]*spanNode, len(spans)),
	}

	if len(spans) == 0 {
		return tree
	}

	// First pass: create nodes for all spans, initially mark all as leaves
	for _, info := range spans {
		node := &spanNode{
			span:       info.span,
			scopeSpans: info.scopeSpans,
			isLeaf:     true, // assume leaf until a child links to it
		}
		tree.nodeByID[info.span.SpanID()] = node
	}

	// Second pass: link parent-child relationships and update leaf status
	// Pre-allocate slices with reasonable capacity
	tree.orphans = make([]*spanNode, 0, len(spans)/10)
	var rootCount int

	for _, node := range tree.nodeByID {
		parentID := node.span.ParentSpanID()
		if parentID.IsEmpty() {
			// This is a root span (no parent)
			rootCount++
		} else if parent, exists := tree.nodeByID[parentID]; exists {
			// Link to parent and mark parent as non-leaf
			node.parent = parent
			parent.isLeaf = false
			if parent.children == nil {
				parent.children = make([]*spanNode, 0, 4)
			}
			parent.children = append(parent.children, node)
		} else {
			// Parent not in trace - this is an orphan
			tree.orphans = append(tree.orphans, node)
		}
	}

	// Third pass: collect leaves (nodes still marked as leaf). nodeByID is a
	// map, so iteration order is random; sort the leaves into a stable order
	// (start time, then span ID) so downstream grouping, summary anchoring, and
	// outlier reparenting are deterministic across runs.
	tree.leaves = make([]*spanNode, 0, len(spans)/4)
	for _, node := range tree.nodeByID {
		if node.isLeaf {
			tree.leaves = append(tree.leaves, node)
		}
	}
	sort.Slice(tree.leaves, func(i, j int) bool {
		return nodeOrderLess(tree.leaves[i], tree.leaves[j])
	})

	// Log warnings for incomplete traces
	if rootCount > 1 {
		p.logger.Debug("multiple root spans found",
			zap.Int("rootCount", rootCount))
	} else if rootCount == 0 && len(tree.orphans) > 0 {
		p.logger.Debug("no root span found, trace may be incomplete")
	}

	if len(tree.orphans) > 0 {
		p.logger.Debug("orphaned spans detected",
			zap.Int("orphanCount", len(tree.orphans)))
	}

	return tree
}

// getLeaves returns the pre-computed leaf nodes (spans with no children).
func (t *traceTree) getLeaves() []*spanNode {
	return t.leaves
}

// nodeOrderLess orders span nodes by start time, then span ID. It gives a
// stable order that does not depend on map iteration, so leaf grouping, parent
// candidate grouping, and summary anchoring are deterministic across runs.
func nodeOrderLess(a, b *spanNode) bool {
	sa, sb := a.span, b.span
	if sa.StartTimestamp() != sb.StartTimestamp() {
		return sa.StartTimestamp() < sb.StartTimestamp()
	}
	aID, bID := sa.SpanID(), sb.SpanID()
	return bytes.Compare(aID[:], bID[:]) < 0
}

// depth returns the node's depth in the trace tree (root spans and orphans are
// depth 0). It walks the cached parent chain, so cost is proportional to tree
// height.
func (n *spanNode) depth() int {
	d := 0
	for p := n.parent; p != nil; p = p.parent {
		d++
	}
	return d
}

// markOutlierSubtree records root as a preserved-outlier root and marks every
// node in its subtree (root included) as protected, so the whole subtree is kept
// (never aggregated). It is outlier-specific (it sets isPreservedOutlier); other
// preservation features should not reuse it.
func markOutlierSubtree(root *spanNode) {
	root.isPreservedOutlier = true
	for _, n := range subtreeNodes(root) {
		n.protected = true
	}
}

// markExemplarSubtree records root as a sampled-exemplar root and marks every
// node in its subtree (root included) as protected, so the whole subtree is kept
// intact (never aggregated). It is exemplar-specific (it sets isExemplar); it
// deliberately does not reuse markOutlierSubtree so exemplars never acquire an
// outlier flag.
func markExemplarSubtree(root *spanNode) {
	root.isExemplar = true
	for _, n := range subtreeNodes(root) {
		n.protected = true
	}
}

// subtreeNodes returns root and all of its descendants.
func subtreeNodes(root *spanNode) []*spanNode {
	nodes := []*spanNode{root}
	for i := 0; i < len(nodes); i++ {
		nodes = append(nodes, nodes[i].children...)
	}
	return nodes
}

// preservedSpanCount returns the total number of spans across the subtrees
// rooted at the given preserved-outlier roots.
func preservedSpanCount(roots []*spanNode) int {
	total := 0
	for _, r := range roots {
		total += len(subtreeNodes(r))
	}
	return total
}

// collectParentCandidates returns unique parents of marked nodes for the
// next aggregation depth iteration.
func collectParentCandidates(markedNodes []*spanNode) []*spanNode {
	if len(markedNodes) == 0 {
		return nil
	}

	seen := make(map[*spanNode]struct{}, len(markedNodes)/2)
	candidates := make([]*spanNode, 0, len(markedNodes)/2)

	for _, node := range markedNodes {
		if node.parent != nil {
			if _, exists := seen[node.parent]; !exists {
				seen[node.parent] = struct{}{}
				candidates = append(candidates, node.parent)
			}
		}
	}

	return candidates
}

// isEligibleForParentAggregation verifies that a node meets the criteria for
// parent aggregation: not a leaf, not a root, not already aggregated, not itself
// a preserved outlier, and every child either aggregated or part of a preserved
// outlier subtree. A preserved subtree is reparented onto this node's summary,
// so it does not block aggregation. The outlier keeps everything beneath it but
// not its ancestors.
func (*spanPruningProcessor) isEligibleForParentAggregation(node *spanNode) bool {
	// Must have children (not a leaf)
	if node.isLeaf {
		return false
	}

	// Must have a parent (not root)
	if node.parent == nil {
		return false
	}

	// Must not already be marked for removal, and must not itself be within a
	// protected subtree (preserved outliers and sampled exemplars are kept, not
	// aggregated).
	if node.markedForRemoval || node.protected {
		return false
	}

	// All children must be aggregated or part of a protected (preserved outlier
	// or sampled exemplar) subtree.
	for _, child := range node.children {
		if !child.markedForRemoval && !child.protected {
			return false
		}
	}

	return true
}
