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

// TestBuildPlan_StructuralMetrics_ShowTree exercises a query that combines a
// structural operator, a compound AND predicate (not pushable), a simple
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
//	    └── StructuralOp descendant(>>)
//	        ├── SpansetFilter(...)               ← compound AND stays (not pushable)
//	        │   └── TraceScan
//	        │       └── ResourceScan
//	        │           └── InstrumentationScopeScan
//	        │               └── SpanScan         ← no conditions pushed
//	        └── TraceScan                        ← simple predicate was pushed
//	            └── ResourceScan
//	                └── InstrumentationScopeScan
//	                    └── SpanScan(span.db.system = "postgresql")
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

	// --- child: StructuralOpNode(parent >>) ---
	structOp, ok := cot.Children()[0].(*StructuralOpNode)
	require.True(t, ok, "expected StructuralOpNode as CountOverTime child, got %T", cot.Children()[0])
	require.Equal(t, StructuralOpDescendant, structOp.Op)

	left, right := structOp.Children()[0], structOp.Children()[1]

	// --- left branch: compound AND keeps a SpansetFilterNode ---
	// isSimplePerSpanFilter rejects BinaryOperation(AND), so the filter node survives.
	var leftFilter *SpansetFilterNode
	WalkPlan(left, &funcVisitor{
		pre: func(n PlanNode) bool {
			if f, ok := n.(*SpansetFilterNode); ok {
				leftFilter = f
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.NotNil(t, leftFilter, "left branch: expected SpansetFilterNode (compound AND can't be pushed down)")

	// The SpanScan on the left should have no pushed-down conditions.
	var leftSpanScan *SpanScanNode
	WalkPlan(left, &funcVisitor{
		pre: func(n PlanNode) bool {
			if s, ok := n.(*SpanScanNode); ok {
				leftSpanScan = s
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.NotNil(t, leftSpanScan)
	require.Empty(t, leftSpanScan.Conditions, "left SpanScan: compound AND must not be pushed down")

	// --- right branch: simple equality pushed into SpanScan ---
	// isSimplePerSpanFilter accepts BinaryOperation(span attr, static), so the
	// SpansetFilterNode is eliminated and its predicate lives in SpanScan.
	var rightFilter *SpansetFilterNode
	WalkPlan(right, &funcVisitor{
		pre: func(n PlanNode) bool {
			if f, ok := n.(*SpansetFilterNode); ok {
				rightFilter = f
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.Nil(t, rightFilter, "right branch: SpansetFilterNode should be eliminated by predicate pushdown")

	var rightSpanScan *SpanScanNode
	WalkPlan(right, &funcVisitor{
		pre: func(n PlanNode) bool {
			if s, ok := n.(*SpanScanNode); ok {
				rightSpanScan = s
			}
			return true
		},
		post: func(PlanNode) {},
	})
	require.NotNil(t, rightSpanScan)
	require.NotEmpty(t, rightSpanScan.Conditions, "right SpanScan: span.db.system predicate must be pushed down")
	require.Equal(t, "span.db.system", rightSpanScan.Conditions[0].Attribute.String())
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

// TestStructuralOp_EvalCrossCheck verifies that the plan's two-scan strategy
// (StructuralOpSpansetIter fed by two independent iterators) produces the same
// evaluation result as pipeline.evaluate() — the current single-scan strategy.
//
// Trace layout used by both strategies:
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

	// --- Strategy 1: pipeline.evaluate() (current single-scan approach) ---
	// All spans from each trace arrive in one spanset; evaluate() handles filtering
	// and the structural join internally.
	ast, err := Parse(query)
	require.NoError(t, err)

	singleScanResult, err := ast.Pipeline.evaluate([]*Spanset{
		{TraceID: traceID1, Spans: []Span{httpSpan, dbChild, dbOrphan}},
		{TraceID: traceID2, Spans: []Span{httpPost, dbChild2}},
	})
	require.NoError(t, err)

	t.Logf("pipeline.evaluate: %d spanset(s)", len(singleScanResult))
	for i, ss := range singleScanResult {
		t.Logf("  [%d] traceID=%v  spans=%d", i, ss.TraceID, len(ss.Spans))
		for j, s := range ss.Spans {
			t.Logf("    span[%d] id=%v", j, s.ID())
		}
	}

	require.Len(t, singleScanResult, 1, "only trace1 should match")
	require.Len(t, singleScanResult[0].Spans, 1, "only dbChild survives")
	require.Equal(t, dbChild.id, singleScanResult[0].Spans[0].ID())

	// --- Strategy 2: StructuralOpSpansetIter (plan's two-scan approach) ---
	// Left iterator is pre-filtered to LHS-matching spans; right to RHS-matching spans.
	// The iterator merge-joins by traceID and evaluates the structural relationship.
	//
	// Note: trace2 has no entry in left because httpPost doesn't match LHS.
	leftIter := &sliceSpansetIter{spansets: []*Spanset{
		{TraceID: traceID1, Spans: []Span{httpSpan}},
	}}
	rightIter := &sliceSpansetIter{spansets: []*Spanset{
		{TraceID: traceID1, Spans: []Span{dbChild, dbOrphan}},
		{TraceID: traceID2, Spans: []Span{dbChild2}},
	}}

	iter := StructuralOpSpansetIter(StructuralOpDescendant, leftIter, rightIter)
	defer iter.Close()

	var twoScanResult []*Spanset
	for {
		ss, err := iter.Next(context.Background())
		require.NoError(t, err)
		if ss == nil {
			break
		}
		twoScanResult = append(twoScanResult, ss)
	}

	t.Logf("StructuralOpSpansetIter: %d spanset(s)", len(twoScanResult))
	for i, ss := range twoScanResult {
		t.Logf("  [%d] traceID=%v  spans=%d", i, ss.TraceID, len(ss.Spans))
		for j, s := range ss.Spans {
			t.Logf("    span[%d] id=%v", j, s.ID())
		}
	}

	require.Len(t, twoScanResult, 1, "only trace1 should match")
	require.Len(t, twoScanResult[0].Spans, 1, "only dbChild survives")
	require.Equal(t, dbChild.id, twoScanResult[0].Spans[0].ID())
}

// TestStructuralOp_MismatchedOrder documents the merge-join ordering requirement.
//
// StructuralOpSpansetIter is a merge-join: when left.TraceID < right.TraceID it
// discards the left spanset and advances; when left > right it discards right.
// This only produces correct results when BOTH iterators emit spansets in the
// same sorted order (both ascending or both descending).
//
// If the orders differ the join silently drops matches:
//
//	left  (reversed):  [trace2, trace1]
//	right (normal):    [trace1, trace2]
//
//	step 1: left=trace2, right=trace1 → cmp>0 → discard right(trace1!), keep left pending
//	step 2: left=trace2(pending), right=trace2 → cmp==0 → evaluate trace2 (no match)
//	step 3: left exhausted → done
//
//	trace1 — the only real match — was silently discarded.
func TestStructuralOp_MismatchedOrder(t *testing.T) {
	traceID1 := []byte{1}
	traceID2 := []byte{2}

	// Trace 1: GET parent → postgresql child (structural match).
	httpSpan1 := newMockSpan([]byte{1, 1}).
		WithSpanString("http.method", "GET").
		WithNestedSetInfo(0, 1, 6)
	dbChild1 := newMockSpan([]byte{1, 2}).
		WithSpanString("db.system", "postgresql").
		WithNestedSetInfo(1, 2, 5) // descendant: 2>1 && 2<6 ✓

	// Trace 2: GET parent → postgresql child (also a structural match).
	httpSpan2 := newMockSpan([]byte{2, 1}).
		WithSpanString("http.method", "GET").
		WithNestedSetInfo(0, 1, 6)
	dbChild2 := newMockSpan([]byte{2, 2}).
		WithSpanString("db.system", "postgresql").
		WithNestedSetInfo(1, 2, 5) // descendant ✓

	t.Run("same order (ascending) — both matches found", func(t *testing.T) {
		left := &sliceSpansetIter{spansets: []*Spanset{
			{TraceID: traceID1, Spans: []Span{httpSpan1}},
			{TraceID: traceID2, Spans: []Span{httpSpan2}},
		}}
		right := &sliceSpansetIter{spansets: []*Spanset{
			{TraceID: traceID1, Spans: []Span{dbChild1}},
			{TraceID: traceID2, Spans: []Span{dbChild2}},
		}}
		results := drainIter(t, StructuralOpSpansetIter(StructuralOpDescendant, left, right))
		require.Len(t, results, 2, "both traces match")
	})

	t.Run("mismatched order — match silently dropped", func(t *testing.T) {
		// Left emits trace2 first, right emits trace1 first.
		//
		// merge-join step 1: left=trace2, right=trace1 → cmp>0
		//   → right(trace1) discarded, left(trace2) held as pending
		// merge-join step 2: left=trace2(pending), right=trace2 → cmp==0
		//   → evaluate trace2, return match
		// merge-join step 3: left=trace1, right=nil → right exhausted → done
		//
		// trace1 is never joined: its left side arrives after right is exhausted.
		left := &sliceSpansetIter{spansets: []*Spanset{
			{TraceID: traceID2, Spans: []Span{httpSpan2}}, // trace2 first
			{TraceID: traceID1, Spans: []Span{httpSpan1}}, // trace1 second
		}}
		right := &sliceSpansetIter{spansets: []*Spanset{
			{TraceID: traceID1, Spans: []Span{dbChild1}}, // trace1 first
			{TraceID: traceID2, Spans: []Span{dbChild2}}, // trace2 second
		}}
		results := drainIter(t, StructuralOpSpansetIter(StructuralOpDescendant, left, right))
		// Only trace2 is returned; trace1 was silently dropped.
		require.Len(t, results, 1, "mismatched order drops trace1")
		require.Equal(t, traceID2, results[0].TraceID)
	})
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
