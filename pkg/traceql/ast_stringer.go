package traceql

import (
	"fmt"
	"strconv"
	"strings"
)

var stringerOps = map[int]string{
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

var stringerAggs = map[int]string{
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
	return binaryOp(stringerOps[o.op], o.lhs, o.rhs)
}

func (a Aggregate) String() string {
	if a.e == nil {
		return stringerAggs[a.agg] + "()"
	}

	return stringerAggs[a.agg] + "(" + a.e.String() + ")"
}

func (o SpansetOperation) String() string {
	return binaryOp(stringerOps[o.op], o.lhs, o.rhs)
}

func (f SpansetFilter) String() string {
	return "{ " + f.e.String() + " }"
}

func (f ScalarFilter) String() string {
	return binaryOp(stringerOps[f.op], f.lhs, f.rhs)
}

func (o BinaryOperation) String() string {
	return binaryOp(stringerOps[o.op], o.lhs, o.rhs)
}

func (o UnaryOperation) String() string {
	return unaryOp(stringerOps[o.op], o.e)
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
	case typeIntrinsic:
		switch n.n {
		case intrinsicDuration:
			return "duration"
		case intrinsicChildCount:
			return "childCount"
		case intrinsicName:
			return "name"
		case intrinsicStatus:
			return "status"
		case intrinsicParent:
			return "parent"
		default:
			return fmt.Sprintf("intrinsic(%d)", n.n)
		}
	case typeNil:
		return "nil"
	case typeDuration:
		return n.d.String()
	case typeStatus:
		switch n.n {
		case statusError:
			return "error"
		case statusOk:
			return "ok"
		case statusUnset:
			return "unset"
		default:
			return fmt.Sprintf("status(%d)", n.n)
		}
	}

	return fmt.Sprintf("static(%d)", n.staticType)
}

func (a Attribute) String() string {
	scope := ""

	switch a.scope {
	case attributeScopeNone:
		scope = "."
	case attributeScopeParent:
		scope = "parent."
	case attributeScopeSpan:
		scope = "span."
	case attributeScopeResource:
		scope = "resource."
	default:
		scope = fmt.Sprintf("att(%d).", a.scope)
	}

	return scope + a.att
}

func binaryOp(op string, lhs element, rhs element) string {
	return wrapElement(lhs) + " " + op + " " + wrapElement(rhs)
}

func unaryOp(op string, e element) string {
	return op + wrapElement(e)
}

func wrapElement(e element) string {
	static, ok := e.(Static)
	if ok {
		return static.String()
	}
	att, ok := e.(Attribute)
	if ok {
		return att.String()
	}
	return "(" + e.String() + ")"
}
