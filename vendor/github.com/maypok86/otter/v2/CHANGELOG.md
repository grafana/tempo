## 2.4.0 - 2026-10-20

### 📝 Description

This release is mostly about correctness. The fixes come from audits of races between loads, refreshes, writes and expiration, of how the cache handles panics and `runtime.Goexit` in user code, and of its lifecycle. A new test checks random concurrent histories of the cache for linearizability with [porcupine](https://github.com/anishathalye/porcupine). Loading and refresh are now linearizable: a load or refresh that started before a write no longer replaces the value the write stored ([comparison with other caches](https://maypok86.github.io/otter/performance/linearizability/)). Updates of live entries now happen in place, which makes them 2–19× faster depending on the value type and contention. Binaries are smaller too: a program with one cache is 3.15 MB instead of 4.41 MB, and each additional cache type adds about 160 KB instead of 350 KB.

### ✨Features

- Added `CoarseClock`, a `Clock` that reads the time once per resolution and can be shared by several caches. With expiration it makes `GetIfPresent` 26–57% faster ([#281](https://github.com/maypok86/otter/pull/281))
- A `Get` or `BulkGet` that waits for another goroutine's load now follows its own context. It returns `ctx.Err()` when its context is done; the load keeps running and caches its value. If the load failed because the owner's context was done, a waiter with a live context loads the key again ([#237](https://github.com/maypok86/otter/pull/237))

### 🚀 Improvements

- Updates of live entries happen in place, without a new node or a write-buffer task, and values of at most one word are updated without allocating. With 12 goroutines an int update is 95% faster (399.8 → 21.1 ns) and a string update 93% faster ([#194](https://github.com/maypok86/otter/pull/194), [#195](https://github.com/maypok86/otter/pull/195), [#196](https://github.com/maypok86/otter/pull/196), [#197](https://github.com/maypok86/otter/pull/197), [#198](https://github.com/maypok86/otter/pull/198))
- Reads with `ExpiryWriting` and `ExpiryCreating` no longer call `ExpireAfterRead` (up to 29% faster), and reads with access-based expiration don't write a deadline that moves by less than a second (37–48% faster from 4+ goroutines) ([#225](https://github.com/maypok86/otter/pull/225), [#226](https://github.com/maypok86/otter/pull/226))
- The read buffer picks its stripe without `sync.Pool`, loses fewer reads and no longer shares cache lines with other objects. A mix of 75% reads on 8 threads is 55% faster ([#219](https://github.com/maypok86/otter/pull/219), [#236](https://github.com/maypok86/otter/pull/236))
- The hash table hashes integer keys without `maphash` and unlinks empty overflow buckets, which saves ~34 MB per 1M entries and speeds up misses. Resizes and allocating a large `InitialCapacity` are faster too ([#211](https://github.com/maypok86/otter/pull/211), [#212](https://github.com/maypok86/otter/pull/212), [#213](https://github.com/maypok86/otter/pull/213), [#214](https://github.com/maypok86/otter/pull/214), [#224](https://github.com/maypok86/otter/pull/224))
- Writes and evictions are faster: singleflight calls are looked up without locking, scheduling a drain doesn't allocate, and policy nodes are linked through raw pointers ([#221](https://github.com/maypok86/otter/pull/221), [#222](https://github.com/maypok86/otter/pull/222), [#246](https://github.com/maypok86/otter/pull/246), [#283](https://github.com/maypok86/otter/pull/283))
- Write-buffer tasks are allocated instead of pooled. Inserts are up to 16% faster, but each insert, replacement or deletion now allocates 48 B ([#223](https://github.com/maypok86/otter/pull/223))
- Node code is compiled once per program, so each additional cache type adds 154 KB to the binary instead of 476 KB ([#293](https://github.com/maypok86/otter/pull/293))
- New benchmarks (eviction, expiration, loading, scaling), regenerated performance pages and a [comparison of the linearizability](https://maypok86.github.io/otter/performance/linearizability/) of the benchmarked caches ([#229](https://github.com/maypok86/otter/pull/229), [#238](https://github.com/maypok86/otter/pull/238), [#284](https://github.com/maypok86/otter/pull/284))
- The documentation now says which cache methods a callback may call, what an `Executor` must not do and what `StopAllGoroutines` stops. Places where it disagreed with the code are fixed ([#258](https://github.com/maypok86/otter/pull/258), [#262](https://github.com/maypok86/otter/pull/262), [#266](https://github.com/maypok86/otter/pull/266), [#294](https://github.com/maypok86/otter/pull/294), [#295](https://github.com/maypok86/otter/pull/295))

### ⚠️ Behavior changes

- An `Executor` must not block the submitter. A bounded pool should run the task on the caller when it is full ([#263](https://github.com/maypok86/otter/issues/263))
- Panics in `OnDeletion`, `OnAtomicDeletion`, the stats recorder and the logger are logged and dropped. Panics in `Compute`, the weigher and the calculators still reach the caller, but the entry is left unchanged ([#191](https://github.com/maypok86/otter/pull/191))
- `SetExpiresAfter` and `SetRefreshableAfter` take the key's bucket lock, so calling them from `Compute*`, `OnAtomicDeletion`, a weigher or a calculator deadlocks ([#247](https://github.com/maypok86/otter/pull/247))
- An expired entry that hasn't been removed yet is absent for every operation: `Set` returns `(value, true)`, `Invalidate` reports nothing invalidated, and a write over it is a creation (`ExpireAfterCreate`, `RefreshAfterCreate`) ([#208](https://github.com/maypok86/otter/pull/208), [#230](https://github.com/maypok86/otter/pull/230))
- On a miss, `Get` and `BulkGet` with a context that is already done return `ctx.Err()` without calling the loader. The loader may be called more than once for one `Get` when the key is invalidated while it loads ([#288](https://github.com/maypok86/otter/pull/288), [#295](https://github.com/maypok86/otter/pull/295))
- `BulkGet` no longer returns keys that the bulk loader leaves out, and extra keys returned by the bulk loader only fill gaps instead of overwriting values ([#202](https://github.com/maypok86/otter/pull/202), [#206](https://github.com/maypok86/otter/pull/206))
- An expired entry removed by a write or by size eviction is reported as `CauseExpiration` and counted in `Evictions`; evictions are recorded before `OnDeletion` runs ([#235](https://github.com/maypok86/otter/pull/235), [#253](https://github.com/maypok86/otter/pull/253))
- With access-based expiration, a deadline set by a read is accurate to within a second for lifetimes over a second ([#225](https://github.com/maypok86/otter/pull/225))
- `OnAtomicDeletion` runs before the node is retired and may be called again for the same removal if it exits with `runtime.Goexit` ([#286](https://github.com/maypok86/otter/pull/286))
- Iteration (`All`, `Keys`, `Values`, `Hottest`, `Coldest`) is weakly consistent rather than a snapshot ([#245](https://github.com/maypok86/otter/pull/245))
- The cache no longer uses `runtime.AddCleanup`: the periodic clean-up goroutine holds a weak pointer and exits after the cache is collected ([#279](https://github.com/maypok86/otter/pull/279))
- `New` panics when `InitialCapacity` is too large to allocate ([#252](https://github.com/maypok86/otter/pull/252))

### 🐞 Bug Fixes

Loading and refresh:

- `Get` and `BulkGet` could call the loader more than once for one key ([#152](https://github.com/maypok86/otter/issues/152), [#184](https://github.com/maypok86/otter/pull/184), [#241](https://github.com/maypok86/otter/pull/241))
- A load or a refresh could overwrite a value that `Set` or `Compute` wrote while it ran; with the default executor 970 of 1,000 `Set`s after a stale `Get` were lost ([#203](https://github.com/maypok86/otter/pull/203), [#240](https://github.com/maypok86/otter/pull/240), [#243](https://github.com/maypok86/otter/pull/243))
- A key could be refreshed twice, a read racing `Invalidate` could bring the key back, and a `Get` that joined a discarded or failed refresh got that refresh's result instead of the key's value ([#243](https://github.com/maypok86/otter/pull/243), [#288](https://github.com/maypok86/otter/pull/288))
- Stale reads could fill a bounded blocking executor with tasks that only waited for a refresh, and deadlock the cache ([#243](https://github.com/maypok86/otter/pull/243))
- A refresh was dropped when an old node of its key was evicted or an expired entry was removed ([#200](https://github.com/maypok86/otter/pull/200), [#243](https://github.com/maypok86/otter/pull/243))
- Eviction cancelled in-flight loads ([#188](https://github.com/maypok86/otter/issues/188), [#191](https://github.com/maypok86/otter/pull/191))
- `Compute` returning `CancelOp` over an expired entry cancelled the key's in-flight load ([#210](https://github.com/maypok86/otter/pull/210))
- `BulkGet` reported a key missing or lost the error of a load it joined when its own bulk loader also returned that key ([#288](https://github.com/maypok86/otter/pull/288))
- A manual `Refresh` that joined a `Get` whose context was done reported that cancellation instead of loading the key ([#292](https://github.com/maypok86/otter/pull/292))
- A nil loader or context left the key's load registered forever, so later `Get`s hung ([#257](https://github.com/maypok86/otter/pull/257))

Expiration:

- Durations of `math.MaxInt64` overflowed into a past deadline, so the entry expired at once and every `Get` reloaded it ([#204](https://github.com/maypok86/otter/pull/204))
- An entry that expired while a write waited for the bucket lock was treated as live, and `SetExpiresAfter` could revive an expired entry ([#208](https://github.com/maypok86/otter/pull/208), [#249](https://github.com/maypok86/otter/pull/249))
- `SetExpiresAfter` and `SetRefreshableAfter` were undone by a concurrent write or a failed reload ([#247](https://github.com/maypok86/otter/pull/247))
- A deadline moved earlier by `SetExpiresAfter` could be ignored by the timer wheel until the old deadline ([#209](https://github.com/maypok86/otter/pull/209))
- A deadline the timer wheel had already passed was put into a bucket ~6.5 days ahead ([#196](https://github.com/maypok86/otter/pull/196))
- `RefreshCreating` reloaded a value written over an expired entry on its first read ([#230](https://github.com/maypok86/otter/pull/230))
- `CleanUp`, `Entry.HasExpired` and `LoadCacheFrom` treated an entry as live at its exact deadline. A clock started before 1970 expired every entry at once, and one whose readings crossed zero left passed deadlines unexpired ([#234](https://github.com/maypok86/otter/pull/234), [#268](https://github.com/maypok86/otter/pull/268))

Eviction and policy:

- Setting a key twice before the first insert was applied left a node that was counted in the size but could never be evicted, so the cache grew without bound ([#189](https://github.com/maypok86/otter/pull/189))
- Write-buffer tasks of one key replayed out of order left dead nodes and their weight in the policy ([#193](https://github.com/maypok86/otter/pull/193))
- An idle cache without expiration stayed over its maximum after `Hottest`, `Coldest` or `InvalidateAll` ([#288](https://github.com/maypok86/otter/pull/288))
- The hill climber kept a quota from the old maximum after `SetMaximum`, collapsed the window of a cache created with a small `InitialCapacity`, and counted a refresh as an access ([#260](https://github.com/maypok86/otter/pull/260), [#271](https://github.com/maypok86/otter/pull/271))
- `ComputeIfPresent` recorded a hit and called `ExpireAfterRead` before running its function ([#271](https://github.com/maypok86/otter/pull/271))
- An `InitialCapacity` above ~8e9 wrapped the table size to zero ([#252](https://github.com/maypok86/otter/pull/252))
- `InvalidateAll` and iteration (including `SaveCacheTo`) skipped keys whose node was replaced meanwhile, and `Hottest`/`Coldest` merged the window and probation queues in the wrong order ([#205](https://github.com/maypok86/otter/pull/205), [#207](https://github.com/maypok86/otter/pull/207), [#245](https://github.com/maypok86/otter/pull/245))

Listeners and stats:

- `OnDeletion` and `OnAtomicDeletion` could get different causes for one removal ([#196](https://github.com/maypok86/otter/pull/196))
- With a caller-runs executor, an eviction caused by a write was reported before the replacement that caused it ([#272](https://github.com/maypok86/otter/pull/272))
- `Set` or `Compute` racing a read could report an entry as present and as expired at the same time ([#290](https://github.com/maypok86/otter/pull/290))
- An expired entry evicted for size was reported as `CauseOverflow`, an expired entry removed by a write wasn't counted as an eviction, and a load whose result panicked wasn't counted at all ([#201](https://github.com/maypok86/otter/pull/201), [#235](https://github.com/maypok86/otter/pull/235), [#253](https://github.com/maypok86/otter/pull/253))

Persistence:

- `LoadCacheFrom` filled the cache in the snapshot's units instead of its own, moved restored deadlines later, recorded phantom hits, and lost its warm-up without `InitialCapacity` ([#259](https://github.com/maypok86/otter/pull/259), [#270](https://github.com/maypok86/otter/pull/270), [#274](https://github.com/maypok86/otter/pull/274))
- `SaveCacheToFile` wasn't atomic ([#268](https://github.com/maypok86/otter/pull/268))

Panics, `runtime.Goexit` and lifecycle:

- A panic or `runtime.Goexit` in a callback, loader, clock, executor, logger or stats recorder could leave a bucket or the eviction lock held, waiters hanging, a write outside the policies or a refresh unreported ([#191](https://github.com/maypok86/otter/pull/191), [#264](https://github.com/maypok86/otter/pull/264), [#265](https://github.com/maypok86/otter/pull/265), [#268](https://github.com/maypok86/otter/pull/268), [#286](https://github.com/maypok86/otter/pull/286), [#287](https://github.com/maypok86/otter/pull/287), [#295](https://github.com/maypok86/otter/pull/295))
- A cache whose callbacks referenced it was never garbage-collected, and collecting a cache created in a `testing/synctest` bubble crashed the program ([#279](https://github.com/maypok86/otter/pull/279))
- `StopAllGoroutines` deadlocked when called from `Hottest`, `Coldest` or a `SaveCacheTo` writer; a closed `Tick` channel made the clean-up goroutine spin; the default clock leaked its ticker ([#294](https://github.com/maypok86/otter/pull/294))
- `testing` and testify were compiled into every program that imported otter ([#185](https://github.com/maypok86/otter/issues/185), [#186](https://github.com/maypok86/otter/pull/186))

## 2.3.0 - 2025-12-22

### ✨Features

- A [plugin](https://github.com/maypok86/otter/tree/main/plugin/pslog) for the `slog` logger has been added.
- The `StopAllGoroutines` method has been added.

### 🚀 Improvements

- docs: add Nuclei to projects list ([#169](https://github.com/maypok86/otter/pull/169))

### 🐞 Bug Fixes

- fix: the return value check in the err-not-found example ([#156](https://github.com/maypok86/otter/issues/156))
- Panics when running under synctest ([#158](https://github.com/maypok86/otter/issues/158))
- Does not build with GOARCH=wasm ([#161](https://github.com/maypok86/otter/issues/161))
- fix: treat expired item refresh as create operation in ExpiryCreating ([#154](https://github.com/maypok86/otter/pull/154))

## 2.2.1 - 2025-07-22

### 🚀 Improvements

- Added more detailed explanations of the mechanics related to returning `ErrNotFound` ([#136](https://github.com/maypok86/otter/issues/136))

### 🐞 Bug Fixes

- Fix inconsistent singleflight results if the key is invalidated on the way ([#137](https://github.com/maypok86/otter/issues/137))
- Fix panic during concurrent execution of `InvalidateAll` and `Get` under high contention ([#139](https://github.com/maypok86/otter/issues/139))

## 2.2.0 - 2025-07-07

This release focuses on improving the integration experience with pull-based metric collectors.

### ✨Features

- Added `IsWeighted`, `IsRecordingStats` and `Stats` methods for cache ([#131](https://github.com/maypok86/otter/issues/131))
- Added `Minus` and `Plus` methods for `stats.Stats`
- Added `stats.Snapshoter` and `stats.SnapshotRecorder` interfaces

### 🚀 Improvements

- Reduced memory consumption of `stats.Counter` by 4 times

## 2.1.1 - 2025-07-05

### 🚀 Improvements

- `Get` now returns the value from the loader even when an error is returned. ([#132](https://github.com/maypok86/otter/issues/132))

## 2.1.0 - 2025-06-29

### ✨Features

- Added `Compute`, `ComputeIfAbsent` and `ComputeIfPresent` methods
- Added `LoadCacheFrom`, `LoadCacheFromFile`, `SaveCacheTo` and `SaveCacheToFile` functions
- Added `Clock` interface and option for time mocking
- Added `Keys` and `Values` iterators
- Added `Hottest` and `Coldest` iterators

### 🚀 Improvements

- Slightly reduced memory consumption
- Cache became significantly faster in cases when it's lightly populated
- Reduced number of allocations during refresh

### 🐞 Bug Fixes

- Fixed a bug in timer wheel ([#64](https://github.com/Yiling-J/theine-go/issues/64))
- Added usage of `context.WithoutCancel` during refresh execution ([#124](https://github.com/maypok86/otter/issues/124))

## 2.0.0 - 2025-06-18

### 📝 Description

Otter v2 has been completely redesigned for better performance and usability.

Key improvements:
- Completely rethought API for greater flexibility
- Added [loading](https://maypok86.github.io/otter/user-guide/v2/features/loading/) and [refreshing](https://maypok86.github.io/otter/user-guide/v2/features/refresh/) features ([#26](https://github.com/maypok86/otter/issues/26))
- Added [entry pinning](https://maypok86.github.io/otter/user-guide/v2/features/eviction/#pinning-entries)
- Replaced eviction policy with adaptive W-TinyLFU, enabling Otter to achieve one of the highest hit rates across **all** workloads.
- Added HashDoS protection against potential attacks
- The task scheduling mechanism has been completely reworked, allowing users to manage it themselves when needed
- Added more efficient write buffer
- Added auto-configurable lossy read buffer
- Optimized hash table
- Test coverage increased to 97%

### 🚨 Breaking Changes

1. **Cache Creation**
   - Removed `Builder` pattern in favor of canonical `Options` struct
   - `MustBuilder` and `Builder` methods are replaced with `Must` and `New` functions
   - `Cost` renamed to `Weight`
   - Replaced unified `capacity` with explicit `MaximumSize` and `MaximumWeight` parameters
   - Replaced `DeletionListener` with `OnDeletion` and `OnAtomicDeletion` handlers
   - The ability to create a cache with any combination of features

2. **Cache API Changes**
    - `Get` method renamed to `GetIfPresent`
    - `Set` method signature changed to return both value and bool
    - `SetIfAbsent` method signature changed to return both value and bool
    - `Delete` method renamed to `Invalidate`
    - `Clear` method renamed to `InvalidateAll`
    - `Size` method renamed to `EstimatedSize`
    - `Capacity` method renamed to `GetMaximum`
    - `Range` method removed in favor of `All` iterator
    - `Has` method removed
    - `DeleteByFunc` method removed
    - `Stats` method removed in favor of `stats.Recorder` interface
    - `Close` method removed

3. **Expiration**
    - Expiration API is now more flexible with `ExpiryCalculator` interface
    - `ExpiryCreating`, `ExpiryWriting` and `ExpiryAccessing` functions introduced

4. **Statistics**
    - Moved statistics to a separate package `stats`
    - `stats.Recorder` interface and `stats.Counter` struct introduced

5. **Extension**
    - `Extension` struct removed in favor of methods from `Cache`

### ✨Features

1. **Loading**
    - Added `Get` method for obtaining values if necessary
    - Added `Loader` interface for retrieving values from the data source
    - Added `ErrNotFound` error for indicating missing entries

2. **Refresh**
    - Added flexible refresh API with `RefreshCalculator` interface
   - `RefreshCreating`, `RefreshWriting` functions introduced
   - Added `Refresh` method for refreshing values asynchronously

3. **Bulk Operations**
    - Added `BulkGet` for loading multiple values at once
    - Added `BulkRefresh` for refreshing multiple values asynchronously
    - Added `BulkLoader` interface for retrieving multiple values from the data source at once

4. **Cache Methods**
    - Added `All` method for iterating over all entries
    - Added `CleanUp` method for performing pending maintenance operations
    - Added `WeightedSize` method for weight-based caches

5. **Enhanced Configuration**
    - Added `Executor` option for customizing async operations
    - Added `Logger` interface for custom logging

6. **Entry Management**
   - Added `SetExpiresAfter` and `SetRefreshableAfter` for per-entry time control
   - Added `GetEntry` and `GetEntryQuietly` methods for accessing cache entries
   - Most `Entry`'s methods replaced with public fields for direct access.

7. **Deletion Notifications**
   - Replaced `DeletionListener` with `OnDeletion` and `OnAtomicDeletion` handlers
   - Deletion causes renamed for clarity and consistency.
   - Added `IsEviction` method
   - Added `DeletionEvent` struct

8. **Performance Improvements**
    - Replaced `S3-FIFO` with adaptive `W-TinyLFU`
    - Added more efficient write buffer
    - Added auto-configurable lossy read buffer
    - The task scheduling mechanism has been completely reworked, allowing users to manage it themselves when needed.

### 🚀 Improvements

- Added [loading](https://maypok86.github.io/otter/user-guide/v2/features/loading/) and [refreshing](https://maypok86.github.io/otter/user-guide/v2/features/refresh/) features ([#26](https://github.com/maypok86/otter/issues/26))
- You can now pass a custom implementation of the `stats.Recorder` interface ([#119](https://github.com/maypok86/otter/issues/119))
- You can now use a TTL shorter than `time.Second` ([#115](https://github.com/maypok86/otter/issues/115))

## 1.2.4 - 2024-11-23

### 🐞 Bug Fixes

- Fixed a bug due to changing [gammazero/deque](https://github.com/gammazero/deque/pull/33) contracts without v2 release. ([#112](https://github.com/maypok86/otter/issues/112))

## 1.2.3 - 2024-09-30

### 🐞 Bug Fixes

- Added collection of eviction statistics for expired entries. ([#108](https://github.com/maypok86/otter/issues/108))

## 1.2.2 - 2024-08-14

### ✨️Features

- Implemented `fmt.Stringer` interface for `DeletionReason` type ([#100](https://github.com/maypok86/otter/issues/100))

### 🐞 Bug Fixes

- Fixed processing of an expired entry in the `Get` method ([#98](https://github.com/maypok86/otter/issues/98))
- Fixed inconsistent deletion listener behavior ([#98](https://github.com/maypok86/otter/issues/98))
- Fixed the behavior of `checkedAdd` when over/underflow ([#91](https://github.com/maypok86/otter/issues/91))

## 1.2.1 - 2024-04-15

### 🐞 Bug Fixes

- Fixed uint32 capacity overflow.

## 1.2.0 - 2024-03-12

The main innovation of this release is the addition of an `Extension`, which makes it easy to add a huge number of features to otter.

Usage example:

```go
key := 1
...
entry, ok := cache.Extension().GetEntry(key)
...
key := entry.Key()
value := entry.Value()
cost := entry.Cost()
expiration := entry.Expiration()
ttl := entry.TTL()
hasExpired := entry.HasExpired()
```

### ✨️Features

- Added `DeletionListener` to the builder ([#63](https://github.com/maypok86/otter/issues/63))
- Added `Extension` ([#56](https://github.com/maypok86/otter/issues/56))

### 🚀 Improvements

- Added support for Go 1.22
- Memory consumption with small cache sizes is reduced to the level of other libraries ([#66](https://github.com/maypok86/otter/issues/66))

## 1.1.1 - 2024-03-06

### 🐞 Bug Fixes

- Fixed alignment issues on 32-bit archs

## 1.1.0 - 2024-03-04

The main innovation of this release is node code generation. Thanks to it, the cache will no longer consume more memory due to features that it does not use. For example, if you do not need an expiration policy, then otter will not store the expiration time of each entry. It also allows otter to use more effective expiration policies.

Another expected improvement is the correction of minor synchronization problems due to the state machine. Now otter, unlike other contention-free caches in Go, should not have them at all.

### ✨️Features

- Added `DeleteByFunc` function to cache ([#44](https://github.com/maypok86/otter/issues/44))
- Added `InitialCapacity` function to builder ([#47](https://github.com/maypok86/otter/issues/47))
- Added collection of additional statistics ([#57](https://github.com/maypok86/otter/issues/57))

### 🚀 Improvements

- Added proactive queue-based and timer wheel-based expiration policies with O(1) time complexity ([#55](https://github.com/maypok86/otter/issues/55))
- Added node code generation ([#55](https://github.com/maypok86/otter/issues/55))
- Fixed the race condition when changing the order of events ([#59](https://github.com/maypok86/otter/issues/59))
- Reduced memory consumption on small caches

## 1.0.0 - 2024-01-26

### ✨️Features

- Builder pattern support
- Cleaner API compared to other caches ([#40](https://github.com/maypok86/otter/issues/40))
- Added `SetIfAbsent` and `Range` functions ([#27](https://github.com/maypok86/otter/issues/27))
- Statistics collection ([#4](https://github.com/maypok86/otter/issues/4))
- Cost based eviction
- Support for generics and any comparable types as keys
- Support ttl ([#14](https://github.com/maypok86/otter/issues/14))
- Excellent speed ([benchmark results](https://github.com/maypok86/otter?tab=readme-ov-file#-performance-))
- O(1) worst case time complexity for S3-FIFO instead of O(n)
- Improved hit ratio of S3-FIFO on many traces ([simulator results](https://github.com/maypok86/otter?tab=readme-ov-file#-hit-ratio-))
