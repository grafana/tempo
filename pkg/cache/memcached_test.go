package cache_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/gomemcache/memcache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/pkg/cache"
)

func TestMemcached(t *testing.T) {
	t.Run("unbatched", func(t *testing.T) {
		client := newMockMemcache()
		memcache := cache.NewMemcached(cache.MemcachedConfig{}, client,
			"test", 0, nil, log.NewNopLogger())

		testMemcache(t, memcache)
	})
}

func testMemcache(t *testing.T, memcache *cache.Memcached) {
	numKeys := 1000

	ctx := context.Background()
	keysIncMissing := make([]string, 0, numKeys)
	keys := make([]string, 0, numKeys)
	bufs := make([][]byte, 0, numKeys)

	// Insert 1000 keys skipping all multiples of 5.
	for i := 0; i < numKeys; i++ {
		keysIncMissing = append(keysIncMissing, fmt.Sprint(i))
		if i%5 == 0 {
			continue
		}

		keys = append(keys, fmt.Sprint(i))
		bufs = append(bufs, []byte(fmt.Sprint(i)))
	}
	memcache.Store(ctx, keys, bufs)

	found, bufs, missing := memcache.Fetch(ctx, keysIncMissing)
	for i := 0; i < numKeys; i++ {
		if i%5 == 0 {
			require.Equal(t, fmt.Sprint(i), missing[0])
			missing = missing[1:]
			continue
		}

		require.Equal(t, fmt.Sprint(i), found[0])
		require.Equal(t, fmt.Sprint(i), string(bufs[0]))
		found = found[1:]
		bufs = bufs[1:]
	}

	_, foundKey := memcache.FetchKey(ctx, "1")
	assert.True(t, foundKey)

	_, foundKey = memcache.FetchKey(ctx, "5")
	assert.False(t, foundKey)
}

// mockMemcache whose calls fail 1/3rd of the time.
type mockMemcacheFailing struct {
	*mockMemcache
	calls atomic.Uint64
}

func newMockMemcacheFailing() *mockMemcacheFailing {
	return &mockMemcacheFailing{
		mockMemcache: newMockMemcache(),
	}
}

func (c *mockMemcacheFailing) GetMulti(ctx context.Context, keys []string, _ ...memcache.Option) (map[string]*memcache.Item, error) {
	calls := c.calls.Inc()
	if calls%3 == 0 {
		return nil, errors.New("fail")
	}

	return c.mockMemcache.GetMulti(ctx, keys)
}

func TestMemcacheFailure(t *testing.T) {
	t.Run("unbatched", func(t *testing.T) {
		client := newMockMemcacheFailing()
		memcache := cache.NewMemcached(cache.MemcachedConfig{}, client,
			"test", 0, nil, log.NewNopLogger())

		testMemcacheFailing(t, memcache)
	})
}

func testMemcacheFailing(t *testing.T, memcache *cache.Memcached) {
	numKeys := 1000

	ctx := context.Background()
	keysIncMissing := make([]string, 0, numKeys)
	keys := make([]string, 0, numKeys)
	bufs := make([][]byte, 0, numKeys)
	// Insert 1000 keys skipping all multiples of 5.
	for i := 0; i < numKeys; i++ {
		keysIncMissing = append(keysIncMissing, fmt.Sprint(i))
		if i%5 == 0 {
			continue
		}
		keys = append(keys, fmt.Sprint(i))
		bufs = append(bufs, []byte(fmt.Sprint(i)))
	}
	memcache.Store(ctx, keys, bufs)

	for i := 0; i < 10; i++ {
		found, bufs, missing := memcache.Fetch(ctx, keysIncMissing)

		require.Equal(t, len(found), len(bufs))
		for i := range found {
			require.Equal(t, found[i], string(bufs[i]))
		}

		keysReturned := make(map[string]struct{})
		for _, key := range found {
			_, ok := keysReturned[key]
			require.False(t, ok, "duplicate key returned")

			keysReturned[key] = struct{}{}
		}
		for _, key := range missing {
			_, ok := keysReturned[key]
			require.False(t, ok, "duplicate key returned")

			keysReturned[key] = struct{}{}
		}

		for _, key := range keys {
			_, ok := keysReturned[key]
			require.True(t, ok, "key missing %s", key)
		}
	}
}

func TestMemcacheStop(t *testing.T) {
	t.Run("unbatched", func(_ *testing.T) {
		client := newMockMemcacheFailing()
		memcache := cache.NewMemcached(cache.MemcachedConfig{}, client,
			"test", 0, nil, log.NewNopLogger())

		testMemcachedStopping(memcache)
	})
}

func testMemcachedStopping(memcache *cache.Memcached) {
	numKeys := 1000
	ctx := context.Background()
	keys := make([]string, 0, numKeys)
	bufs := make([][]byte, 0, numKeys)
	for i := 0; i < numKeys; i++ {
		keys = append(keys, fmt.Sprint(i))
		bufs = append(bufs, []byte(fmt.Sprint(i)))
	}

	memcache.Store(ctx, keys, bufs)

	go memcache.Fetch(ctx, keys)
	memcache.Stop()
}

func TestMemcachedCacheMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	client := newMockMemcache()
	mc := cache.NewMemcached(cache.MemcachedConfig{}, client, "test", 0, reg, log.NewNopLogger())
	defer mc.Stop()

	ctx := context.Background()
	mc.Store(ctx, []string{"hit1", "hit2"}, [][]byte{[]byte("v1"), []byte("v2")})

	// Fetch: 2 hits, 1 miss
	mc.Fetch(ctx, []string{"hit1", "hit2", "miss1"})

	// FetchKey: 1 hit, 1 miss
	mc.FetchKey(ctx, "hit1")
	mc.FetchKey(ctx, "miss2")

	expected := strings.NewReader(`
# HELP tempo_cache_requests_total Total number of cache requests by status (hit, miss, fail).
# TYPE tempo_cache_requests_total counter
tempo_cache_requests_total{name="test",status="hit",type="memcached"} 3
tempo_cache_requests_total{name="test",status="miss",type="memcached"} 2
`)
	require.NoError(t, testutil.GatherAndCompare(reg, expected,
		"tempo_cache_requests_total"))
}

func TestMemcachedRespectsCancelledContext(t *testing.T) {
	client := newMockMemcache()
	memcache := cache.NewMemcached(cache.MemcachedConfig{}, client,
		"test", 0, nil, log.NewNopLogger())
	defer memcache.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	found, bufs, missing := memcache.Fetch(ctx, []string{"1"})
	require.Nil(t, found)
	require.Nil(t, bufs)
	require.Equal(t, []string{"1"}, missing)

	val, f := memcache.FetchKey(ctx, "1")
	require.Nil(t, val)
	require.False(t, f)

	memcache.Store(ctx, []string{"1"}, [][]byte{[]byte("1")})
	// confirm that the value is not stored
	mi, err := client.Get("1")
	require.ErrorContains(t, err, "cache miss")
	require.Nil(t, mi)
}
