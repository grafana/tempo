// Copyright (c) 2024 Alexey Mayshev and contributors. All rights reserved.
// Copyright (c) 2021 Andrey Pechkurov. All rights reserved.
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
// Copyright notice. This code is a fork of xsync.MapOf from this file with some changes:
// https://github.com/puzpuzpuz/xsync/blob/main/mapof_test.go
//
// Use of this source code is governed by a MIT license that can be found
// at https://github.com/puzpuzpuz/xsync/blob/main/LICENSE

package hashmap

import (
	"fmt"
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/maypok86/otter/v2/internal/xmath"
	"github.com/maypok86/otter/v2/internal/xruntime"
)

type mapResizeHint int

const (
	mapGrowHint   mapResizeHint = 0
	mapShrinkHint mapResizeHint = 1
)

const (
	// number of Map nodes per bucket; 5 nodes lead to size of 64B
	// (one cache line) on 64-bit machines.
	nodesPerMapBucket = 5
	// an empty slot has a zero meta byte, and an occupied one has the high
	// bit set (see h2), so that a zeroed bucket is empty and a new table
	// needs no initialization.
	metaMask           uint64 = 0xffffffffff
	occupiedMeta       uint64 = 0x8080808080808080
	occupiedMetaMasked        = occupiedMeta & metaMask

	// threshold fraction of table occupation to start a table shrinking
	// when deleting the last entry in a bucket chain.
	mapShrinkFraction = 128
	// map load factor to trigger a table resize during insertion;
	// a map holds up to mapLoadFactor*nodesPerMapBucket*mapTableLen
	// key-value pairs (this is a soft limit).
	mapLoadFactor = 0.75
	// minimal table size, i.e. number of buckets; thus, minimal map
	// capacity can be calculated as nodesPerMapBucket*defaultMinMapTableLen.
	defaultMinMapTableLen = 32
	// minimum counter stripes to use.
	minMapCounterLen = 8
	// maximum counter stripes to use; stands for around 4KB of memory.
	maxMapCounterLen = 32
	// minimum buckets per goroutine during parallel resize.
	minBucketsPerGoroutine = 64
)

// Map is like a Go map[K]V but is safe for concurrent
// use by multiple goroutines without additional locking or
// coordination. It follows the interface of sync.Map with
// a number of valuable extensions like Compute or Size.
//
// A Map must not be copied after first use.
//
// Map uses a modified version of Cache-Line Hash Table (CLHT)
// data structure: https://github.com/LPD-EPFL/CLHT
//
// CLHT is built around idea to organize the hash table in
// cache-line-sized buckets, so that on all modern CPUs update
// operations complete with at most one cache-line transfer.
// Also, Get operations involve no write to memory, as well as no
// mutexes or any other sort of locks. Due to this design, in all
// considered scenarios Map outperforms sync.Map.
//
// Map also borrows ideas from Java's j.u.c.ConcurrentHashMap
// (immutable K/V pair structs instead of atomic snapshots)
// and C++'s absl::flat_hash_map (meta memory and SWAR-based
// lookups).
type Map[K comparable, N mapNode] struct {
	totalGrowths atomic.Int64
	totalShrinks atomic.Int64
	resizing     atomic.Bool                 // resize in progress flag
	resizeMu     sync.Mutex                  // only used along with resizeCond
	resizeCond   sync.Cond                   // used to wake up resize waiters (concurrent modifications)
	table        atomic.Pointer[mapTable[K]] // *mapTable
	nodeManager  mapNodeManager[N]
	// keyOffset is the node manager's key offset, cached so that a lookup compares keys without
	// calling the node manager.
	keyOffset   uintptr
	minTableLen int
}

type counterStripe struct {
	c int64
	//lint:ignore U1000 prevents false sharing
	pad [xruntime.CacheLineSize - 8]byte
}

type mapTable[K comparable] struct {
	buckets []bucketPadded
	// striped counter for number of table nodes;
	// used to determine if a table shrinking is needed
	// occupies min(buckets_memory/1024, 64KB) of memory
	size   []counterStripe
	hasher xruntime.Hasher[K]
}

// bucketPadded is a CL-sized map bucket holding up to
// nodesPerMapBucket nodes.
type bucketPadded struct {
	//lint:ignore U1000 pads a bucket to 64 bytes on 32-bit archs; it is already 64 bytes on 64-bit ones
	pad [64 - unsafe.Sizeof(bucket{})]byte
	bucket
}

