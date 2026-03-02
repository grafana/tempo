package overrides

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/tempodb/backend"
)

func TestRuntimeConfigOverrides_loadPerTenantOverrides(t *testing.T) {
	validator := &mockValidator{}

	loader := loadPerTenantOverrides(validator, ConfigTypeNew, false)

	perTenantOverrides := perTenantOverrides{
		TenantLimits: map[string]*Overrides{
			"foo": {Ingestion: IngestionOverrides{TenantShardSize: 6}},
			"bar": {Ingestion: IngestionOverrides{TenantShardSize: 1}},
			"bzz": {Ingestion: IngestionOverrides{TenantShardSize: 3}},
		},
	}
	overridesBytes, err := yaml.Marshal(&perTenantOverrides)
	assert.NoError(t, err)

	// load overrides - validator should pass
	_, err = loader(bytes.NewReader(overridesBytes))
	assert.NoError(t, err)

	// load overrides - validator should reject bar
	validator.f = func(overrides *Overrides) error {
		if overrides.Ingestion.TenantShardSize == 1 {
			return errors.New("no")
		}
		return nil
	}

	_, err = loader(bytes.NewReader(overridesBytes))
	assert.ErrorContains(t, err, "validating overrides for bar failed: no")
}

func TestRuntimeConfigOverrides(t *testing.T) {
	tests := []struct {
		name                        string
		defaultLimits               Overrides
		perTenantOverrides          *perTenantOverrides
		expectedMaxLocalTraces      map[string]int
		expectedMaxGlobalTraces     map[string]int
		expectedMaxBytesPerTrace    map[string]int
		expectedIngestionRateSpans  map[string]int
		expectedIngestionBurstSpans map[string]int
		expectedMaxSearchDuration   map[string]int
	}{
		{
			name: "limits only",
			defaultLimits: Overrides{
				Ingestion: IngestionOverrides{
					MaxGlobalTracesPerUser: 1,
					MaxLocalTracesPerUser:  2,
					BurstSizeBytes:         4,
					RateLimitBytes:         5,
				},
				Global: GlobalOverrides{
					MaxBytesPerTrace: 3,
				},
			},
			expectedMaxGlobalTraces:     map[string]int{"user1": 1, "user2": 1},
			expectedMaxLocalTraces:      map[string]int{"user1": 2, "user2": 2},
			expectedMaxBytesPerTrace:    map[string]int{"user1": 3, "user2": 3},
			expectedIngestionBurstSpans: map[string]int{"user1": 4, "user2": 4},
			expectedIngestionRateSpans:  map[string]int{"user1": 5, "user2": 5},
			expectedMaxSearchDuration:   map[string]int{"user1": 0, "user2": 0},
		},
		{
			name: "basic Overrides",
			defaultLimits: Overrides{
				Ingestion: IngestionOverrides{
					MaxGlobalTracesPerUser: 1,
					MaxLocalTracesPerUser:  2,
					BurstSizeBytes:         4,
					RateLimitBytes:         5,
				},
				Global: GlobalOverrides{
					MaxBytesPerTrace: 3,
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						Ingestion: IngestionOverrides{
							MaxGlobalTracesPerUser: 6,
							MaxLocalTracesPerUser:  7,
							BurstSizeBytes:         9,
							RateLimitBytes:         10,
						},
						Global: GlobalOverrides{
							MaxBytesPerTrace: 8,
						},
						Read: ReadOverrides{
							MaxSearchDuration: model.Duration(11 * time.Second),
						},
					},
				},
			},
			expectedMaxGlobalTraces:     map[string]int{"user1": 6, "user2": 1},
			expectedMaxLocalTraces:      map[string]int{"user1": 7, "user2": 2},
			expectedMaxBytesPerTrace:    map[string]int{"user1": 8, "user2": 3},
			expectedIngestionBurstSpans: map[string]int{"user1": 9, "user2": 4},
			expectedIngestionRateSpans:  map[string]int{"user1": 10, "user2": 5},
			expectedMaxSearchDuration:   map[string]int{"user1": int(11 * time.Second), "user2": 0},
		},
		{
			name: "wildcard override",
			defaultLimits: Overrides{
				Ingestion: IngestionOverrides{
					MaxGlobalTracesPerUser: 1,
					MaxLocalTracesPerUser:  2,
					BurstSizeBytes:         4,
					RateLimitBytes:         5,
				},
				Global: GlobalOverrides{
					MaxBytesPerTrace: 3,
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						Ingestion: IngestionOverrides{
							MaxGlobalTracesPerUser: 6,
							MaxLocalTracesPerUser:  7,
							BurstSizeBytes:         9,
							RateLimitBytes:         10,
						},
						Global: GlobalOverrides{
							MaxBytesPerTrace: 8,
						},
					},
					"*": {
						Ingestion: IngestionOverrides{
							MaxGlobalTracesPerUser: 11,
							MaxLocalTracesPerUser:  12,
							BurstSizeBytes:         14,
							RateLimitBytes:         15,
						},
						Global: GlobalOverrides{
							MaxBytesPerTrace: 13,
						},
						Read: ReadOverrides{
							MaxSearchDuration: model.Duration(16 * time.Second),
						},
						CostAttribution: CostAttributionOverrides{Dimensions: map[string]string{"foo": "bar"}},
					},
				},
			},
			expectedMaxGlobalTraces:     map[string]int{"user1": 6, "user2": 11},
			expectedMaxLocalTraces:      map[string]int{"user1": 7, "user2": 12},
			expectedMaxBytesPerTrace:    map[string]int{"user1": 8, "user2": 13},
			expectedIngestionBurstSpans: map[string]int{"user1": 9, "user2": 14},
			expectedIngestionRateSpans:  map[string]int{"user1": 10, "user2": 15},
			expectedMaxSearchDuration:   map[string]int{"user1": int(16 * time.Second), "user2": int(16 * time.Second)}, // user1 inherits from wildcard for unset fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, tt.defaultLimits, toYamlBytes(t, tt.perTenantOverrides))
			defer cleanup()

			for user, expectedVal := range tt.expectedMaxLocalTraces {
				assert.Equal(t, expectedVal, overrides.MaxLocalTracesPerUser(user))
			}

			for user, expectedVal := range tt.expectedMaxGlobalTraces {
				assert.Equal(t, expectedVal, overrides.MaxGlobalTracesPerUser(user))
			}

			for user, expectedVal := range tt.expectedIngestionBurstSpans {
				assert.Equal(t, expectedVal, overrides.IngestionBurstSizeBytes(user))
			}

			for user, expectedVal := range tt.expectedIngestionRateSpans {
				assert.Equal(t, float64(expectedVal), overrides.IngestionRateLimitBytes(user))
			}

			for user, expectedVal := range tt.expectedMaxSearchDuration {
				assert.Equal(t, time.Duration(expectedVal), overrides.MaxSearchDuration(user))
			}
		})

		t.Run(fmt.Sprintf("%s (legacy)", tt.name), func(t *testing.T) {
			cfg := Config{
				Defaults: tt.defaultLimits,
			}

			if tt.perTenantOverrides != nil {
				overridesFile := filepath.Join(t.TempDir(), "Overrides.yaml")

				legacyOverrides := &perTenantLegacyOverrides{}
				legacyOverrides.TenantLimits = make(map[string]*LegacyOverrides)
				for tenantID, limits := range tt.perTenantOverrides.TenantLimits {
					legacyLimits := limits.toLegacy()
					legacyOverrides.TenantLimits[tenantID] = &legacyLimits
				}
				buff, err := yaml.Marshal(legacyOverrides)
				require.NoError(t, err)

				err = os.WriteFile(overridesFile, buff, 0o700)
				require.NoError(t, err)

				cfg.PerTenantOverrideConfig = overridesFile
				cfg.PerTenantOverridePeriod = model.Duration(time.Hour)
			}

			prometheus.DefaultRegisterer = prometheus.NewRegistry() // have to overwrite the registry or test panics with multiple metric reg
			overrides, err := newRuntimeConfigOverrides(cfg, &mockValidator{}, prometheus.DefaultRegisterer)
			require.NoError(t, err)
			err = services.StartAndAwaitRunning(context.TODO(), overrides)
			require.NoError(t, err)

			for user, expectedVal := range tt.expectedMaxLocalTraces {
				assert.Equal(t, expectedVal, overrides.MaxLocalTracesPerUser(user))
			}

			for user, expectedVal := range tt.expectedMaxGlobalTraces {
				assert.Equal(t, expectedVal, overrides.MaxGlobalTracesPerUser(user))
			}

			for user, expectedVal := range tt.expectedIngestionBurstSpans {
				assert.Equal(t, expectedVal, overrides.IngestionBurstSizeBytes(user))
			}

			for user, expectedVal := range tt.expectedIngestionRateSpans {
				assert.Equal(t, float64(expectedVal), overrides.IngestionRateLimitBytes(user))
			}

			for user, expectedVal := range tt.expectedMaxSearchDuration {
				assert.Equal(t, time.Duration(expectedVal), overrides.MaxSearchDuration(user))
			}

			err = services.StopAndAwaitTerminated(context.TODO(), overrides)
			require.NoError(t, err)
		})
	}
}

