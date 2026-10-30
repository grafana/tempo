package misc

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/integration/util"
	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestQueryTrackerRoutesDisabledByDefault(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeSingleBinary,
	}, func(h *util.TempoHarness) {
		client := &http.Client{Timeout: 5 * time.Second}
		baseURL := h.BaseURL()
		for _, tc := range []struct {
			name   string
			path   string
			status int
		}{
			{name: "tenant list", path: api.PathQueryTracker, status: http.StatusNotFound},
			{name: "tenant detail", path: api.PathQueryTracker + "/tenant-a", status: http.StatusNotFound},
			{name: "unknown status endpoint", path: "/status/not-an-endpoint", status: http.StatusNotFound},
			{name: "combined status", path: "/status", status: http.StatusOK},
			{name: "status version", path: "/status/version", status: http.StatusOK},
			{name: "status services", path: "/status/services", status: http.StatusOK},
			{name: "status endpoints", path: "/status/endpoints", status: http.StatusOK},
			{name: "status runtime config", path: "/status/runtime_config", status: http.StatusOK},
			{name: "status config", path: "/status/config", status: http.StatusOK},
		} {
			t.Run(tc.name, func(t *testing.T) {
				for _, accept := range []string{"text/html", "application/json"} {
					t.Run(accept, func(t *testing.T) {
						req, err := http.NewRequest(http.MethodGet, baseURL+tc.path, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", accept)
						resp, err := client.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()
						require.Equal(t, tc.status, resp.StatusCode)
						if tc.status == http.StatusNotFound {
							body, err := io.ReadAll(resp.Body)
							require.NoError(t, err)
							require.Equal(t, "404 page not found\n", string(body))
						}
					})
				}
			})
		}
	})
}
