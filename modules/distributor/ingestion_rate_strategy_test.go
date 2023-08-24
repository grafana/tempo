package distributor

import (
	"testing"

	"github.com/grafana/dskit/limiter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
)

func TestIngestionRateStrategy(t *testing.T) {
	tests := map[string]struct {
		limits        overrides.Overrides
		ring          ReadLifecycler
		expectedLimit float64
		expectedBurst int
	}{
		"local rate limiter should just return configured limits": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.LocalIngestionRateStrategy,
					RateLimitBytes: 5,
					BurstSizeBytes: 2,
				},
			},
			ring:          nil,
			expectedLimit: 5,
			expectedBurst: 2,
		},
		"global rate limiter should share the limit across the number of distributors": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 5,
					BurstSizeBytes: 2,
				},
			},
			ring: func() ReadLifecycler {
				ring := newReadLifecyclerMock()
				ring.On("HealthyInstancesCount").Return(2)
				return ring
			}(),
			expectedLimit: 2.5,
			expectedBurst: 2,
		},
	}

	for testName, testData := range tests {
		testData := testData

		t.Run(testName, func(t *testing.T) {
			var strategy limiter.RateLimiterStrategy

			// Init limits overrides
			o, err := overrides.NewOverrides(overrides.Config{
				Defaults: testData.limits,
			})
			require.NoError(t, err)

			// Instance the strategy
			switch testData.limits.Ingestion.RateStrategy {
			case overrides.LocalIngestionRateStrategy:
				strategy = newLocalIngestionRateStrategy(o)
			case overrides.GlobalIngestionRateStrategy:
				strategy = newGlobalIngestionRateStrategy(o, testData.ring)
			default:
				require.Fail(t, "Unknown strategy")
			}

			assert.Equal(t, testData.expectedLimit, strategy.Limit("test"))
			assert.Equal(t, testData.expectedBurst, strategy.Burst("test"))
		})
	}
}

type readLifecyclerMock struct {
	mock.Mock
}

func newReadLifecyclerMock() *readLifecyclerMock {
	return &readLifecyclerMock{}
}

func (m *readLifecyclerMock) HealthyInstancesCount() int {
	args := m.Called()
	return args.Int(0)
}
