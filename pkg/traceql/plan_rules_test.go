package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPredicatePushdownRule verifies that a SpansetFilterNode wrapping a simple
// span-scope BinaryOperation is eliminated and its condition pushed into the
// SpanScanNode below.
func TestPredicatePushdownRule(t *testing.T) {
	// Build: span.foo = "bar"
	attr := NewScopedAttribute(AttributeScopeSpan, false, "foo")
	binop := &BinaryOperation{
		Op:  OpEqual,
		LHS: attr,
		RHS: NewStaticString("bar"),
	}
	filter := NewSpansetFilterNode(newSpansetFilter(binop), NewSpanScanNode(nil, nil))

	rule := PredicatePushdownRule()
	result, changed := rule.Apply(filter)

	require.True(t, changed, "expected pushdown to fire")
	scan, ok := result.(*SpanScanNode)
	require.True(t, ok, "expected SpanScanNode after pushdown, got %T", result)
	require.NotEmpty(t, scan.Conditions)
}

// TestPredicatePushdownRule_Resource verifies that a resource-scope predicate
// is pushed into ResourceScanNode and the filter eliminated.
func TestPredicatePushdownRule_Resource(t *testing.T) {
	attr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	binop := &BinaryOperation{Op: OpEqual, LHS: attr, RHS: NewStaticString("frontend")}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(binop), traceScan)

	rule := PredicatePushdownRule()
	result, changed := rule.Apply(filter)

	require.True(t, changed, "expected pushdown to fire for resource-scope")
	// Filter is eliminated; root is now the TraceScan.
	_, ok := result.(*TraceScanNode)
	require.True(t, ok, "expected TraceScanNode after pushdown, got %T", result)
	// Condition pushed into ResourceScan, not SpanScan.
	require.NotEmpty(t, resourceScan.Conditions, "resource condition must be in ResourceScan")
	require.Empty(t, spanScan.Conditions, "SpanScan must have no conditions for a resource-scope predicate")
	require.Equal(t, "resource.service.name", resourceScan.Conditions[0].Attribute.String())
}

// TestPredicatePushdownRule_AND verifies that an AND of two span-scope predicates
// is pushed into SpanScanNode and the filter eliminated.
func TestPredicatePushdownRule_AND(t *testing.T) {
	a := NewScopedAttribute(AttributeScopeSpan, false, "http.method")
	b := NewScopedAttribute(AttributeScopeSpan, false, "http.status_code")
	binopA := &BinaryOperation{Op: OpEqual, LHS: a, RHS: NewStaticString("GET")}
	binopB := &BinaryOperation{Op: OpGreater, LHS: b, RHS: NewStaticInt(499)}
	and := &BinaryOperation{Op: OpAnd, LHS: binopA, RHS: binopB}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(and), traceScan)

	rule := PredicatePushdownRule()
	result, changed := rule.Apply(filter)

	require.True(t, changed, "expected AND pushdown to fire")
	_, ok := result.(*TraceScanNode)
	require.True(t, ok, "expected TraceScanNode after pushdown, got %T", result)
	require.Len(t, spanScan.Conditions, 2, "both span conditions must be pushed")
	require.True(t, traceScan.AllConditions, "AllConditions must be true for AND pushdown")
}

// TestPredicatePushdownRule_AND_MixedScope verifies that an AND with one
// span-scope and one resource-scope predicate routes each to the correct scan.
func TestPredicatePushdownRule_AND_MixedScope(t *testing.T) {
	spanAttr := NewScopedAttribute(AttributeScopeSpan, false, "name")
	resAttr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	binopSpan := &BinaryOperation{Op: OpEqual, LHS: spanAttr, RHS: NewStaticString("checkout")}
	binopRes := &BinaryOperation{Op: OpEqual, LHS: resAttr, RHS: NewStaticString("frontend")}
	and := &BinaryOperation{Op: OpAnd, LHS: binopSpan, RHS: binopRes}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(and), traceScan)

	rule := PredicatePushdownRule()
	result, changed := rule.Apply(filter)

	require.True(t, changed)
	_, ok := result.(*TraceScanNode)
	require.True(t, ok)
	require.Len(t, spanScan.Conditions, 1, "span condition routed to SpanScan")
	require.Len(t, resourceScan.Conditions, 1, "resource condition routed to ResourceScan")
	require.Equal(t, "span.name", spanScan.Conditions[0].Attribute.String())
	require.Equal(t, "resource.service.name", resourceScan.Conditions[0].Attribute.String())
}

