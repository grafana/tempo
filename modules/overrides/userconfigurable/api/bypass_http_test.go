package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/v3/pkg/sharedconfig"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
)

// Exercise real HTTP requests, persistent override storage, and asynchronous
// tenant reloads using the same merge-patch flow a UI toggle would use.
func TestServiceGraphsBypassHTTPToggle(t *testing.T) {
	storage := client.Config{Backend: backend.Local, Local: &local.Config{Path: t.TempDir()}}
	cfg := overrides.Config{
		Defaults: overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{Processor: overrides.ProcessorOverrides{ServiceGraphs: overrides.ServiceGraphsOverrides{
			EnableServiceGraphEdgeStorage: boolPtr(true), BypassServices: []string{"runtime-gateway"}, BypassServiceRegex: []string{`^runtime-proxy$`},
		}}}},
		UserConfigurableOverridesConfig: overrides.UserConfigurableOverridesConfig{Enabled: true, API: overrides.UserConfigurableOverridesAPIConfig{EnableServiceGraphBypass: true}, PollInterval: 10 * time.Millisecond, Client: storage},
	}
	manager, err := overrides.NewOverrides(cfg, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), manager))
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), manager)) })
	api, err := New(&cfg.UserConfigurableOverridesConfig.API, &storage, manager, bypassHTTPValidator{})
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/overrides", api.GetHandler)
	mux.HandleFunc("POST /api/overrides", api.PostHandler)
	mux.HandleFunc("PATCH /api/overrides", api.PatchHandler)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(user.InjectOrgID(r.Context(), r.Header.Get("X-Scope-OrgID"))))
	}))
	t.Cleanup(server.Close)
	request := func(t *testing.T, method, path, body string, wantStatus int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("X-Scope-OrgID", "bypass-ui-test")
		req.Header.Set("Content-Type", "application/json")
		if method == http.MethodPost {
			req.Header.Set("If-Match", string(backend.VersionNew))
		}
		res, err := server.Client().Do(req)
		require.NoError(t, err)
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.Equal(t, wantStatus, res.StatusCode, string(data))
		t.Logf("%s %s -> %d", method, path, res.StatusCode)
		return data
	}
	request(t, http.MethodPost, "/api/overrides", `{"forwarders":["keep-me"],"metrics_generator":{"processor":{"span_metrics":{"dimensions":["http.method"]}}}}`, http.StatusOK)
	for _, tc := range []struct {
		name, patch     string
		exact, patterns []string
	}{
		{"on", `{"metrics_generator":{"processor":{"service_graphs":{"enable_service_graph_edge_storage":true,"bypass_services":["gateway"],"bypass_service_regex":["^proxy-.*$"]}}}}`, []string{"gateway"}, []string{`^proxy-.*$`}},
		{"off", `{"metrics_generator":{"processor":{"service_graphs":{"bypass_services":[],"bypass_service_regex":[]}}}}`, []string{}, []string{}},
		{"on again", `{"metrics_generator":{"processor":{"service_graphs":{"bypass_services":["gateway"],"bypass_service_regex":["^proxy-.*$"]}}}}`, []string{"gateway"}, []string{`^proxy-.*$`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request(t, http.MethodPatch, "/api/overrides", tc.patch, http.StatusOK)
			require.Eventually(t, func() bool {
				return slices.Equal(tc.exact, manager.MetricsGeneratorProcessorServiceGraphsBypassServices("bypass-ui-test")) &&
					slices.Equal(tc.patterns, manager.MetricsGeneratorProcessorServiceGraphsBypassServiceRegex("bypass-ui-test"))
			}, 5*time.Second, 10*time.Millisecond)
			var limits client.Limits
			require.NoError(t, json.Unmarshal(request(t, http.MethodGet, "/api/overrides?scope=merged", "", http.StatusOK), &limits))
			sg := limits.GetMetricsGenerator().GetProcessor().GetServiceGraphs()
			enabled, set := sg.GetEnableServiceGraphEdgeStorage()
			require.True(t, set)
			require.True(t, enabled, "toggling bypass must preserve span-kind relaxation")
			exact, set := sg.GetBypassServices()
			require.True(t, set)
			require.Equal(t, tc.exact, exact)
			patterns, set := sg.GetBypassServiceRegex()
			require.True(t, set)
			require.Equal(t, tc.patterns, patterns)
			// PATCH preserves unrelated settings; another tenant keeps runtime defaults.
			forwarders, set := limits.GetForwarders()
			require.True(t, set)
			require.Equal(t, []string{"keep-me"}, forwarders)
			dimensions, set := limits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetDimensions()
			require.True(t, set)
			require.Equal(t, []string{"http.method"}, dimensions)
			require.Equal(t, []string{"runtime-gateway"}, manager.MetricsGeneratorProcessorServiceGraphsBypassServices("another-tenant"))
			require.Equal(t, []string{`^runtime-proxy$`}, manager.MetricsGeneratorProcessorServiceGraphsBypassServiceRegex("another-tenant"))
		})
	}
	request(t, http.MethodPatch, "/api/overrides", `{"metrics_generator":{"processor":{"service_graphs":{"bypass_service_regex":["["]}}}}`, http.StatusBadRequest)
	var stored client.Limits
	require.NoError(t, json.Unmarshal(request(t, http.MethodGet, "/api/overrides", "", http.StatusOK), &stored))
	patterns, _ := stored.GetMetricsGenerator().GetProcessor().GetServiceGraphs().GetBypassServiceRegex()
	require.Equal(t, []string{`^proxy-.*$`}, patterns, "invalid updates must not change persisted settings")
}

type bypassHTTPValidator struct{}

func (bypassHTTPValidator) Validate(limits *client.Limits) error {
	sg := limits.GetMetricsGenerator().GetProcessor().GetServiceGraphs()
	patterns, _ := sg.GetBypassServiceRegex()
	names, _ := sg.GetBypassServices()
	return sharedconfig.ValidateServiceGraphPresentation(names, patterns)
}
