package cache_test

import (
	"context"
	crand "crypto/rand"
	"math/rand"
	"strconv"
	"testing"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestBackground(t *testing.T) {
	c := cache.NewBackground("mock", cache.BackgroundConfig{
		WriteBackGoroutines: 1,
		WriteBackBuffer:     100,
	}, test.NewMockClient(), nil)

	keys, chunks := fillCache(c)
	cache.Flush(c)

	testCacheSingle(t, c, keys, chunks)
	testCacheMultiple(t, c, keys, chunks)
	testCacheMiss(t, c)
}

func fillCache(cache cache.Cache) ([]string, [][]byte) {
	// put a set of chunks, larger than background batch size, with varying timestamps and values
	keys := []string{}
	bufs := [][]byte{}
	for i := 0; i < 111; i++ {

		buf := make([]byte, rand.Intn(100))
		_, err := crand.Read(buf)
		if err != nil {
			panic(err)
		}

		keys = append(keys, strconv.Itoa(i))
		bufs = append(bufs, buf)
	}

	cache.Store(context.Background(), keys, bufs)
	return keys, bufs
}

func testCacheSingle(t *testing.T, cache cache.Cache, keys []string, bufs [][]byte) {
	for i := 0; i < 100; i++ {
		index := rand.Intn(len(keys))
		key := keys[index]

		found, foundBufs, missingKeys := cache.Fetch(context.Background(), []string{key})
		require.Len(t, found, 1)
		require.Len(t, foundBufs, 1)
		require.Len(t, missingKeys, 0)
		require.Equal(t, bufs[index], foundBufs[0])
	}
}

func testCacheMultiple(t *testing.T, cache cache.Cache, keys []string, bufs [][]byte) {
	// test getting them all
	found, foundBufs, missingKeys := cache.Fetch(context.Background(), keys)
	require.Len(t, found, len(keys))
	require.Len(t, foundBufs, len(keys))
	require.Len(t, missingKeys, 0)
	require.Equal(t, bufs, foundBufs)
}

func testCacheMiss(t *testing.T, cache cache.Cache) {
	for i := 0; i < 100; i++ {
		key := strconv.Itoa(rand.Int()) // arbitrary key which should fail: no chunk key is a single integer
		found, bufs, missing := cache.Fetch(context.Background(), []string{key})
		require.Empty(t, found)
		require.Empty(t, bufs)
		require.Len(t, missing, 1)
	}
}
