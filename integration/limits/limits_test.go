package e2e

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/e2e"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	configIngestPartialSucess = "config-ingest-partial-success.yaml"
	configQueryRate           = "config-query-rate.yaml"
	configIngest              = "config-ingest.yaml"
)

func TestIngestionLimits(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		ConfigOverlay: configIngest,
	}, func(h *util.TempoHarness) {
		// should fail b/c the trace is too large. each batch should be ~70 bytes
		batch := util.MakeThriftBatchWithSpanCount(2)
		require.NoError(t, h.JaegerExporter.EmitBatch(context.Background(), batch), "max trace size")

		// push a trace
		require.NoError(t, h.JaegerExporter.EmitBatch(context.Background(), util.MakeThriftBatchWithSpanCount(1)))

		// should fail b/c this will be too many traces
		batch = util.MakeThriftBatch()
		require.NoError(t, h.JaegerExporter.EmitBatch(context.Background(), batch), "too many traces")

		// should fail b/c due to ingestion rate limit
		batch = util.MakeThriftBatchWithSpanCount(10)
		err := h.JaegerExporter.EmitBatch(context.Background(), batch)
		require.Error(t, err)

		// this error must have a retryinfo as expected in otel collector code: https://github.com/open-telemetry/opentelemetry-collector/blob/d7b49df5d9e922df6ce56ad4b64ee1c79f9dbdbe/exporter/otlpexporter/otlp.go#L172
		st, ok := status.FromError(err)
		require.True(t, ok)
		foundRetryInfo := false
		for _, detail := range st.Details() {
			if _, ok := detail.(*errdetails.RetryInfo); ok {
				foundRetryInfo = true
				break
			}
		}
		require.True(t, foundRetryInfo)

		// test limit metrics
		liveStore := h.Services[util.ServiceLiveStoreZoneA]
		err = liveStore.WaitSumMetricsWithOptions(e2e.Equals(2),
			[]string{"tempo_discarded_spans_total"},
			e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "trace_too_large")),
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)
		err = liveStore.WaitSumMetricsWithOptions(e2e.Equals(1),
			[]string{"tempo_discarded_spans_total"},
			e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "live_traces_exceeded")),
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)
		err = h.Services[util.ServiceDistributor].WaitSumMetricsWithOptions(e2e.Equals(10),
			[]string{"tempo_discarded_spans_total"},
			e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "rate_limited")),
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)
	})
}

func TestOTLPLimits(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		ConfigOverlay: configIngest,
	}, func(h *util.TempoHarness) {
		protoSpans := test.MakeProtoSpans(100)

		// gRPC
		grpcClient := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(h.Services[util.ServiceDistributor].Endpoint(4317)),
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{Enabled: false}),
		)
		require.NoError(t, grpcClient.Start(context.Background()))

		grpcErr := grpcClient.UploadTraces(context.Background(), protoSpans)
		assert.Error(t, grpcErr)
		require.Equal(t, codes.ResourceExhausted, status.Code(grpcErr))

		// HTTP
		httpClient := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(h.Services[util.ServiceDistributor].Endpoint(4318)),
			otlptracehttp.WithInsecure(),
			otlptracehttp.WithRetry(otlptracehttp.RetryConfig{Enabled: false}),
		)
		require.NoError(t, httpClient.Start(context.Background()))

		httpErr := httpClient.UploadTraces(context.Background(), protoSpans)
		assert.Error(t, httpErr)
		require.Contains(t, httpErr.Error(), "retry-able request failure")
	})
}

func TestOTLPLimitsVanillaClient(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		ConfigOverlay: configIngest,
	}, func(h *util.TempoHarness) {
		trace := test.MakeTrace(10, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

		testCases := []struct {
			name    string
			payload func() []byte
			headers map[string]string
		}{
			// TODO There is an issue when sending the payload in json format. The server returns a 200 instead of a 429.
			// {
			// 	"JSON format",
			// 	func() []byte {
			// 		b := &bytes.Buffer{}
			// 		err := (&jsonpb.Marshaler{}).Marshal(b, trace)
			// 		require.NoError(t, err)
			// 		return b.Bytes()
			// 	},
			// 	map[string]string{
			// 		"Content-Type": "application/json",
			// 	},
			// },
			{
				"Proto format",
				func() []byte {
					b, err := trace.Marshal()
					require.NoError(t, err)
					return b
				},
				map[string]string{
					"Content-Type": "application/x-protobuf",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPost, "http://"+h.Services[util.ServiceDistributor].Endpoint(4318)+"/v1/traces", bytes.NewReader(tc.payload()))
				require.NoError(t, err)
				for k, v := range tc.headers {
					req.Header.Set(k, v)
				}

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()
				bodyBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				fmt.Println(string(bodyBytes))

				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
			})
		}
	})
}

