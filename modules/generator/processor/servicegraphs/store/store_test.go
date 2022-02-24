package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var noopUpsertCb Callback = func(e *Edge) {}

func TestStore_UpsertEdge(t *testing.T) {
	const keyStr = "key"

	var cbCallCount int
	storeInterface := NewStore(time.Hour, 1, func(e *Edge) {
		cbCallCount++
	})
	s := storeInterface.(*store)
	assert.Equal(t, 0, s.len())

	_, err := s.UpsertEdge(keyStr, func(e *Edge) {})
	require.NoError(t, err)
	assert.Equal(t, 1, s.len())
	assert.False(t, s.shouldEvictHead()) // ttl is set to 1h
	assert.Equal(t, 0, cbCallCount)

	e := getEdge(s, keyStr)
	assert.NotNil(t, e)
	assert.Equal(t, keyStr, e.key)

	_, err = s.UpsertEdge(keyStr+keyStr, func(e *Edge) {})
	assert.Error(t, err)

	_, err = s.UpsertEdge(keyStr, func(e *Edge) {
		e.ClientService = "client"
		e.ServerService = "server"
		e.expiration = 0 // Expire immediately
	})
	require.NoError(t, err)
	assert.Equal(t, 0, cbCallCount)

	e = getEdge(s, keyStr)
	assert.NotNil(t, e)
	assert.Equal(t, "client", e.ClientService)
	assert.Equal(t, "server", e.ServerService)
	assert.True(t, s.shouldEvictHead())

	s.evictHead()
	assert.Equal(t, 0, s.len())
	assert.Equal(t, 1, cbCallCount)
}

func TestStore_expire(t *testing.T) {
	keys := map[string]bool{}
	for i := 0; i < 100; i++ {
		keys[fmt.Sprintf("key-%d", i)] = true
	}

	// all new keys are immediately expired.
	storeInterface := NewStore(-time.Second, 100, func(e *Edge) {
		assert.True(t, keys[e.key])
	})
	s := storeInterface.(*store)

	for key := range keys {
		_, err := s.UpsertEdge(key, noopUpsertCb)
		require.NoError(t, err)
	}

	s.Expire()
	assert.Equal(t, 0, s.len())
}

// TODO add test for maxItems
// tODO add test to verify concurrency

func getEdge(s *store, k string) *Edge {
	ele, ok := s.m[k]
	if !ok {
		return nil
	}
	return ele.Value.(*Edge)
}
