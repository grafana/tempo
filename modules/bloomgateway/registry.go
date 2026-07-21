package bloomgateway

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// Block is one live-or-formerly-live block known to this instance
// (DESIGN.md § Data model's "Block object"). The registry is the single
// source of truth for "this instance may reject this block": only a Block
// in BlockLive or BlockLiveUnsupportedEncoding is in the registry's "live"
// view, and only BlockLive/BlockLiveUnsupportedEncoding blocks are ever
// inserted into A_T (tenant.go) — and only A_T membership makes a block
// rejectable (§ Failure handling's single defense-covers-everything rule).
//
// Concurrency: UUID, TenantID, StartTime, EndTime, and Handle are set once,
// under the registry's lock, at creation (GetOrCreate) and never reassigned
// again — so a *Block returned by any Registry accessor is safe to read
// those fields from indefinitely, with no re-locking, exactly like any
// other immutable-after-construction value. State and DeletedAt DO mutate
// later (CommitLive, MarkDeleted), always under the registry's lock; a
// caller that retains a *Block across a window where a concurrent mutation
// could occur and reads State/DeletedAt directly (bypassing a Registry
// accessor) is reading without the lock's protection. In this design's
// actual call pattern every consumer re-derives a fresh view via
// LookupHandle/LookupUUID/Range/ResolveHandles rather than caching a *Block
// across time, so this is a theoretical sharp edge rather than an exercised
// one — documented here so it stays that way.
type Block struct {
	UUID      backend.UUID
	TenantID  string
	StartTime time.Time
	EndTime   time.Time
	State     BlockState
	Handle    Handle

	// DeletedAt is the zero time.Time unless State == BlockDeleted, in
	// which case it is this instance's own local processing time of the
	// Delete event — not any timestamp carried on the wire. The sweep's
	// tombstone-reclamation check (§7 invariant #9) measures "how long
	// ago did *I* learn this block is gone" against the replay horizon,
	// because that is what bounds the risk of a stale Add still in flight
	// on the topic resurrecting a reclaimed tombstone.
	DeletedAt time.Time
}

// Registry is the UUID<->handle interning table and block state machine
// (DESIGN.md § Data model's "block registry"). One registry lock (not
// striped) covers every mutation — matching § Concurrency's "a registry
// lock covers block commit": block churn is cell-wide ~1/s, orders of
// magnitude below the per-trace leaf hot path that justifies directory.go's
// striping.
type Registry struct {
	mu sync.RWMutex

	byUUID   map[backend.UUID]*Block
	byHandle map[Handle]*Block

	// nextHandle is the monotonic handle allocator. It only ever
	// increments, even across Reclaim — handles are "never reclaimed;
	// 2^32 handles outlast any realistic cell lifetime" (DESIGN.md §
	// Representation notes) — so a handle value, once handed out, is
	// never handed out again for the lifetime of this Registry.
	nextHandle Handle

	// liveCount backs the blocks_live gauge (metrics.go): the number of
	// blocks currently in the registry's "live" view (BlockLive or
	// BlockLiveUnsupportedEncoding). Maintained incrementally at CommitLive's
	// and MarkDeleted's own exactly-once state-transition points (see their
	// doc comments) rather than derived from a Range walk over the
	// ~100k-block registry on every scrape.
	liveCount atomic.Int64
}

// NewRegistry returns an empty registry. nextHandle starts at
// InvalidHandle+1 so InvalidHandle (the zero value, "never allocated")
// itself is never allocated.
func NewRegistry() *Registry {
	return &Registry{
		byUUID:     make(map[backend.UUID]*Block),
		byHandle:   make(map[Handle]*Block),
		nextHandle: InvalidHandle + 1,
	}
}

