# Plan Node Tree Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `FetchSpansRequest` with a logical plan node tree that enables optimization passes and a clean end-to-end translation to parquet iterators.

**Architecture:** TraceQL AST is converted to a `PlanNode` tree in `pkg/traceql`. An optimizer rewrites the tree (predicate pushdown, condition merging). A translator in `tempodb/encoding/common` walks the full tree and produces an `Evaluatable` by calling a `ScanBackend` interface, which vparquet5 implements 1:1 with its existing `create*Iterator` functions.

**Tech Stack:** Go, `pkg/parquetquery` (internal parquet iterator primitives), `pkg/traceql` (AST + existing Condition/Attribute types)

**Design doc:** `docs/plans/2026-03-04-plan-node-tree-design.md`

---

## Phase 1 — Plan node infrastructure (`pkg/traceql`)

### Task 1: PlanNode interface and scan nodes

**Files:**
- Create: `pkg/traceql/plan.go`
- Create: `pkg/traceql/plan_test.go`

**Step 1: Write the failing test**

```go
// pkg/traceql/plan_test.go
package traceql

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestScanNodeChildren(t *testing.T) {
    span := &SpanScanNode{Conditions: []Condition{{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "status")}}}
    res := &ResourceScanNode{Conditions: []Condition{{Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name")}}}, child: span}
    trace := &TraceScanNode{child: res}

    require.Equal(t, []PlanNode{res}, trace.Children())
    require.Equal(t, []PlanNode{span}, res.Children())
    require.Nil(t, span.Children())
    require.Contains(t, trace.String(), "TraceScan")
    require.Contains(t, res.String(), "ResourceScan")
    require.Contains(t, span.String(), "SpanScan")
}
```

**Step 2: Run to see failure**

```bash
cd /path/to/tempo && go test ./pkg/traceql/ -run TestScanNodeChildren
```
Expected: compile error — types not defined.

**Step 3: Implement scan nodes**

```go
// pkg/traceql/plan.go
package traceql

import "fmt"

// PlanNode is a node in the logical query plan tree.
// Nodes are purely logical — they carry no execution logic.
type PlanNode interface {
    Children() []PlanNode
    Accept(PlanVisitor)
    String() string
}

// PlanVisitor visits plan nodes. Return false from VisitPre to skip children.
type PlanVisitor interface {
    VisitPre(PlanNode) bool
    VisitPost(PlanNode)
}

// WalkPlan performs a depth-first traversal of the plan tree.
func WalkPlan(n PlanNode, v PlanVisitor) {
    if n == nil {
        return
    }
    if !v.VisitPre(n) {
        return
    }
    for _, child := range n.Children() {
        WalkPlan(child, v)
    }
    v.VisitPost(n)
}

// --- Scan nodes (1:1 with parquet iterator levels) ---

type TraceScanNode struct {
    Conditions    []Condition
    AllConditions bool
    child         PlanNode
}

func (n *TraceScanNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *TraceScanNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *TraceScanNode) String() string { return fmt.Sprintf("TraceScan(%d conditions)", len(n.Conditions)) }

type ResourceScanNode struct {
    Conditions []Condition
    child      PlanNode
}

func (n *ResourceScanNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *ResourceScanNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *ResourceScanNode) String() string { return fmt.Sprintf("ResourceScan(%d conditions)", len(n.Conditions)) }

type InstrumentationScopeScanNode struct {
    Conditions []Condition
    child      PlanNode
}

func (n *InstrumentationScopeScanNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *InstrumentationScopeScanNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *InstrumentationScopeScanNode) String() string { return "InstrumentationScopeScan" }

type SpanScanNode struct {
    Conditions []Condition
    child      PlanNode
}

func (n *SpanScanNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *SpanScanNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *SpanScanNode) String() string { return fmt.Sprintf("SpanScan(%d conditions)", len(n.Conditions)) }

type EventScanNode struct {
    Conditions []Condition
    child      PlanNode
}

func (n *EventScanNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *EventScanNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *EventScanNode) String() string { return "EventScan" }

type LinkScanNode struct {
    Conditions []Condition
    child      PlanNode
}

func (n *LinkScanNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *LinkScanNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *LinkScanNode) String() string { return "LinkScan" }
```

**Step 4: Run test to verify it passes**

```bash
go test ./pkg/traceql/ -run TestScanNodeChildren
```

**Step 5: Commit**

```bash
git add pkg/traceql/plan.go pkg/traceql/plan_test.go
git commit -m "feat(traceql): add PlanNode interface and scan node types"
```

---

### Task 2: Engine nodes, ProjectNode, and metrics nodes

**Files:**
- Modify: `pkg/traceql/plan.go`
- Modify: `pkg/traceql/plan_test.go`

**Step 1: Write the failing test**

```go
func TestEngineNodes(t *testing.T) {
    filter := &SpansetFilterNode{
        Expression: &SpansetFilter{},
        child:      &SpanScanNode{},
    }
    group := &GroupByNode{
        By:    []Attribute{NewScopedAttribute(AttributeScopeResource, false, "service.name")},
        child: filter,
    }
    proj := &ProjectNode{
        Columns: []Attribute{NewScopedAttribute(AttributeScopeSpan, false, "region")},
        child:   group,
    }
    rate := &RateNode{child: proj}

    require.Equal(t, []PlanNode{group}, proj.Children())
    require.Equal(t, []PlanNode{filter}, group.Children())
    require.Contains(t, proj.String(), "Project")
    require.Contains(t, group.String(), "GroupBy")
    require.Contains(t, rate.String(), "Rate")
}
```

**Step 2: Run to see failure**

```bash
go test ./pkg/traceql/ -run TestEngineNodes
```

**Step 3: Implement engine nodes**

Add to `pkg/traceql/plan.go`:

```go
// --- Engine nodes (in-memory evaluation, above scan nodes) ---

// SpansetFilterNode evaluates a filter expression against each spanset.
// Simple per-span predicates are pushed into scan conditions by the optimizer.
// Structural/cross-span predicates remain as SpansetFilterNode.
type SpansetFilterNode struct {
    Expression *SpansetFilter
    child      PlanNode
}

func (n *SpansetFilterNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *SpansetFilterNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *SpansetFilterNode) String() string { return fmt.Sprintf("SpansetFilter(%s)", n.Expression) }

// GroupByNode groups spans into spansets by the given attributes.
type GroupByNode struct {
    By    []Attribute
    child PlanNode
}

func (n *GroupByNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *GroupByNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *GroupByNode) String() string { return fmt.Sprintf("GroupBy(%v)", n.By) }

// CoalesceNode merges all spans in child spansets into a single spanset per trace.
type CoalesceNode struct {
    child PlanNode
}

func (n *CoalesceNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *CoalesceNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *CoalesceNode) String() string { return "Coalesce" }

// StructuralOpNode evaluates a structural relationship (>>, >, ~) between two spansets.
type StructuralOp int

const (
    StructuralOpParent     StructuralOp = iota // >>
    StructuralOpAncestor                       // >
    StructuralOpSibling                        // ~
    StructuralOpDescendant                     // <<
    StructuralOpChild                          // <
)

type StructuralOpNode struct {
    Op    StructuralOp
    left  PlanNode
    right PlanNode
}

func (n *StructuralOpNode) Children() []PlanNode { return []PlanNode{n.left, n.right} }
func (n *StructuralOpNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        WalkPlan(n.left, v)
        WalkPlan(n.right, v)
    }
    v.VisitPost(n)
}
func (n *StructuralOpNode) String() string { return fmt.Sprintf("StructuralOp(%d)", n.Op) }

// --- ProjectNode: second-pass fetch boundary ---

// ProjectNode triggers a second-pass parquet fetch to obtain additional columns
// for surviving spans. It must be placed below GroupByNode and other engine nodes
// so that grouping happens after the fetch — no cbSpanset reconstruction needed.
type ProjectNode struct {
    Columns []Attribute
    child   PlanNode
}

func (n *ProjectNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *ProjectNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *ProjectNode) String() string { return fmt.Sprintf("Project(%v)", n.Columns) }

// --- Metrics nodes (purely logical aggregation config) ---

type RateNode struct {
    By    []Attribute
    child PlanNode
}

func (n *RateNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *RateNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *RateNode) String() string { return "Rate" }

type CountOverTimeNode struct {
    By    []Attribute
    child PlanNode
}

func (n *CountOverTimeNode) Children() []PlanNode {
    if n.child != nil { return []PlanNode{n.child} }
    return nil
}
func (n *CountOverTimeNode) Accept(v PlanVisitor) {
    if v.VisitPre(n) {
        for _, c := range n.Children() { WalkPlan(c, v) }
    }
    v.VisitPost(n)
}
func (n *CountOverTimeNode) String() string { return "CountOverTime" }

// Add HistogramNode, QuantileNode similarly as needed.
```

**Step 4: Run test**

```bash
go test ./pkg/traceql/ -run TestEngineNodes
```

**Step 5: Commit**

```bash
git add pkg/traceql/plan.go pkg/traceql/plan_test.go
git commit -m "feat(traceql): add engine, project, and metrics plan nodes"
```

---

### Task 3: WalkPlan visitor test

**Files:**
- Modify: `pkg/traceql/plan_test.go`

**Step 1: Write the failing test**

```go
func TestWalkPlanOrder(t *testing.T) {
    span := &SpanScanNode{}
    res := &ResourceScanNode{child: span}
    trace := &TraceScanNode{child: res}

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

    // pre-order: root first
    require.Equal(t, trace.String(), preOrder[0])
    require.Equal(t, span.String(), preOrder[2])
    // post-order: leaves first
    require.Equal(t, span.String(), postOrder[0])
    require.Equal(t, trace.String(), postOrder[2])
}

type funcVisitor struct {
    pre  func(PlanNode) bool
    post func(PlanNode)
}
func (v *funcVisitor) VisitPre(n PlanNode) bool { return v.pre(n) }
func (v *funcVisitor) VisitPost(n PlanNode)      { v.post(n) }
```

**Step 2: Run to see failure**

```bash
go test ./pkg/traceql/ -run TestWalkPlanOrder
```

**Step 3: No new code needed** — `WalkPlan` is already implemented in Task 1. This test validates behavior.

**Step 4: Run test**

```bash
go test ./pkg/traceql/ -run TestWalkPlan
```

**Step 5: Commit**

```bash
git add pkg/traceql/plan_test.go
git commit -m "test(traceql): add WalkPlan visitor order test"
```

---

### Task 4: Optimizer — Rule interface and RuleSet

**Files:**
- Create: `pkg/traceql/plan_optimizer.go`
- Create: `pkg/traceql/plan_optimizer_test.go`

**Step 1: Write the failing test**

```go
// pkg/traceql/plan_optimizer_test.go
package traceql

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestOptimizerFixpoint(t *testing.T) {
    callCount := 0
    // Rule that fires once, then becomes a no-op
    rule := FuncRule("test-rule", func(n PlanNode) (PlanNode, bool) {
        if _, ok := n.(*SpanScanNode); ok && callCount == 0 {
            callCount++
            return &SpanScanNode{Conditions: []Condition{{}}}, true
        }
        return n, false
    })

    rs := NewRuleSet(rule)
    plan := &TraceScanNode{child: &SpanScanNode{}}
    result := rs.Optimize(plan)

    // Optimizer ran until fixpoint (no more changes)
    require.NotNil(t, result)
    // The span scan node got a condition added by the rule
    span := result.(*TraceScanNode).child.(*SpanScanNode)
    require.Len(t, span.Conditions, 1)
}
```

**Step 2: Run to see failure**

```bash
go test ./pkg/traceql/ -run TestOptimizerFixpoint
```