// TestPredicatePushdownRule_UnscopedNotPushed verifies that a true unscoped user
// attribute is NOT handled by PredicatePushdownRule (it requires OR semantics).
func TestPredicatePushdownRule_UnscopedNotPushed(t *testing.T) {
	attr := NewAttribute("foo") // Scope=None, Intrinsic=None
	binop := &BinaryOperation{Op: OpEqual, LHS: attr, RHS: NewStaticString("bar")}
	filter := NewSpansetFilterNode(newSpansetFilter(binop), NewSpanScanNode(nil, nil))

	rule := PredicatePushdownRule()
	_, changed := rule.Apply(filter)

	require.False(t, changed, "unscoped user attribute must not be pushed by PredicatePushdownRule")
}

// TestPredicatePushdownRule_SkipsStructuralFilter ensures structural predicates
// (which have no simple per-span encoding) are NOT pushed down.
func TestPredicatePushdownRule_SkipsStructuralFilter(t *testing.T) {
	// An always-false static is not a simple per-span filter.
	filter := NewSpansetFilterNode(newSpansetFilter(NewStaticBool(false)), NewSpanScanNode(nil, nil))

	rule := PredicatePushdownRule()
	_, changed := rule.Apply(filter)

	require.False(t, changed, "expected no pushdown for non-simple filter")
}

// TestUnscopedAttributePushdownRule verifies that an unscoped user attribute
// gets OpNone fetch conditions added to both SpanScan and ResourceScan,
// with the SpansetFilterNode kept for predicate evaluation.
func TestUnscopedAttributePushdownRule(t *testing.T) {
	attr := NewAttribute("region") // Scope=None, Intrinsic=None
	binop := &BinaryOperation{Op: OpEqual, LHS: attr, RHS: NewStaticString("us-east")}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(binop), traceScan)

	rule := UnscopedAttributePushdownRule()
	result, changed := rule.Apply(filter)

	require.True(t, changed, "expected unscoped pushdown to fire")
	// Filter node is KEPT.
	_, ok := result.(*SpansetFilterNode)
	require.True(t, ok, "SpansetFilterNode must be kept, got %T", result)
	// OpNone fetch conditions added to both scan nodes.
	require.Len(t, spanScan.Conditions, 1)
	require.Equal(t, OpNone, spanScan.Conditions[0].Op, "SpanScan condition must be OpNone (fetch)")
	require.Len(t, resourceScan.Conditions, 1)
	require.Equal(t, OpNone, resourceScan.Conditions[0].Op, "ResourceScan condition must be OpNone (fetch)")
}

// TestUnscopedAttributePushdownRule_Idempotent verifies that applying the rule
// twice does not duplicate conditions (fixpoint safety).
func TestUnscopedAttributePushdownRule_Idempotent(t *testing.T) {
	attr := NewAttribute("region")
	binop := &BinaryOperation{Op: OpEqual, LHS: attr, RHS: NewStaticString("us-east")}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(binop), traceScan)

	rule := UnscopedAttributePushdownRule()
	filter2, _ := rule.Apply(filter)
	_, changed2 := rule.Apply(filter2)

	require.False(t, changed2, "second application must be a no-op")
	require.Len(t, spanScan.Conditions, 1, "no duplicate span conditions")
	require.Len(t, resourceScan.Conditions, 1, "no duplicate resource conditions")
}

// TestUnscopedAttributePushdownRule_SkipsDefiniteScope verifies that span-scope
// or resource-scope attributes are ignored by UnscopedAttributePushdownRule.
func TestUnscopedAttributePushdownRule_SkipsDefiniteScope(t *testing.T) {
	attr := NewScopedAttribute(AttributeScopeSpan, false, "foo")
	binop := &BinaryOperation{Op: OpEqual, LHS: attr, RHS: NewStaticString("bar")}
	filter := NewSpansetFilterNode(newSpansetFilter(binop), NewSpanScanNode(nil, nil))

	rule := UnscopedAttributePushdownRule()
	_, changed := rule.Apply(filter)

	require.False(t, changed, "span-scope attribute must be ignored by UnscopedAttributePushdownRule")
}

