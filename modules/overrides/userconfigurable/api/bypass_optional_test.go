package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
)

func TestServiceGraphsBypassAPIOptIn(t *testing.T) {
	const configured = `{"metrics_generator":{"processor":{"service_graphs":{"enable_service_graph_edge_storage":true,"bypass_services":["gateway"],"bypass_service_regex":["^proxy-.*$"]}}}}`
	for _, tc := range []struct {
		name, method, payload string
		enabled, existing     bool
		wantStatus            int
	}{
		{name: "disabled by default rejects POST", method: http.MethodPost, payload: configured, wantStatus: http.StatusBadRequest},
		{name: "disabled rejects PATCH", method: http.MethodPatch, payload: configured, wantStatus: http.StatusBadRequest},
		{name: "disabled rejects explicit false", method: http.MethodPatch, payload: `{"metrics_generator":{"processor":{"service_graphs":{"enable_service_graph_edge_storage":false}}}}`, wantStatus: http.StatusBadRequest},
		{name: "disabled rejects explicit empty list", method: http.MethodPatch, payload: `{"metrics_generator":{"processor":{"service_graphs":{"bypass_services":[]}}}}`, wantStatus: http.StatusBadRequest},
		{name: "enabled accepts POST", enabled: true, method: http.MethodPost, payload: configured, wantStatus: http.StatusOK},
		{name: "enabled accepts PATCH", enabled: true, method: http.MethodPatch, payload: configured, wantStatus: http.StatusOK},
		{name: "disabled permits unrelated update", method: http.MethodPatch, payload: `{"forwarders":["updated"]}`, wantStatus: http.StatusOK},
		{name: "disabled preserves existing settings on unrelated update", existing: true, method: http.MethodPatch, payload: `{"forwarders":["updated"]}`, wantStatus: http.StatusOK},
		{name: "disabled permits unchanged settings", existing: true, method: http.MethodPost, payload: configured, wantStatus: http.StatusOK},
		{name: "disabled rejects deleting existing bypass settings", existing: true, method: http.MethodDelete, wantStatus: http.StatusBadRequest},
		{name: "enabled permits deleting existing bypass settings", enabled: true, existing: true, method: http.MethodDelete, wantStatus: http.StatusOK},
		{name: "disabled rejects removing existing settings", existing: true, method: http.MethodPatch, payload: `{"metrics_generator":{"processor":{"service_graphs":{"bypass_services":null}}}}`, wantStatus: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			storage := client.Config{Backend: backend.Local, Local: &local.Config{Path: t.TempDir()}}
			manager, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
			require.NoError(t, err)
			api, err := New(&overrides.UserConfigurableOverridesAPIConfig{EnableServiceGraphBypass: tc.enabled}, &storage, manager, bypassHTTPValidator{})
			require.NoError(t, err)
			t.Cleanup(api.client.Shutdown)
			version := backend.VersionNew
			var initial *client.Limits
			if tc.existing {
				initial = &client.Limits{}
				require.NoError(t, json.Unmarshal([]byte(configured), initial))
				version, err = api.client.Set(context.Background(), "tenant", initial, version)
				require.NoError(t, err)
			}
			req := prepareRequest("tenant", tc.method, []byte(tc.payload))
			if tc.method == http.MethodPost || tc.method == http.MethodDelete {
				req.Header.Set(headerIfMatch, string(version))
			}
			response := httptest.NewRecorder()
			switch tc.method {
			case http.MethodPost:
				api.PostHandler(response, req)
			case http.MethodDelete:
				api.DeleteHandler(response, req)
			default:
				api.PatchHandler(response, req)
			}
			require.Equal(t, tc.wantStatus, response.Code, response.Body.String())
			stored, _, err := api.client.Get(context.Background(), "tenant")
			if tc.method == http.MethodDelete && tc.wantStatus == http.StatusOK {
				require.ErrorIs(t, err, backend.ErrDoesNotExist)
				return
			}
			if tc.wantStatus == http.StatusBadRequest {
				require.Contains(t, response.Body.String(), "enable_service_graph_bypass")
				if !tc.existing {
					require.ErrorIs(t, err, backend.ErrDoesNotExist)
					return
				}
				require.NoError(t, err)
				require.Equal(t, initial, stored, "rejected update must not modify persisted settings")
			} else {
				require.NoError(t, err)
				if tc.existing {
					require.Equal(t, initial.MetricsGenerator.Processor.ServiceGraphs, stored.MetricsGenerator.Processor.ServiceGraphs)
				}
			}
		})
	}
}
