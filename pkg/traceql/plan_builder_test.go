package traceql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// planTreeString renders a plan tree as an indented ASCII diagram, e.g.:
//
//	└── Rate
//	    └── GroupBy(resource.service.name)
//	        └── SpansetFilter(...)
//	            └── TraceScan(1 conditions)
//	                └── ResourceScan(0 conditions)
//	                    └── InstrumentationScopeScan(0 conditions)
//	                        └── SpanScan(1 conditions)
func planTreeString(n PlanNode) string {
	var sb strings.Builder
	writePlanNode(&sb, n, "", true)
	return sb.String()
}

func writePlanNode(sb *strings.Builder, n PlanNode, prefix string, isLast bool) {
	connector := "└── "
	childPrefix := prefix + "    "
	if !isLast {
		connector = "├── "
		childPrefix = prefix + "│   "
	}
	sb.WriteString(prefix + connector + n.String() + "\n")
	children := n.Children()
	for i, c := range children {
		writePlanNode(sb, c, childPrefix, i == len(children)-1)
	}
}

func TestBuildPlan_SimpleFilter(t *testing.T) {
	// { span.status = error }
	expr, err := Parse(`{ span.status = error }`)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// After optimizer: predicate should be pushed into a SpanScanNode.
	var spanScan *SpanScanNode
	WalkPlan(plan, &funcVisitor{
		pre: func(n PlanNode) bool {
			if s, ok := n.(*SpanScanNode); ok {
				spanScan = s
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.NotNil(t, spanScan, "expected SpanScanNode in plan")
}

func TestBuildPlan_EmptyFilter(t *testing.T) {
	// { } — trivially true filter
	expr, err := Parse(`{ }`)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)
	require.NotNil(t, plan)
}

func TestBuildPlan_GroupBy(t *testing.T) {
	// { span.status = error } | by(resource.service.name)
	expr, err := Parse(`{ span.status = error } | by(resource.service.name)`)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)

	_, ok := plan.(*GroupByNode)
	require.True(t, ok, "expected GroupByNode at root, got %T", plan)
}

func TestBuildPlan_WithSelect(t *testing.T) {
	// { span.status = error } | by(resource.service.name) | select(span.region)
	// GroupByNode → ProjectNode → ... scan tree ...
	expr, err := Parse(`{ span.status = error } | by(resource.service.name) | select(span.region)`)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)

	group, ok := plan.(*GroupByNode)
	require.True(t, ok, "expected GroupByNode at root, got %T", plan)

	proj, ok := group.Children()[0].(*ProjectNode)
	require.True(t, ok, "expected ProjectNode as GroupByNode child, got %T", group.Children()[0])
	require.Len(t, proj.Columns, 1)
}

func TestBuildPlan_MetricsRate(t *testing.T) {
	expr, err := Parse(`{ } | rate()`)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)

	_, ok := plan.(*RateNode)
	require.True(t, ok, "expected RateNode at root, got %T", plan)
}

func TestBuildPlan_MetricsCountOverTime(t *testing.T) {
	expr, err := Parse(`{ } | count_over_time()`)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)

	_, ok := plan.(*CountOverTimeNode)
	require.True(t, ok, "expected CountOverTimeNode at root, got %T", plan)
}

func TestBuildPlan_HasScanTree(t *testing.T) {
	// Every plan should have a TraceScanNode at its base.
	queries := []string{
		`{ span.foo = "bar" }`,
		`{ } | rate()`,
		`{ } | by(resource.service.name)`,
	}
	for _, q := range queries {
		expr, err := Parse(q)
		require.NoError(t, err, q)

		plan, err := BuildPlan(expr, nil)
		require.NoError(t, err, q)

		var traceScan *TraceScanNode
		WalkPlan(plan, &funcVisitor{
			pre: func(n PlanNode) bool {
				if ts, ok := n.(*TraceScanNode); ok {
					traceScan = ts
				}
				return true
			},
			post: func(PlanNode) {},
		})
		require.NotNil(t, traceScan, "expected TraceScanNode in plan for query: %s", q)
	}
}

// TestBuildPlan_MetricsQuery_ShowTree compiles a TraceQL metric query and
// prints the resulting logical plan tree.
//
// Query: { span.http.status_code >= 500 } | rate() by(resource.service.name)
//
// Expected shape (after optimizer):
//
//	└── Rate
//	    └── TraceScan(1 conditions)         ← http.status_code pushed down
//	        └── ResourceScan(0 conditions)
//	            └── InstrumentationScopeScan(0 conditions)
//	                └── SpanScan(1 conditions)
func TestBuildPlan_MetricsQuery_ShowTree(t *testing.T) {
	const query = `{ span.http.status_code >= 500 } | rate() by(resource.service.name)`

	expr, err := Parse(query)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)
	require.NotNil(t, plan)

	tree := planTreeString(plan)
	t.Logf("Logical plan for %q:\n%s", query, tree)

	// Root must be a RateNode (metrics aggregation).
	rateNode, ok := plan.(*RateNode)
	require.True(t, ok, "expected RateNode at root, got %T", plan)
	require.Equal(t, 1, len(rateNode.By), "expected 1 by-attribute")

	// There must be a TraceScanNode somewhere in the tree (base scan).
	var traceScan *TraceScanNode
	WalkPlan(plan, &funcVisitor{
		pre: func(n PlanNode) bool {
			if ts, ok := n.(*TraceScanNode); ok {
				traceScan = ts
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.NotNil(t, traceScan, "expected TraceScanNode in plan")

	// There must be a SpanScanNode with at least one condition (http.status_code pushed down).
	var spanScan *SpanScanNode
	WalkPlan(plan, &funcVisitor{
		pre: func(n PlanNode) bool {
			if s, ok := n.(*SpanScanNode); ok {
				spanScan = s
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.NotNil(t, spanScan, "expected SpanScanNode in plan")
	require.NotEmpty(t, spanScan.Conditions, "expected http.status_code predicate pushed into SpanScanNode")
}
