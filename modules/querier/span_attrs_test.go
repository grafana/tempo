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
		name       string
		start      func(context.Context) error
		expectedOp string
	}{
		{
			name: "search block",
			start: func(ctx context.Context) error {
				_, span, _, err := startSearchBlockSpan(ctx, "Querier.SearchBlock", &tempopb.SearchBlockRequest{
					BlockID:       "block-a",
					Version:       "vParquet4",
					StartPage:     2,
					PagesToSearch: 3,
					Size_:         1000,
					TotalRecords:  10,
				})
				if err == nil {
					span.End()
				}
				return err
			},
			expectedOp: "search",
		},
		{
			name: "search tags block",
			start: func(ctx context.Context) error {
				_, span, _, err := startTagsBlockSpan(ctx, "Querier.SearchTagsBlocks", &tempopb.SearchTagsBlockRequest{
					BlockID:       "block-a",
					Version:       "vParquet4",
					StartPage:     2,
					PagesToSearch: 3,
					Size_:         1000,
					TotalRecords:  10,
				})
				if err == nil {
					span.End()
				}
				return err
			},
			expectedOp: "metadata",
		},
		{
			name: "search tag values block",
			start: func(ctx context.Context) error {
				_, span, _, err := startTagValuesBlockSpan(ctx, "Querier.SearchTagValuesBlocks", &tempopb.SearchTagValuesBlockRequest{
					BlockID:       "block-a",
					Version:       "vParquet4",
					StartPage:     2,
					PagesToSearch: 3,
					Size_:         1000,
					TotalRecords:  10,
				})
				if err == nil {
					span.End()
				}
				return err
			},
			expectedOp: "metadata",
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
			require.Equal(t, tt.expectedOp, attrs["op"].AsString())
			require.Equal(t, int64(1), attrs["blocks"].AsInt64())
			require.Equal(t, int64(300), attrs["blockBytes"].AsInt64()) // 3 of 10 row groups of 1000 bytes
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
				BlockID:       "invalid",
				Version:       "vParquet4",
				Size_:         1000,
				StartPage:     2,
				PagesToSearch: 2,
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

			attrs := map[string]attribute.Value{}
			for _, attr := range queryRangeSpan.Attributes() {
				attrs[string(attr.Key)] = attr.Value
			}
			require.Equal(t, "metrics", attrs["op"].AsString())
			// the request doesn't have the total number of row groups, so blockBytes can't be calculated
			require.NotContains(t, attrs, "blockBytes")

			if tt.name == "block" {
				require.Equal(t, tt.req.BlockID, attrs["blockID"].AsString())
				require.Equal(t, tt.req.Version, attrs["version"].AsString())
				require.Equal(t, int64(1), attrs["blocks"].AsInt64())
			} else {
				require.NotContains(t, attrs, "blocks")
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
		Blocks: &tempopb.TraceByIDBlocks{Blocks: []*tempopb.TraceByIDBlock{
			{Size_: 100}, {Size_: 250}, {Size_: 1000},
		}},
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
	require.Equal(t, "traces", attrs["op"].AsString())
	require.Equal(t, int64(3), attrs["blocks"].AsInt64())
	require.Equal(t, int64(1350), attrs["blockBytes"].AsInt64())
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
	const noBlockBytes = -1

	tests := []struct {
		name          string
		blockSize     uint64
		startPage     uint32
		pagesToSearch uint32
		totalRecords  uint32
		expectedBytes int64
	}{
		{
			name:          "part of the block",
			blockSize:     1000,
			startPage:     2,
			pagesToSearch: 2,
			totalRecords:  10,
			expectedBytes: 200,
		},
		{
			name:          "whole block",
			blockSize:     1000,
			pagesToSearch: 10,
			totalRecords:  10,
			expectedBytes: 1000,
		},
		{
			name:          "last job covers less than requested",
			blockSize:     1000,
			startPage:     8,
			pagesToSearch: 4,
			totalRecords:  10,
			expectedBytes: 200,
		},
		{
			name:          "job past the end of the block",
			blockSize:     1000,
			startPage:     12,
			pagesToSearch: 2,
			totalRecords:  10,
			expectedBytes: 0,
		},
		{
			name:          "bytes are not rounded down per row group",
			blockSize:     1000,
			pagesToSearch: 3,
			totalRecords:  7,
			expectedBytes: 428,
		},
		{
			// not a realistic block, it guards that blockSize*rowGroups is not computed in one step, which overflows a uint64
			name:          "big block does not overflow",
			blockSize:     100 << 40,
			pagesToSearch: 400_000,
			totalRecords:  1_000_000,
			expectedBytes: 40 << 40,
		},
		{
			name:          "request without total records",
			blockSize:     1000,
			pagesToSearch: 2,
			expectedBytes: noBlockBytes,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := map[string]attribute.Value{}
			for _, attr := range blockJobAttributes(tc.blockSize, tc.startPage, tc.pagesToSearch, tc.totalRecords) {
				attrs[string(attr.Key)] = attr.Value
			}

			require.Equal(t, int64(1), attrs["blocks"].AsInt64())
			if tc.expectedBytes == noBlockBytes {
				require.NotContains(t, attrs, "blockBytes")
				return
			}
			require.Equal(t, tc.expectedBytes, attrs["blockBytes"].AsInt64())
		})
	}
}

func TestStartRequestSpanOp(t *testing.T) {
	tests := []struct {
		name       string
		start      func(context.Context) (oteltrace.Span, error)
		expectedOp string
	}{
		{
			name: "search request",
			start: func(ctx context.Context) (oteltrace.Span, error) {
				_, span, _, err := startSearchRequestSpan(ctx, "Querier.SearchRecent", &tempopb.SearchRequest{})
				return span, err
			},
			expectedOp: "search",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

			oldTracer := tracer
			tracer = tp.Tracer("test")
			defer func() { tracer = oldTracer }()

			span, err := tt.start(user.InjectOrgID(context.Background(), "tenant-a"))
			require.NoError(t, err)
			span.End()

			spans := recorder.Ended()
			require.Len(t, spans, 1)
			require.Contains(t, spans[0].Attributes(), attribute.String("op", tt.expectedOp))
		})
	}
}
