package traceql

// PredicatePushdownRule pushes definite-scope predicates from a SpansetFilterNode
// into the appropriate scan node, eliminating the filter entirely.
//
// Handles three cases:
//  1. Always-true static expression (e.g. "{}") — pushed into SpanScan.
//  2. A single BinaryOperation (attr op static) where attr is span-scope,
//     resource-scope, or a scoped intrinsic — pushed into SpanScan or ResourceScan.
//  3. An AND of multiple such operations — each operand pushed to its scan level.
//
// True unscoped user attributes (.foo) are NOT handled here; they are handled
// by UnscopedAttributePushdownRule, which pushes fetch-only conditions to both
// scan levels and keeps the filter for correct evaluation.
func PredicatePushdownRule() Rule {
	return FuncRule("predicate-pushdown", func(n PlanNode) (PlanNode, bool) {
		filter, ok := n.(*SpansetFilterNode)
		if !ok {
			return n, false
		}
		if !isSimplePerSpanFilter(filter.Expression) {
			return n, false
		}

		// Extract conditions. Start with AllConditions=true so that AND
		// expressions (which never call the OpOr branch) keep it true.
		req := &FetchSpansRequest{AllConditions: true}
		filter.Expression.extractConditions(req)

		// Partition conditions by scan-level destination.
		var spanConds, resourceConds []Condition
		for _, c := range req.Conditions {
			switch c.Attribute.Scope {
			case AttributeScopeSpan:
				spanConds = append(spanConds, c)
			case AttributeScopeResource:
				resourceConds = append(resourceConds, c)
			default:
				// AttributeScopeNone — always an intrinsic here because
				// isSimplePerSpanFilter excludes unscoped user attributes.
				// Push to SpanScan; the storage layer maps intrinsics to
				// their actual scope via intrinsicColumnLookups.
				spanConds = append(spanConds, c)
			}
		}

		if len(spanConds) > 0 {
			spanScan := firstSpanScanNode(filter.child)
			if spanScan == nil {
				return n, false
			}
			spanScan.Conditions = append(spanConds, spanScan.Conditions...)
			spanScan.AllConditions = req.AllConditions
		}

		if len(resourceConds) > 0 {
			resourceScan := firstResourceScanNode(filter.child)
			if resourceScan == nil {
				return n, false
			}
			resourceScan.Conditions = append(resourceConds, resourceScan.Conditions...)
			resourceScan.AllConditions = req.AllConditions
		}

		// Propagate AllConditions to the TraceScanNode when conditions were pushed.
		if len(req.Conditions) > 0 {
			if trace := firstTraceScanNode(filter.child); trace != nil {
				trace.AllConditions = req.AllConditions
			}
		}

		// Remove the filter node — its predicate is now encoded in the scans.
		return filter.child, true
	})
}

// OrPredicatePushdownRule adds fetch-only (OpNone) conditions for each simple
// leaf predicate found inside an OR expression, routing span-scope attributes
// to SpanScanNode and resource-scope attributes to ResourceScanNode.
// The SpansetFilterNode is kept because OR semantics require in-memory
// evaluation — the scan layer cannot express OR across different scopes or
// even across two columns at the same scope.
//
// Conditions are pushed as OpNone rather than with their original Op to avoid
// incorrect row exclusion: pushing a real filter predicate at ResourceScan
// level would drop entire resources before the span-branch of the OR is
// evaluated.
func OrPredicatePushdownRule() Rule {
	return FuncRule("or-predicate-pushdown", func(n PlanNode) (PlanNode, bool) {
		filter, ok := n.(*SpansetFilterNode)
		if !ok || filter.Expression == nil {
			return n, false
		}
		if !containsOr(filter.Expression.Expression) {
			return n, false
		}

		req := &FetchSpansRequest{AllConditions: false}
		filter.Expression.extractConditions(req)

		var spanConds, resourceConds []Condition
		for _, c := range req.Conditions {
			fetch := Condition{Attribute: c.Attribute, Op: OpNone}
			switch c.Attribute.Scope {
			case AttributeScopeSpan:
				spanConds = append(spanConds, fetch)
			case AttributeScopeResource:
				resourceConds = append(resourceConds, fetch)
			default:
				if c.Attribute.Intrinsic != IntrinsicNone {
					spanConds = append(spanConds, fetch)
				}
			}
		}

		changed := false
		if len(spanConds) > 0 {
			if spanScan := firstSpanScanNode(filter.child); spanScan != nil {
				for _, c := range spanConds {
					if !hasConditionForAttr(spanScan.Conditions, c.Attribute) {
						spanScan.Conditions = append(spanScan.Conditions, c)
						changed = true
					}
				}
			}
		}
		if len(resourceConds) > 0 {
			if resourceScan := firstResourceScanNode(filter.child); resourceScan != nil {
				for _, c := range resourceConds {
					if !hasConditionForAttr(resourceScan.Conditions, c.Attribute) {
						resourceScan.Conditions = append(resourceScan.Conditions, c)
						changed = true
					}
				}
			}
		}
		// Keep the filter — OR semantics require in-memory evaluation.
		return n, changed
	})
}

