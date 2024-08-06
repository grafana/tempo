package distributor

import (
	"github.com/grafana/dskit/limiter"

	"github.com/grafana/tempo/v2/modules/overrides"
)

// ReadLifecycler represents the read interface to the lifecycler.
type ReadLifecycler interface {
	HealthyInstancesCount() int
}

type localStrategy struct {
	limits overrides.Interface
}

func newLocalIngestionRateStrategy(limits overrides.Interface) limiter.RateLimiterStrategy {
	return &localStrategy{
		limits: limits,
	}
}

func (s *localStrategy) Limit(userID string) float64 {
	return s.limits.IngestionRateLimitBytes(userID)
}

func (s *localStrategy) Burst(userID string) int {
	return s.limits.IngestionBurstSizeBytes(userID)
}

type globalStrategy struct {
	limits overrides.Interface
	ring   ReadLifecycler
}

func newGlobalIngestionRateStrategy(limits overrides.Interface, ring ReadLifecycler) limiter.RateLimiterStrategy {
	return &globalStrategy{
		limits: limits,
		ring:   ring,
	}
}

func (s *globalStrategy) Limit(userID string) float64 {
	numDistributors := s.ring.HealthyInstancesCount()

	if numDistributors == 0 {
		return s.limits.IngestionRateLimitBytes(userID)
	}

	return s.limits.IngestionRateLimitBytes(userID) / float64(numDistributors)
}

func (s *globalStrategy) Burst(userID string) int {
	// The meaning of burst doesn't change for the global strategy, in order
	// to keep it easier to understand for users / operators.
	return s.limits.IngestionBurstSizeBytes(userID)
}
