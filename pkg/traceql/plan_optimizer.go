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

func (r *funcRule) Name() string                      { return r.name }
func (r *funcRule) Apply(n PlanNode) (PlanNode, bool) { return r.fn(n) }

// RuleSet applies a set of rules to a plan tree until fixpoint.
type RuleSet struct {
	rules []Rule
}

// NewRuleSet creates a RuleSet with the given rules.
func NewRuleSet(rules ...Rule) *RuleSet {
	return &RuleSet{rules: rules}
}

// Add appends a rule to the set.
func (rs *RuleSet) Add(r Rule) {
	rs.rules = append(rs.rules, r)
}

// Optimize applies rules bottom-up until no rule fires (fixpoint).
func (rs *RuleSet) Optimize(root PlanNode) PlanNode {
	for {
		next, changed := rs.applyOnce(root)
		root = next
		if !changed {
			return root
		}
	}
}

// applyOnce does a single bottom-up pass: recurse into children first,
// then apply rules to the (possibly updated) parent.
func (rs *RuleSet) applyOnce(n PlanNode) (PlanNode, bool) {
	if n == nil {
		return nil, false
	}

	changed := false

	// First, rewrite children bottom-up.
	n, changed = rs.rewriteChildren(n, changed)

	// Then apply rules to this node.
	for _, rule := range rs.rules {
		if rewritten, ok := rule.Apply(n); ok {
			n = rewritten
			changed = true
		}
	}
	return n, changed
}

// rewriteChildren rewrites a node's children and returns the updated node.
// Each concrete type that has mutable children implements a WithChildren method;
// we use a type switch to rebuild the node after rewriting.
func (rs *RuleSet) rewriteChildren(n PlanNode, changed bool) (PlanNode, bool) {
	switch node := n.(type) {
	case *TraceScanNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			return node.WithChild(newChild), true
		}
	case *ResourceScanNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			return node.WithChild(newChild), true
		}
	case *InstrumentationScopeScanNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			return node.WithChild(newChild), true
		}
	case *SpanScanNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			return node.WithChild(newChild), true
		}
	case *EventScanNode:
		// EventScanNode has a child field but no engine nodes sit above it in scan trees.
		return n, changed
	case *LinkScanNode:
		return n, changed
	case *SpansetFilterNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			return node.WithChild(newChild), true
		}
	case *GroupByNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			return node.WithChild(newChild), true
		}
	case *CoalesceNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			n = &CoalesceNode{child: newChild}
			return n, true
		}
	case *ProjectNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			return node.WithChild(newChild), true
		}
	case *RateNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			n = &RateNode{By: node.By, child: newChild}
			return n, true
		}
	case *CountOverTimeNode:
		if node.child == nil {
			return n, changed
		}
		newChild, childChanged := rs.applyOnce(node.child)
		if childChanged {
			n = &CountOverTimeNode{By: node.By, child: newChild}
			return n, true
		}
	case *StructuralOpNode:
		leftNew, leftChanged := rs.applyOnce(node.left)
		rightNew, rightChanged := rs.applyOnce(node.right)
		if leftChanged || rightChanged {
			return &StructuralOpNode{Op: node.Op, left: leftNew, right: rightNew}, true
		}
	}
	return n, changed
}
