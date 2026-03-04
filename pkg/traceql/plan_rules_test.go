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

// TestPredicatePushdownRule_SkipsStructuralFilter ensures structural predicates
// (which have no simple per-span encoding) are NOT pushed down.
func TestPredicatePushdownRule_SkipsStructuralFilter(t *testing.T) {
	// An always-false static is not a simple per-span filter.
	filter := NewSpansetFilterNode(newSpansetFilter(NewStaticBool(false)), NewSpanScanNode(nil, nil))

	rule := PredicatePushdownRule()
	_, changed := rule.Apply(filter)

	require.False(t, changed, "expected no pushdown for non-simple filter")
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
