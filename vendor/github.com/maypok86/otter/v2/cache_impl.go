// Copyright (c) 2023 Alexey Mayshev and contributors. All rights reserved.
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

package otter

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/maypok86/otter/v2/internal/deque/queue"
	"github.com/maypok86/otter/v2/internal/expiration"
	"github.com/maypok86/otter/v2/internal/generated/node"
	"github.com/maypok86/otter/v2/internal/hashmap"
	"github.com/maypok86/otter/v2/internal/lossy"
	"github.com/maypok86/otter/v2/internal/xmath"
	"github.com/maypok86/otter/v2/internal/xruntime"
	"github.com/maypok86/otter/v2/stats"
)

const (
	unreachableExpiresAt     = int64(xruntime.MaxDuration)
	unreachableRefreshableAt = int64(xruntime.MaxDuration)
	noTime                   = int64(0)

	minWriteBufferSize = 4
	writeBufferRetries = 100
)

const (
	// A drain is not taking place.
	idle uint32 = 0
	// A drain is required due to a pending write modification.
	required uint32 = 1
	// A drain is in progress and will transition to idle.
	processingToIdle uint32 = 2
	// A drain is in progress and will transition to required.
	processingToRequired uint32 = 3
)

var (
	maxWriteBufferSize   uint32
	maxStripedBufferSize int
)

func init() {
	parallelism := xruntime.Parallelism()
	roundedParallelism := int(xmath.RoundUpPowerOf2(parallelism))
	//nolint:gosec // there will never be an overflow
	maxWriteBufferSize = uint32(128 * roundedParallelism)
	maxStripedBufferSize = 4 * roundedParallelism
}

func zeroValue[V any]() V {
	var v V
	return v
}

// cache is a structure performs a best-effort bounding of a hash table using eviction algorithm
// to determine which entries to evict when the capacity is exceeded.
type cache[K comparable, V any] struct {
	drainStatus        atomic.Uint32
	_                  [xruntime.CacheLineSize - 4]byte
	nodeManager        *node.Manager[K, V]
	hashmap            *hashmap.Map[K, node.Node]
	evictionPolicy     *policy[K]
	timerWheel         *expiration.Wheel
	stats              stats.Recorder
	statsSnapshoter    stats.Snapshoter
	logger             Logger
	clock              timeSource
	statsClock         *realSource
	readBuffer         *lossy.Striped
	writeBuffer        *queue.MPSC[task]
	executor           func(fn func())
	singleflight       *group[K, V]
	evictionMutex      sync.Mutex
	stopOnce           sync.Once
	stop               chan struct{}
	done               chan struct{}
	weigher            func(key K, value V) uint32
	onDeletion         func(e DeletionEvent[K, V])
	onAtomicDeletion   func(e DeletionEvent[K, V])
	expiryCalculator   ExpiryCalculator[K, V]
	refreshCalculator  RefreshCalculator[K, V]
	hasDefaultExecutor bool
	withTime           bool
	withExpiration     bool
	withRefresh        bool
	withEviction       bool
	isWeighted         bool
	withMaintenance    bool
	withStats          bool
	// whether a read may change the expiration time: ExpireAfterRead is not called for the
	// built-in calculators whose reads always keep it
	withReadExpiry bool
	// the time of the current maintenance pass, read once when it starts (with expiration only),
	// so that the timer wheel and size eviction agree on what has expired; guarded by
	// evictionMutex
	maintenanceNanos int64
	// even while a scheduler that holds evictionMutex waits to learn whether its executor ran the
	// drain inline; whoever makes it odd releases that mutex. Each schedule takes a new value, so
	// a scheduler releases only its own lock, and only a drain can take over another's release
	drainToken       atomic.Uint64
	drainBuffersFunc func()
}

// newCache returns a new cache instance based on the settings from Options.
func newCache[K comparable, V any](o *Options[K, V]) *cache[K, V] {
	withWeight := o.MaximumWeight > 0
	nodeManager := node.NewManager[K, V](node.Config{
		WithSize:       o.MaximumSize > 0,
		WithExpiration: o.ExpiryCalculator != nil,
		WithRefresh:    o.RefreshCalculator != nil,
		WithWeight:     withWeight,
	})

	maximum := o.getMaximum()
	withEviction := maximum > 0

	withStats := o.StatsRecorder != nil
	if withStats {
		_, ok := o.StatsRecorder.(*stats.NoopRecorder)
		withStats = !ok
	}
	var logger Logger = &safeLogger{logger: o.getLogger()}
	var statsRecorder stats.Recorder = &stats.NoopRecorder{}
	var statsSnapshoter stats.Snapshoter = &stats.NoopRecorder{}
	if withStats {
		statsRecorder = &safeRecorder{recorder: o.StatsRecorder, logger: logger}
		if snapshoter, ok := o.StatsRecorder.(stats.Snapshoter); ok {
			statsSnapshoter = snapshoter
		}
	}

	c := &cache[K, V]{
		nodeManager:        nodeManager,
		hashmap:            hashmap.NewWithSize[K, node.Node](nodeManager, o.getInitialCapacity()),
		stats:              statsRecorder,
		statsSnapshoter:    statsSnapshoter,
		logger:             logger,
		singleflight:       &group[K, V]{},
		executor:           o.getExecutor(),
		hasDefaultExecutor: o.Executor == nil,
		weigher:            o.getWeigher(),
		onDeletion:         o.OnDeletion,
		onAtomicDeletion:   o.OnAtomicDeletion,
		clock:              newTimeSource(o.Clock),
		statsClock:         &realSource{},
		expiryCalculator:   o.ExpiryCalculator,
		refreshCalculator:  o.RefreshCalculator,
		isWeighted:         withWeight,
		withStats:          withStats,
	}
	c.drainBuffersFunc = c.drainBuffers
	c.singleflight.commit = c.commitCall
	c.singleflight.publish = c.publish
	// read on every refresh, so that tests can replace the executor
	c.singleflight.executor = func(fn func()) { c.executor(fn) }
	c.singleflight.nodeOf = func(key K) node.Node { return c.hashmap.Get(key) }

	if withStats {
		c.statsClock.Init()
	}

	c.withEviction = withEviction
	if c.withEviction {
		var expectedSize uint64
		if o.hasInitialCapacity() {
			//nolint:gosec // there's no overflow
			expectedSize = min(maximum, uint64(o.getInitialCapacity()))
		}
		c.evictionPolicy = newPolicy[K](nodeManager.Kind, withWeight, expectedSize, nodeManager.KeyOffset(), c.evictNode)
	}

	if o.ExpiryCalculator != nil {
		c.timerWheel = expiration.NewWheel(nodeManager.Kind)
	}

	c.withExpiration = o.ExpiryCalculator != nil
	_, creating := o.ExpiryCalculator.(*varExpiryCreating[K, V])
	_, writing := o.ExpiryCalculator.(*varExpiryWriting[K, V])
	c.withReadExpiry = c.withExpiration && !creating && !writing
	c.withRefresh = o.RefreshCalculator != nil
	c.withTime = c.withExpiration || c.withRefresh
	c.withMaintenance = c.withEviction || c.withExpiration

	if c.withMaintenance {
		c.readBuffer = lossy.NewStriped(maxStripedBufferSize, nodeManager.Kind)
		c.writeBuffer = queue.NewMPSC[task](minWriteBufferSize, maxWriteBufferSize)
	}
	if c.withTime {
		c.clock.Init()
	}
	if c.withExpiration {
		// No cleanup of the Cache stops the goroutine: cleanups run outside testing/synctest
		// bubbles, and closing a channel created in one from outside is a fatal error. The
		// goroutine returns by itself once the cache is collected.
		c.stop = make(chan struct{})
		c.done = make(chan struct{})
		go periodicCleanUp(weak.Make(c), nil, nil, c.stop, c.done)
	}

	if c.withEviction {
		c.SetMaximum(maximum)
	}

	return c
}

// newNode creates a node for a write whose weight and timestamps have already been computed.
// With updatable, the node can have its value replaced in place: a value that would be kept
// inline is kept behind an atomic pointer instead.
func (c *cache[K, V]) newNode(key K, value V, weight uint32, expiresAt, refreshableAt int64, updatable bool) node.Node {
	if updatable {
		return c.nodeManager.CreateUpdatable(key, value, expiresAt, refreshableAt, weight)
	}
	return c.nodeManager.Create(key, value, expiresAt, refreshableAt, weight)
}

// writeEntry describes a write before it is applied: the new value and weight with the
// timestamps of the entry it replaces, which are kept unless the calculators change them. old is
// what the mutation found, nil for a creation, which sees no timestamps.
func (c *cache[K, V]) writeEntry(key K, value V, weight uint32, old *previous[K, V], nanos int64) Entry[K, V] {
	nowNano := noTime
	if c.withTime {
		nowNano = nanos
	}

	expiresAt := unreachableExpiresAt
	if c.withExpiration && old != nil {
		// A read that moved the deadline while the mutation ran user code comes before the
		// write, unless it moved it to before nowNano, when the mutation found the entry live.
		expiresAt = old.node.ExpiresAt()
		if expiresAt <= old.nowNano {
			expiresAt = old.nodeExpiresAt
		}
	}

	refreshableAt := unreachableRefreshableAt
	if c.withRefresh && old != nil {
		refreshableAt = old.node.RefreshableAt()
	}

	return Entry[K, V]{
		Key:               key,
		Value:             value,
		Weight:            weight,
		ExpiresAtNano:     expiresAt,
		RefreshableAtNano: refreshableAt,
		SnapshotAtNano:    nowNano,
	}
}

func (c *cache[K, V]) nodeToEntry(n node.Node, nanos int64) Entry[K, V] {
	return c.nodeValueToEntry(n, c.nodeManager.Value(n), nanos)
}

// nodeValueToEntry is nodeToEntry with the value the caller loaded from the node.
func (c *cache[K, V]) nodeValueToEntry(n node.Node, value V, nanos int64) Entry[K, V] {
	nowNano := noTime
	if c.withTime {
		nowNano = nanos
	}

	expiresAt := unreachableExpiresAt
	if c.withExpiration {
		expiresAt = n.ExpiresAt()
	}

	refreshableAt := unreachableRefreshableAt
	if c.withRefresh {
		refreshableAt = n.RefreshableAt()
	}

	return Entry[K, V]{
		Key:               c.nodeManager.Key(n),
		Value:             value,
		Weight:            n.Weight(),
		ExpiresAtNano:     expiresAt,
		RefreshableAtNano: refreshableAt,
		SnapshotAtNano:    nowNano,
	}
}

// has checks if there is an item with the given key in the cache.
func (c *cache[K, V]) has(key K) bool {
	_, ok := c.GetIfPresent(key)
	return ok
}

// GetIfPresent returns the value associated with the key in this cache.
func (c *cache[K, V]) GetIfPresent(key K) (V, bool) {
	nowNano := c.clock.NowNano()
	n, value := c.getNode(key, nowNano)
	if n == nil {
		return zeroValue[V](), false
	}

	return value, true
}

