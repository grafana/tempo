package store

import (
	"fmt"
	"math/rand"
	"reflect"
	"sync"
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

	s := NewStore(time.Hour, 1, countingCallback(&onCompletedCount), countingCallback(&onExpireCount), newTestCounter())
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

	s := NewStore(time.Hour, 1, countingCallback(&onCallbackCounter), countingCallback(&onCallbackCounter), newTestCounter())
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
		assert.True(t, keys[e.Key()])
	}
	// New edges are immediately expired
	s := NewStore(-time.Second, testSize, onComplete, countingCallback(&onExpireCount), newTestCounter())

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
	var wg sync.WaitGroup

	accessor := func(f func()) {
		defer wg.Done()
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	wg.Add(2)
	go accessor(func() {
		key := make([]byte, 24)
		for i := range key {
			key[i] = letters[rand.Intn(len(letters))]
		}

		service := string(key)
		_, err := s.UpsertEdgeFromBytes(key[:16], key[16:], Client, func(e *Edge) {
			e.ClientService = service
		})
		assert.NoError(t, err)
	})

	go accessor(func() {
		s.Expire()
	})

	time.Sleep(100 * time.Millisecond)
	close(end)
	wg.Wait()
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
	s := NewStore(time.Hour, 10, noopCallback, noopCallback, newTestCounter())

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
	_, foundK1 := s.m[edgeKeyFromString("k1")]
	assert.False(t, foundK1)
	_, foundK2 := s.m[edgeKeyFromString("k2")]
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
		s := NewStore(time.Second, 10, noopCallback, noopCallback, newTestCounter())

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
	s := NewStore(time.Hour, 10, noopCallback, noopCallback, newTestCounter())

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
	s := NewStore(10*time.Millisecond, 1, noopCallback, noopCallback, newTestCounter())
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := []byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.UpsertEdgeFromBytes(traceID, spanID, Client, benchmarkUpdateClientEdge); err != nil {
			b.Fatal(err)
		}
		if _, err := s.UpsertEdgeFromBytes(traceID, spanID, Server, benchmarkUpdateServerEdge); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkUpdateClientEdge(e *Edge) {
	e.ClientService = clientService
}

func benchmarkUpdateServerEdge(e *Edge) {
	e.ServerService = "server"
}

func TestStoreUpsertEdgeFromBytes_ValidIDs(t *testing.T) {
	// Verify the callback runs and the edge pairs correctly across two calls.
	s := NewStore(time.Hour, 1, noopCallback, noopCallback, newTestCounter())

	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := []byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8}

	isNew, err := s.UpsertEdgeFromBytes(traceID, spanID, Client, func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)
	require.True(t, isNew)
	assert.Equal(t, 1, s.len())

	isNew, err = s.UpsertEdgeFromBytes(traceID, spanID, Server, func(e *Edge) {
		assert.Equal(t, clientService, e.ClientService)
		e.ServerService = "server"
	})
	require.NoError(t, err)
	require.False(t, isNew)
	// Edge completed and removed.
	assert.Equal(t, 0, s.len())
}

func TestStoreUpsertEdgeFromBytes_OversizedIDs(t *testing.T) {
	// Malformed oversized IDs use an encoded fallback without truncating their
	// contents or changing pairing behavior.
	s := NewStore(time.Hour, 1, noopCallback, noopCallback, newTestCounter())

	bigTraceID := make([]byte, 33)
	bigSpanID := make([]byte, 33)
	for i := range bigTraceID {
		bigTraceID[i] = byte(i + 1)
		bigSpanID[i] = byte(i + 100)
	}

	isNew, err := s.UpsertEdgeFromBytes(bigTraceID, bigSpanID, Client, func(e *Edge) {
		e.ClientService = clientService
	})
	require.NoError(t, err)
	require.True(t, isNew)
	assert.Equal(t, 1, s.len())

	// The same oversized key must hit the existing edge.
	isNew, err = s.UpsertEdgeFromBytes(bigTraceID, bigSpanID, Server, func(e *Edge) {
		assert.Equal(t, clientService, e.ClientService)
		e.ServerService = "server"
	})
	require.NoError(t, err)
	require.False(t, isNew)
	assert.Equal(t, 0, s.len())
}

func TestEdgeKeyFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		traceID []byte
		spanID  []byte
		root    bool
	}{
		{
			name:    "valid IDs",
			traceID: make([]byte, maxTraceIDLen),
			spanID:  make([]byte, maxSpanIDLen),
		},
		{
			name:    "short IDs",
			traceID: []byte{0x01},
			spanID:  []byte{0x02},
		},
		{
			name:    "oversized IDs",
			traceID: make([]byte, maxTraceIDLen+1),
			spanID:  make([]byte, maxSpanIDLen+1),
		},
		{
			name:    "root span",
			traceID: []byte{0x01},
			root:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := edgeKeyFromBytes(tt.traceID, tt.spanID)
			assert.Equal(t, encodeKey(tt.traceID, tt.spanID), key.String())
			assert.Equal(t, tt.root, key.root)
		})
	}

	assert.NotEqual(t,
		edgeKeyFromBytes([]byte{0x01}, []byte{0x02}),
		edgeKeyFromBytes([]byte{0x01, 0x00}, []byte{0x02}),
		"ID lengths must participate in key equality",
	)

	traceID := []byte{0x01}
	key := edgeKeyFromBytes(traceID, []byte{0x02})
	traceID[0] = 0xff
	assert.Equal(t, edgeKeyFromBytes([]byte{0x01}, []byte{0x02}), key)
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
		key:                     edgeKeyFromString("test-key"),
		traceID:                 []byte("trace-123"),
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
	assert.Equal(t, edgeKey{}, e.key, "key should be reset")
	assert.Empty(t, e.traceID, "traceID should be reset (buffer retained, length zero)")
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

func TestStoreUpsertEdgeFromBytes_droppedCounterpart(t *testing.T) {
	s := NewStore(time.Hour, 10, noopCallback, noopCallback, newTestCounter())

	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := []byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8}

	s.AddDroppedSpanSideFromBytes(traceID, spanID, Server)

	isNew, err := s.UpsertEdgeFromBytes(traceID, spanID, Client, func(e *Edge) {
		e.ClientService = clientService
	})
	require.ErrorIs(t, err, ErrDroppedSpanSide)
	assert.True(t, isNew)
	assert.Equal(t, 0, s.len())
}

func TestStore_interiorEdgeRemovalKeepsListIntact(t *testing.T) {
	var onCompleteCount, onExpireCount int
	s := NewStore(time.Hour, 10, countingCallback(&onCompleteCount), countingCallback(&onExpireCount), newTestCounter())

	for _, key := range []string{"key-a", "key-b", "key-c"} {
		isNew, err := s.UpsertEdge(key, Client, func(e *Edge) {
			e.ClientService = clientService
		})
		require.NoError(t, err)
		require.True(t, isNew)
	}
	require.Equal(t, 3, s.len())

	// Complete the middle edge: interior removal, prev and next both non-nil.
	isNew, err := s.UpsertEdge("key-b", Server, func(e *Edge) {
		e.ServerService = "server"
	})
	require.NoError(t, err)
	require.False(t, isNew)
	require.Equal(t, 1, onCompleteCount)
	require.Equal(t, 2, s.len())

	require.Same(t, s.m[edgeKeyFromString("key-a")], s.head)
	require.Same(t, s.m[edgeKeyFromString("key-c")], s.tail)
	require.Same(t, s.tail, s.head.next)
	require.Same(t, s.head, s.tail.prev)

	// Expire only the head: Expire must evict key-a and stop at unexpired key-c.
	_, err = s.UpsertEdge("key-a", Client, func(e *Edge) {
		e.expiration = 0
	})
	require.NoError(t, err)
	s.Expire()
	require.Equal(t, 1, onExpireCount)
	require.Equal(t, 1, s.len())

	// The surviving tail must still be addressable and completable.
	isNew, err = s.UpsertEdge("key-c", Server, func(e *Edge) {
		e.ServerService = "server"
	})
	require.NoError(t, err)
	require.False(t, isNew)
	require.Equal(t, 2, onCompleteCount)
	require.Equal(t, 1, onExpireCount)
	require.Equal(t, 0, s.len())
}

func TestSetTraceID_copiesSourceAndReusesBuffer(t *testing.T) {
	e := &Edge{}

	src := []byte{0x01, 0x02, 0x03, 0x04}
	e.SetTraceID(src)
	src[0] = 0xFF
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, e.TraceID(), "mutating the source must not affect the edge's copy")

	bufPtr := &e.traceID[0]
	resetEdge(e)
	assert.Empty(t, e.TraceID())

	// A shorter ID after recycling must not expose stale bytes of the previous
	// ID and must reuse the retained backing array.
	e.SetTraceID([]byte{0xAA})
	assert.Equal(t, []byte{0xAA}, e.TraceID())
	assert.Same(t, bufPtr, &e.traceID[0], "backing array should be reused across recycles")
}
