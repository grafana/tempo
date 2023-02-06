package traceql

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/pkg/parquetquery"
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
	// jpe need rownumber to go fetch metadata later?
	RowNum     parquetquery.RowNumber
	Attributes map[Attribute]Static
}

type Spanset struct {
	Scalar Static
	Spans  []Span
}

type SpanMetadata struct {
	ID                 []byte
	StartTimeUnixNanos uint64
	EndtimeUnixNanos   uint64
	Attributes         map[Attribute]Static // jpe copy from Span in the fetch part?
}

type SpansetMetadata struct {
	TraceID            []byte
	RootSpanName       string
	RootServiceName    string
	StartTimeUnixNanos uint64
	DurationNanos      uint64
	Spans              []SpanMetadata
}

type SpansetIterator interface {
	Next(context.Context) (*Spanset, error)
}

type FetchSpansResponse struct {
	Results SpansetIterator
	Bytes   func() uint64
}

type SpansetFetcher interface {
	Fetch(context.Context, FetchSpansRequest) (FetchSpansResponse, error)
	FetchMetadata(context.Context, []Spanset) ([]SpansetMetadata, error) // jpe review this method signature
}

// MustExtractFetchSpansRequest parses the given traceql query and returns
// the storage layer conditions. Panics if the query fails to parse.
func MustExtractFetchSpansRequest(query string) FetchSpansRequest {
	c, err := ExtractFetchSpansRequest(query)
	if err != nil {
		panic(err)
	}
	return c
}

// ExtractFetchSpansRequest parses the given traceql query and returns
// the storage layer conditions. Returns an error if the query fails to parse.
func ExtractFetchSpansRequest(query string) (FetchSpansRequest, error) {
	ast, err := Parse(query)
	if err != nil {
		return FetchSpansRequest{}, err
	}

	req := FetchSpansRequest{
		AllConditions: true,
	}

	ast.Pipeline.extractConditions(&req)
	return req, nil
}

type SpansetFetcherWrapper struct {
	f func(ctx context.Context, req FetchSpansRequest) (FetchSpansResponse, error)
}

var _ = (SpansetFetcher)(&SpansetFetcherWrapper{})

func NewSpansetFetcherWrapper(f func(ctx context.Context, req FetchSpansRequest) (FetchSpansResponse, error)) SpansetFetcher {
	return SpansetFetcherWrapper{f}
}

func (s SpansetFetcherWrapper) Fetch(ctx context.Context, request FetchSpansRequest) (FetchSpansResponse, error) {
	return s.f(ctx, request)
}

func (s SpansetFetcherWrapper) FetchMetadata(ctx context.Context, spansets []Spanset) ([]SpansetMetadata, error) {
	// jpe implement
	return nil, fmt.Errorf("not implemented")
}
