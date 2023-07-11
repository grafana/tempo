package overrides

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/grafana/tempo/modules/overrides/userconfigurableapi"
	tempo_api "github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/tempodb/backend/local"
)

const (
	tenant1 = "tenant-1"
	tenant2 = "tenant-2"
)

func TestUserConfigOverridesManager(t *testing.T) {
	defaultLimits := Limits{
		MaxBytesPerTrace: 1024,
		Forwarders:       []string{"my-forwarder"},
	}
	_, mgr := localUserConfigOverrides(t, defaultLimits)

	// Verify default limits are returned
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant2))

	// Update limits for tenant-1
	userConfigurableLimits := api.NewUserConfigurableLimits()
	userConfigurableLimits.Forwarders = &[]string{"my-other-forwarder"}
	err := mgr.client.Set(context.Background(), tenant2, userConfigurableLimits)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// Verify updated limits are returned
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-other-forwarder"}, mgr.Forwarders(tenant2))

	// Delete limits for tenant-1
	err = mgr.client.Delete(context.Background(), tenant2)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// Verify default limits are returned
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant2))
}

func TestUserConfigOverridesManager_populateFromBackend(t *testing.T) {
	defaultLimits := Limits{
		Forwarders: []string{"my-forwarder"},
	}
	tempDir, mgr := localUserConfigOverrides(t, defaultLimits)

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-forwarder"})

	// write directly to backend
	limits := api.NewUserConfigurableLimits()
	limits.Forwarders = &[]string{"my-other-forwarder"}
	writeUserConfigurableOverridesToDisk(t, tempDir, tenant1, limits)

	// reload from backend
	err := mgr.reloadAllTenantLimits(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-other-forwarder"})
}

func TestUserConfigOverridesManager_deletedFromBackend(t *testing.T) {
	defaultLimits := Limits{
		Forwarders: []string{"my-forwarder"},
	}
	tempDir, mgr := localUserConfigOverrides(t, defaultLimits)

	limits := api.NewUserConfigurableLimits()
	limits.Forwarders = &[]string{"my-other-forwarder"}
	err := mgr.client.Set(context.Background(), tenant1, limits)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-other-forwarder"})

	// delete overrides.json directly from the backend
	deleteUserConfigurableOverridesFromDisk(t, tempDir, tenant1)

	// reload from backend
	err = mgr.reloadAllTenantLimits(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, mgr.Forwarders("foo"), []string{"my-forwarder"})
}

func TestUserConfigOverridesManager_backendUnavailable(t *testing.T) {
	defaultLimits := Limits{
		Forwarders: []string{"my-forwarder"},
	}
	_, mgr := localUserConfigOverrides(t, defaultLimits)

	limits := api.NewUserConfigurableLimits()
	limits.Forwarders = &[]string{"my-other-forwarder"}
	err := mgr.client.Set(context.Background(), tenant1, limits)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// replace reader by this uncooperative fella
	mgr.client = badClient{}

	// reloading fails
	assert.Error(t, mgr.reloadAllTenantLimits(context.Background()))

	// but overrides should be cached
	assert.Equal(t, []string{"my-other-forwarder"}, mgr.Forwarders(tenant1))
}

func TestUserConfigOverridesManager_WriteStatusRuntimeConfig(t *testing.T) {
	bl := Limits{Forwarders: []string{"my-forwarder"}}
	_, configurableOverrides := localUserConfigOverrides(t, bl)

	// set user config limits
	configurableOverrides.tenantLimits["test"] = &api.UserConfigurableLimits{
		Version:    "v1",
		Forwarders: &[]string{"my-other-forwarder"},
	}

	tests := []struct {
		name      string
		overrides Service
		req       *http.Request
	}{
		{
			name:      "UserConfigOverrides with ucl",
			overrides: configurableOverrides,
			req:       httptest.NewRequest("GET", "/", nil),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := tc.overrides.WriteStatusRuntimeConfig(w, tc.req)
			require.NoError(t, err)

			data := w.Body.String()
			require.Contains(t, data, "user_configurable_overrides")
			require.Contains(t, data, "my-other-forwarder")

			res := w.Result()
			require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(tempo_api.HeaderContentType))
			require.Equal(t, 200, res.StatusCode)
		})
	}
}

func localUserConfigOverrides(t *testing.T, baseLimits Limits) (string, *userConfigurableOverridesManager) {
	path := t.TempDir()

	cfg := &UserConfigurableOverridesConfig{
		Enabled: true,
		ClientConfig: api.UserConfigurableOverridesClientConfig{
			Backend: "local",
			Local:   &local.Config{Path: path},
		},
	}

	baseOverrides, err := NewOverrides(baseLimits)
	assert.NoError(t, err)

	configurableOverrides, err := newUserConfigOverrides(cfg, baseOverrides)
	assert.NoError(t, err)

	return path, configurableOverrides
}

func writeUserConfigurableOverridesToDisk(t *testing.T, dir string, tenant string, limits *api.UserConfigurableLimits) {
	b, err := json.Marshal(limits)
	assert.NoError(t, err)

	err = os.MkdirAll(path.Join(dir, api.OverridesKeyPath, tenant), os.ModePerm)
	assert.NoError(t, err)

	err = os.WriteFile(path.Join(dir, api.OverridesKeyPath, tenant, api.OverridesFileName), b, 0644)
	assert.NoError(t, err)
}

func deleteUserConfigurableOverridesFromDisk(t *testing.T, dir string, tenant string) {
	err := os.Remove(path.Join(dir, api.OverridesKeyPath, tenant, api.OverridesFileName))
	assert.NoError(t, err)

	err = os.Remove(path.Join(dir, api.OverridesKeyPath, tenant))
	assert.NoError(t, err)
}

type badClient struct{}

func (b badClient) List(context.Context) ([]string, error) {
	return nil, errors.New("no")
}

func (b badClient) Get(context.Context, string) (*api.UserConfigurableLimits, error) {
	return nil, errors.New("no")
}

func (b badClient) Set(context.Context, string, *api.UserConfigurableLimits) error {
	return errors.New("no")
}

func (b badClient) Delete(context.Context, string) error {
	return errors.New("no")
}
