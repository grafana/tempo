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
// - { .a =  "a" || .a =  "b" } => { .a =  ["a", "b"] }
// - { .a != "a" && .a != "b" } => { .a != ["a", "b"] }
// - { .a =~ "a" || .a =~ "b" } => { .a =~ ["a", "b"] }
// - { .a !~ "a" && .a !~ "b" } => { .a !~ ["a", "b"] }
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

func (f *fieldExpressionRewriter) rewriteScalarExpression(se ScalarExpression) (ScalarExpression, int) {
	switch se := se.(type) {
	case Pipeline:
		return f.rewritePipeline(se)
	case ScalarOperation:
		var rwCount int
		lhs, n := f.rewriteScalarExpression(se.LHS)
		rwCount += n
		rhs, n := f.rewriteScalarExpression(se.RHS)
		rwCount += n
		return newScalarOperation(se.Op, lhs, rhs), rwCount
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

	switch fe := fe.(type) {
	case *BinaryOperation:
		lhs, n := f.rewriteFieldExpression(fe.LHS)
		rwCount += n
		fe.LHS = lhs
		rhs, n := f.rewriteFieldExpression(fe.RHS)
		rwCount += n
		fe.RHS = rhs
	case *UnaryOperation:
		e, n := f.rewriteFieldExpression(fe.Expression)
		fe.Expression = e
		rwCount += n
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
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpOr, OpEqual)
}

// rewriteAndToNotIn is a fieldExpressionRewriteFn that rewrites a single { .a != "a" && .b != "b" } to { .a != ["a", "b"] }
func rewriteAndToNotIn(fe FieldExpression) (FieldExpression, int) {
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpAnd, OpNotEqual)
}

// rewriteOrToMatchAny is a fieldExpressionRewriteFn that rewrites a single { .a =~ "a" || .b =~ "b" } to { .a =~ ["a", "b"] }
func rewriteOrToMatchAny(fe FieldExpression) (FieldExpression, int) {
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpOr, OpRegex, TypeString, TypeStringArray)
}

// rewriteAndToMatchNone is a fieldExpressionRewriteFn that rewrites a single { .a !~ "a" && .b !~ "b" } to { .a !~ ["a", "b"]}
func rewriteAndToMatchNone(fe FieldExpression) (FieldExpression, int) {
	return rewriteBinaryOperationsToArrayEquivalent(fe, OpAnd, OpNotRegex, TypeString, TypeStringArray)
}

// rewriteBinaryOperationsToArrayEquivalent rewrites binary operations to an equivalent array form
func rewriteBinaryOperationsToArrayEquivalent(fe FieldExpression, opOuter, opInner Operator, restrictTypes ...StaticType) (FieldExpression, int) {
	opOr, ok := fe.(*BinaryOperation)
	if !ok || opOr.Op != opOuter {
		return fe, 0
	}

	// prepare LHS and RHS operands
	opLHS, ok := opOr.LHS.(*BinaryOperation)
	if !ok {
		return fe, 0
	}
	opRHS, ok := opOr.RHS.(*BinaryOperation)
	if !ok {
		return fe, 0
	}

	attrLHS, valLHS, ok := getOperandsFromBinaryOperation(opLHS, opInner)
	if !ok {
		return fe, 0
	}
	attrRHS, valRHS, ok := getOperandsFromBinaryOperation(opRHS, opInner)
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

	return &BinaryOperation{
		Op:  opInner,
		LHS: attrLHS,
		RHS: array,
	}, 1
}

func getOperandsFromBinaryOperation(exp FieldExpression, operator Operator) (Attribute, Static, bool) {
	op, ok := exp.(*BinaryOperation)
	if !ok || op.Op != operator {
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
