package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	v2 "github.com/grafana/tempo/tempodb/encoding/v2"

	"github.com/grafana/e2e"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/encoding"
)

const (
	configAllEncodings = "./config-encodings.tmpl.yaml"
)

func TestEncodings(t *testing.T) {
	const repeatedSearchCount = 10

	for _, enc := range encoding.AllEncodings() {
		t.Run(enc.Version(), func(t *testing.T) {
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			// copy config template to shared directory and expand template variables
			tmplConfig := map[string]any{"Version": enc.Version()}
			config, err := integration.CopyTemplateToSharedDir(s, configAllEncodings, "config.yaml", tmplConfig)
			require.NoError(t, err)

			// load final config
			var cfg app.Config
			buff, err := os.ReadFile(config)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)

			// set up the backend
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			tempo := integration.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			// Get port for the Jaeger gRPC receiver endpoint
			c, err := integration.NewJaegerGRPCClient(tempo.Endpoint(14250))
			require.NoError(t, err)
			require.NotNil(t, c)

			info := util.NewTraceInfo(time.Now(), "")
			require.NoError(t, info.EmitAllBatches(c))

			expected, err := info.ConstructTraceFromEpoch()
			require.NoError(t, err)

			// test metrics
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(spanCount(expected)), "tempo_distributor_spans_received_total"))

			// test echo
			assertEcho(t, "http://"+tempo.Endpoint(3200)+"/api/echo")

			apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "")

			// query an in-memory trace
			queryAndAssertTrace(t, apiClient, info)

			// wait trace_idle_time and ensure trace is created in ingester
			require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			// flush trace to backend
			callFlush(t, tempo)

			// v2 does not support querying and must be skipped
			if enc.Version() != v2.VersionString {
				// search for trace in backend multiple times with different attributes to make sure
				// we search with different scopes and with attributes from dedicated columns
				for i := 0; i < repeatedSearchCount; i++ {
					integration.SearchAndAssertTrace(t, apiClient, info)
					integration.SearchTraceQLAndAssertTrace(t, apiClient, info)
				}
			}

			// sleep
			time.Sleep(10 * time.Second)

			// force clear completed block
			callFlush(t, tempo)

			// test metrics
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
			require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))
			if enc.Version() != v2.VersionString {
				require.NoError(t, tempo.WaitSumMetrics(e2e.Greater(15), "tempo_query_frontend_queries_total"))
			}

			// query trace - should fetch from backend
			queryAndAssertTrace(t, apiClient, info)

			// create grpc client used for streaming
			grpcClient, err := integration.NewSearchGRPCClient(context.Background(), tempo.Endpoint(3200))
			require.NoError(t, err)

			if enc.Version() == v2.VersionString {
				return // v2 does not support querying and must be skipped
			}

			// search for trace in backend multiple times with different attributes to make sure
			// we search with different scopes and with attributes from dedicated columns
			now := time.Now()
			for i := 0; i < repeatedSearchCount; i++ {
				// search the backend. this works b/c we're passing a start/end AND setting query ingesters within min/max to 0
				integration.SearchAndAssertTraceBackend(t, apiClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())
				// find the trace with streaming. using the http server b/c that's what Grafana will do
				integration.SearchStreamAndAssertTrace(t, grpcClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())
			}
		})
	}
}
