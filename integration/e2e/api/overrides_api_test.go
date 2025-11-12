package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/httpclient"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	configAllInOneS3          = "./../deployments/config-all-in-one-s3.yaml"
	configAllInOneAzurite     = "./../deployments/config-all-in-one-azurite.yaml"
	configAllInOneGCS         = "./../deployments/config-all-in-one-gcs.yaml"
	configAllInOneS3Overrides = "./config-all-in-one-s3-overrides.yaml"
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
			s, tempo := setupTempo(t, tc.configFile)
			defer s.Close()

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
				_, err := apiClient.SetOverrides(initialLimits, "123")
				assert.ErrorContains(t, err, "412") // precondition failed
			}

			fmt.Println("* Creating overrides")
			_, err := apiClient.SetOverrides(initialLimits, "0")
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

func TestOverridesAPI_GET(t *testing.T) {
	s, tempo := setupTempo(t, configAllInOneS3Overrides)
	defer s.Close()

	t.Run("returns 404 when config not found", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-get-1")

		limits, etag, err := apiClient.GetOverrides()
		require.Nil(t, limits) // no limits because it doesn't exist for this tenant
		require.Empty(t, etag) // etag will be ""
		require.ErrorIs(t, err, httpclient.ErrNotFound)
	})

	t.Run("returns config with etag", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-get-2")

		// create initial config with POST
		initialLimits := &client.Limits{
			CostAttribution: client.CostAttribution{
				Dimensions: &map[string]string{"host.name": "host_name"},
			},
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(false),
			},
		}
		setEtag, err := apiClient.SetOverrides(initialLimits, "0")
		require.NoError(t, err)
		require.NotEmpty(t, setEtag)

		// get should return the config and etag
		returnedLimits, etag, err := apiClient.GetOverrides()
		require.NoError(t, err)
		require.Equal(t, setEtag, etag)
		require.Equal(t, initialLimits, returnedLimits)
	})
}

func TestOverridesAPI_POST(t *testing.T) {
	s, tempo := setupTempo(t, configAllInOneS3Overrides)
	defer s.Close()

	t.Run("API returns 428 without if-match header", func(t *testing.T) {
		baseURL := "http://" + tempo.Endpoint(3200)

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/overrides", baseURL), strings.NewReader(`{}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Scope-OrgID", "tenant-post-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusPreconditionRequired, resp.StatusCode)
		err = resp.Body.Close()
		require.NoError(t, err)
	})

	t.Run("creates config with If-Match 0 on new tenant", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-post-2")

		limits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(true),
			},
			CostAttribution: client.CostAttribution{
				Dimensions: &map[string]string{"host.name": "host_name"},
			},
		}
		setEtag, err := apiClient.SetOverrides(limits, "0")
		require.NoError(t, err)
		require.NotEmpty(t, setEtag)

		// verify we can read it back
		returnedLimits, etag, err := apiClient.GetOverrides()
		require.NoError(t, err)
		require.Equal(t, setEtag, etag)
		require.Equal(t, limits, returnedLimits)
		disableCollection, ok := returnedLimits.GetMetricsGenerator().GetDisableCollection()
		require.True(t, ok)
		require.True(t, disableCollection)
	})

	t.Run("tenant with existing config returns 412 with If-Match 0", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-post-3")

		// create initial config for tenant so we can try again
		limits := &client.Limits{
			CostAttribution: client.CostAttribution{
				Dimensions: &map[string]string{"service.name": "service_name"},
			},
		}
		etag, err := apiClient.SetOverrides(limits, "0")
		require.NotEmpty(t, etag)
		require.NoError(t, err)

		// try to create config again with If-Match 0
		etag2, err2 := apiClient.SetOverrides(limits, "0")
		require.Empty(t, etag2)
		require.ErrorContains(t, err2, "failed with response: 412 body: version does not match")
	})

	t.Run("incorrect If-Match returns 412", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-post-4")

		// Create initial config
		limits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(false),
			},
		}
		etag, err := apiClient.SetOverrides(limits, "0")
		require.NotEmpty(t, etag)
		require.NoError(t, err)

		// Try to update with wrong version
		etag2, err2 := apiClient.SetOverrides(limits, "made-up-etag-value")
		require.Empty(t, etag2)
		require.ErrorContains(t, err2, "failed with response: 412 body: version does not match")
	})

	t.Run("with invalid json returns 400", func(t *testing.T) {
		baseURL := "http://" + tempo.Endpoint(3200)

		// invalid config
		badConfig := strings.NewReader(`{"metrics_generator": {"processor": {"service_graphs": {"histogram_buckets": [0.1, "invalid"]}}}}`)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/overrides", baseURL), badConfig)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Scope-OrgID", "tenant-post-5")
		req.Header.Set("If-Match", "0")

		resp, _ := http.DefaultClient.Do(req)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("updates config with correct version", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-post-6")

		// create initial config
		initialLimits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(true),
			},
		}
		setEtag, err := apiClient.SetOverrides(initialLimits, "0")
		require.NoError(t, err)

		// update with correct version
		updatedLimits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processors: map[string]struct{}{"span-metrics": {}},
			},
		}
		newEtag, err := apiClient.SetOverrides(updatedLimits, setEtag)
		require.NoError(t, err)
		require.NotEqual(t, setEtag, newEtag)

		// verify the update
		returnedLimits, _, err := apiClient.GetOverrides()
		require.NoError(t, err)

		// DisableCollection should not be set anymore because it was wiped out by the update
		_, ok := returnedLimits.GetMetricsGenerator().GetDisableCollection()
		require.False(t, ok)

		// Processors should be set because we added then in updated
		processors, ok := returnedLimits.GetMetricsGenerator().GetProcessors()
		require.True(t, ok)
		require.ElementsMatch(t, keys(processors.GetMap()), []string{"span-metrics"})
	})
}