// GetOrCreate returns the existing block for uuid, or creates one in
// BlockPending with a freshly interned handle. start/end are recorded only
// when actually creating (isNew=true) — safe to pass on every call
// regardless of which chunk of a block arrives first, since chunk arrival
// order is not guaranteed (§ Event processing).
//
// Double-checked locking (RLock fast path, re-check under Lock) rather than
// an unconditional write lock: multiple chunks of the same block routinely
// arrive on different worker-pool goroutines concurrently (AMENDMENT A2),
// so the common "block already exists" case should not contend on the
// write lock against unrelated blocks' first-chunk creations.
func (r *Registry) GetOrCreate(uuid backend.UUID, tenantID string, start, end time.Time) (block *Block, isNew bool) {
	r.mu.RLock()
	if b, ok := r.byUUID[uuid]; ok {
		r.mu.RUnlock()
		return b, false
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Re-check: another goroutine may have created uuid's Block between
	// the RUnlock above and this Lock.
	if b, ok := r.byUUID[uuid]; ok {
		return b, false
	}

	h := r.nextHandle
	r.nextHandle++

	b := &Block{
		UUID:      uuid,
		TenantID:  tenantID,
		StartTime: start,
		EndTime:   end,
		State:     BlockPending,
		Handle:    h,
	}
	r.byUUID[uuid] = b
	r.byHandle[h] = b
	return b, true
}

// CommitLive applies the registry's state-machine transitions that a live
// or synthetic Add can trigger (DESIGN.md § Data model, as amended by
// AMENDMENT A1):
//
//	BlockPending                 -> BlockLive                 (unsupportedEncoding=false)
//	BlockPending                 -> BlockLiveUnsupportedEncoding (unsupportedEncoding=true)
//	BlockLive                    -> BlockLive                 (no-op)
//	BlockLive                    -> BlockLiveUnsupportedEncoding (AMENDMENT A1 demotion)
//	BlockLiveUnsupportedEncoding -> BlockLiveUnsupportedEncoding (no-op)
//	BlockLiveUnsupportedEncoding -> BlockLive                 (ILLEGAL, AMENDMENT A1: rejected with an error, state unchanged)
//	BlockDeleted                 -> (anything)                (no-op: tombstone terminality, invariant #4 §7)
//
// This is also the registry-level primitive AMENDMENT A1's
// Applier.CommitUnsupportedEncoding (WP13) is built on: calling
// CommitLive(uuid, true) on an already-BlockLive block performs exactly the
// "demote it — set LiveUnsupportedEncoding" half of that amendment; the
// caller is responsible for the other half (removing the block's handle
// from A_T, tenant.go's TenantSet.RemoveBlock) since Registry has no
// reference to the TenantSet.
//
// Returns an error only for the illegal LiveUnsupportedEncoding->Live
// transition, or if uuid has never been seen (GetOrCreate must always
// precede CommitLive for a real UUID — unlike MarkDeleted, there is no
// documented allowance for committing an unknown block).
func (r *Registry) CommitLive(uuid backend.UUID, unsupportedEncoding bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.byUUID[uuid]
	if !ok {
		return fmt.Errorf("bloomgateway: CommitLive: unknown block %s", uuid)
	}

	switch b.State {
	case BlockDeleted:
		return nil // tombstone terminality: never resurrect

	case BlockPending:
		if unsupportedEncoding {
			b.State = BlockLiveUnsupportedEncoding
		} else {
			b.State = BlockLive
		}
		// blocks_live's exactly-once increment point. BlockPending is
		// BlockState's zero value, set only once (GetOrCreate), and this
		// switch has no branch that ever sets a block back to BlockPending
		// -- so this case fires at most once per block's lifetime, entering
		// the registry's "live" view (BlockLive or
		// BlockLiveUnsupportedEncoding, blocksLive's Help text) exactly
		// once. The Live<->LiveUnsupportedEncoding branches below only ever
		// move BETWEEN the two already-"live" states, so they must not
		// touch this counter.
		r.liveCount.Add(1)
		return nil

	case BlockLive:
		if unsupportedEncoding {
			b.State = BlockLiveUnsupportedEncoding // AMENDMENT A1 demotion
		}
		return nil // Live -> Live is a harmless no-op

	case BlockLiveUnsupportedEncoding:
		if !unsupportedEncoding {
			return fmt.Errorf("bloomgateway: CommitLive: illegal transition for block %s: %s -> %s is not a legal transition (AMENDMENT A1)", uuid, BlockLiveUnsupportedEncoding, BlockLive)
		}
		return nil // stays LiveUnsupportedEncoding

	default:
		return fmt.Errorf("bloomgateway: CommitLive: block %s has unrecognized state %s", uuid, b.State)
	}
}

// MarkDeleted transitions to BlockDeleted (terminal) and stamps DeletedAt
// with this instance's local processing time. A no-op for an unknown uuid
// (§ Event processing: "Delete for an unknown block is a no-op") and
// idempotent for a block already BlockDeleted (redelivery of the same
// Delete must not disturb the original DeletedAt... note it intentionally
// DOES refresh DeletedAt on redelivery below only if not yet deleted; a
// second MarkDeleted call on an already-deleted block is a pure no-op that
// leaves the original DeletedAt untouched, which is what the replay-horizon
// check needs to stay meaningful).
func (r *Registry) MarkDeleted(uuid backend.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.byUUID[uuid]
	if !ok {
		return nil
	}
	if b.State == BlockDeleted {
		return nil
	}

	// blocks_live's exactly-once decrement point, gated by the guard just
	// above (b.State == BlockDeleted -> return nil): this assignment runs at
	// most once per block's lifetime, matching CommitLive's own exactly-once
	// increment. wasLive must be captured before the state is overwritten
	// below, and must only trigger the decrement if the block was actually
	// counted live in the first place -- a block deleted while still
	// BlockPending (a Delete racing ahead of a still-chunking Add, § Event
	// processing) was never counted, and decrementing here would underflow
	// the gauge.
	wasLive := b.State == BlockLive || b.State == BlockLiveUnsupportedEncoding

	b.State = BlockDeleted
	b.DeletedAt = time.Now()
	if wasLive {
		r.liveCount.Add(-1)
	}
	return nil
}

// LiveCount returns the current number of registry blocks in the "live"
// view (BlockLive or BlockLiveUnsupportedEncoding) -- the blocks_live
// gauge's source (metrics.go), maintained incrementally by CommitLive and
// MarkDeleted rather than derived from a Range walk over the registry.
func (r *Registry) LiveCount() int64 {
	return r.liveCount.Load()
}

// LookupHandle returns the block h was interned for, if any.
func (r *Registry) LookupHandle(h Handle) (*Block, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.byHandle[h]
	return b, ok
}

// LookupUUID returns the block registered for uuid, if any.
func (r *Registry) LookupUUID(uuid backend.UUID) (*Block, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.byUUID[uuid]
	return b, ok
}

// State returns uuid's current block state under the registry lock, plus
// whether uuid is known at all. This is the lock-safe way to read State:
// reading it off a *Block returned by LookupUUID/GetOrCreate races with the
// concurrent CommitLive/MarkDeleted that this file's own type doc warns
// about. The event applier (events.go) needs an authoritative "is this block
// already Deleted / no longer Pending" read on the hot Add path, and the
// completion/Delete race-resolution depends on it — so the read must go
// through the lock, not through a cached field.
func (r *Registry) State(uuid backend.UUID) (BlockState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.byUUID[uuid]
	if !ok {
		return BlockPending, false
	}
	return b.State, true
}

// Range visits every block regardless of state — used by the sweep (a scan
// for reclaimable tombstones) and by snapshot serialization. fn returning
// false stops the walk early. Held under a single RLock for the whole
// walk: at the reference ~100k-block registry size this is a fast map
// iteration (§ Sizing: "Block registry (100k blocks) ~10 MiB"), a brief and
// proportionate block on the infrequent (~1/s) writer side.
func (r *Registry) Range(fn func(*Block) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, b := range r.byUUID {
		if !fn(b) {
			return
		}
	}
}

// Reclaim removes a Deleted block's registry entry entirely, from both
// indexes atomically (a concurrent LookupUUID/LookupHandle can never
// observe uuid present in one index but absent from the other, since both
// deletes happen inside the same critical section). Performs the removal
// only — the caller (the sweep) is responsible for verifying its own
// preconditions (zero remaining leaf entries AND DeletedAt past the replay
// horizon, §7 invariant #9) before calling this; Reclaim does not
// re-check State itself.
func (r *Registry) Reclaim(uuid backend.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.byUUID[uuid]
	if !ok {
		return
	}
	delete(r.byUUID, uuid)
	delete(r.byHandle, b.Handle)
}

// Import replaces the registry's entire block set with blocks, preserving
// each block's exact Handle -- the snapshot-restore counterpart to
// TenantSet.Import (tenant.go), needed for the identical reason: leaf
// entries loaded from the same snapshot (snapshot.go's State.CompleteLeaves)
// reference these EXACT Handle values, so a restored block must resolve to
// the SAME handle it had before the restart, not a freshly re-allocated one
// from GetOrCreate's own sequential counter. Callers (the top-level
// orchestrator's snapshot-load path) use Import only on a freshly
// constructed, still-empty Registry -- it replaces wholesale, mirroring
// TenantSet.Import's own documented restriction, and does not merge with
// any pre-existing state.
//
// Deviation from the implementation plan, reported per the harness's own
// instructions: neither this WP's plan sketch nor the snapshot WP's lists
// this method, but the snapshot State's Blocks field records each Block's
// Handle explicitly, and there is no way to restore that handle through
// GetOrCreate's public contract -- its handle allocation is strictly
// sequential and unparameterized, so a bare replay of GetOrCreate calls
// against a fresh Registry cannot reproduce a handle sequence with gaps
// (e.g. from a block reclaimed before the snapshot was taken). This is the
// smallest addition that closes that gap; it carries no new invariant
// beyond what GetOrCreate/nextHandle's own "never reclaimed, never reused"
// contract already promises.
func (r *Registry) Import(blocks []Block) error {
	byUUID := make(map[backend.UUID]*Block, len(blocks))
	byHandle := make(map[Handle]*Block, len(blocks))
	nextHandle := InvalidHandle + 1
	var liveCount int64

	for i := range blocks {
		b := blocks[i]
		if b.Handle == InvalidHandle {
			return fmt.Errorf("bloomgateway: registry import: block %s has invalid handle", b.UUID)
		}
		if _, dup := byUUID[b.UUID]; dup {
			return fmt.Errorf("bloomgateway: registry import: duplicate uuid %s", b.UUID)
		}
		if _, dup := byHandle[b.Handle]; dup {
			return fmt.Errorf("bloomgateway: registry import: duplicate handle %d", b.Handle)
		}

		block := b
		byUUID[b.UUID] = &block
		byHandle[b.Handle] = &block
		if b.Handle >= nextHandle {
			nextHandle = b.Handle + 1
		}
		// liveCount is recomputed wholesale here rather than incrementally:
		// Import bulk-replaces state built outside CommitLive/MarkDeleted's
		// own exactly-once transition points (a snapshot's Blocks already
		// carry their final State), so there is nothing to increment from.
		if b.State == BlockLive || b.State == BlockLiveUnsupportedEncoding {
			liveCount++
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byUUID = byUUID
	r.byHandle = byHandle
	r.nextHandle = nextHandle
	r.liveCount.Store(liveCount)
	return nil
}

// ResolveHandles resolves each handle in hs to its block's UUID under a
// single RLock acquisition — the query path's batch alternative to calling
// LookupHandle once per handle when resolving an unscoped rejection set,
// which can carry on the order of 100k handles (§ Response size).
//
// AMENDMENT A5: added because resolving such a set through one
// Registry.LookupHandle call per handle (one lock acquisition each) is a
// needless hot-path cost; the query gRPC API (WP15) is this method's
// consumer, using it to turn the matched/rejected handle sets into the
// wire-format UUID list (§ Protocol).
//
// A handle with no registered block resolves to the zero UUID; this should
// not happen in practice, since every handle a caller resolves here comes
// from A_T, which only ever holds handles for currently-live blocks still
// present in the registry. Callers that need to distinguish "resolved to
// the zero UUID because unregistered" from "a real, valid zero UUID" (it
// cannot occur in practice, since backend.UUID is generated by
// google/uuid, but is not statically impossible) should use LookupHandle
// instead.
func (r *Registry) ResolveHandles(hs []Handle) []backend.UUID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]backend.UUID, len(hs))
	for i, h := range hs {
		if b, ok := r.byHandle[h]; ok {
			out[i] = b.UUID
		}
	}
	return out
}
