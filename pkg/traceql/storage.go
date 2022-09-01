package traceql

import (
	"context"
	"fmt"
)

type Operation int

const (
	OperationNone Operation = iota
	OperationEq
	OperationLT
	OperationGT
	OperationIn
	OperationRegexIn
)

type Condition struct {
	Selector  string
	Operation Operation
	Operands  []interface{}
}

type FetchSpansRequest struct {
	StartTimeUnixNanos uint64
	EndTimeUnixNanos   uint64
	Conditions         []Condition
}

type Span struct {
	ID                 []byte
	StartTimeUnixNanos uint64
	EndtimeUnixNanos   uint64
	Attributes         map[string]interface{}
}

type Spanset struct {
	TraceID []byte
	Spans   []Span
}

type SpansetIterator interface {
	Next(context.Context) (*Spanset, error)
}

type FetchSpansResponse struct {
	Results SpansetIterator
}

type SpansetFetcher interface {
	Fetch(context.Context, FetchSpansRequest) (FetchSpansResponse, error)
}

func ExtractCondition(query string) (cond Condition, err error) {
	ast, err := Parse(query)
	if err != nil {
		return cond, err
	}

	f, ok := ast.p.p[0].(SpansetFilter)
	if !ok {
		return Condition{}, fmt.Errorf("first pipeline element is not a SpansetFilter")
	}

	setAttribute := func(a Attribute) {
		// LHS = attribute or instrinsic
		if a.intrinsic == intrinsicNone {
			switch a.scope {
			case attributeScopeNone:
				cond.Selector = "."
			case attributeScopeResource:
				cond.Selector = "resource."
			case attributeScopeSpan:
				cond.Selector = "span."
			}
			cond.Selector += a.name
		} else {
			cond.Selector = a.intrinsic.String()
		}
	}

	setOperator := func(op Operator) {
		switch op {
		case opEqual:
			cond.Operation = OperationEq
		case opGreater:
			cond.Operation = OperationGT
		case opLess:
			cond.Operation = OperationLT
		case opRegex:
			cond.Operation = OperationRegexIn
		default:
			err = fmt.Errorf("traceql operator not supported for storage testing: %s", op.String())
		}
	}

	setOperand := func(s Static) {
		// Operands
		switch s.staticType {
		case typeString:
			cond.Operands = append(cond.Operands, s.s)
		case typeInt:
			cond.Operands = append(cond.Operands, s.n)
		case typeFloat:
			cond.Operands = append(cond.Operands, s.f)
		case typeBoolean:
			cond.Operands = append(cond.Operands, s.b)
		case typeDuration:
			cond.Operands = append(cond.Operands, uint64(s.d.Nanoseconds()))
		default:
			err = fmt.Errorf("traceql operand not supported for storage testing: %s", s.String())
		}
	}

	switch e := f.e.(type) {
	case BinaryOperation:
		setAttribute(e.lhs.(Attribute))
		setOperator(e.op)
		setOperand(e.rhs.(Static))
	case Attribute:
		setAttribute(e)
		cond.Operation = OperationNone
		cond.Operands = nil
	}

	return
}
