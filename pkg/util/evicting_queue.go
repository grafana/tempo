package util

import (
	"context"
	"errors"
	"sync"
	"time"
)

type EvictingQueue struct {
	sync.RWMutex

	capacity int
	entries  []interface{}
	onEvict  func()

	subscribers []chan interface{}
}

func NewEvictingQueue(capacity int, interval time.Duration, onEvict func()) (*EvictingQueue, error) {
	if err := validateCapacity(capacity); err != nil {
		return nil, err
	}

	queue := &EvictingQueue{
		onEvict:     onEvict,
		entries:     make([]interface{}, 0, capacity),
		subscribers: make([]chan interface{}, 0),
	}

	err := queue.SetCapacity(capacity)
	if err != nil {
		return nil, err
	}

	if interval > 0 {
		go queue.loop(interval)
	}

	return queue, nil
}

// Append adds an entry to the end of the queue.
// If the queue is full, the oldest entry is removed from the queue.
func (q *EvictingQueue) Append(entry interface{}) {
	q.Lock()
	defer q.Unlock()

	if len(q.entries) >= q.capacity {
		q.evictOldest()
	}

	q.entries = append(q.entries, entry)
}

// Subscribe returns a channel that will receive entries as they are added to the queue.
func (q *EvictingQueue) Subscribe() chan interface{} {
	q.Lock()
	defer q.Unlock()

	sub := make(chan interface{})
	q.subscribers = append(q.subscribers, sub)

	return sub
}

// evictOldest removes the oldest entry from the queue.
func (q *EvictingQueue) evictOldest() {
	q.onEvict()

	start := (len(q.entries) - q.Capacity()) + 1
	q.entries = append(q.entries[:0], q.entries[start:]...)
}

// Pop removes and returns the oldest item in the queue.
func (q *EvictingQueue) Pop() interface{} {
	q.Lock()
	defer q.Unlock()

	return q.pop()
}

// Pop removes and returns the oldest item in the queue.
// Must be called under lock
func (q *EvictingQueue) pop() interface{} {
	if len(q.entries) == 0 {
		return nil
	}

	e := q.entries[0]
	q.entries = append(q.entries[:0], q.entries[1:]...)

	return e
}

// isEmpty returns true if the queue is empty.
// Must be called under lock.
func (q *EvictingQueue) isEmpty() bool {
	return q.length() == 0
}

// Entries returns a copy of the queue's entries.
func (q *EvictingQueue) Entries() []interface{} {
	q.RLock()
	defer q.RUnlock()

	return q.entries
}

// Peek returns the oldest item in the queue, or nil if the queue is empty.
func (q *EvictingQueue) Peek() interface{} {
	q.RLock()
	defer q.RUnlock()

	if len(q.entries) == 0 {
		return nil
	}

	return q.entries[0]
}

func (q *EvictingQueue) Length() int {
	q.RLock()
	defer q.RUnlock()

	return q.length()
}

func (q *EvictingQueue) length() int {
	return len(q.entries)
}

// Capacity returns the capacity of the queue.
func (q *EvictingQueue) Capacity() int {
	return q.capacity
}

// SetCapacity changes the capacity of the queue.
func (q *EvictingQueue) SetCapacity(capacity int) error {
	if err := validateCapacity(capacity); err != nil {
		return err
	}

	q.capacity = capacity
	return nil
}

// Clear removes all entries from the queue.
func (q *EvictingQueue) Clear() {
	q.Lock()
	defer q.Unlock()

	q.entries = q.entries[:0]
}

func (q *EvictingQueue) loop(interval time.Duration) {
	t := time.NewTicker(interval)
	// TODO: add a way to stop the loop
	for {
		select {
		case <-t.C:
			// TODO: add a way to return early from the fan-out loop
			// 	maybe by having a context that can be cancelled
			q.fanOut(context.Background())
		}
	}

}

// fanOut sends all entries to all subscribers.
func (q *EvictingQueue) fanOut(ctx context.Context) {
	q.Lock()
	defer q.Unlock()

	for _, sub := range q.subscribers {
		for !q.isEmpty() {
			select {
			case sub <- q.pop():
			case <-ctx.Done():
				return
			}
		}
	}
}

// validateCapacity returns an error if the capacity is not positive.
func validateCapacity(capacity int) error {
	if capacity <= 0 {
		// a queue of 0 (or smaller) capacity is invalid
		return errors.New("queue cannot have a zero or negative capacity")
	}

	return nil
}