func TestOverridesAPI_PATCH(t *testing.T) {
	s, tempo := setupTempo(t, configAllInOneS3Overrides)
	defer s.Close()

	t.Run("with no existing config creates new config", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-patch-1")

		patch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(true),
			},
		}
		returnedLimits, etag, err := apiClient.PatchOverrides(patch)
		require.NoError(t, err)
		require.NotEmpty(t, etag)

		// verify returned limits match patch
		disableCollection, ok := returnedLimits.GetMetricsGenerator().GetDisableCollection()
		require.True(t, ok)
		require.True(t, disableCollection)

		// verify config was created via GET
		getLimits, getEtag, err := apiClient.GetOverrides()
		require.NoError(t, err)
		require.Equal(t, etag, getEtag)
		require.Equal(t, returnedLimits, getLimits)
	})

	t.Run("preserves existing config sections", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-patch-2")

		// create initial config with processors
		initialLimits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(true),
				Processors:        map[string]struct{}{"span-metrics": {}},
			},
		}
		_, _, err := apiClient.PatchOverrides(initialLimits)
		require.NoError(t, err)

		// PATCH with additional processor config
		patch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processor: client.LimitsMetricsGeneratorProcessor{
					SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
						EnableInstanceLabel: boolPtr(false),
					},
				},
			},
		}
		returnedLimits, _, err := apiClient.PatchOverrides(patch)
		require.NoError(t, err)

		// verify original fields are preserved and not wiped out
		disableCollection, ok := returnedLimits.GetMetricsGenerator().GetDisableCollection()
		require.True(t, ok)
		require.True(t, disableCollection)

		processors, ok := returnedLimits.GetMetricsGenerator().GetProcessors()
		require.True(t, ok)
		require.ElementsMatch(t, keys(processors.GetMap()), []string{"span-metrics"})

		// verify new field was added via PATCH
		enableInstanceLabel, ok := returnedLimits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetEnableInstanceLabel()
		require.True(t, ok)
		require.False(t, enableInstanceLabel)
	})

	t.Run("merges nested processor configs", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-patch-3")

		// create initial config with service graphs
		initialPatch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processor: client.LimitsMetricsGeneratorProcessor{
					ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
						HistogramBuckets: &[]float64{0.1, 0.2},
					},
				},
			},
		}
		_, _, err := apiClient.PatchOverrides(initialPatch)
		require.NoError(t, err)

		// PATCH with span metrics config
		spanMetricsPatch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processor: client.LimitsMetricsGeneratorProcessor{
					SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
						HistogramBuckets: &[]float64{0.3, 0.4},
					},
				},
			},
		}
		returnedLimits, _, err := apiClient.PatchOverrides(spanMetricsPatch)
		require.NoError(t, err)

		// verify both processor configs exist in returned limits
		sgBuckets, ok := returnedLimits.GetMetricsGenerator().GetProcessor().GetServiceGraphs().GetHistogramBuckets()
		require.True(t, ok)
		require.Equal(t, []float64{0.1, 0.2}, sgBuckets)

		smBuckets, ok := returnedLimits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetHistogramBuckets()
		require.True(t, ok)
		require.Equal(t, []float64{0.3, 0.4}, smBuckets)
	})

	t.Run("overwrites field values", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-patch-4")

		// create initial config
		initialPatch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processor: client.LimitsMetricsGeneratorProcessor{
					ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
						HistogramBuckets: &[]float64{0.1, 0.2},
					},
				},
			},
		}
		_, _, err := apiClient.PatchOverrides(initialPatch)
		require.NoError(t, err)

		// PATCH with new histogram buckets (should overwrite)
		updatedPatch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processor: client.LimitsMetricsGeneratorProcessor{
					ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
						HistogramBuckets: &[]float64{0.5, 0.6, 0.7},
					},
				},
			},
		}
		returnedLimits, _, err := apiClient.PatchOverrides(updatedPatch)
		require.NoError(t, err)

		// verify histogram buckets were updated in returned limits
		buckets, ok := returnedLimits.GetMetricsGenerator().GetProcessor().GetServiceGraphs().GetHistogramBuckets()
		require.True(t, ok)
		require.Equal(t, []float64{0.5, 0.6, 0.7}, buckets)
	})

	t.Run("empty top level config doesn't overwrites nested configs", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-patch-5")

		// create initial config
		initialPatch := &client.Limits{
			CostAttribution: client.CostAttribution{
				Dimensions: &map[string]string{"service.name": "service_name"},
			},
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processor: client.LimitsMetricsGeneratorProcessor{
					ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
						HistogramBuckets: &[]float64{0.1, 0.2},
					},
				},
			},
		}
		returnedLimits, _, err := apiClient.PatchOverrides(initialPatch)
		require.NoError(t, err)

		// verify histogram buckets were updated in returned limits
		buckets, ok := returnedLimits.GetMetricsGenerator().GetProcessor().GetServiceGraphs().GetHistogramBuckets()
		require.True(t, ok)
		require.Equal(t, []float64{0.1, 0.2}, buckets)

		// verify that CostAttribution config exists and not touched
		dims, ok := returnedLimits.GetCostAttribution().GetDimensions()
		require.True(t, ok)
		require.Equal(t, map[string]string{"service.name": "service_name"}, dims)

		// PATCH with empty metric generator and it doesn't unset nested configs
		updatedPatch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{},
		}
		_, _, err2 := apiClient.PatchOverrides(updatedPatch)
		require.NoError(t, err2)
		// doing a GET after PATCH to ensure we get fresh config
		returnedLimits2, _, err := apiClient.GetOverrides()
		require.NoError(t, err)

		// verify histogram buckets exist in the config
		config := returnedLimits2.GetMetricsGenerator()
		require.NotNil(t, config)
		buckets, ok = returnedLimits2.GetMetricsGenerator().GetProcessor().GetServiceGraphs().GetHistogramBuckets()
		require.True(t, ok) // should not be wiped out
		require.Equal(t, []float64{0.1, 0.2}, buckets)
	})

	t.Run("version changes after patch", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-patch-6")

		// Create initial config with POST
		initialLimits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(true),
			},
		}
		setEtag, err := apiClient.SetOverrides(initialLimits, "0")
		require.NoError(t, err)

		// Patch config
		patch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				Processor: client.LimitsMetricsGeneratorProcessor{
					ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
						HistogramBuckets: &[]float64{0.1, 0.2},
					},
				},
			},
		}
		_, patchEtag, err := apiClient.PatchOverrides(patch)
		require.NoError(t, err)

		// Verify version changed
		require.NotEqual(t, setEtag, patchEtag)
		require.NotEmpty(t, patchEtag)
	})

	t.Run("handles complex nested config", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-patch-7")

		// Create comprehensive config via PATCH
		patch := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection:              boolPtr(true),
				GenerateNativeHistograms:       histogramModePtr(histograms.HistogramMethodNative),
				NativeHistogramMaxBucketNumber: uint32Ptr(200),
				Processors:                     map[string]struct{}{"span-metrics": {}},
				Processor: client.LimitsMetricsGeneratorProcessor{
					SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
						EnableInstanceLabel: boolPtr(false),
					},
				},
			},
		}
		returnedLimits, _, err := apiClient.PatchOverrides(patch)
		require.NoError(t, err)

		// Verify all fields were set correctly
		disableCollection, ok := returnedLimits.GetMetricsGenerator().GetDisableCollection()
		require.True(t, ok)
		require.True(t, disableCollection)

		generateNativeHistograms, ok := returnedLimits.GetMetricsGenerator().GetGenerateNativeHistograms()
		require.True(t, ok)
		require.Equal(t, histograms.HistogramMethodNative, generateNativeHistograms)

		maxBucketNumber, ok := returnedLimits.GetMetricsGenerator().GetNativeHistogramMaxBucketNumber()
		require.True(t, ok)
		require.Equal(t, uint32(200), maxBucketNumber)

		processors, ok := returnedLimits.GetMetricsGenerator().GetProcessors()
		require.True(t, ok)
		require.ElementsMatch(t, keys(processors.GetMap()), []string{"span-metrics"})

		enableInstanceLabel, ok := returnedLimits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetEnableInstanceLabel()
		require.True(t, ok)
		require.False(t, enableInstanceLabel)
	})
}

