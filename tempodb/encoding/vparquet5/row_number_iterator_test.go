package vparquet5

import (
	"bytes"
	"context"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"

	pq "github.com/grafana/tempo/pkg/parquetquery"
)

func TestVirtualRowNumberIterator_Next(t *testing.T) {
	traces := []Trace{
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{
			{SpanCount: 1, Spans: make([]Span, 1)},
			{SpanCount: 2, Spans: make([]Span, 2)},
		}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 0, Spans: make([]Span, 0)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 3, Spans: make([]Span, 3)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 0, Spans: make([]Span, 0)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 1, Spans: make([]Span, 1)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{
			{SpanCount: 4, Spans: make([]Span, 4)},
			{SpanCount: 5, Spans: make([]Span, 5)},
			{SpanCount: 6, Spans: make([]Span, 6)},
		}}}},
	}

	rowNumbers := []pq.RowNumber{
		{0, 0, 0, 0, -1, -1, -1, -1},
		{0, 0, 1, 0, -1, -1, -1, -1},
		{0, 0, 1, 1, -1, -1, -1, -1},
		{1, 0, 0, -1, -1, -1, -1, -1},
		{2, 0, 0, 0, -1, -1, -1, -1},
		{2, 0, 0, 1, -1, -1, -1, -1},
		{2, 0, 0, 2, -1, -1, -1, -1},
		{3, 0, 0, -1, -1, -1, -1, -1},
		{4, 0, 0, 0, -1, -1, -1, -1},
		{5, 0, 0, 0, -1, -1, -1, -1},
		{5, 0, 0, 1, -1, -1, -1, -1},
		{5, 0, 0, 2, -1, -1, -1, -1},
		{5, 0, 0, 3, -1, -1, -1, -1},
		{5, 0, 1, 0, -1, -1, -1, -1},
		{5, 0, 1, 1, -1, -1, -1, -1},
		{5, 0, 1, 2, -1, -1, -1, -1},
		{5, 0, 1, 3, -1, -1, -1, -1},
		{5, 0, 1, 4, -1, -1, -1, -1},
		{5, 0, 2, 0, -1, -1, -1, -1},
		{5, 0, 2, 1, -1, -1, -1, -1},
		{5, 0, 2, 2, -1, -1, -1, -1},
		{5, 0, 2, 3, -1, -1, -1, -1},
		{5, 0, 2, 4, -1, -1, -1, -1},
		{5, 0, 2, 5, -1, -1, -1, -1},
	}

	pf := makeTestFile(t, traces)
	makeIterator := makeIterFunc(context.Background(), pf.RowGroups(), pf)

	expectIter := makeIterator(columnPathSpanStatusCode, nil, "statusCode") // actual iterator over spans as comparison

	iter := newVirtualRowNumberIterator(makeIterator(columnPathScopeSpansSpanCount, nil, "spanCount"), DefinitionLevelResourceSpansILSSpan)

	for _, expRowNumber := range rowNumbers {
		res, err := iter.Next()
		require.NoError(t, err)
		require.Equal(t, expRowNumber, res.RowNumber)

		exp, err := expectIter.Next()
		require.NoError(t, err)
		require.Equal(t, exp.RowNumber, res.RowNumber)
	}

	res, err := iter.Next()
	require.NoError(t, err)
	require.Nil(t, res)

	exp, err := expectIter.Next()
	require.NoError(t, err)
	require.Nil(t, exp)
}