func TestMetricsGeneratorOverrides(t *testing.T) {
	tests := []struct {
		name                                 string
		defaultLimits                        Overrides
		perTenantOverrides                   *perTenantOverrides
		expectedEnableTargetInfo             map[string]bool
		expectedDimensionMappings            map[string][]sharedconfig.DimensionMappings
		expectedTargetInfoExcludedDimensions map[string][]string
		expectedEnableInstanceLabel          map[string]bool
	}{
		{
			name: "limits only",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					Processor: ProcessorOverrides{
						SpanMetrics: SpanMetricsOverrides{
							EnableTargetInfo: ptrTo(true),
							DimensionMappings: []sharedconfig.DimensionMappings{
								{
									Name:        "test-name",
									SourceLabel: []string{"service.name"},
									Join:        "/",
								},
							},
							EnableInstanceLabel: ptrTo(false),
						},
					},
				},
			},
			expectedEnableTargetInfo: map[string]bool{"user1": true, "user2": true},
			expectedDimensionMappings: map[string][]sharedconfig.DimensionMappings{
				"user1": {
					{
						Name:        "test-name",
						SourceLabel: []string{"service.name"},
						Join:        "/",
					},
				},
				"user2": {
					{
						Name:        "test-name",
						SourceLabel: []string{"service.name"},
						Join:        "/",
					},
				},
			},
			expectedEnableInstanceLabel: map[string]bool{"user1": false, "user2": false},
		},
		{
			name:          "basic Overrides",
			defaultLimits: Overrides{},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						MetricsGenerator: MetricsGeneratorOverrides{
							Processor: ProcessorOverrides{
								SpanMetrics: SpanMetricsOverrides{
									EnableTargetInfo: ptrTo(true),
									DimensionMappings: []sharedconfig.DimensionMappings{
										{
											Name:        "test-name",
											SourceLabel: []string{"service.name"},
											Join:        "/",
										},
									},
									EnableInstanceLabel: ptrTo(false),
								},
							},
						},
					},
				},
			},
			expectedEnableTargetInfo: map[string]bool{"user1": true, "user2": false},
			expectedDimensionMappings: map[string][]sharedconfig.DimensionMappings{
				"user1": {
					{
						Name:        "test-name",
						SourceLabel: []string{"service.name"},
						Join:        "/",
					},
				},
				"user2": nil,
			},
			expectedEnableInstanceLabel: map[string]bool{"user1": false, "user2": true},
		},
		{
			name: "wildcard override",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					Processor: ProcessorOverrides{
						SpanMetrics: SpanMetricsOverrides{
							EnableTargetInfo: ptrTo(false),
							DimensionMappings: []sharedconfig.DimensionMappings{
								{
									Name:        "test-name",
									SourceLabel: []string{"service.name"},
									Join:        "/",
								},
							},
						},
					},
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						MetricsGenerator: MetricsGeneratorOverrides{
							Processor: ProcessorOverrides{
								SpanMetrics: SpanMetricsOverrides{
									EnableTargetInfo: ptrTo(true),
									DimensionMappings: []sharedconfig.DimensionMappings{
										{
											Name:        "another-name",
											SourceLabel: []string{"service.namespace"},
											Join:        "/",
										},
									},
									TargetInfoExcludedDimensions: []string{"some-label"},
								},
							},
						},
					},
					"*": {
						MetricsGenerator: MetricsGeneratorOverrides{
							Processor: ProcessorOverrides{
								SpanMetrics: SpanMetricsOverrides{
									EnableTargetInfo: ptrTo(false),
									DimensionMappings: []sharedconfig.DimensionMappings{
										{
											Name:        "id-name",
											SourceLabel: []string{"service.instance.id"},
											Join:        "/",
										},
										{
											Name:        "job",
											SourceLabel: []string{"service.namespace", "service.name"},
											Join:        "/",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedEnableTargetInfo: map[string]bool{"user1": true, "user2": false},
			expectedDimensionMappings: map[string][]sharedconfig.DimensionMappings{
				"user1": {
					{
						Name:        "another-name",
						SourceLabel: []string{"service.namespace"},
						Join:        "/",
					},
				},
				"user2": {
					{
						Name:        "id-name",
						SourceLabel: []string{"service.instance.id"},
						Join:        "/",
					},
					{
						Name:        "job",
						SourceLabel: []string{"service.namespace", "service.name"},
						Join:        "/",
					},
				},
			},
			expectedTargetInfoExcludedDimensions: map[string][]string{
				"user1": {"some-label"},
			},
			expectedEnableInstanceLabel: map[string]bool{"user1": true, "user2": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, tt.defaultLimits, toYamlBytes(t, tt.perTenantOverrides))
			defer cleanup()

			for user, expectedVal := range tt.expectedEnableTargetInfo {
				enableTargetInfoValue, _ := overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(user)
				assert.Equal(t, expectedVal, enableTargetInfoValue)
			}

			for user, expectedVal := range tt.expectedDimensionMappings {
				assert.Equal(t, expectedVal, overrides.MetricsGeneratorProcessorSpanMetricsDimensionMappings(user))
			}

			for user, expectedVal := range tt.expectedTargetInfoExcludedDimensions {
				assert.Equal(t, expectedVal, overrides.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(user))
			}

			for user, expectedVal := range tt.expectedEnableInstanceLabel {
				EnableInstanceLabelValue, _ := overrides.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(user)
				assert.Equal(t, expectedVal, EnableInstanceLabelValue)
			}

			err := services.StopAndAwaitTerminated(context.TODO(), overrides)
			require.NoError(t, err)
		})
	}
}

func TestTempoDBOverrides(t *testing.T) {
	tests := []struct {
		name                     string
		defaultLimits            Overrides
		perTenantOverrides       string
		expectedDedicatedColumns map[string]backend.DedicatedColumns
	}{
		{
			name: "limits",
			defaultLimits: Overrides{
				Storage: StorageOverrides{
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "resource", Name: "namespace", Type: "string"},
					},
				},
			},
			expectedDedicatedColumns: map[string]backend.DedicatedColumns{
				"user1": {{Scope: "resource", Name: "namespace", Type: "string"}},
				"user2": {{Scope: "resource", Name: "namespace", Type: "string"}},
			},
		},
		{
			name: "basic overrides",
			defaultLimits: Overrides{
				Storage: StorageOverrides{
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "resource", Name: "namespace", Type: "string"},
					},
				},
			},
			perTenantOverrides: `
overrides:
  user2:
    storage:
      parquet_dedicated_columns:
        - scope: "span"
          name: "http.status"
          type: "int"
`,
			expectedDedicatedColumns: map[string]backend.DedicatedColumns{
				"user1": {{Scope: "resource", Name: "namespace", Type: "string"}},
				"user2": {{Scope: "span", Name: "http.status", Type: "int"}},
			},
		},
		{
			name: "empty dedicated columns override global cfg",
			defaultLimits: Overrides{
				Storage: StorageOverrides{
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "resource", Name: "namespace", Type: "string"},
					},
				},
			},
			perTenantOverrides: `
overrides:
  user1:
  user2:
    storage:
      parquet_dedicated_columns: []
`,
			expectedDedicatedColumns: map[string]backend.DedicatedColumns{
				"user1": {{Scope: "resource", Name: "namespace", Type: "string"}},
				"user2": {},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, tc.defaultLimits, []byte(tc.perTenantOverrides))
			defer cleanup()

			for user, expected := range tc.expectedDedicatedColumns {
				assert.Equal(t, expected, overrides.DedicatedColumns(user))
			}
		})
	}
}

