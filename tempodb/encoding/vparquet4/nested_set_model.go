package vparquet4

import (
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

// nestedSetRootParent is used for the root span's ParentID field. this allows the fetch layer (and
// other code) to distinguish between situations in which a span's parent is unknown due to a broken trace
// or simply known to not exist.
const nestedSetRootParent = -1

// spanNode is a wrapper around a span that is used to build and travers spans as a tree.
type spanNode struct {
	parent    *spanNode
	span      *Span
	children  []*spanNode
	nextChild int
}

// assignNestedSetModelBoundsAndServiceStats calculates and assigns the values Span.NestedSetLeft, Span.NestedSetRight,
// and Span.ParentID for all spans in a trace.
// Additionally, it calculates per-service statistics of the trace.
// Returns true if the trace tree is a connected graph which is useful for calculating data quality
func assignNestedSetModelBoundsAndServiceStats(trace *Trace) bool {
	// count spans in order be able to pre-allocate tree nodes
	var spanCount int
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			spanCount += len(ss.Spans)
		}
	}

	// find root spans and map span IDs to tree nodes
	var (
		undoAssignment bool
		allNodes       = make([]spanNode, 0, spanCount)
		nodesByID      = make(map[uint64][]*spanNode, spanCount)
		rootNodes      []*spanNode
	)

	// initialize ServiceStats (spanCount and errorCount per service)
	trace.ServiceStats = map[string]ServiceStats{}

	for _, rs := range trace.ResourceSpans {
		serviceStats := trace.ServiceStats[rs.Resource.ServiceName]

		for _, ss := range rs.ScopeSpans {
			serviceStats.SpanCount += uint32(len(ss.Spans))

			for i, s := range ss.Spans {
				allNodes = append(allNodes, spanNode{span: &ss.Spans[i]})
				node := &allNodes[len(allNodes)-1]

				if s.IsRoot() {
					rootNodes = append(rootNodes, node)
				}

				id := util.SpanIDToUint64(s.SpanID)
				if nodes, ok := nodesByID[id]; ok {
					// zipkin traces may contain client/server spans with the same IDs
					nodes = append(nodes, node)
					nodesByID[id] = nodes
					if len(nodes) > 2 {
						undoAssignment = true
					}
				} else {
					nodesByID[id] = []*spanNode{node}
				}

				if s.StatusCode == int(v1.Status_STATUS_CODE_ERROR) {
					serviceStats.ErrorCount++
				}
			}
		}

		trace.ServiceStats[rs.Resource.ServiceName] = serviceStats
	}

	// check preconditions before assignment
	if len(rootNodes) == 0 {
		return false
	}
	if undoAssignment {
		for _, nodes := range nodesByID {
			for _, n := range nodes {
				n.span.NestedSetLeft = 0
				n.span.NestedSetRight = 0
				n.span.ParentID = 0
			}
		}
		// this trace has over 2 spans with the same span id. the data is invalid and therefore we are preferring "false",
		// but semantically it's different then detecting a disconnected graph
		return false
	}

	connected := true
	// build the tree
	for i := range allNodes {
		node := &allNodes[i]
		parent := findParentNodeInMap(nodesByID, node)
		if parent == nil {
			// if we find a node without a parent that's not root, it's not a connected graph
			if !node.span.IsRoot() {
				connected = false
			}
			continue
		}
		node.parent = parent
		parent.children = append(parent.children, node)
	}

	// traverse the tree depth first. When going down the tree, assign NestedSetLeft
	// and assign NestedSetRight when going up.
	nestedSetBound := int32(1)
	for _, root := range rootNodes {
		node := root
		node.span.NestedSetLeft = nestedSetBound
		node.span.ParentID = nestedSetRootParent
		nestedSetBound++

		for node != nil {
			if node.nextChild < len(node.children) {
				// the current node has children that were not visited: go down to next child

				next := node.children[node.nextChild]
				node.nextChild++

				next.span.NestedSetLeft = nestedSetBound
				next.span.ParentID = node.span.NestedSetLeft // the left bound of the parent serves as numeric span ID
				nestedSetBound++
				node = next
			} else {
				// all children of the current node were visited: go up

				node.span.NestedSetRight = nestedSetBound
				nestedSetBound++

				node = node.parent
			}
		}
	}

	return connected
}

// findParentNodeInMap finds the tree node containing the parent span for another node. zipkin traces can
// contain client/server span pairs with identical span IDs. In those cases the span kind is used to find
// the matching parent span.
func findParentNodeInMap(nodesByID map[uint64][]*spanNode, node *spanNode) *spanNode {
	if node.span.IsRoot() {
		return nil
	}

	parentID := util.SpanIDToUint64(node.span.ParentSpanID)
	nodes := nodesByID[parentID]

	switch len(nodes) {
	case 0:
		return nil
	case 1:
		return nodes[0]
	case 2:
		// handle client/server spans with the same span ID
		kindWant := int(v1.Span_SPAN_KIND_SERVER)
		if node.span.Kind == int(v1.Span_SPAN_KIND_SERVER) {
			kindWant = int(v1.Span_SPAN_KIND_CLIENT)
		}

		if nodes[0].span.Kind == kindWant {
			return nodes[0]
		}
		if nodes[1].span.Kind == kindWant {
			return nodes[1]
		}
	}

	return nil
}
