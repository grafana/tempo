package util

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircularQueueWrite(t *testing.T) {
	cb := NewCircularQueue(10)
	for i := 0; i < 10; i++ {
		cb.Write(i)
	}

	require.Equal(t, 10, cb.Len())
}

func TestCircularQueueEvict(t *testing.T) {
	// Write 5 items with a capacity of 5
	// First 5 items [0,4] will be overwritten
	cb := NewCircularQueue(5)
	for i := 0; i < 10; i++ {
		cb.Write(i)
	}

	for i := 5; i < 10; i++ {
		v := cb.Read()
		assert.Equal(t, i, v)
	}

	require.Equal(t, 0, cb.Len())
}

func TestCircularQueueSafeConcurrentAccess(t *testing.T) {
	cb := NewCircularQueue(5)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				cb.Write(i)
			}
		}()
	}

	wg.Wait()

	require.Equal(t, 5, cb.Len())
}

type queueEntry struct {
	key   string
	value string
}

func BenchmarkCircularQueueWrite(b *testing.B) {
	cb := NewCircularQueue(10)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cb.Write(&queueEntry{
			key:   "hello",
			value: "world",
		})
	}
}
