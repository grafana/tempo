package bloomgateway

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// LeafState is a leaf directory slot's lifecycle state (DESIGN.md § Leaf
// lifecycle) — the completeness invariant's home. Every leaf is in exactly
// one of these states, and the safety of the whole design reduces to the
// rule stated on LeafComplete below.
type LeafState int32

const (
	// LeafNil means not served: not owned by this instance, or owned but
	// not yet constructed. Incoming writes for this leaf are dropped
	// (InsertLive returns applied=false); queries receive ok=false, the
	// safe-fallback signal (absence never rejects).
	LeafNil LeafState = iota

	// LeafConstructing means not served, but accumulating every live
	// write while a reconstruction task concurrently backfills history.
	// Queries still receive ok=false — a leaf must never answer from
	// partial state (DESIGN.md § Design constraints: "a leaf is never
	// served from partial state").
	LeafConstructing

	// LeafComplete means served: the leaf reflects every block committed
	// to the registry that contributes to it, and continuously receives
	// every live write. This is the only state Directory.Lookup answers
	// from.
	LeafComplete
)

// directoryStripes is the fixed number of locks striping the directory
// (DESIGN.md § Concurrency: "Striped leaf locks (leaf_idx mod 1024)").
// Fixed regardless of D: at the reference D=25 this means ~4096 leaves
// share each stripe, an accepted, statistically rare contention cost in
// exchange for a bounded (not 2^D-sized) lock count.
const directoryStripes = 1024

// directorySlot is one leaf address's state-and-reference pair, guarded by
// its stripe's lock. state and leaf change together, atomically with
// respect to any concurrent Lookup/InsertLive on the same index, because
// every mutating method below holds the stripe's write lock across both
// fields' updates.
//
// Representation note: DESIGN.md's § Representation notes' "2^D machine-
// word references = 256 MiB at D=25" describes only the memory for a bare
// per-leaf reference; distinguishing constructing from complete needs a
// second field, which (with the pointer's own alignment) makes each slot
// 16 B rather than 8 B on a 64-bit platform. That note is explicitly
// "non-normative... the logic above never depends on these choices" — this
// is a deliberate, documented simplification, not a correctness gap.
type directorySlot struct {
	state LeafState
	leaf  *Leaf
}

// Directory is the flat 2^D array of leaf slots and the nil/constructing/
// complete lifecycle state machine (DESIGN.md § Leaf directory, § Leaf
// lifecycle). It owns the striped locks so Leaf itself (leaf.go) never has
// to be concurrency-aware — every exported method here takes the
// appropriate stripe lock before touching a slot, and Leaf's own methods
// are only ever called from inside one of those critical sections.
//
// entries/constructing/complete back the entries_total and owned_leaves
// {state} gauges (metrics.go). They are maintained incrementally at this
// file's own centralized state-transition points (BeginConstructing,
// Complete, Shed, Abandon, CompactLeaf, Swap, InsertLive) rather than
// derived by walking the directory on every scrape — the walk this package
// must never reintroduce on a hot cadence (a prior defect, Phase C #5,
// was exactly an O(2^D) walk on a 1s ticker; see bloomgateway.go's
// runOwnershipWatch doc comment). entries is deliberately self-healed by a
// full recount at the end of every complete Sweeper.Pass (sweep.go) against
// any drift a missed accounting path might introduce; the leaf-state
// counters have no analogous recount because every transition between the
// three LeafState values is already centralized in this file with no other
// mutation path, so there is nothing left to drift from.
type Directory struct {
	slots   []directorySlot
	stripes [directoryStripes]sync.RWMutex

	entries      atomic.Int64
	constructing atomic.Int64
	complete     atomic.Int64
}

// NewDirectory allocates a directory with 2^d slots, every one starting in
// LeafNil (Go zero-values a freshly made slice, and LeafNil is LeafState's
// zero value by construction — types.go's BlockPending-is-zero-value
// convention, repeated here). NewDirectory does not itself validate d;
// bounding d to a sane range (DESIGN.md ties it to the ring's 32-bit token
// space) is Config.Validate's job (config.go), keeping this a pure,
// trusting, hot-path-adjacent constructor.
func NewDirectory(d uint8) *Directory {
	size := uint64(1) << uint64(d)
	return &Directory{
		slots: make([]directorySlot, size),
	}
}

