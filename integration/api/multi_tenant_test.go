package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestSingleTenantSearch(t *testing.T) {
	testSearch(t, "test")
}

func TestWildCardTenantSearch(t *testing.T) {
	testSearch(t, "*")
}

func TestTwoTenantsSearch(t *testing.T) {
	testSearch(t, "test|test2")
}

func TestThreeTenantsSearch(t *testing.T) {
	testSearch(t, "test|test2|test3")
}

func testSearch(t *testing.T, tenant string) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		ConfigOverlay: "config-multi-tenant.yaml",
	}, func(h *util.TempoHarness) {
		var info *tempoUtil.TraceInfo

		tenants := strings.Split(tenant, "|")
		tenantSize := len(tenants)

		var expectedSpans float64
		// write traces for all tenants
		for _, tenant := range tenants {
			info = tempoUtil.NewTraceInfo(time.Now(), tenant)
			require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

			trace, err := info.ConstructTraceFromEpoch()
			require.NoError(t, err)

			expectedSpans = expectedSpans + util.SpanCount(trace)
		}

		// assert that we have one trace for each tenant and correct number of spans received
		liveStoreZoneA := h.Services[util.ServiceLiveStoreZoneA]
		require.NoError(t, liveStoreZoneA.WaitSumMetrics(e2e.Equals(float64(tenantSize)), "tempo_live_store_traces_created_total"))

		distributor := h.Services[util.ServiceDistributor]
		require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(expectedSpans), "tempo_distributor_spans_received_total"))

		// check trace by id
		apiClient := httpclient.New("http://"+h.QueryFrontendHTTPEndpoint, tenant)
		util.SearchAndAssertTrace(t, apiClient, info)

		// search and traceql search
		util.SearchTraceQLAndAssertTrace(t, apiClient, info)

		// call search tags endpoints, ensure no errors and results are not empty
		tagsV2Resp, err := apiClient.SearchTagsV2()
		require.NoError(t, err)
		require.Equal(t, 4, len(tagsV2Resp.GetScopes())) // resource, span, event, link, instrumentation intrinsics
		for _, s := range tagsV2Resp.Scopes {
			require.NotEmpty(t, s.Tags)
		}

		tagsValuesV2Resp, err := apiClient.SearchTagValuesV2("span.vulture-0", "{}")
		require.NoError(t, err)
		require.NotEmpty(t, tagsValuesV2Resp.TagValues)

		// test streaming search over grpc
		grpcCtx := user.InjectOrgID(context.Background(), tenant)
		grpcCtx, err = user.InjectIntoGRPCRequest(grpcCtx)
		require.NoError(t, err)

		grpcClient, err := util.NewSearchGRPCClient(grpcCtx, h.QueryFrontendGRPCEndpoint)
		require.NoError(t, err)

		now := time.Now()
		util.SearchStreamAndAssertTrace(t, grpcCtx, grpcClient, info, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
	})
}
