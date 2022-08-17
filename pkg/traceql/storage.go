package traceql

import "context"

type Operation int

const (
	OperationNone Operation = iota
	OperationEq
	OperationLT
	OperationGT
	OperationIn
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
	Spans   []*Span
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