// getNode returns the node associated with the key in this cache, and its value. The value is
// loaded once and is the one ExpireAfterRead is shown, so a read that returns it comes before an
// update in place that replaces it.
func (c *cache[K, V]) getNode(key K, nowNano int64) (n node.Node, value V) {
	n = c.hashmap.Get(key)
	if n == nil {
		c.stats.RecordMisses(1)
		if c.drainStatus.Load() == required {
			c.scheduleDrainBuffers()
		}
		return nil, value
	}
	if n.HasExpired(nowNano) {
		c.stats.RecordMisses(1)
		c.scheduleDrainBuffers()
		return nil, value
	}

	if c.withReadExpiry {
		value = c.calcExpiresAtAfterRead(n, nowNano)
	} else {
		value = c.nodeManager.Value(n)
	}
	c.afterRead(n, true)

	return n, value
}

// getNodeQuietly returns the node associated with the key in this cache.
//
// Unlike getNode, this function does not produce any side effects
// such as updating statistics or the eviction policy. As in getNode, a node that a write is
// replacing or removing is the key's node until the write changes the table.
func (c *cache[K, V]) getNodeQuietly(key K, nowNano int64) node.Node {
	n := c.hashmap.Get(key)
	if n == nil || n.HasExpired(nowNano) {
		return nil
	}

	return n
}

func (c *cache[K, V]) afterRead(got node.Node, recordHit bool) {
	if recordHit {
		c.stats.RecordHits(1)
	}

	delayable := c.skipReadBuffer() || c.readBuffer.Add(got) != lossy.Full
	if c.shouldDrainBuffers(delayable) {
		c.scheduleDrainBuffers()
	}
}

// Set associates the value with the key in this cache.
//
// If the specified key is not already associated with a value, then it returns new value and true.
// An expired entry that has not been removed yet counts as no value.
//
// If the specified key is already associated with a value, then it returns existing value and false.
func (c *cache[K, V]) Set(key K, value V) (V, bool) {
	return c.set(key, value, false)
}

// SetIfAbsent if the specified key is not already associated with a value associates it with the given value.
//
// If the specified key is not already associated with a value, then it returns new value and true.
//
// If the specified key is already associated with a value, then it returns existing value and false.
func (c *cache[K, V]) SetIfAbsent(key K, value V) (V, bool) {
	return c.set(key, value, true)
}

// calcExpiresAtAfterRead moves the deadline of n as ExpireAfterRead asks for the entry with the
// value it loads, and returns that value for the read to return. The cache must have read expiry
// (withReadExpiry).
func (c *cache[K, V]) calcExpiresAtAfterRead(n node.Node, nowNano int64) V {
	value, valuePtr, valueWord := c.nodeManager.ValueAndIdentity(n)
	entry := c.nodeValueToEntry(n, value, nowNano)
	// The new deadline is derived from the entry the calculator was shown, so it may only replace
	// that entry's deadline. A writer can store a new deadline in place in the meantime, and
	// comparing against a fresh read of the node would revert it to one derived from the old value.
	if next, ok := expiresAtAfterRead(c.expiryCalculator, entry, nowNano); ok {
		// The entry may expire while the calculator runs, and a reader may then find the key
		// absent: moving the deadline of an expired entry would bring it back. The clock is
		// read again only here, since the deadline rarely moves (see expireTolerance).
		if entry.ExpiresAtNano <= c.clock.NowNano() {
			return value
		}
		// The read returns value, so it comes before an update in place that replaced the value
		// while the calculator ran, and the update's deadline stands. The CAS alone would not
		// tell: the update may store the very deadline the entry had, for example when the clock
		// has not moved. An update in the few instructions between this check and the CAS still
		// can, as the writer stores the value before the deadline.
		if _, p, w := c.nodeManager.ValueAndIdentity(n); p != valuePtr || w != valueWord {
			return value
		}
		n.CASExpiresAt(entry.ExpiresAtNano, next)
	}
	return value
}

// GetEntry returns the cache entry associated with the key in this cache.
func (c *cache[K, V]) GetEntry(key K) (Entry[K, V], bool) {
	nowNano := c.clock.NowNano()
	n, value := c.getNode(key, nowNano)
	if n == nil {
		return Entry[K, V]{}, false
	}
	return c.nodeValueToEntry(n, value, nowNano), true
}

// GetEntryQuietly returns the cache entry associated with the key in this cache.
//
// Unlike GetEntry, this function does not produce any side effects
// such as updating statistics or the eviction policy.
func (c *cache[K, V]) GetEntryQuietly(key K) (Entry[K, V], bool) {
	nowNano := c.clock.NowNano()
	n := c.getNodeQuietly(key, nowNano)
	if n == nil {
		return Entry[K, V]{}, false
	}
	return c.nodeToEntry(n, nowNano), true
}

// SetExpiresAfter specifies that the entry should be automatically removed from the cache once the duration has
// elapsed. The expiration policy determines when the entry's age is reset. It has no effect on an absent
// or expired entry.
//
// This call locks a hash table bucket, as a write does. Calling it from a function that runs while the
// cache holds a bucket lock, such as a Compute function, OnAtomicDeletion, a weigher or a calculator,
// may deadlock.
func (c *cache[K, V]) SetExpiresAfter(key K, expiresAfter time.Duration) {
	if !c.withExpiration || expiresAfter <= 0 {
		return
	}
	c.setExpiresAt(key, func(nowNano int64) int64 { return deadlineAfter(nowNano, expiresAfter) })
}

// setExpiresAt is SetExpiresAfter with the deadline that deadline returns for the current time.
func (c *cache[K, V]) setExpiresAt(key K, deadline func(nowNano int64) int64) {
	var (
		n       node.Node
		nowNano int64
		earlier bool
	)
	// The deadline is changed under the key's bucket lock, as a write changes it: a write reads
	// the deadline before its calculator runs and stores the result after, so a deadline set
	// without the lock in between would be overwritten with one computed from the former. The
	// clock is read under the lock too, as in mutate.
	c.hashmap.Compute(key, func(cur node.Node) node.Node {
		if cur == nil {
			return nil
		}
		nowNano = c.clock.NowNano()
		// An expired entry that is still in the table is absent: a new deadline would bring its
		// stale value back.
		expiresAt := cur.ExpiresAt()
		if expiresAt <= nowNano {
			return cur
		}
		n = cur
		// The caller asked for this deadline, so it is set exactly.
		if next := deadline(nowNano); next != expiresAt {
			cur.SetExpiresAt(next)
			earlier = next < expiresAt
		}
		return cur
	})
	if n == nil {
		return
	}
	if earlier {
		// The read buffer may drop the access, and the node would then wait in the timer wheel's
		// bucket for its former deadline, so an earlier deadline goes through the write buffer. A
		// later one needs nothing: the wheel re-checks the node when its former bucket expires.
		// The task goes first, as for a write: afterRead may submit a drain to an executor that
		// calls runtime.Goexit.
		c.afterWriteTask(newTask(n, nil, reconcileReason))
	}
	c.afterRead(n, false)
}

// SetRefreshableAfter specifies that each entry should be eligible for reloading once a fixed duration has elapsed.
// The refresh policy determines when the entry's age is reset.
//
// This call locks a hash table bucket, as a write does. Calling it from a function that runs while the
// cache holds a bucket lock, such as a Compute function, OnAtomicDeletion, a weigher or a calculator,
// may deadlock.
func (c *cache[K, V]) SetRefreshableAfter(key K, refreshableAfter time.Duration) {
	if !c.withRefresh || refreshableAfter <= 0 {
		return
	}
	c.setRefreshableAt(key, func(nowNano int64) int64 { return deadlineAfter(nowNano, refreshableAfter) })
}

// setRefreshableAt is SetRefreshableAfter with the time that deadline returns for the current time.
func (c *cache[K, V]) setRefreshableAt(key K, deadline func(nowNano int64) int64) {
	// Under the key's bucket lock, as for setExpiresAt.
	c.hashmap.Compute(key, func(n node.Node) node.Node {
		if n == nil {
			return nil
		}
		if next := deadline(c.clock.NowNano()); next != n.RefreshableAt() {
			n.SetRefreshableAt(next)
		}
		return n
	})
}

func (c *cache[K, V]) set(key K, value V, onlyIfAbsent bool) (V, bool) {
	m := c.mutate(key, fromCaller, func(prev previous[K, V]) (mutationOp, V) {
		if onlyIfAbsent && prev.live {
			if c.withReadExpiry {
				c.calcExpiresAtAfterRead(prev.node, prev.nowNano)
			}
			return opKeep, value
		}
		return opWrite, value
	})
	c.publish(m)
	// An expired entry that is still in the table is absent, as for reads.
	if !m.live {
		return value, true
	}
	if onlyIfAbsent {
		c.afterRead(m.node, false)
	}
	return m.value, false
}

// Compute either sets the computed new value for the key,
// invalidates the value for the key, or does nothing, based on
// the returned [ComputeOp]. When the op returned by remappingFunc
// is [WriteOp], the value is updated to the new value. If
// it is [InvalidateOp], the entry is removed from the cache
// altogether. And finally, if the op is [CancelOp] then the
// entry is left as-is. In other words, if it did not already
// exist, it is not created, and if it did exist, it is not
// updated. This is useful to synchronously execute some
// operation on the value without incurring the cost of
// updating the cache every time.
//
// The ok result indicates whether the entry is present in the cache after the compute operation.
// The actualValue result contains the value of the cache
// if a corresponding entry is present, or the zero value otherwise.
// You can think of these results as equivalent to regular key-value lookups in a map.
//
// This call locks a hash table bucket while the compute function
// is executed. It means that modifications on other entries in
// the bucket will be blocked until the remappingFunc executes. Consider
// this when the function includes long-running operations.
func (c *cache[K, V]) Compute(
	key K,
	remappingFunc func(oldValue V, found bool) (newValue V, op ComputeOp),
) (V, bool) {
	return c.doCompute(key, remappingFunc, true)
}

// ComputeIfAbsent returns the existing value for the key if
// present. Otherwise, it tries to compute the value using the
// provided function. If mappingFunc returns true as the cancel value, the computation is cancelled and the zero value
// for type V is returned.
//
// The ok result indicates whether the entry is present in the cache after the compute operation.
// The actualValue result contains the value of the cache
// if a corresponding entry is present, or the zero value
// otherwise. You can think of these results as equivalent to regular key-value lookups in a map.
//
// This call locks a hash table bucket while the compute function
// is executed. It means that modifications on other entries in
// the bucket will be blocked until the valueFn executes. Consider
// this when the function includes long-running operations.
func (c *cache[K, V]) ComputeIfAbsent(
	key K,
	mappingFunc func() (newValue V, cancel bool),
) (V, bool) {
	nowNano := c.clock.NowNano()
	if n, value := c.getNode(key, nowNano); n != nil {
		return value, true
	}

	return c.doCompute(key, func(oldValue V, found bool) (newValue V, op ComputeOp) {
		if found {
			return oldValue, CancelOp
		}
		newValue, cancel := mappingFunc()
		if cancel {
			return zeroValue[V](), CancelOp
		}
		return newValue, WriteOp
	}, false)
}

