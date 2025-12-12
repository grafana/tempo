package api

import (
	"strings"
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
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
		h.WaitTracesWritable(t)

		var info *tempoUtil.TraceInfo

		tenants := strings.Split(tenant, "|")
		tenantSize := len(tenants)

		// write traces for all tenants
		for _, tenant := range tenants {
			info = tempoUtil.NewTraceInfo(time.Now(), tenant)
			require.NoError(t, h.WriteTraceInfo(info, tenant))
		}

		// assert that we have one trace for each tenant and correct number of spans received
		h.WaitTracesQueryable(t, tenantSize)

		// check trace by id
		apiClient := h.APIClientHTTP(tenant)
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

		grpcClient, ctx, err := h.APIClientGRPC(tenant)
		require.NoError(t, err)

		now := time.Now()
		util.SearchStreamAndAssertTrace(t, ctx, grpcClient, info, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
	})
}
