package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScanNodeChildren verifies that each scan node returns the correct children.
func TestScanNodeChildren(t *testing.T) {
	span := NewSpanScanNode([]Condition{{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "status")}}, nil)
	res := NewResourceScanNode([]Condition{{Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name")}}, span)
	trace := NewTraceScanNode(nil, false, res)

	require.Equal(t, []PlanNode{res}, trace.Children())
	require.Equal(t, []PlanNode{span}, res.Children())
	require.Nil(t, span.Children())
	require.Contains(t, trace.String(), "TraceScan")
	require.Contains(t, res.String(), "ResourceScan")
	require.Contains(t, span.String(), "SpanScan")
}

// TestEngineNodes verifies engine and metrics node children/strings.
func TestEngineNodes(t *testing.T) {
	filter := NewSpansetFilterNode(&SpansetFilter{}, NewSpanScanNode(nil, nil))
	group := NewGroupByNode(NewScopedAttribute(AttributeScopeResource, false, "service.name"), filter)
	proj := NewProjectNode([]Attribute{NewScopedAttribute(AttributeScopeSpan, false, "region")}, group)
	rate := NewRateNode(nil, proj)

	require.Equal(t, []PlanNode{group}, proj.Children())
	require.Equal(t, []PlanNode{filter}, group.Children())
	require.Contains(t, proj.String(), "Project")
	require.Contains(t, group.String(), "GroupBy")
	require.Contains(t, rate.String(), "Rate")
}

// TestWalkPlanOrder verifies depth-first pre- and post-order traversal.
func TestWalkPlanOrder(t *testing.T) {
	span := NewSpanScanNode(nil, nil)
	res := NewResourceScanNode(nil, span)
	trace := NewTraceScanNode(nil, false, res)

	var preOrder, postOrder []string
	v := &funcVisitor{
		pre: func(n PlanNode) bool {
			preOrder = append(preOrder, n.String())
			return true
		},
		post: func(n PlanNode) {
			postOrder = append(postOrder, n.String())
		},
	}
	WalkPlan(trace, v)

	// pre-order: root first, then children depth-first
	require.Equal(t, trace.String(), preOrder[0])
	require.Equal(t, res.String(), preOrder[1])
	require.Equal(t, span.String(), preOrder[2])
	// post-order: leaves first
	require.Equal(t, span.String(), postOrder[0])
	require.Equal(t, res.String(), postOrder[1])
	require.Equal(t, trace.String(), postOrder[2])
}

// TestWalkPlanSkipChildren verifies that returning false from VisitPre skips children.
func TestWalkPlanSkipChildren(t *testing.T) {
	span := NewSpanScanNode(nil, nil)
	res := NewResourceScanNode(nil, span)
	trace := NewTraceScanNode(nil, false, res)

	var visited []string
	v := &funcVisitor{
		pre: func(n PlanNode) bool {
			visited = append(visited, n.String())
			// Stop at ResourceScanNode — don't descend into SpanScanNode
			_, isRes := n.(*ResourceScanNode)
			return !isRes
		},
		post: func(n PlanNode) {},
	}
	WalkPlan(trace, v)

	require.Equal(t, []string{trace.String(), res.String()}, visited)
}

// TestStructuralOpNodeChildren verifies that StructuralOpNode returns both children.
func TestStructuralOpNodeChildren(t *testing.T) {
	left := NewSpanScanNode(nil, nil)
	right := NewSpanScanNode(nil, nil)
	op := NewStructuralOpNode(StructuralOpParent, left, right)

	require.Equal(t, []PlanNode{left, right}, op.Children())
	require.Contains(t, op.String(), "StructuralOp")
}
