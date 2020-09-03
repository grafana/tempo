package validation

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestOverrides(t *testing.T) {

	tests := []struct {
		name                        string
		limits                      Limits
		overrides                   *OverridesConfig
		expectedMaxLocalTraces      map[string]int
		expectedMaxGlobalTraces     map[string]int
		expectedIngestionRateSpans  map[string]int
		expectedIngestionBurstSpans map[string]int
	}{
		{
			name: "limits only",
			limits: Limits{
				MaxGlobalTracesPerUser: 1,
				MaxLocalTracesPerUser:  2,
				IngestionMaxBatchSize:  3,
				IngestionRateSpans:     4,
			},
			expectedMaxGlobalTraces:     map[string]int{"user1": 1, "user2": 1},
			expectedMaxLocalTraces:      map[string]int{"user1": 2, "user2": 2},
			expectedIngestionBurstSpans: map[string]int{"user1": 3, "user2": 3},
			expectedIngestionRateSpans:  map[string]int{"user1": 4, "user2": 4},
		},
		{
			name: "basic override",
			limits: Limits{
				MaxGlobalTracesPerUser: 1,
				MaxLocalTracesPerUser:  2,
				IngestionMaxBatchSize:  3,
				IngestionRateSpans:     4,
			},
			overrides: &OverridesConfig{
				TenantLimits: map[string]*Limits{
					"user1": {
						MaxGlobalTracesPerUser: 5,
						MaxLocalTracesPerUser:  6,
						IngestionMaxBatchSize:  7,
						IngestionRateSpans:     8,
					},
				},
			},
			expectedMaxGlobalTraces:     map[string]int{"user1": 5, "user2": 1},
			expectedMaxLocalTraces:      map[string]int{"user1": 6, "user2": 2},
			expectedIngestionBurstSpans: map[string]int{"user1": 7, "user2": 3},
			expectedIngestionRateSpans:  map[string]int{"user1": 8, "user2": 4},
		},
	}

	for _, tt := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			if tt.overrides != nil {
				overridesFile := filepath.Join(t.TempDir(), "overrides.yaml")

				buff, err := yaml.Marshal(tt.overrides)
				require.NoError(t, err)

				err = ioutil.WriteFile(overridesFile, buff, os.ModePerm)
				require.NoError(t, err)

				tt.limits.PerTenantOverrideConfig = overridesFile
				tt.limits.PerTenantOverridePeriod = time.Hour
			}

			overrides, srv, err := NewOverrides(tt.limits)
			require.NoError(t, err)
			if srv != nil {
				err = services.StartAndAwaitRunning(context.TODO(), srv)
				require.NoError(t, err)
			}

			for user, expectedVal := range tt.expectedMaxLocalTraces {
				assert.Equal(t, expectedVal, overrides.MaxLocalTracesPerUser(user))
			}

			for user, expectedVal := range tt.expectedMaxGlobalTraces {
				assert.Equal(t, expectedVal, overrides.MaxGlobalTracesPerUser(user))
			}

			for user, expectedVal := range tt.expectedIngestionBurstSpans {
				assert.Equal(t, expectedVal, overrides.IngestionMaxBatchSize(user))
			}

			for user, expectedVal := range tt.expectedIngestionRateSpans {
				assert.Equal(t, float64(expectedVal), overrides.IngestionRateSpans(user))
			}

			if srv != nil {
				err = services.StopAndAwaitTerminated(context.TODO(), srv)
				require.NoError(t, err)
			}
		})
	}
}
