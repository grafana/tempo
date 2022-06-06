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

var stringerOps = map[Operator]string{ // jpe remove
	opAdd:               "+",
	opSub:               "-",
	opDiv:               "/",
	opMod:               "%",
	opMult:              "*",
	opEqual:             "=",
	opNotEqual:          "!=",
	opRegex:             "=~",
	opNotRegex:          "!~",
	opGreater:           ">",
	opGreaterEqual:      ">=",
	opLess:              "<",
	opLessEqual:         "<=",
	opPower:             "^",
	opAnd:               "&&",
	opOr:                "||",
	opNot:               "!",
	opSpansetChild:      ">",
	opSpansetDescendant: ">>",
	opSpansetAnd:        "&&",
	opSpansetSibling:    "~",
	opSpansetUnion:      "||",
}

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
		op == opLessEqual
}

func (op Operator) typesValid(lhsT StaticType, rhsT StaticType) bool {
	// attribute types are validated at runtime
	if lhsT == typeAttribute && rhsT == typeAttribute {
		return true
	}

	t := lhsT
	if t == typeAttribute {
		t = rhsT
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

func (op Operator) String() string {
	s, ok := stringerOps[op]
	if ok {
		return s
	}
	return fmt.Sprintf("operator(%d)", op)
}
