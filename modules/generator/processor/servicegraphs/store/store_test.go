package store

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const clientService = "client"

func TestStoreUpsertEdge(t *testing.T) {
	const keyStr = "key"

	var onCompletedCount int
	var onExpireCount int

	storeInterface := NewStore(time.Hour, 1, countingCallback(&onCompletedCount), countingCallback(&onExpireCount), newTestCounter())

	s := storeInterface.(*store)
	assert.Equal(t, 0, s.len())

	// Insert first half of an edge
	isNew, err := s.UpsertEdge(keyStr, Client, func(e *Edge) {
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
	isNew, err = s.UpsertEdge(keyStr, Server, func(e *Edge) {
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
	isNew, err = s.UpsertEdge(keyStr, Client, func(e *Edge) {
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

	storeInterface := NewStore(time.Hour, 1, countingCallback(&onCallbackCounter), countingCallback(&onCallbackCounter), newTestCounter())

	s := storeInterface.(*store)
	assert.Equal(t, 0, s.len())

	isNew, err := s.UpsertEdge("key-1", Client, func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)
	require.Equal(t, true, isNew)
	assert.Equal(t, 1, s.len())

	_, err = s.UpsertEdge("key-2", Client, func(e *Edge) {
		e.ClientService = clientService
	})
	require.ErrorIs(t, err, ErrTooManyItems)
	assert.Equal(t, 1, s.len())

	isNew, err = s.UpsertEdge("key-1", Client, func(e *Edge) {
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
	storeInterface := NewStore(-time.Second, testSize, onComplete, countingCallback(&onExpireCount), newTestCounter())
	s := storeInterface.(*store)

	for key := range keys {
		isNew, err := s.UpsertEdge(key, Client, noopCallback)
		require.NoError(t, err)
		require.Equal(t, true, isNew)
	}

	s.Expire()
	assert.Equal(t, 0, s.len())
	assert.Equal(t, 0, onCompletedCount)
	assert.Equal(t, testSize, onExpireCount)
}

func TestStore_concurrency(t *testing.T) {
	s := NewStore(10*time.Millisecond, 100000, noopCallback, noopCallback, newTestCounter())

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

		_, err := s.UpsertEdge(string(key), Client, func(e *Edge) {
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
	s := NewStore(time.Hour, 10, noopCallback, noopCallback, newTestCounter())

	const key = "trace-span"
	const side = Client
	assert.False(t, s.HasDroppedSpanSide(key, side))

	droppedCounterpart := s.AddDroppedSpanSide(key, side)
	assert.False(t, droppedCounterpart)
	assert.True(t, s.HasDroppedSpanSide(key, side))
}

func TestStore_AddDroppedSpanSide_respectsMaxItems(t *testing.T) {
	s := NewStore(time.Hour, 2, noopCallback, noopCallback, newTestCounter())

	s.AddDroppedSpanSide("k1", Client)
	s.AddDroppedSpanSide("k2", Client)
	s.AddDroppedSpanSide("k3", Client)

	assert.True(t, s.HasDroppedSpanSide("k1", Client))
	assert.True(t, s.HasDroppedSpanSide("k2", Client))
	assert.False(t, s.HasDroppedSpanSide("k3", Client))
}

func TestStore_AddDroppedSpanSide_overflowMetricIncrementsAtCapacity(t *testing.T) {
	overflowCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_store_dropped_span_side_cache_overflow_total",
		Help: "test counter",
	})
	s := NewStore(time.Hour, 2, noopCallback, noopCallback, overflowCounter)

	s.AddDroppedSpanSide("k1", Client)
	s.AddDroppedSpanSide("k2", Client)
	s.AddDroppedSpanSide("k1", Client) // refresh existing, should not overflow
	s.AddDroppedSpanSide("k3", Client) // overflow

	assert.Equal(t, float64(1), testutil.ToFloat64(overflowCounter))
}

func TestStore_AddDroppedSpanSide_expiresWithTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 10, noopCallback, noopCallback, newTestCounter())

		const key = "k-expire"
		s.AddDroppedSpanSide(key, Client)
		assert.True(t, s.HasDroppedSpanSide(key, Client))

		time.Sleep(2 * time.Second)
		s.Expire()

		assert.False(t, s.HasDroppedSpanSide(key, Client))
	})
}

func TestStore_AddDroppedSpanSide_sidesAreIndependent(t *testing.T) {
	s := NewStore(time.Hour, 10, noopCallback, noopCallback, newTestCounter())

	const key = "k-side"
	s.AddDroppedSpanSide(key, Client)

	assert.True(t, s.HasDroppedSpanSide(key, Client))
	assert.False(t, s.HasDroppedSpanSide(key, Server))
}

func TestStore_AddDroppedSpanSide_dropsExistingCounterpartEdge(t *testing.T) {
	si := NewStore(time.Hour, 10, noopCallback, noopCallback, newTestCounter())
	s := si.(*store)

	_, err := s.UpsertEdge("k1", Client, func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)

	_, err = s.UpsertEdge("k2", Client, func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)

	assert.Equal(t, 2, s.len())

	// Server side for k1 was filtered/dropped; existing client-side edge for k1 should be removed.
	droppedCounterpart := s.AddDroppedSpanSide("k1", Server)
	assert.True(t, droppedCounterpart)

	assert.Equal(t, 1, s.len())
	_, foundK1 := s.m["k1"]
	assert.False(t, foundK1)
	_, foundK2 := s.m["k2"]
	assert.True(t, foundK2)
}

func TestStore_AddDroppedSpanSide_refreshesTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 10, noopCallback, noopCallback, newTestCounter())

		const key = "k-refresh"
		s.AddDroppedSpanSide(key, Client)

		time.Sleep(500 * time.Millisecond)
		s.AddDroppedSpanSide(key, Client)

		time.Sleep(700 * time.Millisecond)
		s.Expire()
		assert.True(t, s.HasDroppedSpanSide(key, Client))

		time.Sleep(500 * time.Millisecond)
		s.Expire()
		assert.False(t, s.HasDroppedSpanSide(key, Client))
	})
}

