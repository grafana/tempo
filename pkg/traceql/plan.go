package traceql

import (
	"fmt"
	"strings"

	"github.com/grafana/tempo/pkg/tempopb"
)

// conditionStrings formats a slice of Conditions as human-readable predicates,
// e.g. ["span.http.status_code >= 500", "span.status (fetch)"].
func conditionStrings(conds []Condition) []string {
	out := make([]string, 0, len(conds))
	for _, c := range conds {
		if c.Op == OpNone {
			out = append(out, c.Attribute.String())
		} else {
			parts := []string{c.Attribute.String(), c.Op.String()}
			for _, o := range c.Operands {
				parts = append(parts, o.String())
			}
			out = append(out, strings.Join(parts, " "))
		}
	}
	return out
}

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

// funcVisitor is a PlanVisitor backed by two functions.
// It is used internally by plan rules and tests.
type funcVisitor struct {
	pre  func(PlanNode) bool
	post func(PlanNode)
}

func (v *funcVisitor) VisitPre(n PlanNode) bool { return v.pre(n) }
func (v *funcVisitor) VisitPost(n PlanNode)      { v.post(n) }

// --- Scan nodes (1:1 with parquet iterator levels) ---

// TraceScanNode is the root of a scan tree — maps to the trace-level parquet iterator.
type TraceScanNode struct {
	Conditions    []Condition
	AllConditions bool
	child         PlanNode
}

func NewTraceScanNode(conditions []Condition, allConditions bool, child PlanNode) *TraceScanNode {
	return &TraceScanNode{Conditions: conditions, AllConditions: allConditions, child: child}
}

func (n *TraceScanNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *TraceScanNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *TraceScanNode) String() string {
	if len(n.Conditions) == 0 {
		return "TraceScan"
	}
	return fmt.Sprintf("TraceScan(%s)", strings.Join(conditionStrings(n.Conditions), ", "))
}

// WithChild returns a shallow copy with the child replaced.
func (n *TraceScanNode) WithChild(child PlanNode) *TraceScanNode {
	cp := *n
	cp.child = child
	return &cp
}

// ResourceScanNode maps to the resource-level parquet iterator.
type ResourceScanNode struct {
	Conditions []Condition
	child      PlanNode
}

func NewResourceScanNode(conditions []Condition, child PlanNode) *ResourceScanNode {
	return &ResourceScanNode{Conditions: conditions, child: child}
}

func (n *ResourceScanNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *ResourceScanNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *ResourceScanNode) String() string {
	if len(n.Conditions) == 0 {
		return "ResourceScan"
	}
	return fmt.Sprintf("ResourceScan(%s)", strings.Join(conditionStrings(n.Conditions), ", "))
}

func (n *ResourceScanNode) WithChild(child PlanNode) *ResourceScanNode {
	cp := *n
	cp.child = child
	return &cp
}

// InstrumentationScopeScanNode maps to the instrumentation-scope-level parquet iterator.
type InstrumentationScopeScanNode struct {
	Conditions []Condition
	child      PlanNode
}

func NewInstrumentationScopeScanNode(conditions []Condition, child PlanNode) *InstrumentationScopeScanNode {
	return &InstrumentationScopeScanNode{Conditions: conditions, child: child}
}

func (n *InstrumentationScopeScanNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *InstrumentationScopeScanNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *InstrumentationScopeScanNode) String() string {
	if len(n.Conditions) == 0 {
		return "InstrumentationScopeScan"
	}
	return fmt.Sprintf("InstrumentationScopeScan(%s)", strings.Join(conditionStrings(n.Conditions), ", "))
}

func (n *InstrumentationScopeScanNode) WithChild(child PlanNode) *InstrumentationScopeScanNode {
	cp := *n
	cp.child = child
	return &cp
}

// SpanScanNode maps to the span-level parquet iterator.
type SpanScanNode struct {
	Conditions []Condition
	child      PlanNode
}

func NewSpanScanNode(conditions []Condition, child PlanNode) *SpanScanNode {
	return &SpanScanNode{Conditions: conditions, child: child}
}

func (n *SpanScanNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *SpanScanNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *SpanScanNode) String() string {
	if len(n.Conditions) == 0 {
		return "SpanScan"
	}
	return fmt.Sprintf("SpanScan(%s)", strings.Join(conditionStrings(n.Conditions), ", "))
}

func (n *SpanScanNode) WithChild(child PlanNode) *SpanScanNode {
	cp := *n
	cp.child = child
	return &cp
}

// EventScanNode maps to the event-level parquet iterator.
type EventScanNode struct {
	Conditions []Condition
	child      PlanNode
}

func NewEventScanNode(conditions []Condition, child PlanNode) *EventScanNode {
	return &EventScanNode{Conditions: conditions, child: child}
}

func (n *EventScanNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *EventScanNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *EventScanNode) String() string {
	return fmt.Sprintf("EventScan(%d conditions)", len(n.Conditions))
}

// LinkScanNode maps to the link-level parquet iterator.
type LinkScanNode struct {
	Conditions []Condition
	child      PlanNode
}

func NewLinkScanNode(conditions []Condition, child PlanNode) *LinkScanNode {
	return &LinkScanNode{Conditions: conditions, child: child}
}

func (n *LinkScanNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *LinkScanNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *LinkScanNode) String() string {
	return fmt.Sprintf("LinkScan(%d conditions)", len(n.Conditions))
}

// --- Engine nodes (in-memory evaluation, above scan nodes) ---

// SpansetFilterNode evaluates a filter expression against each spanset.
// Simple per-span predicates are pushed into scan conditions by the optimizer.
// Structural/cross-span predicates remain as SpansetFilterNode.
type SpansetFilterNode struct {
	Expression *SpansetFilter
	child      PlanNode
}

