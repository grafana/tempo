// Copyright (c) 2025 Alexey Mayshev and contributors. All rights reserved.
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
	"github.com/maypok86/otter/v2/internal/generated/node"
)

// A key mutation changes what a key holds, deciding from what it holds now: Set, SetIfAbsent,
// the Compute family, Invalidate and the completion of a load or refresh are all one. mutate
// decides and applies the change under the key's bucket lock, and publish hands it to the
// policies and the OnDeletion listener once the lock is released. In between, the mutation
// carries everything decided under the lock, so that the work after it agrees with it: an
// expired entry is absent for every mutation, and both deletion listeners get the same cause.
//
// Removing a given node (eviction, expiration, InvalidateAll) is not a key mutation: it does
// not decide from the key's value, and its work after the lock belongs to maintenance.

// origin is where a key mutation comes from. It decides which pending loads of the key the
// mutation cancels, whether a write is a reload for the refresh calculator, and how the
// OnDeletion listener is called.
type origin uint8

const (
	// fromCaller is a call of the cache's API. A write or an invalidation cancels the key's
	// pending loads and refreshes, so that they cannot bring back the value it replaced.
	fromCaller origin = iota
	// fromLoad is the completion of a load. It cancels nothing: the load's own call is deleted
	// after the mutation is published.
	fromLoad
	// fromRefresh is the completion of a refresh, which runs as an executor task. It cancels
	// nothing either, and it calls the OnDeletion listener directly instead of submitting it to
	// the executor (see notifyDeletionFrom).
	fromRefresh
)

// mutationOp is what a key mutation does, as decided from the key's current entry.
type mutationOp uint8

const (
	// opKeep leaves the key as it is.
	opKeep mutationOp = iota
	// opWrite writes a value, over the current entry if there is one.
	opWrite
	// opInvalidate removes the current entry, if there is one.
	opInvalidate
	// opRemoveExpired removes the current entry, which has expired, as the timer wheel would:
	// a pending load of the key is kept, since it most likely started because the entry
	// expired, and only a refresh of the expired value is dropped.
	opRemoveExpired
)

// previous is the key's entry when a mutation is decided.
type previous[K comparable, V any] struct {
	// nowNano is the time the mutation is decided at. It is read under the bucket lock: read
	// before, it could be older than a removal that the key's readers have already seen, if the
	// entry expired while the mutation waited for the lock.
	nowNano int64
	// node is the key's node, or nil if the key is absent.
	node node.Node
	// value is node's value, read under the lock.
	value V
	// weight is node's weight, read under the lock: an update in place changes it.
	weight uint32
	// live reports that node holds a value that has not expired.
	live bool
	// nodeExpiresAt is node's deadline, read under the lock with live. A read's ExpireAfterRead
	// CAS can move the deadline while the mutation runs user code, to before nowNano too; a write
	// then gives its calculators this one (see writeEntry), so that an update of the entry found
	// live does not keep a deadline that had passed.
	nodeExpiresAt int64
}

// mutation is a key mutation as decided and applied under the bucket lock.
type mutation[K comparable, V any] struct {
	previous[K, V]
	// op is what the mutation did.
	op mutationOp
	// from is where the mutation came from.
	from origin
	// next is the key's node afterwards, or nil if it is absent. After an update in place it is
	// previous.node.
	next node.Node
	// writeResult is what the write or removal decided about the previous value and, for a
	// write, the new one.
	writeResult
}

// removalCause is the cause that a previous value leaves the cache with: cause if mutate found it
// live, CauseExpiration otherwise. It follows that decision rather than the node's deadline,
// which a read's ExpireAfterRead CAS can move while the mutation runs user code: decide (a
// Compute function) and, for a write, the weigher and the calculators. Read again, it could
// report a value that the caller was given as present as expired.
func removalCause(live bool, cause DeletionCause) DeletionCause {
	if live {
		return cause
	}
	return CauseExpiration
}

// mutate changes what key holds as decide says, under the key's bucket lock. decide is given
// the key's current entry and the time, and returns the op and, for opWrite, the value; it runs
// under the lock, may change the deadlines of the entry it was given, and returns
// opRemoveExpired only for an expired entry. The caller passes the result to publish once mutate
// returns.
func (c *cache[K, V]) mutate(
	key K,
	from origin,
	decide func(prev previous[K, V]) (mutationOp, V),
) mutation[K, V] {
	m := mutation[K, V]{from: from}
	m.next = c.hashmap.Compute(key, func(current node.Node) node.Node {
		nowNano := c.clock.NowNano()
		m.nowNano = nowNano
		m.node = current
		if current != nil {
			m.value = c.nodeManager.Value(current)
			m.weight = current.Weight()
			m.live = !current.HasExpired(nowNano)
			if c.withExpiration {
				m.nodeExpiresAt = current.ExpiresAt()
			}
		}

		var value V
		m.op, value = decide(m.previous)
		switch m.op {
		case opWrite:
			var n node.Node
			n, m.writeResult = c.atomicSet(key, value, m.previous, from)
			return n
		case opInvalidate:
			if from == fromCaller {
				c.singleflight.invalidate(key)
			}
			m.cause = c.retireRemoved(current, removalCause(m.live, CauseInvalidation))
			return nil
		case opRemoveExpired:
			c.singleflight.deleteOnRemoval(key)
			m.cause = c.retireRemoved(current, CauseExpiration)
			return nil
		default:
			return current
		}
	})
	return m
}