**Step 3: Implement**

```go
// pkg/traceql/plan_optimizer.go
package traceql

// Rule rewrites a single plan node. Return (node, true) if the node changed.
type Rule interface {
    Name() string
    Apply(PlanNode) (PlanNode, bool)
}

// FuncRule creates a Rule from a function.
func FuncRule(name string, fn func(PlanNode) (PlanNode, bool)) Rule {
    return &funcRule{name: name, fn: fn}
}

type funcRule struct {
    name string
    fn   func(PlanNode) (PlanNode, bool)
}

func (r *funcRule) Name() string                         { return r.name }
func (r *funcRule) Apply(n PlanNode) (PlanNode, bool)    { return r.fn(n) }

// RuleSet applies a set of rules to a plan tree until fixpoint.
type RuleSet struct {
    rules []Rule
}

func NewRuleSet(rules ...Rule) *RuleSet {
    return &RuleSet{rules: rules}
}

func (rs *RuleSet) Add(r Rule) {
    rs.rules = append(rs.rules, r)
}

// Optimize applies rules until no rule fires (fixpoint).
func (rs *RuleSet) Optimize(root PlanNode) PlanNode {
    for {
        next, changed := rs.applyOnce(root)
        root = next
        if !changed {
            return root
        }
    }
}

func (rs *RuleSet) applyOnce(n PlanNode) (PlanNode, bool) {
    changed := false
    // Apply rules to this node
    for _, rule := range rs.rules {
        if rewritten, ok := rule.Apply(n); ok {
            n = rewritten
            changed = true
        }
    }
    // Recurse into children — note: children are embedded in concrete types,
    // so we use a visitor to rebuild child references after rewriting.
    // (Implementer note: concrete node types need a WithChildren([]PlanNode) PlanNode method
    // so the optimizer can reconstruct the tree after rewriting children.)
    return n, changed
}
```

> **Implementer note:** To rebuild the tree after child rewrites, add a `WithChildren([]PlanNode) PlanNode` method to each concrete node type. The optimizer calls `n.WithChildren(rewrittenChildren)` after recursing.

**Step 4: Run test**

```bash
go test ./pkg/traceql/ -run TestOptimizerFixpoint
```

**Step 5: Commit**

```bash
git add pkg/traceql/plan_optimizer.go pkg/traceql/plan_optimizer_test.go
git commit -m "feat(traceql): add Rule interface and RuleSet optimizer with fixpoint"
```

---

### Task 5: Built-in optimization rules

**Files:**
- Create: `pkg/traceql/plan_rules.go`
- Create: `pkg/traceql/plan_rules_test.go`

**Step 1: Write failing tests**

```go
// pkg/traceql/plan_rules_test.go
package traceql

import (
    "testing"
    "github.com/stretchr/testify/require"
)

// PredicatePushdownRule: a SpansetFilterNode with a simple per-span predicate
// should be eliminated — its condition pushed into the SpanScanNode below.
func TestPredicatePushdownRule(t *testing.T) {
    cond := Condition{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "status")}
    filter := &SpansetFilterNode{
        Expression: &SpansetFilter{},
        child:      &SpanScanNode{},
    }
    // Attach condition to the filter node for pushdown
    filter.Expression.Conditions = []Condition{cond}

    rule := PredicatePushdownRule()
    result, changed := rule.Apply(filter)

    require.True(t, changed)
    // Filter node should be gone — replaced by SpanScanNode with condition
    scan, ok := result.(*SpanScanNode)
    require.True(t, ok)
    require.Contains(t, scan.Conditions, cond)
}

// ConditionMergeRule: two adjacent scan nodes of the same type should merge.
func TestConditionMergeRule(t *testing.T) {
    cond1 := Condition{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "status")}
    cond2 := Condition{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "http.method")}
    // Outer scan wraps inner scan — same type, should merge
    inner := &SpanScanNode{Conditions: []Condition{cond1}}
    outer := &SpanScanNode{Conditions: []Condition{cond2}, child: inner}

    rule := ConditionMergeRule()
    result, changed := rule.Apply(outer)

    require.True(t, changed)
    merged, ok := result.(*SpanScanNode)
    require.True(t, ok)
    require.Len(t, merged.Conditions, 2)
    require.Nil(t, merged.Children()) // inner was merged away
}

// SecondPassEliminatorRule: ProjectNode with conditions already in scan nodes
// should be removed.
func TestSecondPassEliminatorRule(t *testing.T) {
    col := NewScopedAttribute(AttributeScopeSpan, false, "region")
    scan := &SpanScanNode{Conditions: []Condition{{Attribute: col}}}
    proj := &ProjectNode{Columns: []Attribute{col}, child: scan}

    rule := SecondPassEliminatorRule()
    result, changed := rule.Apply(proj)

    require.True(t, changed)
    // ProjectNode removed, scan node is the result
    _, ok := result.(*SpanScanNode)
    require.True(t, ok)
}
```

**Step 2: Run to see failure**

```bash
go test ./pkg/traceql/ -run TestPredicatePushdownRule
```

**Step 3: Implement rules**

