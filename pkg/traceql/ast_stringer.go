package traceql

import (
	"strconv"
	"strings"
)

func (r RootExpr) String() string {
	s := strings.Builder{}
	s.WriteString(r.Pipeline.String())
	if r.MetricsPipeline != nil {
		s.WriteString(" | ")
		s.WriteString(r.MetricsPipeline.String())
	}
	if r.Hints != nil {
		s.WriteString(" ")
		s.WriteString(r.Hints.String())
	}
	return s.String()
}

func (p Pipeline) String() string {
	s := make([]string, 0, len(p.Elements))
	for _, p := range p.Elements {
		s = append(s, p.String())
	}
	return strings.Join(s, "|")
}

func (o GroupOperation) String() string {
	return "by(" + o.Expression.String() + ")"
}

func (o CoalesceOperation) String() string {
	return "coalesce()"
}

func (o SelectOperation) String() string {
	s := make([]string, 0, len(o.attrs))
	for _, e := range o.attrs {
		s = append(s, e.String())
	}
	return "select(" + strings.Join(s, ", ") + ")"
}

func (o ScalarOperation) String() string {
	return binaryOp(o.Op, o.LHS, o.RHS)
}

func (a Aggregate) String() string {
	if a.e == nil {
		return a.op.String() + "()"
	}

	return a.op.String() + "(" + a.e.String() + ")"
}

func (o SpansetOperation) String() string {
	return binaryOp(o.Op, o.LHS, o.RHS)
}

func (f SpansetFilter) String() string {
	return "{ " + f.Expression.String() + " }"
}

func (f ScalarFilter) String() string {
	return binaryOp(f.op, f.lhs, f.rhs)
}

func (o *BinaryOperation) String() string {
	return binaryOp(o.Op, o.LHS, o.RHS)
}

func (o UnaryOperation) String() string {
	return unaryOp(o.Op, o.Expression)
}

func (s StaticNil) String() string {
	return "nil"
}

func (s StaticNil) EncodeToString(_ bool) string {
	return s.String()
}

func (s StaticInt) String() string {
	return strconv.Itoa(s.Int)
}

func (s StaticInt) EncodeToString(_ bool) string {
	return s.String()
}

func (s StaticFloat) String() string {
	return strconv.FormatFloat(s.Float, 'g', -1, 64)
}

func (s StaticFloat) EncodeToString(_ bool) string {
	return s.String()
}

func (s StaticString) String() string {
	return s.EncodeToString(true)
}

func (s StaticString) EncodeToString(quotes bool) string {
	if quotes {
		return "`" + s.Str + "`"
	}
	return s.Str
}

func (s StaticBool) String() string {
	return strconv.FormatBool(s.Bool)
}

func (s StaticBool) EncodeToString(_ bool) string {
	return s.String()
}

func (s StaticDuration) String() string {
	return s.Duration.String()
}

func (s StaticDuration) EncodeToString(_ bool) string {
	return s.String()
}

func (s StaticStatus) String() string {
	return s.Status.String()
}

func (s StaticStatus) EncodeToString(_ bool) string {
	return s.String()
}

func (s StaticKind) String() string {
	return s.Kind.String()
}

func (s StaticKind) EncodeToString(_ bool) string {
	return s.String()
}

func (a Attribute) String() string {
	scopes := []string{}
	if a.Parent {
		scopes = append(scopes, "parent")
	}

	if a.Scope != AttributeScopeNone {
		attributeScope := a.Scope.String()
		scopes = append(scopes, attributeScope)
	}

	att := a.Name
	if a.Intrinsic != IntrinsicNone {
		att = a.Intrinsic.String()
	}

	scope := ""
	if len(scopes) > 0 {
		scope = strings.Join(scopes, ".") + "."
	}

	// Top-level attributes get a "." but top-level intrinsics don't
	if scope == "" && a.Intrinsic == IntrinsicNone && len(att) > 0 {
		scope += "."
	}

	return scope + att
}

func (a MetricsAggregate) String() string {
	s := strings.Builder{}

	s.WriteString(a.op.String())
	s.WriteString("(")
	switch a.op {
	case metricsAggregateQuantileOverTime:
		s.WriteString(a.attr.String())
		s.WriteString(",")
		for i, f := range a.floats {
			s.WriteString(strconv.FormatFloat(f, 'f', 5, 64))
			if i < len(a.floats)-1 {
				s.WriteString(",")
			}
		}
	}
	s.WriteString(")")

	if len(a.by) > 0 {
		s.WriteString("by(")
		for i, b := range a.by {
			s.WriteString(b.String())
			if i < len(a.by)-1 {
				s.WriteString(",")
			}
		}
		s.WriteString(")")
	}
	return s.String()
}

func (h *Hints) String() string {
	hh := make([]string, 0, len(h.Hints))
	for _, hn := range h.Hints {
		hh = append(hh, hn.Name+"="+hn.Value.String())
	}
	return "with(" + strings.Join(hh, ",") + ")"
}

func binaryOp(op Operator, lhs Element, rhs Element) string {
	return wrapElement(lhs) + " " + op.String() + " " + wrapElement(rhs)
}

func unaryOp(op Operator, e Element) string {
	return op.String() + wrapElement(e)
}

func wrapElement(e Element) string {
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
