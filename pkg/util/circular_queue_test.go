package util

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircularQueueWrite(t *testing.T) {
	var overwrites int
	cb := NewCircularQueue(10, func() {
		overwrites++
	})
	for i := 0; i < 10; i++ {
		cb.Write(i)
	}

	require.Equal(t, 10, cb.Len())
	require.Equal(t, 0, overwrites)
}

func TestCircularQueueOverwrite(t *testing.T) {
	var overwrites int
	// Write 5 items with a capacity of 5
	// First 5 items [0,4] will be overwritten
	cb := NewCircularQueue(5, func() {
		overwrites++
	})
	for i := 0; i < 10; i++ {
		cb.Write(i)
	}

	for i := 5; i < 10; i++ {
		v := cb.Read()
		assert.Equal(t, i, v)
	}

	require.Equal(t, 0, cb.Len())
	require.Equal(t, 5, overwrites)
}

func TestCircularQueueSafeConcurrentAccess(t *testing.T) {
	var overwrites int
	cb := NewCircularQueue(5, func() {
		overwrites++
	})

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
	require.Equal(t, 9995, overwrites)
}

type queueEntry struct {
	key   string
	value string
}

func BenchmarkCircularQueueWrite(b *testing.B) {
	cb := NewCircularQueue(10, func() {})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cb.Write(&queueEntry{
			key:   "hello",
			value: "world",
		})
	}
}
