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
	"context"
	"iter"
	"time"

	"github.com/maypok86/otter/v2/stats"
)

// ComputeOp tells the Compute methods what to do.
type ComputeOp int

const (
	// CancelOp signals to Compute to not do anything as a result
	// of executing the lambda. If the entry was not present in
	// the map, nothing happens, and if it was present, the
	// returned value is ignored.
	CancelOp ComputeOp = iota
	// WriteOp signals to Compute to update the entry to the
	// value returned by the lambda, creating it if necessary.
	WriteOp
	// InvalidateOp signals to Compute to always discard the entry
	// from the cache.
	InvalidateOp
)

var computeOpStrings = []string{
	"CancelOp",
	"WriteOp",
	"InvalidateOp",
}

// String implements [fmt.Stringer] interface.
func (co ComputeOp) String() string {
	if co >= 0 && int(co) < len(computeOpStrings) {
		return computeOpStrings[co]
	}
	return "<unknown otter.ComputeOp>"
}

// Cache is an in-memory cache implementation that supports full concurrency of retrievals and multiple ways to bound the cache.
type Cache[K comparable, V any] struct {
	cache *cache[K, V]
}

// Must creates a configured [Cache] instance or
// panics if invalid parameters were specified.
//
// This method does not alter the state of the [Options] instance, so it can be invoked
// again to create multiple independent caches.
func Must[K comparable, V any](o *Options[K, V]) *Cache[K, V] {
	c, err := New(o)
	if err != nil {
		panic(err)
	}
	return c
}

// New creates a configured [Cache] instance or
// returns an error if invalid parameters were specified.
//
// This method does not alter the state of the [Options] instance, so it can be invoked
// again to create multiple independent caches.
func New[K comparable, V any](o *Options[K, V]) (*Cache[K, V], error) {
	if o == nil {
		o = &Options[K, V]{}
	}

	if err := o.validate(); err != nil {
		return nil, err
	}

	cacheImpl := newCache(o)
	c := &Cache[K, V]{
		cache: cacheImpl,
	}
	return c, nil
}

// GetIfPresent returns the value associated with the key in this cache.
func (c *Cache[K, V]) GetIfPresent(key K) (V, bool) {
	return c.cache.GetIfPresent(key)
}

// GetEntry returns the cache entry associated with the key in this cache.
func (c *Cache[K, V]) GetEntry(key K) (Entry[K, V], bool) {
	return c.cache.GetEntry(key)
}

// GetEntryQuietly returns the cache entry associated with the key in this cache.
//
// Unlike GetEntry, this function does not produce any side effects
// such as updating statistics or the eviction policy.
func (c *Cache[K, V]) GetEntryQuietly(key K) (Entry[K, V], bool) {
	return c.cache.GetEntryQuietly(key)
}

// Set associates the value with the key in this cache.
//
// If the specified key is not already associated with a value, then it returns new value and true.
// An expired entry that has not been removed yet counts as no value.
//
// If the specified key is already associated with a value, then it returns existing value and false.
func (c *Cache[K, V]) Set(key K, value V) (V, bool) {
	return c.cache.Set(key, value)
}