// ComputeIfPresent returns the zero value for type V if the key is not found.
// Otherwise, it tries to compute the value using the provided function.
//
// ComputeIfPresent either sets the computed new value for the key,
// invalidates the value for the key, or does nothing, based on
// the returned [ComputeOp]. When the op returned by remappingFunc
// is [WriteOp], the value is updated to the new value. If
// it is [InvalidateOp], the entry is removed from the cache
// altogether. And finally, if the op is [CancelOp] then the
// entry is left as-is. In other words, if it did not already
// exist, it is not created, and if it did exist, it is not
// updated. This is useful to synchronously execute some
// operation on the value without incurring the cost of
// updating the cache every time.
//
// The ok result indicates whether the entry is present in the cache after the compute operation.
// The actualValue result contains the value of the cache
// if a corresponding entry is present, or the zero value
// otherwise. You can think of these results as equivalent to regular key-value lookups in a map.
//
// This call locks a hash table bucket while the compute function
// is executed. It means that modifications on other entries in
// the bucket will be blocked until the valueFn executes. Consider
// this when the function includes long-running operations.
func (c *cache[K, V]) ComputeIfPresent(
	key K,
	remappingFunc func(oldValue V) (newValue V, op ComputeOp),
) (V, bool) {
	// The check only skips locking the bucket of an absent key. It is not a read: as in Compute,
	// the hit or miss is the entry the function is given, and a read of it would also restart
	// its ExpireAfterRead deadline before the function decides what to do with it. An expired
	// entry is removed under the lock, as Compute removes it.
	if c.hashmap.Get(key) == nil {
		c.stats.RecordMisses(1)
		return zeroValue[V](), false
	}

	return c.doCompute(key, func(oldValue V, found bool) (newValue V, op ComputeOp) {
		if found {
			return remappingFunc(oldValue)
		}
		return zeroValue[V](), CancelOp
	}, true)
}

func (c *cache[K, V]) doCompute(
	key K,
	remappingFunc func(oldValue V, found bool) (newValue V, op ComputeOp),
	recordStats bool,
) (V, bool) {
	var (
		result     V
		op         ComputeOp
		notValidOp bool
		panicErr   error
	)
	m := c.mutate(key, fromCaller, func(prev previous[K, V]) (mutationOp, V) {
		var (
			oldValue    V
			actualValue V
		)
		if prev.live {
			oldValue = prev.value
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					panicErr = newPanicError(r)
				}
			}()

			actualValue, op = remappingFunc(oldValue, prev.live)
		}()
		if panicErr != nil {
			return opKeep, actualValue
		}
		switch op {
		case CancelOp:
			if prev.node != nil && !prev.live {
				// The expired entry is removed, as the timer wheel would remove it.
				return opRemoveExpired, actualValue
			}
			result = prev.value
			return opKeep, actualValue
		case WriteOp:
			result = actualValue
			return opWrite, actualValue
		case InvalidateOp:
			return opInvalidate, actualValue
		default:
			notValidOp = true
			return opKeep, actualValue
		}
	})
	if panicErr != nil {
		panic(panicErr)
	}
	if notValidOp {
		panic(fmt.Sprintf("otter: invalid ComputeOp: %d", op))
	}
	c.publish(m)
	// recorded after the mutation is published, so that a recorder that calls runtime.Goexit
	// cannot keep the change out of the policies
	if recordStats {
		if m.live {
			c.stats.RecordHits(1)
		} else {
			c.stats.RecordMisses(1)
		}
	}
	if m.next == nil {
		return zeroValue[V](), false
	}
	// The node may be updated in place by another writer once the bucket lock is released, so
	// the result is the value seen under the lock, not c.nodeManager.Value(m.next).
	return result, true
}

// startRefresh registers a refresh of key and returns its call, or returns the call already
// registered for key. shouldLoad reports that the caller registered the call and must run it.
//
// The refresh's base, the key's live node (nil if absent or expired), is read and the call
// registered under the key's bucket lock. A write or an invalidation cancels the key's calls under
// the same lock, before it publishes its change. So it either comes before the registration, and
// the refresh starts from what it wrote, or cancels the refresh; and a refresh requested after the
// change is visible never joins a call that the change cancelled.
//
// A refresh that a read starts (!isManual) is not registered if the key's node is fresh by then, having
// been refreshed or replaced since the read, or the key is gone: there is nothing left to refresh,
// and cl is nil. Without this, every read that saw the key stale before the first refresh
// completed would reload it again.
//
// again is the manual refresh's way to run a refresh registered in place of a load it joins,
// if the load fails for its owner's context (see loadAgain); it is nil for a refresh that a read
// starts, which never joins a load.
func (c *cache[K, V]) startRefresh(
	key K,
	isManual bool,
	nowNano int64,
	again func(s *call[K, V]),
) (cl *call[K, V], shouldLoad bool) {
	// fast path: a read that saw a change does not find the calls the change cancelled. A call is
	// joined only if it was started from the key's live node, or from an absent key that is
	// still absent, and has not committed (see group.startRefresh); then it is not superseded
	// either. A load is joined only under the lock, see call.again.
	if cl := c.singleflight.getCall(key); cl != nil && cl.isRefresh && !cl.committed.Load() {
		if n := c.getNodeQuietly(key, nowNano); cl.base == n && (n == nil || n.IsAlive()) {
			return cl, false
		}
	}
	c.hashmap.Compute(key, func(current node.Node) node.Node {
		// The time is read again under the lock, as a mutation does: the entry may have expired
		// while the refresh waited for it, and a load of the key registered since must not be
		// replaced by a refresh of the expired value.
		nowNano = max(nowNano, c.clock.NowNano())
		var base node.Node
		if current != nil && !current.HasExpired(nowNano) {
			base = current
		}
		if !isManual && (base == nil || base.IsFresh(nowNano)) {
			return current
		}
		cl, shouldLoad = c.singleflight.startRefresh(key, base)
		if !shouldLoad && !cl.isRefresh && again != nil && cl.again == nil {
			cl.again = &loadAgain[K, V]{submit: again}
		}
		return current
	})
	return cl, shouldLoad
}

func (c *cache[K, V]) refreshKey(
	ctx context.Context,
	key K,
	loader Loader[K, V],
	isManual bool,
) <-chan RefreshResult[K, V] {
	if !c.withRefresh {
		return nil
	}

	var ch chan RefreshResult[K, V]
	if isManual {
		ch = make(chan RefreshResult[K, V], 1)
	}

	// The refresh is registered before it is handed to the executor, so that a write that
	// follows the request cancels it. Registered by the task, a refresh that the executor starts
	// after the write would replace the written value or bring an invalidated key back.
	// The clock is user code, so it is read before: if it panicked after the registration, the
	// call would never be finished, and every later refresh of the key would join it.
	if !isManual {
		// A refresh that a read starts is the cache's own work: a panic of the clock, or of a
		// custom executor submitting it, is logged, as one submitting maintenance or OnDeletion,
		// and the read returns the value it found. submitRefresh has finished the call by then.
		defer recoverCallback(c.logger, "Panicked while starting a refresh")
	}
	var again func(s *call[K, V])
	if isManual {
		again = func(s *call[K, V]) {
			defer recoverCallback(c.logger, "Panicked while starting a refresh")
			c.singleflight.submitRefresh([]*call[K, V]{s}, func() { c.refreshTask(ctx, s, loader, nil) })
		}
	}
	nowNano := c.clock.NowNano()
	cl, shouldLoad := c.startRefresh(key, isManual, nowNano, again)
	if cl == nil {
		return ch
	}
	if !shouldLoad {
		// A load or refresh of the key is in flight, and its result is all there is to report.
		// A task that waited for it would take an executor slot for the whole load: with an
		// executor that runs a bounded number of tasks, enough stale reads of one key would take
		// every slot and block, and so would the task they wait for. A manual refresh waits in a
		// goroutine of its own, and the error is logged by the call's owner.
		if isManual {
			go func() {
				ch <- joinedResult(cl)
			}()
		}
		return ch
	}
	c.singleflight.submitRefresh([]*call[K, V]{cl}, func() { c.refreshTask(ctx, cl, loader, ch) })

	return ch
}

// refreshTask runs cl, a refresh that the caller registered, with loader, logs its error, and
// sends its result to ch unless ch is nil.
func (c *cache[K, V]) refreshTask(ctx context.Context, cl *call[K, V], loader Loader[K, V], ch chan<- RefreshResult[K, V]) {
	// Deferred, so that a loader that calls runtime.Goexit is reported like one that panics. The
	// result is sent from a defer of its own, which runs after the log: a Logger.Error that calls
	// runtime.Goexit would skip the rest of a deferred function shared with it, and the caller
	// would wait for the channel forever.
	if ch != nil {
		defer func() {
			ch <- cl.refreshResult()
		}()
	}
	defer func() {
		if cl.err != nil && !cl.isNotFound {
			c.logger.Error(ctx, "Returned an error during the refreshing", cl.err)
		}
	}()

	// the loader is called in a closure, see doCall
	refresher := func(ctx context.Context, key K) (V, error) {
		if cl.base != nil {
			return loader.Reload(ctx, key, c.nodeManager.Value(cl.base))
		}
		return loader.Load(ctx, key)
	}

	//nolint:errcheck // there is no need to check error
	_ = c.wrapRefresh(func() error {
		return c.singleflight.doCall(ctx, cl, refresher)
	})
}

// Get returns the value associated with key in this cache, obtaining that value from loader if necessary.
// The method improves upon the conventional "if cached, return; otherwise create, cache and return" pattern.
//
// Get can return an ErrNotFound error if the Loader returns it, or if Get waits for a BulkGet
// whose BulkLoader leaves the key out of its result.
// This means that the entry was not found in the data source.
//
// If another call to Get is currently loading the value for key,
// simply waits for that goroutine to finish and returns its loaded value. Note that
// multiple goroutines can concurrently load values for distinct keys.
//
// No observable state associated with this cache is modified until loading completes.
//
// WARNING: Loader.Load must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every loader used with it should compute the same value.
// Otherwise, a call that passes one loader may return the result of another call
// with a differently behaving loader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *cache[K, V]) Get(ctx context.Context, key K, loader Loader[K, V]) (V, error) {
	c.singleflight.init()

	nowNano := c.clock.NowNano()
	// The value is read before the refresh starts: the node may be updated in place by then
	// (with a synchronous executor, by the refresh itself).
	n, value := c.getNode(key, nowNano)
	if n != nil {
		// A node that is being removed is not fresh either, but there is nothing to refresh. It is
		// checked after IsFresh: a node retired in between would pass the check before. A node is
		// retired after OnAtomicDeletion, so a refresh requested while the listener runs waits for
		// the bucket lock, and then finds the key's new node.
		if !n.IsFresh(nowNano) && n.IsAlive() {
			c.refreshKey(ctx, c.nodeManager.Key(n), loader, false)
		}
		return value, nil
	}

	for {
		// A Get whose context has ended does not start a load: the load would fail for its
		// context, and a waiter with a live context that joined it would retry, so callers that
		// give up could keep it from loading the key. A key invalidated faster than it loads
		// refuses every load, and the loader may ignore ctx: checked before each retry as well,
		// ctx bounds the wait whatever the loader does.
		if err := ctx.Err(); err != nil {
			return zeroValue[V](), err
		}
		value, retry, err := c.loadOrJoin(ctx, key, loader, nowNano)
		if !retry {
			return value, err
		}
		nowNano = c.clock.NowNano()
	}
}

