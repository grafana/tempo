package deployments

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestWriteMetrics(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		countTraces := 10
		countSpans := 3 * 3 * countTraces // batches * spans per batch * traces
		countBytes := 0

		for range countTraces {
			trace := test.MakeTraceWithSpanCount(3, 3, test.ValidTraceID(nil))
			countBytes += trace.Size()

			require.NoError(t, h.WriteTempoProtoTraces(trace, ""))
		}

		h.WaitTracesQueryable(t, countTraces)

		// distributor
		distributor := h.Services[util.ServiceDistributor]
		assertMetricEquals(t, distributor, "tempo_distributor_produce_batches_total", float64(countTraces), nil)
		assertMetricEquals(t, distributor, "tempo_distributor_spans_received_total", float64(countSpans), nil)

		sums, err := distributor.SumMetrics([]string{"tempo_distributor_bytes_received_total"})
		require.NoError(t, err)
		actualBytes := sums[0]
		assertMetricInDelta(t, distributor, "tempo_distributor_bytes_received_total", float64(countBytes), .1*float64(countBytes), nil)
		assertMetricInDelta(t, distributor, "tempo_distributor_ingress_bytes_total", float64(countBytes), .1*float64(countBytes), nil)

		// livestore
		livestore := h.Services[util.ServiceLiveStoreZoneA]
		assertMetricInDelta(t, livestore, "tempo_live_store_bytes_received_total", float64(countBytes), actualBytes, nil)
		assertMetricEquals(t, livestore, "tempo_live_store_partition_owned", float64(1), nil)

		// we may not have yet completed a block so wait for this one
		err = livestore.WaitSumMetricsWithOptions(e2e.Greater(0),
			[]string{"tempo_live_store_blocks_completed_total"},
			e2e.WaitMissingMetrics,
		)
		require.NoError(t, err)

		h.WaitTracesWrittenToBackend(t, countTraces)

		// blockbuilder
		blockbuilder := h.Services[util.ServiceBlockBuilder]
		assertMetricEquals(t, blockbuilder, "tempo_block_builder_flushed_blocks", float64(1), nil)
		assertMetricEquals(t, blockbuilder, "tempo_block_builder_owned_partitions", float64(1), nil)
	})
}

func TestReadMetrics(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		// http
		apiClient := h.APIClientHTTP("")

		_, err := apiClient.MetricsQueryInstant("{} | rate()", time.Now().Add(-time.Hour).Unix(), time.Now().Unix(), 0)
		require.NoError(t, err)
		_, err = apiClient.MetricsQueryRange("{} | rate()", time.Now().Add(-time.Hour).Unix(), time.Now().Unix(), "1m", 0)
		require.NoError(t, err)
		id, err := info.TraceID()
		require.NoError(t, err)
		_, err = apiClient.QueryTraceV2(tempoUtil.TraceIDToHexString(id))
		require.NoError(t, err)
		_, err = apiClient.SearchTagValuesV2("span.foo", "")
		require.NoError(t, err)
		_, err = apiClient.SearchTagsV2()
		require.NoError(t, err)
		_, err = apiClient.SearchTraceQL("{}")
		require.NoError(t, err)

		// grpc
		grpcClient, ctx, err := h.APIClientGRPC("")
		require.NoError(t, err)
		instantClient, err := grpcClient.MetricsQueryInstant(ctx, &tempopb.QueryInstantRequest{
			Query: "{} | rate()",
			Start: uint64(time.Now().Add(-time.Hour).Local().UnixNano()),
			End:   uint64(time.Now().Local().UnixNano()),
		})
		require.NoError(t, err)
		err = drainStreamingClient(instantClient)
		require.NoError(t, err)

		rangeClient, err := grpcClient.MetricsQueryRange(ctx, &tempopb.QueryRangeRequest{
			Query:     "{} | rate()",
			Start:     uint64(time.Now().Add(-time.Hour).Local().UnixNano()),
			End:       uint64(time.Now().Local().UnixNano()),
			Step:      uint64(time.Minute.Nanoseconds()),
			Exemplars: 0,
		})
		require.NoError(t, err)
		err = drainStreamingClient(rangeClient)
		require.NoError(t, err)

		searchTagValuesClient, err := grpcClient.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
			TagName: "span.foo",
			Query:   "",
		})
		require.NoError(t, err)
		err = drainStreamingClient(searchTagValuesClient)
		require.NoError(t, err)

		searchTagsClient, err := grpcClient.SearchTagsV2(ctx, &tempopb.SearchTagsRequest{
			Query: "{}",
		})
		require.NoError(t, err)
		err = drainStreamingClient(searchTagsClient)
		require.NoError(t, err)

		searchClient, err := grpcClient.Search(ctx, &tempopb.SearchRequest{
			Query: "{}",
		})
		require.NoError(t, err)
		err = drainStreamingClient(searchClient)
		require.NoError(t, err)

		// query-frontend
		queryFrontend := h.Services[util.ServiceQueryFrontend]

		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_total", float64(4), map[string]string{"op": "metrics", "result": "completed"})
		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_total", float64(4), map[string]string{"op": "metadata", "result": "completed"})
		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_total", float64(2), map[string]string{"op": "search", "result": "completed"})
		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_total", float64(1), map[string]string{"op": "traces", "result": "completed"})

		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_within_slo_total", float64(4), map[string]string{"op": "metrics", "result": "completed"})
		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_within_slo_total", float64(4), map[string]string{"op": "metadata", "result": "completed"})
		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_within_slo_total", float64(2), map[string]string{"op": "search", "result": "completed"})
		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_queries_within_slo_total", float64(1), map[string]string{"op": "traces", "result": "completed"})

		assertMetricGreater(t, queryFrontend, "tempo_query_frontend_bytes_inspected_total", float64(0), map[string]string{"op": "metadata"})
		assertMetricEquals(t, queryFrontend, "tempo_query_frontend_bytes_inspected_total", float64(0), map[string]string{"op": "metrics"}) // metrics is 0? possibly a bug?
		assertMetricGreater(t, queryFrontend, "tempo_query_frontend_bytes_inspected_total", float64(0), map[string]string{"op": "search"})
		assertMetricGreater(t, queryFrontend, "tempo_query_frontend_bytes_inspected_total", float64(0), map[string]string{"op": "traces"})

		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "GET", "route": "api_metrics_query", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "GET", "route": "api_metrics_query_range", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "GET", "route": "api_v2_traces_traceid", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "GET", "route": "api_v2_search_tag_tagname_values", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "GET", "route": "api_v2_search_tags", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "GET", "route": "api_search", "status_code": "200", "ws": "false"})

		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "gRPC", "route": "/tempopb.StreamingQuerier/MetricsQueryInstant", "status_code": "success", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "gRPC", "route": "/tempopb.StreamingQuerier/MetricsQueryRange", "status_code": "success", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "gRPC", "route": "/tempopb.StreamingQuerier/SearchTagValuesV2", "status_code": "success", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "gRPC", "route": "/tempopb.StreamingQuerier/SearchTagsV2", "status_code": "success", "ws": "false"})
		assertMetricCountEquals(t, queryFrontend, "tempo_request_duration_seconds", float64(1), map[string]string{"method": "gRPC", "route": "/tempopb.StreamingQuerier/Search", "status_code": "success", "ws": "false"})

		// querier
		querier := h.Services[util.ServiceQuerier]
		assertMetricGreater(t, querier, "tempo_querier_worker_request_executed_total", float64(0), nil)
		assertMetricEquals(t, querier, "tempo_querier_livestore_clients", float64(2), nil)

		// testing these counts might be a little brittle and we find it's not worthwhile
		assertMetricCountEquals(t, querier, "tempo_request_duration_seconds", float64(4), map[string]string{"method": "GET", "route": "querier_api_metrics_query_range", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, querier, "tempo_request_duration_seconds", float64(50), map[string]string{"method": "GET", "route": "querier_api_v2_traces_traceid", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, querier, "tempo_request_duration_seconds", float64(2), map[string]string{"method": "GET", "route": "querier_api_v2_search_tag_tagname_values", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, querier, "tempo_request_duration_seconds", float64(2), map[string]string{"method": "GET", "route": "querier_api_v2_search_tags", "status_code": "200", "ws": "false"})
		assertMetricCountEquals(t, querier, "tempo_request_duration_seconds", float64(6), map[string]string{"method": "GET", "route": "querier_api_search", "status_code": "200", "ws": "false"})
	})
}

