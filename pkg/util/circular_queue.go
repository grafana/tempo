package util

import (
	"sync"
)

// TODO: Move to generics with Go 1.18

// CircularQueue is a thread-safe queue that can be used to store items.
// The queue is circular and will overwrite the oldest item when the
// maximum size is reached.
type CircularQueue struct {
	mutex sync.RWMutex

	buf  []interface{}
	head int // write index
	tail int // read index
	size int
}

func NewCircularQueue(size int) *CircularQueue {
	if size <= 0 {
		panic("size must be greater than 0")
	}

	return &CircularQueue{
		mutex: sync.RWMutex{},
		buf:   make([]interface{}, size+1),
		size:  size + 1,
	}
}

// Write writes an element to the circular queue.
// If the queue is full, it overwrites the oldest element.
func (cb *CircularQueue) Write(v interface{}) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.buf[cb.head] = v
	cb.head = (cb.head + 1) % cb.size
	if cb.head == cb.tail {
		cb.tail = (cb.tail + 1) % cb.size
	}
}

// Read reads an element from the circular queue.
// If the queue is empty, it returns nil.
func (cb *CircularQueue) Read() interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.read()
}

// ReadAll reads all elements from the circular queue.
func (cb *CircularQueue) ReadAll() []interface{} {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	values := make([]interface{}, 0, cb.len())
	for v := cb.read(); v != nil; v = cb.read() {
		values = append(values, v)
	}

	return values
}

// read returns the oldest element from the circular queue.
// If the queue is empty, it returns nil.
// After a read, the tail is moved forward.
//
// This method is not thread-safe.
func (cb *CircularQueue) read() interface{} {
	if cb.head == cb.tail {
		return nil
	}
	v := cb.buf[cb.tail]
	cb.tail = (cb.tail + 1) % cb.size
	return v
}

// Peek reads the oldest element from the circular queue,
// but does not move the tail forward.
func (cb *CircularQueue) Peek() interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.peek()
}

func (cb *CircularQueue) peek() interface{} {
	if cb.head == cb.tail {
		return nil
	}
	return cb.buf[cb.tail]
}

// Len returns the number of elements in the queue.
func (cb *CircularQueue) Len() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.len()
}

func (cb *CircularQueue) len() int {
	if cb.head >= cb.tail {
		return cb.head - cb.tail
	}
	return cb.size - cb.tail + cb.head
}

func (cb *CircularQueue) CanRead() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.head != cb.tail
}