func TestQueryLimits(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		batch := util.MakeThriftBatchWithSpanCount(5)
		require.NoError(t, h.JaegerExporter.EmitBatch(context.Background(), batch))

		// retroactively make the trace too large so it will fail on querying
		h.UpdateOverrides(map[string]*overrides.Overrides{
			"single-tenant": {
				Global: overrides.GlobalOverrides{
					MaxBytesPerTrace: 1,
				},
			},
		})

		// calc trace id
		traceID := [16]byte{}
		binary.BigEndian.PutUint64(traceID[:8], uint64(batch.Spans[0].TraceIdHigh))
		binary.BigEndian.PutUint64(traceID[8:], uint64(batch.Spans[0].TraceIdLow))

		// now try to query it back. this should fail b/c the trace is too large
		client := h.HTTPClient

		// wait for live store to ingest traces
		liveStore := h.Services[util.ServiceLiveStoreZoneA]
		err := liveStore.WaitSumMetricsWithOptions(e2e.Greater(0),
			[]string{"tempo_live_store_traces_created_total"},
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)

		_, err = client.QueryTrace(tempoUtil.TraceIDToHexString(traceID[:]))
		require.ErrorContains(t, err, trace.ErrTraceTooLarge.Error())
		require.ErrorContains(t, err, "failed with response: 422") // confirm frontend returns 422

		// wait for live store to complete the block
		err = liveStore.WaitSumMetricsWithOptions(e2e.Greater(0),
			[]string{"tempo_live_store_blocks_completed_total"},
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)

		_, err = client.QueryTrace(tempoUtil.TraceIDToHexString(traceID[:]))
		require.ErrorContains(t, err, trace.ErrTraceTooLarge.Error())
		require.ErrorContains(t, err, "failed with response: 422") // confirm frontend returns 422
	})
}

func TestLimitsPartialSuccess(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		ConfigOverlay: configIngestPartialSucess,
	}, func(h *util.TempoHarness) {
		// make request
		traceIDs := make([][]byte, 6)
		for index := range traceIDs {
			traceID := make([]byte, 16)
			_, err := crand.Read(traceID)
			require.NoError(t, err)
			traceIDs[index] = traceID
		}

		// 3 traces with trace_too_large and 3 with no error
		spanCountsByTrace := []int{1, 4, 1, 5, 6, 1}
		req := test.MakeReqWithMultipleTraceWithSpanCount(spanCountsByTrace, traceIDs)

		b, err := req.Marshal()
		require.NoError(t, err)

		// unmarshal into otlp proto
		traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
		require.NoError(t, err)
		require.NotNil(t, traces)

		ctx := user.InjectOrgID(context.Background(), tempoUtil.FakeTenantID)
		ctx, err = user.InjectIntoGRPCRequest(ctx)
		require.NoError(t, err)

		// send traces to tempo
		// partial success = no error
		err = h.OTLPExporter.ConsumeTraces(ctx, traces)
		require.NoError(t, err)

		// shutdown to ensure traces are flushed
		require.NoError(t, h.OTLPExporter.Shutdown(context.Background()))

		// wait for live store to ingest traces
		liveStore := h.Services[util.ServiceLiveStoreZoneA]
		err = liveStore.WaitSumMetricsWithOptions(e2e.Greater(0),
			[]string{"tempo_live_store_traces_created_total"},
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)

		// query for the traces that didn't trigger an error
		client := h.HTTPClient
		for i, count := range spanCountsByTrace {
			if count == 1 {
				result, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceIDs[i]))
				require.NoError(t, err)
				assert.Equal(t, 1, len(result.ResourceSpans))
			}
		}

		// test metrics
		// 3 traces with trace_too_large each with 4+5+6 spans
		err = liveStore.WaitSumMetricsWithOptions(e2e.Equals(15),
			[]string{"tempo_discarded_spans_total"},
			e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "trace_too_large")),
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)
	})
}

func TestQueryRateLimits(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		ConfigOverlay: configQueryRate,
	}, func(h *util.TempoHarness) {
		// todo: do we even need to push a trace?
		batch := util.MakeThriftBatchWithSpanCount(5)
		require.NoError(t, h.JaegerExporter.EmitBatch(context.Background(), batch))

		// now try to query it back. this should fail b/c the frontend queue doesn't have room
		client := h.HTTPClient

		// 429 HTTP Trace ID Lookup
		traceID := []byte{0x01, 0x02}
		_, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceID))
		require.ErrorContains(t, err, "job queue full")
		require.ErrorContains(t, err, "failed with response: 429")

		start := time.Now().Add(-1 * time.Hour).Unix()
		end := time.Now().Add(1 * time.Hour).Unix()

		// 429 HTTP Search
		_, err = client.SearchTraceQLWithRange("{}", start, end)
		require.ErrorContains(t, err, "job queue full")
		require.ErrorContains(t, err, "failed with response: 429")

		// 429 GRPC Search
		grpcClient, err := util.NewSearchGRPCClient(context.Background(), h.Services[util.ServiceQueryFrontend].Endpoint(3200))
		require.NoError(t, err)

		resp, err := grpcClient.Search(context.Background(), &tempopb.SearchRequest{
			Query: "{}",
			Start: uint32(start),
			End:   uint32(end),
		})
		require.NoError(t, err)

		// loop until we get io.EOF or an error
		for {
			_, err = resp.Recv()
			if err != nil {
				break
			}
		}
		require.ErrorContains(t, err, "job queue full")
		require.ErrorContains(t, err, "code = ResourceExhausted")
	})
}
