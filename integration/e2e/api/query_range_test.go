package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configQueryRange                         = "config-query-range.yaml"
	configQueryRangeMaxSeries                = "config-query-range-max-series.yaml"
	configQueryRangeMaxSeriesDisabled        = "config-query-range-max-series-disabled.yaml"
	configQueryRangeMaxSeriesDisabledQuerier = "config-query-range-max-series-disabled-querier.yaml"
)

// Set debugMode to true to print the response body
var debugMode = false

func TestQueryRangeExemplars(t *testing.T) {
	t.Parallel()

	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRange, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	// send one batch every 500ms for 10 seconds
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(),
				util.MakeThriftBatchWithSpanCountAttributeAndName(
					1, "my operation",
					"res_val", "span_val",
					"res_attr", "span_attr",
				),
			))
			require.NoError(t, jaegerClient.EmitBatch(context.Background(),
				util.MakeThriftBatchWithSpanCountAttributeAndName(
					1, "my operation",
					"res_val2", "span_val2",
					"res_attr", "span_attr",
				),
			))
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	for _, exeplarsCase := range []struct {
		name              string
		exemplars         int
		expectedExemplars int
	}{
		{
			name:              "default",
			exemplars:         0,
			expectedExemplars: 100, // if set to 0, then limits to 100
		},
		{
			name:              "5 exemplar",
			exemplars:         5,
			expectedExemplars: 5,
		},
		{
			name:              "25 exemplars",
			exemplars:         25,
			expectedExemplars: 25,
		},
		{
			name:              "capped exemplars",
			exemplars:         1000,
			expectedExemplars: 100, // capped to 100
		},
	} {
		for _, query := range []string{
			"{} | rate()",
			"{} | compare({status=error})",
			"{} | count_over_time()",
			"{} | min_over_time(duration)",
			"{} | max_over_time(duration)",
			"{} | avg_over_time(duration)",
			"{} | sum_over_time(duration)",
			"{} | quantile_over_time(duration, .5)",
			"{} | quantile_over_time(duration, .5, 0.9, 0.99)",

			"{} | count_over_time() by (span.span_attr)",
			"{} | count_over_time() by (resource.res_attr)",
			"{} | count_over_time() by (.span_attr)",
			"{} | count_over_time() by (.res_attr)",

			"{} | histogram_over_time(duration)",
			"{} | count_over_time() by (status)",
			"{status != error} | count_over_time() by (status)",
		} {
			t.Run(fmt.Sprintf("%s: %s", exeplarsCase.name, query), func(t *testing.T) {
				queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), query, exeplarsCase.exemplars, debugMode)
				require.NotNil(t, queryRangeRes)
				require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)
				if query == "{} | quantile_over_time(duration, .5, 0.9, 0.99)" {
					// Bug: https://github.com/grafana/tempo/issues/5167
					t.Skip("Bug in quantile_over_time in calculating exemplars")
				}

				exemplarCount := 0

				for _, series := range queryRangeRes.GetSeries() {
					exemplarCount += len(series.GetExemplars())
				}
				assert.LessOrEqual(t, exemplarCount, exeplarsCase.expectedExemplars)
				assert.GreaterOrEqual(t, exemplarCount, 1)
			})
		}
	}

	// check exemplars in more detail
	for _, testCase := range []struct {
		query                   string
		targetAttribute         string
		targetExemplarAttribute string
	}{
		{
			query:                   "{} | quantile_over_time(duration, .9) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | quantile_over_time(duration, .9) by (resource.res_attr)",
			targetAttribute:         "resource.res_attr",
			targetExemplarAttribute: "resource.res_attr",
		},
		{
			query:                   "{} | quantile_over_time(duration, .9) by (.span_attr)",
			targetAttribute:         ".span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | quantile_over_time(duration, .9) by (.res_attr)",
			targetAttribute:         ".res_attr",
			targetExemplarAttribute: "resource.res_attr",
		},
		{
			query:                   "{} | rate() by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | count_over_time() by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | min_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | max_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | avg_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | sum_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
	} {
		t.Run(testCase.query, func(t *testing.T) {
			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), testCase.query, 100, debugMode)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, len(queryRangeRes.GetSeries()), 2)

			// Verify that all exemplars in this series belongs to the right series
			// by matching attribute values
			for _, series := range queryRangeRes.Series {
				// search attribute value for the series
				var expectedAttrValue string
				for _, label := range series.Labels {
					if label.Key == testCase.targetAttribute {
						expectedAttrValue = label.Value.GetStringValue()
						break
					}
				}
				require.NotEmpty(t, expectedAttrValue)

				// check attribute value in exemplars
				for _, exemplar := range series.Exemplars {
					var actualAttrValue string
					for _, label := range exemplar.Labels {
						if label.Key == testCase.targetExemplarAttribute {
							actualAttrValue = label.Value.GetStringValue()
							break
						}
					}
					require.Equal(t, expectedAttrValue, actualAttrValue)
				}
			}
		})
	}

	// invalid query
	res := doRequest(t, tempo.Endpoint(tempoPort), "{. a}", 100)
	require.Equal(t, 400, res.StatusCode)

	// query with empty results
	for _, query := range []string{
		// existing attribute, no traces
		"{status=error} | count_over_time()",
		// non-existing attribute, no traces
		`{span.randomattr = "doesnotexist"} | count_over_time()`,
	} {
		t.Run(query, func(t *testing.T) {
			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), query, 100, debugMode)
			require.NotNil(t, queryRangeRes)
			// it has time series but they are empty and has no exemplars
			require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)
			exemplarCount := 0
			for _, series := range queryRangeRes.GetSeries() {
				exemplarCount += len(series.GetExemplars())
			}
			require.Equal(t, 0, exemplarCount)
		})
	}
}

