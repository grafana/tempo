package e2e

import (
	"fmt"
	"os"
	"testing"

	"github.com/grafana/e2e"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/httpclient"
)

const (
	configAllInOneS3      = "deployments/config-all-in-one-s3.yaml"
	configAllInOneAzurite = "deployments/config-all-in-one-azurite.yaml"
	configAllInOneGCS     = "deployments/config-all-in-one-gcs.yaml"
)

func TestOverrides(t *testing.T) {
	testBackends := []struct {
		name           string
		skipVersioning bool
		configFile     string
	}{
		{
			name: "local",
			// local backend does not enforce versioning
			skipVersioning: true,
			configFile:     configAllInOneLocal,
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

			// copy config template to shared directory and expand template variables
			tmplConfig := map[string]any{"Prefix": ""}
			configFile, err := util.CopyTemplateToSharedDir(s, tc.configFile, "config.yaml", tmplConfig)
			require.NoError(t, err)

			// set up the backend
			cfg := app.Config{}
			buff, err := os.ReadFile(configFile)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			orgID := ""
			apiClient := httpclient.New("http://"+tempo.Endpoint(3200), orgID)

			// Create overrides
			initialLimits := &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					DisableCollection: boolPtr(true),
				},
			}

			if !tc.skipVersioning {
				fmt.Println("* Creating overrides with non-0 version")
				_, err = apiClient.SetOverrides(initialLimits, "123")
				assert.ErrorContains(t, err, "412") // precondition failed
			}

			fmt.Println("* Creating overrides")
			_, err = apiClient.SetOverrides(initialLimits, "0")
			assert.NoError(t, err)

			limits, version, err := apiClient.GetOverrides()
			assert.NoError(t, err)
			printLimits(limits, version)

			disableCollection, ok := limits.GetMetricsGenerator().GetDisableCollection()
			assert.True(t, ok)
			assert.True(t, disableCollection)

			// Update overrides - POST
			updatedLimits := &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					DisableCollection: nil,
					Processors:        map[string]struct{}{"span-metrics": {}},
				},
			}

			if !tc.skipVersioning {
				fmt.Println("* Update overrides with bogus version number")
				_, err = apiClient.SetOverrides(updatedLimits, "abc")
				assert.ErrorContains(t, err, "412") // precondition failed

				fmt.Println("* Update overrides with backend.VersionNew")
				_, err = apiClient.SetOverrides(updatedLimits, "0")
				assert.ErrorContains(t, err, "412") // precondition failed

				fmt.Println("* Update overrides with wrong version number")
				_, err = apiClient.SetOverrides(updatedLimits, "123")
				assert.ErrorContains(t, err, "412") // precondition failed
			}

			fmt.Println("* Update overrides")
			_, err = apiClient.SetOverrides(updatedLimits, version)
			assert.NoError(t, err)

			limits, version, err = apiClient.GetOverrides()

			assert.NoError(t, err)
			printLimits(limits, version)

			_, ok = limits.GetMetricsGenerator().GetDisableCollection()
			assert.False(t, ok) // is not set anymore
			processors, ok := limits.GetMetricsGenerator().GetProcessors()
			assert.True(t, ok)
			assert.ElementsMatch(t, keys(processors.GetMap()), []string{"span-metrics"})

			// Modify overrides - PATCH
			patch := &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					DisableCollection: boolPtr(true),
				},
			}

			fmt.Println("* Patch overrides")
			limits, version, err = apiClient.PatchOverrides(patch)
			assert.NoError(t, err)

			disableCollection, ok = limits.GetMetricsGenerator().GetDisableCollection()
			assert.True(t, ok)
			assert.True(t, disableCollection)
			processors, ok = limits.GetMetricsGenerator().GetProcessors()
			assert.True(t, ok)
			assert.ElementsMatch(t, keys(processors.GetMap()), []string{"span-metrics"})

			// Delete overrides
			if !tc.skipVersioning && tc.name != "gcs" {
				// Delete with preconditions is not supported by fake-gcs-server https://github.com/fsouza/fake-gcs-server/issues/1282
				fmt.Println("* Deleting overrides - don't respect version")
				err = apiClient.DeleteOverrides("123")
				assert.ErrorContains(t, err, "412") // precondition failed
			}

			fmt.Println("* Deleting overrides")
			err = apiClient.DeleteOverrides(version)
			assert.NoError(t, err)

			// Get overrides - 404
			fmt.Println("* Get overrides - 404")
			_, _, err = apiClient.GetOverrides()
			assert.ErrorIs(t, err, httpclient.ErrNotFound)

			// Recreate overrides - PATCH
			patch = &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					DisableCollection: boolPtr(true),
				},
			}

			fmt.Println("* Patch overrides - overrides don't exist yet")
			_, _, err = apiClient.PatchOverrides(patch)
			assert.NoError(t, err)
		})
	}
}

func printLimits(limits *client.Limits, version string) {
	var str string
	if limits != nil {
		bytes, err := jsoniter.Marshal(limits)
		if err == nil {
			str = string(bytes)
		}
	}
	fmt.Printf("* Overrides (version = %s): %+v\n", version, str)
}

func boolPtr(b bool) *bool {
	return &b
}

func keys(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
