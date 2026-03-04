package traceql

import (
	"fmt"

	"github.com/grafana/tempo/pkg/tempopb"
)

// BuildPlan converts a parsed TraceQL *RootExpr into a logical plan tree,
// populates metrics nodes with time parameters from req, and runs the default
// optimizer passes before returning. req may be nil for non-metrics queries.
func BuildPlan(expr *RootExpr, req *tempopb.QueryRangeRequest) (PlanNode, error) {
	b := &planBuilder{req: req}
	plan, err := b.build(expr)
	if err != nil {
		return nil, err
	}
	return DefaultRuleSet().Optimize(plan), nil
}

// DefaultRuleSet returns the built-in optimization rules.
func DefaultRuleSet() *RuleSet {
	return NewRuleSet(
		PredicatePushdownRule(),
		OrPredicatePushdownRule(),
		UnscopedAttributePushdownRule(),
		ConditionMergeRule(),
		SecondPassEliminatorRule(),
	)
}

type planBuilder struct {
	req *tempopb.QueryRangeRequest
}

// build converts a *RootExpr to a plan tree bottom-up.
func (b *planBuilder) build(expr *RootExpr) (PlanNode, error) {
	// Start with the default scan tree.
	base := b.buildBaseScanTree()

	// Build the engine node layer from the pipeline.
	plan, err := b.buildPipeline(expr.Pipeline, base)
	if err != nil {
		return nil, err
	}

	// Wrap with the metrics node if present.
	if expr.MetricsPipeline != nil {
		plan, err = b.buildMetricsPipeline(expr.MetricsPipeline, plan)
		if err != nil {
			return nil, err
		}
	}

	return plan, nil
}

// buildBaseScanTree builds the default four-level scan tree:
//
//	TraceScanNode → ResourceScanNode → InstrumentationScopeScanNode → SpanScanNode
//
// Conditions are empty at this stage; the optimizer fills them via predicate pushdown.
func (b *planBuilder) buildBaseScanTree() PlanNode {
	span := NewSpanScanNode(nil, nil)
	instr := NewInstrumentationScopeScanNode(nil, span)
	res := NewResourceScanNode(nil, instr)
	trace := NewTraceScanNode(nil, false, res)
	return trace
}

// buildPipeline walks the pipeline elements and wraps the base plan with engine nodes.
func (b *planBuilder) buildPipeline(p Pipeline, base PlanNode) (PlanNode, error) {
	plan := base

	// Track whether we've seen a GroupByNode so we know where to insert ProjectNode.
	var groupNode *GroupByNode

	for _, elem := range p.Elements {
		switch e := elem.(type) {
		case *SpansetFilter:
			plan = NewSpansetFilterNode(e, plan)

		case GroupOperation:
			g := NewGroupByNode(e.Expression, plan)
			groupNode = g
			plan = g

		case CoalesceOperation:
			plan = NewCoalesceNode(plan)

		case SelectOperation:
			// SelectOperation triggers a second-pass fetch.
			// ProjectNode must be placed BELOW GroupByNode (so grouping happens after the fetch).
			if groupNode != nil {
				// Insert ProjectNode between groupNode and its current child.
				proj := NewProjectNode(e.attrs, groupNode.child)
				plan = groupNode.WithChild(proj)
				groupNode = plan.(*GroupByNode) // keep tracking the (new) group node
			} else {
				plan = NewProjectNode(e.attrs, plan)
			}

		case SpansetOperation:
			base := b.buildSpansetOperationBase(e)
			plan = NewSpansetRelationNode(e, base)

		case Pipeline:
			// Sub-pipeline: recurse.
			var err error
			plan, err = b.buildPipeline(e, plan)
			if err != nil {
				return nil, err
			}
		}
	}

	return plan, nil
}

// buildSpansetExpression converts a SpansetExpression (which may be a Pipeline,
// SpansetFilter, or SpansetOperation) into a plan tree.
func (b *planBuilder) buildSpansetExpression(e SpansetExpression) (PlanNode, error) {
	switch expr := e.(type) {
	case Pipeline:
		return b.buildPipeline(expr, b.buildBaseScanTree())
	case *SpansetFilter:
		return b.buildPipeline(newPipeline(expr), b.buildBaseScanTree())
	case SpansetOperation:
		base := b.buildSpansetOperationBase(expr)
		return NewSpansetRelationNode(expr, base), nil
	default:
		return nil, fmt.Errorf("plan_builder: unsupported spanset expression type %T", e)
	}
}

// buildMetricsPipeline wraps a plan with the appropriate metrics aggregation node.
func (b *planBuilder) buildMetricsPipeline(m firstStageElement, child PlanNode) (PlanNode, error) {
	agg, ok := m.(*MetricsAggregate)
	if !ok {
		return nil, fmt.Errorf("plan_builder: unsupported metrics pipeline type %T", m)
	}

	switch agg.op {
	case metricsAggregateRate:
		if b.req != nil {
			return newRateNodeFromReq(agg.by, b.req, child), nil
		}
		return NewRateNode(agg.by, child), nil
	case metricsAggregateCountOverTime:
		if b.req != nil {
			return newCountOverTimeNodeFromReq(agg.by, b.req, child), nil
		}
		return NewCountOverTimeNode(agg.by, child), nil
	default:
		// Unrecognised metrics op — wrap in RateNode as a safe default.
		if b.req != nil {
			return newRateNodeFromReq(agg.by, b.req, child), nil
		}
		return NewRateNode(agg.by, child), nil
	}
}

// buildSpansetOperationBase builds a single scan tree whose conditions cover
// every attribute needed to evaluate op — from both LHS and RHS sides.
// AllConditions is always false because a single span cannot satisfy predicates
// from both sides of a structural / union relationship simultaneously.
func (b *planBuilder) buildSpansetOperationBase(op SpansetOperation) PlanNode {
	req := &FetchSpansRequest{}
	op.extractConditions(req)

	var spanConds, resConds, traceConds []Condition
	for _, c := range req.Conditions {
		switch {
		case c.Attribute.Intrinsic != IntrinsicNone:
			traceConds = append(traceConds, c)
		case c.Attribute.Scope == AttributeScopeResource:
			resConds = append(resConds, c)
		default: // span-scope or unscoped
			spanConds = append(spanConds, c)
		}
	}

	span := NewSpanScanNode(spanConds, nil)
	instr := NewInstrumentationScopeScanNode(nil, span)
	res := NewResourceScanNode(resConds, instr)
	// AllConditions=false: a span need only satisfy one side's conditions.
	return NewTraceScanNode(traceConds, false, res)
}
