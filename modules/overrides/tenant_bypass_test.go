package overrides

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

func TestUserConfigOverridesServiceGraphsBypass(t *testing.T) {
	defaults := Overrides{MetricsGenerator: MetricsGeneratorOverrides{Processor: ProcessorOverrides{ServiceGraphs: ServiceGraphsOverrides{
		EnableServiceGraphEdgeStorage: boolPtr(true), BypassServices: []string{"global-gateway"}, BypassServiceRegex: []string{`^global-proxy$`},
	}}}}
	_, mgr, cleanup := localUserConfigOverrides(t, defaults, nil)
	t.Cleanup(cleanup)
	version := backend.VersionNew
	for _, tc := range []struct {
		name            string
		settings        client.LimitsMetricsGeneratorProcessorServiceGraphs
		enabled         bool
		exact, patterns []string
	}{
		{name: "tenant values win", settings: client.LimitsMetricsGeneratorProcessorServiceGraphs{EnableServiceGraphEdgeStorage: boolPtr(true), BypassServices: &[]string{"gateway"}, BypassServiceRegex: &[]string{`^proxy-.*$`}}, enabled: true, exact: []string{"gateway"}, patterns: []string{`^proxy-.*$`}},
		{name: "empty lists clear inherited values", settings: client.LimitsMetricsGeneratorProcessorServiceGraphs{BypassServices: &[]string{}, BypassServiceRegex: &[]string{}}, enabled: true, exact: []string{}, patterns: []string{}},
		{name: "explicit false disables relaxation", settings: client.LimitsMetricsGeneratorProcessorServiceGraphs{EnableServiceGraphEdgeStorage: boolPtr(false), BypassServices: &[]string{}, BypassServiceRegex: &[]string{}}, exact: []string{}, patterns: []string{}},
		{name: "unset fields restore defaults", enabled: true, exact: []string{"global-gateway"}, patterns: []string{`^global-proxy$`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits := &client.Limits{MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{ServiceGraphs: tc.settings}}}
			var err error
			version, err = mgr.client.Set(context.Background(), tenant1, limits, version)
			require.NoError(t, err)
			require.NoError(t, mgr.reloadAllTenantLimits(context.Background()))
			enabled, isSet := mgr.MetricsGeneratorProcessorServiceGraphsEnableServiceGraphEdgeStorage(tenant1)
			require.True(t, isSet)
			require.Equal(t, tc.enabled, enabled)
			require.Equal(t, tc.exact, mgr.MetricsGeneratorProcessorServiceGraphsBypassServices(tenant1))
			require.Equal(t, tc.patterns, mgr.MetricsGeneratorProcessorServiceGraphsBypassServiceRegex(tenant1))
			// Reloading one tenant must never alter another tenant's matcher settings.
			enabled, isSet = mgr.MetricsGeneratorProcessorServiceGraphsEnableServiceGraphEdgeStorage(tenant2)
			require.True(t, isSet)
			require.True(t, enabled)
			require.Equal(t, []string{"global-gateway"}, mgr.MetricsGeneratorProcessorServiceGraphsBypassServices(tenant2))
			require.Equal(t, []string{`^global-proxy$`}, mgr.MetricsGeneratorProcessorServiceGraphsBypassServiceRegex(tenant2))
		})
	}
	require.NoError(t, mgr.client.Delete(context.Background(), tenant1, version))
	require.NoError(t, mgr.reloadAllTenantLimits(context.Background()))
	require.Equal(t, []string{"global-gateway"}, mgr.MetricsGeneratorProcessorServiceGraphsBypassServices(tenant1))
}

func TestLegacyServiceGraphsBypassConversion(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		legacy := LegacyOverrides{
			MetricsGeneratorProcessorServiceGraphsEnableServiceGraphEdgeStorage: &enabled,
			MetricsGeneratorProcessorServiceGraphsBypassServices:                []string{"gateway"},
			MetricsGeneratorProcessorServiceGraphsBypassServiceRegex:            []string{`^proxy-.*$`},
		}
		converted := legacy.toNewLimits().MetricsGenerator.Processor.ServiceGraphs
		require.NotNil(t, converted.EnableServiceGraphEdgeStorage)
		require.Equal(t, enabled, *converted.EnableServiceGraphEdgeStorage)
		require.Equal(t, []string{"gateway"}, converted.BypassServices)
		require.Equal(t, []string{`^proxy-.*$`}, converted.BypassServiceRegex)
	}
}
