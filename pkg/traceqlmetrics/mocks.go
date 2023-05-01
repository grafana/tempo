package traceqlmetrics

import (
	"context"

	"github.com/grafana/tempo/pkg/traceql"
)

type mockSpan struct {
	duration uint64
	attrs    map[traceql.Attribute]traceql.Static
}

var _ traceql.Span = (*mockSpan)(nil)

func NewMockSpan(duration uint64, nameValuePairs ...string) *mockSpan {
	m := &mockSpan{
		duration: duration,
		attrs:    map[traceql.Attribute]traceql.Static{},
	}

	for i := 0; i < len(nameValuePairs); i += 2 {
		attr := traceql.MustParseIdentifier(nameValuePairs[i])
		value := traceql.NewStaticString(nameValuePairs[i+1])
		m.attrs[attr] = value
	}

	return m
}

func (m *mockSpan) Attributes() map[traceql.Attribute]traceql.Static { return m.attrs }
func (m *mockSpan) ID() []byte                                       { return nil }
func (m *mockSpan) StartTimeUnixNanos() uint64                       { return 0 }
func (m *mockSpan) DurationNanos() uint64                            { return m.duration }

type mockFetcher struct {
	filter   traceql.FilterSpans
	Spansets []*traceql.Spanset
}

var _ traceql.SpansetFetcher = (*mockFetcher)(nil)
var _ traceql.SpansetIterator = (*mockFetcher)(nil)

func (m *mockFetcher) Fetch(_ context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
	m.filter = req.Filter
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