func TestRemoteWriteHeaders(t *testing.T) {
	cfg := Config{
		Defaults: Overrides{
			MetricsGenerator: MetricsGeneratorOverrides{
				RemoteWriteHeaders: map[string]config.Secret{
					"Authorization": "Bearer secret-token",
				},
			},
		},
	}

	overrides, err := newRuntimeConfigOverrides(cfg, &mockValidator{}, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.TODO(), overrides))

	buff := bytes.NewBuffer(nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, overrides.WriteStatusRuntimeConfig(buff, req))

	// Verify the YAML output can be unmarshalled back
	var runtimeConfig struct {
		Defaults           Overrides          `yaml:"defaults"`
		PerTenantOverrides perTenantOverrides `yaml:",inline"`
	}
	require.NoError(t, yaml.UnmarshalStrict(buff.Bytes(), &runtimeConfig))

	assert.Equal(t, "<secret>", string(runtimeConfig.Defaults.MetricsGenerator.RemoteWriteHeaders["Authorization"]))

	fmt.Println(buff.String())
}

func TestExpandEnvOverrides(t *testing.T) {
	const envVar = "TOKEN"
	cfg := Config{
		Defaults: Overrides{
			MetricsGenerator: MetricsGeneratorOverrides{
				RemoteWriteHeaders: map[string]config.Secret{
					"Authorization": "Bearer token",
				},
			},
		},
		ExpandEnv: true,
	}
	// Set the ORG_ID env var
	require.NoError(t, os.Setenv(envVar, "super-secret-token"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv(envVar))
	})

	perTenantOverrides := fmt.Sprintf(`
overrides:
  user1:
    metrics_generator:
      remote_write_headers:
        Authorization: Bearer ${%s}
`, envVar)

	overridesFile := filepath.Join(t.TempDir(), "Overrides.yaml")

	require.NoError(t, os.WriteFile(overridesFile, []byte(perTenantOverrides), 0o700))

	cfg.PerTenantOverrideConfig = overridesFile
	cfg.PerTenantOverridePeriod = model.Duration(time.Hour)

	overrides, err := newRuntimeConfigOverrides(cfg, &mockValidator{}, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.TODO(), overrides))

	expectedRemoteWriteHeaders := map[string]map[string]string{
		"user1": {"Authorization": "Bearer super-secret-token"},
		"user2": {"Authorization": "Bearer token"},
	}
	for user, expected := range expectedRemoteWriteHeaders {
		assert.Equal(t, expected, overrides.MetricsGeneratorRemoteWriteHeaders(user))
	}

	require.NoError(t, services.StopAndAwaitTerminated(context.Background(), overrides))
}

func TestNativeHistogramOverrides(t *testing.T) {
	tests := []struct {
		name                            string
		defaultLimits                   Overrides
		perTenantOverrides              *perTenantOverrides
		nativeHistogramBucketFactor     float64
		nativeHistogramMaxBucketNumber  uint32
		nativeHistogramMinResetDuration time.Duration
	}{
		{
			name: "defaults only",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					NativeHistogramBucketFactor:     1.5,
					NativeHistogramMaxBucketNumber:  20,
					NativeHistogramMinResetDuration: 5 * time.Minute,
				},
			},
			nativeHistogramBucketFactor:     1.5,
			nativeHistogramMaxBucketNumber:  20,
			nativeHistogramMinResetDuration: 5 * time.Minute,
		},
		{
			name: "defaults only",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					NativeHistogramBucketFactor:     1.5,
					NativeHistogramMaxBucketNumber:  20,
					NativeHistogramMinResetDuration: 5 * time.Minute,
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						MetricsGenerator: MetricsGeneratorOverrides{
							NativeHistogramBucketFactor:     2.0,
							NativeHistogramMaxBucketNumber:  30,
							NativeHistogramMinResetDuration: 10 * time.Minute,
						},
					},
				},
			},
			nativeHistogramBucketFactor:     2.0,
			nativeHistogramMaxBucketNumber:  30,
			nativeHistogramMinResetDuration: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, tt.defaultLimits, toYamlBytes(t, tt.perTenantOverrides))
			defer cleanup()

			assert.Equal(t, tt.nativeHistogramBucketFactor, overrides.MetricsGeneratorNativeHistogramBucketFactor("user1"))
			assert.Equal(t, tt.nativeHistogramMaxBucketNumber, overrides.MetricsGeneratorNativeHistogramMaxBucketNumber("user1"))
			assert.Equal(t, tt.nativeHistogramMinResetDuration, overrides.MetricsGeneratorNativeHistogramMinResetDuration("user1"))

			err := services.StopAndAwaitTerminated(context.TODO(), overrides)
			require.NoError(t, err)
		})
	}
}

func TestMetricsGeneratorMaxCardinalityPerLabel(t *testing.T) {
	tests := []struct {
		name               string
		defaultLimits      Overrides
		perTenantOverrides *perTenantOverrides
		expected           map[string]uint64
	}{
		{
			name: "default enabled, no tenant override",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					MaxCardinalityPerLabel: ptrTo(uint64(100)),
				},
			},
			expected: map[string]uint64{"user1": 100, "user2": 100},
		},
		{
			name:          "default disabled, tenant enables",
			defaultLimits: Overrides{},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						MetricsGenerator: MetricsGeneratorOverrides{
							MaxCardinalityPerLabel: ptrTo(uint64(50)),
						},
					},
				},
			},
			expected: map[string]uint64{"user1": 50, "user2": 0},
		},
		{
			name: "default enabled, tenant disables with 0",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					MaxCardinalityPerLabel: ptrTo(uint64(100)),
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						MetricsGenerator: MetricsGeneratorOverrides{
							MaxCardinalityPerLabel: ptrTo(uint64(0)),
						},
					},
				},
			},
			expected: map[string]uint64{"user1": 0, "user2": 100},
		},
		{
			name: "default enabled, tenant overrides with higher value",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					MaxCardinalityPerLabel: ptrTo(uint64(100)),
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: map[string]*Overrides{
					"user1": {
						MetricsGenerator: MetricsGeneratorOverrides{
							MaxCardinalityPerLabel: ptrTo(uint64(500)),
						},
					},
				},
			},
			expected: map[string]uint64{"user1": 500, "user2": 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, tt.defaultLimits, toYamlBytes(t, tt.perTenantOverrides))
			defer cleanup()

			for user, expected := range tt.expected {
				require.Equal(t, expected, overrides.MetricsGeneratorMaxCardinalityPerLabel(user), "user: %s", user)
			}
		})
	}
}