// TestQueryRangeSingleTrace checks count for a single trace
// Single trace creates a block with startTime == endTime
// which covers a few edge cases under the hood.
func TestQueryRangeSingleTrace(t *testing.T) {
	t.Parallel()

	s, err := e2e.NewScenario("tempo_e2e_single_trace")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRange, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	// Emit a single trace
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	// Query the trace by count. As we have only one trace, we should get one dot with value 1
	query := "{} | count_over_time()"
	queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), query, 100, debugMode)
	require.NotNil(t, queryRangeRes)
	require.Equal(t, len(queryRangeRes.GetSeries()), 1)

	series := queryRangeRes.GetSeries()[0]
	assert.Equal(t, len(series.GetExemplars()), 1)

	var sum float64
	for _, sample := range series.GetSamples() {
		sum += sample.Value
	}
	require.InDelta(t, sum, 1, 0.000001)
}

func TestQueryRangeMaxSeries(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRangeMaxSeries, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	query := "{} | rate() by (span:id)"
	url := fmt.Sprintf(
		"http://%s/api/metrics/query_range?q=%s&start=%d&end=%d&step=%s",
		tempo.Endpoint(3200),
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	fmt.Println(string(body))

	queryRangeRes := &tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, queryRangeRes)
	require.NoError(t, err)
	require.NotNil(t, queryRangeRes)

	// max series is 3 so we should get a partial response with 3 series
	require.Equal(t, tempopb.PartialStatus_PARTIAL, queryRangeRes.GetStatus())
	require.Equal(t, "Response exceeds maximum series limit of 3, a partial response is returned. Warning: the accuracy of each individual value is not guaranteed.", queryRangeRes.GetMessage())
	require.Equal(t, 3, len(queryRangeRes.GetSeries()))
}

func TestQueryRangeMaxSeriesDisabled(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRangeMaxSeriesDisabled, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	spanCount := 0
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))
			spanCount++
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	query := "{} | rate() by (span:id)"
	url := fmt.Sprintf(
		"http://%s/api/metrics/query_range?q=%s&start=%d&end=%d&step=%s",
		tempo.Endpoint(3200),
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	fmt.Println(string(body))

	queryRangeRes := &tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, queryRangeRes)
	require.NoError(t, err)
	require.NotNil(t, queryRangeRes)

	// max series is disabled so we should get a complete response with all series
	require.Equal(t, tempopb.PartialStatus_COMPLETE, queryRangeRes.GetStatus())
	require.Equal(t, spanCount, len(queryRangeRes.GetSeries()))
}

func TestQueryRangeMaxSeriesDisabledQuerier(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRangeMaxSeriesDisabledQuerier, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	spanCount := 0
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))
			spanCount++
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

	util.CallFlush(t, tempo)
	time.Sleep(blockFlushTimeout)
	util.CallFlush(t, tempo)

	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(5), "tempo_ingester_blocks_flushed_total"))

	query := "{} | rate() by (span:id)"
	url := fmt.Sprintf(
		"http://%s/api/metrics/query_range?q=%s&start=%d&end=%d&step=%s",
		tempo.Endpoint(3200),
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	fmt.Println(string(body))

	queryRangeRes := &tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, queryRangeRes)
	require.NoError(t, err)
	require.NotNil(t, queryRangeRes)

	// max series is disabled so we should get a complete response with all series
	require.Equal(t, tempopb.PartialStatus_COMPLETE, queryRangeRes.GetStatus())
	require.Equal(t, spanCount, len(queryRangeRes.GetSeries()))
}

func callQueryRange(t *testing.T, endpoint, query string, exemplars int, printBody bool) tempopb.QueryRangeResponse {
	res := doRequest(t, endpoint, query, exemplars)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if printBody {
		fmt.Println(string(body))
	}

	queryRangeRes := tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, &queryRangeRes)
	require.NoError(t, err)
	return queryRangeRes
}

func doRequest(t *testing.T, endpoint, query string, exemplars int) *http.Response {
	url := buildURL(endpoint, fmt.Sprintf("%s with(exemplars=true)", query), exemplars)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return res
}

func buildURL(endpoint, query string, exemplars int) string {
	return fmt.Sprintf(
		"http://%s/api/metrics/query_range?query=%s&start=%d&end=%d&step=%s&exemplars=%d",
		endpoint,
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
		exemplars,
	)
}
