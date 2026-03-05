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
	proj := NewProjectNode([]Attribute{NewScopedAttribute(AttributeScopeSpan, false, "region")}, group, nil)
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

func TestProjectNodeTwoChildren(t *testing.T) {
	drivingChild := &SpanScanNode{Conditions: []Condition{{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"), Op: OpGreater}}}
	fetchSpan := &SpanScanNode{Conditions: []Condition{{Attribute: NewIntrinsic(IntrinsicSpanID), Op: OpNone}}}
	fetchTree := &TraceScanNode{child: &ResourceScanNode{child: &InstrumentationScopeScanNode{child: fetchSpan}}}

	proj := NewProjectNode([]Attribute{NewIntrinsic(IntrinsicSpanID)}, drivingChild, fetchTree)
	children := proj.Children()

	require.Len(t, children, 2)
	require.Equal(t, drivingChild, children[0])
	require.Equal(t, fetchTree, children[1])
	require.Contains(t, proj.String(), "Project")
}

func TestBuildFetchScanTree(t *testing.T) {
	columns := []Attribute{
		NewIntrinsic(IntrinsicTraceRootService), // trace level
		NewIntrinsic(IntrinsicTraceRootSpan),    // trace level
		NewIntrinsic(IntrinsicSpanID),           // span level
		NewIntrinsic(IntrinsicDuration),         // span level
		NewIntrinsic(IntrinsicServiceStats),     // trace level
	}

	tree := BuildFetchScanTree(columns)

	// Root is TraceScanNode with trace-level conditions
	require.NotNil(t, tree)
	traceConds := tree.Conditions
	var traceAttrs []Intrinsic
	for _, c := range traceConds {
		require.Equal(t, OpNone, c.Op)
		traceAttrs = append(traceAttrs, c.Attribute.Intrinsic)
	}
	require.Contains(t, traceAttrs, IntrinsicTraceRootService)
	require.Contains(t, traceAttrs, IntrinsicTraceRootSpan)
	require.Contains(t, traceAttrs, IntrinsicServiceStats)

	// Trace → Resource → InstrScope → Span
	require.Len(t, tree.Children(), 1)
	resNode, ok := tree.Children()[0].(*ResourceScanNode)
	require.True(t, ok)

	require.Len(t, resNode.Children(), 1)
	instrNode, ok := resNode.Children()[0].(*InstrumentationScopeScanNode)
	require.True(t, ok)

	require.Len(t, instrNode.Children(), 1)
	spanNode, ok := instrNode.Children()[0].(*SpanScanNode)
	require.True(t, ok)

	var spanAttrs []Intrinsic
	for _, c := range spanNode.Conditions {
		require.Equal(t, OpNone, c.Op)
		spanAttrs = append(spanAttrs, c.Attribute.Intrinsic)
	}
	require.Contains(t, spanAttrs, IntrinsicSpanID)
	require.Contains(t, spanAttrs, IntrinsicDuration)
}

func TestSearchMetaColumns(t *testing.T) {
	cols := SearchMetaColumns()
	require.Len(t, cols, 9)

	// Verify they match SearchMetaConditions attributes
	conds := SearchMetaConditions()
	for i, col := range cols {
		require.Equal(t, conds[i].Attribute, col)
	}
}

