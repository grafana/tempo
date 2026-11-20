package app

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	queryapi "github.com/grafana/tempo/v3/pkg/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	"github.com/grafana/tempo/v3/tempodb/encoding/common"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
)

func TestServiceGraphsBypassOverridesValidation(t *testing.T) {
	for _, tc := range []struct {
		name            string
		enabled         *bool
		exact, patterns []string
		wantError       string
	}{
		{name: "unset"},
		{name: "too many names", exact: strings.Fields(strings.Repeat("gateway ", 129)), wantError: "128"},
		{name: "too many combined rules", exact: strings.Fields(strings.Repeat("gateway ", 128)), patterns: []string{"proxy"}, wantError: "128"},
		{name: "too much text", exact: []string{strings.Repeat("x", 65537)}, wantError: "65536"},
		{name: "invalid utf8", exact: []string{string([]byte{255})}, wantError: "UTF-8"},
		{name: "empty name", exact: []string{""}, wantError: "nonempty"},
		{name: "inherits relaxation flag", patterns: []string{`^gateway-.*$`}},
		{name: "explicitly enabled", enabled: new(true), exact: []string{"gateway"}},
		{name: "disabled with empty lists", enabled: new(false), exact: []string{}, patterns: []string{}},
		{name: "disabled with exact list", enabled: new(false), exact: []string{"gateway"}},
		{name: "disabled with regex", enabled: new(false), patterns: []string{`^gateway$`}},
		{name: "invalid regex", enabled: new(true), patterns: []string{"["}, wantError: "bypass_service_regex[0]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := &overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{Processor: overrides.ProcessorOverrides{ServiceGraphs: overrides.ServiceGraphsOverrides{EnableServiceGraphEdgeStorage: tc.enabled, BypassServices: tc.exact, BypassServiceRegex: tc.patterns}}}}
			_, runtimeErr := newRuntimeConfigValidator(&Config{}).Validate(runtime)
			limits := &client.Limits{MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{EnableServiceGraphEdgeStorage: tc.enabled, BypassServices: &tc.exact, BypassServiceRegex: &tc.patterns}}}}
			apiErr := newOverridesValidator(&Config{}).Validate(limits)
			for _, err := range []error{runtimeErr, apiErr} {
				if tc.wantError != "" {
					require.ErrorContains(t, err, tc.wantError)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestServiceGraphEdgeStorageVersionValidation(t *testing.T) {
	for _, tc := range []struct {
		name, version string
		enabled       *bool
		wantError     bool
	}{
		{"current format", "vParquet5", new(true), false},
		{"vParquet4 supported", "vParquet4", new(true), false},
		{"unsupported format rejected", "vParquet3", new(true), true},
		{"disabled permits old format", "vParquet4", new(false), false},
		{"unset inherits", "vParquet4", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.StorageConfig.Trace.Block = &common.BlockConfig{Version: tc.version}
			runtime := &overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{Processor: overrides.ProcessorOverrides{ServiceGraphs: overrides.ServiceGraphsOverrides{EnableServiceGraphEdgeStorage: tc.enabled}}}}
			_, runtimeErr := newRuntimeConfigValidator(cfg).Validate(runtime)
			limits := &client.Limits{MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{EnableServiceGraphEdgeStorage: tc.enabled}}}}
			apiErr := newOverridesValidator(cfg).Validate(limits)
			for _, err := range []error{runtimeErr, apiErr} {
				if tc.wantError {
					require.ErrorContains(t, err, "requires vParquet4 or vParquet5")
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

// Generator-only defaults must reach the same override getters used by trace
// handlers, including tenants with unrelated overrides. Explicit empty lists win.
func TestServiceGraphPresentationGeneratorDefaults(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		names, patterns         []string
		wantNames, wantPatterns []string
	}{
		{name: "inherit generator defaults", wantNames: []string{"gateway"}, wantPatterns: []string{"^proxy$"}},
		{name: "override one default", names: []string{"sidecar"}, wantNames: []string{"sidecar"}, wantPatterns: []string{"^proxy$"}},
		{name: "clear both defaults", names: []string{}, patterns: []string{}, wantNames: []string{}, wantPatterns: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.Generator.Processor.ServiceGraphs.BypassServices = []string{"gateway"}
			cfg.Generator.Processor.ServiceGraphs.BypassServiceRegex = []string{"^proxy$"}
			sg := &cfg.Overrides.Defaults.MetricsGenerator.Processor.ServiceGraphs
			sg.BypassServices, sg.BypassServiceRegex = serviceGraphPresentationDefaults(cfg, tc.names, tc.patterns)
			path := filepath.Join(t.TempDir(), "overrides.yaml")
			require.NoError(t, os.WriteFile(path, []byte("overrides:\n  unrelated:\n    metrics_generator:\n      disable_collection: true\n  cleared:\n    metrics_generator:\n      processor:\n        service_graphs:\n          bypass_services: []\n          bypass_service_regex: []\n"), 0o600))
			cfg.Overrides.PerTenantOverrideConfig = path
			cfg.Overrides.PerTenantOverridePeriod = model.Duration(time.Hour)
			o, err := overrides.NewOverrides(cfg.Overrides, newRuntimeConfigValidator(cfg), prometheus.NewRegistry())
			require.NoError(t, err)
			require.NoError(t, services.StartAndAwaitRunning(context.Background(), o))
			t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), o)) })
			for _, tenant := range []string{"default", "unrelated", "cleared"} {
				rules, err := queryapi.ParseServiceGraphPresentationRequest(httptest.NewRequest("GET", "/?include_service_graph_presentation=true", nil), o.MetricsGeneratorProcessorServiceGraphsBypassServices(tenant), o.MetricsGeneratorProcessorServiceGraphsBypassServiceRegex(tenant))
				require.NoError(t, err)
				if tenant == "cleared" {
					require.Empty(t, rules.BypassServices)
					require.Empty(t, rules.BypassServiceRegex)
				} else {
					require.ElementsMatch(t, tc.wantNames, rules.BypassServices)
					require.ElementsMatch(t, tc.wantPatterns, rules.BypassServiceRegex)
				}
			}
			// Tenant-cleared regexes must stay cleared during saved-update validation,
			// even when the global default would exceed the combined rule limit.
			validator := newOverridesValidator(cfg)
			validator.runtimeOverrides = o
			names := strings.Fields(strings.Repeat("gateway ", 128))
			limits := &client.Limits{MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{BypassServices: &names}}}}
			require.NoError(t, validator.ValidateForTenant("cleared", limits))
			err = validator.ValidateForTenant("unrelated", limits)
			if len(tc.wantPatterns) > 0 {
				require.ErrorContains(t, err, "128")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Validation includes inherited lists. A request may clear one list to fit the
// limit, and a replacement must inherit runtime defaults rather than old API data.
func TestServiceGraphPresentationInheritedLimits(t *testing.T) {
	cfg := &Config{}
	cfg.Generator.Processor.ServiceGraphs.BypassServiceRegex = []string{"proxy"}
	names := strings.Fields(strings.Repeat("gateway ", 128))
	runtime := &overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{Processor: overrides.ProcessorOverrides{ServiceGraphs: overrides.ServiceGraphsOverrides{BypassServices: names}}}}
	_, err := newRuntimeConfigValidator(cfg).Validate(runtime)
	require.ErrorContains(t, err, "128")
	runtime.MetricsGenerator.Processor.ServiceGraphs.BypassServiceRegex = []string{}
	_, err = newRuntimeConfigValidator(cfg).Validate(runtime)
	require.NoError(t, err)
	v := newOverridesValidator(cfg)
	limits := &client.Limits{MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{BypassServices: &names}}}}
	require.ErrorContains(t, v.ValidateForTenant("tenant", limits), "128")
	empty := []string{}
	limits.MetricsGenerator.Processor.ServiceGraphs.BypassServiceRegex = &empty
	require.NoError(t, v.ValidateForTenant("tenant", limits))
	require.NoError(t, v.ValidateForTenant("tenant", &client.Limits{}))
}