func createAndInitializeRuntimeOverridesManager(t *testing.T, defaultLimits Overrides, perTenantOverrides []byte) (Service, func()) {
	cfg := Config{
		Defaults: defaultLimits,
	}

	if perTenantOverrides != nil {
		overridesFile := filepath.Join(t.TempDir(), "Overrides.yaml")

		err := os.WriteFile(overridesFile, perTenantOverrides, 0o700)
		require.NoError(t, err)

		cfg.PerTenantOverrideConfig = overridesFile
		cfg.PerTenantOverridePeriod = model.Duration(time.Hour)
	}

	prometheus.DefaultRegisterer = prometheus.NewRegistry() // have to overwrite the registry or test panics with multiple metric reg
	overrides, err := newRuntimeConfigOverrides(cfg, &mockValidator{}, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	err = services.StartAndAwaitRunning(context.TODO(), overrides)
	require.NoError(t, err)

	return overrides, func() {
		err := services.StopAndAwaitTerminated(context.TODO(), overrides)
		require.NoError(t, err)
	}
}

// TestOverrideFallbackChain verifies the per-tenant -> wildcard -> default fallback
// chain for all override accessor methods. This catches cases where a wildcard override
// (e.g. '*': some_tier) shadows the static default config because the wildcard struct
// doesn't set a particular field, causing it to resolve to the zero value instead of
// falling back to the configured default.
//
// Setup:
//   - "tenant1": has explicit per-tenant overrides with distinct non-zero values
//   - "tenant2": not in per-tenant overrides, falls through to wildcard "*"
//   - "*": wildcard with a MINIMAL set of fields (simulates a tier that doesn't set everything)
//   - defaultLimits: has non-zero values for ALL fields
//
// Expected behavior:
//   - tenant1: always gets per-tenant values
//   - tenant2: for fields set on wildcard, gets wildcard values;
//     for fields NOT set on wildcard, should get default values (this is the bug detector)
func TestOverrideFallbackChain(t *testing.T) {
	// defaults with non-zero values for ALL fields across all override types.
	// This is the static config that Tempo loads at startup.
	artificialDelay := 21 * time.Second
	defaults := Overrides{
		Ingestion: IngestionOverrides{
			RateLimitBytes:         1000,
			BurstSizeBytes:         2000,
			MaxLocalTracesPerUser:  3000,
			MaxGlobalTracesPerUser: 4000,
			TenantShardSize:        5,
			MaxAttributeBytes:      6000,
			ArtificialDelay:        &artificialDelay,
			RetryInfoEnabled:       ptrTo(true),
		},
		Read: ReadOverrides{
			MaxBytesPerTagValuesQuery:  7000,
			MaxBlocksPerTagValuesQuery: 8,
			MaxSearchDuration:          model.Duration(9 * time.Minute),
			MaxMetricsDuration:         model.Duration(10 * time.Minute),
			UnsafeQueryHints:           ptrTo(true),
			LeftPadTraceIDs:            ptrTo(true),
		},
		Global: GlobalOverrides{
			MaxBytesPerTrace: 9000,
		},
		Compaction: CompactionOverrides{
			CompactionWindow:   model.Duration(10 * time.Minute),
			BlockRetention:     model.Duration(720 * time.Hour),
			CompactionDisabled: ptrTo(true),
		},
		Forwarders: []string{"default-forwarder"},
		MetricsGenerator: MetricsGeneratorOverrides{
			RingSize:                        11,
			Processors:                      map[string]struct{}{"service-graphs": {}, "span-metrics": {}},
			MaxActiveSeries:                 12000,
			MaxActiveEntities:               13000,
			CollectionInterval:              14 * time.Second,
			DisableCollection:               ptrTo(true),
			IngestionSlack:                  15 * time.Minute,
			TraceIDLabelName:                "default_trace_id",
			SpanNameSanitization:            "default_mode",
			MaxCardinalityPerLabel:          ptrTo(uint64(16000)),
			NativeHistogramBucketFactor:     1.5,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: 17 * time.Minute,
			GenerateNativeHistograms:        "classic",
			RemoteWriteHeaders:              RemoteWriteHeaders{"X-Default": "default-val"},
			Forwarder: ForwarderOverrides{
				QueueSize: 18000,
				Workers:   19,
			},
			Processor: ProcessorOverrides{
				ServiceGraphs: ServiceGraphsOverrides{
					HistogramBuckets:                      []float64{1, 2, 5, 10},
					Dimensions:                            []string{"default-sg-dim"},
					PeerAttributes:                        []string{"default-peer-attr"},
					EnableClientServerPrefix:              ptrTo(true),
					EnableMessagingSystemLatencyHistogram: ptrTo(true),
					EnableVirtualNodeLabel:                ptrTo(true),
					SpanMultiplierKey:                     "default-sg-multiplier",
				},
				SpanMetrics: SpanMetricsOverrides{
					HistogramBuckets:             []float64{0.5, 1, 2},
					Dimensions:                   []string{"default-sm-dim"},
					IntrinsicDimensions:          map[string]bool{"service": true},
					DimensionMappings:            []sharedconfig.DimensionMappings{{Name: "default-mapping", SourceLabel: []string{"service.name"}, Join: "/"}},
					EnableTargetInfo:             ptrTo(true),
					TargetInfoExcludedDimensions: []string{"default-excluded"},
					EnableInstanceLabel:          ptrTo(false),
					SpanMultiplierKey:            "default-sm-multiplier",
				},
				LocalBlocks: LocalBlocksOverrides{
					MaxLiveTraces:        22000,
					MaxBlockDuration:     23 * time.Minute,
					MaxBlockBytes:        24000,
					FlushCheckPeriod:     25 * time.Second,
					TraceIdlePeriod:      26 * time.Second,
					CompleteBlockTimeout: 27 * time.Second,
				},
				HostInfo: HostInfoOverrides{
					HostIdentifiers: []string{"default-host-id"},
					MetricName:      "default_host_metric",
				},
			},
		},
		CostAttribution: CostAttributionOverrides{
			MaxCardinality: 20000,
			Dimensions:     map[string]string{"default-dim": "default-val"},
		},
		Storage: StorageOverrides{
			DedicatedColumns: backend.DedicatedColumns{
				{Scope: "resource", Name: "default-col", Type: "string"},
			},
		},
	}

	// wildcard: only sets a MINIMAL set of fields (simulates a real tier like default_free_pro_tier).
	// All other fields are left at their zero value, which is the scenario that triggers the fallback bug.
	wildcardOverrides := &Overrides{
		Ingestion: IngestionOverrides{
			MaxLocalTracesPerUser: 500,
			RateLimitBytes:        600,
			BurstSizeBytes:        700,
		},
		Global: GlobalOverrides{
			MaxBytesPerTrace: 800,
		},
	}

	// tenant1: has explicit overrides with completely different non-zero values for ALL fields.
	tenant1ArtificialDelay := 210 * time.Millisecond
	tenant1Overrides := &Overrides{
		Ingestion: IngestionOverrides{
			RateLimitBytes:         100,
			BurstSizeBytes:         200,
			MaxLocalTracesPerUser:  300,
			MaxGlobalTracesPerUser: 400,
			TenantShardSize:        50,
			MaxAttributeBytes:      600,
			ArtificialDelay:        &tenant1ArtificialDelay,
			RetryInfoEnabled:       ptrTo(true),
		},
		Read: ReadOverrides{
			MaxBytesPerTagValuesQuery:  700,
			MaxBlocksPerTagValuesQuery: 80,
			MaxSearchDuration:          model.Duration(90 * time.Minute),
			MaxMetricsDuration:         model.Duration(100 * time.Minute),
			UnsafeQueryHints:           ptrTo(true),
			LeftPadTraceIDs:            ptrTo(true),
		},
		Global: GlobalOverrides{
			MaxBytesPerTrace: 900,
		},
		Compaction: CompactionOverrides{
			CompactionWindow:   model.Duration(100 * time.Minute),
			BlockRetention:     model.Duration(100 * time.Hour),
			CompactionDisabled: ptrTo(true),
		},
		Forwarders: []string{"tenant1-forwarder"},
		MetricsGenerator: MetricsGeneratorOverrides{
			RingSize:                        110,
			Processors:                      map[string]struct{}{"span-metrics": {}},
			MaxActiveSeries:                 1200,
			MaxActiveEntities:               1300,
			CollectionInterval:              140 * time.Second,
			DisableCollection:               ptrTo(true),
			IngestionSlack:                  150 * time.Minute,
			TraceIDLabelName:                "tenant1_trace_id",
			SpanNameSanitization:            "tenant1_mode",
			MaxCardinalityPerLabel:          ptrTo(uint64(1600)),
			NativeHistogramBucketFactor:     2.0,
			NativeHistogramMaxBucketNumber:  200,
			NativeHistogramMinResetDuration: 170 * time.Minute,
			GenerateNativeHistograms:        "native",
			RemoteWriteHeaders:              RemoteWriteHeaders{"X-Tenant1": "t1-val"},
			Forwarder: ForwarderOverrides{
				QueueSize: 1800,
				Workers:   190,
			},
			Processor: ProcessorOverrides{
				ServiceGraphs: ServiceGraphsOverrides{
					HistogramBuckets:                      []float64{10, 20, 50},
					Dimensions:                            []string{"tenant1-sg-dim"},
					PeerAttributes:                        []string{"tenant1-peer-attr"},
					EnableClientServerPrefix:              ptrTo(false),
					EnableMessagingSystemLatencyHistogram: ptrTo(false),
					EnableVirtualNodeLabel:                ptrTo(false),
					SpanMultiplierKey:                     "tenant1-sg-multiplier",
				},
				SpanMetrics: SpanMetricsOverrides{
					HistogramBuckets:             []float64{5, 10, 20},
					Dimensions:                   []string{"tenant1-sm-dim"},
					IntrinsicDimensions:          map[string]bool{"span_kind": true},
					DimensionMappings:            []sharedconfig.DimensionMappings{{Name: "tenant1-mapping", SourceLabel: []string{"service.namespace"}, Join: "-"}},
					EnableTargetInfo:             ptrTo(false),
					TargetInfoExcludedDimensions: []string{"tenant1-excluded"},
					EnableInstanceLabel:          ptrTo(true),
					SpanMultiplierKey:            "tenant1-sm-multiplier",
				},
				LocalBlocks: LocalBlocksOverrides{
					MaxLiveTraces:        2200,
					MaxBlockDuration:     230 * time.Minute,
					MaxBlockBytes:        2400,
					FlushCheckPeriod:     250 * time.Second,
					TraceIdlePeriod:      260 * time.Second,
					CompleteBlockTimeout: 270 * time.Second,
				},
				HostInfo: HostInfoOverrides{
					HostIdentifiers: []string{"tenant1-host-id"},
					MetricName:      "tenant1_host_metric",
				},
			},
		},
		CostAttribution: CostAttributionOverrides{
			MaxCardinality: 2000,
			Dimensions:     map[string]string{"tenant1-dim": "t1-val"},
		},
		Storage: StorageOverrides{
			DedicatedColumns: backend.DedicatedColumns{
				{Scope: "span", Name: "tenant1-col", Type: "int"},
			},
		},
	}

	perTenantOverrides := &perTenantOverrides{
		TenantLimits: map[string]*Overrides{
			"tenant1": tenant1Overrides,
			"*":       wildcardOverrides,
		},
	}

	overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, defaults, toYamlBytes(t, perTenantOverrides))
	defer cleanup()

	// Each test case represents one override accessor method.
	// tenant1Expected: the per-tenant value (should always come from tenant1Overrides)
	// tenant2Expected: should come from wildcard if set there, otherwise from defaults
	type testCase struct {
		name            string
		tenant1Got      interface{}
		tenant1Expected interface{}
		tenant2Got      interface{}
		tenant2Expected interface{} // This is the key: should be default when wildcard doesn't set the field
	}

	// --- helpers for *bool multi-return methods ---
	t1EnableTargetInfo, t1EnableTargetInfoIsSet := overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo("tenant1")
	t2EnableTargetInfo, t2EnableTargetInfoIsSet := overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo("tenant2")
	t1EnableInstanceLabel, t1EnableInstanceLabelIsSet := overrides.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel("tenant1")
	t2EnableInstanceLabel, t2EnableInstanceLabelIsSet := overrides.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel("tenant2")
	t1EnableMsgLatency, t1EnableMsgLatencyIsSet := overrides.MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram("tenant1")
	t2EnableMsgLatency, t2EnableMsgLatencyIsSet := overrides.MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram("tenant2")
	t1EnableVirtualNode, t1EnableVirtualNodeIsSet := overrides.MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel("tenant1")
	t2EnableVirtualNode, t2EnableVirtualNodeIsSet := overrides.MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel("tenant2")
	t1ArtificialDelay, t1ArtificialDelayIsSet := overrides.IngestionArtificialDelay("tenant1")
	t2ArtificialDelay, t2ArtificialDelayIsSet := overrides.IngestionArtificialDelay("tenant2")

	tests := []testCase{
		// =====================================================================
		// Ingestion overrides (wildcard DOES set these)
		// =====================================================================
		{
			name:            "MaxLocalTracesPerUser",
			tenant1Got:      overrides.MaxLocalTracesPerUser("tenant1"),
			tenant1Expected: 300,
			tenant2Got:      overrides.MaxLocalTracesPerUser("tenant2"),
			tenant2Expected: 500, // wildcard sets this
		},
		{
			name:            "IngestionRateLimitBytes",
			tenant1Got:      overrides.IngestionRateLimitBytes("tenant1"),
			tenant1Expected: float64(100),
			tenant2Got:      overrides.IngestionRateLimitBytes("tenant2"),
			tenant2Expected: float64(600), // wildcard sets this
		},
		{
			name:            "IngestionBurstSizeBytes",
			tenant1Got:      overrides.IngestionBurstSizeBytes("tenant1"),
			tenant1Expected: 200,
			tenant2Got:      overrides.IngestionBurstSizeBytes("tenant2"),
			tenant2Expected: 700, // wildcard sets this
		},
		{
			name:            "MaxBytesPerTrace",
			tenant1Got:      overrides.MaxBytesPerTrace("tenant1"),
			tenant1Expected: 900,
			tenant2Got:      overrides.MaxBytesPerTrace("tenant2"),
			tenant2Expected: 800, // wildcard sets this
		},

		// =====================================================================
		// Ingestion overrides (wildcard does NOT set these)
		// =====================================================================
		{
			name:            "MaxGlobalTracesPerUser",
			tenant1Got:      overrides.MaxGlobalTracesPerUser("tenant1"),
			tenant1Expected: 400,
			tenant2Got:      overrides.MaxGlobalTracesPerUser("tenant2"),
			tenant2Expected: 4000,
		},
		{
			name:            "IngestionTenantShardSize",
			tenant1Got:      overrides.IngestionTenantShardSize("tenant1"),
			tenant1Expected: 50,
			tenant2Got:      overrides.IngestionTenantShardSize("tenant2"),
			tenant2Expected: 5,
		},
		{
			name:            "IngestionMaxAttributeBytes",
			tenant1Got:      overrides.IngestionMaxAttributeBytes("tenant1"),
			tenant1Expected: 600,
			tenant2Got:      overrides.IngestionMaxAttributeBytes("tenant2"),
			tenant2Expected: 6000,
		},
		// *time.Duration pointer type
		{
			name:            "IngestionArtificialDelay",
			tenant1Got:      t1ArtificialDelay,
			tenant1Expected: 210 * time.Millisecond,
			tenant2Got:      t2ArtificialDelay,
			tenant2Expected: 21 * time.Second,
		},
		{
			name:            "IngestionArtificialDelay_isSet",
			tenant1Got:      t1ArtificialDelayIsSet,
			tenant1Expected: true,
			tenant2Got:      t2ArtificialDelayIsSet,
			tenant2Expected: true, // should be true because default sets it
		},
		// bool type
		{
			name:            "IngestionRetryInfoEnabled",
			tenant1Got:      overrides.IngestionRetryInfoEnabled("tenant1"),
			tenant1Expected: true,
			tenant2Got:      overrides.IngestionRetryInfoEnabled("tenant2"),
			tenant2Expected: true, // default is true, wildcard doesn't set it
		},

		// =====================================================================
		// Read overrides (wildcard does NOT set these)
		// =====================================================================
		{
			name:            "MaxBytesPerTagValuesQuery",
			tenant1Got:      overrides.MaxBytesPerTagValuesQuery("tenant1"),
			tenant1Expected: 700,
			tenant2Got:      overrides.MaxBytesPerTagValuesQuery("tenant2"),
			tenant2Expected: 7000,
		},
		{
			name:            "MaxBlocksPerTagValuesQuery",
			tenant1Got:      overrides.MaxBlocksPerTagValuesQuery("tenant1"),
			tenant1Expected: 80,
			tenant2Got:      overrides.MaxBlocksPerTagValuesQuery("tenant2"),
			tenant2Expected: 8,
		},
		{
			name:            "MaxSearchDuration",
			tenant1Got:      overrides.MaxSearchDuration("tenant1"),
			tenant1Expected: 90 * time.Minute,
			tenant2Got:      overrides.MaxSearchDuration("tenant2"),
			tenant2Expected: 9 * time.Minute,
		},
		{
			name:            "MaxMetricsDuration",
			tenant1Got:      overrides.MaxMetricsDuration("tenant1"),
			tenant1Expected: 100 * time.Minute,
			tenant2Got:      overrides.MaxMetricsDuration("tenant2"),
			tenant2Expected: 10 * time.Minute,
		},
		{
			name:            "UnsafeQueryHints",
			tenant1Got:      overrides.UnsafeQueryHints("tenant1"),
			tenant1Expected: true,
			tenant2Got:      overrides.UnsafeQueryHints("tenant2"),
			tenant2Expected: true,
		},
		{
			name:            "LeftPadTraceIDs",
			tenant1Got:      overrides.LeftPadTraceIDs("tenant1"),
			tenant1Expected: true,
			tenant2Got:      overrides.LeftPadTraceIDs("tenant2"),
			tenant2Expected: true,
		},

		// =====================================================================
		// Global overrides (wildcard does NOT set these)
		// =====================================================================
		{
			name:            "Forwarders",
			tenant1Got:      overrides.Forwarders("tenant1"),
			tenant1Expected: []string{"tenant1-forwarder"},
			tenant2Got:      overrides.Forwarders("tenant2"),
			tenant2Expected: []string{"default-forwarder"},
		},

		// =====================================================================
		// Compaction overrides (wildcard does NOT set these)
		// =====================================================================
		{
			name:            "MaxCompactionRange",
			tenant1Got:      overrides.MaxCompactionRange("tenant1"),
			tenant1Expected: 100 * time.Minute,
			tenant2Got:      overrides.MaxCompactionRange("tenant2"),
			tenant2Expected: 10 * time.Minute,
		},
		{
			name:            "BlockRetention",
			tenant1Got:      overrides.BlockRetention("tenant1"),
			tenant1Expected: 100 * time.Hour,
			tenant2Got:      overrides.BlockRetention("tenant2"),
			tenant2Expected: 720 * time.Hour,
		},
		{
			name:            "CompactionDisabled",
			tenant1Got:      overrides.CompactionDisabled("tenant1"),
			tenant1Expected: true,
			tenant2Got:      overrides.CompactionDisabled("tenant2"),
			tenant2Expected: true,
		},

		// =====================================================================
		// MetricsGenerator scalar overrides (wildcard does NOT set any)
		// =====================================================================
		{
			name:            "MetricsGeneratorRingSize",
			tenant1Got:      overrides.MetricsGeneratorRingSize("tenant1"),
			tenant1Expected: 110,
			tenant2Got:      overrides.MetricsGeneratorRingSize("tenant2"),
			tenant2Expected: 11,
		},
		{
			name:            "MetricsGeneratorProcessors",
			tenant1Got:      overrides.MetricsGeneratorProcessors("tenant1"),
			tenant1Expected: map[string]struct{}{"span-metrics": {}},
			tenant2Got:      overrides.MetricsGeneratorProcessors("tenant2"),
			tenant2Expected: map[string]struct{}{"service-graphs": {}, "span-metrics": {}},
		},
		{
			name:            "MetricsGeneratorMaxActiveSeries",
			tenant1Got:      overrides.MetricsGeneratorMaxActiveSeries("tenant1"),
			tenant1Expected: uint32(1200),
			tenant2Got:      overrides.MetricsGeneratorMaxActiveSeries("tenant2"),
			tenant2Expected: uint32(12000),
		},
		{
			name:            "MetricsGeneratorMaxActiveEntities",
			tenant1Got:      overrides.MetricsGeneratorMaxActiveEntities("tenant1"),
			tenant1Expected: uint32(1300),
			tenant2Got:      overrides.MetricsGeneratorMaxActiveEntities("tenant2"),
			tenant2Expected: uint32(13000),
		},
		{
			name:            "MetricsGeneratorCollectionInterval",
			tenant1Got:      overrides.MetricsGeneratorCollectionInterval("tenant1"),
			tenant1Expected: 140 * time.Second,
			tenant2Got:      overrides.MetricsGeneratorCollectionInterval("tenant2"),
			tenant2Expected: 14 * time.Second,
		},
		{
			name:            "MetricsGeneratorDisableCollection",
			tenant1Got:      overrides.MetricsGeneratorDisableCollection("tenant1"),
			tenant1Expected: true,
			tenant2Got:      overrides.MetricsGeneratorDisableCollection("tenant2"),
			tenant2Expected: true,
		},
		{
			name:            "MetricsGeneratorIngestionSlack",
			tenant1Got:      overrides.MetricsGeneratorIngestionSlack("tenant1"),
			tenant1Expected: 150 * time.Minute,
			tenant2Got:      overrides.MetricsGeneratorIngestionSlack("tenant2"),
			tenant2Expected: 15 * time.Minute,
		},
		{
			name:            "MetricsGeneratorTraceIDLabelName",
			tenant1Got:      overrides.MetricsGeneratorTraceIDLabelName("tenant1"),
			tenant1Expected: "tenant1_trace_id",
			tenant2Got:      overrides.MetricsGeneratorTraceIDLabelName("tenant2"),
			tenant2Expected: "default_trace_id",
		},
		{
			name:            "MetricsGeneratorSpanNameSanitization",
			tenant1Got:      overrides.MetricsGeneratorSpanNameSanitization("tenant1"),
			tenant1Expected: "tenant1_mode",
			tenant2Got:      overrides.MetricsGeneratorSpanNameSanitization("tenant2"),
			tenant2Expected: "default_mode",
		},
		{
			name:            "MetricsGeneratorMaxCardinalityPerLabel",
			tenant1Got:      overrides.MetricsGeneratorMaxCardinalityPerLabel("tenant1"),
			tenant1Expected: uint64(1600),
			tenant2Got:      overrides.MetricsGeneratorMaxCardinalityPerLabel("tenant2"),
			tenant2Expected: uint64(16000),
		},
		{
			name:            "MetricsGeneratorNativeHistogramBucketFactor",
			tenant1Got:      overrides.MetricsGeneratorNativeHistogramBucketFactor("tenant1"),
			tenant1Expected: 2.0,
			tenant2Got:      overrides.MetricsGeneratorNativeHistogramBucketFactor("tenant2"),
			tenant2Expected: 1.5,
		},
		{
			name:            "MetricsGeneratorNativeHistogramMaxBucketNumber",
			tenant1Got:      overrides.MetricsGeneratorNativeHistogramMaxBucketNumber("tenant1"),
			tenant1Expected: uint32(200),
			tenant2Got:      overrides.MetricsGeneratorNativeHistogramMaxBucketNumber("tenant2"),
			tenant2Expected: uint32(100),
		},
		{
			name:            "MetricsGeneratorNativeHistogramMinResetDuration",
			tenant1Got:      overrides.MetricsGeneratorNativeHistogramMinResetDuration("tenant1"),
			tenant1Expected: 170 * time.Minute,
			tenant2Got:      overrides.MetricsGeneratorNativeHistogramMinResetDuration("tenant2"),
			tenant2Expected: 17 * time.Minute,
		},
		{
			name:            "MetricsGeneratorGenerateNativeHistograms",
			tenant1Got:      overrides.MetricsGeneratorGenerateNativeHistograms("tenant1"),
			tenant1Expected: histograms.HistogramMethod("native"),
			tenant2Got:      overrides.MetricsGeneratorGenerateNativeHistograms("tenant2"),
			tenant2Expected: histograms.HistogramMethod("classic"),
		},
		{
			name:            "MetricsGeneratorRemoteWriteHeaders",
			tenant1Got:      overrides.MetricsGeneratorRemoteWriteHeaders("tenant1"),
			tenant1Expected: map[string]string{"X-Tenant1": "<secret>"}, // config.Secret masks values after YAML round-trip
			tenant2Got:      overrides.MetricsGeneratorRemoteWriteHeaders("tenant2"),
			tenant2Expected: map[string]string{"X-Default": "default-val"}, // from raw defaults struct, not YAML round-tripped
		},
		{
			name:            "MetricsGeneratorForwarderQueueSize",
			tenant1Got:      overrides.MetricsGeneratorForwarderQueueSize("tenant1"),
			tenant1Expected: 1800,
			tenant2Got:      overrides.MetricsGeneratorForwarderQueueSize("tenant2"),
			tenant2Expected: 18000,
		},
		{
			name:            "MetricsGeneratorForwarderWorkers",
			tenant1Got:      overrides.MetricsGeneratorForwarderWorkers("tenant1"),
			tenant1Expected: 190,
			tenant2Got:      overrides.MetricsGeneratorForwarderWorkers("tenant2"),
			tenant2Expected: 19,
		},

		// =====================================================================
		// MetricsGenerator Processor - ServiceGraphs (wildcard does NOT set any)
		// =====================================================================
		{
			name:            "ServiceGraphsHistogramBuckets",
			tenant1Got:      overrides.MetricsGeneratorProcessorServiceGraphsHistogramBuckets("tenant1"),
			tenant1Expected: []float64{10, 20, 50},
			tenant2Got:      overrides.MetricsGeneratorProcessorServiceGraphsHistogramBuckets("tenant2"),
			tenant2Expected: []float64{1, 2, 5, 10},
		},
		{
			name:            "ServiceGraphsDimensions",
			tenant1Got:      overrides.MetricsGeneratorProcessorServiceGraphsDimensions("tenant1"),
			tenant1Expected: []string{"tenant1-sg-dim"},
			tenant2Got:      overrides.MetricsGeneratorProcessorServiceGraphsDimensions("tenant2"),
			tenant2Expected: []string{"default-sg-dim"},
		},
		{
			name:            "ServiceGraphsPeerAttributes",
			tenant1Got:      overrides.MetricsGeneratorProcessorServiceGraphsPeerAttributes("tenant1"),
			tenant1Expected: []string{"tenant1-peer-attr"},
			tenant2Got:      overrides.MetricsGeneratorProcessorServiceGraphsPeerAttributes("tenant2"),
			tenant2Expected: []string{"default-peer-attr"},
		},
		{
			name:            "ServiceGraphsEnableClientServerPrefix",
			tenant1Got:      overrides.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix("tenant1"),
			tenant1Expected: false,
			tenant2Got:      overrides.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix("tenant2"),
			tenant2Expected: true, // default sets *bool to true
		},
		{
			name:            "ServiceGraphsEnableMessagingSystemLatencyHistogram",
			tenant1Got:      t1EnableMsgLatency,
			tenant1Expected: false,
			tenant2Got:      t2EnableMsgLatency,
			tenant2Expected: true, // default sets *bool to true
		},
		{
			name:            "ServiceGraphsEnableMessagingSystemLatencyHistogram_isSet",
			tenant1Got:      t1EnableMsgLatencyIsSet,
			tenant1Expected: true,
			tenant2Got:      t2EnableMsgLatencyIsSet,
			tenant2Expected: true, // default sets *bool, should be "isSet=true"
		},
		{
			name:            "ServiceGraphsEnableVirtualNodeLabel",
			tenant1Got:      t1EnableVirtualNode,
			tenant1Expected: false,
			tenant2Got:      t2EnableVirtualNode,
			tenant2Expected: true, // default sets *bool to true
		},
		{
			name:            "ServiceGraphsEnableVirtualNodeLabel_isSet",
			tenant1Got:      t1EnableVirtualNodeIsSet,
			tenant1Expected: true,
			tenant2Got:      t2EnableVirtualNodeIsSet,
			tenant2Expected: true,
		},
		{
			name:            "ServiceGraphsSpanMultiplierKey",
			tenant1Got:      overrides.MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey("tenant1"),
			tenant1Expected: "tenant1-sg-multiplier",
			tenant2Got:      overrides.MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey("tenant2"),
			tenant2Expected: "default-sg-multiplier",
		},

		// =====================================================================
		// MetricsGenerator Processor - SpanMetrics (wildcard does NOT set any)
		// =====================================================================
		{
			name:            "SpanMetricsHistogramBuckets",
			tenant1Got:      overrides.MetricsGeneratorProcessorSpanMetricsHistogramBuckets("tenant1"),
			tenant1Expected: []float64{5, 10, 20},
			tenant2Got:      overrides.MetricsGeneratorProcessorSpanMetricsHistogramBuckets("tenant2"),
			tenant2Expected: []float64{0.5, 1, 2},
		},
		{
			name:            "SpanMetricsDimensions",
			tenant1Got:      overrides.MetricsGeneratorProcessorSpanMetricsDimensions("tenant1"),
			tenant1Expected: []string{"tenant1-sm-dim"},
			tenant2Got:      overrides.MetricsGeneratorProcessorSpanMetricsDimensions("tenant2"),
			tenant2Expected: []string{"default-sm-dim"},
		},
		{
			name:            "SpanMetricsIntrinsicDimensions",
			tenant1Got:      overrides.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions("tenant1"),
			tenant1Expected: map[string]bool{"span_kind": true},
			tenant2Got:      overrides.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions("tenant2"),
			tenant2Expected: map[string]bool{"service": true},
		},
		{
			name:            "SpanMetricsFilterPolicies",
			tenant1Got:      overrides.MetricsGeneratorProcessorSpanMetricsFilterPolicies("tenant1"),
			tenant1Expected: []filterconfig.FilterPolicy(nil),
			tenant2Got:      overrides.MetricsGeneratorProcessorSpanMetricsFilterPolicies("tenant2"),
			tenant2Expected: []filterconfig.FilterPolicy(nil), // both nil since neither set FilterPolicies
		},
		{
			name:            "SpanMetricsDimensionMappings",
			tenant1Got:      overrides.MetricsGeneratorProcessorSpanMetricsDimensionMappings("tenant1"),
			tenant1Expected: []sharedconfig.DimensionMappings{{Name: "tenant1-mapping", SourceLabel: []string{"service.namespace"}, Join: "-"}},
			tenant2Got:      overrides.MetricsGeneratorProcessorSpanMetricsDimensionMappings("tenant2"),
			tenant2Expected: []sharedconfig.DimensionMappings{{Name: "default-mapping", SourceLabel: []string{"service.name"}, Join: "/"}},
		},
		{
			name:            "SpanMetricsEnableTargetInfo",
			tenant1Got:      t1EnableTargetInfo,
			tenant1Expected: false,
			tenant2Got:      t2EnableTargetInfo,
			tenant2Expected: true, // default sets *bool to true
		},
		{
			name:            "SpanMetricsEnableTargetInfo_isSet",
			tenant1Got:      t1EnableTargetInfoIsSet,
			tenant1Expected: true,
			tenant2Got:      t2EnableTargetInfoIsSet,
			tenant2Expected: true,
		},
		{
			name:            "SpanMetricsTargetInfoExcludedDimensions",
			tenant1Got:      overrides.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions("tenant1"),
			tenant1Expected: []string{"tenant1-excluded"},
			tenant2Got:      overrides.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions("tenant2"),
			tenant2Expected: []string{"default-excluded"},
		},
		{
			name:            "SpanMetricsEnableInstanceLabel",
			tenant1Got:      t1EnableInstanceLabel,
			tenant1Expected: true,
			tenant2Got:      t2EnableInstanceLabel,
			tenant2Expected: false, // default sets *bool to false
		},
		{
			name:            "SpanMetricsEnableInstanceLabel_isSet",
			tenant1Got:      t1EnableInstanceLabelIsSet,
			tenant1Expected: true,
			tenant2Got:      t2EnableInstanceLabelIsSet,
			tenant2Expected: true,
		},
		{
			name:            "SpanMetricsSpanMultiplierKey",
			tenant1Got:      overrides.MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey("tenant1"),
			tenant1Expected: "tenant1-sm-multiplier",
			tenant2Got:      overrides.MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey("tenant2"),
			tenant2Expected: "default-sm-multiplier",
		},

		// =====================================================================
		// MetricsGenerator Processor - LocalBlocks (wildcard does NOT set any)
		// =====================================================================
		{
			name:            "LocalBlocksMaxLiveTraces",
			tenant1Got:      overrides.MetricsGeneratorProcessorLocalBlocksMaxLiveTraces("tenant1"),
			tenant1Expected: uint64(2200),
			tenant2Got:      overrides.MetricsGeneratorProcessorLocalBlocksMaxLiveTraces("tenant2"),
			tenant2Expected: uint64(22000),
		},
		{
			name:            "LocalBlocksMaxBlockDuration",
			tenant1Got:      overrides.MetricsGeneratorProcessorLocalBlocksMaxBlockDuration("tenant1"),
			tenant1Expected: 230 * time.Minute,
			tenant2Got:      overrides.MetricsGeneratorProcessorLocalBlocksMaxBlockDuration("tenant2"),
			tenant2Expected: 23 * time.Minute,
		},
		{
			name:            "LocalBlocksMaxBlockBytes",
			tenant1Got:      overrides.MetricsGeneratorProcessorLocalBlocksMaxBlockBytes("tenant1"),
			tenant1Expected: uint64(2400),
			tenant2Got:      overrides.MetricsGeneratorProcessorLocalBlocksMaxBlockBytes("tenant2"),
			tenant2Expected: uint64(24000),
		},
		{
			name:            "LocalBlocksFlushCheckPeriod",
			tenant1Got:      overrides.MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod("tenant1"),
			tenant1Expected: 250 * time.Second,
			tenant2Got:      overrides.MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod("tenant2"),
			tenant2Expected: 25 * time.Second,
		},
		{
			name:            "LocalBlocksTraceIdlePeriod",
			tenant1Got:      overrides.MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod("tenant1"),
			tenant1Expected: 260 * time.Second,
			tenant2Got:      overrides.MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod("tenant2"),
			tenant2Expected: 26 * time.Second,
		},
		{
			name:            "LocalBlocksCompleteBlockTimeout",
			tenant1Got:      overrides.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout("tenant1"),
			tenant1Expected: 270 * time.Second,
			tenant2Got:      overrides.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout("tenant2"),
			tenant2Expected: 27 * time.Second,
		},

		// =====================================================================
		// MetricsGenerator Processor - HostInfo (wildcard does NOT set any)
		// =====================================================================
		{
			name:            "HostInfoHostIdentifiers",
			tenant1Got:      overrides.MetricsGeneratorProcessorHostInfoHostIdentifiers("tenant1"),
			tenant1Expected: []string{"tenant1-host-id"},
			tenant2Got:      overrides.MetricsGeneratorProcessorHostInfoHostIdentifiers("tenant2"),
			tenant2Expected: []string{"default-host-id"},
		},
		{
			name:            "HostInfoMetricName",
			tenant1Got:      overrides.MetricsGeneratorProcessorHostInfoMetricName("tenant1"),
			tenant1Expected: "tenant1_host_metric",
			tenant2Got:      overrides.MetricsGeneratorProcessorHostInfoMetricName("tenant2"),
			tenant2Expected: "default_host_metric",
		},

		// =====================================================================
		// CostAttribution overrides (wildcard does NOT set these)
		// =====================================================================
		{
			name:            "CostAttributionMaxCardinality",
			tenant1Got:      overrides.CostAttributionMaxCardinality("tenant1"),
			tenant1Expected: uint64(2000),
			tenant2Got:      overrides.CostAttributionMaxCardinality("tenant2"),
			tenant2Expected: uint64(20000),
		},
		{
			name:            "CostAttributionDimensions",
			tenant1Got:      overrides.CostAttributionDimensions("tenant1"),
			tenant1Expected: map[string]string{"tenant1-dim": "t1-val"},
			tenant2Got:      overrides.CostAttributionDimensions("tenant2"),
			tenant2Expected: map[string]string{"default-dim": "default-val"},
		},

		// =====================================================================
		// Storage overrides (wildcard does NOT set these)
		// =====================================================================
		{
			name:            "DedicatedColumns",
			tenant1Got:      overrides.DedicatedColumns("tenant1"),
			tenant1Expected: backend.DedicatedColumns{{Scope: "span", Name: "tenant1-col", Type: "int", Options: backend.DedicatedColumnOptions{}}}, // YAML round-trip converts nil Options to empty
			tenant2Got:      overrides.DedicatedColumns("tenant2"),
			tenant2Expected: backend.DedicatedColumns{{Scope: "resource", Name: "default-col", Type: "string"}}, // from raw defaults struct, nil Options
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// tenant1: should always get per-tenant value
			assert.Equal(t, tc.tenant1Expected, tc.tenant1Got, "tenant1 (per-tenant override)")
			// tenant2: should get wildcard value if set, otherwise default
			assert.Equal(t, tc.tenant2Expected, tc.tenant2Got,
				"tenant2 (wildcard fallback to default) - this failure means the override accessor does not fall back to defaultLimits when the wildcard override has a zero value")
		})
	}
}

func toYamlBytes(t *testing.T, perTenantOverrides *perTenantOverrides) []byte {
	buff, err := yaml.Marshal(perTenantOverrides)
	require.NoError(t, err)
	return buff
}

type mockValidator struct {
	f func(*Overrides) error
}

func (m mockValidator) Validate(config *Overrides) (warnings []error, err error) {
	if m.f != nil {
		return nil, m.f(config)
	}
	return nil, nil
}
