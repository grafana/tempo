package api

import (
	"net/http"
	"testing"

	"github.com/grafana/tempo/integration/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatusAndRingAPIs performs a basic smoke test of all status and ring management endpoints.
func TestStatusAndRingAPIs(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeMicroservices,
		Components:     util.ComponentsRecentDataQuerying | util.ComponentsMetricsGeneration | util.ComponentsBackendWork | util.ComponentsBackendQuerying,
	}, func(h *util.TempoHarness) {
		testCases := []struct {
			name           string
			service        string
			endpoint       string
			expectedStatus int
		}{
			// All status endpoints on query-frontend
			{
				name:           "status - all info",
				service:        util.ServiceQueryFrontend,
				endpoint:       "/status",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "status/version",
				service:        util.ServiceQueryFrontend,
				endpoint:       "/status/version",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "status/services",
				service:        util.ServiceQueryFrontend,
				endpoint:       "/status/services",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "status/endpoints",
				service:        util.ServiceQueryFrontend,
				endpoint:       "/status/endpoints",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "status/config",
				service:        util.ServiceQueryFrontend,
				endpoint:       "/status/config",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "status/runtime_config",
				service:        util.ServiceQueryFrontend,
				endpoint:       "/status/runtime_config",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "status/buildinfo",
				service:        util.ServiceQueryFrontend,
				endpoint:       "/api/status/buildinfo",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "status/backendscheduler",
				service:        util.ServiceBackendScheduler,
				endpoint:       "/status/backendscheduler",
				expectedStatus: http.StatusOK,
			},

			// ring endpoints
			{
				name:           "livestore ring",
				service:        util.ServiceQuerier,
				endpoint:       "/live-store/ring",
				expectedStatus: http.StatusOK,
			},
			// { 200s but should 404!
			// 	name:           "generator ring",
			// 	service:        util.ServiceQuerier,
			// 	endpoint:       "/metrics-generator/ring",
			// 	expectedStatus: http.StatusNotFound,
			// },
			// { 404s due to disabled store in config-base.yaml
			// 	name:           "backend worker ring",
			// 	service:        util.ServiceBackendWorker,
			// 	endpoint:       "/backend-worker/ring",
			// 	expectedStatus: http.StatusOK,
			// },
			{
				name:           "partition ring - distributor",
				service:        util.ServiceDistributor,
				endpoint:       "/partition-ring",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "partition ring - livestore",
				service:        util.ServiceLiveStoreZoneA,
				endpoint:       "/partition-ring",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "partition ring - generator",
				service:        util.ServiceMetricsGenerator,
				endpoint:       "/partition-ring",
				expectedStatus: http.StatusOK,
			},

			// memberlist endpoint should be on all services in the memberlist cluster
			{
				name:           "memberlist on distributor",
				service:        util.ServiceDistributor,
				endpoint:       "/memberlist",
				expectedStatus: http.StatusOK,
			},
			{
				name:           "memberlist on livestore",
				service:        util.ServiceLiveStoreZoneA,
				endpoint:       "/memberlist",
				expectedStatus: http.StatusOK,
			},
		}

		// add a status endpoint for all services
		for _, k := range util.AllTempoServices {
			testCases = append(testCases, []struct {
				name           string
				service        string
				endpoint       string
				expectedStatus int
			}{
				{
					name:           "status - " + k,
					service:        k,
					endpoint:       "/status",
					expectedStatus: http.StatusOK,
				},
				{
					name:           "metrics - " + k,
					service:        k,
					endpoint:       "/metrics",
					expectedStatus: http.StatusOK,
				},
			}...)
		}

		client := &http.Client{}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				svc := h.Services[tc.service]

				url := "http://" + svc.Endpoint(3200) + tc.endpoint
				resp, err := client.Get(url)
				require.NoError(t, err, "failed to make request to %s", url)
				defer resp.Body.Close()

				assert.Equal(t, tc.expectedStatus, resp.StatusCode, "unexpected status code for %s", tc.endpoint)
			})
		}
	})
}
