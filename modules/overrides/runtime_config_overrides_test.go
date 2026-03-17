package overrides

import (
	"bytes"
	"context"
	"errors"
	"flag"
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

	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/grafana/tempo/tempodb/backend"
)

func TestRuntimeConfigOverrides_loadPerTenantOverrides(t *testing.T) {
	validator := &mockValidator{}

	// Use RegisterFlagsAndApplyDefaults to ensure all pointer fields are initialized,
	// matching the production code path and preventing nil pointers in merged results.
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	loader := loadPerTenantOverrides(validator, ConfigTypeNew, false, &cfg.Defaults)

	perTenantOverrides := perTenantOverrides{
		TenantLimits: TenantOverrides{
			"foo": {Ingestion: IngestionOverrides{TenantShardSize: ptrTo(6)}},
			"bar": {Ingestion: IngestionOverrides{TenantShardSize: ptrTo(1)}},
			"bzz": {Ingestion: IngestionOverrides{TenantShardSize: ptrTo(3)}},
		},
	}
	overridesBytes, err := yaml.Marshal(&perTenantOverrides)
	assert.NoError(t, err)

	// load overrides - validator should pass
	_, err = loader(bytes.NewReader(overridesBytes))
	assert.NoError(t, err)

	// load overrides - validator should reject bar
	validator.f = func(overrides *Overrides) error {
		if *overrides.Ingestion.TenantShardSize == 1 {
			return errors.New("no")
		}
		return nil
	}

	_, err = loader(bytes.NewReader(overridesBytes))
	assert.ErrorContains(t, err, "validating overrides for bar failed: no")
}

