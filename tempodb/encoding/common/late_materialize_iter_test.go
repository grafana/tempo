package common

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

// mockSpan implements traceql.Span for testing
type mockSpan struct {
	id     []byte
	rowNum parquetquery.RowNumber
	attrs  map[traceql.Attribute]traceql.Static
}

func (s *mockSpan) AllAttributes() map[traceql.Attribute]traceql.Static { return s.attrs }
func (s *mockSpan) AllAttributesFunc(f func(traceql.Attribute, traceql.Static)) {
	for k, v := range s.attrs {
		f(k, v)
	}
}
func (s *mockSpan) AttributeFor(a traceql.Attribute) (traceql.Static, bool) {
	v, ok := s.attrs[a]
	return v, ok
}
func (s *mockSpan) ID() []byte                     { return s.id }
func (s *mockSpan) StartTimeUnixNanos() uint64     { return 0 }
func (s *mockSpan) DurationNanos() uint64          { return 0 }
func (s *mockSpan) RowNum() parquetquery.RowNumber { return s.rowNum }
func (s *mockSpan) SiblingOf(_, _ []traceql.Span, _ bool, _ bool, buf []traceql.Span) []traceql.Span {
	return buf
}
func (s *mockSpan) DescendantOf(_, _ []traceql.Span, _, _, _ bool, buf []traceql.Span) []traceql.Span {
	return buf
}
func (s *mockSpan) ChildOf(_, _ []traceql.Span, _, _, _ bool, buf []traceql.Span) []traceql.Span {
	return buf
}

// mockFetchIter simulates a parquetquery.Iterator for the fetch side
type mockFetchIter struct {
	results []*parquetquery.IteratorResult
	idx     int
}

func (m *mockFetchIter) Next() (*parquetquery.IteratorResult, error) {
	if m.idx >= len(m.results) {
		return nil, nil
	}
	r := m.results[m.idx]
	m.idx++
	return r, nil
}

func (m *mockFetchIter) SeekTo(t parquetquery.RowNumber, d int) (*parquetquery.IteratorResult, error) {
	for m.idx < len(m.results) {
		r := m.results[m.idx]
		if parquetquery.CompareRowNumbers(d, r.RowNumber, t) >= 0 {
			m.idx++
			return r, nil
		}
		m.idx++
	}
	return nil, nil
}

func (m *mockFetchIter) Close()         {}
func (m *mockFetchIter) String() string { return "mockFetchIter" }

// mockSpansetIter returns pre-built spansets
type mockSpansetIter struct {
	spansets []*traceql.Spanset
	idx      int
}

func (m *mockSpansetIter) Next(_ context.Context) (*traceql.Spanset, error) {
	if m.idx >= len(m.spansets) {
		return nil, nil
	}
	ss := m.spansets[m.idx]
	m.idx++
	return ss, nil
}

func (m *mockSpansetIter) Close() {}

func TestLateMaterializeIter(t *testing.T) {
	rn1 := parquetquery.EmptyRowNumber()
	rn1[0] = 0
	rn1[1] = 0
	rn1[2] = 0
	rn1[3] = 0

	span1 := &mockSpan{
		id:     []byte{1},
		rowNum: rn1,
		attrs:  map[traceql.Attribute]traceql.Static{},
	}

	driving := &mockSpansetIter{
		spansets: []*traceql.Spanset{
			{TraceID: []byte{0, 1}, Spans: []traceql.Span{span1}},
		},
	}

	// Fetch iter returns a result at rn1
	fetchResult := &parquetquery.IteratorResult{RowNumber: rn1}

	fetchIter := &mockFetchIter{
		results: []*parquetquery.IteratorResult{fetchResult},
	}

	iter := newLateMaterializeIter(driving, fetchIter, 3)

	ctx := context.Background()
	ss, err := iter.Next(ctx)
	require.NoError(t, err)
	require.NotNil(t, ss)
	require.Len(t, ss.Spans, 1)

	// Second call returns nil (exhausted)
	ss, err = iter.Next(ctx)
	require.NoError(t, err)
	require.Nil(t, ss)

	iter.Close()
}
