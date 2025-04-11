package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const configQueryRange = "config-query-range.yaml"

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
		"{} | histogram_over_time(duration)",
		"{} | count_over_time() by (status)",
		"{status != error} | count_over_time() by (status)",
	} {
		t.Run(query, func(t *testing.T) {
			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), query, debugMode)
			require.NotNil(t, queryRangeRes)
			require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)
			exemplarCount := 0
			for _, series := range queryRangeRes.GetSeries() {
				exemplarCount += len(series.GetExemplars())
			}
			require.GreaterOrEqual(t, exemplarCount, 1)
		})
	}

	// invalid query
	res := doRequest(t, tempo.Endpoint(tempoPort), "{. a}")
	require.Equal(t, 400, res.StatusCode)

	// query with empty results
	for _, query := range []string{
		// existing attribute, no traces
		"{status=error} | count_over_time()",
		// non-existing attribute, no traces
		`{span.randomattr = "doesnotexist"} | count_over_time()`,
	} {
		t.Run(query, func(t *testing.T) {
			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), query, debugMode)
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
	queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), query, debugMode)
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

func callQueryRange(t *testing.T, endpoint, query string, printBody bool) tempopb.QueryRangeResponse {
	res := doRequest(t, endpoint, query)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if printBody {
		fmt.Println(string(body))
	}

	queryRangeRes := tempopb.QueryRangeResponse{}
	require.NoError(t, json.Unmarshal(body, &queryRangeRes))
	return queryRangeRes
}

func doRequest(t *testing.T, endpoint, query string) *http.Response {
	url := buildURL(endpoint, fmt.Sprintf("%s with(exemplars=true)", query))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return res
}

func buildURL(endpoint, query string) string {
	return fmt.Sprintf(
		"http://%s/api/metrics/query_range?query=%s&start=%d&end=%d&step=%s",
		endpoint,
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
	)
}