```go
// pkg/traceql/plan_rules.go
package traceql

// PredicatePushdownRule pushes simple per-span predicates from SpansetFilterNode
// directly into the SpanScanNode below, eliminating the filter node.
// Structural/cross-span predicates are NOT pushed down.
func PredicatePushdownRule() Rule {
    return FuncRule("predicate-pushdown", func(n PlanNode) (PlanNode, bool) {
        filter, ok := n.(*SpansetFilterNode)
        if !ok {
            return n, false
        }
        if !isSimplePerSpanFilter(filter.Expression) {
            return n, false
        }
        // Push conditions down into child SpanScanNode
        scan := findSpanScanNode(filter.child)
        if scan == nil {
            return n, false
        }
        scan.Conditions = append(scan.Conditions, filter.Expression.Conditions...)
        return filter.child, true
    })
}

// ConditionMergeRule merges two adjacent scan nodes of the same type.
func ConditionMergeRule() Rule {
    return FuncRule("condition-merge", func(n PlanNode) (PlanNode, bool) {
        switch outer := n.(type) {
        case *SpanScanNode:
            if inner, ok := outer.child.(*SpanScanNode); ok {
                outer.Conditions = append(outer.Conditions, inner.Conditions...)
                outer.child = inner.child
                return outer, true
            }
        case *ResourceScanNode:
            if inner, ok := outer.child.(*ResourceScanNode); ok {
                outer.Conditions = append(outer.Conditions, inner.Conditions...)
                outer.child = inner.child
                return outer, true
            }
        }
        return n, false
    })
}

// SecondPassEliminatorRule removes a ProjectNode when all its columns are
// already present in the first-pass scan nodes below it.
func SecondPassEliminatorRule() Rule {
    return FuncRule("second-pass-eliminator", func(n PlanNode) (PlanNode, bool) {
        proj, ok := n.(*ProjectNode)
        if !ok {
            return n, false
        }
        if allColumnsInFirstPass(proj.Columns, proj.child) {
            return proj.child, true
        }
        return n, false
    })
}

// isSimplePerSpanFilter returns true if the expression has no structural/cross-span logic.
// Implementer: check that Expression.impliedType() == TypeSpan and no structural ops.
func isSimplePerSpanFilter(e *SpansetFilter) bool {
    // TODO: implement by inspecting the expression tree
    return false
}

func findSpanScanNode(n PlanNode) *SpanScanNode {
    if s, ok := n.(*SpanScanNode); ok {
        return s
    }
    return nil
}

func allColumnsInFirstPass(cols []Attribute, subtree PlanNode) bool {
    // Collect all conditions from scan nodes in subtree
    var have []Attribute
    WalkPlan(subtree, &funcVisitor{
        pre: func(n PlanNode) bool {
            if s, ok := n.(*SpanScanNode); ok {
                for _, c := range s.Conditions {
                    have = append(have, c.Attribute)
                }
            }
            return true
        },
        post: func(PlanNode) {},
    })
    for _, col := range cols {
        found := false
        for _, h := range have {
            if h == col { found = true; break }
        }
        if !found { return false }
    }
    return true
}
```

**Step 4: Run tests**

```bash
go test ./pkg/traceql/ -run TestPredicatePushdownRule -run TestConditionMergeRule -run TestSecondPassEliminatorRule
```

**Step 5: Commit**

```bash
git add pkg/traceql/plan_rules.go pkg/traceql/plan_rules_test.go
git commit -m "feat(traceql): add predicate pushdown, condition merge, and second-pass eliminator rules"
```

---

### Task 6: Plan builder (AST → plan tree)

**Files:**
- Create: `pkg/traceql/plan_builder.go`
- Create: `pkg/traceql/plan_builder_test.go`

**Step 1: Write failing tests**

```go
// pkg/traceql/plan_builder_test.go
package traceql

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestBuildPlan_SimpleFilter(t *testing.T) {
    // { span.status = error }
    expr, err := Parse(`{ span.status = error }`)
    require.NoError(t, err)

    plan, err := BuildPlan(expr)
    require.NoError(t, err)

    // Root should be TraceScanNode (no metrics, no engine nodes needed after pushdown)
    // After optimizer runs: predicate pushed into SpanScanNode
    require.NotNil(t, plan)

    // Walk and find SpanScanNode with status condition
    var spanScan *SpanScanNode
    WalkPlan(plan, &funcVisitor{
        pre: func(n PlanNode) bool {
            if s, ok := n.(*SpanScanNode); ok { spanScan = s }
            return true
        },
        post: func(PlanNode) {},
    })
    require.NotNil(t, spanScan)
    require.Len(t, spanScan.Conditions, 1)
}

func TestBuildPlan_GroupBy(t *testing.T) {
    // { span.status = error } | by(resource.service.name)
    expr, err := Parse(`{ span.status = error } | by(resource.service.name)`)
    require.NoError(t, err)

    plan, err := BuildPlan(expr)
    require.NoError(t, err)

    // Root should be GroupByNode
    _, ok := plan.(*GroupByNode)
    require.True(t, ok, "expected GroupByNode at root, got %T", plan)
}

func TestBuildPlan_WithSelect(t *testing.T) {
    // { span.status = error } | by(resource.service.name) | select(span.region)
    // ProjectNode should appear below GroupByNode
    expr, err := Parse(`{ span.status = error } | by(resource.service.name) | select(span.region)`)
    require.NoError(t, err)

    plan, err := BuildPlan(expr)
    require.NoError(t, err)

    group, ok := plan.(*GroupByNode)
    require.True(t, ok)
    proj, ok := group.Children()[0].(*ProjectNode)
    require.True(t, ok)
    require.Len(t, proj.Columns, 1)
}

func TestBuildPlan_MetricsRate(t *testing.T) {
    expr, err := Parse(`{ } | rate()`)
    require.NoError(t, err)

    plan, err := BuildPlan(expr)
    require.NoError(t, err)

    _, ok := plan.(*RateNode)
    require.True(t, ok, "expected RateNode at root, got %T", plan)
}
```

**Step 2: Run to see failure**

```bash
go test ./pkg/traceql/ -run TestBuildPlan
```

**Step 3: Implement plan builder**