// TestGroupByHoistRule verifies that a GroupByNode inside a ProjectNode is
// hoisted above the ProjectNode, keeping the fetchTree on the ProjectNode.
func TestGroupByHoistRule(t *testing.T) {
	byAttr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resScan)
	groupNode := NewGroupByNode(byAttr, traceScan)
	fetchTree := NewTraceScanNode(nil, false, nil)
	col := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	projNode := NewProjectNode([]Attribute{col}, groupNode, fetchTree)

	rule := GroupByHoistRule()
	result, changed := rule.Apply(projNode)

	require.True(t, changed, "expected hoist to fire")
	group, ok := result.(*GroupByNode)
	require.True(t, ok, "expected GroupByNode at root after hoist, got %T", result)

	proj, ok := group.Children()[0].(*ProjectNode)
	require.True(t, ok, "expected ProjectNode as GroupByNode child, got %T", group.Children()[0])
	// fetchTree must be preserved on the ProjectNode.
	require.NotNil(t, proj.FetchTree(), "ProjectNode must retain the fetchTree after hoist")
	// The original scan tree must be the ProjectNode's child.
	_, ok = proj.Children()[0].(*TraceScanNode)
	require.True(t, ok, "expected TraceScanNode as ProjectNode child after hoist, got %T", proj.Children()[0])
}

// TestGroupByHoistRule_NoProjectNode verifies the rule is a no-op when the
// root is not a ProjectNode.
func TestGroupByHoistRule_NoProjectNode(t *testing.T) {
	byAttr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	groupNode := NewGroupByNode(byAttr, NewSpanScanNode(nil, nil))

	rule := GroupByHoistRule()
	_, changed := rule.Apply(groupNode)

	require.False(t, changed, "rule must not fire when root is not ProjectNode")
}

// TestGroupByHoistRule_NoGroupByChild verifies the rule is a no-op when the
// ProjectNode's child is not a GroupByNode.
func TestGroupByHoistRule_NoGroupByChild(t *testing.T) {
	col := NewScopedAttribute(AttributeScopeSpan, false, "foo")
	proj := NewProjectNode([]Attribute{col}, NewSpanScanNode(nil, nil), nil)

	rule := GroupByHoistRule()
	_, changed := rule.Apply(proj)

	require.False(t, changed, "rule must not fire when ProjectNode child is not GroupByNode")
}

// TestGroupByFetchRule_Resource verifies that a resource-scoped group-by
// attribute is pushed as an OpNone condition into the ResourceScanNode.
func TestGroupByFetchRule_Resource(t *testing.T) {
	byAttr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resScan)
	groupNode := NewGroupByNode(byAttr, traceScan)

	rule := GroupByFetchRule()
	result, changed := rule.Apply(groupNode)

	require.True(t, changed, "expected fetch rule to fire")
	_, ok := result.(*GroupByNode)
	require.True(t, ok)

	require.Len(t, resScan.Conditions, 1, "resource.service.name must be added to ResourceScan")
	require.Equal(t, OpNone, resScan.Conditions[0].Op, "condition must be OpNone (fetch-only)")
	require.Equal(t, "resource.service.name", resScan.Conditions[0].Attribute.String())
	// SpanScan must be untouched.
	require.Empty(t, spanScan.Conditions)
}

// TestGroupByFetchRule_Span verifies that a span-scoped group-by attribute is
// pushed into SpanScanNode.
func TestGroupByFetchRule_Span(t *testing.T) {
	byAttr := NewScopedAttribute(AttributeScopeSpan, false, "http.method")
	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resScan)
	groupNode := NewGroupByNode(byAttr, traceScan)

	rule := GroupByFetchRule()
	_, changed := rule.Apply(groupNode)

	require.True(t, changed)
	require.Len(t, spanScan.Conditions, 1, "span.http.method must be added to SpanScan")
	require.Equal(t, OpNone, spanScan.Conditions[0].Op)
	require.Empty(t, resScan.Conditions)
}

