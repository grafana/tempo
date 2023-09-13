package overrides

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userconfigurableoverrides "github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	tempo_api "github.com/grafana/tempo/pkg/api"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

const (
	tenant1 = "tenant-1"
	tenant2 = "tenant-2"
)

func TestUserConfigOverridesManager(t *testing.T) {
	defaultLimits := Overrides{
		Global: GlobalOverrides{
			MaxBytesPerTrace: 1024,
		},
		Forwarders: []string{"my-forwarder"},
	}
	_, mgr := localUserConfigOverrides(t, defaultLimits)

	// Verify default limits are returned
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant2))

	// Update limits for tenant-1
	userConfigurableLimits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	_, err := mgr.client.Set(context.Background(), tenant2, userConfigurableLimits, backend.VersionNew)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// Verify updated limits are returned
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-other-forwarder"}, mgr.Forwarders(tenant2))

	// Delete limits for tenant-1
	err = mgr.client.Delete(context.Background(), tenant2, backend.VersionNew)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// Verify default limits are returned again
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant2))
}

func TestUserConfigOverridesManager_allFields(t *testing.T) {
	defaultLimits := Overrides{}
	_, mgr := localUserConfigOverrides(t, defaultLimits)

	assert.Empty(t, mgr.Forwarders(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessors(tenant1))
	assert.Equal(t, false, mgr.MetricsGeneratorDisableCollection(tenant1))
	assert.Equal(t, 0*time.Second, mgr.MetricsGeneratorCollectionInterval(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorServiceGraphsDimensions(tenant1))
	assert.Empty(t, false, mgr.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorServiceGraphsPeerAttributes(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorSpanMetricsDimensions(tenant1))
	assert.Equal(t, false, mgr.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorSpanMetricsFilterPolicies(tenant1))

	// Inject user-configurable overrides
	mgr.tenantLimits[tenant1] = &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-forwarder"},
		MetricsGenerator: &userconfigurableoverrides.LimitsMetricsGenerator{
			Processors:         map[string]struct{}{"service-graphs": {}},
			DisableCollection:  boolPtr(true),
			CollectionInterval: &userconfigurableoverrides.Duration{Duration: 60 * time.Second},
			Processor: &userconfigurableoverrides.LimitsMetricsGeneratorProcessor{
				ServiceGraphs: &userconfigurableoverrides.LimitsMetricsGeneratorProcessorServiceGraphs{
					Dimensions:               &[]string{"sg-dimension"},
					EnableClientServerPrefix: boolPtr(true),
					PeerAttributes:           &[]string{"attribute"},
				},
				SpanMetrics: &userconfigurableoverrides.LimitsMetricsGeneratorProcessorSpanMetrics{
					Dimensions:       &[]string{"sm-dimension"},
					EnableTargetInfo: boolPtr(true),
					FilterPolicies: &[]filterconfig.FilterPolicy{
						{
							Include: &filterconfig.PolicyMatch{
								MatchType: filterconfig.Strict,
								Attributes: []filterconfig.MatchPolicyAttribute{
									{
										Key:   "span.kind",
										Value: "SPAN_KIND_SERVER",
									},
								},
							},
							Exclude: &filterconfig.PolicyMatch{
								MatchType: filterconfig.Strict,
								Attributes: []filterconfig.MatchPolicyAttribute{
									{
										Key:   "span.kind",
										Value: "SPAN_KIND_CONSUMER",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Verify we can get the updated overrides
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, map[string]struct{}{"service-graphs": {}}, mgr.MetricsGeneratorProcessors(tenant1))
	assert.Equal(t, true, mgr.MetricsGeneratorDisableCollection(tenant1))
	assert.Equal(t, []string{"sg-dimension"}, mgr.MetricsGeneratorProcessorServiceGraphsDimensions(tenant1))
	assert.Equal(t, 60*time.Second, mgr.MetricsGeneratorCollectionInterval(tenant1))
	assert.Equal(t, true, mgr.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(tenant1))
	assert.Equal(t, []string{"attribute"}, mgr.MetricsGeneratorProcessorServiceGraphsPeerAttributes(tenant1))
	assert.Equal(t, []string{"sm-dimension"}, mgr.MetricsGeneratorProcessorSpanMetricsDimensions(tenant1))
	assert.Equal(t, true, mgr.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(tenant1))

	filterPolicies := mgr.MetricsGeneratorProcessorSpanMetricsFilterPolicies(tenant1)
	assert.NotEmpty(t, filterPolicies)
	assert.Equal(t, filterconfig.Strict, filterPolicies[0].Include.MatchType)
	assert.Equal(t, filterconfig.Strict, filterPolicies[0].Exclude.MatchType)
	assert.Equal(t, "span.kind", filterPolicies[0].Include.Attributes[0].Key)
	assert.Equal(t, "span.kind", filterPolicies[0].Exclude.Attributes[0].Key)
	assert.Equal(t, "SPAN_KIND_SERVER", filterPolicies[0].Include.Attributes[0].Value)
	assert.Equal(t, "SPAN_KIND_CONSUMER", filterPolicies[0].Exclude.Attributes[0].Value)
}

func TestUserConfigOverridesManager_populateFromBackend(t *testing.T) {
	defaultLimits := Overrides{
		Forwarders: []string{"my-forwarder"},
	}
	tempDir, mgr := localUserConfigOverrides(t, defaultLimits)

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-forwarder"})

	// write directly to backend
	limits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	writeUserConfigurableOverridesToDisk(t, tempDir, tenant1, limits)

	// reload from backend
	err := mgr.reloadAllTenantLimits(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-other-forwarder"})
}

func TestUserConfigOverridesManager_deletedFromBackend(t *testing.T) {
	defaultLimits := Overrides{
		Forwarders: []string{"my-forwarder"},
	}
	tempDir, mgr := localUserConfigOverrides(t, defaultLimits)

	limits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	_, err := mgr.client.Set(context.Background(), tenant1, limits, backend.VersionNew)
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
	defaultLimits := Overrides{
		Forwarders: []string{"my-forwarder"},
	}
	_, mgr := localUserConfigOverrides(t, defaultLimits)

	limits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	_, err := mgr.client.Set(context.Background(), tenant1, limits, backend.VersionNew)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// replace reader by this uncooperative fella
	mgr.client = &badClient{}

	// reloading fails
	assert.Error(t, mgr.reloadAllTenantLimits(context.Background()))

	// but overrides should be cached
	assert.Equal(t, []string{"my-other-forwarder"}, mgr.Forwarders(tenant1))
}

func TestUserConfigOverridesManager_WriteStatusRuntimeConfig(t *testing.T) {
	bl := Overrides{Forwarders: []string{"my-forwarder"}}
	_, configurableOverrides := localUserConfigOverrides(t, bl)

	// set user config limits
	configurableOverrides.tenantLimits["test"] = &userconfigurableoverrides.Limits{
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

func localUserConfigOverrides(t *testing.T, baseLimits Overrides) (string, *userConfigurableOverridesManager) {
	path := t.TempDir()

	cfg := &UserConfigurableOverridesConfig{
		Enabled: true,
		Client: userconfigurableoverrides.Config{
			Backend: backend.Local,
			Local:   &local.Config{Path: path},
		},
	}

	baseOverrides, err := NewOverrides(Config{Defaults: baseLimits})
	assert.NoError(t, err)

	configurableOverrides, err := newUserConfigOverrides(cfg, baseOverrides)
	assert.NoError(t, err)

	return path, configurableOverrides
}

func writeUserConfigurableOverridesToDisk(t *testing.T, dir string, tenant string, limits *userconfigurableoverrides.Limits) {
	client, err := userconfigurableoverrides.New(&userconfigurableoverrides.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: dir},
	})
	assert.NoError(t, err)

	_, err = client.Set(context.Background(), tenant, limits, backend.VersionNew)
	assert.NoError(t, err)
}

func deleteUserConfigurableOverridesFromDisk(t *testing.T, dir string, tenant string) {
	client, err := userconfigurableoverrides.New(&userconfigurableoverrides.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: dir},
	})
	assert.NoError(t, err)

	err = client.Delete(context.Background(), tenant, backend.VersionNew)
	assert.NoError(t, err)
}

type badClient struct{}

var _ userconfigurableoverrides.Client = (*badClient)(nil)

func (b *badClient) List(context.Context) ([]string, error) {
	return nil, errors.New("no")
}

func (b *badClient) Get(context.Context, string) (*userconfigurableoverrides.Limits, backend.Version, error) {
	return nil, "", errors.New("no")
}

func (b *badClient) Set(context.Context, string, *userconfigurableoverrides.Limits, backend.Version) (backend.Version, error) {
	return "", errors.New("no")
}

func (b *badClient) Delete(context.Context, string, backend.Version) error {
	return errors.New("no")
}

func (b badClient) Shutdown() {
}

func boolPtr(b bool) *bool {
	return &b
}
