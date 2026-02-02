package flushqueues

import (
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

	q := New[mockOp](1, gauge)
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
	_ = q.Dequeue(0)
	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(length))

	err = q.Requeue(op)
	assert.NoError(t, err)

	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))

	// dequeue -> clearkey -> enqueue
	_ = q.Dequeue(0)
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

func TestMultipleQueues(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "testersons",
	})

	totalQueues := 10
	totalItems := 10
	q := New[mockOp](totalQueues, gauge)

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

	// each queue should have 1 thing
	for i := 0; i < totalQueues; i++ {
		op := q.Dequeue(i)
		assert.NotNil(t, op)

		length, err := test.GetGaugeValue(gauge)
		assert.NoError(t, err)
		assert.Equal(t, totalQueues-(i+1), int(length))
	}
}

func TestExclusiveQueueLocks(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "testersons",
	})

	queue := New[simpleItem](1, gauge)
	job := simpleItem(1)

	assert.NoError(t, queue.Enqueue(job))
	i := queue.Dequeue(0)
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
		queue.Dequeue(0)
		done <- struct{}{}
	}()
	select {
	case <-done:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}
