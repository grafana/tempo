package traceql

// PredicatePushdownRule pushes conditions from a SpansetFilterNode that wraps
// only a trivial (always-true) expression down into the child SpanScanNode,
// eliminating the filter node entirely. More sophisticated pushdown (arbitrary
// FieldExpression → Condition extraction) is a follow-up.
func PredicatePushdownRule() Rule {
	return FuncRule("predicate-pushdown", func(n PlanNode) (PlanNode, bool) {
		filter, ok := n.(*SpansetFilterNode)
		if !ok {
			return n, false
		}
		if !isSimplePerSpanFilter(filter.Expression) {
			return n, false
		}
		// Extract conditions from the filter's FieldExpression and push them
		// into the first SpanScanNode in the subtree below.
		req := &FetchSpansRequest{}
		filter.Expression.extractConditions(req)

		scan := firstSpanScanNode(filter.child)
		if scan == nil {
			return n, false
		}
		scan.Conditions = append(req.Conditions, scan.Conditions...)
		// Remove the filter node — its predicate is now encoded in the scan.
		return filter.child, true
	})
}

// ConditionMergeRule merges two adjacent SpanScanNodes (or ResourceScanNodes)
// of the same type into a single node, combining their conditions.
func ConditionMergeRule() Rule {
	return FuncRule("condition-merge", func(n PlanNode) (PlanNode, bool) {
		switch outer := n.(type) {
		case *SpanScanNode:
			if inner, ok := outer.child.(*SpanScanNode); ok {
				merged := &SpanScanNode{
					Conditions: append(append([]Condition{}, outer.Conditions...), inner.Conditions...),
					child:      inner.child,
				}
				return merged, true
			}
		case *ResourceScanNode:
			if inner, ok := outer.child.(*ResourceScanNode); ok {
				merged := &ResourceScanNode{
					Conditions: append(append([]Condition{}, outer.Conditions...), inner.Conditions...),
					child:      inner.child,
				}
				return merged, true
			}
		}
		return n, false
	})
}

// SecondPassEliminatorRule removes a ProjectNode when all of its requested
// columns are already present in the first-pass scan nodes below it.
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

// isSimplePerSpanFilter returns true if the SpansetFilter contains only
// simple span-scope predicates that can be evaluated at the scan level.
// Currently recognises two cases:
//   1. A trivially-true static expression (e.g. "{}") — no conditions needed.
//   2. A BinaryOperation whose both sides are a span-scope Attribute and a Static.
func isSimplePerSpanFilter(f *SpansetFilter) bool {
	if f == nil {
		return false
	}
	switch e := f.Expression.(type) {
	case Static:
		// Always-true empty filter — safe to push down.
		b, ok := e.Bool()
		return ok && b
	case *BinaryOperation:
		return isSimpleBinaryOp(e)
	}
	return false
}

func isSimpleBinaryOp(op *BinaryOperation) bool {
	lAttr, lIsAttr := op.LHS.(Attribute)
	_, rIsStatic := op.RHS.(Static)
	if lIsAttr && rIsStatic {
		return lAttr.Scope == AttributeScopeSpan || lAttr.Scope == AttributeScopeNone
	}
	rAttr, rIsAttr := op.RHS.(Attribute)
	_, lIsStatic := op.LHS.(Static)
	if rIsAttr && lIsStatic {
		return rAttr.Scope == AttributeScopeSpan || rAttr.Scope == AttributeScopeNone
	}
	return false
}

// firstSpanScanNode returns the first *SpanScanNode found by walking the subtree,
// or nil if none exists.
func firstSpanScanNode(n PlanNode) *SpanScanNode {
	var result *SpanScanNode
	WalkPlan(n, &funcVisitor{
		pre: func(node PlanNode) bool {
			if result != nil {
				return false
			}
			if s, ok := node.(*SpanScanNode); ok {
				result = s
				return false
			}
			return true
		},
		post: func(PlanNode) {},
	})
	return result
}

// allColumnsInFirstPass returns true when every attribute in cols is already
// present as a condition in some SpanScanNode below subtree.
func allColumnsInFirstPass(cols []Attribute, subtree PlanNode) bool {
	if len(cols) == 0 {
		return true
	}
	// Collect all attributes already requested by scan nodes.
	have := make(map[Attribute]struct{})
	WalkPlan(subtree, &funcVisitor{
		pre: func(n PlanNode) bool {
			if s, ok := n.(*SpanScanNode); ok {
				for _, c := range s.Conditions {
					have[c.Attribute] = struct{}{}
				}
			}
			if s, ok := n.(*ResourceScanNode); ok {
				for _, c := range s.Conditions {
					have[c.Attribute] = struct{}{}
				}
			}
			return true
		},
		post: func(PlanNode) {},
	})
	for _, col := range cols {
		if _, ok := have[col]; !ok {
			return false
		}
	}
	return true
}
