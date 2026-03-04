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
	proj := NewProjectNode([]Attribute{col}, scan)

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
	proj := NewProjectNode([]Attribute{col}, scan)

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
// expression gets OpNone fetch conditions pushed into SpanScan, while the
// SpansetFilterNode is kept for in-memory evaluation.
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
	// Both span conditions pushed as OpNone.
	require.Len(t, spanScan.Conditions, 2, "both span attrs must be fetched")
	for _, c := range spanScan.Conditions {
		require.Equal(t, OpNone, c.Op, "OR pushdown must use OpNone")
	}
	// ResourceScan untouched.
	require.Empty(t, resourceScan.Conditions)
}

// TestOrPredicatePushdownRule_MixedScope verifies that a mixed resource+span OR
// routes resource attrs to ResourceScan and span attrs to SpanScan, all OpNone.
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
	require.Equal(t, OpNone, spanScan.Conditions[0].Op)
	require.Equal(t, "span.name", spanScan.Conditions[0].Attribute.String())
	require.Len(t, resourceScan.Conditions, 1)
	require.Equal(t, OpNone, resourceScan.Conditions[0].Op)
	require.Equal(t, "resource.service.name", resourceScan.Conditions[0].Attribute.String())
}

// TestOrPredicatePushdownRule_Idempotent verifies that applying the rule twice
// does not duplicate conditions (fixpoint safety).
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
