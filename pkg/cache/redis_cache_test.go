package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kit/log"
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

func TestRedisCache_MaxItemSizeReturnsConfigured(t *testing.T) {
	c, err := mockRedisCache()
	require.NoError(t, err)
	defer c.redis.Close()

	require.Equal(t, c.MaxItemSize(), 10*1024*1024,
		"RedisCache.MaxItemSize() must return the configured max size")
}

func mockRedisCache() (*RedisCache, error) {
	redisServer, err := miniredis.Run()
	if err != nil {
		return nil, err
	}
	cfg := &RedisConfig{
		Expiration:  time.Minute,
		Timeout:     100 * time.Millisecond,
		Endpoint:    strings.Join([]string{redisServer.Addr()}, ","),
		MaxItemSize: 10 * 1024 * 1024,
	}
	redisClient := NewRedisClient(cfg, "mock", nil)
	return NewRedisCache("mock", redisClient, cfg.MaxItemSize, nil, log.NewNopLogger()), nil
}
