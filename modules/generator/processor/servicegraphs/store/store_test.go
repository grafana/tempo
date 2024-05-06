package store

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const clientService = "client"

func TestStoreUpsertEdge(t *testing.T) {
	const keyStr = "key"

	var onCompletedCount int
	var onExpireCount int

	storeInterface := NewStore(time.Hour, 1, countingCallback(&onCompletedCount), countingCallback(&onExpireCount))

	s := storeInterface.(*store)
	assert.Equal(t, 0, s.len())

	// Insert first half of an edge
	isNew, err := s.UpsertEdge(keyStr, func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)
	require.Equal(t, true, isNew)
	assert.Equal(t, 1, s.len())

	// Nothing should be evicted as TTL is set to 1h
	assert.False(t, s.tryEvictHead())
	assert.Equal(t, 0, onCompletedCount)
	assert.Equal(t, 0, onExpireCount)

	// Insert the second half of an edge
	isNew, err = s.UpsertEdge(keyStr, func(e *Edge) {
		assert.Equal(t, clientService, e.ClientService)
		e.ServerService = "server"
	})
	require.NoError(t, err)
	require.Equal(t, false, isNew)
	// Edge is complete and should have been removed
	assert.Equal(t, 0, s.len())

	assert.Equal(t, 1, onCompletedCount)
	assert.Equal(t, 0, onExpireCount)

	// Insert an edge that will immediately expire
	isNew, err = s.UpsertEdge(keyStr, func(e *Edge) {
		e.ClientService = clientService
		e.expiration = 0
	})
	require.NoError(t, err)
	require.Equal(t, true, isNew)
	assert.Equal(t, 1, s.len())
	assert.Equal(t, 1, onCompletedCount)
	assert.Equal(t, 0, onExpireCount)

	assert.True(t, s.tryEvictHead())
	assert.Equal(t, 0, s.len())
	assert.Equal(t, 1, onCompletedCount)
	assert.Equal(t, 1, onExpireCount)
}

func TestStoreUpsertEdge_errTooManyItems(t *testing.T) {
	var onCallbackCounter int

	storeInterface := NewStore(time.Hour, 1, countingCallback(&onCallbackCounter), countingCallback(&onCallbackCounter))

	s := storeInterface.(*store)
	assert.Equal(t, 0, s.len())

	isNew, err := s.UpsertEdge("key-1", func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)
	require.Equal(t, true, isNew)
	assert.Equal(t, 1, s.len())

	_, err = s.UpsertEdge("key-2", func(e *Edge) {
		e.ClientService = clientService
	})
	require.ErrorIs(t, err, ErrTooManyItems)
	assert.Equal(t, 1, s.len())

	isNew, err = s.UpsertEdge("key-1", func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)
	require.Equal(t, false, isNew)
	assert.Equal(t, 1, s.len())

	assert.Equal(t, 0, onCallbackCounter)
}

func TestStoreExpire(t *testing.T) {
	const testSize = 100

	keys := map[string]bool{}
	for i := 0; i < testSize; i++ {
		keys[fmt.Sprintf("key-%d", i)] = true
	}

	var onCompletedCount int
	var onExpireCount int

	onComplete := func(e *Edge) {
		onCompletedCount++
		assert.True(t, keys[e.key])
	}
	// New edges are immediately expired
	storeInterface := NewStore(-time.Second, testSize, onComplete, countingCallback(&onExpireCount))
	s := storeInterface.(*store)

	for key := range keys {
		isNew, err := s.UpsertEdge(key, noopCallback)
		require.NoError(t, err)
		require.Equal(t, true, isNew)
	}

	s.Expire()
	assert.Equal(t, 0, s.len())
	assert.Equal(t, 0, onCompletedCount)
	assert.Equal(t, testSize, onExpireCount)
}

func TestStore_concurrency(t *testing.T) {
	s := NewStore(10*time.Millisecond, 100000, noopCallback, noopCallback)

	end := make(chan struct{})

	accessor := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	go accessor(func() {
		key := make([]rune, 6)
		for i := range key {
			key[i] = letters[rand.Intn(len(letters))]
		}

		_, err := s.UpsertEdge(string(key), func(e *Edge) {
			e.ClientService = string(key)
		})
		assert.NoError(t, err)
	})

	go accessor(func() {
		s.Expire()
	})

	time.Sleep(100 * time.Millisecond)
	close(end)
}

func BenchmarkStoreUpsertEdge(b *testing.B) {
	// Benchmark the performance of UpsertEdge with edge pooling enabled
	s := NewStore(10*time.Millisecond, 1e10, noopCallback, noopCallback)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, err := s.UpsertEdge(key, func(e *Edge) {
			e.ClientService = clientService
		})
		require.NoError(b, err)
	}
}

func noopCallback(_ *Edge) {
}

func countingCallback(counter *int) func(*Edge) {
	return func(_ *Edge) {
		*counter++
	}
}
