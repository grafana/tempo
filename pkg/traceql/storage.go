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

	// Hints

	// By default the storage layer fetches spans meeting any of the criteria.
	// This hint is for common cases like { x && y && z } where the storage layer
	// can make extra optimizations by returning only spansets that meet
	// all criteria.
	AllConditions bool
}

func (f *FetchSpansRequest) appendCondition(c ...Condition) {
	f.Conditions = append(f.Conditions, c...)
}

type Span struct {
	ID                 []byte
	StartTimeUnixNanos uint64
	EndtimeUnixNanos   uint64
	Attributes         map[Attribute]Static
}

type Spanset struct {
	TraceID            []byte
	RootSpanName       string
	RootServiceName    string
	StartTimeUnixNanos uint64
	DurationNanos      uint64
	Spans              []Span
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

// MustExtractCondition from the first spanset filter in the traceql query.
// I.e. given a query { .foo=`bar`} it will extract the condition attr
// foo EQ str(bar). Panics if the query fails to parse or contains a
// different structure. For testing purposes.
func MustExtractCondition(query string) Condition {
	c, err := ExtractCondition(query)
	if err != nil {
		panic(err)
	}
	return c
}

// ExtractCondition from the first spanset filter in the traceql query.
// I.e. given a query { .foo=`bar`} it will extract the condition attr
// foo EQ str(bar). For testing purposes.
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