// loadOrJoin loads the value for key, which Get has not found, or waits for the load already
// running for it. retry reports that the joined load failed for its owner's context while ctx
// is live, so the key must be loaded again.
func (c *cache[K, V]) loadOrJoin(
	ctx context.Context,
	key K,
	loader Loader[K, V],
	nowNano int64,
) (value V, retry bool, err error) {
	cl, shouldLoad := c.singleflight.startLoad(key, nowNano)
	if shouldLoad {
		// Re-check the map before loading. The getNode above runs before this
		// call slot is claimed, so it can miss a value that a concurrently
		// completing load publishes in between, leaving this caller to load the
		// same value a second time. group.finish frees a call slot only after
		// committing the value to the table, so winning the slot means any such
		// value is already visible here; adopt it rather than loading it again. Quiet lookup: the
		// getNode above already recorded the miss for this request.
		if c.getNodeQuietly(key, nowNano) != nil {
			if value, expiresAt, n := c.readForAdoption(cl, nowNano); n != nil {
				c.singleflight.adopt(cl, value, expiresAt, n)
				return value, false, nil
			}
		}
		//nolint:errcheck // there is no need to check error
		_ = c.wrapLoad(func() error {
			// the loader is called in a closure, see doCall
			return c.singleflight.doCall(ctx, cl, func(ctx context.Context, key K) (V, error) {
				return loader.Load(ctx, key)
			})
		})
		if cl.refused {
			value, ok := c.valueAfterRefusal(key)
			return value, !ok, nil
		}
		return cl.value, false, cl.err
	}

	if err := cl.waitContext(ctx); err != nil {
		return zeroValue[V](), false, err
	}
	if cl.refused {
		value, ok := c.valueAfterRefusal(key)
		return value, !ok, nil
	}
	if c.withExpiration && cl.expiredBefore(nowNano) {
		// The Get joined a load after its value had been committed and had expired: a Get that
		// started after the expiration loads the key again.
		return zeroValue[V](), true, nil
	}
	if cl.failedForOwner() && ctx.Err() == nil {
		return zeroValue[V](), true, nil
	}
	return cl.value, false, cl.err
}

// commitCall writes the result of a finished load or refresh, if it may still be written, for
// group.finish: not found removes the entry, an error keeps it (a failed refresh moves its
// refresh time), and a value is written over it.
func (c *cache[K, V]) commitCall(cl *call[K, V]) mutation[K, V] {
	from := fromLoad
	if cl.isRefresh {
		from = fromRefresh
	}
	m := c.mutate(cl.key, from, func(prev previous[K, V]) (mutationOp, V) {
		// From here on, an eviction or expiration of the key drops the call (see
		// group.deleteOnRemoval): it can no longer bring the removed value back.
		cl.committed.Store(true)
		if !cl.isRefresh && prev.live {
			// A load starts on a miss, so a write has given the key a value during the load. The
			// write may not have cancelled the load: a write that inserts or replaces a node
			// cancels the calls before the node is stored, so a load registered in between is not
			// cancelled. The value stays, and the load returns it, whatever it loaded, as a Get that
			// ran after the write would.
			cl.value, cl.err, cl.isNotFound = prev.value, nil, false
			cl.expiresAt, cl.node = c.expiresAtOf(prev.node), prev.node
			return opKeep, cl.value
		}
		authoritative := c.singleflight.authoritative(cl, prev, prev.nowNano)
		cl.refused = !authoritative && (cl.isRefresh || cl.err == nil)
		if authoritative && cl.isNotFound {
			return opInvalidate, cl.value
		}
		if cl.err != nil {
			// A failed load reports its error: no value of the key is involved. A failed refresh
			// of an absent key leaves an expired node that it matched as it is.
			if authoritative && cl.again != nil && cl.failedForOwner() {
				// The failure comes from the context of the Get or BulkGet that ran the load and says
				// nothing about the key, and a manual refresh, which no context cancels, waits
				// for it: the refresh loads the key again, as a joined Get would. Registered here,
				// the refresh starts from the state the load found and is cancelled by every
				// change since the request, as the load would have been.
				cl.again.successor = c.singleflight.succeed(cl)
			}
			if authoritative && cl.isRefresh && prev.live {
				// A failed refresh of a live value keeps it, and a Get that joined the refresh
				// after missing the key returns that value, as a hit would: the reload's error
				// is not a result of the Get's own load. Set first: the refresh calculator below
				// may panic, and the waiters would then get that panic.
				cl.refused = true
				if c.withRefresh {
					entry := c.nodeToEntry(prev.node, prev.nowNano)
					prev.node.SetRefreshableAt(refreshableAtAfterWrite(c.refreshCalculator, reloadFailure, entry, entry.Value, cl.err, prev.nowNano))
				}
			}
			return opKeep, cl.value
		}
		if !authoritative {
			return opKeep, cl.value
		}
		return opWrite, cl.value
	})
	if m.op == opWrite {
		// The node is created after the decision. The expiration time is the one the write gave
		// the value: a write that follows can update the node in place before this line. Both
		// are set before the waiters are woken.
		cl.node = m.next
		if c.withExpiration {
			cl.expiresAt = m.expiresAt
		}
	}
	return m
}

// readForAdoption returns the key's live value, when it expires and its node, for cl, a load that the
// caller registered and finishes with them (group.adopt) instead of loading. The value is read
// under the key's bucket lock, where cl is marked committed: an eviction or expiration of the
// value after that drops cl (group.deleteOnRemoval), so that a Get that starts after the removal
// does not join cl and get the removed value. Read without the lock, the value could be removed
// before cl is finished, and cl would stay registered with it. The node is nil if the key has
// no live value any more.
func (c *cache[K, V]) readForAdoption(cl *call[K, V], nowNano int64) (value V, expiresAt int64, live node.Node) {
	c.hashmap.Compute(cl.key, func(n node.Node) node.Node {
		if n != nil && !n.HasExpired(nowNano) {
			value, expiresAt, live = c.nodeManager.Value(n), c.expiresAtOf(n), n
			cl.committed.Store(true)
		}
		return n
	})
	return value, expiresAt, live
}

// valueAfterRefusal returns the value that a Get or BulkGet takes from a call whose value the
// cache refused (call.refused): a write or an invalidation cancelled it, the node a refresh
// was requested for was replaced or expired, so the value is not one the key had, or a refresh
// of a live value failed, which keeps the value. The key's live
// value is returned, as a hit would return it; ok is false if there is none, and the key is
// loaded again. A load that an invalidation cancelled may have read its data source before the
// change that the invalidation follows.
func (c *cache[K, V]) valueAfterRefusal(key K) (value V, ok bool) {
	if n := c.getNodeQuietly(key, c.clock.NowNano()); n != nil {
		return c.nodeManager.Value(n), true
	}
	return zeroValue[V](), false
}

// expiresAtOf returns when n expires, or 0 if entries do not expire (see call.expiresAt).
func (c *cache[K, V]) expiresAtOf(n node.Node) int64 {
	if !c.withExpiration {
		return 0
	}
	return n.ExpiresAt()
}

func (c *cache[K, V]) bulkRefreshKeys(
	ctx context.Context,
	keys []K,
	bulkLoader BulkLoader[K, V],
	isManual bool,
) <-chan []RefreshResult[K, V] {
	if !c.withRefresh {
		return nil
	}
	var ch chan []RefreshResult[K, V]
	if isManual {
		ch = make(chan []RefreshResult[K, V], 1)
	}
	if len(keys) == 0 {
		if isManual {
			ch <- []RefreshResult[K, V]{}
		}
		return ch
	}

	// Registered before the task is submitted, with the clock read before, as in refreshKey.
	if !isManual {
		// as in refreshKey
		defer recoverCallback(c.logger, "Panicked while starting a refresh")
	}
	var again func(s *call[K, V])
	if isManual {
		again = func(s *call[K, V]) {
			defer recoverCallback(c.logger, "Panicked while starting a refresh")
			c.singleflight.submitRefresh([]*call[K, V]{s}, func() {
				c.bulkRefreshTask(ctx, bulkLoader, map[K]*call[K, V]{s.key: s}, nil)
			})
		}
	}
	nowNano := c.clock.NowNano()
	var (
		toLoadCalls   map[K]*call[K, V]
		toReloadCalls map[K]*call[K, V]
		foundCalls    []*call[K, V]
		owned         []*call[K, V]
		registered    bool
	)
	// startRefresh reads the clock again under each key's bucket lock. If it panics or calls
	// runtime.Goexit for a key, the calls registered for the earlier keys are finished here, as
	// submitRefresh does: left registered, they would block every later load of their keys.
	// Deferred after recoverCallback, so that it runs first and the panic is still logged.
	defer func() {
		if registered {
			return
		}
		r := recover()
		c.singleflight.abandon(owned, r)
		if r != nil {
			panic(r)
		}
	}()
	for i, key := range keys {
		cl, shouldLoad := c.startRefresh(key, isManual, nowNano, again)
		if shouldLoad {
			owned = append(owned, cl)
			if cl.base != nil {
				if toReloadCalls == nil {
					toReloadCalls = make(map[K]*call[K, V], len(keys)-i)
				}
				toReloadCalls[key] = cl
			} else {
				if toLoadCalls == nil {
					toLoadCalls = make(map[K]*call[K, V], len(keys)-i)
				}
				toLoadCalls[key] = cl
			}
		} else if isManual {
			if foundCalls == nil {
				foundCalls = make([]*call[K, V], 0, len(keys)-i)
			}
			foundCalls = append(foundCalls, cl)
		}
	}
	registered = true
	if len(owned) == 0 {
		// Every key is loading or refreshing already: nothing to submit, see refreshKey.
		if isManual {
			sendAfterJoined(ch, make([]RefreshResult[K, V], 0, len(foundCalls)), foundCalls)
		}
		return ch
	}

	c.singleflight.submitRefresh(owned, func() {
		// Deferred, so that a bulk loader that calls runtime.Goexit, like one that panics, does not
		// skip the results. They are sent after the load and the reload, which bulkRefreshTask
		// runs in a defer of its own.
		if isManual {
			defer func() {
				results := make([]RefreshResult[K, V], 0, len(keys))
				for _, cl := range owned {
					results = append(results, cl.refreshResult())
				}
				sendAfterJoined(ch, results, foundCalls)
			}()
		}
		c.bulkRefreshTask(ctx, bulkLoader, toLoadCalls, toReloadCalls)
	})

	return ch
}

