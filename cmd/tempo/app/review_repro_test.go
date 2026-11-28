package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/modules/overrides"
	overridesapi "github.com/grafana/tempo/v3/modules/overrides/userconfigurable/api"
	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
	queryapi "github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

// A valid runtime reload must not make a previously accepted saved blacklist
// unusable for a request that merely asks for its tenant's current defaults.
func TestReviewRuntimeReloadKeepsSavedPresentationQueryable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.yaml")
	require.NoError(t, os.WriteFile(path, []byte("overrides: {}\n"), 0o600))
	cfg := &Config{}
	cfg.Overrides = overrides.Config{PerTenantOverrideConfig: path, PerTenantOverridePeriod: model.Duration(10 * time.Millisecond), UserConfigurableOverridesConfig: overrides.UserConfigurableOverridesConfig{
		Enabled: true, PollInterval: 10 * time.Millisecond,
		Client: client.Config{Backend: backend.Local, Local: &local.Config{Path: t.TempDir()}},
		API:    overrides.UserConfigurableOverridesAPIConfig{EnableServiceGraphBypass: true},
	}}
	manager, err := overrides.NewOverrides(cfg.Overrides, newRuntimeConfigValidator(cfg), prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), manager))
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), manager)) })
	validator := newOverridesValidator(cfg)
	validator.runtimeOverrides = manager
	apiCfg := cfg.Overrides.UserConfigurableOverridesConfig
	api, err := overridesapi.New(&apiCfg.API, &apiCfg.Client, manager, validator)
	require.NoError(t, err)
	names := strings.Fields(strings.Repeat("gateway ", 128))
	body, err := json.Marshal(map[string]interface{}{"metrics_generator": map[string]interface{}{"processor": map[string]interface{}{"service_graphs": map[string]interface{}{"bypass_services": names}}}})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPatch, "/api/overrides", strings.NewReader(string(body)))
	response := httptest.NewRecorder()
	api.PatchHandler(response, request.WithContext(user.InjectOrgID(context.Background(), "tenant")))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Eventually(t, func() bool { return len(manager.MetricsGeneratorProcessorServiceGraphsBypassServices("tenant")) == 128 }, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte("overrides:\n  tenant:\n    metrics_generator:\n      processor:\n        service_graphs:\n          bypass_service_regex: ['^proxy$']\n"), 0o600))
	require.Eventually(t, func() bool {
		return len(manager.GetRuntimeOverridesFor("tenant").MetricsGenerator.Processor.ServiceGraphs.BypassServiceRegex) == 1
	}, 5*time.Second, 10*time.Millisecond)
	names, patterns := manager.MetricsGeneratorProcessorServiceGraphsPresentation("tenant")
	_, err = queryapi.ParseServiceGraphPresentationRequest(httptest.NewRequest(http.MethodGet, "/?include_service_graph_presentation=true", nil), names, patterns)
	require.NoError(t, err, "a previously accepted blacklist must remain queryable after runtime reload")
	require.Len(t, names, 128)
	require.Empty(t, patterns, "an invalid inherited combination must retain the last valid pair")
	// A later valid API update must recover automatically, without a restart.
	body, err = json.Marshal(map[string]interface{}{"metrics_generator": map[string]interface{}{"processor": map[string]interface{}{"service_graphs": map[string]interface{}{"bypass_services": names[:127]}}}})
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodPatch, "/api/overrides", strings.NewReader(string(body)))
	response = httptest.NewRecorder()
	api.PatchHandler(response, request.WithContext(user.InjectOrgID(context.Background(), "tenant")))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Eventually(t, func() bool {
		names, patterns = manager.MetricsGeneratorProcessorServiceGraphsPresentation("tenant")
		return len(names) == 127 && len(patterns) == 1
	}, 5*time.Second, 10*time.Millisecond)
	_, err = queryapi.ParseServiceGraphPresentationRequest(httptest.NewRequest(http.MethodGet, "/?include_service_graph_presentation=true", nil), names, patterns)
	require.NoError(t, err)
}