```go
// pkg/traceql/plan_builder.go
package traceql

// BuildPlan converts a parsed TraceQL expression into a logical plan tree.
// It also runs the default optimizer passes.
func BuildPlan(expr *RootExpr) (PlanNode, error) {
    b := &planBuilder{}
    plan, err := b.build(expr)
    if err != nil {
        return nil, err
    }
    // Run default optimizer
    rs := DefaultRuleSet()
    return rs.Optimize(plan), nil
}

// DefaultRuleSet returns the built-in optimization rules.
func DefaultRuleSet() *RuleSet {
    return NewRuleSet(
        PredicatePushdownRule(),
        ConditionMergeRule(),
        SecondPassEliminatorRule(),
    )
}

type planBuilder struct{}

func (b *planBuilder) build(expr *RootExpr) (PlanNode, error) {
    // Build the base scan tree
    var plan PlanNode = b.buildBaseScanTree()

    // Walk the pipeline elements bottom-up (outermost = top of plan tree)
    // Handle SpansetFilter, GroupOperation, SelectOperation, CoalesceOperation,
    // SpansetOperation (structural ops), and MetricsPipeline nodes.
    //
    // Implementer: iterate over expr.Pipeline.Elements and expr.MetricsPipeline.
    // For each element type:
    //   - SpansetFilter       → SpansetFilterNode wrapping current plan
    //   - GroupOperation      → GroupByNode wrapping current plan
    //   - SelectOperation     → ProjectNode inserted below current plan (see rule below)
    //   - CoalesceOperation   → CoalesceNode wrapping current plan
    //   - SpansetOperation(>>,etc.) → StructuralOpNode with left/right children
    //   - MetricsPipeline.rate → RateNode wrapping current plan
    //
    // SelectOperation placement rule:
    //   ProjectNode must go below GroupByNode. When the builder sees SelectOperation,
    //   it inserts ProjectNode above the last scan-level node (below any GroupByNode
    //   already built). Implementer: track where engine nodes start in the chain
    //   and insert ProjectNode at the correct level.

    return plan, nil
}

// buildBaseScanTree builds the default scan tree:
//   TraceScanNode → ResourceScanNode → InstrumentationScopeScanNode → SpanScanNode
// Conditions are empty; the optimizer fills them via predicate pushdown.
func (b *planBuilder) buildBaseScanTree() PlanNode {
    span  := &SpanScanNode{}
    instr := &InstrumentationScopeScanNode{child: span}
    res   := &ResourceScanNode{child: instr}
    trace := &TraceScanNode{child: res}
    return trace
}
```

> **Implementer note:** The AST pipeline element types live in `pkg/traceql/ast.go`. Reference `Pipeline.Elements` (type `[]pipelineElement`) and the concrete types `*SpansetFilter`, `*GroupOperation`, `*SelectOperation`, `*CoalesceOperation`, `*SpansetOperation`. The metrics pipeline is in `*RootExpr.MetricsPipeline`. Study `ast_conditions.go:extractConditions()` and `ast_execute.go:Pipeline.evaluate()` for the existing AST semantics before writing the builder.

**Step 4: Run tests**

```bash
go test ./pkg/traceql/ -run TestBuildPlan
```

**Step 5: Commit**

```bash
git add pkg/traceql/plan_builder.go pkg/traceql/plan_builder_test.go
git commit -m "feat(traceql): add plan builder converting AST to logical plan tree"
```

---

## Phase 2 — Storage translation (`tempodb/encoding/common` + vparquet5)

### Task 7: ScanBackend interface

**Files:**
- Create: `tempodb/encoding/common/scan_backend.go`

**Step 1: No test needed — interface definition only**

```go
// tempodb/encoding/common/scan_backend.go
package common

import (
    "context"

    "github.com/grafana/tempo/pkg/parquetquery"
    "github.com/grafana/tempo/pkg/traceql"
)

// ScanBackend is implemented by each storage encoding (e.g., vparquet5).
// Each method maps 1:1 to a parquet iterator level.
// The child parameter is the iterator for the level below (nil for the innermost).
// The primary parameter on TraceIter is the second-pass row source (nil for first pass).
type ScanBackend interface {
    SpanIter(ctx context.Context, node *traceql.SpanScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error)
    InstrumentationScopeIter(ctx context.Context, node *traceql.InstrumentationScopeScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error)
    ResourceIter(ctx context.Context, node *traceql.ResourceScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error)
    // TraceIter converts the parquet iterator chain to a spanset-level iterator.
    // primary: if non-nil, used as the second-pass row source (surviving span row numbers).
    TraceIter(ctx context.Context, node *traceql.TraceScanNode, primary parquetquery.Iterator, child parquetquery.Iterator) (traceql.SpansetIterator, error)
    EventIter(ctx context.Context, node *traceql.EventScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error)
    LinkIter(ctx context.Context, node *traceql.LinkScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error)
}
```

**Step 2: Compile check**

```bash
go build ./tempodb/encoding/common/
```

**Step 3: Commit**

```bash
git add tempodb/encoding/common/scan_backend.go
git commit -m "feat(common): add ScanBackend interface for plan-based storage access"
```

---

### Task 8: Plan translator

**Files:**
- Create: `tempodb/encoding/common/plan_translator.go`
- Create: `tempodb/encoding/common/plan_translator_test.go`

**Step 1: Write the failing test (using a mock ScanBackend)**