// bulkRefreshTask runs the refreshes in toLoadCalls and toReloadCalls, which the caller
// registered, with bulkLoader, and logs their errors.
func (c *cache[K, V]) bulkRefreshTask(
	ctx context.Context,
	bulkLoader BulkLoader[K, V],
	toLoadCalls, toReloadCalls map[K]*call[K, V],
) {
	// Deferred, so that a bulk loader that calls runtime.Goexit, like one that panics, does not
	// skip the reload.
	if len(toReloadCalls) > 0 {
		defer func() {
			reload := func(ctx context.Context, keys []K) (map[K]V, error) {
				oldValues := make([]V, 0, len(keys))
				for _, k := range keys {
					// read when the reload starts, as for a single refresh: a write that changed
					// it since the request has cancelled the refresh
					oldValues = append(oldValues, c.nodeManager.Value(toReloadCalls[k].base))
				}
				return bulkLoader.BulkReload(ctx, keys, oldValues)
			}

			reloadErr := c.wrapRefresh(func() error {
				return c.singleflight.doBulkCall(ctx, toReloadCalls, reload)
			})
			if reloadErr != nil {
				c.logger.Error(ctx, "BulkReload returned an error", reloadErr)
			}
		}()
	}

	if len(toLoadCalls) > 0 {
		loadErr := c.wrapRefresh(func() error {
			// the loader is called in a closure, see doCall
			return c.singleflight.doBulkCall(ctx, toLoadCalls, func(ctx context.Context, keys []K) (map[K]V, error) {
				return bulkLoader.BulkLoad(ctx, keys)
			})
		})
		if loadErr != nil {
			c.logger.Error(ctx, "BulkLoad returned an error", loadErr)
		}
	}
}

// sendAfterJoined sends results to ch with the results of joined, calls that other loads or
// refreshes own, appended once they have finished (see joinedResult). They are waited for in a
// goroutine of its own, not in an executor task, for the reason given in refreshKey.
func sendAfterJoined[K comparable, V any](ch chan<- []RefreshResult[K, V], results []RefreshResult[K, V], joined []*call[K, V]) {
	if len(joined) == 0 {
		ch <- results
		return
	}
	go func() {
		for _, cl := range joined {
			results = append(results, joinedResult(cl))
		}
		ch <- results
	}()
}

// joinedResult waits for cl, a call that a manual refresh joined, and returns the result the
// refresh reports: cl's, or that of the refresh registered in its place if cl is a load that
// failed for its owner's context (see loadAgain).
func joinedResult[K comparable, V any](cl *call[K, V]) RefreshResult[K, V] {
	cl.wait()
	if a := cl.again; a != nil && a.successor != nil {
		a.successor.wait()
		return a.successor.refreshResult()
	}
	return cl.refreshResult()
}

// BulkGet returns the value associated with key in this cache, obtaining that value from loader if necessary.
// The method improves upon the conventional "if cached, return; otherwise create, cache and return" pattern.
//
// If another call to Get (BulkGet) is currently loading the value for key,
// simply waits for that goroutine to finish and returns its loaded value. Note that
// multiple goroutines can concurrently load values for distinct keys.
//
// No observable state associated with this cache is modified until loading completes.
//
// WARNING: BulkLoader.BulkLoad must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every bulkLoader used with it should compute the same value.
// Otherwise, a call that passes one bulkLoader may return the result of another call
// with a differently behaving bulkLoader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *cache[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader BulkLoader[K, V]) (map[K]V, error) {
	c.singleflight.init()

	nowNano := c.clock.NowNano()
	result := make(map[K]V, len(keys))
	var (
		misses    map[K]*call[K, V]
		toRefresh []K
	)
	for _, key := range keys {
		if _, found := result[key]; found {
			continue
		}
		if _, found := misses[key]; found {
			continue
		}

		n, value := c.getNode(key, nowNano)
		if n != nil {
			// a node that is being removed is not refreshed, see Get
			if !n.IsFresh(nowNano) && n.IsAlive() {
				if toRefresh == nil {
					toRefresh = make([]K, 0, len(keys)-len(result))
				}

				toRefresh = append(toRefresh, key)
			}

			result[key] = value
			continue
		}

		if misses == nil {
			misses = make(map[K]*call[K, V], len(keys)-len(result))
		}
		misses[key] = nil
	}

	c.bulkRefreshKeys(ctx, toRefresh, bulkLoader, false)

	var errsFromCalls []error
	for len(misses) > 0 {
		// as in Get
		if err := ctx.Err(); err != nil {
			return result, err
		}
		var err error
		misses, errsFromCalls, err = c.bulkLoadOrJoin(ctx, misses, bulkLoader, result, errsFromCalls)
		if err != nil {
			return result, err
		}
	}

	var err error
	if len(errsFromCalls) > 0 {
		err = errors.Join(errsFromCalls...)
	}

	return result, err
}

// bulkLoadOrJoin loads the values for misses, the keys BulkGet has not found, and waits for the
// loads already running for some of them. It adds the loaded values to result and the errors of
// joined loads to errsFromCalls. It returns the keys whose joined load failed for its owner's
// context while ctx is live, which must be loaded again, and an error that ends BulkGet: the bulk
// load failed or ctx is done.
func (c *cache[K, V]) bulkLoadOrJoin(
	ctx context.Context,
	misses map[K]*call[K, V],
	bulkLoader BulkLoader[K, V],
	result map[K]V,
	errsFromCalls []error,
) (map[K]*call[K, V], []error, error) {
	nowNano := c.clock.NowNano()
	var toLoadCalls map[K]*call[K, V]
	i := 0
	for key := range misses {
		cl, shouldLoad := c.singleflight.startLoad(key, nowNano)
		if shouldLoad {
			// Winning the call slot means that a value written before is visible, see Get.
			if c.getNodeQuietly(key, nowNano) != nil {
				if value, expiresAt, n := c.readForAdoption(cl, nowNano); n != nil {
					c.singleflight.adopt(cl, value, expiresAt, n)
					result[key] = value
					delete(misses, key)
					i++
					continue
				}
			}
			if toLoadCalls == nil {
				toLoadCalls = make(map[K]*call[K, V], len(misses)-i)
			}
			toLoadCalls[key] = cl
		}
		misses[key] = cl
		i++
	}

	var loadErr error
	if len(toLoadCalls) > 0 {
		loadErr = c.wrapLoad(func() error {
			// the loader is called in a closure, see doCall
			return c.singleflight.doBulkCall(ctx, toLoadCalls, func(ctx context.Context, keys []K) (map[K]V, error) {
				return bulkLoader.BulkLoad(ctx, keys)
			})
		})
	}
	if loadErr != nil {
		return nil, errsFromCalls, loadErr
	}

	var retry map[K]*call[K, V]
	i = 0
	for key, cl := range misses {
		if err := cl.waitContext(ctx); err != nil {
			return nil, errsFromCalls, err
		}
		i++

		if cl.refused {
			if value, ok := c.valueAfterRefusal(key); ok {
				result[key] = value
				continue
			}
			if retry == nil {
				retry = make(map[K]*call[K, V], len(misses)-i+1)
			}
			retry[key] = nil
			continue
		}
		// doBulkCall adds a fake call to toLoadCalls for each key the loader returned unasked, so
		// a key this call joined may be there as well: it is its own only if the call it waited
		// for is the one it registered.
		own := toLoadCalls[key] == cl
		if !own && c.withExpiration && cl.expiredBefore(nowNano) {
			// a value that expired before this load, see loadOrJoin
			if retry == nil {
				retry = make(map[K]*call[K, V], len(misses)-i+1)
			}
			retry[key] = nil
			continue
		}
		if cl.err == nil {
			result[key] = cl.Value()
			continue
		}
		if own || cl.isNotFound {
			continue
		}
		if cl.failedForOwner() && ctx.Err() == nil {
			if retry == nil {
				retry = make(map[K]*call[K, V], len(misses)-i+1)
			}
			retry[key] = nil
			continue
		}
		if errsFromCalls == nil {
			errsFromCalls = make([]error, 0, len(misses)-i+1)
		}
		errsFromCalls = append(errsFromCalls, cl.err)
	}

	return retry, errsFromCalls, nil
}

func (c *cache[K, V]) wrapLoad(fn func() error) (err error) {
	startTime := c.statsClock.NowNano()

	// fn panics if writing a loaded value panics. The load is still recorded, as a failure,
	// since the caller gets the panic instead of the value.
	completed := false
	defer func() {
		loadTime := time.Duration(c.statsClock.NowNano() - startTime)
		if completed && (err == nil || errors.Is(err, ErrNotFound)) {
			c.stats.RecordLoadSuccess(loadTime)
		} else {
			c.stats.RecordLoadFailure(loadTime)
		}
	}()

	err = fn()
	completed = true

	var pe *panicError
	if errors.As(err, &pe) {
		panic(pe)
	}

	return err
}

// wrapRefresh is wrapLoad for refreshes, which run on the executor: a panic of the loader, or of
// writing its result, is returned as an error instead of crashing the executor's goroutine. The
// refreshed calls already hold that error, so it reaches the refresh results and the log like
// any other refresh error.
func (c *cache[K, V]) wrapRefresh(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(*panicError); ok {
				err = pe
				return
			}
			err = newPanicError(r)
		}
	}()
	return c.wrapLoad(fn)
}

// Refresh loads a new value for the key, asynchronously. While the new value is loading the
// previous value (if any) will continue to be returned by any Get unless it is evicted.
// If the new value is loaded successfully, it will replace the previous value in the cache;
// If refreshing returned an error, the previous value will remain, and the error will be logged
// using Logger and swallowed, except for [ErrNotFound], which removes the entry. If a load or refresh of the
// key that started from its current state is in flight, this method joins it instead of loading again.
// If the key is written, invalidated or expires after the refresh was requested, the loaded
// value is discarded and the cache keeps the newer state.
//
// cache will call Loader.Reload if the cache currently contains a value for the key,
// and Loader.Load otherwise.
//
// WARNING: Loader.Load and Loader.Reload must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every loader used with it should compute the same value.
// Otherwise, a call that passes one loader may return the result of another call
// with a differently behaving loader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *cache[K, V]) Refresh(ctx context.Context, key K, loader Loader[K, V]) <-chan RefreshResult[K, V] {
	if !c.withRefresh {
		return nil
	}

	c.singleflight.init()

	return c.refreshKey(ctx, key, loader, true)
}

