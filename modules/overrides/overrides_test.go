package overrides

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
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
