package traceql

import (
	"context"
	"fmt"
)

type Operands []Static

type Condition struct {
	Attribute Attribute
	Op        Operator
	Operands  Operands
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

// ExtractCondition from the first spanset filter in the traceql query.
// For testing purposes.
func ExtractCondition(query string) (cond Condition, err error) {
	ast, err := Parse(query)
	if err != nil {
		return cond, err
	}

	f, ok := ast.Pipeline.Elements[0].(SpansetFilter)
	if !ok {
		return Condition{}, fmt.Errorf("first pipeline element is not a SpansetFilter")
	}

	switch e := f.Expression.(type) {
	case BinaryOperation:
		cond.Attribute = e.LHS.(Attribute)
		cond.Op = e.Op
		cond.Operands = []Static{e.RHS.(Static)}
	case Attribute:
		cond.Attribute = e
		cond.Op = OpNone
		cond.Operands = nil
	}

	return
}