// BulkRefresh loads a new value for each key, asynchronously. While the new value is loading the
// previous value (if any) will continue to be returned by any Get unless it is evicted.
// If the new value is loaded successfully, it will replace the previous value in the cache;
// If refreshing returned an error, the previous values will remain, and the error will be logged
// using Logger and swallowed, an [ErrNotFound] for the whole bulk included. A key that the
// [BulkLoader] leaves out of its result is not found, and its entry is removed. If a load or refresh of the
// key that started from its current state is in flight, this method joins it instead of loading again.
// If the key is written, invalidated or expires after the refresh was requested, the loaded
// value is discarded and the cache keeps the newer state.
//
// cache will call BulkLoader.BulkReload for existing keys, and BulkLoader.BulkLoad otherwise.
//
// WARNING: BulkLoader.BulkLoad and BulkLoader.BulkReload must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every bulkLoader used with it should compute the same value.
// Otherwise, a call that passes one bulkLoader may return the result of another call
// with a differently behaving loader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *cache[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader BulkLoader[K, V]) <-chan []RefreshResult[K, V] {
	if !c.withRefresh {
		return nil
	}

	c.singleflight.init()

	uniq := make(map[K]struct{}, len(keys))
	toRefresh := make([]K, 0, len(keys))
	for _, k := range keys {
		if _, ok := uniq[k]; !ok {
			uniq[k] = struct{}{}
			toRefresh = append(toRefresh, k)
		}
	}

	return c.bulkRefreshKeys(ctx, toRefresh, bulkLoader, true)
}

// Invalidate discards any cached value for the key.
//
// Returns previous value if any. The invalidated result reports whether the key was
// present; an expired entry that has not been removed yet is not.
func (c *cache[K, V]) Invalidate(key K) (value V, invalidated bool) {
	m := c.mutate(key, fromCaller, func(prev previous[K, V]) (mutationOp, V) {
		return opInvalidate, prev.value
	})
	c.publish(m)
	// An expired entry that was still in the table is removed, but it was not present.
	if m.live {
		return m.value, true
	}
	return zeroValue[V](), false
}

// nodeRemoval is why maintenance removes a given node, rather than whatever its key holds.
type nodeRemoval uint8

const (
	// removeForSize: the eviction policy chose the node. The node is kept if a writer has set its
	// weight to zero in place since then: a zero-weight entry is never evicted for size.
	removeForSize nodeRemoval = iota
	// removeExpired: the timer wheel found the node expired. The node is kept if a writer has
	// updated it in place since then, extending its lifetime.
	removeExpired
	// removeInvalidated: InvalidateAll discards the node, and cancels the key's pending loads.
	removeInvalidated
)

// cause is the deletion cause of a removal, unless the node turns out to have expired.
func (r nodeRemoval) cause() DeletionCause {
	switch r {
	case removeForSize:
		return CauseOverflow
	case removeExpired:
		return CauseExpiration
	default:
		return CauseInvalidation
	}
}

// deleteNodeFromMap removes n from the hash table if it is still there, counts the removal in the
// stats if it is an eviction (countEviction), and returns the cause reported to the synchronous
// listener. kept reports that n stays, as described by removal.
func (c *cache[K, V]) deleteNodeFromMap(
	n node.Node,
	nowNano int64,
	removal nodeRemoval,
) (deleted node.Node, deletedCause DeletionCause, kept bool) {
	c.hashmap.Compute(c.nodeManager.Key(n), func(current node.Node) node.Node {
		// n is not nil, so a current n is in the table
		isCurrent := n == current
		if isCurrent {
			switch {
			case removal == removeExpired && !current.HasExpired(nowNano):
				// after the wheel found it expired, a write, read or SetExpiresAfter that
				// saw it live moved its deadline later
				kept = true
				return current
			case removal == removeForSize && c.isWeighted && current.Weight() == 0 &&
				!current.HasExpired(nowNano):
				// The policy chose the node by the weight it accounted, but a writer has since set
				// its weight to zero in place. The node stays linked until its reweigh task brings
				// the accounted weight to zero.
				kept = true
				return current
			}
		}
		if removal == removeInvalidated {
			c.singleflight.invalidate(c.nodeManager.Key(n))
		} else if isCurrent {
			// An eviction must not cancel an in-flight load: it was most likely started
			// because this very node has expired, and dropping it would make the loaded
			// value silently disappear. Only a pending refresh of the evicted value, or a load
			// that has already committed it, is dropped, and only if n is still the key's node:
			// the policy may evict a node that was already replaced, and the call then belongs
			// to the value that replaced it.
			c.singleflight.deleteOnRemoval(c.nodeManager.Key(n))
		}
		if !isCurrent {
			return current
		}
		deleted = current
		deletedCause = removalCause(!deleted.HasExpired(nowNano), removal.cause())
		c.retire(deleted, c.nodeManager.Value(deleted), deletedCause)
		return nil
	})
	if deleted != nil {
		// A retired node is never updated in place, so its weight is the one it was removed with.
		c.countEviction(deletedCause, deleted.Weight())
	}
	return deleted, deletedCause, kept
}

func (c *cache[K, V]) deleteNode(n node.Node, nowNano int64) bool {
	deleted, cause, _ := c.deleteNodeFromMap(n, nowNano, removeInvalidated)
	c.afterDelete(deleted, cause, true, false)
	return deleted != nil
}

// afterDelete takes the cause that the synchronous listener was given under the lock.
func (c *cache[K, V]) afterDelete(deleted node.Node, cause DeletionCause, alreadyLocked, onExecutor bool) {
	if deleted == nil {
		return
	}

	// Reported by the removal, before its task, as afterWrite reports a write.
	if c.withMaintenance {
		defer c.afterDeletePolicies(deleted, alreadyLocked)
	}
	c.notifyDeletionFrom(onExecutor, c.nodeManager.Key(deleted), c.nodeManager.Value(deleted), cause)
}

// afterDeletePolicies hands a removal to the policies.
func (c *cache[K, V]) afterDeletePolicies(deleted node.Node, alreadyLocked bool) {
	t := newTask(deleted, nil, deleteReason)
	if alreadyLocked {
		c.runTask(t)
	} else {
		c.afterWriteTask(t)
	}
}

// countEviction is called for every removal, and counts it in the stats when the cause the
// listeners are given is an eviction, so that the two never disagree. It is called once the
// bucket lock is released, and before the OnDeletion listener is notified, so that the listener
// sees the eviction it is told about. weight is the removed entry's weight, read when it was
// removed; causeUnknown means that nothing was removed.
func (c *cache[K, V]) countEviction(cause DeletionCause, weight uint32) {
	if cause != causeUnknown && cause.IsEviction() {
		c.stats.RecordEviction(weight)
	}
}

// notifyDeletionFrom calls the OnDeletion listener directly if the caller already runs as an
// executor task, and submits it to the executor otherwise. Submitting another task from a task
// deadlocks an executor that runs a bounded number of tasks and blocks the submitter: once all
// of its slots are taken by such callers, none of them can finish.
func (c *cache[K, V]) notifyDeletionFrom(onExecutor bool, key K, value V, cause DeletionCause) {
	if c.onDeletion == nil {
		return
	}

	if onExecutor {
		c.callOnDeletion(key, value, cause)
		return
	}

	// A panic of the listener, or of a custom executor submitting it, is logged: this may run
	// during maintenance, under the eviction lock.
	defer recoverCallback(c.logger, "Executor panicked while submitting OnDeletion")
	c.executor(func() {
		c.callOnDeletion(key, value, cause)
	})
}

func (c *cache[K, V]) callOnDeletion(key K, value V, cause DeletionCause) {
	defer recoverCallback(c.logger, "OnDeletion panicked")
	c.onDeletion(DeletionEvent[K, V]{
		Key:   key,
		Value: value,
		Cause: cause,
	})
}

func (c *cache[K, V]) notifyAtomicDeletion(key K, value V, cause DeletionCause) {
	if c.onAtomicDeletion == nil {
		return
	}

	// The listener runs under the hash table's bucket lock, before the change is visible: an update
	// in place calls it before the new value is set, and a replacement or removal before the node
	// is retired (see retire) and the table is changed. A panic is logged so that the change
	// completes. A runtime.Goexit cannot be stopped: the node stays alive and in the table, and the
	// key's calls that the change cancelled before the listener stay cancelled.
	defer recoverCallback(c.logger, "OnAtomicDeletion panicked")
	c.onAtomicDeletion(DeletionEvent[K, V]{
		Key:   key,
		Value: value,
		Cause: cause,
	})
}

// periodicCleanUp holds the cache only while it cleans it up, so that the cache can be collected
// between the ticks; then the goroutine returns at the next tick. It gets the channels, not the
// cache, because it must not keep the cache reachable: a callback that the cache holds may hold
// the cache itself, and then it would never be collected. It returns when stop is closed, and
// closes done then. On every return it stops the ticks, if the clock can stop them.
//
// A clean-up runs user code (the listeners, the clock, the recorder and the executor), which may
// call runtime.Goexit. That ends the goroutine, and the loop goes on in a new one with the same
// ticks: the cache keeps its periodic clean-ups, and done is closed once. The ticks are nil only
// in the first goroutine, and a Goexit in Clock.Tick ends the periodic clean-ups, as a panic of
// Tick does: a new goroutine would call Tick again, and exit again.
func periodicCleanUp[K comparable, V any](
	wc weak.Pointer[cache[K, V]],
	tick <-chan time.Time,
	stopTick func(),
	stop, done chan struct{},
) {
	returned := false
	defer func() {
		if !returned && tick != nil {
			go periodicCleanUp(wc, tick, stopTick, stop, done)
			return
		}
		if stopTick != nil {
			stopTick()
		}
		close(done)
	}()

	if tick == nil {
		c := wc.Value()
		if c == nil {
			return
		}
		tick, stopTick = c.tick()
		if tick == nil {
			// Without ticks, nothing would tell the goroutine that the cache is gone.
			return
		}
	}
	for {
		select {
		case <-stop:
			returned = true
			return
		case _, ok := <-tick:
			if !ok {
				// A closed channel is always ready: the loop would spin and keep the cache
				// reachable. It means no more ticks, as a nil one does.
				returned = true
				return
			}
			c := wc.Value()
			if c == nil {
				returned = true
				return
			}
			c.cleanUpOnTick()
			c.clock.ProcessTick()
		}
	}
}

// cleanUpOnTick does not wait while evictionMutex is held: waiting would make StopAllGoroutines,
// called by the holder, wait for its own caller. It marks a drain required instead, as a write
// does, so that the holder runs one when it releases the lock (on the default executor; see
// rescheduleCleanUpIfIncomplete), with the time of the tick or later. A pass that holds the lock
// may have read the time before the tick, and there may be no other tick or write to come.
func (c *cache[K, V]) cleanUpOnTick() {
	if !c.evictionMutex.TryLock() {
		c.scheduleAfterWrite()
		return
	}
	c.maintenanceAndUnlock(nil)
	c.rescheduleCleanUpIfIncomplete()
}

// tick returns the clock's ticks for the periodic clean-up. The clock is user code, and that
// goroutine is the cache's own: if Tick panics, the panic is logged, nil is returned, and the cache
// does without periodic clean-ups. So does it if Tick calls runtime.Goexit (see periodicCleanUp).
func (c *cache[K, V]) tick() (tick <-chan time.Time, stopTick func()) {
	defer recoverCallback(c.logger, "Clock.Tick panicked")
	return c.clock.Ticker(time.Second)
}

// evictNode removes n, which the eviction policy chose to evict for size, and reports whether n
// leaves the policy, which then unlinks it. The policy has no clock, so the time of the
// maintenance pass is used: an entry that has expired, but that the timer wheel has not reached
// yet, is reported as expired, as every other path reports it.
func (c *cache[K, V]) evictNode(n node.Node) bool {
	return c.removeNode(n, c.maintenanceNanos, removeForSize)
}

