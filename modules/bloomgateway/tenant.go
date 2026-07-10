package bloomgateway

import (
	"fmt"
	"sync"
	"time"

	"github.com/RoaringBitmap/roaring/v2"
)

// bucketDuration is the width of a single A_T bucket (DESIGN.md § Data
// model: "1h buckets"). A package constant, not a config knob: DESIGN.md
// states 1h without framing it as tunable (WP12's own plan text).
const bucketDuration = time.Hour

// bucketKey identifies one bucketDuration-wide bucket: the number of whole
// bucketDuration periods since the Unix epoch. An integer key (rather than
// carrying full time.Time values, with their monotonic-reading baggage, as
// map keys) is cheaper to hash/compare and unambiguous.
type bucketKey int64

// bucketKeyUpperBound returns the largest bucketKey whose bucket
// [k*bucketDuration, (k+1)*bucketDuration] could contain a point <= t —
// i.e. floor(t / bucketDuration). Used as the "last" bound derived from a
// range's end.
func bucketKeyUpperBound(t time.Time) bucketKey {
	return bucketKey(floorDiv(t.UnixNano(), int64(bucketDuration)))
}

// bucketKeyLowerBound returns the smallest bucketKey whose bucket
// [k*bucketDuration, (k+1)*bucketDuration] could contain a point >= t.
//
// Buckets are treated as closed intervals that share their boundary
// instant with their neighbor, deliberately: DESIGN.md § Time-range
// scoping states that bucketing errors may only ever omit (safe: a block
// opened unnecessarily) or over-include (safe: rejecting a block the QF
// wasn't going to open anyway) — never cause a wrong rejection. Treating a
// timestamp that lands exactly on an hour boundary as touching BOTH
// adjacent buckets is the over-inclusion side of that tolerance, applied
// consistently to every boundary rather than picking one neighbor
// arbitrarily.
//
// Concretely: if t is an exact multiple of bucketDuration (t = m *
// bucketDuration), it is the shared right endpoint of bucket m-1 as well as
// the left endpoint of bucket m, so the lower bound is m-1, one earlier
// than plain floor(t/bucketDuration) would give. If t is not an exact
// multiple, t only touches the bucket floor(t/bucketDuration) is in, so the
// lower bound is that floor value. Both cases are exactly
// ceil(t/bucketDuration) - 1.
func bucketKeyLowerBound(t time.Time) bucketKey {
	return bucketKey(ceilDiv(t.UnixNano(), int64(bucketDuration)) - 1)
}

// bucketRange returns the inclusive [first, last] bucketKey range that
// [start, end] (start <= end) overlaps, under the closed-interval,
// over-inclusive-at-boundaries convention documented on
// bucketKeyLowerBound. Both AddBlock (a block's own time range) and Window
// (a query's time range) share this one derivation, so a block landing
// exactly on a bucket boundary and a query window landing exactly on one
// are treated identically.
func bucketRange(start, end time.Time) (first, last bucketKey) {
	first = bucketKeyLowerBound(start)
	last = bucketKeyUpperBound(end)
	if last < first {
		// Defensive guard, not a documented semantic: only reachable if a
		// caller passes start > end, which never happens for a real
		// block's time range or a real query window. Falls back to the
		// single bucket touching start rather than an inverted/empty
		// range.
		last = first
	}
	return first, last
}

// floorDiv and ceilDiv are floor/ceiling integer division, correct for
// negative a (Go's native / truncates toward zero, not toward negative
// infinity) — defensive correctness for timestamps before the Unix epoch,
// which does not arise from real block metadata but costs nothing to
// handle properly. b is always bucketDuration's positive nanosecond count
// in this file's callers.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func ceilDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) == (b < 0) {
		q++
	}
	return q
}

// tenant is one tenant's A_T state: 1h buckets of block handles, plus a
// reverse Handle -> buckets index so RemoveBlock needs no external
// time-range input (it isn't in DESIGN.md's own representation notes, which
// only describe the forward direction, but is small — one []bucketKey per
// live handle, typically length 1 or 2 — and is what lets Delete-event
// application remove a block's A_T membership without re-deriving its
// bucket span from a StartTime/EndTime the caller may not have handy).
type tenant struct {
	mu            sync.RWMutex
	buckets       map[bucketKey]*roaring.Bitmap
	handleBuckets map[Handle][]bucketKey
}

// TenantSet is every tenant's A_T state (DESIGN.md § Data model's "Tenant
// object"). Two lock levels: TenantSet.mu guards the tenants map itself
// (creating a new tenant on first use); each tenant's own mu guards its
// buckets — matching § Concurrency's "a per-tenant lock covers A_T" (one
// lock per tenant, not a single cell-wide lock, and not per-bucket
// striping — bucket counts per tenant are small, § Sizing: "~100 KiB/
// tenant").
type TenantSet struct {
	mu      sync.RWMutex
	tenants map[string]*tenant
}

