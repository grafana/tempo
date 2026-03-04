package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptimizerFixpoint(t *testing.T) {
	callCount := 0
	// Rule that fires once on the first SpanScanNode it sees, then becomes a no-op.
	rule := FuncRule("test-rule", func(n PlanNode) (PlanNode, bool) {
		if _, ok := n.(*SpanScanNode); ok && callCount == 0 {
			callCount++
			return &SpanScanNode{Conditions: []Condition{{}}}, true
		}
		return n, false
	})

	rs := NewRuleSet(rule)
	plan := NewTraceScanNode(nil, false, NewSpanScanNode(nil, nil))
	result := rs.Optimize(plan)

	// Optimizer ran until fixpoint (no more changes after the first rewrite).
	require.NotNil(t, result)
	// The SpanScanNode got a condition added by the rule.
	trace, ok := result.(*TraceScanNode)
	require.True(t, ok)
	span, ok := trace.Children()[0].(*SpanScanNode)
	require.True(t, ok)
	require.Len(t, span.Conditions, 1)
}

func TestOptimizerNoRuleFires(t *testing.T) {
	rs := NewRuleSet()
	plan := NewTraceScanNode(nil, false, NewSpanScanNode(nil, nil))
	result := rs.Optimize(plan)
	// No rules → tree unchanged
	require.Equal(t, plan, result)
}

func TestOptimizerRewritesChildren(t *testing.T) {
	// Rule that adds a condition to SpanScanNode
	rule := FuncRule("add-cond", func(n PlanNode) (PlanNode, bool) {
		if s, ok := n.(*SpanScanNode); ok && len(s.Conditions) == 0 {
			return &SpanScanNode{Conditions: []Condition{{}}}, true
		}
		return n, false
	})

	rs := NewRuleSet(rule)
	plan := NewTraceScanNode(nil, false, NewSpanScanNode(nil, nil))
	result := rs.Optimize(plan)

	trace := result.(*TraceScanNode)
	span := trace.Children()[0].(*SpanScanNode)
	require.Len(t, span.Conditions, 1)
}
