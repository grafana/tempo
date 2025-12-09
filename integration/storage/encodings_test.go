package storage

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/encoding"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

func TestEncodings(t *testing.T) {
	const repeatedSearchCount = 10

	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			util.WithTempoHarness(t, util.TestHarnessConfig{
				ConfigOverlay: "./config-encodings.yaml",
				ConfigTemplateData: map[string]any{
					"Version": enc.Version(),
				},
				DeploymentMode: util.DeploymentModeMicroservices,
				Components:     util.ComponentsBackendQuerying | util.ComponentRecentDataQuerying,
			}, func(h *util.TempoHarness) {
				info := tempoUtil.NewTraceInfo(time.Now(), "")
				require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

				_, err := info.ConstructTraceFromEpoch()
				require.NoError(t, err)

				liveStore := h.Services[util.ServiceLiveStoreZoneA]
				err = liveStore.WaitSumMetricsWithOptions(e2e.Greater(0),
					[]string{"tempo_live_store_traces_created_total"},
					e2e.WaitMissingMetrics,
				)
				require.NoError(t, err)

				util.QueryAndAssertTrace(t, h.HTTPClient, info)

				// wait for live store to complete the block
				err = liveStore.WaitSumMetricsWithOptions(e2e.Greater(0),
					[]string{"tempo_live_store_blocks_completed_total"},
					e2e.WaitMissingMetrics,
				)
				require.NoError(t, err)

				// v2 does not support querying and must be skipped
				if enc.Version() == v2.VersionString {
					return
				}

				// search for trace in backend multiple times with different attributes to make sure
				// we search with different scopes and with attributes from dedicated columns
				for range repeatedSearchCount {
					util.SearchTraceQLAndAssertTrace(t, h.HTTPClient, info)
				}

				// confirm a block was flushed, this means that the trace is in the backend
				err = h.Services[util.ServiceBlockBuilder].WaitSumMetricsWithOptions(e2e.Greater(0), // jpe - need to spin up block builders in microservices mode
					[]string{"tempo_block_builder_flushed_blocks"},
					e2e.WaitMissingMetrics,
				)
				require.NoError(t, err) // jpe - fails occassionally? context timeout? why?

				err = h.Services[util.ServiceQueryFrontend].WaitSumMetricsWithOptions(e2e.Greater(0),
					[]string{"tempodb_blocklist_length"},
					e2e.WaitMissingMetrics,
				)
				require.NoError(t, err)

				frontend := h.Services[util.ServiceQueryFrontend]
				require.NoError(t, h.RestartServiceWithConfigOverlay(t, frontend, "../util/config-query-backend.yaml"))

				grpcClient, err := util.NewSearchGRPCClient(context.Background(), h.QueryFrontendGRPCEndpoint) // jpe -add grpc client to harness?
				require.NoError(t, err)

				err = h.Services[util.ServiceQueryFrontend].WaitSumMetricsWithOptions(e2e.Greater(0),
					[]string{"tempodb_blocklist_length"},
					e2e.WaitMissingMetrics,
				)
				require.NoError(t, err)

				now := time.Now() // jpe - is there a way to do trace by id with the backend
				for range repeatedSearchCount {
					// search the backend. this works b/c we're passing a start/end AND setting query ingesters within min/max to 0
					util.SearchTraceQLAndAssertTraceWithRange(t, h.HTTPClient, info, now.Add(-time.Hour).Unix(), now.Unix())
					// find the trace with streaming. using the http server b/c that's what Grafana will do
					util.SearchStreamAndAssertTrace(t, context.Background(), grpcClient, info, now.Add(-time.Hour).Unix(), now.Unix())
				}
			})
		})
	}
}