// publish hands a mutation to the policies, through the write or read buffer, counts the removal
// of the previous value in the stats if it is an eviction (countEviction), and reports the
// previous value to the OnDeletion listener with the cause decided under the lock.
func (c *cache[K, V]) publish(m mutation[K, V]) {
	onExecutor := m.from == fromRefresh
	switch m.op {
	case opWrite:
		// counted by afterWrite, once the hand-off to the policies cannot be lost
		c.afterWrite(m, onExecutor)
	case opInvalidate, opRemoveExpired:
		// Only an expired entry is counted, and a removal whose task is lost leaves it in the
		// timer wheel, which removes it on the next pass.
		c.countEviction(m.cause, m.weight)
		c.afterDelete(m.node, m.cause, false, onExecutor)
	default:
	}
}

// afterWrite publishes a write. It takes the previous value from the mutation, because after an
// in-place update the previous node is the written one.
func (c *cache[K, V]) afterWrite(m mutation[K, V], onExecutor bool) {
	// The previous value is reported when the write is applied, not when maintenance replays its
	// task, and before the write reaches the policies: a task handed to the write buffer may be
	// replayed at once on this goroutine (by an executor that runs tasks on the caller, or when the
	// buffer is full), and that maintenance may evict or expire the new value, whose event must not
	// come first. The write reaches the policies even if the recorder, the listener or the
	// executor calls runtime.Goexit: the table has changed already.
	if c.withMaintenance {
		defer c.afterWritePolicies(m)
	}
	c.countEviction(m.cause, m.weight)
	if m.node != nil {
		c.notifyDeletionFrom(onExecutor, c.nodeManager.Key(m.node), m.value, m.cause)
	}
}

// afterWritePolicies hands a write to the policies.
func (c *cache[K, V]) afterWritePolicies(m mutation[K, V]) {
	n, old := m.next, m.node
	// The read that started a refresh was the request's access, so its completion is not another
	// one.
	quiet := m.from == fromRefresh
	switch {
	case m.inPlace:
		// Updated in place: the policies observe an access through the lossy read buffer, and a
		// changed weight through a task, which must not be lost. The expiration time can only
		// move later (see atomicSet), which needs no task: the timer wheel reschedules a node that
		// it finds has not expired yet. The task is handed off first: afterRead may submit a
		// drain, and an executor that calls runtime.Goexit would lose it.
		if m.reconcile {
			c.afterWriteTask(newTask(n, nil, reconcileReason))
		}
		if !quiet {
			c.afterRead(n, false)
		}
	case old == nil:
		c.afterWriteTask(newTask(n, nil, addReason))
	default:
		t := newTask(n, old, updateReason)
		t.quiet = quiet
		c.afterWriteTask(t)
	}
}