// bucket keeps mu ahead of next, so that next is the field at the tail.
//
// A []bucketPadded slice is not guaranteed to start on a cache line: for
// tables of 16 to 511 buckets the Go allocator puts an 8-byte malloc header
// in front of it, so each bucket straddles two cache lines, and its last
// 8 bytes share a line with the first 56 bytes of the next bucket. mu is
// written by every write to the bucket, while next only changes when an
// overflow chain grows. With mu at the tail, every write would invalidate
// the meta and nodes that lock-free readers of the neighboring bucket load.
type bucket struct {
	meta  atomic.Uint64
	nodes [nodesPerMapBucket]unsafe.Pointer // node.Node
	mu    sync.Mutex
	next  atomic.Pointer[bucketPadded]
}

// NewWithSize creates a new Map instance with capacity enough
// to hold size nodes. If size is zero or negative, the value
// is ignored.
func NewWithSize[K comparable, N mapNode](nodeManager mapNodeManager[N], size int) *Map[K, N] {
	return newMap[K, N](nodeManager, size)
}

// New creates a new Map instance.
func New[K comparable, N mapNode](nodeManager mapNodeManager[N]) *Map[K, N] {
	return newMap[K, N](nodeManager, defaultMinMapTableLen*nodesPerMapBucket)
}

func newMap[K comparable, N mapNode](nodeManager mapNodeManager[N], sizeHint int) *Map[K, N] {
	m := &Map[K, N]{
		nodeManager: nodeManager,
		keyOffset:   nodeManager.KeyOffset(),
	}
	m.resizeCond = *sync.NewCond(&m.resizeMu)
	table := newMapTable(tableLenFor(sizeHint), xruntime.NewHasher[K]())
	m.minTableLen = len(table.buckets)
	m.table.Store(table)
	return m
}

func tableLenFor(sizeHint int) int {
	if sizeHint <= defaultMinMapTableLen*nodesPerMapBucket {
		return defaultMinMapTableLen
	}
	return int(xmath.RoundUpPowerOf264(uint64((float64(sizeHint) / nodesPerMapBucket) / mapLoadFactor)))
}

func newMapTable[K comparable](minTableLen int, hasher xruntime.Hasher[K]) *mapTable[K] {
	buckets := make([]bucketPadded, minTableLen)
	counterLen := minTableLen >> 10
	if counterLen < minMapCounterLen {
		counterLen = minMapCounterLen
	} else if counterLen > maxMapCounterLen {
		counterLen = maxMapCounterLen
	}
	counter := make([]counterStripe, counterLen)
	t := &mapTable[K]{
		buckets: buckets,
		size:    counter,
		hasher:  hasher,
	}
	return t
}

func zeroValue[V any]() V {
	var zero V
	return zero
}

func (m *Map[K, N]) keyOf(nptr unsafe.Pointer) K {
	return *(*K)(unsafe.Add(nptr, m.keyOffset))
}

// Get returns the node stored in the map for a key, or nil
// if no value is present.
func (m *Map[K, N]) Get(key K) N {
	table := m.table.Load()
	hash := table.hasher.Hash(key)
	h1 := h1(hash)
	h2w := broadcast(h2(hash))
	//nolint:gosec // there is no overflow
	bidx := uint64(len(table.buckets)-1) & h1
	b := &table.buckets[bidx]
	for {
		metaw := b.meta.Load()
		markedw := markZeroBytes(metaw^h2w) & metaMask
		for markedw != 0 {
			idx := firstMarkedByteIndex(markedw)
			nptr := atomic.LoadPointer(&b.nodes[idx])
			if nptr != nil && m.keyOf(nptr) == key {
				return m.nodeManager.FromPointer(nptr)
			}
			markedw &= markedw - 1
		}
		b = b.next.Load()
		if b == nil {
			return zeroValue[N]()
		}
	}
}

// isNil reports whether n is a missing node: the zero value of N, a nil pointer or a nil
// interface.
func isNil[N comparable](n N) bool {
	var zero N
	return n == zero
}

// callUnlockingOnPanic calls computeFunc with mu held and releases mu if computeFunc panics
// or exits the goroutine, so that a failing callback does not leave the bucket locked. The
// bucket is left unchanged in that case. On a normal return mu stays locked.
func callUnlockingOnPanic[N any](mu *sync.Mutex, computeFunc func(n N) N, n N) N {
	returned := false
	defer func() {
		if !returned {
			mu.Unlock()
		}
	}()
	result := computeFunc(n)
	returned = true
	return result
}