func TestStore_AddDroppedSpanSide_refreshAllowedAtMaxItems(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 2, noopCallback, noopCallback, newTestCounter())

		s.AddDroppedSpanSide("k1", Client)
		s.AddDroppedSpanSide("k2", Client)
		time.Sleep(700 * time.Millisecond)
		// At capacity: this should refresh k1, not be blocked.
		s.AddDroppedSpanSide("k1", Client)
		// Still at capacity: new key should be rejected.
		s.AddDroppedSpanSide("k3", Client)

		assert.True(t, s.HasDroppedSpanSide("k1", Client))
		assert.True(t, s.HasDroppedSpanSide("k2", Client))
		assert.False(t, s.HasDroppedSpanSide("k3", Client))

		time.Sleep(500 * time.Millisecond)
		s.Expire()
		// k2 should be expired; k1 should remain because it was refreshed.
		assert.True(t, s.HasDroppedSpanSide("k1", Client))
		assert.False(t, s.HasDroppedSpanSide("k2", Client))
	})
}

func TestStore_ExpireDroppedSpanSide_mixedTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewStore(time.Second, 10, noopCallback, noopCallback, newTestCounter())

		s.AddDroppedSpanSide("old", Client)
		time.Sleep(700 * time.Millisecond)
		s.AddDroppedSpanSide("new", Client)

		time.Sleep(500 * time.Millisecond)
		s.Expire()

		assert.False(t, s.HasDroppedSpanSide("old", Client))
		assert.True(t, s.HasDroppedSpanSide("new", Client))
	})
}

func TestStore_ExpireDroppedSpanSide_doesNotAffectEdges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		si := NewStore(time.Second, 10, noopCallback, noopCallback, newTestCounter())
		s := si.(*store)

		s.AddDroppedSpanSide("d1", Client)
		_, err := s.UpsertEdge("e1", Client, func(e *Edge) {
			e.ClientService = clientService
		})
		require.NoError(t, err)
		assert.Equal(t, 1, s.len())

		time.Sleep(2 * time.Second)
		s.Expire()

		assert.False(t, s.HasDroppedSpanSide("d1", Client))
		// Edge should also be expired by normal edge-expire path.
		assert.Equal(t, 0, s.len())
	})
}

func TestStore_UpsertEdge_newEdgeWithDroppedCounterpart_returnsErrDroppedSpanSide(t *testing.T) {
	si := NewStore(time.Hour, 10, noopCallback, noopCallback, newTestCounter())
	s := si.(*store)

	const key = "trace-span"
	s.AddDroppedSpanSide(key, Server)

	isNew, err := s.UpsertEdge(key, Client, func(e *Edge) {
		e.ClientService = clientService
	})

	require.ErrorIs(t, err, ErrDroppedSpanSide)
	assert.True(t, isNew)
	assert.Equal(t, 0, s.len())
}

func BenchmarkStoreUpsertEdge(b *testing.B) {
	// Benchmark the performance of UpsertEdge with edge pooling enabled
	s := NewStore(10*time.Millisecond, 1e10, noopCallback, noopCallback, newTestCounter())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, err := s.UpsertEdge(key, Client, func(e *Edge) {
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

func newTestCounter() prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{})
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
