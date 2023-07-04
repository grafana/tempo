package e2e

import (
	"fmt"
	"os"
	"testing"

	"github.com/grafana/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/modules/overrides"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

func TestOverrides(t *testing.T) {
	testBackends := []struct {
		name       string
		configFile string
	}{
		{
			name:       "local",
			configFile: configAllInOneLocal,
		},
		{
			name:       "s3",
			configFile: configAllInOneS3,
		},
		{
			name:       "azure",
			configFile: configAllInOneAzurite,
		},
		{
			name:       "gcs",
			configFile: configAllInOneGCS,
		},
	}
	for _, tc := range testBackends {
		t.Run(tc.name, func(t *testing.T) {
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			// set up the backend
			cfg := app.Config{}
			buff, err := os.ReadFile(tc.configFile)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			require.NoError(t, util.CopyFileToSharedDir(s, tc.configFile, "config.yaml"))
			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			orgID := ""
			apiClient := Client{tempoUtil.NewClient("http://"+tempo.Endpoint(3200), orgID)}

			// Get default overrides
			limits, err := apiClient.GetOverrides()
			printLimits(limits)
			require.NoError(t, err)

			// We fetched the overrides exactly once
			require.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_overrides_user_configurable_overrides_fetch_total"))

			require.NotNil(t, limits)
			assert.Empty(t, nil, limits.Forwarders)

			// Modify overrides
			fmt.Println("* Setting overrides.forwarders")
			err = apiClient.SetOverrides(&overrides.UserConfigurableLimits{
				Forwarders: &[]string{"my-forwarder"},
			})
			require.NoError(t, err)

			limits, err = apiClient.GetOverrides()
			printLimits(limits)
			require.NoError(t, err)

			require.NotNil(t, limits)
			require.NotNil(t, limits.Forwarders)
			assert.ElementsMatch(t, *limits.Forwarders, []string{"my-forwarder"})

			// We fetched the overrides 2 times manually, but we also expect at least one poll_interval to have happened
			require.NoError(t, tempo.WaitSumMetrics(e2e.Greater(2), "tempo_overrides_user_configurable_overrides_fetch_total"))

			// Clear overrides
			fmt.Println("* Deleting overrides")
			err = apiClient.DeleteOverrides()
			require.NoError(t, err)

			limits, err = apiClient.GetOverrides()
			printLimits(limits)
			require.NoError(t, err)

			require.NotNil(t, limits)
			assert.Empty(t, nil, limits.Forwarders)
		})
	}
}

func printLimits(limits *overrides.UserConfigurableLimits) {
	fmt.Printf("* Overrides: %+v\n", limits)
	if limits != nil && limits.Forwarders != nil {
		fmt.Printf("*   Fowarders: %+v\n", *limits.Forwarders)
	}
}