func TestRuntimeConfigOverrides(t *testing.T) {
	// Build defaults from RegisterFlagsAndApplyDefaults and override the fields under test.
	baseCfg := Config{}
	baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	defaultLimits := baseCfg.Defaults
	defaultLimits.Ingestion.MaxGlobalTracesPerUser = ptrTo(1)
	defaultLimits.Ingestion.MaxLocalTracesPerUser = ptrTo(2)
	defaultLimits.Ingestion.BurstSizeBytes = ptrTo(4)
	defaultLimits.Ingestion.RateLimitBytes = ptrTo(5)
	defaultLimits.Global.MaxBytesPerTrace = ptrTo(3)

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
			name:                        "limits only",
			defaultLimits:               defaultLimits,
			expectedMaxGlobalTraces:     map[string]int{"user1": 1, "user2": 1},
			expectedMaxLocalTraces:      map[string]int{"user1": 2, "user2": 2},
			expectedMaxBytesPerTrace:    map[string]int{"user1": 3, "user2": 3},
			expectedIngestionBurstSpans: map[string]int{"user1": 4, "user2": 4},
			expectedIngestionRateSpans:  map[string]int{"user1": 5, "user2": 5},
			expectedMaxSearchDuration:   map[string]int{"user1": 0, "user2": 0},
		},
		{
			name:          "basic Overrides",
			defaultLimits: defaultLimits,
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: TenantOverrides{
					"user1": {
						Ingestion: IngestionOverrides{
							MaxGlobalTracesPerUser: ptrTo(6),
							MaxLocalTracesPerUser:  ptrTo(7),
							BurstSizeBytes:         ptrTo(9),
							RateLimitBytes:         ptrTo(10),
						},
						Global: GlobalOverrides{
							MaxBytesPerTrace: ptrTo(8),
						},
						Read: ReadOverrides{
							MaxSearchDuration: ptrTo(model.Duration(11 * time.Second)),
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
			name:          "wildcard override",
			defaultLimits: defaultLimits,
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: TenantOverrides{
					"user1": {
						Ingestion: IngestionOverrides{
							MaxGlobalTracesPerUser: ptrTo(6),
							MaxLocalTracesPerUser:  ptrTo(7),
							BurstSizeBytes:         ptrTo(9),
							RateLimitBytes:         ptrTo(10),
						},
						Global: GlobalOverrides{
							MaxBytesPerTrace: ptrTo(8),
						},
					},
					"*": {
						Ingestion: IngestionOverrides{
							MaxGlobalTracesPerUser: ptrTo(11),
							MaxLocalTracesPerUser:  ptrTo(12),
							BurstSizeBytes:         ptrTo(14),
							RateLimitBytes:         ptrTo(15),
						},
						Global: GlobalOverrides{
							MaxBytesPerTrace: ptrTo(13),
						},
						Read: ReadOverrides{
							MaxSearchDuration: ptrTo(model.Duration(16 * time.Second)),
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
			expectedMaxSearchDuration:   map[string]int{"user1": 0, "user2": int(16 * time.Second)},
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
				TenantLimits: TenantOverrides{
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
				TenantLimits: TenantOverrides{
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
					NativeHistogramBucketFactor:     ptrTo(1.5),
					NativeHistogramMaxBucketNumber:  ptrTo(uint32(20)),
					NativeHistogramMinResetDuration: ptrTo(5 * time.Minute),
				},
			},
			nativeHistogramBucketFactor:     1.5,
			nativeHistogramMaxBucketNumber:  20,
			nativeHistogramMinResetDuration: 5 * time.Minute,
		},
		{
			name: "per-tenant override",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					NativeHistogramBucketFactor:     ptrTo(1.5),
					NativeHistogramMaxBucketNumber:  ptrTo(uint32(20)),
					NativeHistogramMinResetDuration: ptrTo(5 * time.Minute),
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: TenantOverrides{
					"user1": {
						MetricsGenerator: MetricsGeneratorOverrides{
							NativeHistogramBucketFactor:     ptrTo(2.0),
							NativeHistogramMaxBucketNumber:  ptrTo(uint32(30)),
							NativeHistogramMinResetDuration: ptrTo(10 * time.Minute),
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
			name: "default disabled, tenant enables",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					MaxCardinalityPerLabel: ptrTo(uint64(0)),
				},
			},
			perTenantOverrides: &perTenantOverrides{
				TenantLimits: TenantOverrides{
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
				TenantLimits: TenantOverrides{
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
				TenantLimits: TenantOverrides{
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

func TestOverrideResolutionChain(t *testing.T) {
	t.Run("tenant does not inherit from wildcard", func(t *testing.T) {
		baseCfg := Config{}
		baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
		baseCfg.Defaults.Ingestion.MaxLocalTracesPerUser = ptrTo(100)
		baseCfg.Defaults.Global.MaxBytesPerTrace = ptrTo(500)

		pto := &perTenantOverrides{
			TenantLimits: TenantOverrides{
				"user1": {
					// Only overrides MaxBytesPerTrace; should NOT get wildcard's MaxLocalTracesPerUser or MaxSearchDuration
					Global: GlobalOverrides{MaxBytesPerTrace: ptrTo(999)},
				},
				"*": {
					Ingestion: IngestionOverrides{MaxLocalTracesPerUser: ptrTo(200)},
					Read:      ReadOverrides{MaxSearchDuration: ptrTo(model.Duration(10 * time.Second))},
				},
			},
		}

		overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, baseCfg.Defaults, toYamlBytes(t, pto))
		defer cleanup()

		// user1: per-tenant merged with defaults (NOT wildcard)
		require.Equal(t, 100, overrides.MaxLocalTracesPerUser("user1"), "should come from defaults, not wildcard's 200")
		require.Equal(t, 999, overrides.MaxBytesPerTrace("user1"), "should come from per-tenant override")
		require.Equal(t, time.Duration(0), overrides.MaxSearchDuration("user1"), "should come from defaults, not wildcard's 10s")

		// user2: no per-tenant entry falls to wildcard (merged with defaults)
		require.Equal(t, 200, overrides.MaxLocalTracesPerUser("user2"), "should come from wildcard")
		require.Equal(t, 500, overrides.MaxBytesPerTrace("user2"), "should come from defaults (wildcard doesn't set it)")
		require.Equal(t, 10*time.Second, overrides.MaxSearchDuration("user2"), "should come from wildcard")
	})

	t.Run("pointer to zero overrides non-zero default", func(t *testing.T) {
		baseCfg := Config{}
		baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
		// Defaults set by RegisterFlagsAndApplyDefaults: MaxLocalTracesPerUser=10000, MaxBytesPerTrace=5000000

		pto := &perTenantOverrides{
			TenantLimits: TenantOverrides{
				"user1": {
					Ingestion: IngestionOverrides{MaxLocalTracesPerUser: ptrTo(0)},
					Global:    GlobalOverrides{MaxBytesPerTrace: ptrTo(0)},
				},
			},
		}

		overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, baseCfg.Defaults, toYamlBytes(t, pto))
		defer cleanup()

		// user1: ptrTo(0) should win over default
		require.Equal(t, 0, overrides.MaxLocalTracesPerUser("user1"), "ptrTo(0) must override non-zero default")
		require.Equal(t, 0, overrides.MaxBytesPerTrace("user1"), "ptrTo(0) must override non-zero default")

		// user2: no override, gets production defaults
		require.Equal(t, 10_000, overrides.MaxLocalTracesPerUser("user2"))
		require.Equal(t, 5_000_000, overrides.MaxBytesPerTrace("user2"))
	})
}

func TestAllAccessorsNilSafe(t *testing.T) {
	baseCfg := Config{}
	baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	// Minimal per-tenant override: only one field set
	pto := &perTenantOverrides{
		TenantLimits: TenantOverrides{
			"tenant1": {
				Global: GlobalOverrides{MaxBytesPerTrace: ptrTo(1)},
			},
			"*": {
				Ingestion: IngestionOverrides{RateLimitBytes: ptrTo(1)},
			},
		},
	}

	overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, baseCfg.Defaults, toYamlBytes(t, pto))
	defer cleanup()

	// per-tenant path
	t.Run("per-tenant", func(t *testing.T) {
		require.NotPanics(t, func() { callAllAccessors(overrides, "tenant1") })
	})

	// wildcard path
	t.Run("wildcard", func(t *testing.T) {
		require.NotPanics(t, func() { callAllAccessors(overrides, "unknown") })
	})

	// Also test without any runtime config (defaults-only path)
	defaultsOnly, cleanupDefaults := createAndInitializeRuntimeOverridesManager(t, baseCfg.Defaults, nil)
	defer cleanupDefaults()
	t.Run("defaults only", func(t *testing.T) {
		require.NotPanics(t, func() { callAllAccessors(defaultsOnly, "any-user") })
	})
}

func TestCollectMetrics(t *testing.T) {
	t.Run("tempo_limits_defaults", func(t *testing.T) {
		baseCfg := Config{}
		baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

		// Config.Collect emits tempo_limits_defaults (static defaults, no tenant label)
		ch := make(chan prometheus.Metric, 100)
		require.NotPanics(t, func() { baseCfg.Collect(ch) })
		close(ch)

		var metrics []prometheus.Metric
		for m := range ch {
			metrics = append(metrics, m)
		}
		// 12 default metrics: max_local_traces, max_global_traces, ingestion_rate_limit_bytes,
		// ingestion_burst_size_bytes, max_bytes_per_tag_values_query, max_blocks_per_tag_values_query,
		// max_bytes_per_trace, block_retention, compaction_window, compaction_disabled,
		// metrics_generator_max_active_series, metrics_generator_dry_run_enabled
		require.Len(t, metrics, 12, "should emit all default limit metrics")
	})

	t.Run("tempo_limits_overrides with per-tenant", func(t *testing.T) {
		baseCfg := Config{}
		baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

		pto := &perTenantOverrides{
			TenantLimits: TenantOverrides{
				"test-tenant": {
					Ingestion: IngestionOverrides{
						MaxLocalTracesPerUser: ptrTo(5000),
						RateLimitBytes:        ptrTo(100),
					},
					Compaction: CompactionOverrides{
						CompactionDisabled: ptrTo(true),
					},
				},
			},
		}

		reg := prometheus.NewRegistry()
		prometheus.DefaultRegisterer = reg

		overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, baseCfg.Defaults, toYamlBytes(t, pto))
		defer cleanup()

		// runtimeConfigOverridesManager.Collect emits tempo_limits_overrides (per-tenant merged values)
		ch := make(chan prometheus.Metric, 100)
		require.NotPanics(t, func() { overrides.Collect(ch) })
		close(ch)

		var metrics []prometheus.Metric
		for m := range ch {
			metrics = append(metrics, m)
		}
		// 10 metrics per tenant: max_local_traces, max_global_traces, ingestion_rate_limit_bytes,
		// ingestion_burst_size_bytes, max_bytes_per_trace, block_retention, compaction_window,
		// compaction_disabled, metrics_generator_max_active_series, metrics_generator_dry_run_enabled
		require.Len(t, metrics, 10, "should emit 10 override metrics for one tenant")
	})

	t.Run("tempo_limits_overrides without per-tenant", func(t *testing.T) {
		baseCfg := Config{}
		baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

		reg := prometheus.NewRegistry()
		prometheus.DefaultRegisterer = reg

		overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, baseCfg.Defaults, nil)
		defer cleanup()

		ch := make(chan prometheus.Metric, 100)
		require.NotPanics(t, func() { overrides.Collect(ch) })
		close(ch)

		var metrics []prometheus.Metric
		for m := range ch {
			metrics = append(metrics, m)
		}
		require.Empty(t, metrics, "should not emit override metrics when no per-tenant overrides exist")
	})
}

func TestWriteStatusRuntimeConfigDiffMode(t *testing.T) {
	baseCfg := Config{}
	baseCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	pto := &perTenantOverrides{
		TenantLimits: TenantOverrides{
			"test-tenant": {
				Ingestion: IngestionOverrides{
					MaxLocalTracesPerUser: ptrTo(5000),
				},
			},
		},
	}

	overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, baseCfg.Defaults, toYamlBytes(t, pto))
	defer cleanup()

	var buf bytes.Buffer
	req := httptest.NewRequest("GET", "/?mode=diff", nil)
	err := overrides.WriteStatusRuntimeConfig(&buf, req)
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "test-tenant", "diff output should contain tenant ID")
}

func TestDefaultOverridesValidation(t *testing.T) {
	t.Run("valid defaults pass validation", func(t *testing.T) {
		cfg := Config{}
		cfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

		validatorCalled := false
		validator := &mockValidator{f: func(config *Overrides) error {
			validatorCalled = true
			require.Equal(t, cfg.Defaults.Ingestion.RateLimitBytes, config.Ingestion.RateLimitBytes)
			require.Equal(t, cfg.Defaults.Global.MaxBytesPerTrace, config.Global.MaxBytesPerTrace)
			return nil
		}}

		overrides, err := newRuntimeConfigOverrides(cfg, validator, prometheus.NewRegistry())
		require.NoError(t, err)
		require.NotNil(t, overrides)
		require.True(t, validatorCalled, "validator should have been called with defaults")
	})

	t.Run("invalid defaults fail startup", func(t *testing.T) {
		cfg := Config{}
		cfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

		validatorCalled := false
		validator := &mockValidator{f: func(config *Overrides) error {
			validatorCalled = true
			return errors.New("invalid default config")
		}}

		_, err := newRuntimeConfigOverrides(cfg, validator, prometheus.NewRegistry())
		require.True(t, validatorCalled, "validator should have been called with defaults")
		require.ErrorContains(t, err, "validating default overrides failed")
		require.ErrorContains(t, err, "invalid default config")
	})

	t.Run("nil validator skips validation", func(t *testing.T) {
		cfg := Config{}
		cfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

		overrides, err := newRuntimeConfigOverrides(cfg, nil, prometheus.NewRegistry())
		require.NoError(t, err)
		require.NotNil(t, overrides)
	})
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

// callAllAccessors calls every method on the Interface for the given userID.
// Used as a smoke test to verify no accessor panics on nil pointer dereference.
func callAllAccessors(o Service, userID string) {
	o.GetRuntimeOverridesFor(userID)
	o.GetTenantIDs()

	o.IngestionRateStrategy()
	o.MaxLocalTracesPerUser(userID)
	o.MaxGlobalTracesPerUser(userID)
	o.MaxBytesPerTrace(userID)
	o.IngestionArtificialDelay(userID)
	o.IngestionRetryInfoEnabled(userID)
	o.MaxCompactionRange(userID)
	o.Forwarders(userID)
	o.MaxBytesPerTagValuesQuery(userID)
	o.MaxBlocksPerTagValuesQuery(userID)
	o.IngestionRateLimitBytes(userID)
	o.IngestionBurstSizeBytes(userID)
	o.IngestionTenantShardSize(userID)
	o.IngestionMaxAttributeBytes(userID)
	o.MetricsGeneratorIngestionSlack(userID)
	o.MetricsGeneratorRingSize(userID)
	o.MetricsGeneratorProcessors(userID)
	o.MetricsGeneratorMaxActiveSeries(userID)
	o.MetricsGeneratorMaxActiveEntities(userID)
	o.MetricsGeneratorCollectionInterval(userID)
	o.MetricsGeneratorDisableCollection(userID)
	o.MetricsGeneratorGenerateNativeHistograms(userID)
	o.MetricsGeneratorTraceIDLabelName(userID)
	o.MetricsGeneratorRemoteWriteHeaders(userID)
	o.MetricsGeneratorForwarderQueueSize(userID)
	o.MetricsGeneratorForwarderWorkers(userID)
	o.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID)
	o.MetricsGeneratorProcessorServiceGraphsDimensions(userID)
	o.MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID)
	o.MetricsGeneratorProcessorServiceGraphsFilterPolicies(userID)
	o.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID)
	o.MetricsGeneratorProcessorSpanMetricsDimensions(userID)
	o.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID)
	o.MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID)
	o.MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID)
	o.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID)
	o.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID)
	o.MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram(userID)
	o.MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel(userID)
	o.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID)
	o.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(userID)
	o.MetricsGeneratorProcessorHostInfoHostIdentifiers(userID)
	o.MetricsGeneratorProcessorHostInfoMetricName(userID)
	o.MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey(userID)
	o.MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey(userID)
	o.MetricsGeneratorNativeHistogramBucketFactor(userID)
	o.MetricsGeneratorNativeHistogramMaxBucketNumber(userID)
	o.MetricsGeneratorNativeHistogramMinResetDuration(userID)
	o.MetricsGeneratorSpanNameSanitization(userID)
	o.MetricsGeneratorMaxCardinalityPerLabel(userID)
	o.BlockRetention(userID)
	o.CompactionDisabled(userID)
	o.MaxSearchDuration(userID)
	o.MaxMetricsDuration(userID)
	o.DedicatedColumns(userID)
	o.UnsafeQueryHints(userID)
	o.LeftPadTraceIDs(userID)
	o.CostAttributionMaxCardinality(userID)
	o.CostAttributionDimensions(userID)
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