// Compute either sets the computed new value for the key or deletes
// the value for the key.
//
// This call locks a hash table bucket while the compute function
// is executed. It means that modifications on other nodes in
// the bucket will be blocked until the computeFn executes. Consider
// this when the function includes long-running operations.
func (m *Map[K, N]) Compute(key K, computeFunc func(n N) N) N {
	for {
	compute_attempt:
		var (
			emptyb   *bucketPadded
			emptyidx int
		)
		table := m.table.Load()
		tableLen := len(table.buckets)
		hash := table.hasher.Hash(key)
		h1 := h1(hash)
		h2 := h2(hash)
		h2w := broadcast(h2)
		//nolint:gosec // there is no overflow
		bidx := uint64(len(table.buckets)-1) & h1
		rootb := &table.buckets[bidx]
		rootb.mu.Lock()
		// The following two checks must go in reverse to what's
		// in the resize method.
		if m.resizeInProgress() {
			// Resize is in progress. Wait, then go for another attempt.
			rootb.mu.Unlock()
			m.waitForResize()
			goto compute_attempt
		}
		if m.newerTableExists(table) {
			// Someone resized the table. Go for another attempt.
			rootb.mu.Unlock()
			goto compute_attempt
		}
		b := rootb
		var prevb *bucketPadded
		for {
			metaw := b.meta.Load()
			markedw := markZeroBytes(metaw^h2w) & metaMask
			for markedw != 0 {
				idx := firstMarkedByteIndex(markedw)
				nptr := b.nodes[idx]
				if nptr != nil && m.keyOf(nptr) == key {
					oldNode := m.nodeManager.FromPointer(nptr)
					// In-place update/delete.
					newNode := callUnlockingOnPanic(&rootb.mu, computeFunc, oldNode)
					// oldNode != nil
					if isNil(newNode) {
						// Deletion.
						// First we update the hash, then the node.
						newmetaw := setByte(metaw, 0, idx)
						b.meta.Store(newmetaw)
						atomic.StorePointer(&b.nodes[idx], nil)
						if newmetaw == 0 && prevb != nil {
							// Unlink the empty overflow bucket. Readers that are in it
							// go on to its next bucket, which stays the same.
							prevb.next.Store(b.next.Load())
						}
						rootb.mu.Unlock()
						table.addSize(bidx, -1)
						// Might need to shrink the table if we left bucket empty.
						if newmetaw == 0 {
							m.resize(table, mapShrinkHint)
						}
						return newNode
					}
					if oldNode.AsPointer() != newNode.AsPointer() {
						atomic.StorePointer(&b.nodes[idx], newNode.AsPointer())
					}
					rootb.mu.Unlock()
					return newNode
				}
				markedw &= markedw - 1
			}
			if emptyb == nil {
				// Search for empty nodes (up to 5 per bucket).
				emptyw := ^metaw & occupiedMetaMasked
				if emptyw != 0 {
					idx := firstMarkedByteIndex(emptyw)
					emptyb = b
					emptyidx = idx
				}
			}
			if b.next.Load() == nil {
				if emptyb != nil {
					// Insertion into an existing bucket.
					var zeroNode N
					// oldNode == nil.
					newNode := callUnlockingOnPanic(&rootb.mu, computeFunc, zeroNode)
					if isNil(newNode) {
						// no op.
						rootb.mu.Unlock()
						return newNode
					}
					// First we update meta, then the node.
					emptyb.meta.Store(setByte(emptyb.meta.Load(), h2, emptyidx))
					atomic.StorePointer(&emptyb.nodes[emptyidx], newNode.AsPointer())
					rootb.mu.Unlock()
					table.addSize(bidx, 1)
					return newNode
				}
				growThreshold := float64(tableLen) * nodesPerMapBucket * mapLoadFactor
				if table.sumSize() > int64(growThreshold) {
					// Need to grow the table. Then go for another attempt.
					rootb.mu.Unlock()
					m.resize(table, mapGrowHint)
					goto compute_attempt
				}
				// Insertion into a new bucket.
				var zeroNode N
				// oldNode == nil
				newNode := callUnlockingOnPanic(&rootb.mu, computeFunc, zeroNode)
				if isNil(newNode) {
					rootb.mu.Unlock()
					return newNode
				}
				// Create and append a bucket.
				newb := new(bucketPadded)
				newb.meta.Store(setByte(0, h2, 0))
				newb.nodes[0] = newNode.AsPointer()
				b.next.Store(newb)
				rootb.mu.Unlock()
				table.addSize(bidx, 1)
				return newNode
			}
			prevb = b
			b = b.next.Load()
		}
	}
}