// stripeFor returns idx's stripe index (DESIGN.md § Concurrency:
// "leaf_idx mod 1024").
func stripeFor(idx uint32) uint32 {
	return idx % directoryStripes
}

// Lookup resolves idx's leaf (if complete) and looks up fp in one locked
// call — the query path's sole entrypoint. ok=false means idx is nil or
// constructing (not served: the safe-fallback case, invariant #1, §7). A
// complete leaf with zero matching handles returns ok=true, handles=nil —
// a genuine, distinguishable answer (invariant #7, §7): the two ok values
// are never conflated in this signature.
func (dir *Directory) Lookup(idx uint32, fp uint16) (handles []Handle, ok bool) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.RLock()
	defer stripe.RUnlock()

	slot := &dir.slots[idx]
	if slot.state != LeafComplete {
		return nil, false
	}
	return slot.leaf.Lookup(fp), true
}

// InsertLive is the hot write-path entrypoint (§ Event processing step 2):
// a nil slot drops the write (applied=false, "if directory[leaf_idx] is
// nil, drop it"); a constructing or complete slot inserts under the stripe
// lock (insert-if-absent), which is exactly what lets a constructing leaf
// "accumulate every live write" while its backfill runs concurrently.
//
// entries_total's hot-path budget is one atomic Add per APPLIED entry, never
// per call: InsertIfAbsent's own return value gates it, so redelivery of an
// already-present (fp, h) pair -- the common case under at-least-once
// replay -- costs no atomic operation at all here.
func (dir *Directory) InsertLive(idx uint32, fp uint16, h Handle) (applied bool) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.Lock()
	defer stripe.Unlock()

	slot := &dir.slots[idx]
	if slot.state == LeafNil {
		return false
	}
	if slot.leaf.InsertIfAbsent(fp, h) {
		dir.entries.Add(1)
	}
	return true
}

// BeginConstructing transitions nil -> constructing, allocating a fresh
// empty *Leaf that starts accumulating live writes immediately (every
// subsequent InsertLive(idx, ...) call inserts into this exact object). A
// no-op (started=false, leaf=nil) if idx is already constructing or
// complete — reconstruction enqueuing an already-owned range must not
// discard in-flight or already-served state.
func (dir *Directory) BeginConstructing(idx uint32) (leaf *Leaf, started bool) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.Lock()
	defer stripe.Unlock()

	slot := &dir.slots[idx]
	if slot.state != LeafNil {
		return nil, false
	}
	slot.state = LeafConstructing
	slot.leaf = NewLeaf()
	dir.constructing.Add(1)
	return slot.leaf, true
}

// Complete transitions constructing -> complete, swapping in leaf as idx's
// served leaf. Returns an error if idx is not currently constructing
// (BeginConstructing was never called, or Complete was already called for
// this construction episode) — unlike BeginConstructing's benign no-op,
// this is a caller-contract violation worth surfacing rather than
// silently swallowing.
//
// The caller (the reconstruction queue) must guarantee leaf already
// reflects every live write applied since BeginConstructing — the "backfill
// pass is done and topic replay has caught up past the backfill's capture
// point" rule from DESIGN.md § Leaf lifecycle. Complete itself has no way
// to verify this; it only performs the state-and-reference swap.
//
// entries_total accounting: leaf is usually the SAME object BeginConstructing
// handed back (the reconstruction queue's normal path, processBatch in
// reconstruction.go), already fully counted via InsertLive as it accumulated
// -- in that case old and new length are equal and this is a zero-cost no-op
// delta. The one caller that passes a DIFFERENT object is
// bloomgateway.go's reconcileStartup, swapping in a snapshot-loaded leaf
// whose entries were never counted via InsertLive at all (they existed
// before this process's atomic counter did); the delta below is what
// accounts for them, exactly once, without needing reconcileStartup itself
// to know anything about entries_total.
func (dir *Directory) Complete(idx uint32, leaf *Leaf) error {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.Lock()
	defer stripe.Unlock()

	slot := &dir.slots[idx]
	if slot.state != LeafConstructing {
		return fmt.Errorf("bloomgateway: Complete called on leaf %d in state %v, want %v (LeafConstructing)", idx, slot.state, LeafConstructing)
	}

	var oldLen, newLen int
	if slot.leaf != nil {
		oldLen = slot.leaf.Len()
	}
	if leaf != nil {
		newLen = leaf.Len()
	}
	if delta := int64(newLen) - int64(oldLen); delta != 0 {
		dir.entries.Add(delta)
	}

	slot.state = LeafComplete
	slot.leaf = leaf
	dir.constructing.Add(-1)
	dir.complete.Add(1)
	return nil
}

