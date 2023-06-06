package overrides

import (
	"context"
	"os"
	"testing"

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
	assert.Equal(t, userConfigOverridesMgr.Forwarders("tenant-1"), []string{"other-forwarder"})
}

func TestNewUserConfigurableOverrides_readFromBackend(t *testing.T) {
	tempDir := t.TempDir()

	err := os.MkdirAll(tempDir+"/overrides/foo", 0777)
	require.NoError(t, err)
	err = os.WriteFile(tempDir+"/overrides/foo/overrides.json", []byte(`{"version":"v1","forwarders":["my-other-forwarder"]}`), 0644)
	require.NoError(t, err)

	cfg := UserConfigOverridesConfig{
		Enabled: true,
		Backend: "local",
		Local: &local.Config{
			Path: tempDir,
		},
	}
	tempoOverrides, _ := NewOverrides(Limits{
		Forwarders: []string{"my-forwarder"},
	})

	configurableOverrides, err := NewUserConfigOverrides(cfg, tempoOverrides)
	assert.NoError(t, err)

	// force a refresh
	err = configurableOverrides.refreshAllTenantLimits(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-other-forwarder"})
}

func TestConfigurableOverrides_setAndDelete(t *testing.T) {
	tempDir := t.TempDir()

	cfg := UserConfigOverridesConfig{
		Enabled: true,
		Backend: "local",
		Local: &local.Config{
			Path: tempDir,
		},
	}
	tempoOverrides, _ := NewOverrides(Limits{
		Forwarders: []string{"my-forwarder"},
	})

	configurableOverrides, err := NewUserConfigOverrides(cfg, tempoOverrides)
	assert.NoError(t, err)

	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-forwarder"})

	err = configurableOverrides.SetLimits(context.Background(), "foo", &UserConfigurableLimits{
		Version:    "",
		Forwarders: &[]string{"my-other-forwarder"},
	})
	assert.NoError(t, err)

	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-other-forwarder"})

	bytes, err := os.ReadFile(tempDir + "/overrides/foo/overrides.json")
	assert.NoError(t, err)
	assert.Equal(t, string(bytes), `{"version":"v1","forwarders":["my-other-forwarder"]}`)

	err = configurableOverrides.DeleteLimits(context.Background(), "foo")
	assert.NoError(t, err)

	// back to original value
	assert.Equal(t, configurableOverrides.Forwarders("foo"), []string{"my-forwarder"})

	// TODO we need to fix our delete function
	// assert.NoFileExists(t, tempDir+"/overrides/foo/overrides.json")
}

func TestNewUserConfigurableOverrides_backendDown(t *testing.T) {
	// TODO test we can fall back when backend is not responsive
}
