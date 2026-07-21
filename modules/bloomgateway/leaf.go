package bloomgateway

// Leaf is the sorted (fingerprint, handle) entry array — the core
// membership+attribution structure a leaf directory slot (directory.go)
// holds (DESIGN.md § Data model, § Leaf lifecycle). One entry exists per
// (trace, live block) pair whose hash lands in this leaf, across every
// tenant; the map fuses membership and attribution, so there is no separate
// contributor list to maintain.
//
// Storage: two parallel slices, fps and handles, always the same length and
// kept sorted together by (fp, handle) — fp primary, handle secondary. This
// is what actually achieves DESIGN.md's stated "fp16 + handle32 = 6 B/entry"
// v1 encoding with no Go struct padding (a []struct{fp uint16; h Handle}
// would pad each entry to 8 B on a 64-bit platform), and it is directly
// binary-searchable on fps alone before disambiguating on handles.
//
// Concurrency: Leaf has NO internal locking. Every method here assumes the
// caller already holds the appropriate lock — directory.go's striped
// per-leaf-index locks (§ Concurrency: "leaf_idx mod 1024") — around every
// call, including read-only ones. This keeps Leaf itself a plain, fast,
// allocation-conscious data structure and puts exactly one place (the
// directory) in charge of concurrency safety for leaves.
type Leaf struct {
	fps     []uint16
	handles []Handle
}

// NewLeaf returns an empty leaf. Nil backing slices are deliberate: a leaf
// that never receives an insert (e.g. most leaves immediately after a
// reconstruction range flips to constructing, before any block's hash lands
// in it) should not pay for an allocation it may never use. Growth is
// amortized by Go's own append, matching DESIGN.md's "leaf arrays are
// slab-allocated with amortized growth" at the single-leaf level (cross-leaf
// slab sharing is a representation-notes-level optimization this type does
// not implement).
func NewLeaf() *Leaf {
	return &Leaf{}
}

// Len returns the number of entries currently in the leaf, including any
// garbage (entries referencing since-deleted blocks) awaiting the sweep.
func (l *Leaf) Len() int {
	return len(l.fps)
}

// InsertIfAbsent inserts (fp, h) iff the exact pair is not already present,
// maintaining sort order. Returns whether an insert actually happened.
//
// Insert-if-absent on the exact pair is what makes redelivery idempotent
// (DESIGN.md § Leaf object): re-applying the same AddChunk (retry,
// redelivery, replay) must never create a duplicate entry.
//
// NOT safe for concurrent use on its own — see the type doc comment.
func (l *Leaf) InsertIfAbsent(fp uint16, h Handle) (inserted bool) {
	idx := l.lowerBound(fp, h)
	if idx < len(l.fps) && l.fps[idx] == fp && l.handles[idx] == h {
		return false
	}

	// Standard sorted-slice insertion: grow by one, then shift everything
	// from idx onward one slot to the right (copy is memmove-safe on
	// overlapping slices per the Go spec), then write the new pair into
	// the opened slot. This is the O(n) memmove WP9's plan calls out as
	// the leaf's main cost driver at scale; BenchmarkLeafInsert measures
	// it directly at the reference ~596-entries/leaf sizing.
	l.fps = append(l.fps, 0)
	copy(l.fps[idx+1:], l.fps[idx:])
	l.fps[idx] = fp

	l.handles = append(l.handles, InvalidHandle)
	copy(l.handles[idx+1:], l.handles[idx:])
	l.handles[idx] = h

	return true
}

// Lookup returns every handle whose entry matches fp — zero, one, or more
// (collisions and legitimate multi-block duplication both produce >1).
// Returns nil (not an allocated empty slice) on zero matches; callers that
// need to distinguish "no leaf" from "leaf answered zero matches" do so one
// level up, in Directory.Lookup's (handles, ok) signature — that
// distinction is not this type's concern.
//
// NOT safe for concurrent use on its own — see the type doc comment.
func (l *Leaf) Lookup(fp uint16) []Handle {
	idx := l.lowerBound(fp, InvalidHandle)

	var out []Handle
	for idx < len(l.fps) && l.fps[idx] == fp {
		out = append(out, l.handles[idx])
		idx++
	}
	return out
}

// RemoveWhere deletes every entry whose handle does NOT satisfy keep,
// compacting the arrays in place and preserving relative (and therefore
// sort) order among the surviving entries. Returns the count removed.
//
// This is the sweep's compaction primitive (§ Garbage collection): the
// sweep gathers the set of currently-deleted handles once per pass (from
// the registry) and calls this per leaf with a set-membership predicate,
// rather than querying the registry once per entry.
//
// NOT safe for concurrent use on its own — see the type doc comment.
func (l *Leaf) RemoveWhere(keep func(Handle) bool) int {
	write := 0
	for read := 0; read < len(l.fps); read++ {
		if keep(l.handles[read]) {
			l.fps[write] = l.fps[read]
			l.handles[write] = l.handles[read]
			write++
		}
	}
	removed := len(l.fps) - write
	l.fps = l.fps[:write]
	l.handles = l.handles[:write]
	return removed
}

// Clone deep-copies the leaf: the returned Leaf shares no backing array
// with l, so mutating one never affects the other. This is the "build
// aside" half of the copy-rewire-place-back mutation mode (§ Mutation
// modes) — reconstruction and sweep-compaction build a replacement leaf via
// Clone (+ RemoveWhere, or a backfill pass), then hand it to
// Directory.Complete/Swap to rewire the directory slot in one atomic
// pointer swap.
//
// NOT safe for concurrent use on its own — see the type doc comment. In
// particular, Clone itself does not lock: the caller must hold whatever
// lock protects l for the duration of the copy, exactly as for any other
// method on this type.
func (l *Leaf) Clone() *Leaf {
	clone := &Leaf{}
	if l.fps != nil {
		clone.fps = make([]uint16, len(l.fps))
		copy(clone.fps, l.fps)
	}
	if l.handles != nil {
		clone.handles = make([]Handle, len(l.handles))
		copy(clone.handles, l.handles)
	}
	return clone
}

// lowerBound returns the smallest index i such that (l.fps[i], l.handles[i])
// is not less than (fp, h) in the leaf's own sort order (fp primary, handle
// secondary) — the standard sorted-slice insertion point for the pair.
//
// Lookup calls this with h = InvalidHandle (Handle's zero value, the
// smallest possible Handle) to find the start of fp's run regardless of
// which real handles are present: any actual handle is >= InvalidHandle, so
// (fp, InvalidHandle) sorts at or before every real (fp, *) entry.
//
// A plain inlined binary search (no comparator closure) is deliberate: this
// is the hot path BenchmarkLeafInsert/BenchmarkLeafLookup measure, and an
// indirect call per comparison would be pure overhead here — unlike e.g.
// vparquet5's binarySearch(compare func(int)(int,error)), there is no
// fallible I/O per comparison to justify the abstraction.
func (l *Leaf) lowerBound(fp uint16, h Handle) int {
	lo, hi := 0, len(l.fps)
	for lo < hi {
		mid := (lo + hi) / 2
		if l.fps[mid] < fp || (l.fps[mid] == fp && l.handles[mid] < h) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}
