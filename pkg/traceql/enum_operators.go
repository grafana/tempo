package traceql

import "fmt"

type Operator int

const (
	OpNone Operator = iota
	OpAdd
	OpSub
	OpDiv
	OpMod
	OpMult
	OpEqual
	OpNotEqual
	OpRegex
	OpNotRegex
	OpGreater
	OpGreaterEqual
	OpLess
	OpLessEqual
	OpPower
	OpAnd
	OpOr
	OpNot
	OpSpansetChild
	OpSpansetParent
	OpSpansetDescendant
	OpSpansetAncestor
	OpSpansetAnd
	OpSpansetUnion
	OpSpansetSibling
	OpSpansetNotChild
	OpSpansetNotParent
	OpSpansetNotSibling
	OpSpansetNotAncestor
	OpSpansetNotDescendant
	OpSpansetUnionChild
	OpSpansetUnionParent
	OpSpansetUnionSibling
	OpSpansetUnionAncestor
	OpSpansetUnionDescendant

	// The following operators are used internally and only exist in the AST. They are not parseable in TraceQL
	OpExists
	OpNotExists
	OpIn
	OpNotIn
	OpRegexMatchAny
	OpRegexMatchNone
)

func (op Operator) isArithmetic() bool {
	return op == OpAdd || op == OpSub || op == OpMult || op == OpDiv
}

func (op Operator) isBoolean() bool {
	return op == OpOr ||
		op == OpAnd ||
		op == OpEqual ||
		op == OpNotEqual ||
		op == OpRegex ||
		op == OpNotRegex ||
		op == OpGreater ||
		op == OpGreaterEqual ||
		op == OpLess ||
		op == OpLessEqual ||
		op == OpNot ||
		op == OpExists ||
		op == OpNotExists ||
		op == OpIn ||
		op == OpNotIn ||
		op == OpRegexMatchAny ||
		op == OpRegexMatchNone
}

func (op Operator) binaryTypesValid(lhsT StaticType, rhsT StaticType) bool {
	return binaryTypeValid(op, lhsT) && binaryTypeValid(op, rhsT)
}

func binaryTypeValid(op Operator, t StaticType) bool {
	if t == TypeAttribute {
		return true
	}

	switch t {
	case TypeBoolean, TypeBooleanArray:
		return op == OpAnd ||
			op == OpOr ||
			op == OpEqual ||
			op == OpNotEqual ||
			op == OpIn ||
			op == OpNotIn
	case TypeFloat, TypeFloatArray, TypeInt, TypeIntArray, TypeDuration:
		return op == OpAdd ||
			op == OpSub ||
			op == OpMult ||
			op == OpDiv ||
			op == OpMod ||
			op == OpPower ||
			op == OpEqual ||
			op == OpNotEqual ||
			op == OpGreater ||
			op == OpGreaterEqual ||
			op == OpLess ||
			op == OpLessEqual ||
			op == OpIn ||
			op == OpNotIn
	case TypeString, TypeStringArray:
		return op == OpEqual ||
			op == OpNotEqual ||
			op == OpRegex ||
			op == OpNotRegex ||
			op == OpGreater ||
			op == OpGreaterEqual ||
			op == OpLess ||
			op == OpLessEqual ||
			op == OpIn ||
			op == OpNotIn ||
			op == OpRegexMatchAny ||
			op == OpRegexMatchNone
	case TypeNil, TypeStatus, TypeKind:
		return op == OpEqual || op == OpNotEqual
	}

	return false
}

func (op Operator) unaryTypesValid(t StaticType) bool {
	if t == TypeAttribute {
		return true
	}

	switch op {
	case OpSub:
		return t.isNumeric()
	case OpNot:
		return t == TypeBoolean
	case OpExists:
		return true
	case OpNotExists:
		return true
	}

	return false
}

// isArrayOp returns true if the operator is a dedicated array operator like IN, NOT IN, MATCH ANY, or MATCH NONE. It
// returns false for all other operators, even if those operators can operate on arrays like = or !=.
func (op Operator) isArrayOp() bool {
	return op == OpIn || op == OpNotIn || op == OpRegexMatchAny || op == OpRegexMatchNone
}

// toElementOp returns the equivalent element operator for the given array operator
func (op Operator) toElementOp() Operator {
	switch op {
	case OpIn:
		return OpEqual
	case OpNotIn:
		return OpNotEqual
	case OpRegexMatchAny:
		return OpRegex
	case OpRegexMatchNone:
		return OpNotRegex
	default:
		return op
	}
}

func (op Operator) String() string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpDiv:
		return "/"
	case OpMod:
		return "%"
	case OpMult:
		return "*"
	case OpEqual:
		return "="
	case OpNotEqual:
		return "!="
	case OpRegex:
		return "=~"
	case OpNotRegex:
		return "!~"
	case OpGreater:
		return ">"
	case OpGreaterEqual:
		return ">="
	case OpLess:
		return "<"
	case OpLessEqual:
		return "<="
	case OpPower:
		return "^"
	case OpAnd:
		return "&&"
	case OpOr:
		return "||"
	case OpNot:
		return "!"
	case OpSpansetChild:
		return ">"
	case OpSpansetParent:
		return "<"
	case OpSpansetDescendant:
		return ">>"
	case OpSpansetAncestor:
		return "<<"
	case OpSpansetAnd:
		return "&&"
	case OpSpansetSibling:
		return "~"
	case OpSpansetUnion:
		return "||"
	case OpSpansetNotChild:
		return "!>"
	case OpSpansetNotParent:
		return "!<"
	case OpSpansetNotSibling:
		return "!~"
	case OpSpansetNotAncestor:
		return "!<<"
	case OpSpansetNotDescendant:
		return "!>>"
	case OpSpansetUnionChild:
		return "&>"
	case OpSpansetUnionParent:
		return "&<"
	case OpSpansetUnionSibling:
		return "&~"
	case OpSpansetUnionAncestor:
		return "&<<"
	case OpSpansetUnionDescendant:
		return "&>>"

	// Operators IN, NOT IN, MATCH ANY, and MATCH NONE do not exist in the TraceQL syntax (only used internally)
	case OpIn:
		return "IN"
	case OpNotIn:
		return "NOT IN"
	case OpRegexMatchAny:
		return "MATCH ANY"
	case OpRegexMatchNone:
		return "MATCH NONE"
	}

	return fmt.Sprintf("operator(%d)", op)
}