// expireNode is called by the timer wheel for a node whose expiration time has passed, and
// reports whether it removed the node. The wheel reads that time before the hash table's lock is
// taken, and a writer can update the node in place in between, extending its lifetime. Such a
// node stays in the cache, and the wheel schedules it again.
func (c *cache[K, V]) expireNode(n node.Node, nowNanos int64) bool {
	if !c.removeNode(n, nowNanos, removeExpired) {
		return false
	}
	c.makeDead(n)
	return true
}

// removeNode removes n from the hash table and the timer wheel, and reports the removal unless n
// had already left the table (it was replaced or removed, and that is reported by the mutation
// that did it).
// It returns false if n is kept, as described by removal. The caller unlinks n from the eviction
// policy.
func (c *cache[K, V]) removeNode(n node.Node, nowNanos int64, removal nodeRemoval) bool {
	// Once n has left the table, user code may call runtime.Goexit: RecordEviction, which counts
	// the removal, OnDeletion, or the executor OnDeletion is submitted to. Then the caller never
	// unlinks n, and its walk does not resume, so n is unlinked here. After a Goexit in
	// OnAtomicDeletion, n is still alive and in the table (see retire), and it stays linked.
	finished := false
	defer func() {
		if !finished && n.IsRetired() {
			if c.withExpiration {
				c.timerWheel.Unschedule(n)
			}
			c.makeDead(n)
		}
	}()
	deleted, cause, kept := c.deleteNodeFromMap(n, nowNanos, removal)
	if kept {
		finished = true
		return false
	}
	if c.withExpiration {
		c.timerWheel.Unschedule(n)
	}
	if deleted != nil {
		c.notifyDeletionFrom(false, c.nodeManager.Key(n), c.nodeManager.Value(n), cause)
	}
	finished = true
	return true
}

// current returns the key's node for n, a node that an iteration found before it reports it:
// n itself while it is the key's node, also while a write is replacing or removing it; the node
// that replaced it; or nil if the key has been removed. A replaced node would otherwise be skipped,
// and its key with it: the hash table's Range copies a bucket's nodes and reports them after
// releasing its lock, and in a bounded cache the node that replaced it is not in the policy until
// its task runs. Each key is reported once: Range copies a bucket with all of its keys at once, and
// a replacing node is not linked into the policy while it is walked under evictionMutex.
func (c *cache[K, V]) current(n node.Node) node.Node {
	if n.IsAlive() {
		return n
	}
	return c.hashmap.Get(c.nodeManager.Key(n))
}

func (c *cache[K, V]) nodes() iter.Seq[node.Node] {
	return func(yield func(node.Node) bool) {
		c.hashmap.Range(func(n node.Node) bool {
			nowNano := c.clock.NowNano()
			if n = c.current(n); n == nil || n.HasExpired(nowNano) {
				c.scheduleDrainBuffers()
				return true
			}

			return yield(n)
		})
	}
}

func (c *cache[K, V]) entries() iter.Seq[Entry[K, V]] {
	return func(yield func(Entry[K, V]) bool) {
		for n := range c.nodes() {
			if !yield(c.nodeToEntry(n, c.clock.NowNano())) {
				return
			}
		}
	}
}

// All returns an iterator over all entries in the cache.
//
// Iterator is at least weakly consistent: he is safe for concurrent use,
// but if the cache is modified (including by eviction) after the iterator is
// created, it is undefined which of the changes (if any) will be reflected in that iterator.
func (c *cache[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for n := range c.nodes() {
			if !yield(c.nodeManager.Key(n), c.nodeManager.Value(n)) {
				return
			}
		}
	}
}

// Keys returns an iterator over all keys in the cache.
// The iteration order is not specified and is not guaranteed to be the same from one call to the next.
//
// Iterator is at least weakly consistent: he is safe for concurrent use,
// but if the cache is modified (including by eviction) after the iterator is
// created, it is undefined which of the changes (if any) will be reflected in that iterator.
func (c *cache[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for n := range c.nodes() {
			if !yield(c.nodeManager.Key(n)) {
				return
			}
		}
	}
}

// Values returns an iterator over all values in the cache.
// The iteration order is not specified and is not guaranteed to be the same from one call to the next.
//
// Iterator is at least weakly consistent: he is safe for concurrent use,
// but if the cache is modified (including by eviction) after the iterator is
// created, it is undefined which of the changes (if any) will be reflected in that iterator.
func (c *cache[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for n := range c.nodes() {
			if !yield(c.nodeManager.Value(n)) {
				return
			}
		}
	}
}

// InvalidateAll discards all entries in the cache. The behavior of this operation is undefined for an entry
// that is being loaded (or reloaded) and is otherwise not present.
func (c *cache[K, V]) InvalidateAll() {
	nodes := c.invalidateAllLocked()
	// Writers that ran while the lock was held could not take it, and left a drain required.
	c.rescheduleCleanUpIfIncomplete()

	for _, n := range nodes {
		c.Invalidate(c.nodeManager.Key(n))
	}
}

// invalidateAllLocked discards entries under evictionMutex while the write buffer has room, and
// returns the rest to be invalidated one by one, together with the keys whose node was replaced
// after the snapshot.
func (c *cache[K, V]) invalidateAllLocked() []node.Node {
	c.evictionMutex.Lock()
	defer c.evictionMutex.Unlock()

	if c.withMaintenance {
		// the replayed writes can evict
		c.readMaintenanceTime()
		c.readBuffer.DrainTo(func(n node.Node) {})
		for {
			t := c.writeBuffer.TryPop()
			if t == nil {
				break
			}
			c.runTask(t)
		}
	}
	// Discard all entries, falling back to one-by-one to avoid excessive lock hold times
	nodes := make([]node.Node, 0, c.EstimatedSize())
	threshold := uint64(maxWriteBufferSize / 2)
	c.hashmap.Range(func(n node.Node) bool {
		nodes = append(nodes, n)
		return true
	})
	nowNano := c.clock.NowNano()
	var replaced []node.Node
	for len(nodes) > 0 && c.writeBuffer.Size() < threshold {
		n := nodes[len(nodes)-1]
		nodes = nodes[:len(nodes)-1]
		if !c.deleteNode(n, nowNano) && c.hashmap.Get(c.nodeManager.Key(n)) != nil {
			// The key's node was replaced after the snapshot, e.g. by a reload that completed in
			// the meantime. The new node's insertion may still be in the write buffer, so it is
			// not removed here under the lock: the key is invalidated with the rest.
			replaced = append(replaced, n)
		}
	}
	return append(nodes, replaced...)
}

// CleanUp performs any pending maintenance operations needed by the cache. Exactly which activities are
// performed -- if any -- is implementation-dependent.
func (c *cache[K, V]) CleanUp() {
	c.performCleanUp(nil)
}

func (c *cache[K, V]) shouldDrainBuffers(delayable bool) bool {
	drainStatus := c.drainStatus.Load()
	switch drainStatus {
	case idle:
		return !delayable
	case required:
		return true
	case processingToIdle, processingToRequired:
		return false
	default:
		panic(fmt.Sprintf("Invalid drain status: %d", drainStatus))
	}
}

func (c *cache[K, V]) skipReadBuffer() bool {
	return !c.withMaintenance || // without read buffer
		(!c.withExpiration && c.withEviction && c.evictionPolicy.sketch.isNotInitialized())
}

func (c *cache[K, V]) afterWriteTask(t *task) {
	// In scenarios where the writing goroutines cannot make progress then they attempt to provide
	// assistance by performing the eviction work directly. This can resolve cases where the
	// maintenance task is scheduled but not running. Deferred, so that t is not lost when the
	// executor, or maintenance run by a synchronous one, calls runtime.Goexit while the buffer is
	// full.
	pushed := false
	defer func() {
		if !pushed {
			c.performCleanUp(t)
		}
	}()
	for i := 0; i < writeBufferRetries; i++ {
		if c.writeBuffer.TryPush(t) {
			pushed = true
			c.scheduleAfterWrite()
			return
		}
		c.scheduleDrainBuffers()
		runtime.Gosched()
	}
}

func (c *cache[K, V]) scheduleAfterWrite() {
	for {
		drainStatus := c.drainStatus.Load()
		switch drainStatus {
		case idle:
			c.drainStatus.CompareAndSwap(idle, required)
			c.scheduleDrainBuffers()
			return
		case required:
			c.scheduleDrainBuffers()
			return
		case processingToIdle:
			if c.drainStatus.CompareAndSwap(processingToIdle, processingToRequired) {
				return
			}
		case processingToRequired:
			return
		default:
			panic(fmt.Sprintf("Invalid drain status: %d", drainStatus))
		}
	}
}

func (c *cache[K, V]) scheduleDrainBuffers() {
	if c.drainStatus.Load() >= processingToIdle {
		return
	}

	if c.evictionMutex.TryLock() {
		drainStatus := c.drainStatus.Load()
		if drainStatus >= processingToIdle {
			c.evictionMutex.Unlock()
			return
		}

		c.drainStatus.Store(processingToIdle)

		token := (c.drainToken.Load() | 1) + 1
		c.drainToken.Store(token)
		// If the drain did not take over the lock, it is released here, also when a custom executor,
		// or maintenance run by a synchronous one, panics or calls runtime.Goexit. Then the drain is
		// left required for the next attempt, and a panic is logged: the caller's write is already
		// done.
		returned := false
		defer func() {
			r := recover()
			if c.drainToken.CompareAndSwap(token, token+1) {
				if !returned {
					c.drainStatus.Store(required)
				}
				c.evictionMutex.Unlock()
			}
			if r != nil {
				c.logger.Error(context.Background(), "Maintenance panicked", newPanicError(r))
			}
		}()
		c.executor(c.drainBuffersFunc)
		returned = true
	}
}

func (c *cache[K, V]) drainBuffers() {
	if c.evictionMutex.TryLock() {
		c.maintenanceAndUnlock(nil)
		c.rescheduleCleanUpIfIncomplete()
	} else {
		// already locked
		if token := c.drainToken.Load(); token&1 == 0 && c.drainToken.CompareAndSwap(token, token+1) {
			// executor is sync
			c.maintenanceAndUnlock(nil)
			c.rescheduleCleanUpIfIncomplete()
		} else {
			// executor is async
			c.performCleanUp(nil)
		}
	}
}

func (c *cache[K, V]) performCleanUp(t *task) {
	c.evictionMutex.Lock()
	c.maintenanceAndUnlock(t)
	c.rescheduleCleanUpIfIncomplete()
}

// maintenanceAndUnlock runs maintenance with evictionMutex held by the caller and releases it,
// also when maintenance panics, so that a panic does not stop maintenance and every writer.
func (c *cache[K, V]) maintenanceAndUnlock(t *task) {
	defer c.evictionMutex.Unlock()
	c.maintenance(t)
}

