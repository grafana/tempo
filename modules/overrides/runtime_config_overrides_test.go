package overrides

import (
	"bytes"
	"context"
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
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/grafana/tempo/tempodb/backend"
)

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

				err = os.WriteFile(overridesFile, buff, os.ModePerm)
				require.NoError(t, err)

				cfg.PerTenantOverrideConfig = overridesFile
				cfg.PerTenantOverridePeriod = model.Duration(time.Hour)
			}

			prometheus.DefaultRegisterer = prometheus.NewRegistry() // have to overwrite the registry or test panics with multiple metric reg
			overrides, err := newRuntimeConfigOverrides(cfg, prometheus.DefaultRegisterer)
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
	}{
		{
			name: "limits only",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					Processor: ProcessorOverrides{
						SpanMetrics: SpanMetricsOverrides{
							EnableTargetInfo: true,
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
									EnableTargetInfo: true,
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
		},
		{
			name: "wildcard override",
			defaultLimits: Overrides{
				MetricsGenerator: MetricsGeneratorOverrides{
					Processor: ProcessorOverrides{
						SpanMetrics: SpanMetricsOverrides{
							EnableTargetInfo: false,
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
									EnableTargetInfo: true,
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
									EnableTargetInfo: false,
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides, cleanup := createAndInitializeRuntimeOverridesManager(t, tt.defaultLimits, toYamlBytes(t, tt.perTenantOverrides))
			defer cleanup()

			for user, expectedVal := range tt.expectedEnableTargetInfo {
				assert.Equal(t, expectedVal, overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(user))
			}

			for user, expectedVal := range tt.expectedDimensionMappings {
				assert.Equal(t, expectedVal, overrides.MetricsGeneratorProcessorSpanMetricsDimensionMappings(user))
			}

			for user, expectedVal := range tt.expectedTargetInfoExcludedDimensions {
				assert.Equal(t, expectedVal, overrides.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(user))
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

	overrides, err := newRuntimeConfigOverrides(cfg, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.TODO(), overrides))

	buff := bytes.NewBuffer(nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, overrides.WriteStatusRuntimeConfig(buff, req))

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

	require.NoError(t, os.WriteFile(overridesFile, []byte(perTenantOverrides), os.ModePerm))

	cfg.PerTenantOverrideConfig = overridesFile
	cfg.PerTenantOverridePeriod = model.Duration(time.Hour)

	overrides, err := newRuntimeConfigOverrides(cfg, prometheus.NewRegistry())
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

func createAndInitializeRuntimeOverridesManager(t *testing.T, defaultLimits Overrides, perTenantOverrides []byte) (Service, func()) {
	cfg := Config{
		Defaults: defaultLimits,
	}

	if perTenantOverrides != nil {
		overridesFile := filepath.Join(t.TempDir(), "Overrides.yaml")

		err := os.WriteFile(overridesFile, perTenantOverrides, os.ModePerm)
		require.NoError(t, err)

		cfg.PerTenantOverrideConfig = overridesFile
		cfg.PerTenantOverridePeriod = model.Duration(time.Hour)
	}

	prometheus.DefaultRegisterer = prometheus.NewRegistry() // have to overwrite the registry or test panics with multiple metric reg
	overrides, err := newRuntimeConfigOverrides(cfg, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	err = services.StartAndAwaitRunning(context.TODO(), overrides)
	require.NoError(t, err)

	return overrides, func() {
		err := services.StopAndAwaitTerminated(context.TODO(), overrides)
		require.NoError(t, err)
	}
}

func toYamlBytes(t *testing.T, perTenantOverrides *perTenantOverrides) []byte {
	buff, err := yaml.Marshal(perTenantOverrides)
	require.NoError(t, err)
	return buff
}
