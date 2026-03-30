package traceql

import "slices"

func ApplyDefaultASTRewrites(r *RootExpr) *RootExpr {
	chain := rewriteChain{
		newBinaryOpToArrayOpRewriter(),
	}
	return chain.RewriteRoot(r)
}

// ASTRewriter modifies a TraceQL AST without changing its meaning.
type ASTRewriter interface {
	RewriteRoot(*RootExpr) *RootExpr
}

// rewriteChain is an ASTRewriter consisting of multiple ASTRewriter instances.
type rewriteChain []ASTRewriter

func (c rewriteChain) RewriteRoot(r *RootExpr) *RootExpr {
	for _, rw := range c {
		r = rw.RewriteRoot(r)
	}
	return r
}

// newBinaryOpToArrayOpRewriter creates a ASTRewriter that rewrites certain BinaryOperation expressions to
// equivalent array operations in the TraceQL AST. It handles the following cases:
// - { .a =  "a" || .a =  "b" } => { .a IN  ["a", "b"] }
// - { .a != "a" && .a != "b" } => { .a NOT IN ["a", "b"] }
// - { .a =~ "a" || .a =~ "b" } => { .a MATCH ANY ["a", "b"] }
// - { .a !~ "a" && .a !~ "b" } => { .a MATCH NONE ["a", "b"] }
func newBinaryOpToArrayOpRewriter() *fieldExpressionRewriter {
	return &fieldExpressionRewriter{
		rewriteFunctions: []fieldExpressionRewriteFn{
			rewriteOrToIn,
			rewriteAndToNotIn,
			rewriteOrToMatchAny,
			rewriteAndToMatchNone,
		},
	}
}

// fieldExpressionRewriteFn is a function that rewrites single FieldExpression implementations should not descent into subexpressions.
type fieldExpressionRewriteFn func(FieldExpression) (FieldExpression, int)

// fieldExpressionRewriter is an ASTRewriter that descends into PipelineElement. The rewriter traverses all FieldExpression
// in Post Order and applies available fieldExpressionRewriteFn functions.
// fieldExpressionRewriter is intended to be used as helper to implement new ASTRewriter implementations that modify FieldExpressions.
type fieldExpressionRewriter struct {
	rewriteFunctions []fieldExpressionRewriteFn
}

func (f *fieldExpressionRewriter) RewriteRoot(r *RootExpr) *RootExpr {
	if r == nil {
		return r
	}

	pipeline, rwCount := f.rewritePipeline(r.Pipeline)

	return &RootExpr{
		Pipeline:           pipeline,
		MetricsPipeline:    r.MetricsPipeline,
		MetricsSecondStage: r.MetricsSecondStage,
		Hints:              r.Hints,
		OptimizationCount:  r.OptimizationCount + rwCount,
	}
}

func (f *fieldExpressionRewriter) rewritePipeline(p Pipeline) (Pipeline, int) {
	var rwCount int

	elements := make([]PipelineElement, 0, len(p.Elements))
	for _, elem := range p.Elements {
		switch elem := elem.(type) {
		case *SpansetFilter:
			if elem == nil {
				continue
			}
			fe, n := f.rewriteFieldExpression(elem.Expression)
			elements = append(elements, newSpansetFilter(fe))
			rwCount += n
		case ScalarFilter:
			lhs, n := f.rewriteScalarExpression(elem.LHS)
			rwCount += n
			rhs, n := f.rewriteScalarExpression(elem.RHS)
			rwCount += n
			elements = append(elements, newScalarFilter(elem.Op, lhs, rhs))
		case GroupOperation:
			fe, n := f.rewriteFieldExpression(elem.Expression)
			rwCount += n
			elements = append(elements, newGroupOperation(fe))
		case SpansetOperation:
			lhs, n := f.rewriteSpansetExpression(elem.LHS)
			rwCount += n
			rhs, n := f.rewriteSpansetExpression(elem.RHS)
			rwCount += n
			elements = append(elements, newSpansetOperation(elem.Op, lhs, rhs))
		case Pipeline:
			fe, n := f.rewritePipeline(elem)
			rwCount += n
			elements = append(elements, fe)
		default:
			elements = append(elements, elem)
		}
	}

	return newPipeline(elements...), rwCount
}

func (f *fieldExpressionRewriter) rewriteSpansetExpression(se SpansetExpression) (SpansetExpression, int) {
	switch se := se.(type) {
	case *SpansetFilter:
		if se == nil {
			return se, 0
		}
		fe, n := f.rewriteFieldExpression(se.Expression)
		return newSpansetFilter(fe), n
	case ScalarFilter:
		lhs, n1 := f.rewriteScalarExpression(se.LHS)
		rhs, n2 := f.rewriteScalarExpression(se.RHS)
		return newScalarFilter(se.Op, lhs, rhs), n1 + n2
	case SpansetOperation:
		lhs, n1 := f.rewriteSpansetExpression(se.LHS)
		rhs, n2 := f.rewriteSpansetExpression(se.RHS)
		return newSpansetOperation(se.Op, lhs, rhs), n1 + n2
	case Pipeline:
		return f.rewritePipeline(se)
	}

	return se, 0
}