// atomicSet applies a write of value to the key whose current node is old (nil if absent).
// The user's weigher, expiry and refresh calculators are called once per write.
//
// A write over a live node that can set its value is applied in place, keeping the node: plain
// updates, loads and refreshes. Swapping in a new node forces an update task through the bounded
// write buffer so that the policies can relink the node. Under an update-heavy workload that
// buffer overflows and every writer falls back to performCleanUp, serializing all writes on
// evictionMutex. When the node identity is kept, the policies only need to observe an access,
// which goes through the lossy read buffer, unless the weight changed: the eviction policy
// accounts the weight separately (policyWeight), so that change goes through a reconciliation
// task.
//
// A write that moves the expiration time earlier, or over an entry that has expired and not been
// removed yet, swaps in a new node. Until an in-place write stores the new deadline, the node
// holds the new value under the old one, and a lock-free reader, which loads the deadline before
// the value, may pair them. That is harmless only while the old deadline is live and not earlier
// than the new one: over an expired entry, a reader that finds the old deadline passed reports
// the key absent after one with an earlier clock returned the new value, and with an earlier
// deadline, a reader that starts after it still returns the value.
//
// A value that fits in one atomic word (a pointer-shaped value, or one without pointers of up to
// 8 bytes) can always be set in place (CanSetValue); the cases above still swap in a new node. A
// larger value is kept inline, and an inline value is never written after publication, so that a
// concurrent reader cannot observe a torn or cleared value: the first update of such a live entry
// replaces the node with a boxed one, whose value lives behind an atomic pointer, so that later
// updates of this (likely hot) key can be applied in place.
//
// prev is what mutate decided about the old entry: the kind of the write, and so the calculator
// it calls, the kind of a new node and the cause follow it, so that they agree with what the
// caller was given (see removalCause). A deadline that a read has moved since is the read's,
// unless it is before nowNano (see writeEntry). The choice to write in place reads old's deadline
// again: in place, a reader could pair the new value with a deadline a read moved before nowNano.
func (c *cache[K, V]) atomicSet(
	key K,
	value V,
	prev previous[K, V],
	from origin,
) (node.Node, writeResult) {
	old, live, nowNano, oldValue := prev.node, prev.live, prev.nowNano, prev.value
	// The user's weigher and calculators run before anything is changed: if one of them panics,
	// the write is not applied and the panic reaches the caller.
	weight := c.weigher(key, value)
	kind := newWriteKind(live, from == fromRefresh)
	base := &prev
	if kind == createWrite {
		base = nil
	}
	entry := c.writeEntry(key, value, weight, base, nowNano)
	if c.withExpiration {
		entry.ExpiresAtNano = expiresAtAfterWrite(c.expiryCalculator, kind, entry, oldValue, nowNano)
	}
	if c.withRefresh {
		// after the expiration time, so that the refresh calculator sees the new one
		entry.RefreshableAtNano = refreshableAtAfterWrite(c.refreshCalculator, kind, entry, oldValue, nil, nowNano)
	}
	// The calls are cancelled before the new value is published: a refresh requested by a read
	// that saw it must not join a call that this write cancels, which would discard the refresh.
	// When the node is updated in place, superseded cannot tell such a call from a live one.
	c.cancelCalls(key, from)

	// in place only over a live entry whose deadline does not move earlier, see above
	if live && c.withMaintenance && !old.HasExpired(nowNano) && old.CanSetValue() &&
		(!c.withExpiration || entry.ExpiresAtNano >= old.ExpiresAt()) {
		written := writeResult{
			// the old entry is live: an expired one is replaced
			cause:     CauseReplacement,
			inPlace:   true,
			reconcile: weight != old.Weight(),
			expiresAt: entry.ExpiresAtNano,
		}
		// The synchronous listener runs before the new value is published, as it does when the
		// node is replaced: readers see the previous value until the write completes.
		c.notifyAtomicDeletion(key, oldValue, written.cause)
		// The value is stored before the deadline: a read that moves the deadline at the same
		// time checks the value before its CAS (see calcExpiresAtAfterRead).
		c.nodeManager.SetValue(old, value)
		if c.isWeighted {
			// only weighted nodes store a weight; the others always weigh 1
			old.SetWeight(weight)
		}
		if c.withExpiration {
			old.SetExpiresAt(entry.ExpiresAtNano)
		}
		if c.withRefresh {
			old.SetRefreshableAt(entry.RefreshableAtNano)
		}
		return old, written
	}

	n := c.newNode(key, value, weight, entry.ExpiresAtNano, entry.RefreshableAtNano, live && c.withMaintenance)
	written := writeResult{expiresAt: entry.ExpiresAtNano}
	if old != nil {
		written.cause = removalCause(live, CauseReplacement)
		c.retire(old, oldValue, written.cause)
	}
	return n, written
}

// cancelCalls cancels the pending load or refresh of key when a write from the cache's API
// changes it. It runs under the key's bucket lock before the new value is published, and a
// refresh is registered under that lock too: a refresh requested while the write is in progress
// either is cancelled or sees the new value.
func (c *cache[K, V]) cancelCalls(key K, from origin) {
	if from == fromCaller {
		c.singleflight.delete(key)
	}
}

// writeResult is what a write or a removal decided under the bucket lock about the previous
// value and, for a write, the new one, so that the work done after the lock is released agrees
// with it.
type writeResult struct {
	// cause is why the previous value left the cache: CauseReplacement or CauseInvalidation, or
	// CauseExpiration when it had expired. It is reported to both deletion listeners; recomputing it later could
	// disagree, because the deadline of a replaced node can still be changed by a reader that
	// holds it. Unset when there was no previous node.
	cause DeletionCause
	// inPlace reports that the existing node was updated instead of being replaced.
	inPlace bool
	// reconcile reports a changed weight of an in-place update, which the policies cannot pick up
	// from an access.
	reconcile bool
	// expiresAt is the expiration time a write gave the new value. Once the lock is released,
	// another write can update the node again, so it is not read back from the node.
	expiresAt int64
}

// retireRemoved retires old, which the caller removes from the hash table, without touching the
// key's pending calls, and returns cause, which it reports to the synchronous listener.
func (c *cache[K, V]) retireRemoved(old node.Node, cause DeletionCause) DeletionCause {
	if old == nil {
		return 0
	}
	c.retire(old, c.nodeManager.Value(old), cause)
	return cause
}
