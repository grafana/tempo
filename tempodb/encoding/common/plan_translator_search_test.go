package common

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

func TestSearchEvaluatable(t *testing.T) {
	span1 := &mockSpan{
		id:     []byte{1},
		rowNum: parquetquery.EmptyRowNumber(),
		attrs:  map[traceql.Attribute]traceql.Static{},
	}

	driving := &mockSpansetIter{
		spansets: []*traceql.Spanset{
			{
				TraceID:            []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
				RootServiceName:    "test-service",
				RootSpanName:       "test-span",
				StartTimeUnixNanos: 1000,
				DurationNanos:      500_000_000,
				Spans:              []traceql.Span{span1},
			},
		},
	}

	eval := newSearchEvaluatable(driving, 10)
	err := eval.Do(context.Background())
	require.NoError(t, err)

	resp := eval.Response()
	require.NotNil(t, resp)
	require.Len(t, resp.Traces, 1)
	require.Equal(t, "test-service", resp.Traces[0].RootServiceName)
	require.Equal(t, "test-span", resp.Traces[0].RootTraceName)
}

func TestSearchEvaluatableLimit(t *testing.T) {
	spansets := make([]*traceql.Spanset, 5)
	for i := range spansets {
		spansets[i] = &traceql.Spanset{
			TraceID:            []byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			RootServiceName:    "svc",
			StartTimeUnixNanos: uint64(i * 1000),
			Spans:              []traceql.Span{&mockSpan{id: []byte{byte(i)}, rowNum: parquetquery.EmptyRowNumber(), attrs: map[traceql.Attribute]traceql.Static{}}},
		}
	}

	driving := &mockSpansetIter{spansets: spansets}
	eval := newSearchEvaluatable(driving, 3)
	err := eval.Do(context.Background())
	require.NoError(t, err)

	resp := eval.Response()
	require.NotNil(t, resp)
	require.Len(t, resp.Traces, 3) // limited to 3
}
