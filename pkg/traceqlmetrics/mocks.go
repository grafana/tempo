package traceqlmetrics

import (
	"context"

	"github.com/grafana/tempo/pkg/traceql"
)

type mockSpan struct {
	start    uint64
	duration uint64
	attrs    map[traceql.Attribute]traceql.Static
}

var _ traceql.Span = (*mockSpan)(nil)

func newMockSpan() *mockSpan {
	return &mockSpan{
		attrs: map[traceql.Attribute]traceql.Static{},
	}
}

func (m *mockSpan) WithDuration(d uint64) *mockSpan {
	m.duration = d
	return m
}

func (m *mockSpan) WithAttributes(nameValuePairs ...string) *mockSpan {
	for i := 0; i < len(nameValuePairs); i += 2 {
		attr := traceql.MustParseIdentifier(nameValuePairs[i])
		value := traceql.NewStaticString(nameValuePairs[i+1])
		m.attrs[attr] = value
	}

	return m
}

func (m *mockSpan) WithStart(t uint64) *mockSpan {
	m.start = t
	return m
}

func (m *mockSpan) WithErr() *mockSpan {
	m.attrs[traceql.NewIntrinsic(traceql.IntrinsicStatus)] = traceql.NewStaticStatus(traceql.StatusError)
	return m
}

func (m *mockSpan) Attributes() map[traceql.Attribute]traceql.Static { return m.attrs }
func (m *mockSpan) ID() []byte                                       { return nil }
func (m *mockSpan) StartTimeUnixNanos() uint64                       { return m.start }
func (m *mockSpan) DurationNanos() uint64                            { return m.duration }
func (m *mockSpan) DescendantOf([]traceql.Span, []traceql.Span, bool, bool, []traceql.Span) []traceql.Span {
	return nil
}

func (m *mockSpan) SiblingOf([]traceql.Span, []traceql.Span, bool, []traceql.Span) []traceql.Span {
	return nil
}

func (m *mockSpan) ChildOf([]traceql.Span, []traceql.Span, bool, bool, []traceql.Span) []traceql.Span {
	return nil
}

type mockFetcher struct {
	filter   traceql.SecondPassFn
	Spansets []*traceql.Spanset
}

var (
	_ traceql.SpansetFetcher  = (*mockFetcher)(nil)
	_ traceql.SpansetIterator = (*mockFetcher)(nil)
)

func (m *mockFetcher) Fetch(_ context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
	m.filter = req.SecondPass
	return traceql.FetchSpansResponse{
		Results: m,
	}, nil
}

func (m *mockFetcher) Next(context.Context) (*traceql.Spanset, error) {
	if len(m.Spansets) == 0 {
		return nil, nil
	}

	// Pop first
	s := m.Spansets[0]
	m.Spansets = m.Spansets[1:]

	// Return as-is
	if m.filter == nil {
		return s, nil
	}

	// Run through filter which may return multiple
	ss, err := m.filter(s)
	if err != nil {
		return nil, err
	}

	// Just return the first - this will need to change if we ever use
	// this mock for more advanced stuff
	if len(ss) == 0 {
		return nil, nil
	}

	return ss[0], nil
}

func (m *mockFetcher) Close() {}
