package overrides

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/tempo/tempodb/backend"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestOverrides(t *testing.T) {

	tests := []struct {
		name                        string
		limits                      Limits
		overrides                   *perTenantOverrides
		expectedMaxLocalTraces      map[string]int
		expectedMaxGlobalTraces     map[string]int
		expectedMaxBytesPerTrace    map[string]int
		expectedIngestionRateSpans  map[string]int
		expectedIngestionBurstSpans map[string]int
		expectedMaxSearchDuration   map[string]int
	}{
		{
			name: "limits only",
			limits: Limits{
				MaxGlobalTracesPerUser:  1,
				MaxLocalTracesPerUser:   2,
				MaxBytesPerTrace:        3,
				IngestionBurstSizeBytes: 4,
				IngestionRateLimitBytes: 5,
			},
			expectedMaxGlobalTraces:     map[string]int{"user1": 1, "user2": 1},
			expectedMaxLocalTraces:      map[string]int{"user1": 2, "user2": 2},
			expectedMaxBytesPerTrace:    map[string]int{"user1": 3, "user2": 3},
			expectedIngestionBurstSpans: map[string]int{"user1": 4, "user2": 4},
			expectedIngestionRateSpans:  map[string]int{"user1": 5, "user2": 5},
			expectedMaxSearchDuration:   map[string]int{"user1": 0, "user2": 0},
		},
		{
			name: "basic overrides",
			limits: Limits{
				MaxGlobalTracesPerUser:  1,
				MaxLocalTracesPerUser:   2,
				MaxBytesPerTrace:        3,
				IngestionBurstSizeBytes: 4,
				IngestionRateLimitBytes: 5,
			},
			overrides: &perTenantOverrides{
				TenantLimits: map[string]*Limits{
					"user1": {
						MaxGlobalTracesPerUser:  6,
						MaxLocalTracesPerUser:   7,
						MaxBytesPerTrace:        8,
						IngestionBurstSizeBytes: 9,
						IngestionRateLimitBytes: 10,
						MaxSearchDuration:       model.Duration(11 * time.Second),
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
			limits: Limits{
				MaxGlobalTracesPerUser:  1,
				MaxLocalTracesPerUser:   2,
				MaxBytesPerTrace:        3,
				IngestionBurstSizeBytes: 4,
				IngestionRateLimitBytes: 5,
			},
			overrides: &perTenantOverrides{
				TenantLimits: map[string]*Limits{
					"user1": {
						MaxGlobalTracesPerUser:  6,
						MaxLocalTracesPerUser:   7,
						MaxBytesPerTrace:        8,
						IngestionBurstSizeBytes: 9,
						IngestionRateLimitBytes: 10,
					},
					"*": {
						MaxGlobalTracesPerUser:  11,
						MaxLocalTracesPerUser:   12,
						MaxBytesPerTrace:        13,
						IngestionBurstSizeBytes: 14,
						IngestionRateLimitBytes: 15,
						MaxSearchDuration:       model.Duration(16 * time.Second),
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
			if tt.overrides != nil {
				overridesFile := filepath.Join(t.TempDir(), "overrides.yaml")

				buff, err := yaml.Marshal(tt.overrides)
				require.NoError(t, err)

				err = os.WriteFile(overridesFile, buff, os.ModePerm)
				require.NoError(t, err)

				tt.limits.PerTenantOverrideConfig = overridesFile
				tt.limits.PerTenantOverridePeriod = model.Duration(time.Hour)
			}

			prometheus.DefaultRegisterer = prometheus.NewRegistry() // have to overwrite the registry or test panics with multiple metric reg
			overrides, err := NewOverrides(tt.limits)
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

			//if srv != nil {
			err = services.StopAndAwaitTerminated(context.TODO(), overrides)
			require.NoError(t, err)
			//}
		})
	}
}

func TestMetricsGeneratorOverrides(t *testing.T) {

	tests := []struct {
		name                      string
		limits                    Limits
		overrides                 *perTenantOverrides
		expectedEnableTargetInfo  map[string]bool
		expectedDimensionMappings map[string][]sharedconfig.DimensionMappings
	}{
		{
			name: "limits only",
			limits: Limits{
				MetricsGeneratorProcessorSpanMetricsEnableTargetInfo: true,
				MetricsGeneratorProcessorSpanMetricsDimensionMappings: []sharedconfig.DimensionMappings{
					{
						Name:        "test-name",
						SourceLabel: []string{"service.name"},
						Join:        "/",
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
			name:   "basic overrides",
			limits: Limits{},
			overrides: &perTenantOverrides{
				TenantLimits: map[string]*Limits{
					"user1": {
						MetricsGeneratorProcessorSpanMetricsEnableTargetInfo: true,
						MetricsGeneratorProcessorSpanMetricsDimensionMappings: []sharedconfig.DimensionMappings{
							{
								Name:        "test-name",
								SourceLabel: []string{"service.name"},
								Join:        "/",
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
			limits: Limits{
				MetricsGeneratorProcessorSpanMetricsEnableTargetInfo: false,
				MetricsGeneratorProcessorSpanMetricsDimensionMappings: []sharedconfig.DimensionMappings{
					{
						Name:        "test-name",
						SourceLabel: []string{"service.name"},
						Join:        "/",
					},
				},
			},
			overrides: &perTenantOverrides{
				TenantLimits: map[string]*Limits{
					"user1": {
						MetricsGeneratorProcessorSpanMetricsEnableTargetInfo: true,
						MetricsGeneratorProcessorSpanMetricsDimensionMappings: []sharedconfig.DimensionMappings{
							{
								Name:        "another-name",
								SourceLabel: []string{"service.namespace"},
								Join:        "/",
							},
						},
					},
					"*": {
						MetricsGeneratorProcessorSpanMetricsEnableTargetInfo: false,
						MetricsGeneratorProcessorSpanMetricsDimensionMappings: []sharedconfig.DimensionMappings{
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.overrides != nil {
				overridesFile := filepath.Join(t.TempDir(), "overrides.yaml")

				buff, err := yaml.Marshal(tt.overrides)
				require.NoError(t, err)

				err = os.WriteFile(overridesFile, buff, os.ModePerm)
				require.NoError(t, err)

				tt.limits.PerTenantOverrideConfig = overridesFile
				tt.limits.PerTenantOverridePeriod = model.Duration(time.Hour)
			}

			prometheus.DefaultRegisterer = prometheus.NewRegistry() // have to overwrite the registry or test panics with multiple metric reg
			overrides, err := NewOverrides(tt.limits)
			require.NoError(t, err)
			err = services.StartAndAwaitRunning(context.TODO(), overrides)
			require.NoError(t, err)

			for user, expectedVal := range tt.expectedEnableTargetInfo {
				assert.Equal(t, expectedVal, overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(user))
			}

			for user, expectedVal := range tt.expectedDimensionMappings {
				assert.Equal(t, expectedVal, overrides.MetricsGeneratorProcessorSpanMetricsDimensionMappings(user))
			}

			//if srv != nil {
			err = services.StopAndAwaitTerminated(context.TODO(), overrides)
			require.NoError(t, err)
			//}
		})
	}
}

func TestTempoDBOverrides(t *testing.T) {

	tests := []struct {
		name                     string
		limits                   Limits
		overrides                *perTenantOverrides
		expectedDedicatedColumns map[string][]backend.DedicatedColumn
	}{
		{
			name: "limits",
			limits: Limits{
				DedicatedColumns: []backend.DedicatedColumn{
					{Scope: "resource", Name: "namespace", Type: "string"},
				},
			},
			expectedDedicatedColumns: map[string][]backend.DedicatedColumn{
				"user1": {{Scope: "resource", Name: "namespace", Type: "string"}},
				"user2": {{Scope: "resource", Name: "namespace", Type: "string"}},
			},
		},
		{
			name: "basic overrides",
			limits: Limits{
				DedicatedColumns: []backend.DedicatedColumn{
					{Scope: "resource", Name: "namespace", Type: "string"},
				},
			},
			overrides: &perTenantOverrides{TenantLimits: map[string]*Limits{
				"user2": {
					DedicatedColumns: []backend.DedicatedColumn{{Scope: "span", Name: "http.status", Type: "int"}},
				},
			}},
			expectedDedicatedColumns: map[string][]backend.DedicatedColumn{
				"user1": {{Scope: "resource", Name: "namespace", Type: "string"}},
				"user2": {{Scope: "span", Name: "http.status", Type: "int"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.overrides != nil {
				overridesFile := filepath.Join(t.TempDir(), "overrides.yaml")

				buff, err := yaml.Marshal(tc.overrides)
				require.NoError(t, err)

				err = os.WriteFile(overridesFile, buff, os.ModePerm)
				require.NoError(t, err)

				tc.limits.PerTenantOverrideConfig = overridesFile
				tc.limits.PerTenantOverridePeriod = model.Duration(time.Hour)
			}

			prometheus.DefaultRegisterer = prometheus.NewRegistry() // have to overwrite the registry or test panics with multiple metric reg
			overrides, err := NewOverrides(tc.limits)
			require.NoError(t, err)
			err = services.StartAndAwaitRunning(context.TODO(), overrides)
			require.NoError(t, err)

			for user, expected := range tc.expectedDedicatedColumns {
				assert.Equal(t, expected, overrides.DedicatedColumns(user))
			}

			err = services.StopAndAwaitTerminated(context.TODO(), overrides)
			require.NoError(t, err)
		})
	}
}
