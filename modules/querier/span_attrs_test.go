package querier

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/tempopb"
)

func TestQuerierSpanAttributesAndMetrics(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	_, span := tp.Tracer("test").Start(context.Background(), "test-span")
	setQuerierSpanAttributes(span, "tenant-a", `{ status = error }`, attribute.String("blockID", "block-a"))
	setQuerierSpanMetrics(span, &tempopb.SearchMetrics{
		InspectedBytes:  123,
		InspectedTraces: 2,
		InspectedSpans:  3,
		BackendReads:    4,
		BackendBytes:    567,
		TotalBlocks:     6,
		CompletedJobs:   7,
		TotalJobs:       8,
		TotalBlockBytes: 9,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricCacheHits: 10,
		},
	})
	span.End()

	spans := recorder.Ended()
	require.Len(t, spans, 1)

	attrs := map[string]attribute.Value{}
	for _, attr := range spans[0].Attributes() {
		attrs[string(attr.Key)] = attr.Value
	}

	require.Equal(t, "tenant-a", attrs["tenant"].AsString())
	require.Equal(t, `{ status = error }`, attrs["query"].AsString())
	require.Equal(t, "block-a", attrs["blockID"].AsString())
	require.Equal(t, int64(123), attrs["inspectedBytes"].AsInt64())
	require.Equal(t, int64(2), attrs["inspectedTraces"].AsInt64())
	require.Equal(t, int64(3), attrs["inspectedSpans"].AsInt64())
	require.Equal(t, int64(4), attrs["backendReads"].AsInt64())
	require.Equal(t, int64(567), attrs["backendBytes"].AsInt64())
	require.Equal(t, int64(10), attrs["additionalMetrics.cacheHits"].AsInt64())
}

func TestFinishQuerierSpanHandlesTypedNilTraceByIDMetrics(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	_, span := tp.Tracer("test").Start(context.Background(), "test-span")
	var resp *tempopb.TraceByIDResponse
	require.NotPanics(t, func() {
		finishQuerierSpan(span, nil, resp.GetMetrics())
	})

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Empty(t, spans[0].Attributes())
	require.Empty(t, spans[0].Events())
}

func TestStartBlockSpanWithoutSearchRequest(t *testing.T) {
	tests := []struct {
		name  string
		start func(context.Context) error
	}{
		{
			name: "search block",
			start: func(ctx context.Context) error {
				_, span, _, err := startSearchBlockSpan(ctx, "Querier.SearchBlock", &tempopb.SearchBlockRequest{
					BlockID:       "block-a",
					Version:       "vParquet4",
					StartPage:     2,
					PagesToSearch: 3,
				})
				if err == nil {
					span.End()
				}
				return err
			},
		},
		{
			name: "search tags block",
			start: func(ctx context.Context) error {
				_, span, _, err := startTagsBlockSpan(ctx, "Querier.SearchTagsBlocks", &tempopb.SearchTagsBlockRequest{
					BlockID:       "block-a",
					Version:       "vParquet4",
					StartPage:     2,
					PagesToSearch: 3,
				})
				if err == nil {
					span.End()
				}
				return err
			},
		},
		{
			name: "search tag values block",
			start: func(ctx context.Context) error {
				_, span, _, err := startTagValuesBlockSpan(ctx, "Querier.SearchTagValuesBlocks", &tempopb.SearchTagValuesBlockRequest{
					BlockID:       "block-a",
					Version:       "vParquet4",
					StartPage:     2,
					PagesToSearch: 3,
				})
				if err == nil {
					span.End()
				}
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

			oldTracer := tracer
			tracer = tp.Tracer("test")
			defer func() { tracer = oldTracer }()

			ctx := user.InjectOrgID(context.Background(), "tenant-a")
			require.NoError(t, tt.start(ctx))

			spans := recorder.Ended()
			require.Len(t, spans, 1)

			attrs := map[string]attribute.Value{}
			for _, attr := range spans[0].Attributes() {
				attrs[string(attr.Key)] = attr.Value
			}
			require.Equal(t, "block-a", attrs["blockID"].AsString())
			require.Equal(t, "vParquet4", attrs["version"].AsString())
			require.Equal(t, int64(2), attrs["startPage"].AsInt64())
			require.Equal(t, int64(3), attrs["pagesToSearch"].AsInt64())
			require.NotContains(t, attrs, "query")
			require.NotContains(t, attrs, "scope")
			require.NotContains(t, attrs, "tagName")
			require.NotContains(t, attrs, "startUnixSeconds")
			require.NotContains(t, attrs, "endUnixSeconds")
			require.NotContains(t, attrs, "rangeSeconds")
		})
	}
}

