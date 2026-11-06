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
//
// This is a port of lossy buffers from Caffeine.
// https://github.com/ben-manes/caffeine/blob/master/caffeine/src/main/java/com/github/benmanes/caffeine/cache/StripedBuffer.java

package lossy

import (
	"sync/atomic"
	"unsafe"

	"github.com/maypok86/otter/v2/internal/generated/node"
	"github.com/maypok86/otter/v2/internal/xruntime"
)

const (
	attempts = 3
)

// striped is the table of stripes. Every Add reads it, so it is padded on both sides: otherwise
// it would share a cache line with whatever the allocator put next to it, and a neighbor that is
// written often would make every Add miss the cache.
type striped struct {
	_       [xruntime.CacheLineSize]byte
	buffers []atomic.Pointer[ring]
	len     int
	_       [xruntime.CacheLineSize]byte
}

// newBuffers returns the stripes of a table of length n. They are allocated with a cache line
// of padding on each side for the same reason as the table itself.
func newBuffers(n int) []atomic.Pointer[ring] {
	pad := int(xruntime.CacheLineSize / unsafe.Sizeof(atomic.Pointer[ring]{}))
	buffers := make([]atomic.Pointer[ring], n+2*pad)
	return buffers[pad : pad+n : pad+n]
}

// Striped is a multiple-producer / single-consumer buffer that rejects new elements if it is full or
// fails spuriously due to contention. Unlike a queue and stack, a buffer does not guarantee an
// ordering of elements in either FIFO or LIFO order.
type Striped struct {
	// Every Add loads the table, so the fields are padded like the table itself.
	_       [xruntime.CacheLineSize]byte
	kind    *node.Kind
	maxLen  int
	striped atomic.Pointer[striped]
	busy    atomic.Uint32
	_       [xruntime.CacheLineSize]byte
}

func NewStriped(maxLen int, kind *node.Kind) *Striped {
	return &Striped{
		kind:   kind,
		maxLen: maxLen,
	}
}

// Add inserts the specified element into this buffer if it is possible to do so immediately without
// violating capacity restrictions. The addition is allowed to fail spuriously if multiple
// goroutines insert concurrently.
func (s *Striped) Add(n node.Node) Status {
	// A goroutine keeps adding to the same stripe, while goroutines running at the same
	// time mostly add to different ones.
	h := xruntime.StackHash()

	bs := s.striped.Load()
	if bs == nil {
		return s.expandOrRetry(n, h, true)
	}

	//nolint:gosec // len will never overflow uint32
	buffer := bs.buffers[h&uint32(bs.len-1)].Load()
	if buffer == nil {
		return s.expandOrRetry(n, h, true)
	}

	result := buffer.add(n)
	if result == Failed {
		return s.expandOrRetry(n, h, false)
	}

	return result
}

func (s *Striped) expandOrRetry(n node.Node, h uint32, wasUncontended bool) Status {
	result := Failed
	// True if the CAS also failed on another stripe. It starts false, so the table grows
	// only when the contention shows up again after a rehash, not on a single collision.
	collide := false

	for attempt := 0; attempt < attempts; attempt++ {
		bs := s.striped.Load()
		if bs != nil && bs.len > 0 {
			//nolint:gosec // len will never overflow uint32
			buffer := bs.buffers[h&uint32(bs.len-1)].Load()
			//nolint:gocritic // the switch statement looks even worse here
			if buffer == nil {
				if s.busy.Load() == 0 && s.busy.CompareAndSwap(0, 1) {
					// Try to attach new buffer.
					created := false
					rs := s.striped.Load()
					if rs != nil && rs.len > 0 {
						// Recheck under lock.
						//nolint:gosec // len will never overflow uint32
						j := h & uint32(rs.len-1)
						if rs.buffers[j].Load() == nil {
							rs.buffers[j].Store(newRing(s.kind, n))
							created = true
						}
					}
					s.busy.Store(0)
					if created {
						result = Success
						break
					}
					// Slot is now non-empty.
					continue
				}
				collide = false
			} else if !wasUncontended {
				// CAS already known to fail.
				// Continue after rehash.
				wasUncontended = true
			} else {
				result = buffer.add(n)
				//nolint:gocritic // the switch statement looks even worse here
				if result != Failed {
					break
				} else if bs.len >= s.maxLen || s.striped.Load() != bs {
					// At max size or stale.
					collide = false
				} else if !collide {
					collide = true
				} else if s.busy.Load() == 0 && s.busy.CompareAndSwap(0, 1) {
					if s.striped.Load() == bs {
						length := bs.len << 1
						striped := &striped{
							buffers: newBuffers(length),
							len:     length,
						}
						for j := 0; j < bs.len; j++ {
							striped.buffers[j].Store(bs.buffers[j].Load())
						}
						s.striped.Store(striped)
					}
					s.busy.Store(0)
					collide = false
					continue
				}
			}
			h = xruntime.Fastrand()
		} else if s.busy.Load() == 0 && s.striped.Load() == bs && s.busy.CompareAndSwap(0, 1) {
			init := false
			if s.striped.Load() == bs {
				striped := &striped{
					buffers: newBuffers(1),
					len:     1,
				}
				striped.buffers[0].Store(newRing(s.kind, n))
				s.striped.Store(striped)
				init = true
			}
			s.busy.Store(0)
			if init {
				result = Success
				break
			}
		}
	}

	return result
}

// DrainTo drains the buffer, sending each element to the consumer for processing. The caller must ensure
// that a consumer has exclusive read access to the buffer.
func (s *Striped) DrainTo(consumer func(n node.Node)) {
	bs := s.striped.Load()
	if bs == nil {
		return
	}
	for i := 0; i < bs.len; i++ {
		b := bs.buffers[i].Load()
		if b != nil {
			b.drainTo(consumer)
		}
	}
}

func (s *Striped) Len() int {
	result := 0
	bs := s.striped.Load()
	if bs == nil {
		return result
	}
	for i := 0; i < bs.len; i++ {
		b := bs.buffers[i].Load()
		if b == nil {
			continue
		}
		result += b.len()
	}
	return result
}

/*
func (s *Striped) Clear() {
	bs := s.striped.Load()
	if bs == nil {
		return
	}
	for s.busy.Load() != 0 || !s.busy.CompareAndSwap(0, 1) {
		runtime.Gosched()
	}
	for i := 0; i < bs.len; i++ {
		b := bs.buffers[i].Load()
		if b != nil {
			b.clear()
		}
	}
	s.busy.Store(0)
}
*/
