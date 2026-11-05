package distributor

import (
	"github.com/grafana/dskit/limiter"

	"github.com/grafana/tempo/v3/modules/overrides"
)

// ReadLifecycler represents the read interface to the lifecycler.
type ReadLifecycler interface {
	HealthyInstancesCount() int
}

// localStrategy and globalStrategy take a limits key: the tenant ID plus any tenant metadata ("tenant:key=value").
type localStrategy struct {
	limits overrides.Interface
}

func newLocalIngestionRateStrategy(limits overrides.Interface) limiter.RateLimiterStrategy {
	return &localStrategy{
		limits: limits,
	}
}

func (s *localStrategy) Limit(limitsKey string) float64 {
	return s.limits.IngestionRateLimitBytes(limitsKey)
}

func (s *localStrategy) Burst(limitsKey string) int {
	return s.limits.IngestionBurstSizeBytes(limitsKey)
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

func (s *globalStrategy) Limit(limitsKey string) float64 {
	numDistributors := s.ring.HealthyInstancesCount()

	if numDistributors == 0 {
		return s.limits.IngestionRateLimitBytes(limitsKey)
	}

	return s.limits.IngestionRateLimitBytes(limitsKey) / float64(numDistributors)
}

func (s *globalStrategy) Burst(limitsKey string) int {
	// The meaning of burst doesn't change for the global strategy, in order
	// to keep it easier to understand for users / operators.
	return s.limits.IngestionBurstSizeBytes(limitsKey)
}