func TestBlockSearchMethodsCreateSinglePublicSpan(t *testing.T) {
	tests := []struct {
		name     string
		spanName string
		wantErr  bool
		call     func(context.Context, *Querier) error
	}{
		{
			name:     "search tags",
			spanName: "Querier.SearchTagsBlocks",
			call: func(ctx context.Context, q *Querier) error {
				_, err := q.SearchTagsBlocks(ctx, &tempopb.SearchTagsBlockRequest{
					SearchReq: &tempopb.SearchTagsRequest{Scope: api.ParamScopeIntrinsic},
				})
				return err
			},
		},
		{
			name:     "search tag values",
			spanName: "Querier.SearchTagValuesBlocks",
			wantErr:  true,
			call: func(ctx context.Context, q *Querier) error {
				_, err := q.SearchTagValuesBlocks(ctx, &tempopb.SearchTagValuesBlockRequest{})
				return err
			},
		},
		{
			name:     "search tags v2",
			spanName: "Querier.SearchTagsBlocksV2",
			call: func(ctx context.Context, q *Querier) error {
				_, err := q.SearchTagsBlocksV2(ctx, &tempopb.SearchTagsBlockRequest{
					SearchReq: &tempopb.SearchTagsRequest{Scope: api.ParamScopeIntrinsic},
				})
				return err
			},
		},
		{
			name:     "search tag values v2",
			spanName: "Querier.SearchTagValuesBlocksV2",
			wantErr:  true,
			call: func(ctx context.Context, q *Querier) error {
				_, err := q.SearchTagValuesBlocksV2(ctx, &tempopb.SearchTagValuesBlockRequest{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

			oldTracer := tracer
			tracer = tp.Tracer("test")
			defer func() { tracer = oldTracer }()

			ctx := user.InjectOrgID(context.Background(), "tenant-a")
			err := tt.call(ctx, &Querier{})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			spans := recorder.Ended()
			require.Len(t, spans, 1)
			require.Equal(t, tt.spanName, spans[0].Name())
		})
	}
}

func TestQueryRangeCreatesOnlyPublicQueryRangeSpan(t *testing.T) {
	tests := []struct {
		name      string
		req       *tempopb.QueryRangeRequest
		spanCount int
	}{
		{
			name: "recent",
			req: &tempopb.QueryRangeRequest{
				Query:     "{} | rate()",
				QueryMode: QueryModeRecent,
				Start:     1,
				End:       2,
				Step:      1,
			},
			spanCount: 2, // QueryRange and forLiveStoreMetricsRing.
		},
		{
			name: "block",
			req: &tempopb.QueryRangeRequest{
				BlockID: "invalid",
				Version: "vParquet4",
			},
			spanCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

			oldTracer := tracer
			tracer = tp.Tracer("test")
			defer func() { tracer = oldTracer }()

			ctx := user.InjectOrgID(context.Background(), "tenant-a")
			_, err := (&Querier{}).QueryRange(ctx, tt.req)
			require.Error(t, err)

			spans := recorder.Ended()
			require.Len(t, spans, tt.spanCount)

			var queryRangeSpan sdktrace.ReadOnlySpan
			for _, span := range spans {
				require.NotEqual(t, "Querier.queryRangeRecent", span.Name())
				require.NotEqual(t, "Querier.queryBlock", span.Name())
				if span.Name() == "Querier.QueryRange" {
					require.Nil(t, queryRangeSpan)
					queryRangeSpan = span
				}
			}
			require.NotNil(t, queryRangeSpan)

			if tt.name == "block" {
				attrs := map[string]attribute.Value{}
				for _, attr := range queryRangeSpan.Attributes() {
					attrs[string(attr.Key)] = attr.Value
				}
				require.Equal(t, tt.req.BlockID, attrs["blockID"].AsString())
				require.Equal(t, tt.req.Version, attrs["version"].AsString())
			}
		})
	}
}

func TestTraceByIDSpanAttributesAndMetrics(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	oldTracer := tracer
	tracer = tp.Tracer("test")
	defer func() { tracer = oldTracer }()

	start := time.Unix(123, 0)
	end := time.Unix(456, 0)
	ctx := user.InjectOrgID(context.Background(), "tenant-a")
	_, span, _, err := startTraceByIDSpan(ctx, "Querier.FindTraceByID", &tempopb.TraceByIDRequest{
		TraceID:           []byte{0x01, 0x02, 0x03},
		BlockStart:        "block-start",
		BlockEnd:          "block-end",
		QueryMode:         QueryModeAll,
		AllowPartialTrace: true,
		Start:             start,
		End:               end,
	})
	require.NoError(t, err)
	finishQuerierSpan(span, nil, &tempopb.TraceByIDMetrics{
		InspectedBytes: 11,
		BackendReads:   12,
		BackendBytes:   13,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricCacheHits: 14,
		},
	})

	spans := recorder.Ended()
	require.Len(t, spans, 1)

	attrs := map[string]attribute.Value{}
	for _, attr := range spans[0].Attributes() {
		attrs[string(attr.Key)] = attr.Value
	}

	require.Equal(t, "tenant-a", attrs["tenant"].AsString())
	require.Equal(t, "010203", attrs["traceID"].AsString())
	require.Equal(t, QueryModeAll, attrs["queryMode"].AsString())
	require.Equal(t, "block-start", attrs["blockStart"].AsString())
	require.Equal(t, "block-end", attrs["blockEnd"].AsString())
	require.True(t, attrs["allowPartialTrace"].AsBool())
	require.Equal(t, start.UnixNano(), attrs["startUnixNanos"].AsInt64())
	require.Equal(t, end.UnixNano(), attrs["endUnixNanos"].AsInt64())
	require.Equal(t, end.Sub(start).Nanoseconds(), attrs["rangeNanos"].AsInt64())
	require.Equal(t, int64(11), attrs["inspectedBytes"].AsInt64())
	require.Equal(t, int64(12), attrs["backendReads"].AsInt64())
	require.Equal(t, int64(13), attrs["backendBytes"].AsInt64())
	require.Equal(t, int64(14), attrs["additionalMetrics.cacheHits"].AsInt64())
}

