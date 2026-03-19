package overrides

import (
	"context"
	"testing"

	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
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
			expectErr:  "trying to load deprecated legacy overrides config format but legacy overrides are disabled by default",
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
