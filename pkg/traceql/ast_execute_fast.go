package traceql

import "strings"

// The boolean fast path evaluates the most common filter shapes straight to a bool:
//
//   - an attribute compared with a scalar static, e.g. `span.foo = "bar"` or `duration > 1s`
//   - && and || over operands that always evaluate to a boolean
//
// The generic path returns a 64-byte Static through every level of the expression tree, and
// copying those values dominates filter evaluation. The fast path returns a bool instead.
//
// The fast path must give exactly the same results as the generic path in
// BinaryOperation.execute, which stays the reference implementation. It only handles the cases
// it mirrors line by line and reports ok=false for everything else, which then takes the
// generic path.

type fastPath uint8

const (
	fastPathUnknown fastPath = iota // not classified yet
	fastPathNone                    // always use the generic path
	fastPathLogic                   // && or || with boolean-valued operands
	fastPathCompare                 // attribute compared with a scalar static
)

// useFastPath reports whether execute should try the fast path for o.
func (o *BinaryOperation) useFastPath(span Span) bool {
	if o.fastPath == fastPathUnknown {
		if span == nil {
			// constant folding while the AST is built, leave o untouched
			return false
		}
		o.classifyFastPath()
	}

	// while the branch optimizer samples it times the generic path
	return o.fastPath != fastPathNone && !o.b.Recording
}

// classifyFastPath runs once, on the first execution. The classification only depends on the
// operator and the operand types, which don't change afterwards: the branch optimizer swaps the
// operands of && and || but doesn't replace them.
func (o *BinaryOperation) classifyFastPath() {
	o.fastPath = fastPathNone

	switch o.Op {
	case OpAnd, OpOr:
		if isBooleanValued(o.LHS) && isBooleanValued(o.RHS) {
			o.fastPath = fastPathLogic
		}
	case OpEqual, OpNotEqual, OpGreater, OpGreaterEqual, OpLess, OpLessEqual, OpRegex, OpNotRegex:
		attr, ok := o.LHS.(Attribute)
		if !ok {
			return
		}
		static, ok := o.RHS.(Static)
		if !ok || static.Type.isArray() {
			return
		}
		o.compareAttr = attr
		o.compareStatic = static
		o.fastPath = fastPathCompare
	}
}

// isBooleanValued returns true if e always evaluates to a boolean or fails with an error.
func isBooleanValued(e FieldExpression) bool {
	switch e := e.(type) {
	case *BinaryOperation:
		return e.Op.isBoolean()
	case UnaryOperation:
		return e.Op == OpNot || e.Op == OpExists || e.Op == OpNotExists
	}
	return false
}

// executeFast evaluates o, which must be classified for the fast path. ok is false if the
// fast path can't evaluate o for this span and the generic path must be used instead.
func (o *BinaryOperation) executeFast(span Span) (b, ok bool, err error) {
	if o.fastPath == fastPathLogic {
		return o.executeLogic(span)
	}
	b, ok = o.executeCompare(span)
	return b, ok, nil
}

// executeLogic mirrors the generic path for && and || outside of branch sampling.
func (o *BinaryOperation) executeLogic(span Span) (b, ok bool, err error) {
	lhs, ok, err := executeBoolOperand(o.LHS, span)
	if err != nil || !ok {
		return false, false, err
	}

	// same short-circuit as the generic path
	if o.Op == OpAnd && !lhs {
		return false, true, nil
	}
	if o.Op == OpOr && lhs {
		return true, true, nil
	}

	rhs, ok, err := executeBoolOperand(o.RHS, span)
	if err != nil || !ok {
		return false, false, err
	}

	// lhs is true for && and false for ||, so the result is rhs
	return rhs, true, nil
}

// executeBoolOperand evaluates an operand of a fast && or ||. ok is false if the operand
// didn't evaluate to a boolean, which isBooleanValued rules out.
func executeBoolOperand(e FieldExpression, span Span) (b, ok bool, err error) {
	if o, isBinOp := e.(*BinaryOperation); isBinOp && o.useFastPath(span) {
		b, ok, err = o.executeFast(span)
		if ok || err != nil {
			return b, ok, err
		}
		// execute tries the fast path again, which is cheap and gives the same answer
	}

	result, err := e.execute(span)
	if err != nil {
		return false, false, err
	}

	b, ok = result.Bool()
	return b, ok, nil
}

// executeCompare mirrors the generic path for an attribute compared with a scalar static. ok is
// false for the cases it doesn't mirror: array attributes, and regexes that aren't compiled yet
// or are applied to non-string operands. The generic path compiles the regex and reports errors.
func (o *BinaryOperation) executeCompare(span Span) (b, ok bool) {
	// same as Attribute.execute
	lhs, found := span.AttributeFor(o.compareAttr)
	if !found {
		lhs = StaticNil
	}
	rhs := &o.compareStatic

	lhsT := lhs.Type
	rhsT := rhs.Type

	if lhsT.isArray() {
		return false, false
	}

	if !lhsT.isMatchingOperand(rhsT) {
		return false, true
	}
	if !o.Op.binaryTypesValid(lhsT, rhsT) {
		return false, true
	}

	switch {
	case lhsT == TypeString && rhsT == TypeString:
		switch o.Op {
		case OpGreater:
			return strings.Compare(lhs.EncodeToString(false), rhs.EncodeToString(false)) > 0, true
		case OpGreaterEqual:
			return strings.Compare(lhs.EncodeToString(false), rhs.EncodeToString(false)) >= 0, true
		case OpLess:
			return strings.Compare(lhs.EncodeToString(false), rhs.EncodeToString(false)) < 0, true
		case OpLessEqual:
			return strings.Compare(lhs.EncodeToString(false), rhs.EncodeToString(false)) <= 0, true
		case OpRegex, OpNotRegex:
			if len(o.compiledExpressions) != 1 {
				return false, false
			}
			return o.compiledExpressions[0].MatchString(lhs.EncodeToString(false)), true
		}

	case lhsT == TypeInt && rhsT == TypeInt:
		lhsN, _ := lhs.Int()
		rhsN, _ := rhs.Int()

		switch o.Op {
		case OpGreater:
			return lhsN > rhsN, true
		case OpGreaterEqual:
			return lhsN >= rhsN, true
		case OpLess:
			return lhsN < rhsN, true
		case OpLessEqual:
			return lhsN <= rhsN, true
		}
	}

	// same float catch-all as the generic path
	switch o.Op {
	case OpGreater:
		return lhs.Float() > rhs.Float(), true
	case OpGreaterEqual:
		return lhs.Float() >= rhs.Float(), true
	case OpLess:
		return lhs.Float() < rhs.Float(), true
	case OpLessEqual:
		return lhs.Float() <= rhs.Float(), true
	case OpEqual:
		return lhs.Equals(rhs), true
	case OpNotEqual:
		return lhs.NotEquals(rhs), true
	}

	return false, false
}
