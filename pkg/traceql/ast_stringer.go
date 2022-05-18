package traceql

import (
	"strconv"
	"strings"
)

var ops = map[int]string{ // jpe - if we use the lexer constants we can not do this
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
}

var aggs = map[int]string{
	aggregateCount: "count",
	aggregateMax:   "max",
	aggregateMin:   "min",
	aggregateSum:   "sum",
	aggregateAvg:   "avg",
}

func (r RootExpr) String() string {
	return r.p.String()
}

func (p Pipeline) String() string {
	s := make([]string, 0, len(p.p))
	for _, p := range p.p {
		s = append(s, p.String())
	}
	return strings.Join(s, "|")
}

func (o GroupOperation) String() string {
	return "by(" + o.e.String() + ")"
}

func (o CoalesceOperation) String() string {
	return "coalesce()"
}

func (o ScalarOperation) String() string {
	return o.lhs.String() + ops[o.op] + o.rhs.String()
}

func (a Aggregate) String() string {
	if a.e == nil {
		return aggs[a.agg] + "()"
	}

	return aggs[a.agg] + "(" + a.e.String() + ")"
}

func (o SpansetOperation) String() string {
	return o.lhs.String() + ops[o.op] + o.rhs.String()
}

func (f SpansetFilter) String() string {
	return "{" + f.e.String() + "}"
}

func (f ScalarFilter) String() string {
	return f.lhs.String() + ops[f.op] + f.rhs.String()
}

func (o BinaryOperation) String() string {
	return o.lhs.String() + ops[o.op] + o.rhs.String()
}

func (o UnaryOperation) String() string {
	return ops[o.op] + o.e.String()
}

func (n Static) String() string {
	switch n.staticType {
	case typeInt:
		return strconv.Itoa(n.n)
	case typeFloat:
		return strconv.FormatFloat(n.f, 'f', 5, 64)
	case typeString:
		return "`" + n.s + "`"
	case typeBoolean:
		return strconv.FormatBool(n.b)
	case typeIdentifier:
		return n.s
	case typeNil:
		return "nil"
	case typeDuration:
		return n.d.String()
	}

	return "??"
}