func (m *Map[K, N]) newerTableExists(table *mapTable[K]) bool {
	return table != m.table.Load()
}

func (m *Map[K, N]) resizeInProgress() bool {
	return m.resizing.Load()
}

func (m *Map[K, N]) waitForResize() {
	m.resizeMu.Lock()
	for m.resizeInProgress() {
		m.resizeCond.Wait()
	}
	m.resizeMu.Unlock()
}

func (m *Map[K, N]) resize(knownTable *mapTable[K], hint mapResizeHint) {
	knownTableLen := len(knownTable.buckets)
	// Fast path for shrink attempts.
	if hint == mapShrinkHint {
		if m.minTableLen == knownTableLen ||
			knownTable.sumSize() > int64((knownTableLen*nodesPerMapBucket)/mapShrinkFraction) {
			return
		}
	}
	// Slow path.
	if !m.resizing.CompareAndSwap(false, true) {
		// Someone else started resize. Wait for it to finish.
		m.waitForResize()
		return
	}
	var newTable *mapTable[K]
	table := m.table.Load()
	tableLen := len(table.buckets)
	switch hint {
	case mapGrowHint:
		if table != knownTable {
			// The caller saw knownTable full, but another writer has grown it
			// since. Growing again would double a table that has room.
			m.finishResize()
			return
		}
		// Grow the table with factor of 2. The new table keeps the hasher,
		// so that the copy can write to its buckets without locking them
		// (see copyBuckets).
		m.totalGrowths.Add(1)
		newTable = newMapTable(tableLen<<1, table.hasher)
	case mapShrinkHint:
		shrinkThreshold := int64((tableLen * nodesPerMapBucket) / mapShrinkFraction)
		if tableLen > m.minTableLen && table.sumSize() <= shrinkThreshold {
			// Shrink the table with factor of 2, keeping the hasher as when growing.
			m.totalShrinks.Add(1)
			newTable = newMapTable(tableLen>>1, table.hasher)
		} else {
			// No need to shrink. Wake up all waiters and give up.
			m.finishResize()
			return
		}
	default:
		panic(fmt.Sprintf("unexpected resize hint: %d", hint))
	}
	m.copyBuckets(table, newTable)
	// Publish the new table and wake up all waiters.
	m.table.Store(newTable)
	m.finishResize()
}

// finishResize clears the resize flag and wakes up the writers waiting for it.
func (m *Map[K, N]) finishResize() {
	m.resizeMu.Lock()
	m.resizing.Store(false)
	m.resizeCond.Broadcast()
	m.resizeMu.Unlock()
}

// copyBuckets copies the nodes of table into newTable, in parallel if the
// table is large enough.
//
// The two tables share the hasher and their lengths differ by a factor of 2,
// so a node can only move between buckets with equal indexes modulo
// baseLen = min(len(table), len(newTable)): when growing, source bucket i
// goes to destination buckets i and i+baseLen, and when shrinking, source
// buckets i and i+baseLen go to destination bucket i. The work is split by
// i, so every destination bucket is written by one goroutine only and needs
// no lock. Nobody else can see newTable until it is published.
func (m *Map[K, N]) copyBuckets(table, newTable *mapTable[K]) {
	tableLen := len(table.buckets)
	baseLen := min(tableLen, len(newTable.buckets))
	copyRange := func(start, end int) {
		copied := 0
		for i := start; i < end; i++ {
			for src := i; src < tableLen; src += baseLen {
				copied += m.copyBucket(&table.buckets[src], newTable)
			}
		}
		//nolint:gosec // there is no overflow
		newTable.addSize(uint64(start), copied)
	}

	chunks := 1
	if tableLen >= minBucketsPerGoroutine*2 {
		chunks = min(tableLen/minBucketsPerGoroutine, runtime.GOMAXPROCS(0))
		chunks = max(chunks, 1)
	}
	if chunks == 1 {
		copyRange(0, baseLen)
		return
	}
	var copyWg sync.WaitGroup
	chunkSize := (baseLen + chunks - 1) / chunks
	for start := 0; start < baseLen; start += chunkSize {
		copyWg.Add(1)
		go func(start, end int) {
			defer copyWg.Done()
			copyRange(start, end)
		}(start, min(start+chunkSize, baseLen))
	}
	copyWg.Wait()
}