// SetIfAbsent if the specified key is not already associated with a value associates it with the given value.
//
// If the specified key is not already associated with a value, then it returns new value and true.
//
// If the specified key is already associated with a value, then it returns existing value and false.
func (c *Cache[K, V]) SetIfAbsent(key K, value V) (V, bool) {
	return c.cache.SetIfAbsent(key, value)
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
// if a corresponding entry is present, or the zero value
// otherwise. You can think of these results as equivalent to regular key-value lookups in a map.
//
// This call locks a hash table bucket while the compute function
// is executed. It means that modifications on other entries in
// the bucket will be blocked until the remappingFunc executes, and so will
// refreshes of them, a growth of the hash table and an eviction or expiration of an entry in the
// bucket, during which writes of every key wait.
// Consider this when the function includes long-running operations.
//
// WARNING: The function may only read entries with [Cache.GetIfPresent],
// [Cache.GetEntry] or [Cache.GetEntryQuietly]. Any other call of this cache,
// even a write of another key, may deadlock it: its locks are not reentrant.
func (c *Cache[K, V]) Compute(
	key K,
	remappingFunc func(oldValue V, found bool) (newValue V, op ComputeOp),
) (actualValue V, ok bool) {
	return c.cache.Compute(key, remappingFunc)
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
// the bucket will be blocked until the valueFn executes, and so will
// refreshes of them, a growth of the hash table and an eviction or expiration of an entry in the
// bucket, during which writes of every key wait.
// Consider this when the function includes long-running operations.
//
// WARNING: The function may only read entries with [Cache.GetIfPresent],
// [Cache.GetEntry] or [Cache.GetEntryQuietly]. Any other call of this cache,
// even a write of another key, may deadlock it: its locks are not reentrant.
func (c *Cache[K, V]) ComputeIfAbsent(
	key K,
	mappingFunc func() (newValue V, cancel bool),
) (actualValue V, ok bool) {
	return c.cache.ComputeIfAbsent(key, mappingFunc)
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
// the bucket will be blocked until the valueFn executes, and so will
// refreshes of them, a growth of the hash table and an eviction or expiration of an entry in the
// bucket, during which writes of every key wait.
// Consider this when the function includes long-running operations.
//
// WARNING: The function may only read entries with [Cache.GetIfPresent],
// [Cache.GetEntry] or [Cache.GetEntryQuietly]. Any other call of this cache,
// even a write of another key, may deadlock it: its locks are not reentrant.
func (c *Cache[K, V]) ComputeIfPresent(
	key K,
	remappingFunc func(oldValue V) (newValue V, op ComputeOp),
) (actualValue V, ok bool) {
	return c.cache.ComputeIfPresent(key, remappingFunc)
}

// SetExpiresAfter specifies that the entry should be automatically removed from the cache once the duration has
// elapsed. The expiration policy determines when the entry's age is reset. It has no effect on an absent
// or expired entry.
//
// This call locks a hash table bucket, as a write does. Calling it from a function that runs while the
// cache holds a bucket lock, such as a Compute function, OnAtomicDeletion, a weigher or a calculator,
// may deadlock.
func (c *Cache[K, V]) SetExpiresAfter(key K, expiresAfter time.Duration) {
	c.cache.SetExpiresAfter(key, expiresAfter)
}

// SetRefreshableAfter specifies that each entry should be eligible for reloading once a fixed duration has elapsed.
// The refresh policy determines when the entry's age is reset.
//
// This call locks a hash table bucket, as a write does. Calling it from a function that runs while the
// cache holds a bucket lock, such as a Compute function, OnAtomicDeletion, a weigher or a calculator,
// may deadlock.
func (c *Cache[K, V]) SetRefreshableAfter(key K, refreshableAfter time.Duration) {
	c.cache.SetRefreshableAfter(key, refreshableAfter)
}

// Get returns the value associated with key in this cache, obtaining that value from loader if necessary.
// The method improves upon the conventional "if cached, return; otherwise create, cache and return" pattern.
//
// Get can return an [ErrNotFound] error if the [Loader] returns it, or if Get waits for a [Cache.BulkGet]
// whose [BulkLoader] leaves the key out of its result.
// This means that the entry was not found in the data source.
// This allows the cache to recognize when a record is missing from the data source
// and subsequently delete the cached entry.
// It also enables proper metric collection, as the cache doesn't classify [ErrNotFound] as a load error.
//
// If another call to Get is currently loading the value for key,
// simply waits for that goroutine to finish and returns its loaded value. Note that
// multiple goroutines can concurrently load values for distinct keys.
//
// A write or an invalidation of key during the load discards the loaded value: Get then returns
// the value key has or loads key again, until ctx is done, so the loader may be called more than
// once. A Get whose ctx is already done returns a cached value but does not load a missing one.
//
// No observable state associated with this cache is modified until loading completes.
//
// WARNING: When performing a refresh (see [RefreshCalculator]),
// the [Loader] will receive a context wrapped in [context.WithoutCancel].
// If you need to control refresh cancellation, you can use closures or values stored in the context.
// A Get that starts a refresh may wait for a lock of the cache, see [Options.RefreshCalculator].
//
// WARNING: [Loader] must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every loader used with it should compute the same value.
// Otherwise, a call that passes one loader may return the result of another call
// with a differently behaving loader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *Cache[K, V]) Get(ctx context.Context, key K, loader Loader[K, V]) (V, error) {
	return c.cache.Get(ctx, key, loader)
}

// BulkGet returns the value associated with key in this cache, obtaining that value from loader if necessary.
// The method improves upon the conventional "if cached, return; otherwise create, cache and return" pattern.
//
// If another call to Get (BulkGet) is currently loading the value for key,
// simply waits for that goroutine to finish and returns its loaded value. Note that
// multiple goroutines can concurrently load values for distinct keys.
//
// A write or an invalidation of a key during the load discards its loaded value: BulkGet then
// takes the value the key has or loads the key again, until ctx is done, so the bulk loader may be
// called more than once. A BulkGet whose ctx is already done returns the cached values but does not
// load the missing ones.
//
// No observable state associated with this cache is modified until loading completes.
//
// NOTE: duplicate elements in keys will be ignored.
//
// WARNING: When performing a refresh (see [RefreshCalculator]),
// the [BulkLoader] will receive a context wrapped in [context.WithoutCancel].
// If you need to control refresh cancellation, you can use closures or values stored in the context.
// A BulkGet that starts a refresh may wait for a lock of the cache, see [Options.RefreshCalculator].
//
// WARNING: [BulkLoader] must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every bulkLoader used with it should compute the same value.
// Otherwise, a call that passes one bulkLoader may return the result of another call
// with a differently behaving bulkLoader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *Cache[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader BulkLoader[K, V]) (map[K]V, error) {
	return c.cache.BulkGet(ctx, keys, bulkLoader)
}

// Refresh loads a new value for the key, asynchronously. While the new value is loading the
// previous value (if any) will continue to be returned by any Get unless it is evicted.
// If the new value is loaded successfully, it will replace the previous value in the cache;
// If refreshing returned an error, the previous value will remain, and the error will be logged
// using [Logger] and swallowed, except for [ErrNotFound], which removes the entry. If a load or refresh of the
// key that started from its current state is in flight, this method joins it instead of loading again.
// A joined load by Get or BulkGet whose loader returns an error other than [ErrNotFound] after its
// caller's context has ended is loaded again, unless the key was written or invalidated since the
// refresh was requested.
// If the key is written, invalidated or expires after the refresh was requested, the loaded
// value is discarded and the cache keeps the newer state.
//
// [Cache] will call Loader.Reload if the cache currently contains a value for the key,
// and Loader.Load otherwise.
// Loading is asynchronous by delegating to the configured Executor.
//
// Refresh returns a channel that will receive the result when it is ready. The returned channel will not be closed.
//
// WARNING: When performing a refresh (see [RefreshCalculator]),
// the [Loader] will receive a context wrapped in [context.WithoutCancel].
// If you need to control refresh cancellation, you can use closures or values stored in the context.
//
// WARNING: If the cache was constructed without [RefreshCalculator], then Refresh will return the nil channel.
//
// WARNING: Loader.Load and Loader.Reload must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every loader used with it should compute the same value.
// Otherwise, a call that passes one loader may return the result of another call
// with a differently behaving loader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *Cache[K, V]) Refresh(ctx context.Context, key K, loader Loader[K, V]) <-chan RefreshResult[K, V] {
	return c.cache.Refresh(ctx, key, loader)
}

