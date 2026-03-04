package traceql

import (
	"context"
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

// TestBuildPlan_SpansetOperation_SingleScan verifies that a structural query
// generates a SpansetRelationNode (single scan) rather than a StructuralOpNode
// (two scans).
func TestBuildPlan_SpansetOperation_SingleScan(t *testing.T) {
	expr, err := Parse(`{ span.http.method = "GET" } >> { span.db.system = "postgresql" }`)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)

	t.Logf("plan:\n%s", planTreeString(plan))

	// Root must be SpansetRelationNode, NOT StructuralOpNode.
	exprNode, ok := plan.(*SpansetRelationNode)
	require.True(t, ok, "expected SpansetRelationNode at root, got %T", plan)
	require.Equal(t, OpSpansetDescendant, exprNode.Expr.Op)

	// Must have exactly ONE child (single scan tree).
	require.Len(t, exprNode.Children(), 1)

	// Single scan: both span predicates must be present somewhere.
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
	require.NotNil(t, spanScan)
	require.GreaterOrEqual(t, len(spanScan.Conditions), 2,
		"conditions from both LHS and RHS must be in the merged scan")
}

// TestBuildPlan_StructuralMetrics_ShowTree exercises a query that combines a
// structural operator, a compound AND predicate (now pushable), a simple
// predicate (pushable), and a metrics aggregation with a by-clause.
//
// Query:
//
//	{ span.http.method = "GET" && span.http.status_code >= 500 }
//	  >> { span.db.system = "postgresql" }
//	| count_over_time() by (resource.service.name)
//
// Expected plan after optimizer:
//
//	└── CountOverTime by(resource.service.name)
//	    └── SpansetExpr(>>)
//	        └── TraceScan                        ← single merged scan
//	            └── ResourceScan
//	                └── InstrumentationScopeScan
//	                    └── SpanScan(span.http.method, span.http.status_code, span.db.system)
func TestBuildPlan_StructuralMetrics_ShowTree(t *testing.T) {
	const query = `{ span.http.method = "GET" && span.http.status_code >= 500 } >> { span.db.system = "postgresql" } | count_over_time() by (resource.service.name)`

	expr, err := Parse(query)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)
	require.NotNil(t, plan)

	tree := planTreeString(plan)
	t.Logf("Logical plan for:\n  %s\n\n%s", query, tree)

	// --- root: CountOverTimeNode with one by-attribute ---
	cot, ok := plan.(*CountOverTimeNode)
	require.True(t, ok, "expected CountOverTimeNode at root, got %T", plan)
	require.Len(t, cot.By, 1)
	require.Equal(t, "resource.service.name", cot.By[0].String())

	// --- child: SpansetRelationNode (single-scan strategy) ---
	exprNode, ok := cot.Children()[0].(*SpansetRelationNode)
	require.True(t, ok, "expected SpansetRelationNode as CountOverTime child, got %T", cot.Children()[0])
	require.Equal(t, OpSpansetDescendant, exprNode.Expr.Op)

	// Single scan: all predicates (from both LHS and RHS) merged into one SpanScan.
	var spanScan *SpanScanNode
	WalkPlan(exprNode, &funcVisitor{
		pre: func(n PlanNode) bool {
			if s, ok := n.(*SpanScanNode); ok {
				spanScan = s
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.NotNil(t, spanScan)
	require.GreaterOrEqual(t, len(spanScan.Conditions), 3,
		"merged scan must contain predicates from both sides of >>")
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

// TestBuildPlan_PipelineAB_ShowTree shows the logical plan for { A } | { B }.
// The pipe operator sequences two filter stages; this test prints the tree so
// we can see what shape the plan builder and optimizer produce.
func TestBuildPlan_PipelineAB_ShowTree(t *testing.T) {
	const query = `{ span.http.method = "GET" } | { span.db.system = "postgresql" }`

	expr, err := Parse(query)
	require.NoError(t, err)

	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)

	t.Logf("Plan for %q:\n%s", query, planTreeString(plan))
}

func TestBuildPlan_PipelineSpanResource_ShowTree(t *testing.T) {
	const query = `{ span.name = "checkout" } | { resource.service.name = "frontend" }`
	expr, err := Parse(query)
	require.NoError(t, err)
	plan, err := BuildPlan(expr, nil)
	require.NoError(t, err)
	t.Logf("Plan for %q:\n%s", query, planTreeString(plan))
}

// TestStructuralOp_EvalCrossCheck verifies that RelationSpansetIter (the plan's
// single-scan strategy) produces the same result as pipeline.evaluate().
//
// Trace layout:
//
//	Trace 1 (traceID {1})
//	  httpSpan  left=1 right=6  span.http.method="GET" span.http.status_code=503  ← matches LHS
//	  dbChild   left=2 right=5  span.db.system="postgresql"                        ← matches RHS + descendant of httpSpan ✓
//	  dbOrphan  left=7 right=8  span.db.system="postgresql"                        ← matches RHS but NOT descendant ✗
//
//	Trace 2 (traceID {2})
//	  httpPost  left=1 right=4  span.http.method="POST"                            ← does NOT match LHS
//	  dbChild2  left=2 right=3  span.db.system="postgresql"                        ← matches RHS but parent fails LHS ✗
//
// Expected: only trace1/dbChild survives in both approaches.
func TestStructuralOp_EvalCrossCheck(t *testing.T) {
	const query = `{ span.http.method = "GET" && span.http.status_code >= 500 } >> { span.db.system = "postgresql" }`

	traceID1 := []byte{1}
	traceID2 := []byte{2}

	// Trace 1 spans
	httpSpan := newMockSpan([]byte{1, 1}).
		WithSpanString("http.method", "GET").
		WithSpanInt("http.status_code", 503).
		WithNestedSetInfo(0, 1, 6)
	dbChild := newMockSpan([]byte{1, 2}).
		WithSpanString("db.system", "postgresql").
		WithNestedSetInfo(1, 2, 5) // descendant: 2>1 && 2<6 ✓
	dbOrphan := newMockSpan([]byte{1, 3}).
		WithSpanString("db.system", "postgresql").
		WithNestedSetInfo(-1, 7, 8) // not descendant: 7<6 is false ✗

	// Trace 2 spans
	httpPost := newMockSpan([]byte{2, 1}).
		WithSpanString("http.method", "POST").
		WithSpanInt("http.status_code", 503).
		WithNestedSetInfo(0, 1, 4)
	dbChild2 := newMockSpan([]byte{2, 2}).
		WithSpanString("db.system", "postgresql").
		WithNestedSetInfo(1, 2, 3)

	ast, err := Parse(query)
	require.NoError(t, err)

	spansets := []*Spanset{
		{TraceID: traceID1, Spans: []Span{httpSpan, dbChild, dbOrphan}},
		{TraceID: traceID2, Spans: []Span{httpPost, dbChild2}},
	}

	// --- Strategy 1: pipeline.evaluate() ---
	pipelineResult, err := ast.Pipeline.evaluate(spansets)
	require.NoError(t, err)
	require.Len(t, pipelineResult, 1, "only trace1 should match")
	require.Len(t, pipelineResult[0].Spans, 1, "only dbChild survives")
	require.Equal(t, dbChild.id, pipelineResult[0].Spans[0].ID())

	// --- Strategy 2: RelationSpansetIter (plan's single-scan strategy) ---
	// All spans per trace arrive in one spanset; RelationSpansetIter calls
	// SpansetOperation.evaluate in-memory, matching pipeline.evaluate exactly.
	op, ok := ast.Pipeline.Elements[0].(SpansetOperation)
	require.True(t, ok)

	childIter := &sliceSpansetIter{spansets: []*Spanset{
		{TraceID: traceID1, Spans: []Span{httpSpan, dbChild, dbOrphan}},
		{TraceID: traceID2, Spans: []Span{httpPost, dbChild2}},
	}}
	exprResult := drainIter(t, RelationSpansetIter(op, childIter))

	require.Len(t, exprResult, 1, "only trace1 should match")
	require.Len(t, exprResult[0].Spans, 1, "only dbChild survives")
	require.Equal(t, dbChild.id, exprResult[0].Spans[0].ID())
}

func drainIter(t *testing.T, iter SpansetIterator) []*Spanset {
	t.Helper()
	defer iter.Close()
	var out []*Spanset
	for {
		ss, err := iter.Next(context.Background())
		require.NoError(t, err)
		if ss == nil {
			break
		}
		out = append(out, ss)
	}
	return out
}

// sliceSpansetIter is a trivial SpansetIterator backed by a pre-built slice.
type sliceSpansetIter struct {
	spansets []*Spanset
	idx      int
}

func (s *sliceSpansetIter) Next(context.Context) (*Spanset, error) {
	if s.idx >= len(s.spansets) {
		return nil, nil
	}
	ss := s.spansets[s.idx]
	s.idx++
	return ss, nil
}

func (s *sliceSpansetIter) Close() {}
