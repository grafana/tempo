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
type fieldExpressionRewriteFn func(e FieldExpression) FieldExpression

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

	for _, fn := range f.rewriteFunctions {
		e = fn(e)
	}

	return e
}

// rewriteOrToIn is a fieldExpressionRewriteFn that rewrites a single { .a = "a" || .b = "b" } to { .a = ["a", "b"] }
func rewriteOrToIn(e FieldExpression) FieldExpression {
	return rewriteBinaryOperationsToArrayEquivalent(e, OpOr, OpEqual)
}

// rewriteAndToNotIn is a fieldExpressionRewriteFn that rewrites a single { .a != "a" && .b != "b" } to { .a != ["a", "b"] }
func rewriteAndToNotIn(e FieldExpression) FieldExpression {
	return rewriteBinaryOperationsToArrayEquivalent(e, OpAnd, OpNotEqual)
}

// rewriteOrToMatchAny is a fieldExpressionRewriteFn that rewrites a single { .a =~ "a" || .b =~ "b" } to { .a =~ ["a", "b"] }
func rewriteOrToMatchAny(e FieldExpression) FieldExpression {
	return rewriteBinaryOperationsToArrayEquivalent(e, OpOr, OpRegex, TypeString, TypeStringArray)
}

// rewriteAndToMatchNone is a fieldExpressionRewriteFn that rewrites a single { .a !~ "a" && .b !~ "b" } to { .a !~ ["a", "b"]}
func rewriteAndToMatchNone(e FieldExpression) FieldExpression {
	return rewriteBinaryOperationsToArrayEquivalent(e, OpAnd, OpNotRegex, TypeString, TypeStringArray)
}

// rewriteBinaryOperationsToArrayEquivalent rewrites binary operations to an equivalent array form
func rewriteBinaryOperationsToArrayEquivalent(e FieldExpression, opOuter, opInner Operator, restrictTypes ...StaticType) FieldExpression {
	opOr, ok := e.(*BinaryOperation)
	if !ok || opOr.Op != opOuter {
		return e
	}

	// prepare LHS and RHS operands
	opLHS, ok := opOr.LHS.(*BinaryOperation)
	if !ok {
		return e
	}
	opRHS, ok := opOr.RHS.(*BinaryOperation)
	if !ok {
		return e
	}

	lhsAttr, lhsVal, ok := getOperandsFromBinaryOperation(opLHS, opInner)
	if !ok {
		return e
	}
	rhsAttr, rhsVal, ok := getOperandsFromBinaryOperation(opRHS, opInner)
	if !ok || lhsAttr != rhsAttr {
		return e
	}

	// if the types are restricted to a specific set, ensure they are valid
	if len(restrictTypes) > 0 {
		if !slices.Contains(restrictTypes, lhsVal.Type) {
			return e
		}
		if !slices.Contains(restrictTypes, rhsVal.Type) {
			return e
		}
	}

	// create an equivalent array operation
	array, ok := lhsVal.append(rhsVal)
	if !ok {
		return e
	}

	return &BinaryOperation{
		Op:  opInner,
		LHS: lhsAttr,
		RHS: array,
	}
}

func getOperandsFromBinaryOperation(exp FieldExpression, operator Operator) (Attribute, Static, bool) {
	binOp, ok := exp.(*BinaryOperation)
	if !ok || binOp.Op != operator {
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
