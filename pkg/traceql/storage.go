package traceql

import (
	"context"
)

type Operands []Static

type Condition struct {
	Attribute Attribute
	Op        Operator
	Operands  Operands
}

// FilterSpans is a hint that allows the calling code to filter down spans to only
// those that metadata needs to be retrieved for. If the returned Spanset has no
// spans it is discarded and will not appear in FetchSpansResponse. The bool
// return value is used to indicate if the Fetcher should continue iterating or if
// it can bail out.
type FilterSpans func(*Spanset) ([]*Spanset, error)

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

	// For some implementations retrieving all of the metadata for the spans
	// can be quite costly. This hint allows the calling code to filter down
	// spans before the span metadata is fetched, but after the data requested
	// in the Conditions is fetched. If this is unset then all metadata
	// for all matching spansets is returned.
	// If this is set it must be called by the storage layer even if there is
	// no opportunity to pull metadata independently of span data.
	Filter FilterSpans
}

func (f *FetchSpansRequest) appendCondition(c ...Condition) {
	f.Conditions = append(f.Conditions, c...)
}

type Span interface {
	// these are the actual fields used by the engine to evaluate queries
	// if a Filter parameter is passed the spans returned will only have this field populated
	Attributes() map[Attribute]Static

	ID() []byte
	StartTimeUnixNanos() uint64
	EndtimeUnixNanos() uint64
}

type Spanset struct {
	// these fields are actually used by the engine to evaluate queries
	Scalar Static
	Spans  []Span

	TraceID            []byte
	RootSpanName       string
	RootServiceName    string
	StartTimeUnixNanos uint64
	DurationNanos      uint64
}

func (s *Spanset) clone() *Spanset {
	return &Spanset{
		TraceID:            s.TraceID,
		Scalar:             s.Scalar,
		RootSpanName:       s.RootSpanName,
		RootServiceName:    s.RootServiceName,
		StartTimeUnixNanos: s.StartTimeUnixNanos,
		DurationNanos:      s.DurationNanos,
		Spans:              s.Spans, // we're not deep cloning into the spans themselves
	}
}

type SpansetIterator interface {
	Next(context.Context) (*Spanset, error)
	Close()
}

type FetchSpansResponse struct {
	Results SpansetIterator
	Bytes   func() uint64
}

type SpansetFetcher interface {
	Fetch(context.Context, FetchSpansRequest) (FetchSpansResponse, error)
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