// Shed transitions to nil unconditionally: one atomic state-and-reference
// change that stops serving and write-application in the same instant —
// there is no window where writes stop but serving continues, or vice
// versa, because both fields change together under the single stripe lock
// (DESIGN.md § Leaf lifecycle). Safe to call on any current state
// (complete, per the design's own "ownership is shed" trigger; also
// constructing, e.g. an in-flight backfill abandoned by an ownership
// change — dropping it is always safe, nothing was ever served from it).
// A no-op if idx is already nil.
//
// entries_total/owned_leaves accounting: whichever counter the PREVIOUS
// state belongs to (constructing or complete) is decremented, and any
// entries the discarded leaf held are subtracted -- ownership loss discards
// the leaf's memory outright, so it must leave both gauges exactly as it
// left the live structures.
func (dir *Directory) Shed(idx uint32) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.Lock()
	defer stripe.Unlock()

	slot := &dir.slots[idx]
	switch slot.state {
	case LeafConstructing:
		dir.constructing.Add(-1)
	case LeafComplete:
		dir.complete.Add(-1)
	case LeafNil:
		// Already nil: no-op, per this method's own documented contract.
	}
	if slot.leaf != nil {
		dir.entries.Add(-int64(slot.leaf.Len()))
	}

	slot.state = LeafNil
	slot.leaf = nil
}

// Swap replaces idx's leaf object in place without changing lifecycle
// state — the sweep's and reconstruction's "copy, rewire, place back" mode
// (§ Mutation modes) for a leaf that is already owned (constructing or
// complete): build a replacement aside (Leaf.Clone + RemoveWhere, or a
// backfill pass for an already-complete leaf being rebuilt under a D/F/seed
// change), then call Swap to rewire the directory slot in one atomic
// pointer swap that readers/writers on this index observe indivisibly.
//
// Swap does not itself validate idx's current state; calling it on a
// LeafNil slot plants an orphaned *Leaf that Lookup/InsertLive will never
// reach (both gate on state, not on leaf != nil) until a later
// BeginConstructing overwrites it — inert, but not a real intended use.
// Callers in this package only ever call Swap on an already-owned slot.
//
// entries_total accounting mirrors Complete's: the delta between the
// outgoing and incoming leaf's lengths is applied once, under the same
// stripe lock as the swap itself, so entries_total stays correct regardless
// of whether the replacement leaf shares no entries, all entries, or some
// entries with the one it replaces.
func (dir *Directory) Swap(idx uint32, leaf *Leaf) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.Lock()
	defer stripe.Unlock()

	var oldLen, newLen int
	if dir.slots[idx].leaf != nil {
		oldLen = dir.slots[idx].leaf.Len()
	}
	if leaf != nil {
		newLen = leaf.Len()
	}
	if delta := int64(newLen) - int64(oldLen); delta != 0 {
		dir.entries.Add(delta)
	}

	dir.slots[idx].leaf = leaf
}

// Range visits every non-nil slot in increasing idx order; fn returning
// false stops the walk early. Each slot's stripe lock is acquired and
// released individually (not held for the whole walk), so Range never
// blocks writers for longer than one slot's critical section — the sweep's
// and metrics' incremental-walk primitive (DESIGN.md § Garbage collection:
// "the sweep walks the leaf directory incrementally").
func (dir *Directory) Range(fn func(idx uint32, state LeafState) bool) {
	for i := 0; i < len(dir.slots); i++ {
		idx := uint32(i)
		stripe := &dir.stripes[stripeFor(idx)]
		stripe.RLock()
		state := dir.slots[i].state
		stripe.RUnlock()

		if state == LeafNil {
			continue
		}
		if !fn(idx, state) {
			return
		}
	}
}

// State returns idx's current lifecycle state.
func (dir *Directory) State(idx uint32) LeafState {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.RLock()
	defer stripe.RUnlock()
	return dir.slots[idx].state
}

