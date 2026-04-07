package overrides

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func TestLegacyOverridesDisabledByDefault(t *testing.T) {
	tests := []struct {
		name                  string
		configType            ConfigType
		enableLegacyOverrides bool
		expectErr             string
	}{
		{
			name:       "rejects legacy when EnableLegacyOverrides is false",
			configType: ConfigTypeLegacy,
			expectErr:  "DEPRECATED: legacy overrides config format detected but legacy overrides are disabled by default",
		},
		{
			name:                  "accepts legacy when EnableLegacyOverrides is true",
			configType:            ConfigTypeLegacy,
			enableLegacyOverrides: true,
		},
		{
			name:       "accepts new config regardless of EnableLegacyOverrides",
			configType: ConfigTypeNew,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				ConfigType:            tc.configType,
				EnableLegacyOverrides: tc.enableLegacyOverrides,
			}

			o, err := NewOverrides(cfg, nil, prometheus.NewRegistry())
			if tc.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErr)
				return
			}

			require.NoError(t, err)
			require.NoError(t, services.StartAndAwaitRunning(context.TODO(), o))
			require.NoError(t, services.StopAndAwaitTerminated(context.TODO(), o))
		})
	}
}

func TestPerTenantLegacyOverridesDisabledByDefault(t *testing.T) {
	legacyOverrides := &perTenantLegacyOverrides{
		TenantLimits: map[string]*LegacyOverrides{
			"tenant1": {MaxBytesPerTrace: 1000},
		},
	}
	buff, err := yaml.Marshal(legacyOverrides)
	require.NoError(t, err)

	tests := []struct {
		name         string
		enableLegacy bool
		expectErr    string
	}{
		{
			name:      "rejects legacy per-tenant overrides when enableLegacy is false",
			expectErr: "DEPRECATED: legacy overrides config format is in use. per-tenant overrides file uses the legacy format but legacy overrides are disabled by default",
		},
		{
			name:         "accepts legacy per-tenant overrides when enableLegacy is true",
			enableLegacy: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loader := loadPerTenantOverrides(&mockValidator{}, ConfigTypeNew, false, tc.enableLegacy)
			result, err := loader(bytes.NewReader(buff))
			if tc.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}
