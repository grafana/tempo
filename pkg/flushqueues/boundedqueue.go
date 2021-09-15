// Copyright (c) 2018 Christian R. Petrin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package queueimpl7 implements an unbounded, dynamically growing FIFO queue.
// Internally, queue store the values in fixed sized slices that are linked using
// a singly linked list.
// This implementation tests the queue performance when performing lazy creation of
// the internal slice as well as starting with a 1 sized slice, allowing it to grow
// up to 16 by using the builtin append function. Subsequent slices are created with
// 128 fixed size.
package flushqueues

import "sync"

// Keeping below as var so it is possible to run the slice size bench tests with no coding changes.
var (
	// firstSliceSize holds the size of the first slice.
	firstSliceSize = 1

	// maxFirstSliceSize holds the maximum size of the first slice.
	maxFirstSliceSize = 16

	// maxInternalSliceSize holds the maximum size of each internal slice.
	maxInternalSliceSize = 128
)

// Queueimpl7 represents an unbounded, dynamically growing FIFO queue.
// The zero value for queue is an empty queue ready to use.
type Queueimpl7 struct {
	// Head points to the first node of the linked list.
	head *Node

	// Tail points to the last node of the linked list.
	// In an empty queue, head and tail points to the same node.
	tail *Node

	// Hp is the index pointing to the current first element in the queue
	// (i.e. first element added in the current queue values).
	hp int

	// Len holds the current queue values length.
	len int

	// lastSliceSize holds the size of the last created internal slice.
	lastSliceSize int

	mtx sync.Mutex
}

// Node represents a queue node.
// Each node holds a slice of user managed values.
type Node struct {
	// v holds the list of user added values in this node.
	v []interface{}

	// n points to the next node in the linked list.
	n *Node
}

// New returns an initialized queue.
func NewQ() *Queueimpl7 {
	return new(Queueimpl7).Init()
}

// Init initializes or clears queue q.
func (q *Queueimpl7) Init() *Queueimpl7 {
	q.head = nil
	q.tail = nil
	q.hp = 0
	q.len = 0
	return q
}

// Len returns the number of elements of queue q.
// The complexity is O(1).
func (q *Queueimpl7) Len() int {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	return q.len
}

// Front returns the first element of queue q or nil if the queue is empty.
// The second, bool result indicates whether a valid value was returned;
//   if the queue is empty, false will be returned.
// The complexity is O(1).
func (q *Queueimpl7) Front() (interface{}, bool) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if q.head == nil {
		return nil, false
	}
	return q.head.v[q.hp], true
}

// Push adds a value to the queue.
// The complexity is O(1).
func (q *Queueimpl7) Push(v interface{}) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if q.head == nil {
		h := newNode(firstSliceSize)
		q.head = h
		q.tail = h
		q.lastSliceSize = maxFirstSliceSize
	} else if len(q.tail.v) >= q.lastSliceSize {
		n := newNode(maxInternalSliceSize)
		q.tail.n = n
		q.tail = n
		q.lastSliceSize = maxInternalSliceSize
	}

	q.tail.v = append(q.tail.v, v)
	q.len++
}

// Pop retrieves and removes the current element from the queue.
// The second, bool result indicates whether a valid value was returned;
// 	if the queue is empty, false will be returned.
// The complexity is O(1).
func (q *Queueimpl7) Pop() (interface{}, bool) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if q.head == nil {
		return nil, false
	}

	v := q.head.v[q.hp]
	q.head.v[q.hp] = nil // Avoid memory leaks
	q.len--
	q.hp++
	if q.hp >= len(q.head.v) {
		n := q.head.n
		q.head.n = nil // Avoid memory leaks
		q.head = n
		q.hp = 0
	}
	return v, true
}

// newNode returns an initialized node.
func newNode(capacity int) *Node {
	return &Node{
		v: make([]interface{}, 0, capacity),
	}
}