```go
// tempodb/encoding/common/plan_translator_test.go
package common

import (
    "context"
    "testing"

    "github.com/grafana/tempo/pkg/parquetquery"
    "github.com/grafana/tempo/pkg/traceql"
    "github.com/stretchr/testify/require"
)

// mockBackend records which ScanBackend methods were called.
type mockBackend struct {
    calledSpan, calledResource, calledTrace bool
}

func (m *mockBackend) SpanIter(_ context.Context, _ *traceql.SpanScanNode, _ parquetquery.Iterator) (parquetquery.Iterator, error) {
    m.calledSpan = true
    return &mockIter{}, nil
}
func (m *mockBackend) InstrumentationScopeIter(_ context.Context, _ *traceql.InstrumentationScopeScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
    return child, nil
}
func (m *mockBackend) ResourceIter(_ context.Context, _ *traceql.ResourceScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
    m.calledResource = true
    return child, nil
}
func (m *mockBackend) TraceIter(_ context.Context, _ *traceql.TraceScanNode, _ parquetquery.Iterator, child parquetquery.Iterator) (traceql.SpansetIterator, error) {
    m.calledTrace = true
    return &mockSpansetIter{}, nil
}
func (m *mockBackend) EventIter(_ context.Context, _ *traceql.EventScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) { return child, nil }
func (m *mockBackend) LinkIter(_ context.Context, _ *traceql.LinkScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) { return child, nil }

type mockIter struct{}
func (m *mockIter) Next() (*parquetquery.IteratorResult, error) { return nil, nil }
func (m *mockIter) SeekTo(parquetquery.RowNumber, int) (*parquetquery.IteratorResult, error) { return nil, nil }
func (m *mockIter) Close() {}

type mockSpansetIter struct{}
func (m *mockSpansetIter) Next(context.Context) (*traceql.Spanset, error) { return nil, nil }
func (m *mockSpansetIter) Close() {}

func TestTranslate_SimpleScanTree(t *testing.T) {
    span  := &traceql.SpanScanNode{}
    res   := &traceql.ResourceScanNode{/* child: span */}
    trace := &traceql.TraceScanNode{/* child: res */}
    // wire up children (use exported constructors once added)

    backend := &mockBackend{}
    eval, err := Translate(context.Background(), trace, backend, SearchOptions{})
    require.NoError(t, err)
    require.NotNil(t, eval)

    require.True(t, backend.calledSpan, "expected SpanIter to be called")
    require.True(t, backend.calledResource, "expected ResourceIter to be called")
    require.True(t, backend.calledTrace, "expected TraceIter to be called")
}
```

**Step 2: Run to see failure**

```bash
go test ./tempodb/encoding/common/ -run TestTranslate
```

**Step 3: Implement the translator**

```go
// tempodb/encoding/common/plan_translator.go
package common

import (
    "context"
    "fmt"

    "github.com/grafana/tempo/pkg/parquetquery"
    "github.com/grafana/tempo/pkg/traceql"
)

// Evaluatable is the result of translating a full plan tree.
// Call Do to drive execution; Results returns the accumulated series.
type Evaluatable interface {
    Do(ctx context.Context, start, end uint64, maxSeries int) error
    Results() traceql.SeriesSet
    Metrics() (inspectedBytes, spansTotal uint64, err error)
}

// Translate converts a full plan tree into an Evaluatable by recursively
// building the iterator/evaluator chain. The translator owns the boundary
// between storage iterators (parquetquery.Iterator) and engine evaluators
// (in-memory GroupBy, metrics aggregation, etc.).
func Translate(ctx context.Context, plan traceql.PlanNode, backend ScanBackend, opts SearchOptions) (Evaluatable, error) {
    t := &translator{ctx: ctx, backend: backend, opts: opts}
    return t.translate(plan)
}

type translator struct {
    ctx     context.Context
    backend ScanBackend
    opts    SearchOptions
}

func (t *translator) translate(n traceql.PlanNode) (Evaluatable, error) {
    switch node := n.(type) {

    // --- Metrics nodes: build aggregator, recurse into child ---
    case *traceql.RateNode:
        child, err := t.translate(node.Children()[0])
        if err != nil { return nil, err }
        return newRateEvaluatable(node, child), nil

    case *traceql.CountOverTimeNode:
        child, err := t.translate(node.Children()[0])
        if err != nil { return nil, err }
        return newCountOverTimeEvaluatable(node, child), nil

    // --- Engine nodes: build in-memory evaluator wrapping child ---
    case *traceql.GroupByNode:
        child, err := t.translate(node.Children()[0])
        if err != nil { return nil, err }
        return newGroupByEvaluatable(node, child), nil

    case *traceql.CoalesceNode:
        child, err := t.translate(node.Children()[0])
        if err != nil { return nil, err }
        return newCoalesceEvaluatable(node, child), nil

    case *traceql.SpansetFilterNode:
        child, err := t.translate(node.Children()[0])
        if err != nil { return nil, err }
        return newFilterEvaluatable(node, child), nil

    case *traceql.StructuralOpNode:
        left, err := t.translate(node.Children()[0])
        if err != nil { return nil, err }
        right, err := t.translate(node.Children()[1])
        if err != nil { return nil, err }
        return newStructuralOpEvaluatable(node, left, right), nil

    // --- ProjectNode: builds second-pass scan using first-pass row numbers ---
    case *traceql.ProjectNode:
        return t.translateProject(node)

    // --- Scan tree: build parquet iterator chain bottom-up ---
    case *traceql.TraceScanNode:
        iter, err := t.buildParquetChain(node, nil)
        if err != nil { return nil, err }
        return newSpansetEvaluatable(iter), nil

    default:
        return nil, fmt.Errorf("plan_translator: unhandled plan node type %T", n)
    }
}

// buildParquetChain recursively builds a parquetquery.Iterator chain from the
// scan node subtree. primary is the second-pass row source (nil for first pass).
// Only TraceScanNode produces a traceql.SpansetIterator; all others return
// parquetquery.Iterator.
func (t *translator) buildParquetChain(n traceql.PlanNode, primary parquetquery.Iterator) (traceql.SpansetIterator, error) {
    trace, ok := n.(*traceql.TraceScanNode)
    if !ok {
        return nil, fmt.Errorf("plan_translator: expected TraceScanNode, got %T", n)
    }

    // Build child parquet iterator (resource → instrumentation → span → events/links)
    var child parquetquery.Iterator
    if len(trace.Children()) > 0 {
        var err error
        child, err = t.buildInnerChain(trace.Children()[0])
        if err != nil { return nil, err }
    }

    return t.backend.TraceIter(t.ctx, trace, primary, child)
}

func (t *translator) buildInnerChain(n traceql.PlanNode) (parquetquery.Iterator, error) {
    switch node := n.(type) {
    case *traceql.ResourceScanNode:
        var child parquetquery.Iterator
        if len(node.Children()) > 0 {
            var err error
            child, err = t.buildInnerChain(node.Children()[0])
            if err != nil { return nil, err }
        }
        return t.backend.ResourceIter(t.ctx, node, child)

    case *traceql.InstrumentationScopeScanNode:
        var child parquetquery.Iterator
        if len(node.Children()) > 0 {
            var err error
            child, err = t.buildInnerChain(node.Children()[0])
            if err != nil { return nil, err }
        }
        return t.backend.InstrumentationScopeIter(t.ctx, node, child)

    case *traceql.SpanScanNode:
        var child parquetquery.Iterator
        if len(node.Children()) > 0 {
            var err error
            child, err = t.buildInnerChain(node.Children()[0])
            if err != nil { return nil, err }
        }
        return t.backend.SpanIter(t.ctx, node, child)

    case *traceql.EventScanNode:
        return t.backend.EventIter(t.ctx, node, nil)

    case *traceql.LinkScanNode:
        return t.backend.LinkIter(t.ctx, node, nil)

    default:
        return nil, fmt.Errorf("plan_translator: unexpected inner node %T", n)
    }
}

// translateProject builds:
//  1. The first-pass chain from ProjectNode's child (scan + filter nodes)
//  2. A ProjectIter that re-fetches Columns for surviving spans via a second-pass scan
func (t *translator) translateProject(node *traceql.ProjectNode) (Evaluatable, error) {
    // Recurse into child to get first-pass evaluatable
    firstPass, err := t.translate(node.Children()[0])
    if err != nil { return nil, err }

    // ProjectIter wraps firstPass and the backend so it can build the second pass.
    // When iterated, it collects surviving span row numbers from firstPass, then
    // calls buildParquetChain with primary=rowNumbers to re-fetch node.Columns.
    return newProjectEvaluatable(node, firstPass, t), nil
}

// Stub constructors — implementer fills these in:
func newRateEvaluatable(n *traceql.RateNode, child Evaluatable) Evaluatable           { panic("TODO") }
func newCountOverTimeEvaluatable(n *traceql.CountOverTimeNode, child Evaluatable) Evaluatable { panic("TODO") }
func newGroupByEvaluatable(n *traceql.GroupByNode, child Evaluatable) Evaluatable     { panic("TODO") }
func newCoalesceEvaluatable(n *traceql.CoalesceNode, child Evaluatable) Evaluatable   { panic("TODO") }
func newFilterEvaluatable(n *traceql.SpansetFilterNode, child Evaluatable) Evaluatable{ panic("TODO") }
func newStructuralOpEvaluatable(n *traceql.StructuralOpNode, l, r Evaluatable) Evaluatable { panic("TODO") }
func newSpansetEvaluatable(iter traceql.SpansetIterator) Evaluatable                  { panic("TODO") }
func newProjectEvaluatable(n *traceql.ProjectNode, child Evaluatable, t *translator) Evaluatable { panic("TODO") }
```

