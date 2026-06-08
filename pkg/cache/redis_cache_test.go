package cache

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
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

func TestRedisStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil_is_200", err: nil, want: "200"},
		{name: "redis_nil_is_404", err: redis.Nil, want: "404"},
		{name: "deadline_is_504", err: context.DeadlineExceeded, want: "504"},
		{name: "canceled_is_504", err: context.Canceled, want: "504"},
		{name: "generic_is_500", err: errors.New("boom"), want: "500"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, redisStatusCode(tc.err))
		})
	}
}

func TestRedisCache_MaxItemSizeReturnsConfigured(t *testing.T) {
	c, _, err := mockRedisCacheWithRegistry()
	require.NoError(t, err)
	defer c.redis.Close()

	require.Equal(t, 10*1024*1024, c.MaxItemSize(),
		"RedisCache.MaxItemSize() must return the configured max size")
}

// TestRedisCache_StoreIsMeasured guards parity with the memcached client: every
// write path must contribute a sample to the request-duration histogram so that
// dashboards and alerts on cache write latency/errors keep working.
func TestRedisCache_StoreIsMeasured(t *testing.T) {
	c, reg, err := mockRedisCacheWithRegistry()
	require.NoError(t, err)
	defer c.redis.Close()

	const metric = "tempo_rediscache_request_duration_seconds"
	count, err := testutil.GatherAndCount(reg, metric)
	require.NoError(t, err)
	require.Equal(t, 0, count, "histogram must be empty before Store")

	c.Store(context.Background(), []string{"k1", "k2"}, [][]byte{[]byte("v1"), []byte("v2")})

	count, err = testutil.GatherAndCount(reg, metric)
	require.NoError(t, err)
	require.Equal(t, 1, count, "Store must emit exactly one observation on %s", metric)

	mfs, err := reg.Gather()
	require.NoError(t, err)
	var labels map[string]string
	for _, mf := range mfs {
		if mf.GetName() != metric {
			continue
		}
		require.Len(t, mf.GetMetric(), 1)
		labels = make(map[string]string, 3)
		for _, l := range mf.GetMetric()[0].GetLabel() {
			labels[l.GetName()] = l.GetValue()
		}
	}
	require.Equal(t, "RedisCache.MSet", labels["method"])
	require.Equal(t, "200", labels["status_code"])
	require.Equal(t, "mock", labels["name"])
}

func mockRedisCache() (*RedisCache, error) {
	c, _, err := mockRedisCacheWithRegistry()
	return c, err
}

func mockRedisCacheWithRegistry() (*RedisCache, *prometheus.Registry, error) {
	redisServer, err := miniredis.Run()
	if err != nil {
		return nil, nil, err
	}
	cfg := &RedisConfig{
		Expiration:  time.Minute,
		Timeout:     100 * time.Millisecond,
		Endpoint:    strings.Join([]string{redisServer.Addr()}, ","),
		SingleNode:  true,
		MaxItemSize: 10 * 1024 * 1024,
	}
	reg := prometheus.NewRegistry()
	redisClient, err := NewRedisClient(cfg, "mock", prometheus.NewRegistry())
	if err != nil {
		return nil, nil, err
	}
	return NewRedisCache("mock", redisClient, cfg.MaxItemSize, reg, log.NewNopLogger()), reg, nil
}
