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
	OpSpansetDescendant
	OpSpansetAnd
	OpSpansetUnion
	OpSpansetSibling
)

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
		op == OpNot
}

func (op Operator) binaryTypesValid(lhsT StaticType, rhsT StaticType) bool {
	return binaryTypeValid(op, lhsT) && binaryTypeValid(op, rhsT)
}

func binaryTypeValid(op Operator, t StaticType) bool {
	if t == TypeAttribute {
		return true
	}

	switch t {
	case TypeBoolean:
		return op == OpAnd ||
			op == OpOr ||
			op == OpEqual ||
			op == OpNotEqual
	case TypeFloat:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeDuration:
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
			op == OpLessEqual
	case TypeString:
		return op == OpEqual ||
			op == OpNotEqual ||
			op == OpRegex ||
			op == OpNotRegex
	case TypeNil:
		fallthrough
	case TypeStatus:
		return op == OpEqual || op == OpNotEqual
	case TypeKind:
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
	}

	return false
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
	case OpSpansetDescendant:
		return ">>"
	case OpSpansetAnd:
		return "&&"
	case OpSpansetSibling:
		return "~"
	case OpSpansetUnion:
		return "||"
	}

	return fmt.Sprintf("operator(%d)", op)
}