func TestOverridesAPI_DELETE(t *testing.T) {
	s, tempo := setupTempo(t, configAllInOneS3Overrides)
	defer s.Close()

	t.Run("config is deleted with correct etag", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-delete-1")

		// create initial config
		limits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(true),
			},
		}
		setEtag, err := apiClient.SetOverrides(limits, "0")
		require.NoError(t, err)

		// delete config
		err = apiClient.DeleteOverrides(setEtag)
		require.NoError(t, err)

		// verify config is gone
		deletedLimits, deletedEtag, err := apiClient.GetOverrides()
		require.Nil(t, deletedLimits)
		require.Empty(t, deletedEtag)
		require.ErrorIs(t, err, httpclient.ErrNotFound)
	})

	t.Run("API returns 428 without if-match header", func(t *testing.T) {
		baseURL := "http://" + tempo.Endpoint(3200)

		// create config first
		apiClient := httpclient.New(baseURL, "tenant-delete-2")
		limits := &client.Limits{
			CostAttribution: client.CostAttribution{
				Dimensions: &map[string]string{"region": "region_label"},
			},
		}
		_, err := apiClient.SetOverrides(limits, "0")
		require.NoError(t, err)

		// try to delete without If-Match header
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/overrides", baseURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Scope-OrgID", "tenant-delete-2")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusPreconditionRequired, resp.StatusCode)
	})

	t.Run("API returns 412 with wrong version ", func(t *testing.T) {
		apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "tenant-delete-wrongver")

		// create config
		limits := &client.Limits{
			MetricsGenerator: client.LimitsMetricsGenerator{
				DisableCollection: boolPtr(false),
			},
		}
		setEtag, err := apiClient.SetOverrides(limits, "0")
		require.NoError(t, err)
		require.NotEmpty(t, setEtag)

		// try to delete with wrong version
		err = apiClient.DeleteOverrides("wrong-version")
		require.ErrorContains(t, err, "412")

		// verify config still exists
		returnedLimits, gotEtag, err := apiClient.GetOverrides()
		require.NoError(t, err)
		require.Equal(t, setEtag, gotEtag)
		require.Equal(t, limits, returnedLimits)
	})
}

// Helper functions for overrides API tests

// setupTempo stands up a new NewTempoAllInOne for e2e test
func setupTempo(t *testing.T, configFilePath string) (*e2e.Scenario, *e2e.HTTPService) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)

	// copy config template to shared directory and expand template variables
	tmplConfig := map[string]any{"Prefix": ""}
	configFile, err := util.CopyTemplateToSharedDir(s, configFilePath, "config.yaml", tmplConfig)
	require.NoError(t, err)

	// set up the backend using the config file
	cfg := app.Config{}
	buff, err := os.ReadFile(configFile)
	require.NoError(t, err)
	err = yaml.UnmarshalStrict(buff, &cfg)
	require.NoError(t, err)
	_, err = backend.New(s, cfg)
	require.NoError(t, err)

	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	return s, tempo
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

func histogramModePtr(h histograms.HistogramMethod) *histograms.HistogramMethod {
	return &h
}

func uint32Ptr(u uint32) *uint32 {
	return &u
}

func keys(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
