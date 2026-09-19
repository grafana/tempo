package overrides

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverridesWithObjectStorage runs against all 3 object storage backends, each as its own
// parallel subtest - the harness itself runs Backends sequentially, so looping here is what
// actually gets the 3x wall-clock cost back.
func TestOverridesWithObjectStorage(t *testing.T) {
	for _, be := range util.BackendTestCases(util.BackendObjectStorageAll) {
		t.Run(be.Name, func(t *testing.T) {
			// RunIntegrationTests itself calls t.Parallel() on this subtest's t.
			testOverridesWithObjectStorage(t, be.Backend)
		})
	}
}

func testOverridesWithObjectStorage(t *testing.T, backend util.BackendsMask) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Backends:       backend,
		DeploymentMode: util.DeploymentModeSingleBinary,
		ConfigOverlay:  configOverrides,
	}, func(h *util.TempoHarness) {
		apiClient := h.APIClientHTTP("single-tenant")

		// Create overrides
		initialLimits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(true),
			},
		}

		fmt.Println("* Creating overrides with non-0 version")
		_, err := apiClient.SetOverrides(initialLimits, "123")
		assert.ErrorContains(t, err, "412") // precondition failed

		fmt.Println("* Creating overrides")
		_, err = apiClient.SetOverrides(initialLimits, "0")
		assert.NoError(t, err)

		limits, version, err := apiClient.GetOverrides()
		assert.NoError(t, err)
		EnableInstanceLabel, EnableInstanceLabelIsSet := limits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetEnableInstanceLabel()
		assert.True(t, EnableInstanceLabel)
		assert.False(t, EnableInstanceLabelIsSet)
		printLimits(limits, version)

		disableCollection, ok := limits.GetMetricsGenerator().GetDisableCollection()
		assert.True(t, ok)
		assert.True(t, disableCollection)

		// Update overrides - POST
		updatedLimits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: nil,
				Processors:        &listtomap.ListToMap{"span-metrics": {}},
			},
		}

		fmt.Println("* Update overrides with bogus version number")
		_, err = apiClient.SetOverrides(updatedLimits, "abc")
		assert.ErrorContains(t, err, "412") // precondition failed

		fmt.Println("* Update overrides with backend.VersionNew")
		_, err = apiClient.SetOverrides(updatedLimits, "0")
		assert.ErrorContains(t, err, "412") // precondition failed

		fmt.Println("* Update overrides with wrong version number")
		_, err = apiClient.SetOverrides(updatedLimits, "123")
		assert.ErrorContains(t, err, "412") // precondition failed

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
				GenerateNativeHistograms:       histogramModePtr(histograms.HistogramMethodNative),
				NativeHistogramMaxBucketNumber: uint32Ptr(200),
				DisableCollection:              boolPtr(true),
				Processor: client.LimitsMetricsGeneratorProcessor{
					SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
						EnableInstanceLabel: boolPtr(false),
					},
				},
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
		EnableInstanceLabel, EnableInstanceLabelIsSet = limits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetEnableInstanceLabel()
		assert.False(t, EnableInstanceLabel)
		assert.True(t, EnableInstanceLabelIsSet)

		generateNativeHistograms, ok := limits.GetMetricsGenerator().GetGenerateNativeHistograms()
		assert.True(t, ok)
		assert.Equal(t, histograms.HistogramMethodNative, generateNativeHistograms)
		nativeHistogramMaxBucketNumber, ok := limits.GetMetricsGenerator().GetNativeHistogramMaxBucketNumber()
		assert.True(t, ok)
		assert.Equal(t, uint32(200), nativeHistogramMaxBucketNumber)

		// Delete overrides
		cfg, err := h.GetConfig()
		require.NoError(t, err)
		if cfg.StorageConfig.Trace.Backend != "gcs" {
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

func printLimits(limits *client.Limits, version string) {
	var str string
	if limits != nil {
		bytes, err := json.Marshal(limits)
		if err == nil {
			str = string(bytes)
		}
	}
	fmt.Printf("* Overrides (version = %s): %+v\n", version, str)
}
