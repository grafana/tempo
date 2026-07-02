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

	"github.com/grafana/tempo/pkg/tempopb"
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
	}, start, end)
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