// TestGroupByFetchRule_Unscoped verifies that an unscoped group-by attribute is
// pushed to both SpanScan and ResourceScan (OR semantics).
func TestGroupByFetchRule_Unscoped(t *testing.T) {
	byAttr := NewAttribute("region") // Scope=None, Intrinsic=None
	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resScan)
	groupNode := NewGroupByNode(byAttr, traceScan)

	rule := GroupByFetchRule()
	_, changed := rule.Apply(groupNode)

	require.True(t, changed)
	require.Len(t, spanScan.Conditions, 1, "unscoped attr must be added to SpanScan")
	require.Equal(t, OpNone, spanScan.Conditions[0].Op)
	require.Len(t, resScan.Conditions, 1, "unscoped attr must be added to ResourceScan")
	require.Equal(t, OpNone, resScan.Conditions[0].Op)
}

// TestGroupByFetchRule_Idempotent verifies that applying the rule twice does
// not duplicate conditions (fixpoint safety).
func TestGroupByFetchRule_Idempotent(t *testing.T) {
	byAttr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resScan)
	groupNode := NewGroupByNode(byAttr, traceScan)

	rule := GroupByFetchRule()
	result, _ := rule.Apply(groupNode)
	_, changed2 := rule.Apply(result)

	require.False(t, changed2, "second application must be a no-op")
	require.Len(t, resScan.Conditions, 1, "no duplicate conditions")
}

// TestGroupByFetchRule_AllConditionsUnchanged verifies that GroupByFetchRule
// does not set AllConditions to false. Unlike the legacy engine (which forced
// AllConditions=false to avoid requiring the fetch-only group-by attribute as
// a filter), the plan-based approach keeps filter and fetch-only conditions
// separate, so AllConditions from the filter pushdown remains valid.
func TestGroupByFetchRule_AllConditionsUnchanged(t *testing.T) {
	byAttr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	spanScan := NewSpanScanNode(
		[]Condition{{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"), Op: OpEqual}},
		nil,
	)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, true, resScan) // AllConditions=true from filter pushdown
	groupNode := NewGroupByNode(byAttr, traceScan)

	rule := GroupByFetchRule()
	rule.Apply(groupNode)

	require.True(t, traceScan.AllConditions, "GroupByFetchRule must not change AllConditions on TraceScanNode")
}

// TestConditionMergeRule verifies that two adjacent SpanScanNodes merge into one.
func TestConditionMergeRule(t *testing.T) {
	cond1 := Condition{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "status")}
	cond2 := Condition{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "http.method")}
	inner := NewSpanScanNode([]Condition{cond1}, nil)
	outer := NewSpanScanNode([]Condition{cond2}, inner)

	rule := ConditionMergeRule()
	result, changed := rule.Apply(outer)

	require.True(t, changed)
	merged, ok := result.(*SpanScanNode)
	require.True(t, ok)
	require.Len(t, merged.Conditions, 2)
	require.Nil(t, merged.Children())
}

