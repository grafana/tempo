package overrides

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/grafana/tempo/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend/local"
)

func TestNewUserConfigurableOverrides_priorityLogic(t *testing.T) {
	tempDir := t.TempDir()

	cfg := UserConfigOverridesConfig{
		Enabled: true,
		Backend: "local",
		Local: &local.Config{
			Path: tempDir,
		},
	}

	tempoOverrides, err := NewOverrides(Limits{
		MaxBytesPerTrace: 1024,
		Forwarders:       []string{"my-forwarder"},
	})
	assert.NoError(t, err)

	userConfigOverridesMgr, err := NewUserConfigOverrides(cfg, tempoOverrides)
	assert.NoError(t, err)

	// Manually update tenantLimits for tenant-1
	userConfigOverridesMgr.tenantLimits["tenant-1"] = &UserConfigurableLimits{
		Version:    "v1",
		Forwarders: &[]string{"other-forwarder"},
	}

	// Tenant without user-configurable overrides
	assert.Equal(t, userConfigOverridesMgr.MaxBytesPerTrace("tenant-2"), 1024)
	assert.Equal(t, userConfigOverridesMgr.Forwarders("tenant-2"), []string{"my-forwarder"})

	// Tenant with user-configurable overrides
	assert.Equal(t, userConfigOverridesMgr.MaxBytesPerTrace("tenant-1"), 1024)
	assert.Equal(t, userConfigOverridesMgr.Forwarders("tenant-1"), []string{"other-forwarder"})
}

func TestNewUserConfigurableOverrides_readFromBackend(t *testing.T) {
	tempDir := t.TempDir()

	limits := newUserConfigurableLimits()
	limits.Forwarders = &[]string{"my-other-forwarder"}

	writeUserConfigurableOverrides(t, tempDir, "foo", limits)

	bl := Limits{Forwarders: []string{"my-forwarder"}}
	configurableOverrides := localUserConfigOverrides(t, tempDir, bl)

	// force a refresh
	err := configurableOverrides.refreshAllTenantLimits(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-other-forwarder"})
}

func TestConfigurableOverrides_setAndDelete(t *testing.T) {
	tempDir := t.TempDir()
	bl := Limits{Forwarders: []string{"my-forwarder"}}
	configurableOverrides := localUserConfigOverrides(t, tempDir, bl)

	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-forwarder"})

	err := configurableOverrides.setLimits(context.Background(), "foo", &UserConfigurableLimits{
		Version:    "",
		Forwarders: &[]string{"my-other-forwarder"},
	})
	assert.NoError(t, err)

	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-other-forwarder"})

	assert.FileExists(t, tempDir+"/overrides/foo/overrides.json")

	err = configurableOverrides.DeleteLimits(context.Background(), "foo")
	assert.NoError(t, err)

	// back to original value
	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-forwarder"})

	assert.NoFileExists(t, tempDir+"/overrides/foo/overrides.json")
}

func TestNewUserConfigurableOverrides_backendDown(t *testing.T) {
	// TODO test we can fall back when backend is not responsive
}

func TestUserConfigOverridesManager_WriteStatusRuntimeConfig(t *testing.T) {
	tempDir := t.TempDir()
	bl := Limits{Forwarders: []string{"my-forwarder"}}
	configurableOverrides := localUserConfigOverrides(t, tempDir, bl)

	// set user config limits
	configurableOverrides.tenantLimits["test"] = &UserConfigurableLimits{
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
			require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(api.HeaderContentType))
			require.Equal(t, 200, res.StatusCode)
		})
	}
}

func writeUserConfigurableOverrides(t *testing.T, dir string, tenant string, limits *UserConfigurableLimits) {
	err := os.MkdirAll(dir+"/overrides/"+tenant, 0777)
	require.NoError(t, err)

	b, err := json.Marshal(limits)
	require.NoError(t, err)

	err = os.WriteFile(dir+"/overrides/"+tenant+"/overrides.json", b, 0644)
	require.NoError(t, err)
}

func localUserConfigOverrides(t *testing.T, tempDir string, baseLimits Limits) *userConfigOverridesManager {
	cfg := UserConfigOverridesConfig{
		Enabled: true,
		Backend: "local",
		Local:   &local.Config{Path: tempDir},
	}

	baseOverrides, err := NewOverrides(baseLimits)
	assert.NoError(t, err)

	configurableOverrides, err := NewUserConfigOverrides(cfg, baseOverrides)
	assert.NoError(t, err)

	return configurableOverrides
}
