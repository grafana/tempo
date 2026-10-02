package cache

import (
	"context"
	"math/rand/v2"
)

type probabilisticCache struct {
	Cache
	probability float64
}

var _ Cache = (*probabilisticCache)(nil)

// NewProbabilistic returns a Cache that forwards each Store call with the given probability.
// Probability must be between 0 and 1, inclusive.
func NewProbabilistic(cache Cache, probability float64) Cache {
	if probability >= 1 {
		return cache
	}

	return &probabilisticCache{
		Cache:       cache,
		probability: probability,
	}
}

func (c *probabilisticCache) Store(ctx context.Context, keys []string, bufs [][]byte) {
	if rand.Float64() < c.probability { //nolint:gosec // G404: cache admission does not require cryptographic randomness.
		c.Cache.Store(ctx, keys, bufs)
	}
}
