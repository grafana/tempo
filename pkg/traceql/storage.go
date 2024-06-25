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

func SearchMetaConditions() []Condition {
	return []Condition{
		{NewIntrinsic(IntrinsicTraceRootService), OpNone, nil},
		{NewIntrinsic(IntrinsicTraceRootSpan), OpNone, nil},
		{NewIntrinsic(IntrinsicTraceDuration), OpNone, nil},
		{NewIntrinsic(IntrinsicTraceID), OpNone, nil},
		{NewIntrinsic(IntrinsicTraceStartTime), OpNone, nil},
		{NewIntrinsic(IntrinsicSpanID), OpNone, nil},
		{NewIntrinsic(IntrinsicSpanStartTime), OpNone, nil},
		{NewIntrinsic(IntrinsicDuration), OpNone, nil},
		{NewIntrinsic(IntrinsicServiceStats), OpNone, nil},
	}
}

func SearchMetaConditionsWithout(remove []Condition) []Condition {
	metaConds := SearchMetaConditions()
	retConds := make([]Condition, 0, len(metaConds))
	for _, c := range metaConds {
		// if we can't find c in the remove conditions then add it to retConds
		found := false
		for _, e := range remove {
			if e.Attribute == c.Attribute {
				found = true
				break
			}
		}
		if !found {
			retConds = append(retConds, c)
		}
	}

	return retConds
}

// SecondPassFn is a method that is called in between the first and second
// pass of a fetch spans request. See below.
type SecondPassFn func(*Spanset) ([]*Spanset, error)

type FetchSpansRequest struct {
	StartTimeUnixNanos uint64
	EndTimeUnixNanos   uint64
	Conditions         []Condition

	// mdisibio - Better to push trace by ID filtering into Conditions with a new between op?
	ShardID    uint32
	ShardCount uint32

	// Hints

	// By default the storage layer fetches spans meeting any of the criteria.
	// This hint is for common cases like { x && y && z } where the storage layer
	// can make extra optimizations by returning only spansets that meet
	// all criteria.
	AllConditions bool

	// SecondPassFn and Conditions allow a caller to retrieve one set of data
	// in the first pass, filter using the SecondPassFn callback and then
	// request a different set of data in the second pass. This is particularly
	// useful for retrieving data required to resolve a TraceQL query in the first
	// pass and only selecting metadata in the second pass.
	// TODO: extend this to an arbitrary number of passes
	SecondPass           SecondPassFn
	SecondPassConditions []Condition
	SecondPassSelectAll  bool // Ignore second pass conditions and select all attributes
}

func (f *FetchSpansRequest) appendCondition(c ...Condition) {
	f.Conditions = append(f.Conditions, c...)
}

func (f *FetchSpansRequest) HasAttribute(a Attribute) bool {
	for _, cc := range f.Conditions {
		if cc.Attribute == a {
			return true
		}
	}
	for _, cc := range f.SecondPassConditions {
		if cc.Attribute == a {
			return true
		}
	}

	return false
}

type Span interface {
	// AttributeFor returns the attribute for the given key. If the attribute is not found then
	// the second return value will be false.
	AttributeFor(Attribute) (Static, bool)
	// AllAttributes returns a map of all attributes for this span. AllAttributes should be used sparingly
	// and is expected to be significantly slower than AttributeFor.
	AllAttributes() map[Attribute]Static
	// AllAttributesFunc is a way to access all attributes for this span, letting the span determine the
	// optimal method. Avoids allocating a map like AllAttributes.
	AllAttributesFunc(func(Attribute, Static))

	ID() []byte
	StartTimeUnixNanos() uint64
	DurationNanos() uint64

	// SiblingOf returns all spans on the RHS side that have siblings in the LHS. If falseForAll is true
	// then the returned spans will be those that do not have siblings in the LHS. buffer is an optional
	// buffer to use to avoid allocations.
	SiblingOf(lhs []Span, rhs []Span, falseForAll bool, union bool, buffer []Span) []Span
	// DescendantOf returns all spans on the RHS side that have descendants in the LHS. If falseForAll is true
	// then the returned spans will be those that do not have descendants in the LHS. invert is used to invert
	// the relationship. If invert is true then this will behave like "AncestorOf". buffer is an optional
	// buffer to use to avoid allocations.
	DescendantOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, union bool, buffer []Span) []Span
	// ChildOf returns all spans on the RHS side that have children in the LHS. If falseForAll is true
	// then the returned spans will be those that do not have children in the LHS. invert is used to invert
	// the relationship. If invert is true then this will behave like "ParentOf". buffer is an optional
	// buffer to use to avoid allocations.
	ChildOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, union bool, buffer []Span) []Span
}

// should we just make matched a field on the spanset instead of a special attribute?
const attributeMatched = "__matched"

type SpansetAttribute struct {
	Name string
	Val  Static
}

type ServiceStats struct {
	SpanCount  uint32
	ErrorCount uint32
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
	ServiceStats       map[string]ServiceStats
	Attributes         []*SpansetAttribute

	// Set this function to provide upstream callers with a method to
	// release this spanset and all its spans when finished. This method will be
	// called with the spanset itself as the argument. This is done for a worthwhile
	// memory savings as the same function pointer can then be reused across spansets.
	ReleaseFn func(*Spanset)
}

func (s *Spanset) AddAttribute(key string, value Static) {
	s.Attributes = append(s.Attributes, &SpansetAttribute{Name: key, Val: value})
}

// Release the spanset and all its span. This is just a wrapper of ReleaseFn that
// performs nil checks.
func (s *Spanset) Release() {
	if s.ReleaseFn != nil {
		s.ReleaseFn(s)
	}
}

func (s *Spanset) clone() *Spanset {
	ss := *s
	return &ss
}

type SpansetIterator interface {
	Next(context.Context) (*Spanset, error)
	Close()
}

type FetchSpansResponse struct {
	Results SpansetIterator
	// callback to get the size of data read during Fetch
	Bytes func() uint64
}

type SpansetFetcher interface {
	Fetch(context.Context, FetchSpansRequest) (FetchSpansResponse, error)
}

// FetchTagValuesCallback is called to collect unique tag values.
// Returns true if it has exceeded the maximum number of results.
type FetchTagValuesCallback func(static Static) bool

type FetchTagValuesRequest struct {
	Conditions []Condition
	TagName    Attribute
	// TODO: Add start and end time?
}

type TagValuesFetcher interface {
	Fetch(context.Context, FetchTagValuesRequest, FetchTagValuesCallback) error
}

type TagValuesFetcherWrapper struct {
	f func(context.Context, FetchTagValuesRequest, FetchTagValuesCallback) error
}

var _ TagValuesFetcher = (*TagValuesFetcherWrapper)(nil)

func NewTagValuesFetcherWrapper(f func(context.Context, FetchTagValuesRequest, FetchTagValuesCallback) error) TagValuesFetcher {
	return TagValuesFetcherWrapper{f}
}

func (s TagValuesFetcherWrapper) Fetch(ctx context.Context, request FetchTagValuesRequest, callback FetchTagValuesCallback) error {
	return s.f(ctx, request, callback)
}

// MustExtractFetchSpansRequestWithMetadata parses the given traceql query and returns
// the storage layer conditions. Panics if the query fails to parse.
func MustExtractFetchSpansRequestWithMetadata(query string) FetchSpansRequest {
	c, err := ExtractFetchSpansRequest(query)
	if err != nil {
		panic(err)
	}
	c.SecondPass = func(s *Spanset) ([]*Spanset, error) { return []*Spanset{s}, nil }
	c.SecondPassConditions = SearchMetaConditions()
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
