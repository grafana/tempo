package traceql

const (
	// TransformationDeMorgan is the name of the rewriter that pushes ! down to the comparisons.
	TransformationDeMorgan = "demorgan"
)

// newDeMorganRewriter creates an ASTRewriter that pushes ! down to the comparisons using De Morgan's laws.
func newDeMorganRewriter() *fieldExpressionRewriter {
	return &fieldExpressionRewriter{
		rewriteFunctions: []fieldExpressionRewriteFn{
			rewriteDeMorgan,
		},
	}
}

// rewriteDeMorgan is a fieldExpressionRewriteFn that pushes a negation down to the comparisons using
// De Morgan's laws and by flipping the comparison operators:
//
//	{ !(.a = "a") }             => { .a != "a" }
//	{ !(.a = "a" && .b = "b") } => { .a != "a" || .b != "b" }
//
// A comparison with a missing attribute or with a mismatched type is false. The negated comparison is
// false as well, like NULL in SQL: { !(.a = "a") } doesn't match spans without .a. This differs from
// evaluating the ! as written, which is true for a missing attribute. Only nil checks match a
// missing attribute, and they are never false for both: { !(.a != nil) } is { .a = nil }.
//
// Comparisons that can't be flipped keep their negation, and with it the evaluation as written.
// For example { !(.a > 1) } still matches spans without .a, because an attribute can be an array
// and the opposite operator is not the complement.
func rewriteDeMorgan(fe FieldExpression) (FieldExpression, int) {
	u, ok := fe.(UnaryOperation)
	if !ok || u.Op != OpNot {
		return fe, 0
	}

	negated, ok := negateExpression(u.Expression)
	if !ok {
		return fe, 0
	}
	return negated, 1
}

// negateExpression returns the negation of e with the negation pushed down to the comparisons. The
// returned bool is false when nothing could be pushed down.
func negateExpression(e FieldExpression) (FieldExpression, bool) {
	switch e := e.(type) {
	case *BinaryOperation:
		if e.Op == OpAnd || e.Op == OpOr {
			lhs, lhsOk := negateExpression(e.LHS)
			rhs, rhsOk := negateExpression(e.RHS)
			if !lhsOk && !rhsOk {
				return nil, false
			}
			if !lhsOk {
				lhs = newUnaryOperation(OpNot, e.LHS)
			}
			if !rhsOk {
				rhs = newUnaryOperation(OpNot, e.RHS)
			}

			op := OpAnd
			if e.Op == OpAnd {
				op = OpOr
			}
			return newBinaryOperation(op, lhs, rhs), true
		}

		op, ok := negateComparison(e)
		if !ok {
			return nil, false
		}
		return newBinaryOperation(op, e.LHS, e.RHS), true

	case UnaryOperation:
		// { .a != nil } and { .a = nil } check that the attribute exists. They are never false for both.
		attr, ok := e.Expression.(Attribute)
		if !ok || hasMultipleValues(attr) {
			return nil, false
		}
		switch e.Op {
		case OpExists:
			return newUnaryOperation(OpNotExists, attr), true
		case OpNotExists:
			return newUnaryOperation(OpExists, attr), true
		}
	}

	return nil, false
}

// negateComparison returns the operator that is the complement of the comparison, if there is one.
func negateComparison(b *BinaryOperation) (Operator, bool) {
	var attributes, statics int
	onlyScalarAttributes := true

	for _, operand := range []FieldExpression{b.LHS, b.RHS} {
		switch o := operand.(type) {
		case Attribute:
			if hasMultipleValues(o) {
				return OpNone, false
			}
			attributes++
			onlyScalarAttributes = onlyScalarAttributes && isScalarIntrinsic(o)
		case Static:
			// nil comparisons are exists checks
			if o.Type == TypeNil {
				return OpNone, false
			}
			statics++
		default:
			return OpNone, false
		}
	}
	// only the comparison of an attribute with a static value. Two attributes can both be arrays.
	if attributes != 1 || statics != 1 {
		return OpNone, false
	}

	switch b.Op {
	case OpEqual:
		return OpNotEqual, true
	case OpNotEqual:
		return OpEqual, true
	case OpRegex:
		return OpNotRegex, true
	case OpNotRegex:
		return OpRegex, true
	case OpIn:
		return OpNotIn, true
	case OpNotIn:
		return OpIn, true
	case OpRegexMatchAny:
		return OpRegexMatchNone, true
	case OpRegexMatchNone:
		return OpRegexMatchAny, true
	}

	// Attributes can be arrays, which match when any element matches. For them the opposite ordering
	// operator is not the complement, so it's only flipped for the intrinsics that are never arrays.
	if onlyScalarAttributes {
		switch b.Op {
		case OpLess:
			return OpGreaterEqual, true
		case OpLessEqual:
			return OpGreater, true
		case OpGreater:
			return OpLessEqual, true
		case OpGreaterEqual:
			return OpLess, true
		}
	}

	return OpNone, false
}

// hasMultipleValues returns true for attributes that can have a value per event or link of a span.
func hasMultipleValues(a Attribute) bool {
	switch a.Scope {
	case AttributeScopeEvent, AttributeScopeLink:
		return true
	}

	switch a.Intrinsic {
	case IntrinsicEventName, IntrinsicEventTimeSinceStart, IntrinsicLinkSpanID, IntrinsicLinkTraceID:
		return true
	}

	return false
}

// isScalarIntrinsic returns true for intrinsics that have a single scalar value, never an array.
func isScalarIntrinsic(a Attribute) bool {
	switch a.Intrinsic {
	case IntrinsicDuration,
		IntrinsicName,
		IntrinsicStatusMessage,
		IntrinsicChildCount,
		IntrinsicTraceRootService,
		IntrinsicTraceRootSpan,
		IntrinsicTraceDuration,
		IntrinsicNestedSetLeft,
		IntrinsicNestedSetRight,
		IntrinsicNestedSetParent,
		IntrinsicInstrumentationName,
		IntrinsicInstrumentationVersion:
		return true
	}
	return false
}
