package traceql

// Normalize rewrites the AST into a canonical form suitable for hashing.
// It preserves semantics and only reorders commutative logical expressions.
func Normalize(e Element) {
	switch n := e.(type) {
	case *RootExpr:
		if n == nil {
			return
		}
		n.Pipeline = normalizePipeline(n.Pipeline)
	}
}

func normalizePipeline(p Pipeline) Pipeline {
	for i, elem := range p.Elements {
		p.Elements[i] = normalizePipelineElement(elem)
	}
	return p
}

func normalizePipelineElement(e PipelineElement) PipelineElement {
	switch n := e.(type) {
	case Pipeline:
		return normalizePipeline(n)

	case *SpansetFilter:
		n.Expression = normalizeFieldExpr(n.Expression)
		return n

	case SpansetOperation:
		n.LHS = normalizePipelineElement(n.LHS).(SpansetExpression)
		n.RHS = normalizePipelineElement(n.RHS).(SpansetExpression)

		if n.Op != OpSpansetAnd && n.Op != OpSpansetUnion {
			return n
		}

		if n.LHS.String() > n.RHS.String() {
			n.LHS, n.RHS = n.RHS, n.LHS
		}
		return n

	case ScalarFilter:
		n.LHS = normalizeScalarExpr(n.LHS)
		n.RHS = normalizeScalarExpr(n.RHS)
		return n

	case Aggregate:
		if n.e != nil {
			n.e = normalizeFieldExpr(n.e)
		}
		return n

	case GroupOperation:
		n.Expression = normalizeFieldExpr(n.Expression)
		return n
	}

	return e
}

func normalizeFieldExpr(e FieldExpression) FieldExpression {
	switch n := e.(type) {
	case *BinaryOperation:
		return normalizeBinaryOperation(n)

	case UnaryOperation:
		n.Expression = normalizeFieldExpr(n.Expression)
		return n

	case *UnaryOperation:
		n.Expression = normalizeFieldExpr(n.Expression)
		return n
	}

	return e
}

func normalizeScalarExpr(e ScalarExpression) ScalarExpression {
	switch n := e.(type) {
	case Pipeline:
		return normalizePipeline(n)

	case ScalarOperation:
		n.LHS = normalizeScalarExpr(n.LHS)
		n.RHS = normalizeScalarExpr(n.RHS)
		return n

	case *ScalarOperation:
		n.LHS = normalizeScalarExpr(n.LHS)
		n.RHS = normalizeScalarExpr(n.RHS)
		return n

	case Aggregate:
		if n.e != nil {
			n.e = normalizeFieldExpr(n.e)
		}
		return n

	case *Aggregate:
		if n.e != nil {
			n.e = normalizeFieldExpr(n.e)
		}
		return n
	}

	return e
}

func normalizeBinaryOperation(op *BinaryOperation) *BinaryOperation {
	if op == nil {
		return nil
	}

	// Normalize children first
	if op.LHS != nil {
		op.LHS = normalizeFieldExpr(op.LHS)
	}
	if op.RHS != nil {
		op.RHS = normalizeFieldExpr(op.RHS)
	}

	// Only reorder commutative logical operators
	if op.Op != OpAnd && op.Op != OpOr {
		return op
	}

	leftKey := op.LHS.String()
	rightKey := op.RHS.String()

	if leftKey > rightKey {
		op.LHS, op.RHS = op.RHS, op.LHS
	}

	return op
}