// Leaf returns idx's current leaf object and lifecycle state, atomically,
// under its stripe's lock -- regardless of state, including LeafNil
// (leaf=nil) and LeafConstructing (the in-progress, live-write-accumulating
// leaf).
//
// Safety note: the returned *Leaf is the LIVE object -- once this call
// returns, the lock is released, and a concurrent InsertLive/Swap/Shed for
// the same idx can mutate or replace it at any time. This is only safe for
// a caller that either (a) never mutates or reads the returned leaf's
// contents once a concurrent writer could be active (e.g. snapshot.go's
// Save, documented to run only while the caller has already paused all
// mutation, § Snapshots consistency), or (b) is a single-threaded test
// asserting on final state after all concurrent activity has quiesced.
// Anything that runs alongside live writes and needs to safely read a
// leaf's full contents (the sweep, sweep.go) must use CloneLeaf instead.
//
// Implementation-plan deviation, reported prominently per the harness's own
// instructions for this case: the plan's WP16 (sweep.go) names only
// "Directory.Range/Swap" as its needed Directory surface, but Range's
// callback shape (idx, state) has no way to hand back the *Leaf itself --
// and the sweep's copy-rewire-place-back pass (§ Mutation modes) needs the
// actual object to Clone and RemoveWhere from. Lookup deliberately refuses
// to answer for anything but a complete leaf (by design, for the query
// path), so it cannot substitute either. This method (plus CloneLeaf
// below) is the smallest addition that unblocks the sweep without changing
// Range's, Lookup's, or Swap's existing signatures or behavior; it carries
// no new invariant of its own beyond what State+Swap already individually
// guarantee.
func (dir *Directory) Leaf(idx uint32) (*Leaf, LeafState) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.RLock()
	defer stripe.RUnlock()
	slot := &dir.slots[idx]
	return slot.leaf, slot.state
}

// CloneLeaf returns a safe, independent deep copy of idx's current leaf
// (nil if idx is LeafNil), taken atomically under the stripe's lock. This
// is the primitive a caller running ALONGSIDE live writes must use instead
// of Leaf -- Leaf.Clone() itself is documented as NOT safe for concurrent
// use on its own (leaf.go: "every method here assumes the caller already
// holds the appropriate lock... around every call, including read-only
// ones"), and calling it on a pointer obtained from Leaf AFTER that call's
// lock has already been released races a concurrent InsertLive (found by
// this WP's own -race test, TestSweep_ConcurrentPassWithLiveWrites, before
// this method existed). CloneLeaf holds the lock for the Clone call itself,
// closing that window; the returned copy shares no memory with the live
// leaf, so every subsequent operation on it (RemoveWhere, Lookup) needs no
// further synchronization.
func (dir *Directory) CloneLeaf(idx uint32) (*Leaf, LeafState) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.RLock()
	defer stripe.RUnlock()
	slot := &dir.slots[idx]
	if slot.leaf == nil {
		return nil, slot.state
	}
	return slot.leaf.Clone(), slot.state
}

// CompactLeaf filters idx's live leaf in place under a single stripe WRITE
// lock: it removes every entry whose handle does not satisfy keep and reports
// how many entries it visited and removed, plus whether the slot was complete
// (only complete leaves are compacted; nil/constructing slots are left
// untouched and report compacted=false).
//
// This exists specifically to close the sweep's lost-update race: doing the
// read (CloneLeaf) and the write-back (Swap) as two separate critical
// sections leaves a window in which a concurrent InsertLive appends to the
// live leaf, which the subsequent Swap of a stale filtered copy then silently
// discards — a dropped live entry is a false negative (the block that entry
// attributes to could later be wrongly rejected), the one error class this
// whole design forbids. Performing the filter in place while holding the
// write lock makes the whole read-modify-write atomic against InsertLive
// (which takes the same lock), so no concurrent insert can be lost. The
// per-leaf filter is O(entries) over the reference ~596-entry leaf — tens of
// microseconds — a proportionate cost to hold the stripe lock for, and the
// sweep is background work.
//
// entries_total is decremented by removed under this same lock -- the sweep
// is also entries_total's self-healing recount point (sweep.go's Pass), so
// this incremental decrement and that pass's own end-of-walk recount are
// deliberately redundant, not in tension: the decrement keeps the gauge
// accurate between passes, and the recount corrects it if it ever drifts.
func (dir *Directory) CompactLeaf(idx uint32, keep func(Handle) bool) (visited, removed int, compacted bool) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.Lock()
	defer stripe.Unlock()

	slot := &dir.slots[idx]
	if slot.state != LeafComplete || slot.leaf == nil {
		return 0, 0, false
	}
	visited = slot.leaf.Len()
	removed = slot.leaf.RemoveWhere(keep)
	if removed > 0 {
		dir.entries.Add(-int64(removed))
	}
	return visited, removed, true
}