func (m *Map[K, N]) copyBucket(b *bucketPadded, destTable *mapTable[K]) (copied int) {
	rootb := b
	rootb.mu.Lock()
	//nolint:gocritic // nesting is normal here
	for {
		for i := 0; i < nodesPerMapBucket; i++ {
			if b.nodes[i] != nil {
				hash := destTable.hasher.Hash(m.keyOf(b.nodes[i]))
				//nolint:gosec // there is no overflow
				bidx := uint64(len(destTable.buckets)-1) & h1(hash)
				destb := &destTable.buckets[bidx]
				appendToBucket(h2(hash), b.nodes[i], destb)
				copied++
			}
		}
		if next := b.next.Load(); next == nil {
			rootb.mu.Unlock()
			return copied
		} else {
			b = next
		}
	}
}

// Range calls f sequentially for each key and value present in the
// map. If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot
// of the Map's contents: no key will be visited more than once, but
// if the value for any key is stored or deleted concurrently, Range
// may reflect any mapping for that key from any point during the
// Range call.
//
// It is safe to modify the map while iterating it, including entry
// creation, modification and deletion. However, the concurrent
// modification rule apply, i.e. the changes may be not reflected
// in the subsequently iterated nodes.
func (m *Map[K, N]) Range(fn func(n N) bool) {
	var zeroPtr unsafe.Pointer
	// Pre-allocate array big enough to fit nodes for most hash tables.
	bnodes := make([]unsafe.Pointer, 0, 16*nodesPerMapBucket)
	table := m.table.Load()
	for i := range table.buckets {
		rootb := &table.buckets[i]
		b := rootb
		// Prevent concurrent modifications and copy all nodes into
		// the intermediate slice.
		rootb.mu.Lock()
		for {
			for i := 0; i < nodesPerMapBucket; i++ {
				if b.nodes[i] != nil {
					bnodes = append(bnodes, b.nodes[i])
				}
			}
			if next := b.next.Load(); next == nil {
				rootb.mu.Unlock()
				break
			} else {
				b = next
			}
		}
		// Call the function for all copied nodes.
		for j := range bnodes {
			n := m.nodeManager.FromPointer(bnodes[j])
			if !fn(n) {
				return
			}
			// Remove the reference to avoid preventing the copied
			// nodes from being GCed until this method finishes.
			bnodes[j] = zeroPtr
		}
		bnodes = bnodes[:0]
	}
}

// Size returns current size of the map.
func (m *Map[K, N]) Size() int {
	table := m.table.Load()
	return int(table.sumSize())
}

func appendToBucket(h2 uint8, nodePtr unsafe.Pointer, b *bucketPadded) {
	for {
		for i := 0; i < nodesPerMapBucket; i++ {
			if b.nodes[i] == nil {
				b.meta.Store(setByte(b.meta.Load(), h2, i))
				b.nodes[i] = nodePtr
				return
			}
		}
		if next := b.next.Load(); next == nil {
			newb := new(bucketPadded)
			newb.meta.Store(setByte(0, h2, 0))
			newb.nodes[0] = nodePtr
			b.next.Store(newb)
			return
		} else {
			b = next
		}
	}
}

func (table *mapTable[K]) addSize(bucketIdx uint64, delta int) {
	//nolint:gosec // there is no overflow
	cidx := uint64(len(table.size)-1) & bucketIdx
	atomic.AddInt64(&table.size[cidx].c, int64(delta))
}

func (table *mapTable[K]) sumSize() int64 {
	sum := int64(0)
	for i := range table.size {
		sum += atomic.LoadInt64(&table.size[i].c)
	}
	return max(sum, 0)
}

func h1(h uint64) uint64 {
	return h >> 7
}

func h2(h uint64) uint8 {
	//nolint:gosec // there is no overflow
	return 0x80 | uint8(h&0x7f)
}

func broadcast(b uint8) uint64 {
	return 0x101010101010101 * uint64(b)
}

func firstMarkedByteIndex(w uint64) int {
	return bits.TrailingZeros64(w) >> 3
}

// SWAR byte search: may produce false positives, e.g. for 0x0100,
// so make sure to double-check bytes found by this function.
func markZeroBytes(w uint64) uint64 {
	return ((w - 0x0101010101010101) & (^w) & 0x8080808080808080)
}

func setByte(w uint64, b uint8, idx int) uint64 {
	shift := idx << 3
	return (w &^ (0xff << shift)) | (uint64(b) << shift)
}
