package api

import (
	"encoding/json"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
)

func TestLimitsFromOverridesServiceGraphsBypass(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings overrides.ServiceGraphsOverrides
	}{
		{name: "unset fields remain omitted"},
		{name: "configured fields round trip", settings: overrides.ServiceGraphsOverrides{EnableServiceGraphEdgeStorage: boolPtr(true), BypassServices: []string{"gateway"}, BypassServiceRegex: []string{`^proxy-.*$`}}},
		{name: "explicit false and empty lists round trip", settings: overrides.ServiceGraphsOverrides{EnableServiceGraphEdgeStorage: boolPtr(false), BypassServices: []string{}, BypassServiceRegex: []string{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{Processor: overrides.ProcessorOverrides{ServiceGraphs: tc.settings}}}}, nil, prometheus.NewRegistry())
			require.NoError(t, err)
			data, err := json.Marshal(limitsFromOverrides(o, "tenant"))
			require.NoError(t, err)
			var decoded client.Limits
			require.NoError(t, json.Unmarshal(data, &decoded))
			sg := decoded.GetMetricsGenerator().GetProcessor().GetServiceGraphs()
			enabled, set := sg.GetEnableServiceGraphEdgeStorage()
			require.Equal(t, tc.settings.EnableServiceGraphEdgeStorage != nil, set)
			if set {
				require.Equal(t, *tc.settings.EnableServiceGraphEdgeStorage, enabled)
			}
			exact, set := sg.GetBypassServices()
			require.Equal(t, tc.settings.BypassServices != nil, set)
			require.Equal(t, tc.settings.BypassServices, exact)
			patterns, set := sg.GetBypassServiceRegex()
			require.Equal(t, tc.settings.BypassServiceRegex != nil, set)
			require.Equal(t, tc.settings.BypassServiceRegex, patterns)
		})
	}
}
