package distributor

import (
	"testing"

	"github.com/cortexproject/cortex/pkg/util/limiter"
	"github.com/grafana/tempo/pkg/util/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	bytesInMB = 1048576
)

func TestIngestionRateStrategy(t *testing.T) {
	tests := map[string]struct {
		limits        validation.Limits
		ring          ReadLifecycler
		expectedLimit float64
		expectedBurst int
	}{
		"local rate limiter should just return configured limits": {
			limits: validation.Limits{
				IngestionRateStrategy: validation.LocalIngestionRateStrategy,
				IngestionRate:         5,
				IngestionMaxBatchSize: 2,
			},
			ring:          nil,
			expectedLimit: 5,
			expectedBurst: 2,
		},
		"global rate limiter should share the limit across the number of distributors": {
			limits: validation.Limits{
				IngestionRateStrategy: validation.GlobalIngestionRateStrategy,
				IngestionRate:         5,
				IngestionMaxBatchSize: 2,
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
			overrides, err := validation.NewOverrides(testData.limits, nil)
			require.NoError(t, err)

			// Instance the strategy
			switch testData.limits.IngestionRateStrategy {
			case validation.LocalIngestionRateStrategy:
				strategy = newLocalIngestionRateStrategy(overrides)
			case validation.GlobalIngestionRateStrategy:
				strategy = newGlobalIngestionRateStrategy(overrides, testData.ring)
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
