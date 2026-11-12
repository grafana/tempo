package app

import (
	"testing"

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
		{"old format rejected", "vParquet4", new(true), true},
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
					require.ErrorContains(t, err, "requires vParquet5")
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}