func NewSpansetFilterNode(expr *SpansetFilter, child PlanNode) *SpansetFilterNode {
	return &SpansetFilterNode{Expression: expr, child: child}
}

func (n *SpansetFilterNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *SpansetFilterNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *SpansetFilterNode) String() string {
	return fmt.Sprintf("SpansetFilter(%s)", n.Expression)
}

func (n *SpansetFilterNode) WithChild(child PlanNode) *SpansetFilterNode {
	cp := *n
	cp.child = child
	return &cp
}

// GroupByNode groups spans into spansets by the given attribute expression.
type GroupByNode struct {
	By    FieldExpression
	child PlanNode
}

func NewGroupByNode(by FieldExpression, child PlanNode) *GroupByNode {
	return &GroupByNode{By: by, child: child}
}

func (n *GroupByNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *GroupByNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *GroupByNode) String() string       { return fmt.Sprintf("GroupBy(%v)", n.By) }

func (n *GroupByNode) WithChild(child PlanNode) *GroupByNode {
	cp := *n
	cp.child = child
	return &cp
}

// CoalesceNode merges all spans in child spansets into a single spanset per trace.
type CoalesceNode struct {
	child PlanNode
}

func NewCoalesceNode(child PlanNode) *CoalesceNode {
	return &CoalesceNode{child: child}
}

func (n *CoalesceNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *CoalesceNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *CoalesceNode) String() string       { return "Coalesce" }

// StructuralOp represents a structural relationship (>>, >, ~, <<, <) between two spansets.
type StructuralOp int

const (
	StructuralOpParent     StructuralOp = iota // >>
	StructuralOpAncestor                       // >
	StructuralOpSibling                        // ~
	StructuralOpDescendant                     // <<
	StructuralOpChild                          // <
)

// StructuralOpNode evaluates a structural relationship between two spansets.
type StructuralOpNode struct {
	Op    StructuralOp
	left  PlanNode
	right PlanNode
}

func NewStructuralOpNode(op StructuralOp, left, right PlanNode) *StructuralOpNode {
	return &StructuralOpNode{Op: op, left: left, right: right}
}

func (n *StructuralOpNode) Children() []PlanNode { return []PlanNode{n.left, n.right} }
func (n *StructuralOpNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *StructuralOpNode) String() string {
	// TraceQL syntax: child(>), parent(<), descendant(>>), ancestor(<<), sibling(~)
	names := [...]string{"parent(<)", "ancestor(<<)", "sibling(~)", "descendant(>>)", "child(>)"}
	if int(n.Op) < len(names) {
		return "StructuralOp " + names[n.Op]
	}
	return fmt.Sprintf("StructuralOp(%d)", n.Op)
}

// --- ProjectNode: second-pass fetch boundary ---

// ProjectNode triggers a second-pass parquet fetch for additional columns
// on surviving spans. It is placed above the scan tree but below engine nodes
// so that grouping happens after the fetch.
type ProjectNode struct {
	Columns []Attribute
	child   PlanNode
}

func NewProjectNode(columns []Attribute, child PlanNode) *ProjectNode {
	return &ProjectNode{Columns: columns, child: child}
}

func (n *ProjectNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *ProjectNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *ProjectNode) String() string       { return fmt.Sprintf("Project(%v)", n.Columns) }

func (n *ProjectNode) WithChild(child PlanNode) *ProjectNode {
	cp := *n
	cp.child = child
	return &cp
}

// --- Metrics nodes (purely logical aggregation config) ---

// RateNode aggregates spans into a rate time series.
type RateNode struct {
	By        []Attribute
	Start     uint64
	End       uint64
	Step      uint64
	Exemplars uint32
	child     PlanNode
}

func NewRateNode(by []Attribute, child PlanNode) *RateNode {
	return &RateNode{By: by, child: child}
}

func newRateNodeFromReq(by []Attribute, req *tempopb.QueryRangeRequest, child PlanNode) *RateNode {
	return &RateNode{
		By:        by,
		Start:     req.Start,
		End:       req.End,
		Step:      req.Step,
		Exemplars: req.Exemplars,
		child:     child,
	}
}

func (n *RateNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *RateNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *RateNode) String() string {
	if len(n.By) == 0 {
		return "Rate"
	}
	parts := make([]string, len(n.By))
	for i, a := range n.By {
		parts[i] = a.String()
	}
	return fmt.Sprintf("Rate by(%s)", strings.Join(parts, ", "))
}

// CountOverTimeNode aggregates spans into a count-over-time time series.
type CountOverTimeNode struct {
	By        []Attribute
	Start     uint64
	End       uint64
	Step      uint64
	Exemplars uint32
	child     PlanNode
}

func NewCountOverTimeNode(by []Attribute, child PlanNode) *CountOverTimeNode {
	return &CountOverTimeNode{By: by, child: child}
}

func newCountOverTimeNodeFromReq(by []Attribute, req *tempopb.QueryRangeRequest, child PlanNode) *CountOverTimeNode {
	return &CountOverTimeNode{
		By:        by,
		Start:     req.Start,
		End:       req.End,
		Step:      req.Step,
		Exemplars: req.Exemplars,
		child:     child,
	}
}

func (n *CountOverTimeNode) Children() []PlanNode {
	if n.child != nil {
		return []PlanNode{n.child}
	}
	return nil
}
func (n *CountOverTimeNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *CountOverTimeNode) String() string {
	if len(n.By) == 0 {
		return "CountOverTime"
	}
	parts := make([]string, len(n.By))
	for i, a := range n.By {
		parts[i] = a.String()
	}
	return fmt.Sprintf("CountOverTime by(%s)", strings.Join(parts, ", "))
}
