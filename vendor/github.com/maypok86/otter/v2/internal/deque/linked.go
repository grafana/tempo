// Copyright (c) 2024 Alexey Mayshev and contributors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deque

import (
	"iter"
	"unsafe"

	"github.com/maypok86/otter/v2/internal/generated/node"
)

// Linked is an intrusive doubly-linked list of nodes. It follows the nodes' eviction policy links
// through raw pointers at the offset the Kind reports, so linking and unlinking do not call the
// nodes' methods; a node.Node is made only for the nodes it hands out.
type Linked struct {
	head   unsafe.Pointer
	tail   unsafe.Pointer
	len    int
	offset uintptr
	kind   *node.Kind
}

func NewLinked(kind *node.Kind) *Linked {
	return &Linked{
		offset: kind.LinksOffset(),
		kind:   kind,
	}
}

func (d *Linked) links(p unsafe.Pointer) *node.Links {
	return (*node.Links)(unsafe.Add(p, d.offset))
}

func (d *Linked) PushBack(n node.Node) {
	d.pushBack(node.PointerOf(n))
}

func (d *Linked) pushBack(p unsafe.Pointer) {
	if d.IsEmpty() {
		d.head = p
		d.tail = p
	} else {
		d.links(p).Prev = d.tail
		d.links(d.tail).Next = p
		d.tail = p
	}

	d.len++
}

func (d *Linked) UpdateNode(n, old node.Node) {
	p := node.PointerOf(n)
	o := node.PointerOf(old)
	nl := d.links(p)
	ol := d.links(o)

	if ol.Next == nil {
		if d.tail == o {
			d.tail = p
		}
	} else {
		d.links(ol.Next).Prev = p
		nl.Next = ol.Next
		ol.Next = nil
	}

	if ol.Prev == nil {
		if d.head == o {
			d.head = p
		}
	} else {
		d.links(ol.Prev).Next = p
		nl.Prev = ol.Prev
		ol.Prev = nil
	}
}

func (d *Linked) PushFront(n node.Node) {
	d.pushFront(node.PointerOf(n))
}

func (d *Linked) pushFront(p unsafe.Pointer) {
	if d.IsEmpty() {
		d.head = p
		d.tail = p
	} else {
		d.links(p).Next = d.head
		d.links(d.head).Prev = p
		d.head = p
	}

	d.len++
}

func (d *Linked) PopFront() node.Node {
	if d.IsEmpty() {
		return nil
	}

	result := d.head
	d.delete(result)
	return d.kind.FromPointer(result)
}

func (d *Linked) NotContains(n node.Node) bool {
	return !d.Contains(n)
}

func (d *Linked) Contains(n node.Node) bool {
	p := node.PointerOf(n)
	l := d.links(p)
	return l.Prev != nil || l.Next != nil || d.head == p
}

func (d *Linked) MoveToBack(n node.Node) {
	p := node.PointerOf(n)
	if p != d.tail {
		d.delete(p)
		d.pushBack(p)
	}
}

func (d *Linked) MoveToFront(n node.Node) {
	p := node.PointerOf(n)
	if p != d.head {
		d.delete(p)
		d.pushFront(p)
	}
}

func (d *Linked) Delete(n node.Node) {
	d.delete(node.PointerOf(n))
}

func (d *Linked) delete(p unsafe.Pointer) {
	l := d.links(p)
	next := l.Next
	prev := l.Prev

	if prev == nil {
		if next == nil && d.head != p {
			return
		}

		d.head = next
	} else {
		d.links(prev).Next = next
		l.Prev = nil
	}

	if next == nil {
		d.tail = prev
	} else {
		d.links(next).Prev = prev
		l.Next = nil
	}

	d.len--
}

func (d *Linked) Clear() {
	for !d.IsEmpty() {
		d.delete(d.head)
	}
}

func (d *Linked) Len() int {
	return d.len
}

func (d *Linked) IsEmpty() bool {
	return d.Len() == 0
}

func (d *Linked) Head() node.Node {
	return d.kind.FromPointer(d.head)
}

func (d *Linked) Tail() node.Node {
	return d.kind.FromPointer(d.tail)
}

func (d *Linked) All() iter.Seq[node.Node] {
	return func(yield func(node.Node) bool) {
		cursor := d.head
		for cursor != nil {
			if !yield(d.kind.FromPointer(cursor)) {
				return
			}
			cursor = d.links(cursor).Next
		}
	}
}

func (d *Linked) Backward() iter.Seq[node.Node] {
	return func(yield func(node.Node) bool) {
		cursor := d.tail
		for cursor != nil {
			if !yield(d.kind.FromPointer(cursor)) {
				return
			}
			cursor = d.links(cursor).Prev
		}
	}
}