// Abandon reverts idx from constructing back to nil, returning whether it did
// (a no-op, abandoned=false, if idx is not currently constructing). Unlike
// Shed — which drops from ANY state and is the ownership-change primitive —
// Abandon only ever undoes an in-flight construction episode, so a
// reconstruction batch that fails partway can roll back exactly the leaves it
// itself transitioned nil -> constructing without risking a complete leaf
// that some other path finished in the meantime. Without this, a failed batch
// would strand its leaves in constructing forever, and the readiness gate
// (which requires zero constructing leaves) would never open.
//
// entries_total/owned_leaves accounting: a failed batch's constructing leaf
// may already have accumulated live writes via InsertLive before the
// failure (reconstruction never pauses live application, DESIGN.md § Leaf
// lifecycle); Abandon discards that leaf outright, so its entries and its
// constructing-count contribution must both be given back here, exactly as
// Shed does for an ownership change.
func (dir *Directory) Abandon(idx uint32) (abandoned bool) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.Lock()
	defer stripe.Unlock()

	slot := &dir.slots[idx]
	if slot.state != LeafConstructing {
		return false
	}
	if slot.leaf != nil {
		dir.entries.Add(-int64(slot.leaf.Len()))
	}
	slot.state = LeafNil
	slot.leaf = nil
	dir.constructing.Add(-1)
	return true
}

// EntryTotal returns the current total leaf-entry count across every leaf
// this instance owns (complete or constructing), including any garbage
// awaiting the sweep -- the entries_total gauge's source (metrics.go),
// maintained incrementally by every mutation method above and self-healed
// once per complete Sweeper.Pass (sweep.go's own end-of-walk recount, via
// SetEntryTotal below).
func (dir *Directory) EntryTotal() int64 {
	return dir.entries.Load()
}

// SetEntryTotal overwrites the entries counter -- used exactly once per
// complete background sweep pass (sweep.go's Pass) to self-heal any drift
// the incremental accounting above might have accumulated, from a full,
// authoritative recount taken during that same pass's own directory walk.
// Not for use outside that one call site: every other caller should be
// adjusting the counter via a mutation above, not replacing it wholesale.
func (dir *Directory) SetEntryTotal(n int64) {
	dir.entries.Store(n)
}

// LeafStateCounts returns the current number of leaves this instance owns
// in LeafConstructing and LeafComplete respectively -- the owned_leaves
// {state="constructing|complete"} gauge's source, maintained incrementally
// at BeginConstructing/Complete/Shed/Abandon (the only places a leaf's
// state ever changes) rather than derived by a Range walk.
func (dir *Directory) LeafStateCounts() (constructing, complete int64) {
	return dir.constructing.Load(), dir.complete.Load()
}

// EntryLen returns idx's current leaf's entry count and lifecycle state,
// read together under the stripe's read lock so a concurrent InsertLive (or
// any other mutator, all of which take the write lock) can never be
// observed mid-mutation. This exists for the sweep's end-of-pass recount
// (sweep.go's Pass): CompactLeaf reports visited/removed only for COMPLETE
// leaves, so a constructing leaf's current length -- accumulating live
// writes throughout the pass -- needs this separate, still lock-safe read
// instead. Calling Leaf(idx) and then this leaf's own Len() would NOT be
// safe here: Leaf's own doc comment is explicit that the lock is released
// before it returns, so a concurrent InsertLive could be appending to the
// same backing slice this call would then read without synchronization.
func (dir *Directory) EntryLen(idx uint32) (int, LeafState) {
	stripe := &dir.stripes[stripeFor(idx)]
	stripe.RLock()
	defer stripe.RUnlock()
	slot := &dir.slots[idx]
	if slot.leaf == nil {
		return 0, slot.state
	}
	return slot.leaf.Len(), slot.state
}