// TestConditionMergeRule_ResourceMerge verifies merging of ResourceScanNodes.
func TestConditionMergeRule_ResourceMerge(t *testing.T) {
	cond1 := Condition{Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name")}
	cond2 := Condition{Attribute: NewScopedAttribute(AttributeScopeResource, false, "cluster")}
	inner := NewResourceScanNode([]Condition{cond1}, nil)
	outer := NewResourceScanNode([]Condition{cond2}, inner)

	rule := ConditionMergeRule()
	result, changed := rule.Apply(outer)

	require.True(t, changed)
	merged, ok := result.(*ResourceScanNode)
	require.True(t, ok)
	require.Len(t, merged.Conditions, 2)
}

// TestSecondPassEliminatorRule verifies that a ProjectNode is removed when all
// its columns are already present in the first-pass scan nodes.
func TestSecondPassEliminatorRule(t *testing.T) {
	col := NewScopedAttribute(AttributeScopeSpan, false, "region")
	scan := NewSpanScanNode([]Condition{{Attribute: col}}, nil)
	proj := NewProjectNode([]Attribute{col}, scan, nil)

	rule := SecondPassEliminatorRule()
	result, changed := rule.Apply(proj)

	require.True(t, changed)
	_, ok := result.(*SpanScanNode)
	require.True(t, ok, "expected SpanScanNode after elimination, got %T", result)
}

// TestSecondPassEliminatorRule_Kept verifies that a ProjectNode is kept when
// its columns are NOT present in the scan tree.
func TestSecondPassEliminatorRule_Kept(t *testing.T) {
	col := NewScopedAttribute(AttributeScopeSpan, false, "region")
	// Scan does NOT have the column.
	scan := NewSpanScanNode(nil, nil)
	proj := NewProjectNode([]Attribute{col}, scan, nil)

	rule := SecondPassEliminatorRule()
	_, changed := rule.Apply(proj)

	require.False(t, changed)
}

// newScanTree returns a full TraceScan→ResourceScan→InstrScan→SpanScan stack
// with the given pre-populated SpanScan and ResourceScan nodes.
func newScanTree(spanScan *SpanScanNode, resourceScan *ResourceScanNode) (*TraceScanNode, *SpanScanNode, *ResourceScanNode) {
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan2 := NewResourceScanNode(resourceScan.Conditions, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan2)
	return traceScan, spanScan, resourceScan2
}

// TestOrPredicatePushdownRule_SpanOnly verifies that a pure span-scope OR
// expression gets real-operator conditions pushed into SpanScan so parquet can
// skip non-matching row groups, while the SpansetFilterNode is kept for
// in-memory evaluation.
func TestOrPredicatePushdownRule_SpanOnly(t *testing.T) {
	a := NewScopedAttribute(AttributeScopeSpan, false, "http.method")
	b := NewScopedAttribute(AttributeScopeSpan, false, "http.status_code")
	or := &BinaryOperation{
		Op:  OpOr,
		LHS: &BinaryOperation{Op: OpEqual, LHS: a, RHS: NewStaticString("GET")},
		RHS: &BinaryOperation{Op: OpGreater, LHS: b, RHS: NewStaticInt(499)},
	}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(or), traceScan)

	rule := OrPredicatePushdownRule()
	result, changed := rule.Apply(filter)

	require.True(t, changed, "expected OR pushdown to fire")
	// Filter node is KEPT.
	_, ok := result.(*SpansetFilterNode)
	require.True(t, ok, "SpansetFilterNode must be kept for OR, got %T", result)
	// Both span conditions pushed with their real operators (not OpNone) so
	// the parquet layer can skip row groups where neither predicate can match.
	require.Len(t, spanScan.Conditions, 2, "both span attrs must be fetched")
	condByAttr := make(map[string]Condition, len(spanScan.Conditions))
	for _, c := range spanScan.Conditions {
		condByAttr[c.Attribute.String()] = c
	}
	methodCond, ok := condByAttr["span.http.method"]
	require.True(t, ok, "span.http.method condition expected")
	require.Equal(t, OpEqual, methodCond.Op, "http.method must use real OpEqual")
	statusCond, ok := condByAttr["span.http.status_code"]
	require.True(t, ok, "span.http.status_code condition expected")
	require.Equal(t, OpGreater, statusCond.Op, "http.status_code must use real OpGreater")
	// ResourceScan untouched.
	require.Empty(t, resourceScan.Conditions)
}

// TestOrPredicatePushdownRule_MixedScope verifies that a mixed resource+span OR
// routes resource attrs to ResourceScan and span attrs to SpanScan, each with
// its real operator.
func TestOrPredicatePushdownRule_MixedScope(t *testing.T) {
	spanAttr := NewScopedAttribute(AttributeScopeSpan, false, "name")
	resAttr := NewScopedAttribute(AttributeScopeResource, false, "service.name")
	or := &BinaryOperation{
		Op:  OpOr,
		LHS: &BinaryOperation{Op: OpEqual, LHS: resAttr, RHS: NewStaticString("frontend")},
		RHS: &BinaryOperation{Op: OpEqual, LHS: spanAttr, RHS: NewStaticString("checkout")},
	}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(or), traceScan)

	rule := OrPredicatePushdownRule()
	result, changed := rule.Apply(filter)

	require.True(t, changed)
	_, ok := result.(*SpansetFilterNode)
	require.True(t, ok, "SpansetFilterNode must be kept")
	require.Len(t, spanScan.Conditions, 1)
	require.Equal(t, OpEqual, spanScan.Conditions[0].Op, "span.name must use real OpEqual")
	require.Equal(t, "span.name", spanScan.Conditions[0].Attribute.String())
	require.Len(t, resourceScan.Conditions, 1)
	require.Equal(t, OpEqual, resourceScan.Conditions[0].Op, "resource.service.name must use real OpEqual")
	require.Equal(t, "resource.service.name", resourceScan.Conditions[0].Attribute.String())
}