func (c *cache[K, V]) rescheduleCleanUpIfIncomplete() {
	if c.drainStatus.Load() != required {
		return
	}

	// An immediate scheduling cannot be performed on a custom executor because it may use a
	// caller-runs policy. This could cause the caller's penalty to exceed the amortized threshold,
	// e.g. repeated concurrent writes could result in a retry loop. The drain then waits for the
	// next read, write or periodic clean-up (see Options.Executor).
	if c.hasDefaultExecutor {
		c.scheduleDrainBuffers()
		return
	}
}

func (c *cache[K, V]) maintenance(t *task) {
	completed := false
	// The writer's task t is not in the write buffer, so the pass runs it even if replaying the
	// buffer stops it: a task that reports an eviction may run an OnDeletion, or hand it to an
	// executor, that panics or calls runtime.Goexit. The order of the tasks does not matter to
	// the policies. A task that was started is not run again.
	ranTask := t == nil
	defer func() {
		if !completed {
			// Interrupted by a panic or runtime.Goexit, also of the clock: the buffers may still
			// hold work. The status is stored first, since the writer's task may run user code
			// that exits again.
			c.drainStatus.Store(required)
			if !ranTask {
				c.runTask(t)
			}
		}
	}()
	c.drainStatus.Store(processingToIdle)
	c.readMaintenanceTime()

	c.drainReadBuffer()
	c.drainWriteBuffer()
	ranTask = true
	c.runTask(t)
	c.expireNodes()
	c.evictNodes()
	c.climb()
	completed = true

	if c.drainStatus.Load() != processingToIdle || !c.drainStatus.CompareAndSwap(processingToIdle, idle) {
		c.drainStatus.Store(required)
	}
}

// readMaintenanceTime reads the time of a maintenance pass, under evictionMutex. Reading the clock
// once per pass rather than per eviction keeps it off the single-threaded maintenance path.
//
// The clock is user code. If it panics, the pass runs with the time of the previous one and the
// panic is logged: the pass may hold a task that a writer handed it, which would be lost, and on
// the default executor the panic would crash the program. Entries then expire a pass later.
func (c *cache[K, V]) readMaintenanceTime() {
	if c.withExpiration {
		defer recoverCallback(c.logger, "Clock panicked during maintenance")
		c.maintenanceNanos = c.clock.NowNano()
	}
}

func (c *cache[K, V]) drainReadBuffer() {
	if c.skipReadBuffer() {
		return
	}

	c.readBuffer.DrainTo(c.onAccess)
}

func (c *cache[K, V]) drainWriteBuffer() {
	if !c.withMaintenance {
		return
	}

	for i := uint32(0); i <= maxWriteBufferSize; i++ {
		t := c.writeBuffer.TryPop()
		if t == nil {
			return
		}
		c.runTask(t)
	}
	c.drainStatus.Store(processingToRequired)
}

func (c *cache[K, V]) runTask(t *task) {
	if t == nil {
		return
	}

	n := t.node()
	switch t.writeReason {
	case addReason:
		if c.withExpiration {
			c.timerWheel.Schedule(n)
		}
		if c.withEviction {
			c.evictionPolicy.add(n)
		}
	case updateReason:
		old := t.oldNode()
		if c.withExpiration {
			c.timerWheel.Unschedule(old)
			c.timerWheel.Schedule(n)
		}
		if c.withEviction {
			c.evictionPolicy.update(n, old, t.quiet)
		}
	case reconcileReason:
		// A weight or an expiration time changed in place. Both are brought in line with the
		// node's current state, so replaying the task late, twice or out of order is harmless.
		if c.withEviction {
			c.evictionPolicy.reweigh(n)
		}
		if c.withExpiration {
			c.timerWheel.Schedule(n)
		}
	case deleteReason:
		if c.withExpiration {
			c.timerWheel.Unschedule(n)
		}
		if c.withEviction {
			c.evictionPolicy.makeDead(n)
		}
	default:
		panic(fmt.Sprintf("Invalid task type: %d", t.writeReason))
	}
}

func (c *cache[K, V]) onAccess(n node.Node) {
	if c.withEviction {
		c.evictionPolicy.access(n)
	}
	if c.withExpiration {
		c.timerWheel.Schedule(n)
	}
}

func (c *cache[K, V]) expireNodes() {
	if c.withExpiration {
		c.timerWheel.Advance(c.maintenanceNanos, c.expireNode)
	}
}

func (c *cache[K, V]) evictNodes() {
	if !c.withEviction {
		return
	}
	c.evictionPolicy.evictNodes()
}

func (c *cache[K, V]) climb() {
	if !c.withEviction {
		return
	}
	c.evictionPolicy.climb()
}

// newTask allocates a task. Tasks are not pooled: writers on every P would take them and
// maintenance on one P would return them, so writers would mostly steal from the pool of that P,
// which costs more than the allocation.
func newTask(n, old node.Node, writeReason reason) *task {
	return &task{
		n:           n,
		old:         old,
		writeReason: writeReason,
	}
}

// SetMaximum specifies the maximum total size of this cache. This value may be interpreted as the weighted
// or unweighted threshold size based on how this cache was constructed. If the cache currently
// exceeds the new maximum size this operation eagerly evict entries until the cache shrinks to
// the appropriate size.
func (c *cache[K, V]) SetMaximum(maximum uint64) {
	if !c.withEviction {
		return
	}
	c.evictionMutex.Lock()
	c.evictionPolicy.setMaximumSize(maximum)
	c.maintenanceAndUnlock(nil)
	c.rescheduleCleanUpIfIncomplete()
}

// GetMaximum returns the maximum total weighted or unweighted size of this cache, depending on how the
// cache was constructed. If this cache does not use a (weighted) size bound, then the method will return math.MaxUint64.
func (c *cache[K, V]) GetMaximum() uint64 {
	if !c.withEviction {
		return uint64(math.MaxUint64)
	}

	result := func() uint64 {
		c.evictionMutex.Lock()
		defer c.evictionMutex.Unlock()
		if c.drainStatus.Load() == required {
			c.maintenance(nil)
		}
		return c.evictionPolicy.maximum
	}()
	c.rescheduleCleanUpIfIncomplete()
	return result
}

func (c *cache[K, V]) StopAllGoroutines() bool {
	if c.stop == nil {
		return false
	}
	stopped := false
	c.stopOnce.Do(func() {
		close(c.stop)
		stopped = true
	})
	<-c.done
	return stopped
}

// EstimatedSize returns the approximate number of entries in this cache. The value returned is an estimate; the
// actual count may differ if there are concurrent insertions or deletions, or if some entries are
// pending deletion due to expiration. In the case of stale entries
// this inaccuracy can be mitigated by performing a CleanUp first.
func (c *cache[K, V]) EstimatedSize() int {
	return c.hashmap.Size()
}

// IsWeighted returns whether the cache is bounded by a maximum weight.
func (c *cache[K, V]) IsWeighted() bool {
	return c.isWeighted
}

// IsRecordingStats returns whether the cache statistics are being accumulated.
func (c *cache[K, V]) IsRecordingStats() bool {
	return c.withStats
}

// Stats returns a current snapshot of this cache's cumulative statistics.
// All statistics are initialized to zero and are monotonically increasing over the lifetime of the cache.
// Due to the performance penalty of maintaining statistics,
// some implementations may not record the usage history immediately or at all.
//
// NOTE: If your [stats.Recorder] implementation doesn't also implement [stats.Snapshoter],
// this method will always return a zero-value snapshot.
func (c *cache[K, V]) Stats() stats.Stats {
	return c.statsSnapshoter.Snapshot()
}

// WeightedSize returns the approximate accumulated weight of entries in this cache. If this cache does not
// use a weighted size bound, then the method will return 0.
func (c *cache[K, V]) WeightedSize() uint64 {
	if !c.isWeighted {
		return 0
	}

	result := func() uint64 {
		c.evictionMutex.Lock()
		defer c.evictionMutex.Unlock()
		if c.drainStatus.Load() == required {
			c.maintenance(nil)
		}
		return c.evictionPolicy.weightedSize
	}()
	c.rescheduleCleanUpIfIncomplete()
	return result
}

// Hottest returns an iterator for ordered traversal of the cache entries. The order of
// iteration is from the entries most likely to be retained (hottest) to the entries least
// likely to be retained (coldest). This order is determined by the eviction policy's best guess
// at the start of the iteration.
//
// WARNING: Beware that this iteration is performed within the eviction policy's exclusive lock, so the
// iteration should be short and simple. While the iteration is in progress further eviction
// maintenance will be halted.
func (c *cache[K, V]) Hottest() iter.Seq[Entry[K, V]] {
	return c.evictionOrder(true)
}

// Coldest returns an iterator for ordered traversal of the cache entries. The order of
// iteration is from the entries least likely to be retained (coldest) to the entries most
// likely to be retained (hottest). This order is determined by the eviction policy's best guess
// at the start of the iteration.
//
// WARNING: Beware that this iteration is performed within the eviction policy's exclusive lock, so the
// iteration should be short and simple. While the iteration is in progress further eviction
// maintenance will be halted.
func (c *cache[K, V]) Coldest() iter.Seq[Entry[K, V]] {
	return c.evictionOrder(false)
}

func (c *cache[K, V]) evictionOrder(hottest bool) iter.Seq[Entry[K, V]] {
	if !c.withEviction {
		return c.entries()
	}

	return func(yield func(Entry[K, V]) bool) {
		seq := c.evictionPolicy.coldest()
		if hottest {
			seq = c.evictionPolicy.hottest()
		}

		// Deferred before the unlock, so that it runs after it: writers that ran during the
		// iteration could not take the lock, and left a drain required.
		defer c.rescheduleCleanUpIfIncomplete()
		c.evictionMutex.Lock()
		defer c.evictionMutex.Unlock()
		c.maintenance(nil)

		for n := range seq {
			nowNano := c.clock.NowNano()
			if n = c.current(n); n == nil || n.HasExpired(nowNano) {
				continue
			}
			if !yield(c.nodeToEntry(n, nowNano)) {
				return
			}
		}
	}
}

// retire reports the removal or replacement of n, which held value, to OnAtomicDeletion and
// retires n, under the bucket lock that the caller holds. The listener runs first: if it calls
// runtime.Goexit, n stays alive and in the table, and the timer wheel and the policy still hold
// it, so that a removal by maintenance is retried on its next pass. Retired first, an expired n
// would also leave the timer wheel, which does not reschedule a retired node, and in a cache
// without a maximum size only a write of its key would remove it.
func (c *cache[K, V]) retire(n node.Node, value V, cause DeletionCause) {
	c.notifyAtomicDeletion(c.nodeManager.Key(n), value, cause)
	c.makeRetired(n)
}

func (c *cache[K, V]) makeRetired(n node.Node) {
	if n != nil && c.withMaintenance && n.IsAlive() {
		n.Retire()
	}
}

func (c *cache[K, V]) makeDead(n node.Node) {
	if !c.withMaintenance {
		return
	}

	if c.withEviction {
		c.evictionPolicy.makeDead(n)
	} else if !n.IsDead() {
		n.Die()
	}
}
