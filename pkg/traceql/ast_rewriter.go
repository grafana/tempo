package traceql

// Rewriter modifies a TraceQL AST
type Rewriter interface {
	RewriteRoot(*RootExpr) *RootExpr
}

type rewriteChain []Rewriter

func (c rewriteChain) RewriteRoot(r *RootExpr) *RootExpr {
	for _, rw := range c {
		r = rw.RewriteRoot(r)
	}
	return r
}

func newOrToInRewriter() *fieldExpressionRewriter {
	return &fieldExpressionRewriter{rewriteFn: rewriteOrToIn}
}

type fieldExpressionRewriter struct {
	rewriteFn func(e FieldExpression) FieldExpression
}

func (f *fieldExpressionRewriter) RewriteRoot(r *RootExpr) *RootExpr {
	if r == nil {
		return r
	}
	return &RootExpr{
		Pipeline:           f.rewritePipeline(r.Pipeline),
		MetricsPipeline:    r.MetricsPipeline,
		MetricsSecondStage: r.MetricsSecondStage,
		Hints:              r.Hints,
	}
}

func (f *fieldExpressionRewriter) rewritePipeline(p Pipeline) Pipeline {
	elements := make([]PipelineElement, 0, len(p.Elements))
	for _, el := range p.Elements {
		switch el := el.(type) {
		case *SpansetFilter:
			if el == nil {
				continue
			}
			elements = append(elements, newSpansetFilter(f.rewriteFieldExpression(el.Expression)))
		case ScalarFilter:
			elements = append(elements, newScalarFilter(el.Op, f.rewriteScalarExpression(el.LHS), f.rewriteScalarExpression(el.RHS)))
		case GroupOperation:
			elements = append(elements, newGroupOperation(f.rewriteFieldExpression(el.Expression)))
		case Pipeline:
			elements = append(elements, f.rewritePipeline(el))
		default:
			elements = append(elements, el)
		}
	}
	return newPipeline(elements...)
}

func (f *fieldExpressionRewriter) rewriteScalarExpression(se ScalarExpression) ScalarExpression {
	switch se := se.(type) {
	case Pipeline:
		return f.rewritePipeline(se)
	case ScalarOperation:
		return newScalarOperation(se.Op, f.rewriteScalarExpression(se.LHS), f.rewriteScalarExpression(se.RHS))
	case Aggregate:
		return newAggregate(se.op, f.rewriteFieldExpression(se.e))
	case Static:
		s := f.rewriteFieldExpression(se)
		if s, ok := s.(Static); ok {
			return s
		}
	}
	return se
}

func (f *fieldExpressionRewriter) rewriteFieldExpression(e FieldExpression) FieldExpression {
	switch e := e.(type) {
	case *BinaryOperation:
		e.LHS = f.rewriteFieldExpression(e.LHS)
		e.RHS = f.rewriteFieldExpression(e.RHS)
	case *UnaryOperation:
		e.Expression = f.rewriteFieldExpression(e.Expression)
	}
	return f.rewriteFn(e)
}

func rewriteOrToIn(e FieldExpression) FieldExpression {
	opOr, ok := e.(*BinaryOperation)
	if !ok || opOr.Op != OpOr {
		return e
	}

	// LHS and RHS operands
	opLHS, ok := opOr.LHS.(*BinaryOperation)
	if !ok || (opLHS.Op != OpIn && opLHS.Op != OpEqual) {
		return e
	}
	opRHS, ok := opOr.RHS.(*BinaryOperation)
	if !ok || (opLHS.Op != OpIn && opLHS.Op != OpEqual) {
		return e
	}

	lhsAttr, lhsVal, ok := operandsFromEqualOrIn(opLHS)
	if !ok {
		return e
	}
	rhsAttr, rhsVal, ok := operandsFromEqualOrIn(opRHS)
	if !ok || lhsAttr != rhsAttr {
		return e
	}

	array, ok := lhsVal.append(rhsVal)
	if !ok {
		return e
	}

	return &BinaryOperation{
		Op:  OpIn,
		LHS: lhsAttr,
		RHS: array,
	}
}

func operandsFromEqualOrIn(exp FieldExpression) (Attribute, Static, bool) {
	binOp, ok := exp.(*BinaryOperation)
	if !ok || (binOp.Op != OpIn && binOp.Op != OpEqual) {
		return Attribute{}, StaticNil, false
	}

	rhs := binOp.RHS

	attr, ok := binOp.LHS.(Attribute)
	if !ok {
		attr, ok = binOp.RHS.(Attribute)
		if !ok {
			return Attribute{}, StaticNil, false
		}
		rhs = binOp.LHS
	}

	val, ok := rhs.(Static)
	if !ok {
		return Attribute{}, StaticNil, false
	}

	return attr, val, true
}

// ApplyDefaultRewrites applies all default rewrite passes.
// TODO remove
func ApplyDefaultRewrites(r *RootExpr) *RootExpr {
	return r
}