// containsOr reports whether e contains at least one OpOr binary operation.
func containsOr(e FieldExpression) bool {
	switch expr := e.(type) {
	case *BinaryOperation:
		if expr.Op == OpOr {
			return true
		}
		return containsOr(expr.LHS) || containsOr(expr.RHS)
	case UnaryOperation:
		return containsOr(expr.Expression)
	}
	return false
}

// UnscopedAttributePushdownRule adds fetch-only conditions for unscoped user
// attributes (.foo) to both SpanScanNode and ResourceScanNode so the storage
// layer pre-fetches the attribute at both levels. The SpansetFilterNode is kept
// because unscoped attributes require OR semantics (span OR resource level),
// which cannot be expressed as a scan-level filter.
//
// This rule is separate from PredicatePushdownRule so it can be
// enabled or disabled independently.
func UnscopedAttributePushdownRule() Rule {
	return FuncRule("unscoped-attribute-pushdown", func(n PlanNode) (PlanNode, bool) {
		filter, ok := n.(*SpansetFilterNode)
		if !ok || filter.Expression == nil {
			return n, false
		}

		// Collect unscoped user attributes referenced in the expression.
		var attrs []Attribute
		collectUnscopedUserAttrs(filter.Expression.Expression, &attrs)
		if len(attrs) == 0 {
			return n, false
		}

		// Push OpNone (fetch-only) conditions to both scan levels.
		// OpNone tells the storage layer to fetch the attribute without filtering.
		changed := false
		if spanScan := firstSpanScanNode(filter.child); spanScan != nil {
			for _, attr := range attrs {
				if !hasConditionForAttr(spanScan.Conditions, attr) {
					spanScan.Conditions = append(spanScan.Conditions, Condition{Attribute: attr, Op: OpNone})
					changed = true
				}
			}
		}
		if resourceScan := firstResourceScanNode(filter.child); resourceScan != nil {
			for _, attr := range attrs {
				if !hasConditionForAttr(resourceScan.Conditions, attr) {
					resourceScan.Conditions = append(resourceScan.Conditions, Condition{Attribute: attr, Op: OpNone})
					changed = true
				}
			}
		}

		// Keep the filter node — it performs the actual predicate evaluation.
		return n, changed
	})
}

// GroupByHoistRule hoists a GroupByNode above a ProjectNode when the
// GroupByNode is the direct child of the ProjectNode:
//
//	Before: ProjectNode(cols, child=GroupByNode(by, child=X), fetchTree=F)
//	After:  GroupByNode(by, child=ProjectNode(cols, child=X, fetchTree=F))
//
// This is required for correct late-materialization in search queries.
// lateMaterializeIter drives on a forward-only fetch iterator and performs
// exactly one SeekTo per spanset. GroupByNode can emit multiple spansets for
// the same trace (one per group value), which would require seeking backward
// to the same trace row — something a forward iterator cannot do.
// By hoisting GroupByNode above ProjectNode, metadata is fetched 1-to-1 per
// trace first; GroupByNode then partitions the already-enriched spansets.
func GroupByHoistRule() Rule {
	return FuncRule("group-by-hoist", func(n PlanNode) (PlanNode, bool) {
		proj, ok := n.(*ProjectNode)
		if !ok {
			return n, false
		}
		group, ok := proj.child.(*GroupByNode)
		if !ok {
			return n, false
		}
		// Sink ProjectNode below GroupByNode, keeping the same fetchTree.
		newProj := NewProjectNode(proj.Columns, group.child, proj.fetchTree)
		return group.WithChild(newProj), true
	})
}

