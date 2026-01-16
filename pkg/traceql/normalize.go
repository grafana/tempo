package traceql

// Normalize rewrites the AST into a canonical form suitable for hashing.
// It preserves semantics and only reorders commutative logical expressions.
func Normalize(e Element) {
	if e == nil {
		return
	}
	normalizeElement(e)
}

func normalizeElement(e Element) {
	switch n := e.(type) {

	case *RootExpr:
		for _, elem := range n.Pipeline.Elements {
			normalizePipelineElement(elem)
		}

	case Pipeline:
		for _, elem := range n.Elements {
			normalizePipelineElement(elem)
		}
	}
}

func normalizePipelineElement(e PipelineElement) {
	switch n := e.(type) {

	case *SpansetFilter:
		normalizeFieldExpr(n.Expression)

	case SpansetOperation:
		normalizePipelineElement(n.LHS)
		normalizePipelineElement(n.RHS)

	case ScalarFilter:
		normalizeScalarExpr(n.LHS)
		normalizeScalarExpr(n.RHS)

	case Aggregate:
		if n.e != nil {
			normalizeFieldExpr(n.e)
		}

	case GroupOperation:
		normalizeFieldExpr(n.Expression)
	}
}

func normalizeFieldExpr(e FieldExpression) {
	switch n := e.(type) {

	case *BinaryOperation:
		normalizeBinaryOperation(n)

	case UnaryOperation:
		normalizeFieldExpr(n.Expression)
	}
}

func normalizeScalarExpr(e ScalarExpression) {
	switch n := e.(type) {

	case ScalarOperation:
		normalizeScalarExpr(n.LHS)
		normalizeScalarExpr(n.RHS)

	case Aggregate:
		if n.e != nil {
			normalizeFieldExpr(n.e)
		}
	}
}

func normalizeBinaryOperation(op *BinaryOperation) {
	if op == nil {
		return
	}

	// Normalize children first
	if op.LHS != nil {
		normalizeFieldExpr(op.LHS)
	}
	if op.RHS != nil {
		normalizeFieldExpr(op.RHS)
	}

	// Only reorder commutative logical operators
	if op.Op != OpAnd && op.Op != OpOr {
		return
	}

	leftKey := op.LHS.String()
	rightKey := op.RHS.String()

	if leftKey > rightKey {
		op.LHS, op.RHS = op.RHS, op.LHS
	}
}