type streamingClient[T any] interface {
	Recv() (*T, error)
}

func drainStreamingClient[T any](stream streamingClient[T]) error {
	for {
		resp, err := stream.Recv()
		if resp != nil {
			break
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func assertMetricEquals(t *testing.T, service *e2e.HTTPService, metric string, expected float64, labelValues map[string]string) {
	t.Helper()
	opts := []e2e.MetricsOption{}
	if len(labelValues) > 0 {
		matchers := make([]*labels.Matcher, 0, len(labelValues))
		for name, value := range labelValues {
			matchers = append(matchers, &labels.Matcher{Type: labels.MatchEqual, Name: name, Value: value})
		}
		opts = append(opts, e2e.WithLabelMatchers(matchers...))
	}
	sums, err := service.SumMetrics([]string{metric}, opts...)
	require.NoError(t, err)
	require.Equal(t, expected, sums[0])
}

// nolint:unparam
func assertMetricCountEquals(t *testing.T, service *e2e.HTTPService, metric string, expected float64, labelValues map[string]string) {
	t.Helper()
	opts := []e2e.MetricsOption{e2e.WithMetricCount}
	if len(labelValues) > 0 {
		matchers := make([]*labels.Matcher, 0, len(labelValues))
		for name, value := range labelValues {
			matchers = append(matchers, &labels.Matcher{Type: labels.MatchEqual, Name: name, Value: value})
		}
		opts = append(opts, e2e.WithLabelMatchers(matchers...))
	}
	sums, err := service.SumMetrics([]string{metric}, opts...)
	require.NoError(t, err)
	require.Equal(t, expected, sums[0])
}

func assertMetricInDelta(t *testing.T, service *e2e.HTTPService, metric string, expected, delta float64, labelValues map[string]string) {
	t.Helper()
	opts := []e2e.MetricsOption{}
	if len(labelValues) > 0 {
		matchers := make([]*labels.Matcher, 0, len(labelValues))
		for name, value := range labelValues {
			matchers = append(matchers, &labels.Matcher{Type: labels.MatchEqual, Name: name, Value: value})
		}
		opts = append(opts, e2e.WithLabelMatchers(matchers...))
	}
	sums, err := service.SumMetrics([]string{metric}, opts...)
	require.NoError(t, err)
	require.InDelta(t, expected, sums[0], delta)
}

// nolint:unparam
func assertMetricGreater(t *testing.T, service *e2e.HTTPService, metric string, minValue float64, labelValues map[string]string) {
	t.Helper()
	opts := []e2e.MetricsOption{}
	if len(labelValues) > 0 {
		matchers := make([]*labels.Matcher, 0, len(labelValues))
		for name, value := range labelValues {
			matchers = append(matchers, &labels.Matcher{Type: labels.MatchEqual, Name: name, Value: value})
		}
		opts = append(opts, e2e.WithLabelMatchers(matchers...))
	}
	sums, err := service.SumMetrics([]string{metric}, opts...)
	require.NoError(t, err)
	require.Greater(t, sums[0], minValue)
}