func TestBlockJobAttributes(t *testing.T) {
	tests := []struct {
		name          string
		blockSize     uint64
		startPage     uint32
		pagesToSearch uint32
		totalRecords  uint32
		expectedBytes *int64
	}{
		{
			name:          "part of the block",
			blockSize:     1000,
			startPage:     2,
			pagesToSearch: 2,
			totalRecords:  10,
			expectedBytes: ptr(int64(200)),
		},
		{
			name:          "whole block",
			blockSize:     1000,
			pagesToSearch: 10,
			totalRecords:  10,
			expectedBytes: ptr(int64(1000)),
		},
		{
			name:          "last job covers less than requested",
			blockSize:     1000,
			startPage:     8,
			pagesToSearch: 4,
			totalRecords:  10,
			expectedBytes: ptr(int64(200)),
		},
		{
			name:          "job past the end of the block",
			blockSize:     1000,
			startPage:     12,
			pagesToSearch: 2,
			totalRecords:  10,
			expectedBytes: ptr(int64(0)),
		},
		{
			name:          "job starts at the end of the block",
			blockSize:     1000,
			startPage:     10,
			pagesToSearch: 2,
			totalRecords:  10,
			expectedBytes: ptr(int64(0)),
		},
		{
			name:          "bytes are not rounded down per row group",
			blockSize:     1000,
			pagesToSearch: 3,
			totalRecords:  7,
			expectedBytes: ptr(int64(428)),
		},
		{
			name:          "big block does not overflow",
			blockSize:     100 << 40, // 100 TiB, multiplied with the row groups this is more than a uint64
			pagesToSearch: 400_000,
			totalRecords:  1_000_000,
			expectedBytes: ptr(int64(40 << 40)),
		},
		{
			name:          "request without total records",
			blockSize:     1000,
			pagesToSearch: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := map[string]attribute.Value{}
			for _, attr := range blockJobAttributes(api.OpSearch, tc.blockSize, tc.startPage, tc.pagesToSearch, tc.totalRecords) {
				attrs[string(attr.Key)] = attr.Value
			}

			require.Equal(t, api.OpSearch, attrs["op"].AsString())
			require.Equal(t, int64(1), attrs["blocks"].AsInt64())
			if tc.expectedBytes == nil {
				require.NotContains(t, attrs, "blockBytes")
				return
			}
			require.Contains(t, attrs, "blockBytes")
			require.Equal(t, *tc.expectedBytes, attrs["blockBytes"].AsInt64())
		})
	}
}

