package drain

import (
	"iter"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type logClusterCache struct {
	cache *otter.Cache[int, *LogCluster]
}

func newLogClusterCache(maxAge time.Duration, maxSize int, evictions prometheus.Counter, expired prometheus.Counter) *logClusterCache {
	return &logClusterCache{
		cache: otter.Must(&otter.Options[int, *LogCluster]{
			MaximumSize:      maxSize,
			ExpiryCalculator: otter.ExpiryAccessing[int, *LogCluster](maxAge),
			OnAtomicDeletion: func(e otter.DeletionEvent[int, *LogCluster]) {
				switch e.Cause {
				case otter.CauseOverflow:
					evictions.Inc()
				case otter.CauseExpiration:
					expired.Inc()
				}
			},
		}),
	}
}

func (c *logClusterCache) Values() iter.Seq[*LogCluster] {
	return c.cache.Values()
}

// Set adds a cluster to the cache. This method may evict other clusters from the cache if it's full.
func (c *logClusterCache) Put(cluster *LogCluster) {
	c.cache.Set(cluster.id, cluster)
}

// Remove invalidates a cluster from the cache.
func (c *logClusterCache) Remove(key int) {
	c.cache.Invalidate(key)
}

// Get retrieves a cluster from the cache and updates the access time to prevent ttl-based eviction.
func (c *logClusterCache) Get(key int) *LogCluster {
	cluster, ok := c.cache.GetIfPresent(key)
	if !ok {
		return nil
	}
	return cluster
}

// GetQuietly retrieves a cluster from the cache without updating the access time.
func (c *logClusterCache) GetQuietly(key int) *LogCluster {
	entry, ok := c.cache.GetEntryQuietly(key)
	if !ok {
		return nil
	}
	return entry.Value
}

// NotExists checks if a cluster does not exist in the cache without updating the access time.
func (c *logClusterCache) NotExists(key int) bool {
	_, ok := c.cache.GetEntryQuietly(key)
	return !ok
}
