package flushqueues

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOp struct {
	key string
}

func (m mockOp) Key() string {
	return m.key
}

func (m mockOp) Priority() int64 {
	return 0
}

func TestExclusiveQueues(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "testersons",
	})

	q := New[mockOp](gauge)
	op := mockOp{
		key: "not unique",
	}

	// enqueue twice
	err := q.Enqueue(op)
	assert.NoError(t, err)

	length, err := test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))

	err = q.Enqueue(op)
	assert.NoError(t, err)

	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))

	// dequeue -> requeue
	_ = q.Dequeue()
	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(length))

	err = q.Requeue(op)
	assert.NoError(t, err)

	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))

	// dequeue -> clearkey -> enqueue
	_ = q.Dequeue()
	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(length))

	q.Clear(op)
	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(length))

	err = q.Enqueue(op)
	assert.NoError(t, err)

	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))
}

func TestMultipleItems(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "testersons",
	})

	totalItems := 10
	q := New[mockOp](gauge)

	// add stuff to the queue and confirm the length matches expected
	for i := 0; i < totalItems; i++ {
		op := mockOp{
			key: uuid.New().String(),
		}

		err := q.Enqueue(op)
		assert.NoError(t, err)

		length, err := test.GetGaugeValue(gauge)
		assert.NoError(t, err)
		assert.Equal(t, i+1, int(length))
	}

	// dequeue all items
	for i := 0; i < totalItems; i++ {
		op := q.Dequeue()
		assert.NotNil(t, op)

		length, err := test.GetGaugeValue(gauge)
		assert.NoError(t, err)
		assert.Equal(t, totalItems-(i+1), int(length))
	}
}

func TestExclusiveQueueLocks(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "testersons",
	})

	queue := New[simpleItem](gauge)
	job := simpleItem(1)

	assert.NoError(t, queue.Enqueue(job))
	i := queue.Dequeue()
	assert.Equal(t, job, i, "Expected to dequeue job")

	// Requeueing the same job again will be added to the queue
	assert.NoError(t, queue.Requeue(job))
	require.True(t, waitForDequeue(queue), "Requeue didn't unblock Dequeue")

	// Clearing the key also allows to queue and dequeue the job again
	queue.Clear(job)
	assert.NoError(t, queue.Enqueue(job))
	require.True(t, waitForDequeue(queue), "Enqueue didn't unblock Dequeue")

	// However, enqueuing the same job without clearing the key again will be dropped and not added to the queue
	assert.NoError(t, queue.Enqueue(job))
	require.False(t, waitForDequeue(queue), "Enqueue unlocked Dequeue")
}

func waitForDequeue(queue *ExclusiveQueues[simpleItem]) bool {
	done := make(chan struct{})
	go func() {
		queue.Dequeue()
		done <- struct{}{}
	}()
	select {
	case <-done:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

// TestConcurrentDequeue verifies that multiple goroutines calling Dequeue
// concurrently on a shared queue process every item exactly once.
func TestConcurrentDequeue(t *testing.T) {
	q := New[mockOp](nil)

	totalItems := 500
	numWorkers := 10

	// Enqueue all items up front
	keys := make([]string, totalItems)
	for i := range totalItems {
		keys[i] = uuid.New().String()
		require.NoError(t, q.Enqueue(mockOp{key: keys[i]}))
	}

	// Track which keys each worker dequeued
	var mu sync.Mutex
	seen := make(map[string]int) // key -> count

	// itemsDone tracks when all items have been processed
	var itemsDone sync.WaitGroup
	itemsDone.Add(totalItems)

	var workers sync.WaitGroup
	for range numWorkers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				op := q.Dequeue()
				if op.Key() == "" {
					return // queue closed
				}
				mu.Lock()
				seen[op.Key()]++
				mu.Unlock()
				q.Clear(op)
				itemsDone.Done()
			}
		}()
	}

	// Wait for all items to be processed, then shut down workers
	itemsDone.Wait()
	q.Stop()
	workers.Wait()

	// Every key must have been dequeued exactly once
	for _, key := range keys {
		assert.Equal(t, 1, seen[key], "key %s dequeued %d times", key, seen[key])
	}
	assert.Equal(t, totalItems, len(seen), "expected %d unique keys, got %d", totalItems, len(seen))
}

// TestStopUnblocksAllWaiters verifies that calling Stop on an empty queue
// unblocks all goroutines waiting in Dequeue, returning zero values.
func TestStopUnblocksAllWaiters(t *testing.T) {
	q := New[mockOp](nil)
	numWorkers := 5

	var allStarted sync.WaitGroup
	allStarted.Add(numWorkers)

	var workers sync.WaitGroup
	for i := range numWorkers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			allStarted.Done()
			op := q.Dequeue()
			assert.Equal(t, mockOp{}, op, "worker %d: expected zero value from closed queue", i)
		}()
	}

	allStarted.Wait()
	q.Stop()
	workers.Wait()
}