// TestOrPredicatePushdownRule_Idempotent verifies that applying the rule twice
// does not duplicate conditions (fixpoint safety).
// span.foo=a || span.foo=b uses the same attribute with different values, so
// resolveOrConditions collapses it to a single OpNone fetch condition.
func TestOrPredicatePushdownRule_Idempotent(t *testing.T) {
	spanAttr := NewScopedAttribute(AttributeScopeSpan, false, "foo")
	or := &BinaryOperation{
		Op:  OpOr,
		LHS: &BinaryOperation{Op: OpEqual, LHS: spanAttr, RHS: NewStaticString("a")},
		RHS: &BinaryOperation{Op: OpEqual, LHS: spanAttr, RHS: NewStaticString("b")},
	}

	spanScan := NewSpanScanNode(nil, nil)
	instrScan := NewInstrumentationScopeScanNode(nil, spanScan)
	resourceScan := NewResourceScanNode(nil, instrScan)
	traceScan := NewTraceScanNode(nil, false, resourceScan)
	filter := NewSpansetFilterNode(newSpansetFilter(or), traceScan)

	rule := OrPredicatePushdownRule()
	result, _ := rule.Apply(filter)
	_, changed2 := rule.Apply(result)

	require.False(t, changed2, "second application must be a no-op")
	require.Len(t, spanScan.Conditions, 1, "no duplicate conditions")
	// Same attribute with different values → conflict → collapsed to OpNone.
	require.Equal(t, OpNone, spanScan.Conditions[0].Op, "same-attr conflict must use OpNone")
}

// TestOrPredicatePushdownRule_NoOrSkips verifies that the rule does not fire
// on expressions that contain no OR operator.
func TestOrPredicatePushdownRule_NoOrSkips(t *testing.T) {
	a := NewScopedAttribute(AttributeScopeSpan, false, "foo")
	binop := &BinaryOperation{Op: OpEqual, LHS: a, RHS: NewStaticString("bar")}
	filter := NewSpansetFilterNode(newSpansetFilter(binop), NewSpanScanNode(nil, nil))

	rule := OrPredicatePushdownRule()
	_, changed := rule.Apply(filter)

	require.False(t, changed, "non-OR expression must not trigger OrPredicatePushdownRule")
}

// TestOrPredicatePushdownRule_TraceLevelIntrinsic verifies that trace-level
// intrinsics inside OR expressions are routed to TraceScanNode, not SpanScanNode.
// Regression test for: { rootServiceName = `x` && (status = error || span.http.status_code = 500) }
func TestOrPredicatePushdownRule_TraceLevelIntrinsic(t *testing.T) {
	// Simulate: status = error || rootServiceName = `x`
	// status is span-scope; rootServiceName is trace-scope (IntrinsicLevel == AttributeScopeNone)
	statusAttr := NewIntrinsic(IntrinsicStatus)
	rootSvcAttr := NewIntrinsic(IntrinsicTraceRootService)

	orExpr := &BinaryOperation{
		Op:  OpOr,
		LHS: &BinaryOperation{Op: OpEqual, LHS: statusAttr, RHS: NewStaticString("error")},
		RHS: &BinaryOperation{Op: OpEqual, LHS: rootSvcAttr, RHS: NewStaticString("svc")},
	}

	spanScan := NewSpanScanNode(nil, nil)
	instr := NewInstrumentationScopeScanNode(nil, spanScan)
	res := NewResourceScanNode(nil, instr)
	traceScan := NewTraceScanNode(nil, false, res)
	filter := NewSpansetFilterNode(newSpansetFilter(orExpr), traceScan)

	rule := OrPredicatePushdownRule()
	_, changed := rule.Apply(filter)

	require.True(t, changed, "rule must fire for OR expression with trace-level intrinsic")

	// rootServiceName must land on TraceScanNode
	foundOnTrace := false
	for _, c := range traceScan.Conditions {
		if c.Attribute.Intrinsic == IntrinsicTraceRootService {
			foundOnTrace = true
		}
	}
	require.True(t, foundOnTrace, "rootServiceName must be pushed to TraceScanNode")

	// rootServiceName must NOT land on SpanScanNode
	for _, c := range spanScan.Conditions {
		require.NotEqual(t, IntrinsicTraceRootService, c.Attribute.Intrinsic,
			"rootServiceName must not be pushed to SpanScanNode")
	}
}

