package traceql

import "fmt"

type Operator int

const (
	opAdd Operator = iota
	opSub
	opDiv
	opMod
	opMult
	opEqual
	opNotEqual
	opRegex
	opNotRegex
	opGreater
	opGreaterEqual
	opLess
	opLessEqual
	opPower
	opAnd
	opOr
	opNot
	opSpansetChild
	opSpansetDescendant
	opSpansetAnd
	opSpansetUnion
	opSpansetSibling
)

func (op Operator) isBoolean() bool {
	return op == opOr ||
		op == opAnd ||
		op == opEqual ||
		op == opNotEqual ||
		op == opRegex ||
		op == opNotRegex ||
		op == opGreater ||
		op == opGreaterEqual ||
		op == opLess ||
		op == opLessEqual ||
		op == opNot
}

func (op Operator) binaryTypesValid(lhsT StaticType, rhsT StaticType) bool {
	return binaryTypeValid(op, lhsT) && binaryTypeValid(op, rhsT)
}

func binaryTypeValid(op Operator, t StaticType) bool {
	if t == typeAttribute {
		return true
	}

	switch t {
	case typeBoolean:
		return op == opAnd ||
			op == opOr ||
			op == opEqual ||
			op == opNotEqual
	case typeFloat:
		fallthrough
	case typeInt:
		fallthrough
	case typeDuration:
		return op == opAdd ||
			op == opSub ||
			op == opMult ||
			op == opDiv ||
			op == opMod ||
			op == opPower ||
			op == opEqual ||
			op == opNotEqual ||
			op == opGreater ||
			op == opGreaterEqual ||
			op == opLess ||
			op == opLessEqual
	case typeString:
		return op == opEqual ||
			op == opNotEqual ||
			op == opRegex ||
			op == opNotRegex
	case typeNil:
		fallthrough
	case typeStatus:
		return op == opEqual || op == opNotEqual
	}

	return false
}

func (op Operator) unaryTypesValid(t StaticType) bool {
	if t == typeAttribute {
		return true
	}

	switch op {
	case opSub:
		return t.isNumeric()
	case opNot:
		return t == typeBoolean
	}

	return false
}

func (op Operator) String() string {

	switch op {
	case opAdd:
		return "+"
	case opSub:
		return "-"
	case opDiv:
		return "/"
	case opMod:
		return "%"
	case opMult:
		return "*"
	case opEqual:
		return "="
	case opNotEqual:
		return "!="
	case opRegex:
		return "=~"
	case opNotRegex:
		return "!~"
	case opGreater:
		return ">"
	case opGreaterEqual:
		return ">="
	case opLess:
		return "<"
	case opLessEqual:
		return "<="
	case opPower:
		return "^"
	case opAnd:
		return "&&"
	case opOr:
		return "||"
	case opNot:
		return "!"
	case opSpansetChild:
		return ">"
	case opSpansetDescendant:
		return ">>"
	case opSpansetAnd:
		return "&&"
	case opSpansetSibling:
		return "~"
	case opSpansetUnion:
		return "||"
	}

	return fmt.Sprintf("operator(%d)", op)
}
