package store

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/synctest"
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

func TestStore_AddDroppedSpanSide(t *testing.T) {
	s := NewStore(time.Hour, 10, noopCallback, noopCallback)

	const key = "trace-span"
	const side = ClientSide
	assert.False(t, s.HasDroppedSpanSide(key, side))

	s.AddDroppedSpanSide(key, side)
	assert.True(t, s.HasDroppedSpanSide(key, side))
}

func TestStore_AddDroppedSpanSide_respectsMaxItems(t *testing.T) {
	s := NewStore(time.Hour, 2, noopCallback, noopCallback)

	s.AddDroppedSpanSide("k1", ClientSide)
	s.AddDroppedSpanSide("k2", ClientSide)
	s.AddDroppedSpanSide("k3", ClientSide)

	assert.True(t, s.HasDroppedSpanSide("k1", ClientSide))
	assert.True(t, s.HasDroppedSpanSide("k2", ClientSide))
	assert.False(t, s.HasDroppedSpanSide("k3", ClientSide))
}

func TestStore_AddDroppedSpanSide_expiresWithTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 10, noopCallback, noopCallback)

		const key = "k-expire"
		s.AddDroppedSpanSide(key, ClientSide)
		assert.True(t, s.HasDroppedSpanSide(key, ClientSide))

		time.Sleep(2 * time.Second)
		s.Expire()

		assert.False(t, s.HasDroppedSpanSide(key, ClientSide))
	})
}

func TestStore_AddDroppedSpanSide_sidesAreIndependent(t *testing.T) {
	s := NewStore(time.Hour, 10, noopCallback, noopCallback)

	const key = "k-side"
	s.AddDroppedSpanSide(key, ClientSide)

	assert.True(t, s.HasDroppedSpanSide(key, ClientSide))
	assert.False(t, s.HasDroppedSpanSide(key, ServerSide))
}

func TestStore_AddDroppedSpanSide_refreshesTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 10, noopCallback, noopCallback)

		const key = "k-refresh"
		s.AddDroppedSpanSide(key, ClientSide)

		time.Sleep(500 * time.Millisecond)
		s.AddDroppedSpanSide(key, ClientSide)

		time.Sleep(700 * time.Millisecond)
		s.Expire()
		assert.True(t, s.HasDroppedSpanSide(key, ClientSide))

		time.Sleep(500 * time.Millisecond)
		s.Expire()
		assert.False(t, s.HasDroppedSpanSide(key, ClientSide))
	})
}

func TestStore_AddDroppedSpanSide_refreshAllowedAtMaxItems(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 2, noopCallback, noopCallback)

		s.AddDroppedSpanSide("k1", ClientSide)
		s.AddDroppedSpanSide("k2", ClientSide)
		time.Sleep(700 * time.Millisecond)
		// At capacity: this should refresh k1, not be blocked.
		s.AddDroppedSpanSide("k1", ClientSide)
		// Still at capacity: new key should be rejected.
		s.AddDroppedSpanSide("k3", ClientSide)

		assert.True(t, s.HasDroppedSpanSide("k1", ClientSide))
		assert.True(t, s.HasDroppedSpanSide("k2", ClientSide))
		assert.False(t, s.HasDroppedSpanSide("k3", ClientSide))

		time.Sleep(500 * time.Millisecond)
		s.Expire()
		// k2 should be expired; k1 should remain because it was refreshed.
		assert.True(t, s.HasDroppedSpanSide("k1", ClientSide))
		assert.False(t, s.HasDroppedSpanSide("k2", ClientSide))
	})
}

func TestStore_ExpireDroppedSpanSide_mixedTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 10, noopCallback, noopCallback)

		s.AddDroppedSpanSide("old", ClientSide)
		time.Sleep(700 * time.Millisecond)
		s.AddDroppedSpanSide("new", ClientSide)

		time.Sleep(500 * time.Millisecond)
		s.Expire()

		assert.False(t, s.HasDroppedSpanSide("old", ClientSide))
		assert.True(t, s.HasDroppedSpanSide("new", ClientSide))
	})
}

func TestStore_ExpireDroppedSpanSide_doesNotAffectEdges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		si := NewStore(time.Second, 10, noopCallback, noopCallback)
		s := si.(*store)

		s.AddDroppedSpanSide("d1", ClientSide)
		_, err := s.UpsertEdge("e1", func(e *Edge) {
			e.ClientService = clientService
		})
		require.NoError(t, err)
		assert.Equal(t, 1, s.len())

		time.Sleep(2 * time.Second)
		s.Expire()

		assert.False(t, s.HasDroppedSpanSide("d1", ClientSide))
		// Edge should also be expired by normal edge-expire path.
		assert.Equal(t, 0, s.len())
	})
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

func TestResetEdge(t *testing.T) {
	// Create an edge with all fields set to non-zero values
	dimensions := map[string]string{"key1": "value1", "key2": "value2"}
	e := &Edge{
		key:                     "test-key",
		TraceID:                 "trace-123",
		ConnectionType:          Database,
		ServerService:           "server-svc",
		ClientService:           "client-svc",
		ServerLatencySec:        1.5,
		ClientLatencySec:        2.5,
		ServerStartTimeUnixNano: 1234567890,
		ClientEndTimeUnixNano:   9876543210,
		Failed:                  true,
		Dimensions:              dimensions,
		PeerNode:                "peer-node",
		expiration:              999999,
		SpanMultiplier:          5.0,
	}

	// Reset the edge
	resetEdge(e)

	// Verify all fields are properly reset
	assert.Equal(t, "", e.key, "key should be preserved")
	assert.Equal(t, "", e.TraceID, "TraceID should be reset")
	assert.Equal(t, Unknown, e.ConnectionType, "ConnectionType should be reset to Unknown")
	assert.Equal(t, "", e.ServerService, "ServerService should be reset")
	assert.Equal(t, "", e.ClientService, "ClientService should be reset")
	assert.Equal(t, 0.0, e.ServerLatencySec, "ServerLatencySec should be reset")
	assert.Equal(t, 0.0, e.ClientLatencySec, "ClientLatencySec should be reset")
	assert.Equal(t, uint64(0), e.ServerStartTimeUnixNano, "ServerStartTimeUnixNano should be reset")
	assert.Equal(t, uint64(0), e.ClientEndTimeUnixNano, "ClientEndTimeUnixNano should be reset")
	assert.Equal(t, false, e.Failed, "Failed should be reset")
	assert.NotNil(t, e.Dimensions, "Dimensions should not be nil (preserved from original)")
	assert.Equal(t, 0, len(e.Dimensions), "Dimensions should be empty (cleared)")
	assert.Equal(t, "", e.PeerNode, "PeerNode should be reset")
	assert.Equal(t, int64(0), e.expiration, "expiration should be reset")
	assert.Equal(t, 1.0, e.SpanMultiplier, "SpanMultiplier should be reset to 1")

	assert.NotNil(t, e.Dimensions, "Dimensions should not be nil")
	originalPtr := reflect.ValueOf(dimensions).Pointer()
	newPtr := reflect.ValueOf(e.Dimensions).Pointer()
	assert.Equal(t, originalPtr, newPtr, "Dimensions map should be the exact same instance, not reallocated")
}
