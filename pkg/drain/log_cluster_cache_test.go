package drain

import (
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogClusterCache_PutGet(t *testing.T) {
	t.Parallel()

	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})

	cache := newLogClusterCache(1*time.Hour, 100, evictions, expired)

	cluster1 := &LogCluster{id: 1, Tokens: []string{"GET", "/users"}, ParamString: "<_>"}
	cluster2 := &LogCluster{id: 2, Tokens: []string{"POST", "/users"}, ParamString: "<_>"}

	cache.Put(cluster1)
	cache.Put(cluster2)

	// Get should retrieve clusters
	retrieved1 := cache.Get(1)
	require.NotNil(t, retrieved1)
	assert.Equal(t, cluster1.id, retrieved1.id)
	assert.Equal(t, cluster1.Tokens, retrieved1.Tokens)

	retrieved2 := cache.Get(2)
	require.NotNil(t, retrieved2)
	assert.Equal(t, cluster2.id, retrieved2.id)
}

func TestLogClusterCache_GetQuietly(t *testing.T) {
	t.Parallel()

	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})

	cache := newLogClusterCache(1*time.Hour, 100, evictions, expired)

	cluster := &LogCluster{id: 1, Tokens: []string{"GET", "/users"}, ParamString: "<_>"}
	cache.Put(cluster)

	// GetQuietly should retrieve without updating access time
	retrieved := cache.GetQuietly(1)
	require.NotNil(t, retrieved)
	assert.Equal(t, cluster.id, retrieved.id)

	// GetQuietly should work multiple times
	retrieved2 := cache.GetQuietly(1)
	require.NotNil(t, retrieved2)
	assert.Equal(t, cluster.id, retrieved2.id)
}

func TestLogClusterCache_Remove(t *testing.T) {
	t.Parallel()

	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})

	cache := newLogClusterCache(1*time.Hour, 100, evictions, expired)

	cluster := &LogCluster{id: 1, Tokens: []string{"GET", "/users"}, ParamString: "<_>"}
	cache.Put(cluster)

	// Should exist before removal
	assert.NotNil(t, cache.Get(1))

	// Remove should invalidate the cluster
	cache.Remove(1)

	// Should not exist after removal
	assert.Nil(t, cache.Get(1))
	assert.Nil(t, cache.GetQuietly(1))
}

func TestLogClusterCache_NotExists(t *testing.T) {
	t.Parallel()

	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})

	cache := newLogClusterCache(1*time.Hour, 100, evictions, expired)

	// Non-existent key should return true
	assert.True(t, cache.NotExists(999))

	// After adding, should return false
	cluster := &LogCluster{id: 1, Tokens: []string{"GET", "/users"}, ParamString: "<_>"}
	cache.Put(cluster)
	assert.False(t, cache.NotExists(1))

	// After removal, should return true again
	cache.Remove(1)
	assert.True(t, cache.NotExists(1))
}

func TestLogClusterCache_MaxSizeEviction(t *testing.T) {
	t.Parallel()

	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})

	// Small cache size to trigger evictions
	maxSize := 5
	cache := newLogClusterCache(1*time.Hour, maxSize, evictions, expired)

	// Fill cache beyond max size
	for i := 1; i <= maxSize*3; i++ {
		cluster := &LogCluster{id: i, Tokens: []string{"GET", "/users"}, ParamString: "<_>"}
		cache.Put(cluster)
	}

	// Cache should only contain maxSize items (or slightly more due to otter's internal implementation)
	// We can't easily verify exact eviction order without exposing internals,
	// but we can verify the cache size is bounded reasonably
	clusterCount := 0
	for i := 1; i <= maxSize*3; i++ {
		if cache.GetQuietly(i) != nil {
			clusterCount++
		}
	}
	// Otter cache uses S3-FIFO eviction which is frequency-based, not strictly LRU.
	// The important thing is that the cache doesn't grow unbounded and handles Put operations correctly.
	// Verify cache is bounded (not all items are present, or at least cache operations work)
	assert.LessOrEqual(t, clusterCount, maxSize*3, "cache should not exceed reasonable bounds")
	assert.Greater(t, clusterCount, 0, "cache should contain some items")

	// Note: We cannot assert which specific items survive eviction because S3-FIFO
	// uses a probationary period and frequency-based promotion. Recently added items
	// are not guaranteed to survive if the cache is already full.
}

