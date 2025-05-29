package distributor

import (
	"testing"
	"time"

	"github.com/grafana/dskit/limiter"
	"github.com/prometheus/client_golang/prometheus"
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
			ring:          createMockRingWithHealthyInstances(2),
			expectedLimit: 2.5,
			expectedBurst: 2,
		},
		"global rate limiter should divide the limit but not the burst": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 10,
					BurstSizeBytes: 8,
				},
			},
			ring:          createMockRingWithHealthyInstances(5),
			expectedLimit: 2, // 10 ÷ 5 = 2
			expectedBurst: 8, // Burst is NOT divided
		},
		"global rate limiter with different limit and burst values": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 100,
					BurstSizeBytes: 50,
				},
			},
			ring:          createMockRingWithHealthyInstances(4),
			expectedLimit: 25, // 100 ÷ 4 = 25
			expectedBurst: 50, // Burst is NOT divided
		},
		"global rate limiter with very large burst size": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 1000,
					BurstSizeBytes: 10000, // 10x the rate limit
				},
			},
			ring:          createMockRingWithHealthyInstances(10),
			expectedLimit: 100,   // 1000 ÷ 10 = 100
			expectedBurst: 10000, // Burst will NOT be divided regardless of size
		},
		"edge case with zero burst size": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 200,
					BurstSizeBytes: 0, // Zero burst size
				},
			},
			ring:          createMockRingWithHealthyInstances(2),
			expectedLimit: 100, // 200 ÷ 2 = 100
			expectedBurst: 0,   // Zero burst size is respected
		},
	}

	for testName, testData := range tests {
		testData := testData

		t.Run(testName, func(t *testing.T) {
			var strategy limiter.RateLimiterStrategy

			// Init limits overrides
			o, err := overrides.NewOverrides(overrides.Config{
				Defaults: testData.limits,
			}, nil, prometheus.DefaultRegisterer)
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

func TestIngestionRateStrategyAllowN(t *testing.T) {
	tests := map[string]struct {
		limits           overrides.Overrides
		ring             ReadLifecycler
		tokensTryToAllow int
		expectedAllowed  bool
	}{
		"local rate limiter allows tokens within the limit": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.LocalIngestionRateStrategy,
					RateLimitBytes: 10,
					BurstSizeBytes: 5,
				},
			},
			ring:             nil,
			tokensTryToAllow: 4,
			expectedAllowed:  true,
		},
		"local rate limiter denies tokens over rate limit but under burst": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.LocalIngestionRateStrategy,
					RateLimitBytes: 10,
					BurstSizeBytes: 20,
				},
			},
			ring:             nil,
			tokensTryToAllow: 15,
			expectedAllowed:  true,
		},
		"local rate limiter denies tokens over rate limit and under burst": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.LocalIngestionRateStrategy,
					RateLimitBytes: 10,
					BurstSizeBytes: 10,
				},
			},
			ring:             nil,
			tokensTryToAllow: 15,
			expectedAllowed:  false,
		},
		"local rate limiter denies tokens over burst": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.LocalIngestionRateStrategy,
					RateLimitBytes: 10,
					BurstSizeBytes: 5,
				},
			},
			ring:             nil,
			tokensTryToAllow: 6,
			expectedAllowed:  false,
		},
		"global rate limiter allows tokens within the distributed limit": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 20,
					BurstSizeBytes: 10,
				},
			},
			ring:             createMockRingWithHealthyInstances(2),
			tokensTryToAllow: 9, // 10 is per instance limit
			expectedAllowed:  true,
		},
		"global rate limiter allows tokens under burst but over rate limit": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 20,
					BurstSizeBytes: 20,
				},
			},
			ring:             createMockRingWithHealthyInstances(2),
			tokensTryToAllow: 15, // 10 is per instance limit
			expectedAllowed:  true,
		},
		"global rate limiter denies tokens over burst and rate limit": {
			limits: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy:   overrides.GlobalIngestionRateStrategy,
					RateLimitBytes: 20,
					BurstSizeBytes: 10,
				},
			},
			ring:             createMockRingWithHealthyInstances(2),
			tokensTryToAllow: 15, // 10 is per instance limit
			expectedAllowed:  false,
		},
	}

	for testName, testData := range tests {
		testData := testData

		t.Run(testName, func(t *testing.T) {
			var strategy limiter.RateLimiterStrategy

			// Init limits overrides
			o, err := overrides.NewOverrides(overrides.Config{
				Defaults: testData.limits,
			}, nil, prometheus.DefaultRegisterer)
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

			rateLimiter := limiter.NewRateLimiter(strategy, time.Minute)

			// Test if the requested number of tokens is allowed
			allowed := rateLimiter.AllowN(time.Now(), "test", testData.tokensTryToAllow)
			require.Equal(t, testData.expectedAllowed, allowed)
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

func createMockRingWithHealthyInstances(count int) ReadLifecycler {
	ring := newReadLifecyclerMock()
	ring.On("HealthyInstancesCount").Return(count)
	return ring
}