// NewTenantSet returns an empty TenantSet.
func NewTenantSet() *TenantSet {
	return &TenantSet{tenants: make(map[string]*tenant)}
}

// getOrCreateTenant returns tenantID's tenant, creating it lazily on first
// use (§ Multi-tenant cells: "first Add creates state lazily"). Double-
// checked locking: the common case (tenant already exists) only needs the
// cheap RLock path.
func (ts *TenantSet) getOrCreateTenant(tenantID string) *tenant {
	ts.mu.RLock()
	t, ok := ts.tenants[tenantID]
	ts.mu.RUnlock()
	if ok {
		return t
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()
	// Re-check: another goroutine may have created tenantID's first-ever
	// block concurrently between the RUnlock above and this Lock.
	if t, ok = ts.tenants[tenantID]; ok {
		return t
	}
	t = &tenant{
		buckets:       make(map[bucketKey]*roaring.Bitmap),
		handleBuckets: make(map[Handle][]bucketKey),
	}
	ts.tenants[tenantID] = t
	return t
}

// AddBlock adds h to every 1h bucket [start, end] overlaps (invariant #8,
// §7: membership in EVERY overlapping bucket, not just one — see
// bucketRange's boundary-inclusive derivation), lazily creating the tenant
// on first use.
//
// A no-op if h is already recorded for tenantID (defensive insert-if-
// absent, matching this design's idempotent-application convention
// elsewhere): the caller (the Add-applier) is expected to call AddBlock at
// most once per handle, exactly when a block's single Pending -> Live
// transition fires, but guarding here costs nothing and directly prevents
// a double-insert if that invariant is ever violated upstream.
func (ts *TenantSet) AddBlock(tenantID string, h Handle, start, end time.Time) {
	t := ts.getOrCreateTenant(tenantID)
	first, last := bucketRange(start, end)

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.handleBuckets[h]; exists {
		return
	}

	keys := make([]bucketKey, 0, last-first+1)
	for k := first; k <= last; k++ {
		bm, ok := t.buckets[k]
		if !ok {
			bm = roaring.New()
			t.buckets[k] = bm
		}
		bm.Add(uint32(h))
		keys = append(keys, k)
	}
	t.handleBuckets[h] = keys
}

// RemoveBlock removes h from every bucket it was ever added to, via the
// reverse index — no start/end input needed. A no-op if tenantID or h is
// unknown (mirrors the registry's Delete-for-unknown-block-is-a-no-op
// convention, § Event processing).
func (ts *TenantSet) RemoveBlock(tenantID string, h Handle) {
	ts.mu.RLock()
	t, ok := ts.tenants[tenantID]
	ts.mu.RUnlock()
	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	keys, ok := t.handleBuckets[h]
	if !ok {
		return
	}
	for _, k := range keys {
		if bm, ok := t.buckets[k]; ok {
			bm.Remove(uint32(h))
		}
	}
	delete(t.handleBuckets, h)
}

// Window returns the union of every one of tenantID's A_T buckets
// overlapping [start, end) — the query path's A_X_window (§ Query path step
// 3). A zero start AND zero end means unscoped: union of every bucket the
// tenant has. More generally, an individually-zero start or end is treated
// as unbounded on that side (the proto request's own "0 == unbounded"
// convention per bound, § Protocol) — the fully-unscoped case is simply
// what happens when both sides are independently unbounded, not a
// separate code path.
//
// Returns a non-nil, empty bitmap (never nil) for an unknown tenant or a
// window with no overlapping buckets — callers can unconditionally call
// ToArray()/GetCardinality() on the result.
func (ts *TenantSet) Window(tenantID string, start, end time.Time) *roaring.Bitmap {
	ts.mu.RLock()
	t, ok := ts.tenants[tenantID]
	ts.mu.RUnlock()
	if !ok {
		return roaring.New()
	}

	var first, last bucketKey
	var hasFirst, hasLast bool
	if !start.IsZero() {
		first = bucketKeyLowerBound(start)
		hasFirst = true
	}
	if !end.IsZero() {
		last = bucketKeyUpperBound(end)
		hasLast = true
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	bitmaps := make([]*roaring.Bitmap, 0, len(t.buckets))
	for k, bm := range t.buckets {
		if hasFirst && k < first {
			continue
		}
		if hasLast && k > last {
			continue
		}
		bitmaps = append(bitmaps, bm)
	}
	// FastOr reads its inputs and returns a wholly independent result
	// bitmap (verified against this vendored version: its internal lazyOR
	// only ever appends copies of source containers into a fresh answer,
	// never mutates x1/x2) — safe to call while holding only a read lock,
	// and safe to return without holding any lock afterward.
	return roaring.FastOr(bitmaps...)
}

// DropTenant removes tenantID's A_T state immediately (leaf entries remain
// until swept — § Multi-tenant cells' tenant lifecycle).
func (ts *TenantSet) DropTenant(tenantID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.tenants, tenantID)
}

// DropEmptyBuckets removes tenantID's now-empty buckets (§ Garbage
// collection: "Empty A_T buckets are dropped on the same pass" as the
// sweep). Safe to call unconditionally: a bucket can only be empty here if
// every handle it ever held has already been removed via RemoveBlock,
// which also removes that bucket's key from the removed handle's own
// handleBuckets entry — so no dangling reverse-index reference to a
// dropped bucket can exist.
func (ts *TenantSet) DropEmptyBuckets(tenantID string) {
	ts.mu.RLock()
	t, ok := ts.tenants[tenantID]
	ts.mu.RUnlock()
	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	for k, bm := range t.buckets {
		if bm.IsEmpty() {
			delete(t.buckets, k)
		}
	}
}

// TenantSetSnapshot is TenantSet's on-disk-serializable form (DESIGN.md §
// Snapshots: "... A_T sets"): tenantID -> bucketKey -> that bucket's
// roaring bitmap, pre-serialized via roaring.Bitmap.ToBytes() so
// snapshot.go's own binary encoding never needs to reach into roaring's
// internals -- it treats each bucket's bytes as one opaque, already-framed
// blob.
//
// WP12 addendum (per the implementation plan's own WP17 dependency note:
// "if WP12 has already landed without it by the time this is read, treat
// it as a one-method addendum to WP12, not a new WP"): TenantSet had no
// Export/Import pair when WP17 (snapshot.go) needed one, so this type and
// the two methods below were added directly to this file rather than as a
// separate work package.
type TenantSetSnapshot struct {
	Buckets map[string]map[bucketKey][]byte
}

// Export returns a serializable snapshot of every tenant's current A_T
// state. Every bitmap is copied out via ToBytes, so the returned snapshot
// shares no memory with the live TenantSet: mutating either afterward
// never affects the other.
func (ts *TenantSet) Export() (TenantSetSnapshot, error) {
	ts.mu.RLock()
	tenantIDs := make([]string, 0, len(ts.tenants))
	tenants := make([]*tenant, 0, len(ts.tenants))
	for id, t := range ts.tenants {
		tenantIDs = append(tenantIDs, id)
		tenants = append(tenants, t)
	}
	ts.mu.RUnlock()

	out := TenantSetSnapshot{Buckets: make(map[string]map[bucketKey][]byte, len(tenantIDs))}
	for i, id := range tenantIDs {
		t := tenants[i]

		t.mu.RLock()
		buckets := make(map[bucketKey][]byte, len(t.buckets))
		for k, bm := range t.buckets {
			b, err := bm.ToBytes()
			if err != nil {
				t.mu.RUnlock()
				return TenantSetSnapshot{}, fmt.Errorf("bloomgateway: exporting tenant %q bucket %d: %w", id, k, err)
			}
			buckets[k] = b
		}
		t.mu.RUnlock()

		out.Buckets[id] = buckets
	}
	return out, nil
}

// Import replaces ts's entire tenant/bucket state with snap's contents,
// rebuilding each tenant's handleBuckets reverse index from the decoded
// bitmaps. Callers use Import only on a freshly constructed TenantSet
// (snapshot.go's Load path, via the orchestrator) -- it replaces
// wholesale, it does not merge with any pre-existing state.
func (ts *TenantSet) Import(snap TenantSetSnapshot) error {
	tenants := make(map[string]*tenant, len(snap.Buckets))
	for id, buckets := range snap.Buckets {
		t := &tenant{
			buckets:       make(map[bucketKey]*roaring.Bitmap, len(buckets)),
			handleBuckets: make(map[Handle][]bucketKey),
		}
		for k, raw := range buckets {
			bm := roaring.New()
			// UnmarshalBinary (not FromBuffer) is deliberate: FromBuffer's
			// own doc warns it can alias its input buffer (copy-on-write
			// containers referencing raw's backing array), and raw here
			// came from a decoded snapshot buffer this method does not
			// control the lifetime of -- UnmarshalBinary always copies.
			if err := bm.UnmarshalBinary(raw); err != nil {
				return fmt.Errorf("bloomgateway: importing tenant %q bucket %d: %w", id, k, err)
			}
			t.buckets[k] = bm
			for _, v := range bm.ToArray() {
				h := Handle(v)
				t.handleBuckets[h] = append(t.handleBuckets[h], k)
			}
		}
		tenants[id] = t
	}

	ts.mu.Lock()
	ts.tenants = tenants
	ts.mu.Unlock()
	return nil
}