func TestVirtualRowNumberIterator_SeekTo(t *testing.T) {
	traces := []Trace{
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 1, Spans: make([]Span, 1)}, {SpanCount: 2, Spans: make([]Span, 2)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 0, Spans: make([]Span, 0)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 3, Spans: make([]Span, 3)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 0, Spans: make([]Span, 0)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 1, Spans: make([]Span, 1)}}}}},
		{ResourceSpans: []ResourceSpans{{ScopeSpans: []ScopeSpans{{SpanCount: 4, Spans: make([]Span, 4)}, {SpanCount: 5, Spans: make([]Span, 5)}, {SpanCount: 6, Spans: make([]Span, 6)}}}}},
	}
	seekPositions := []struct {
		seekRow      pq.RowNumber
		seekLevel    int
		expectedRow  pq.RowNumber
		expectedNext pq.RowNumber
	}{
		{
			seekRow:      pq.RowNumber{0, 0, 1, 0, -1, -1, -1, -1},
			seekLevel:    DefinitionLevelResourceSpansILSSpan,
			expectedRow:  pq.RowNumber{0, 0, 1, 0, -1, -1, -1, -1},
			expectedNext: pq.RowNumber{0, 0, 1, 1, -1, -1, -1, -1},
		},
		{
			seekRow:      pq.RowNumber{2, 0, 0, 0, -1, -1, -1, -1},
			seekLevel:    DefinitionLevelResourceSpansILSSpan,
			expectedRow:  pq.RowNumber{2, 0, 0, 0, -1, -1, -1, -1},
			expectedNext: pq.RowNumber{2, 0, 0, 1, -1, -1, -1, -1},
		},
		{
			seekRow:      pq.RowNumber{2, 0, 0, 3, -1, -1, -1, -1}, // seek past last virtual row
			seekLevel:    DefinitionLevelResourceSpansILSSpan,
			expectedRow:  pq.RowNumber{3, 0, 0, -1, -1, -1, -1, -1},
			expectedNext: pq.RowNumber{4, 0, 0, 0, -1, -1, -1, -1},
		},
		{
			seekRow:      pq.RowNumber{5, 0, 1, 0, 0, -1, -1, -1}, // seek on higher level
			seekLevel:    DefinitionLevelResourceSpansILSSpan + 1,
			expectedRow:  pq.RowNumber{5, 0, 1, 1, -1, -1, -1, -1},
			expectedNext: pq.RowNumber{5, 0, 1, 2, -1, -1, -1, -1},
		},
		{
			seekRow:      pq.RowNumber{5, 0, 1, 4, 0, -1, -1, -1}, // seek past last virtual row on higher level
			seekLevel:    DefinitionLevelResourceSpansILSSpan + 1,
			expectedRow:  pq.RowNumber{5, 0, 2, 0, -1, -1, -1, -1},
			expectedNext: pq.RowNumber{5, 0, 2, 1, -1, -1, -1, -1},
		},
		{
			seekRow:      pq.RowNumber{5, 0, 2, 4, -1, -1, -1, -1}, // seek on higher level and row number with fewer levels
			seekLevel:    DefinitionLevelResourceSpansILSSpan + 1,
			expectedRow:  pq.RowNumber{5, 0, 2, 4, -1, -1, -1, -1},
			expectedNext: pq.RowNumber{5, 0, 2, 5, -1, -1, -1, -1},
		},
	}

	pf := makeTestFile(t, traces)
	makeIterator := makeIterFunc(context.Background(), pf.RowGroups(), pf)

	expectIter := makeIterator(columnPathSpanStatusCode, nil, "statusCode") // actual iterator over spans as comparison

	iter := newVirtualRowNumberIterator(makeIterator(columnPathScopeSpansSpanCount, nil, "spanCount"), DefinitionLevelResourceSpansILSSpan)

	for _, seekPos := range seekPositions {
		res, err := iter.SeekTo(seekPos.seekRow, seekPos.seekLevel)
		require.NoError(t, err)
		require.Equal(t, seekPos.expectedRow, res.RowNumber)

		exp, err := expectIter.SeekTo(seekPos.seekRow, seekPos.seekLevel)
		require.NoError(t, err)
		require.Equal(t, exp.RowNumber, res.RowNumber)

		res, err = iter.Next()
		require.NoError(t, err)
		require.Equal(t, seekPos.expectedNext, res.RowNumber)

		exp, err = expectIter.Next()
		require.NoError(t, err)
		require.Equal(t, exp.RowNumber, res.RowNumber)
	}
}

func makeTestFile(t testing.TB, traces []Trace) *parquet.File {
	var buf bytes.Buffer

	w := parquet.NewGenericWriter[Trace](&buf)
	n, err := w.Write(traces)
	require.NoError(t, err)
	require.Equal(t, len(traces), n)

	err = w.Close()
	require.NoError(t, err)

	data := buf.Bytes()
	pf, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	return pf
}
