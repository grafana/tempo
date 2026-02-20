package api

import (
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestInvalidTenants(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay:  "config-multi-tenant.yaml",
		DeploymentMode: util.DeploymentModeSingleBinary,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		longTenant := strings.Repeat("a", tenant.MaxTenantIDLength+1)

		testCases := []struct {
			name        string
			orgID       string
			multitenant bool
			errContains string
		}{
			{
				name:        "invalid unsupported character",
				orgID:       "tenant#123",
				errContains: "unsupported character",
			},
			{
				name:        "invalid slash character",
				orgID:       "tenant/123",
				errContains: "unsupported character",
			},
			{
				name:        "invalid unsafe path segment",
				orgID:       "..",
				errContains: "tenant ID is '.' or '..'",
			},
			{
				name:        "invalid too long tenant id",
				orgID:       longTenant,
				errContains: "tenant ID is too long",
			},
			{
				name:        "invalid empty tenant",
				orgID:       "",
				errContains: "no org id",
			},
			{
				name:        "invalid leading separator",
				orgID:       "|tenantA",
				multitenant: true,
				errContains: "no org id",
			},
			{
				name:        "invalid trailing separator",
				orgID:       "tenantA|",
				multitenant: true,
				errContains: "no org id",
			},
			{
				name:        "invalid double separator",
				orgID:       "tenantA||tenantB",
				multitenant: true,
				errContains: "no org id",
			},
			{
				name:        "invalid multi tenant unsupported character",
				orgID:       "tenantA|tenant#B",
				multitenant: true,
				errContains: "tenant#B",
			},
			{
				name:        "invalid multi tenant unsafe segment",
				orgID:       "tenantA|..",
				multitenant: true,
				errContains: "tenant ID is '.' or '..'",
			},
			{
				name:        "invalid multi tenant too long segment",
				orgID:       "tenantA|" + longTenant,
				multitenant: true,
				errContains: "tenant ID is too long",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assertHTTP400 := func(t *testing.T, err error) {
					require.Error(t, err)
					// no org ID in single-tenant returns 401 from Auth middleware
					// others return 400
					require.Contains(t, err.Error(), "response: 40")
					require.Contains(t, err.Error(), tc.errContains)
				}

				apiClient := h.APIClientHTTP(tc.orgID)

				_, err := apiClient.QueryTrace("0047ac2c027451d89e3f3ba6d2a6b7b")
				assertHTTP400(t, err)

				_, err = apiClient.QueryTraceV2("0047ac2c027451d89e3f3ba6d2a6b7b")
				assertHTTP400(t, err)

				_, err = apiClient.SearchTraceQL("{}")
				assertHTTP400(t, err)

				_, err = apiClient.SearchTags()
				assertHTTP400(t, err)

				_, err = apiClient.SearchTagsV2()
				assertHTTP400(t, err)

				_, err = apiClient.SearchTagValues("span.name")
				assertHTTP400(t, err)

				_, err = apiClient.SearchTagValuesV2("span.name", "{}")
				assertHTTP400(t, err)

				if !tc.multitenant {
					_, err = apiClient.MetricsQueryRange("{} | count_over_time()", 0, 0, "", 0)
					assertHTTP400(t, err)
				}
			})

			if !tc.multitenant {
				t.Run(tc.name+"_write", func(t *testing.T) {
					writeTraceInfo := tempoUtil.NewTraceInfo(time.Now(), tc.orgID)
					writeErr := h.WriteTraceInfo(writeTraceInfo, tc.orgID)

					require.Error(t, writeErr)
					require.Contains(t, writeErr.Error(), "code = InvalidArgument")
					require.Contains(t, writeErr.Error(), tc.errContains)
				})
			}
		}
	})
}
