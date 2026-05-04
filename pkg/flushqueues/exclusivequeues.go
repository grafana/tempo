package flushqueues

import (
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

type ExclusiveQueues[T Op] struct {
	queue      *PriorityQueue[T]
	activeKeys sync.Map
	stopped    atomic.Bool
}

// New creates a new set of flush queues with a prom gauge to track current depth
func New[T Op](metric prometheus.Gauge) *ExclusiveQueues[T] {
	return &ExclusiveQueues[T]{
		queue: NewPriorityQueue[T](metric),
	}
}

// Enqueue adds the op to the queue and prevents any other items to be added with this key
func (f *ExclusiveQueues[T]) Enqueue(op T) error {
	_, loaded := f.activeKeys.LoadOrStore(op.Key(), struct{}{})
	if loaded {
		return nil
	}

	return f.Requeue(op)
}

// Dequeue removes the next op from the queue. After dequeueing the calling
// process either needs to call Clear or Requeue.
// Multiple goroutines can call Dequeue concurrently; the underlying PriorityQueue
// serializes access and blocks each caller until an item is available.
func (f *ExclusiveQueues[T]) Dequeue() T {
	return f.queue.Dequeue()
}

// Requeue adds an op that is presumed to already be covered by activeKeys
func (f *ExclusiveQueues[T]) Requeue(op T) error {
	_, err := f.queue.Enqueue(op)
	return err
}

// Clear unblocks the requested op.  This should be called only after a flush has been successful
func (f *ExclusiveQueues[T]) Clear(op T) {
	f.activeKeys.Delete(op.Key())
}

func (f *ExclusiveQueues[T]) IsEmpty() bool {
	length := 0

	f.activeKeys.Range(func(_, _ interface{}) bool {
		length++
		return false
	})

	return length <= 0
}

// Stop closes the queue
func (f *ExclusiveQueues[T]) Stop() {
	f.stopped.Store(true)
	f.queue.Close()
}

func (f *ExclusiveQueues[T]) IsStopped() bool {
	return f.stopped.Load()
}
