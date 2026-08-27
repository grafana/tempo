package drain

import (
	"iter"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type deletedClusterQueue struct {
	mtx sync.Mutex
	// Avoid taking mtx on every Train when no cache callback has run.
	pending atomic.Bool
	ids     []int
}

type logClusterCache struct {
	cache   *otter.Cache[int, *LogCluster]
	deleted *deletedClusterQueue
}

func newLogClusterCache(maxAge time.Duration, maxSize int, evictions prometheus.Counter, expired prometheus.Counter) *logClusterCache {
	// Otter retains this callback through its runtime cleanup argument. Keep the
	// callback state separate so it cannot reach the public cache and prevent cleanup.
	deleted := &deletedClusterQueue{}
	c := &logClusterCache{deleted: deleted}
	c.cache = otter.Must(&otter.Options[int, *LogCluster]{
		MaximumSize:      maxSize,
		ExpiryCalculator: otter.ExpiryAccessing[int, *LogCluster](maxAge),
		OnAtomicDeletion: func(e otter.DeletionEvent[int, *LogCluster]) {
			switch e.Cause {
			case otter.CauseOverflow:
				evictions.Inc()
			case otter.CauseExpiration:
				expired.Inc()
			}
			// Replacement keeps the same cluster ID active and must not
			// invalidate its index entry.
			if e.Cause != otter.CauseReplacement {
				deleted.record(e.Key)
			}
		},
	})
	return c
}

func (q *deletedClusterQueue) record(id int) {
	q.mtx.Lock()
	q.ids = append(q.ids, id)
	q.pending.Store(true)
	q.mtx.Unlock()
}

func (q *deletedClusterQueue) take() []int {
	if !q.pending.Load() {
		return nil
	}
	q.mtx.Lock()
	deleted := q.ids
	q.ids = nil
	q.pending.Store(false)
	q.mtx.Unlock()
	return deleted
}

func (c *logClusterCache) TakeDeleted() []int {
	return c.deleted.take()
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