func TestJobSpansHaveEstimateInputs(t *testing.T) {
	tests := []struct {
		name  string
		start func(context.Context) (oteltrace.Span, error)
		// expected attributes, nil means that the attribute is not set
		expectedOp         string
		expectedBlocks     *int64
		expectedBlockBytes *int64
	}{
		{
			name: "search block",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startSearchBlockSpan(ctx, "Querier.SearchBlock", &tempopb.SearchBlockRequest{
					BlockID: "block-a", Size_: 1000, StartPage: 2, PagesToSearch: 2, TotalRecords: 10,
				})
				return span, err
			},
			expectedOp:         "search",
			expectedBlocks:     ptr(int64(1)),
			expectedBlockBytes: ptr(int64(200)),
		},
		{
			name: "search request",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startSearchRequestSpan(ctx, "Querier.SearchRecent", &tempopb.SearchRequest{})
				return span, err
			},
			expectedOp: "search",
		},
		{
			name: "search tags block",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startTagsBlockSpan(ctx, "Querier.SearchTagsBlocks", &tempopb.SearchTagsBlockRequest{
					BlockID: "block-a", Size_: 1000, StartPage: 5, PagesToSearch: 10, TotalRecords: 10,
				})
				return span, err
			},
			expectedOp:         "metadata",
			expectedBlocks:     ptr(int64(1)),
			expectedBlockBytes: ptr(int64(500)),
		},
		{
			name: "search tag values block",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startTagValuesBlockSpan(ctx, "Querier.SearchTagValuesBlocks", &tempopb.SearchTagValuesBlockRequest{
					BlockID: "block-a", Size_: 1000, PagesToSearch: 1, TotalRecords: 10,
				})
				return span, err
			},
			expectedOp:         "metadata",
			expectedBlocks:     ptr(int64(1)),
			expectedBlockBytes: ptr(int64(100)),
		},
		{
			name: "search tags request",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startTagsRequestSpan(ctx, "Querier.SearchTags", &tempopb.SearchTagsRequest{})
				return span, err
			},
			expectedOp: "metadata",
		},
		{
			name: "search tag values request",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startTagValuesRequestSpan(ctx, "Querier.SearchTagValues", &tempopb.SearchTagValuesRequest{})
				return span, err
			},
			expectedOp: "metadata",
		},
		{
			name: "trace by id with blocks",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startTraceByIDSpan(ctx, "Querier.FindTraceByID", &tempopb.TraceByIDRequest{
					QueryMode: QueryModeBlocks,
					Blocks: &tempopb.TraceByIDBlocks{Blocks: []*tempopb.TraceByIDBlock{
						{Size_: 100}, {Size_: 250}, {Size_: 1000},
					}},
				})
				return span, err
			},
			expectedOp:         "traces",
			expectedBlocks:     ptr(int64(3)),
			expectedBlockBytes: ptr(int64(1350)),
		},
		{
			name: "trace by id without blocks",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startTraceByIDSpan(ctx, "Querier.FindTraceByID", &tempopb.TraceByIDRequest{QueryMode: QueryModeIngesters})
				return span, err
			},
			expectedOp: "traces",
		},
		{
			name: "trace by id with empty list of blocks",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startTraceByIDSpan(ctx, "Querier.FindTraceByID", &tempopb.TraceByIDRequest{
					QueryMode: QueryModeBlocks,
					Blocks:    &tempopb.TraceByIDBlocks{},
				})
				return span, err
			},
			expectedOp:         "traces",
			expectedBlocks:     ptr(int64(0)),
			expectedBlockBytes: ptr(int64(0)),
		},
		{
			name: "query range recent",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startQueryRangeSpan(ctx, "Querier.QueryRange", &tempopb.QueryRangeRequest{QueryMode: QueryModeRecent})
				return span, err
			},
			expectedOp: "metrics",
		},
		{
			name: "query range block",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				req := &tempopb.QueryRangeRequest{BlockID: "block-a", Size_: 1000, StartPage: 2, PagesToSearch: 2}
				_, span, _, err := startQueryRangeSpan(ctx, "Querier.QueryRange", req)
				if err == nil {
					setQueryRangeBlockSpanAttributes(span, req)
				}
				return span, err
			},
			expectedOp:     "metrics",
			expectedBlocks: ptr(int64(1)),
			// the request doesn't have the total number of row groups of the block
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

			oldTracer := tracer
			tracer = tp.Tracer("test")
			defer func() { tracer = oldTracer }()

			span, err := tc.start(user.InjectOrgID(context.Background(), "tenant-a"))
			require.NoError(t, err)
			span.End()

			spans := recorder.Ended()
			require.Len(t, spans, 1)
			attrs := map[string]attribute.Value{}
			for _, attr := range spans[0].Attributes() {
				attrs[string(attr.Key)] = attr.Value
			}

			require.Equal(t, tc.expectedOp, attrs["op"].AsString())
			for key, expected := range map[string]*int64{"blocks": tc.expectedBlocks, "blockBytes": tc.expectedBlockBytes} {
				if expected == nil {
					require.NotContains(t, attrs, key)
					continue
				}
				require.Contains(t, attrs, key)
				require.Equal(t, *expected, attrs[key].AsInt64(), key)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
