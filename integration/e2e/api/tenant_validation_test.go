package api

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/integration/e2e/backend"
)

func TestInvalidTenants(t *testing.T) {
	t.Parallel()

	s, err := e2e.NewScenario(generateNetworkName())
	require.NoError(t, err)
	defer s.Close()

	cfg := app.Config{}
	buff, err := os.ReadFile(configMultiTenant)
	require.NoError(t, err)
	err = yaml.UnmarshalStrict(buff, &cfg)
	require.NoError(t, err)
	_, err = backend.New(s, cfg)
	require.NoError(t, err)

	require.NoError(t, util.CopyFileToSharedDir(s, configMultiTenant, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	prometheus := util.NewPrometheus()
	require.NoError(t, s.StartAndWaitReady(tempo, prometheus))

	baseURL := "http://" + tempo.Endpoint(tempoPort)

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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			client := httpclient.New(baseURL, tc.orgID)

			assertHTTP400 := func(t *testing.T, err error) {
				require.Error(t, err)
				// no org ID in single-tenant returns 401 from Auth middleware
				// others return 400
				require.Contains(t, err.Error(), "response: 40")
				require.Contains(t, err.Error(), tc.errContains)
			}

			_, err := client.QueryTrace("0047ac2c027451d89e3f3ba6d2a6b7b")
			assertHTTP400(t, err)

			_, err = client.QueryTraceV2("0047ac2c027451d89e3f3ba6d2a6b7b")
			assertHTTP400(t, err)

			_, err = client.SearchTraceQL("{}")
			assertHTTP400(t, err)

			_, err = client.SearchTags()
			assertHTTP400(t, err)

			_, err = client.SearchTagsV2()
			assertHTTP400(t, err)

			_, err = client.SearchTagValues("span.name")
			assertHTTP400(t, err)

			_, err = client.SearchTagValuesV2("span.name", "{}")
			assertHTTP400(t, err)

			if !tc.multitenant {
				_, err := client.MetricsSummary("{}", "name", 0, 0)
				assertHTTP400(t, err)

				_, err = client.MetricsQueryRange("{} | count_over_time()", 0, 0, "", 0)
				assertHTTP400(t, err)
			}
		})

		if !tc.multitenant {
			t.Run(tc.name+"_write", func(t *testing.T) {
				exporter, err := util.NewJaegerToOTLPExporterWithAuth(tempo.Endpoint(4317), tc.orgID, "", false)
				require.NoError(t, err)
				require.NotNil(t, exporter)

				writeTraceInfo := tempoUtil.NewTraceInfo(time.Now(), tc.orgID)
				writeErr := writeTraceInfo.EmitAllBatches(exporter)
				require.Error(t, writeErr)
				require.Contains(t, writeErr.Error(), "code = InvalidArgument")
				require.Contains(t, writeErr.Error(), tc.errContains)
			})
		}
	}
}