// BulkRefresh loads a new value for each key, asynchronously. While the new value is loading the
// previous value (if any) will continue to be returned by any Get unless it is evicted.
// If the new value is loaded successfully, it will replace the previous value in the cache;
// If refreshing returned an error, the previous values will remain, and the error will be logged
// using [Logger] and swallowed, an [ErrNotFound] for the whole bulk included. A key that the
// [BulkLoader] leaves out of its result is not found, and its entry is removed. If a load or refresh of the
// key that started from its current state is in flight, this method joins it instead of loading again.
// A joined load by Get or BulkGet whose loader returns an error other than [ErrNotFound] after its
// caller's context has ended is loaded again, unless the key was written or invalidated since the
// refresh was requested.
// If the key is written, invalidated or expires after the refresh was requested, the loaded
// value is discarded and the cache keeps the newer state.
//
// [Cache] will call BulkLoader.BulkReload for existing keys, and BulkLoader.BulkLoad otherwise.
// Loading is asynchronous by delegating to the configured Executor.
//
// BulkRefresh returns a channel that will receive the results when they are ready. The returned channel will not be closed.
//
// NOTE: duplicate elements in keys will be ignored.
//
// WARNING: When performing a refresh (see [RefreshCalculator]),
// the [BulkLoader] will receive a context wrapped in [context.WithoutCancel].
// If you need to control refresh cancellation, you can use closures or values stored in the context.
//
// WARNING: If the cache was constructed without [RefreshCalculator], then BulkRefresh will return the nil channel.
//
// WARNING: BulkLoader.BulkLoad and BulkLoader.BulkReload must not attempt to update any mappings of this cache directly.
//
// WARNING: For any given key, every bulkLoader used with it should compute the same value.
// Otherwise, a call that passes one bulkLoader may return the result of another call
// with a differently behaving loader. For example, a call that requests a short timeout
// for an RPC may wait for a similar call that requests a long timeout, or a call by an
// unprivileged user may return a resource accessible only to a privileged user making a similar call.
func (c *Cache[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader BulkLoader[K, V]) <-chan []RefreshResult[K, V] {
	return c.cache.BulkRefresh(ctx, keys, bulkLoader)
}

// Invalidate discards any cached value for the key.
//
// Returns previous value if any. The invalidated result reports whether the key was
// present; an expired entry that has not been removed yet is not.
func (c *Cache[K, V]) Invalidate(key K) (value V, invalidated bool) {
	return c.cache.Invalidate(key)
}

// All returns an iterator over all key-value pairs in the cache.
// The iteration order is not specified and is not guaranteed to be the same from one call to the next.
//
// Iterator is at least weakly consistent: he is safe for concurrent use,
// but if the cache is modified (including by eviction) after the iterator is
// created, it is undefined which of the changes (if any) will be reflected in that iterator.
func (c *Cache[K, V]) All() iter.Seq2[K, V] {
	return c.cache.All()
}

// Keys returns an iterator over all keys in the cache.
// The iteration order is not specified and is not guaranteed to be the same from one call to the next.
//
// Iterator is at least weakly consistent: he is safe for concurrent use,
// but if the cache is modified (including by eviction) after the iterator is
// created, it is undefined which of the changes (if any) will be reflected in that iterator.
func (c *Cache[K, V]) Keys() iter.Seq[K] {
	return c.cache.Keys()
}

// Values returns an iterator over all values in the cache.
// The iteration order is not specified and is not guaranteed to be the same from one call to the next.
//
// Iterator is at least weakly consistent: he is safe for concurrent use,
// but if the cache is modified (including by eviction) after the iterator is
// created, it is undefined which of the changes (if any) will be reflected in that iterator.
func (c *Cache[K, V]) Values() iter.Seq[V] {
	return c.cache.Values()
}

// InvalidateAll discards all entries in the cache. The behavior of this operation is undefined for an entry
// that is being loaded (or reloaded) and is otherwise not present.
func (c *Cache[K, V]) InvalidateAll() {
	c.cache.InvalidateAll()
}

// CleanUp performs any pending maintenance operations needed by the cache. Exactly which activities are
// performed -- if any -- is implementation-dependent.
func (c *Cache[K, V]) CleanUp() {
	c.cache.CleanUp()
}

// SetMaximum specifies the maximum total size of this cache. This value may be interpreted as the weighted
// or unweighted threshold size based on how this cache was constructed. If the cache currently
// exceeds the new maximum size this operation eagerly evict entries until the cache shrinks to
// the appropriate size.
func (c *Cache[K, V]) SetMaximum(maximum uint64) {
	c.cache.SetMaximum(maximum)
}

// GetMaximum returns the maximum total weighted or unweighted size of this cache, depending on how the
// cache was constructed. If this cache does not use a (weighted) size bound, then the method will return math.MaxUint64.
func (c *Cache[K, V]) GetMaximum() uint64 {
	return c.cache.GetMaximum()
}

// EstimatedSize returns the approximate number of entries in this cache. The value returned is an estimate; the
// actual count may differ if there are concurrent insertions or deletions, or if some entries are
// pending deletion due to expiration. In the case of stale entries
// this inaccuracy can be mitigated by performing a CleanUp first.
func (c *Cache[K, V]) EstimatedSize() int {
	return c.cache.EstimatedSize()
}

// IsWeighted returns whether the cache is bounded by a maximum weight.
func (c *Cache[K, V]) IsWeighted() bool {
	return c.cache.IsWeighted()
}

// WeightedSize returns the approximate accumulated weight of entries in this cache. If this cache does not
// use a weighted size bound, then the method will return 0.
func (c *Cache[K, V]) WeightedSize() uint64 {
	return c.cache.WeightedSize()
}

// IsRecordingStats returns whether the cache statistics are being accumulated.
func (c *Cache[K, V]) IsRecordingStats() bool {
	return c.cache.IsRecordingStats()
}

// Stats returns a current snapshot of this cache's cumulative statistics.
// All statistics are initialized to zero and are monotonically increasing over the lifetime of the cache.
// Due to the performance penalty of maintaining statistics,
// some implementations may not record the usage history immediately or at all.
//
// NOTE: If your [stats.Recorder] implementation doesn't also implement [stats.Snapshoter],
// this method will always return a zero-value snapshot.
func (c *Cache[K, V]) Stats() stats.Stats {
	return c.cache.Stats()
}

// Hottest returns an iterator for ordered traversal of the cache entries. The order of
// iteration is from the entries most likely to be retained (hottest) to the entries least
// likely to be retained (coldest). This order is determined by the eviction policy's best guess
// at the start of the iteration.
//
// WARNING: Beware that this iteration is performed within the eviction policy's exclusive lock, so the
// iteration should be short and simple. While the iteration is in progress further eviction
// maintenance will be halted. The loop body may read entries without loading them, but must not
// write to the cache or call [Cache.CleanUp], [Cache.InvalidateAll], [Cache.SetMaximum],
// [Cache.GetMaximum], [Cache.WeightedSize], [Cache.Hottest], [Cache.Coldest] or [SaveCacheTo],
// which take this lock. To change the cache based on this order, collect the keys first.
//
// If the cache has no size bound, the entries come in no particular order, as from [Cache.All], and
// the eviction policy's lock is not held.
func (c *Cache[K, V]) Hottest() iter.Seq[Entry[K, V]] {
	return c.cache.Hottest()
}

// Coldest returns an iterator for ordered traversal of the cache entries. The order of
// iteration is from the entries least likely to be retained (coldest) to the entries most
// likely to be retained (hottest). This order is determined by the eviction policy's best guess
// at the start of the iteration.
//
// WARNING: Beware that this iteration is performed within the eviction policy's exclusive lock, so the
// iteration should be short and simple. While the iteration is in progress further eviction
// maintenance will be halted. The loop body may read entries without loading them, but must not
// write to the cache or call [Cache.CleanUp], [Cache.InvalidateAll], [Cache.SetMaximum],
// [Cache.GetMaximum], [Cache.WeightedSize], [Cache.Hottest], [Cache.Coldest] or [SaveCacheTo],
// which take this lock. To change the cache based on this order, collect the keys first.
//
// If the cache has no size bound, the entries come in no particular order, as from [Cache.All], and
// the eviction policy's lock is not held.
func (c *Cache[K, V]) Coldest() iter.Seq[Entry[K, V]] {
	return c.cache.Coldest()
}

// StopAllGoroutines stops the goroutine that periodically cleans up a cache with expiration.
// It returns false if an earlier call has stopped it or the cache has no expiration, and true
// otherwise, also when the goroutine has already returned by itself.
// It neither stops nor waits for the tasks given to the Executor (loads, refreshes, OnDeletion
// calls), and later operations still submit them.
//
// NOTE: In the vast majority of cases, you do not need to call this method: once the cache is no
// longer reachable, the goroutine stops at the next tick of the Clock. However, it can sometimes be
// useful, for example with a Clock that does not tick. With testing/synctest, call it before the
// bubble's function returns: time in the bubble stops then, and the goroutine would never see its
// next tick.
// You can think of this method as analogous to the Stop method of [time.Timer].
// It does not stop the goroutine of a [CoarseClock], which belongs to the clock.
//
// WARNING: StopAllGoroutines must not be called from a function that the cache runs, such as a
// deletion listener: it waits for the periodic clean-up goroutine, which may be the one running it.
//
// NOTE: This method only stops the goroutines and does not invalidate entries in the cache.
// To invalidate entries, you can use the Invalidate / InvalidateAll methods.
func (c *Cache[K, V]) StopAllGoroutines() bool {
	return c.cache.StopAllGoroutines()
}

func (c *Cache[K, V]) has(key K) bool {
	return c.cache.has(key)
}
