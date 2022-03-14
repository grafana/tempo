package util

import (
	"context"
	"errors"
	"sync"
)

type EvictingQueue struct {
	sync.RWMutex
	closeCh chan struct{}

	capacity int
	entries  []interface{}
	onEvict  func()

	outCh chan interface{}
}

func NewEvictingQueue(capacity int, onEvict func()) (*EvictingQueue, error) {
	if err := validateCapacity(capacity); err != nil {
		return nil, err
	}

	queue := &EvictingQueue{
		closeCh: make(chan struct{}),
		onEvict: onEvict,
		entries: make([]interface{}, 0, capacity),
		outCh:   make(chan interface{}),
	}

	err := queue.SetCapacity(capacity)
	if err != nil {
		return nil, err
	}

	return queue, nil
}

func (q *EvictingQueue) Close() {
	q.Lock()
	defer q.Unlock()

	q.onEvict = nil
	q.capacity = 0
	q.entries = nil
	close(q.closeCh)
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

// Out returns a channel which will emit the queue's entries as they are
// removed from the queue.
func (q *EvictingQueue) Out(ctx context.Context) <-chan interface{} {
	q.Lock()

	// TODO: Creating a new channel each time is not efficient.
	//  We should be able to use the same channel.
	outCh := make(chan interface{})
	go func() {
		defer q.Unlock()
		defer close(outCh)
		for {
			select {
			case <-ctx.Done():
				return
			case <-q.closeCh:
				return
			default:
				e := q.pop()
				if e == nil {
					return
				}
				outCh <- e
			}
		}
	}()

	return outCh
}

// validateCapacity returns an error if the capacity is not positive.
func validateCapacity(capacity int) error {
	if capacity <= 0 {
		// a queue of 0 (or smaller) capacity is invalid
		return errors.New("queue cannot have a zero or negative capacity")
	}

	return nil
}
