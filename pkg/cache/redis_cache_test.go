package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kit/log"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCache(t *testing.T) {
	c, err := mockRedisCache()
	require.Nil(t, err)
	defer c.redis.Close()

	keys := []string{"key1", "key2", "key3"}
	bufs := [][]byte{[]byte("data1"), []byte("data2"), []byte("data3")}
	miss := []string{"miss1", "miss2"}

	// ensure input correctness
	nHit := len(keys)
	require.Len(t, bufs, nHit)

	nMiss := len(miss)

	ctx := context.Background()

	c.Store(ctx, keys, bufs)

	// test hits
	found, data, missed := c.Fetch(ctx, keys)

	require.Len(t, found, nHit)
	require.Len(t, missed, 0)
	for i := 0; i < nHit; i++ {
		require.Equal(t, keys[i], found[i])
		require.Equal(t, bufs[i], data[i])
	}

	_, foundKey := c.FetchKey(ctx, "key1")
	assert.True(t, foundKey)

	// test misses
	found, _, missed = c.Fetch(ctx, miss)

	require.Len(t, found, 0)
	require.Len(t, missed, nMiss)
	for i := 0; i < nMiss; i++ {
		require.Equal(t, miss[i], missed[i])
	}

	_, foundKey = c.FetchKey(ctx, miss[0])
	assert.False(t, foundKey)
}

func TestRedisCacheMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	redisClient := &RedisClient{
		expiration: time.Minute,
		timeout:    100 * time.Millisecond,
		rdb: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: []string{redisServer.Addr()},
		}),
	}
	c := NewRedisCache("test", redisClient, reg, log.NewNopLogger())
	defer c.redis.Close()

	ctx := context.Background()
	c.Store(ctx, []string{"hit1", "hit2"}, [][]byte{[]byte("v1"), []byte("v2")})

	// Fetch: 2 hits, 1 miss
	c.Fetch(ctx, []string{"hit1", "hit2", "miss1"})

	// FetchKey: 1 hit, 1 miss
	c.FetchKey(ctx, "hit1")
	c.FetchKey(ctx, "miss2")

	expected := strings.NewReader(`
# HELP tempo_cache_hits_total Total number of cache hits.
# TYPE tempo_cache_hits_total counter
tempo_cache_hits_total{name="test",type="redis"} 3
# HELP tempo_cache_misses_total Total number of cache misses.
# TYPE tempo_cache_misses_total counter
tempo_cache_misses_total{name="test",type="redis"} 2
`)
	require.NoError(t, testutil.GatherAndCompare(reg, expected,
		"tempo_cache_hits_total", "tempo_cache_misses_total"))
}

func mockRedisCache() (*RedisCache, error) {
	redisServer, err := miniredis.Run()
	if err != nil {
		return nil, err
	}
	redisClient := &RedisClient{
		expiration: time.Minute,
		timeout:    100 * time.Millisecond,
		rdb: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: []string{redisServer.Addr()},
		}),
	}
	return NewRedisCache("mock", redisClient, nil, log.NewNopLogger()), nil
}