func (f *fieldExpressionRewriter) rewriteScalarExpression(se ScalarExpression) (ScalarExpression, int) {
	switch se := se.(type) {
	case Pipeline:
		return f.rewritePipeline(se)
	case ScalarOperation:
		lhs, n1 := f.rewriteScalarExpression(se.LHS)
		rhs, n2 := f.rewriteScalarExpression(se.RHS)
		return newScalarOperation(se.Op, lhs, rhs), n1 + n2
	case Aggregate:
		fe, n := f.rewriteFieldExpression(se.e)
		return newAggregate(se.op, fe), n
	case Static:
		fe, n := f.rewriteFieldExpression(se)
		if s, ok := fe.(Static); ok {
			return s, n
		}
	}
	return se, 0
}

func (f *fieldExpressionRewriter) rewriteFieldExpression(fe FieldExpression) (FieldExpression, int) {
	var rwCount int

	switch e := fe.(type) {
	case *BinaryOperation:
		lhs, n := f.rewriteFieldExpression(e.LHS)
		rwCount += n
		rhs, n := f.rewriteFieldExpression(e.RHS)
		rwCount += n
		fe = newBinaryOperation(e.Op, lhs, rhs)
	case *UnaryOperation:
		exp, n := f.rewriteFieldExpression(e.Expression)
		rwCount += n
		fe = newUnaryOperation(e.Op, exp)
	}

	for _, fn := range f.rewriteFunctions {
		e, n := fn(fe)
		fe = e
		rwCount += n
	}

	return fe, rwCount
}

// rewriteOrToIn is a fieldExpressionRewriteFn that rewrites a single { .a = "a" || .b = "b" } to { .a = ["a", "b"] }
func rewriteOrToIn(fe FieldExpression) (FieldExpression, int) {
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpOr, OpEqual, OpIn)
}

// rewriteAndToNotIn is a fieldExpressionRewriteFn that rewrites a single { .a != "a" && .b != "b" } to { .a != ["a", "b"] }
func rewriteAndToNotIn(fe FieldExpression) (FieldExpression, int) {
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpAnd, OpNotEqual, OpNotIn)
}

// rewriteOrToMatchAny is a fieldExpressionRewriteFn that rewrites a single { .a =~ "a" || .b =~ "b" } to { .a =~ ["a", "b"] }
func rewriteOrToMatchAny(fe FieldExpression) (FieldExpression, int) {
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpOr, OpRegex, OpRegexMatchAny, TypeString, TypeStringArray)
}

// rewriteAndToMatchNone is a fieldExpressionRewriteFn that rewrites a single { .a !~ "a" && .b !~ "b" } to { .a !~ ["a", "b"]}
func rewriteAndToMatchNone(fe FieldExpression) (FieldExpression, int) {
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpAnd, OpNotRegex, OpRegexMatchNone, TypeString, TypeStringArray)
}

// rewriteBinaryOperationsToArrayEquivalent rewrites binary operations to an equivalent array form
func rewriteBinaryOperationsToArrayEquivalent(fe FieldExpression, opOuter, opScalar, opArray Operator, restrictTypes ...StaticType) (FieldExpression, int) {
	binOp, ok := fe.(*BinaryOperation)
	if !ok || binOp.Op != opOuter {
		return fe, 0
	}

	// prepare LHS and RHS operands
	opLHS, ok := binOp.LHS.(*BinaryOperation)
	if !ok {
		return fe, 0
	}
	opRHS, ok := binOp.RHS.(*BinaryOperation)
	if !ok {
		return fe, 0
	}

	attrLHS, valLHS, ok := getOperandsFromBinaryOperation(opLHS, opScalar, opArray)
	if !ok {
		return fe, 0
	}
	attrRHS, valRHS, ok := getOperandsFromBinaryOperation(opRHS, opScalar, opArray)
	if !ok || attrLHS != attrRHS {
		return fe, 0
	}

	// if the types are restricted to a specific set, ensure they are valid
	if len(restrictTypes) > 0 {
		if !slices.Contains(restrictTypes, valLHS.Type) {
			return fe, 0
		}
		if !slices.Contains(restrictTypes, valRHS.Type) {
			return fe, 0
		}
	}

	// create an equivalent array operation
	array, ok := valLHS.append(valRHS)
	if !ok {
		return fe, 0
	}

	return newBinaryOperation(opArray, attrLHS, array), 1
}

func getOperandsFromBinaryOperation(exp FieldExpression, operators ...Operator) (Attribute, Static, bool) {
	op, ok := exp.(*BinaryOperation)
	if !ok || !slices.Contains(operators, op.Op) {
		return Attribute{}, StaticNil, false
	}

	rhs := op.RHS

	attr, ok := op.LHS.(Attribute)
	if !ok {
		attr, ok = op.RHS.(Attribute)
		if !ok {
			return Attribute{}, StaticNil, false
		}
		rhs = op.LHS
	}

	val, ok := rhs.(Static)
	if !ok {
		return Attribute{}, StaticNil, false
	}

	return attr, val, true
}