> **Implementer note:** The stub constructors mirror the existing evaluator types in `pkg/traceql/engine_metrics.go` and `ast_execute.go`. Study those files to build the concrete implementations. The `RateEvaluatable`, `GroupByEvaluatable`, etc. should closely follow the existing `MetricsEvaluator` and `GroupingAggregator` patterns.

**Step 4: Run test**

```bash
go test ./tempodb/encoding/common/ -run TestTranslate
```

**Step 5: Commit**

```bash
git add tempodb/encoding/common/plan_translator.go tempodb/encoding/common/plan_translator_test.go
git commit -m "feat(common): add plan translator building evaluatable from plan tree via ScanBackend"
```

---

### Task 9: vparquet5 ScanBackend implementation

**Files:**
- Create: `tempodb/encoding/vparquet5/scan_backend.go`
- Create: `tempodb/encoding/vparquet5/scan_backend_test.go`

**Step 1: Write the failing test**

```go
// tempodb/encoding/vparquet5/scan_backend_test.go
package vparquet5

import (
    "context"
    "testing"

    "github.com/grafana/tempo/pkg/traceql"
    "github.com/stretchr/testify/require"
)

func TestScanBackend_ImplementsInterface(t *testing.T) {
    // Compile-time check: *blockScanBackend implements common.ScanBackend
    var _ = common.ScanBackend((*blockScanBackend)(nil))
}

func TestScanBackend_SpanIter_ReturnsIter(t *testing.T) {
    // Use an existing test block (see block_traceql_test.go for helpers)
    b := makeTestBlock(t)
    backend := newBlockScanBackend(b, []parquet.RowGroup{}, nil)

    node := &traceql.SpanScanNode{
        Conditions: []traceql.Condition{
            {Attribute: traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, "status")},
        },
    }
    iter, err := backend.SpanIter(context.Background(), node, nil)
    require.NoError(t, err)
    require.NotNil(t, iter)
    iter.Close()
}
```

**Step 2: Run to see failure**

```bash
go test ./tempodb/encoding/vparquet5/ -run TestScanBackend
```

**Step 3: Implement by wrapping existing create*Iterator functions**