// GroupByFetchRule adds OpNone (fetch-only) conditions for the GroupByNode's
// By expression to the appropriate scan nodes below it, so the storage layer
// pre-fetches the group-by attribute.
//
//   - resource-scoped attributes → ResourceScanNode
//   - span-scoped attributes     → SpanScanNode
//   - unscoped user attributes   → both (OR semantics, same as UnscopedAttributePushdownRule)
//   - scoped intrinsics          → SpanScanNode (storage layer resolves actual column)
//
// AllConditions on scan nodes is intentionally NOT changed. In the legacy
// engine GroupOperation.extractConditions forced AllConditions=false because it
// mixed fetch-only and filter conditions in the same request. In the plan-based
// approach they are kept separate: filter predicates already control selectivity
// and OpNone conditions are purely additive fetches.
func GroupByFetchRule() Rule {
	return FuncRule("group-by-fetch", func(n PlanNode) (PlanNode, bool) {
		group, ok := n.(*GroupByNode)
		if !ok {
			return n, false
		}
		req := &FetchSpansRequest{AllConditions: false}
		group.By.extractConditions(req)
		if len(req.Conditions) == 0 {
			return n, false
		}

		changed := false
		for _, c := range req.Conditions {
			fetch := Condition{Attribute: c.Attribute, Op: OpNone}
			switch c.Attribute.Scope {
			case AttributeScopeResource:
				if res := firstResourceScanNode(group.child); res != nil {
					if !hasConditionForAttr(res.Conditions, c.Attribute) {
						res.Conditions = append(res.Conditions, fetch)
						changed = true
					}
				}
			case AttributeScopeSpan:
				if span := firstSpanScanNode(group.child); span != nil {
					if !hasConditionForAttr(span.Conditions, c.Attribute) {
						span.Conditions = append(span.Conditions, fetch)
						changed = true
					}
				}
			case AttributeScopeNone:
				if c.Attribute.Intrinsic != IntrinsicNone {
					// Scoped intrinsic: storage maps it to the real column.
					if span := firstSpanScanNode(group.child); span != nil {
						if !hasConditionForAttr(span.Conditions, c.Attribute) {
							span.Conditions = append(span.Conditions, fetch)
							changed = true
						}
					}
				} else {
					// Unscoped user attribute: push to both levels (OR semantics).
					if span := firstSpanScanNode(group.child); span != nil {
						if !hasConditionForAttr(span.Conditions, c.Attribute) {
							span.Conditions = append(span.Conditions, fetch)
							changed = true
						}
					}
					if res := firstResourceScanNode(group.child); res != nil {
						if !hasConditionForAttr(res.Conditions, c.Attribute) {
							res.Conditions = append(res.Conditions, fetch)
							changed = true
						}
					}
				}
			}
		}
		return n, changed
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

// FetchTreeDeduplicationRule removes OpNone conditions from the fetch scan
// tree that are already covered by conditions in the drive scan tree.
// When the drive pass has already evaluated or fetched an attribute (either
// as a filter or a fetch-only condition), the fetch pass does not need to
// re-read it from storage.
func FetchTreeDeduplicationRule() Rule {
	return FuncRule("fetch-tree-dedup", func(n PlanNode) (PlanNode, bool) {
		proj, ok := n.(*ProjectNode)
		if !ok || proj.fetchTree == nil {
			return n, false
		}

		// Collect all attributes already present in the drive tree's scan nodes.
		driveAttrs := make(map[Attribute]struct{})
		WalkPlan(proj.child, &funcVisitor{
			pre: func(node PlanNode) bool {
				switch s := node.(type) {
				case *SpanScanNode:
					for _, c := range s.Conditions {
						driveAttrs[c.Attribute] = struct{}{}
					}
				case *ResourceScanNode:
					for _, c := range s.Conditions {
						driveAttrs[c.Attribute] = struct{}{}
					}
				case *TraceScanNode:
					for _, c := range s.Conditions {
						driveAttrs[c.Attribute] = struct{}{}
					}
				}
				return true
			},
			post: func(PlanNode) {},
		})

		if len(driveAttrs) == 0 {
			return n, false
		}

		// Remove those attributes from the fetch tree.
		changed := false
		WalkPlan(proj.fetchTree, &funcVisitor{
			pre: func(node PlanNode) bool {
				switch s := node.(type) {
				case *SpanScanNode:
					s.Conditions = filterOutAttrs(s.Conditions, driveAttrs, &changed)
				case *ResourceScanNode:
					s.Conditions = filterOutAttrs(s.Conditions, driveAttrs, &changed)
				case *TraceScanNode:
					s.Conditions = filterOutAttrs(s.Conditions, driveAttrs, &changed)
				}
				return true
			},
			post: func(PlanNode) {},
		})

		return n, changed
	})
}

// filterOutAttrs returns a copy of conds with any condition whose attribute
// appears in attrs removed, setting *changed to true for each removal.
func filterOutAttrs(conds []Condition, attrs map[Attribute]struct{}, changed *bool) []Condition {
	result := conds[:0]
	for _, c := range conds {
		if _, ok := attrs[c.Attribute]; ok {
			*changed = true
			continue
		}
		result = append(result, c)
	}
	return result
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

// isSimplePerSpanFilter returns true if the SpansetFilter expression can be fully
// pushed into scan nodes with the filter node eliminated. See isPushable.
func isSimplePerSpanFilter(f *SpansetFilter) bool {
	if f == nil {
		return false
	}
	return isPushable(f.Expression)
}

// isPushable returns true if e can be completely encoded as scan conditions,
// allowing the enclosing SpansetFilterNode to be eliminated.
//
// Accepted forms:
//   - Always-true Static
//   - BinaryOperation(attr op static) where attr is span-scope, resource-scope,
//     or a scoped intrinsic (Scope=None but Intrinsic≠None)
//   - AND of any combination of the above (recursively)
//
// True unscoped user attributes (Scope=None, Intrinsic=None) are excluded
// because the storage layer applies them with OR semantics across scan levels,
// which cannot be fully encoded as a filter condition.
func isPushable(e FieldExpression) bool {
	switch expr := e.(type) {
	case Static:
		b, ok := expr.Bool()
		return ok && b
	case *BinaryOperation:
		if expr.Op == OpAnd {
			return isPushable(expr.LHS) && isPushable(expr.RHS)
		}
		return isSimpleBinaryOp(expr)
	}
	return false
}

// isSimpleBinaryOp returns true when op is a (attr, static) or (static, attr)
// comparison whose attribute has a definite scan-level scope: span, resource,
// or a scoped intrinsic (Scope=None, Intrinsic≠None).
//
// True unscoped user attributes (Scope=None, Intrinsic=None) return false —
// they require OR semantics and are handled by UnscopedAttributePushdownRule.
func isSimpleBinaryOp(op *BinaryOperation) bool {
	isAcceptableScope := func(attr Attribute) bool {
		switch attr.Scope {
		case AttributeScopeSpan, AttributeScopeResource:
			return true
		case AttributeScopeNone:
			// Intrinsics have a known scope resolved by the storage layer.
			return attr.Intrinsic != IntrinsicNone
		}
		return false
	}
	lAttr, lIsAttr := op.LHS.(Attribute)
	_, rIsStatic := op.RHS.(Static)
	if lIsAttr && rIsStatic {
		return isAcceptableScope(lAttr)
	}
	rAttr, rIsAttr := op.RHS.(Attribute)
	_, lIsStatic := op.LHS.(Static)
	if rIsAttr && lIsStatic {
		return isAcceptableScope(rAttr)
	}
	return false
}

// collectUnscopedUserAttrs walks e and appends every Attribute that is an
// unscoped user attribute (Scope=None, Intrinsic=None).
func collectUnscopedUserAttrs(e FieldExpression, attrs *[]Attribute) {
	switch expr := e.(type) {
	case Attribute:
		if expr.Scope == AttributeScopeNone && expr.Intrinsic == IntrinsicNone {
			*attrs = append(*attrs, expr)
		}
	case *BinaryOperation:
		collectUnscopedUserAttrs(expr.LHS, attrs)
		collectUnscopedUserAttrs(expr.RHS, attrs)
	case UnaryOperation:
		collectUnscopedUserAttrs(expr.Expression, attrs)
	}
}

// hasConditionForAttr reports whether conds already contains a condition for attr.
func hasConditionForAttr(conds []Condition, attr Attribute) bool {
	for _, c := range conds {
		if c.Attribute == attr {
			return true
		}
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

// firstResourceScanNode returns the first *ResourceScanNode found by walking
// the subtree, or nil if none exists.
func firstResourceScanNode(n PlanNode) *ResourceScanNode {
	var result *ResourceScanNode
	WalkPlan(n, &funcVisitor{
		pre: func(node PlanNode) bool {
			if result != nil {
				return false
			}
			if r, ok := node.(*ResourceScanNode); ok {
				result = r
				return false
			}
			return true
		},
		post: func(PlanNode) {},
	})
	return result
}

// firstTraceScanNode returns the first *TraceScanNode found by walking the
// subtree, or nil if none exists.
func firstTraceScanNode(n PlanNode) *TraceScanNode {
	var result *TraceScanNode
	WalkPlan(n, &funcVisitor{
		pre: func(node PlanNode) bool {
			if result != nil {
				return false
			}
			if t, ok := node.(*TraceScanNode); ok {
				result = t
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
