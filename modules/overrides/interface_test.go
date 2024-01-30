package overrides

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
)

func TestRuntimeConfigOverridesManager_GetRuntimeOverridesFor_runtimeConfigOverridesManager(t *testing.T) {
	defaults := Overrides{
		MetricsGenerator: MetricsGeneratorOverrides{
			CollectionInterval: 60 * time.Second,
		},
	}
	tenantOverrides := &perTenantOverrides{
		TenantLimits: map[string]*Overrides{
			"foo": {
				MetricsGenerator: MetricsGeneratorOverrides{
					CollectionInterval: 15 * time.Second,
				},
			},
		},
	}

	runtimeOverridesManager, cleanup := createAndInitializeRuntimeOverridesManager(t, defaults, toYamlBytes(t, tenantOverrides))
	defer cleanup()

	overrides := runtimeOverridesManager.GetRuntimeOverridesFor("default")
	assert.NotNil(t, overrides)
	assert.Equal(t, 60*time.Second, overrides.MetricsGenerator.CollectionInterval)

	overrides = runtimeOverridesManager.GetRuntimeOverridesFor("foo")
	assert.NotNil(t, overrides)
	assert.Equal(t, 15*time.Second, overrides.MetricsGenerator.CollectionInterval)
}

func TestRuntimeConfigOverridesManager_GetRuntimeOverridesFor_userConfigurableOverridesManager(t *testing.T) {
	defaults := Overrides{
		MetricsGenerator: MetricsGeneratorOverrides{
			CollectionInterval: 60 * time.Second,
		},
	}
	tenantOverrides := &perTenantOverrides{
		TenantLimits: map[string]*Overrides{
			"foo": {
				MetricsGenerator: MetricsGeneratorOverrides{
					CollectionInterval: 15 * time.Second,
				},
			},
		},
	}

	_, userConfigurableOverridesManager, cleanup := localUserConfigOverrides(t, defaults, toYamlBytes(t, tenantOverrides))
	defer cleanup()

	// put in some user-configurable limits - these will be ignored anyway
	userConfigurableOverridesManager.setTenantLimit("foo", &client.Limits{
		Forwarders: nil,
		MetricsGenerator: client.LimitsMetricsGenerator{
			CollectionInterval: &client.Duration{Duration: 5 * time.Minute},
		},
	})

	overrides := userConfigurableOverridesManager.GetRuntimeOverridesFor("default")
	assert.NotNil(t, overrides)
	assert.Equal(t, 60*time.Second, userConfigurableOverridesManager.MetricsGeneratorCollectionInterval("default"))
	assert.Equal(t, 60*time.Second, overrides.MetricsGenerator.CollectionInterval)

	overrides = userConfigurableOverridesManager.GetRuntimeOverridesFor("foo")
	assert.NotNil(t, overrides)
	assert.Equal(t, 5*time.Minute, userConfigurableOverridesManager.MetricsGeneratorCollectionInterval("foo"))
	assert.Equal(t, 15*time.Second, overrides.MetricsGenerator.CollectionInterval)
}