```go
// tempodb/encoding/vparquet5/scan_backend.go
package vparquet5

import (
    "context"

    "github.com/grafana/parquet-go/parquet"
    "github.com/grafana/tempo/pkg/parquetquery"
    "github.com/grafana/tempo/pkg/traceql"
    "github.com/grafana/tempo/tempodb/backend"
    "github.com/grafana/tempo/tempodb/encoding/common"
)

// blockScanBackend implements common.ScanBackend for a single vparquet5 block.
type blockScanBackend struct {
    block            *backendBlock
    rowGroups        []parquet.RowGroup
    dedicatedColumns backend.DedicatedColumns
    // makeIter and makeNilIter match what createAllIterator uses internally
    makeIter    makeIterFn
    makeNilIter makeIterFn
}

func newBlockScanBackend(block *backendBlock, rgs []parquet.RowGroup, dc backend.DedicatedColumns) *blockScanBackend {
    // Implementer: initialize makeIter/makeNilIter using the same pattern as
    // createAllIterator does when it sets up row group iterators.
    return &blockScanBackend{block: block, rowGroups: rgs, dedicatedColumns: dc}
}

var _ common.ScanBackend = (*blockScanBackend)(nil)

func (b *blockScanBackend) SpanIter(ctx context.Context, node *traceql.SpanScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
    // Delegate to createSpanIterator.
    // Implementer: extract the necessary parameters from node.Conditions,
    // node.AllConditions, and b.dedicatedColumns.
    return createSpanIterator(b.makeIter, b.makeNilIter, nil, node.Conditions, false, b.dedicatedColumns, false, 0, 0)
}

func (b *blockScanBackend) InstrumentationScopeIter(ctx context.Context, node *traceql.InstrumentationScopeScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
    return createInstrumentationIterator(b.makeIter, b.makeNilIter, child, node.Conditions, false, false)
}

func (b *blockScanBackend) ResourceIter(ctx context.Context, node *traceql.ResourceScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
    return createResourceIterator(b.makeIter, b.makeNilIter, child, node.Conditions, false, false, b.dedicatedColumns, false)
}

func (b *blockScanBackend) TraceIter(ctx context.Context, node *traceql.TraceScanNode, primary parquetquery.Iterator, child parquetquery.Iterator) (traceql.SpansetIterator, error) {
    iter, err := createTraceIterator(b.makeIter, child, node.Conditions, 0, 0, false, false, nil)
    if err != nil {
        return nil, err
    }
    return newSpansetIterator(newRebatchIterator(iter)), nil
}

func (b *blockScanBackend) EventIter(ctx context.Context, node *traceql.EventScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
    // Implementer: delegate to createEventIterator (add this function if it doesn't exist,
    // modeled on createSpanIterator but for the events columns).
    panic("TODO: createEventIterator")
}

func (b *blockScanBackend) LinkIter(ctx context.Context, node *traceql.LinkScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
    panic("TODO: createLinkIterator")
}
```

> **Implementer note:** Study `createAllIterator` (line 1657 in `block_traceql.go`) to understand how `makeIter`/`makeNilIter` are set up and how `categorizeConditions` splits conditions by scope. The `TraceIter` method with `primary != nil` is the second-pass path — pass `primary` as the first argument to `createTraceIterator` the same way `createAllIterator` does when `primaryIter != nil`.

**Step 4: Run tests**

```bash
go test ./tempodb/encoding/vparquet5/ -run TestScanBackend
```

**Step 5: Commit**

```bash
git add tempodb/encoding/vparquet5/scan_backend.go tempodb/encoding/vparquet5/scan_backend_test.go
git commit -m "feat(vparquet5): implement ScanBackend wrapping existing create*Iterator functions"
```

---

## Phase 3 — Wire up the querier

### Task 10: Update queryBlock to use plan-based flow

**Files:**
- Modify: `modules/querier/querier_query_range.go`
- Modify: `modules/querier/querier_query_range_test.go` (if it exists; otherwise create)

**Step 1: Write the failing test**

Add an integration test that calls `queryBlock` with a real request and asserts results come back. The existing test suite in `modules/querier/` should have fixtures to reference.

```go
// Verify the new plan-based path produces the same results as the old path.
// Implementer: copy an existing queryBlock test, add a variant using BuildPlan + Translate.
func TestQueryBlock_PlanBased(t *testing.T) {
    // Use existing test infrastructure in the querier package.
    // Assert that results from the plan-based path match the legacy path.
}
```

**Step 2: Implement**

Replace the existing `CompileMetricsQueryRange` + `MetricsEvaluator.Do` call in `queryBlock` with:

```go
// modules/querier/querier_query_range.go (inside queryBlock)

// 1. Build plan tree from the parsed expression
plan, err := traceql.BuildPlan(expr)
if err != nil {
    return nil, err
}

// 2. Get a ScanBackend from the store for this block
backend, err := q.store.ScanBackend(meta, opts)
if err != nil {
    return nil, err
}

// 3. Translate the full plan tree into an Evaluatable
eval, err := common.Translate(ctx, plan, backend, common.DefaultSearchOptions())
if err != nil {
    return nil, err
}

// 4. Drive execution
err = eval.Do(ctx, uint64(meta.StartTime.UnixNano()), uint64(meta.EndTime.UnixNano()), int(req.MaxSeries))
if err != nil {
    return nil, err
}

res := eval.Results()
inspectedBytes, spansTotal, _ := eval.Metrics()
```

> **Implementer note:** `q.store.ScanBackend(meta, opts)` requires adding a `ScanBackend` method to the `Store` interface and implementing it in vparquet5's `backendBlock`. The method returns a `*blockScanBackend` set up for the given block metadata and search options (start page, total pages, row groups).

**Step 3: Run all querier tests**

```bash
go test ./modules/querier/... -run TestQueryBlock
```

**Step 4: Run broader integration tests**

```bash
go test ./tempodb/... -count=1 -timeout 120s
```

**Step 5: Commit**

```bash
git add modules/querier/querier_query_range.go
git commit -m "feat(querier): switch queryBlock to plan-based translation path"
```

---

## Phase 4 — Cleanup (separate PR after Phase 3 is stable)

1. Remove `FetchSpansRequest` from `pkg/traceql/storage.go` once all callers are migrated.
2. Remove `Pipeline.evaluate()` AST dual-responsibility (keep AST structure, move eval to `Evaluatable` impls).
3. Remove `bridgeIterator` and `rebatchIterator` from `block_traceql.go` if no longer referenced.
4. Run full test suite and benchmarks to confirm no regression.

---

**Plan complete and saved to `docs/plans/2026-03-04-plan-node-tree-impl.md`. Two execution options:**

**1. Subagent-Driven (this session)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Parallel Session (separate)** — Open a new session with executing-plans, batch execution with checkpoints.

Which approach?
