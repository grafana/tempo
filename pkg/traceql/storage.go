package traceql

import (
	"context"
	"fmt"
)

type Condition struct {
	Selector string
	Op       Operator
	Operands []interface{}
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

	f, ok := ast.Pipeline.Elements[0].(SpansetFilter)
	if !ok {
		return Condition{}, fmt.Errorf("first pipeline element is not a SpansetFilter")
	}

	setAttribute := func(a Attribute) {
		// LHS = attribute or instrinsic
		if a.Intrinsic == IntrinsicNone {
			switch a.Scope {
			case AttributeScopeNone:
				cond.Selector = "."
			case AttributeScopeResource:
				cond.Selector = "resource."
			case AttributeScopeSpan:
				cond.Selector = "span."
			}
			cond.Selector += a.Name
		} else {
			cond.Selector = a.Intrinsic.String()
		}
	}

	setOperand := func(s Static) {
		// Operands
		switch s.Type {
		case TypeString:
			cond.Operands = append(cond.Operands, s.S)
		case TypeInt:
			cond.Operands = append(cond.Operands, s.N)
		case TypeFloat:
			cond.Operands = append(cond.Operands, s.F)
		case TypeBoolean:
			cond.Operands = append(cond.Operands, s.B)
		case TypeDuration:
			cond.Operands = append(cond.Operands, uint64(s.D.Nanoseconds()))
		default:
			err = fmt.Errorf("traceql operand not supported for storage testing: %s", s.String())
		}
	}

	switch e := f.Expression.(type) {
	case BinaryOperation:
		setAttribute(e.LHS.(Attribute))
		setOperand(e.RHS.(Static))
		cond.Op = e.Op
	case Attribute:
		setAttribute(e)
		cond.Op = OpNone
		cond.Operands = nil
	}

	return
}