// TestFetchTreeDeduplicationRule verifies that attributes already present in the
// drive tree are removed from the fetch tree.
func TestFetchTreeDeduplicationRule(t *testing.T) {
	durationAttr := NewIntrinsic(IntrinsicDuration)

	// Drive tree: SpanScanNode already has duration as a filter condition.
	driveSpan := NewSpanScanNode([]Condition{{Attribute: durationAttr, Op: OpGreater}}, nil)
	driveInstr := NewInstrumentationScopeScanNode(nil, driveSpan)
	driveRes := NewResourceScanNode(nil, driveInstr)
	driveTrace := NewTraceScanNode(nil, true, driveRes)

	// Fetch tree: SpanScanNode has duration as a fetch-only condition (redundant).
	spanID := NewIntrinsic(IntrinsicSpanID)
	fetchSpan := NewSpanScanNode([]Condition{
		{Attribute: spanID, Op: OpNone},
		{Attribute: durationAttr, Op: OpNone}, // should be removed
	}, nil)
	fetchInstr := NewInstrumentationScopeScanNode(nil, fetchSpan)
	fetchRes := NewResourceScanNode(nil, fetchInstr)
	fetchTrace := NewTraceScanNode(nil, false, fetchRes)

	proj := NewProjectNode([]Attribute{durationAttr, spanID}, driveTrace, fetchTrace)

	rule := FetchTreeDeduplicationRule()
	_, changed := rule.Apply(proj)

	require.True(t, changed, "expected fetch-tree-dedup to fire")
	require.Len(t, fetchSpan.Conditions, 1, "duration should have been removed from fetch tree")
	require.Equal(t, spanID, fetchSpan.Conditions[0].Attribute)
}

// TestFetchTreeDeduplicationRule_NoFetchTree verifies the rule skips nodes without a fetch tree.
func TestFetchTreeDeduplicationRule_NoFetchTree(t *testing.T) {
	driveSpan := NewSpanScanNode([]Condition{{Attribute: NewIntrinsic(IntrinsicDuration), Op: OpGreater}}, nil)
	proj := NewProjectNode([]Attribute{NewIntrinsic(IntrinsicDuration)}, driveSpan, nil)

	rule := FetchTreeDeduplicationRule()
	_, changed := rule.Apply(proj)

	require.False(t, changed, "rule must not fire when there is no fetch tree")
}

// TestFetchTreeDeduplicationRule_NoDriveConditions verifies the rule skips when the drive tree
// has no conditions (nothing to deduplicate against).
func TestFetchTreeDeduplicationRule_NoDriveConditions(t *testing.T) {
	driveSpan := NewSpanScanNode(nil, nil)
	fetchSpan := NewSpanScanNode([]Condition{{Attribute: NewIntrinsic(IntrinsicDuration), Op: OpNone}}, nil)
	fetchTrace := NewTraceScanNode(nil, false, fetchSpan)
	proj := NewProjectNode([]Attribute{NewIntrinsic(IntrinsicDuration)}, driveSpan, fetchTrace)

	rule := FetchTreeDeduplicationRule()
	_, changed := rule.Apply(proj)

	require.False(t, changed, "rule must not fire when drive tree has no conditions")
	require.Len(t, fetchSpan.Conditions, 1, "fetch tree must be unchanged")
}

// TestFetchTreeDeduplicationRule_Integrated verifies end-to-end that BuildSearchTracePlan
// does not include duration in the fetch tree when it is already filtered in the drive tree.
func TestFetchTreeDeduplicationRule_Integrated(t *testing.T) {
	// { duration > 200ms } — duration is pushed into the drive SpanScanNode.
	expr, err := Parse(`{ duration > 200ms }`)
	require.NoError(t, err)

	plan, err := BuildSearchTracePlan(expr)
	require.NoError(t, err)

	proj, ok := plan.(*ProjectNode)
	require.True(t, ok, "expected ProjectNode at root")

	fetchSpan := firstSpanScanNode(proj.FetchTree())
	require.NotNil(t, fetchSpan)

	durationAttr := NewIntrinsic(IntrinsicDuration)
	for _, c := range fetchSpan.Conditions {
		require.NotEqual(t, durationAttr, c.Attribute,
			"duration must not appear in fetch tree when already filtered in drive tree")
	}
}
