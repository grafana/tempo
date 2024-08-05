package e2e

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
	"github.com/grafana/tempo/v2/integration/util"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

const configQueryRange = "config-query-range.yaml"

// Set debugMode to true to print the response body
var debugMode = false

func TestQueryRangeExemplars(t *testing.T) {
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
	} {
		t.Run(query, func(t *testing.T) {
			callQueryRange(t, tempo.Endpoint(3200), query, debugMode)
		})
	}
}

func callQueryRange(t *testing.T, endpoint, query string, printBody bool) {
	url := buildURL(endpoint, fmt.Sprintf("%s with(exemplars=true)", query))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if printBody {
		fmt.Println(string(body))
	}

	queryRangeRes := tempopb.QueryRangeResponse{}
	require.NoError(t, json.Unmarshal(body, &queryRangeRes))
	require.NotNil(t, queryRangeRes)
	require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)
	exemplarCount := 0
	for _, series := range queryRangeRes.GetSeries() {
		exemplarCount += len(series.GetExemplars())
	}
	require.GreaterOrEqual(t, exemplarCount, 1)
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
