package flushqueues

import (
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type simpleItem int64

func (i simpleItem) Priority() int64 {
	return int64(i)
}

func (i simpleItem) Key() string {
	return strconv.FormatInt(int64(i), 10)
}

func TestPriorityQueueBasic(t *testing.T) {
	queue := NewPriorityQueue[simpleItem](nil)
	assert.Equal(t, 0, queue.Length(), "Expected length = 0")

	_, err := queue.Enqueue(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, queue.Length(), "Expected length = 1")

	i := queue.Dequeue()
	assert.Equal(t, simpleItem(1), i, "Expected to dequeue simpleItem(1)")

	queue.Close()
	assert.Zero(t, queue.Dequeue(), "Expect zero value dequeue")
}

func TestPriorityQueuePriorities(t *testing.T) {
	queue := NewPriorityQueue[simpleItem](nil)

	_, err := queue.Enqueue(1)
	assert.NoError(t, err)

	_, err = queue.Enqueue(2)
	assert.NoError(t, err)

	assert.Equal(t, simpleItem(2), queue.Dequeue(), "Expected to dequeue simpleItem(2)")
	assert.Equal(t, simpleItem(1), queue.Dequeue(), "Expected to dequeue simpleItem(1)")

	queue.Close()
	assert.Zero(t, queue.Dequeue(), "Expect zero value dequeue")
}

func TestPriorityQueuePriorities2(t *testing.T) {
	queue := NewPriorityQueue[simpleItem](nil)

	_, err := queue.Enqueue(2)
	assert.NoError(t, err)

	_, err = queue.Enqueue(1)
	assert.NoError(t, err)

	assert.Equal(t, simpleItem(2), queue.Dequeue(), "Expected to dequeue simpleItem(2)")
	assert.Equal(t, simpleItem(1), queue.Dequeue(), "Expected to dequeue simpleItem(1)")

	queue.Close()
	assert.Zero(t, queue.Dequeue(), "Expect zero value dequeue")
}

func TestPriorityQueueWait(t *testing.T) {
	queue := NewPriorityQueue[simpleItem](nil)

	done := make(chan struct{})
	go func() {
		assert.Zero(t, queue.Dequeue(), "Expect zero value dequeue")
		close(done)
	}()

	queue.Close()
	runtime.Gosched()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Close didn't unblock Dequeue.")
	}
}