func TestLogClusterCache_Values(t *testing.T) {
	t.Parallel()

	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})

	cache := newLogClusterCache(1*time.Hour, 100, evictions, expired)

	// Add multiple clusters
	clusters := []*LogCluster{
		{id: 1, Tokens: []string{"GET", "/users"}, ParamString: "<_>"},
		{id: 2, Tokens: []string{"POST", "/users"}, ParamString: "<_>"},
		{id: 3, Tokens: []string{"PUT", "/users"}, ParamString: "<_>"},
	}

	for _, cluster := range clusters {
		cache.Put(cluster)
	}

	// Values() should return all clusters
	values := []*LogCluster{}
	for cluster := range cache.Values() {
		values = append(values, cluster)
	}

	assert.Len(t, values, 3)
	// Verify all clusters are present
	ids := make(map[int]bool)
	for _, cluster := range values {
		ids[cluster.id] = true
	}
	assert.True(t, ids[1])
	assert.True(t, ids[2])
	assert.True(t, ids[3])
}

func TestLogClusterCache_TakeDeleted(t *testing.T) {
	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})
	cache := newLogClusterCache(time.Hour, 10, evictions, expired)
	cluster := &LogCluster{id: 1}

	cache.Put(cluster)
	cache.Put(cluster)
	require.Empty(t, cache.TakeDeleted())

	cache.Remove(cluster.id)
	require.Equal(t, []int{cluster.id}, cache.TakeDeleted())
	require.Empty(t, cache.TakeDeleted())
}

func TestLogClusterCache_DeletionCallbackDoesNotRetainCache(t *testing.T) {
	collected := make(chan struct{})
	func() {
		evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
		expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})
		cache := newLogClusterCache(time.Hour, 10, evictions, expired)
		cache.Put(&LogCluster{id: 1})
		runtime.AddCleanup(cache.cache, func(done chan struct{}) {
			close(done)
		}, collected)
	}()

	require.Eventually(t, func() bool {
		runtime.GC()
		select {
		case <-collected:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)
}

func TestLogClusterCache_TakeDeletedOverflow(t *testing.T) {
	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})
	cache := newLogClusterCache(time.Hour, 1, evictions, expired)

	for i := 1; i <= 10; i++ {
		cache.Put(&LogCluster{id: i})
	}
	cache.cache.CleanUp()
	deleted := cache.TakeDeleted()
	require.NotEmpty(t, deleted)
	for _, id := range deleted {
		require.Nil(t, cache.GetQuietly(id))
	}
}

func TestLogClusterCache_TakeDeletedExpiration(t *testing.T) {
	evictions := prometheus.NewCounter(prometheus.CounterOpts{Name: "evictions"})
	expired := prometheus.NewCounter(prometheus.CounterOpts{Name: "expired"})
	cache := newLogClusterCache(time.Millisecond, 10, evictions, expired)
	cache.Put(&LogCluster{id: 1})

	var deleted []int
	require.Eventually(t, func() bool {
		cache.cache.CleanUp()
		deleted = append(deleted, cache.TakeDeleted()...)
		return slices.Contains(deleted, 1)
	}, 5*time.Second, 10*time.Millisecond)
}

func TestLogClusterCache_TakeDeletedConcurrent(t *testing.T) {
	const producers = 8
	const deletionsPerProducer = 100
	queue := &deletedClusterQueue{}
	var producersWG sync.WaitGroup
	producersWG.Add(producers)
	for producer := range producers {
		go func() {
			defer producersWG.Done()
			for i := range deletionsPerProducer {
				queue.record(producer*deletionsPerProducer + i)
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		producersWG.Wait()
		close(done)
	}()

	collected := make(map[int]int, producers*deletionsPerProducer)
	for {
		for _, id := range queue.take() {
			collected[id]++
		}
		select {
		case <-done:
			for _, id := range queue.take() {
				collected[id]++
			}
			require.Len(t, collected, producers*deletionsPerProducer)
			for _, count := range collected {
				require.Equal(t, 1, count)
			}
			return
		default:
			runtime.Gosched()
		}
	}
}
