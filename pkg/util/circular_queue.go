package util

import (
	"sync"
)

type CircularQueue interface {
	Write(interface{})
	Read() interface{}
	ReadAll() []interface{}
	Peek() interface{}
	Len() int
}

type circularQueue struct {
	mutex sync.RWMutex

	buf  []interface{}
	head int // write index
	tail int // read index
	size int
}

func NewCircularQueue(size int) CircularQueue {
	if size <= 0 {
		panic("size must be greater than 0")
	}

	return &circularQueue{
		mutex: sync.RWMutex{},
		buf:   make([]interface{}, size+1),
		size:  size + 1,
	}
}

// Write writes an element to the circular queue.
// If the queue is full, it overwrites the oldest element.
func (cb *circularQueue) Write(v interface{}) {
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
func (cb *circularQueue) Read() interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.read()
}

// ReadAll reads all elements from the circular queue.
func (cb *circularQueue) ReadAll() []interface{} {
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
func (cb *circularQueue) read() interface{} {
	if cb.head == cb.tail {
		return nil
	}
	v := cb.buf[cb.tail]
	cb.tail = (cb.tail + 1) % cb.size
	return v
}

// Peek reads the oldest element from the circular queue,
// but does not move the tail forward.
func (cb *circularQueue) Peek() interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.peek()
}

func (cb *circularQueue) peek() interface{} {
	if cb.head == cb.tail {
		return nil
	}
	return cb.buf[cb.tail]
}

// Len returns the number of elements in the queue.
func (cb *circularQueue) Len() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.len()
}

func (cb *circularQueue) len() int {
	if cb.head >= cb.tail {
		return cb.head - cb.tail
	}
	return cb.size - cb.tail + cb.head
}
