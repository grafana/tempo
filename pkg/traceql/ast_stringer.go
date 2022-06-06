package traceql

import (
	"fmt"
	"strconv"
	"strings"
)

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
	return binaryOp(o.op, o.lhs, o.rhs)
}

func (a Aggregate) String() string {
	if a.e == nil {
		return a.agg.String() + "()"
	}

	return a.agg.String() + "(" + a.e.String() + ")"
}

func (o SpansetOperation) String() string {
	return binaryOp(o.op, o.lhs, o.rhs)
}

func (f SpansetFilter) String() string {
	return "{ " + f.e.String() + " }"
}

func (f ScalarFilter) String() string {
	return binaryOp(f.op, f.lhs, f.rhs)
}

func (o BinaryOperation) String() string {
	return binaryOp(o.op, o.lhs, o.rhs)
}

func (o UnaryOperation) String() string {
	return unaryOp(o.op, o.e)
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
	case typeNil:
		return "nil"
	case typeDuration:
		return n.d.String()
	case typeStatus:
		return n.status.String()
	}

	return fmt.Sprintf("static(%d)", n.staticType)
}

func (a Attribute) String() string {
	scopes := []string{}
	if a.parent {
		scopes = append(scopes, "parent")
	}

	if a.scope != attributeScopeNone {
		attributeScope := a.scope.String()
		scopes = append(scopes, attributeScope)
	}

	att := a.name
	if a.intrinsic != intrinsicNone {
		att = a.intrinsic.String()
	}

	scope := ""
	if len(scopes) > 0 {
		scope = strings.Join(scopes, ".")
	}
	scope += "."

	return scope + att
}

func binaryOp(op Operator, lhs element, rhs element) string {
	return wrapElement(lhs) + " " + op.String() + " " + wrapElement(rhs)
}

func unaryOp(op Operator, e element) string {
	return op.String() + wrapElement(e)
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
