package bloomgateway

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hourMark returns the exact start-of-bucket instant for bucket n, measured
// directly from the Unix epoch (NOT from any calendar date) — so a test's
// literal bucketKey expectations (0, 1, 2, ...) line up exactly with
// bucketKeyLowerBound/UpperBound's own epoch-relative arithmetic, with no
// offset math needed in the tests themselves.
func hourMark(n int) time.Time {
	return time.Unix(int64(n)*int64(bucketDuration/time.Second), 0).UTC()
}

func TestFloorDivCeilDiv(t *testing.T) {
	tests := []struct {
		a, b      int64
		wantFloor int64
		wantCeil  int64
	}{
		{10, 3, 3, 4},
		{9, 3, 3, 3},
		{-10, 3, -4, -3},
		{-9, 3, -3, -3},
		{0, 3, 0, 0},
		{10, -3, -4, -3},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d div %d", tt.a, tt.b), func(t *testing.T) {
			assert.Equal(t, tt.wantFloor, floorDiv(tt.a, tt.b), "floorDiv")
			assert.Equal(t, tt.wantCeil, ceilDiv(tt.a, tt.b), "ceilDiv")
		})
	}
}

// TestBucketRange pins bucketRange's boundary-inclusive semantics directly
// at the unit level (both AddBlock and Window are built on it) — see its
// doc comment in tenant.go for the rationale.
func TestBucketRange(t *testing.T) {
	tests := []struct {
		name                string
		start, end          time.Time
		wantFirst, wantLast bucketKey
	}{
		{"strictly inside one bucket, no boundary touch", hourMark(5).Add(10 * time.Minute), hourMark(5).Add(50 * time.Minute), 5, 5},
		{"spans 3 consecutive buckets", hourMark(0).Add(30 * time.Minute), hourMark(2).Add(30 * time.Minute), 0, 2},
		{"zero-width, exactly on a boundary", hourMark(5), hourMark(5), 4, 5},
		{"start on boundary, end later in the same bucket", hourMark(5), hourMark(5).Add(20 * time.Minute), 4, 5},
		{"end on boundary, start earlier in the same bucket", hourMark(5).Add(40 * time.Minute), hourMark(6), 5, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last := bucketRange(tt.start, tt.end)
			assert.Equal(t, tt.wantFirst, first, "first")
			assert.Equal(t, tt.wantLast, last, "last")
		})
	}
}

// TestTenantSet_MembershipAcrossOverlappingBuckets is the named test for
// invariant #8 (§7): a block is a member of EVERY bucket its time range
// overlaps, not just one.
func TestTenantSet_MembershipAcrossOverlappingBuckets(t *testing.T) {
	ts := NewTenantSet()
	h := Handle(1)

	start := hourMark(0).Add(30 * time.Minute)
	end := hourMark(2).Add(30 * time.Minute) // spans buckets 0, 1, 2
	ts.AddBlock("tenant-a", h, start, end)

	tn := ts.tenants["tenant-a"]
	require.NotNil(t, tn)

	for _, n := range []int{0, 1, 2} {
		bm, ok := tn.buckets[bucketKey(n)]
		require.True(t, ok, "bucket %d must exist", n)
		assert.True(t, bm.Contains(uint32(h)), "bucket %d must contain h", n)
	}
	_, ok := tn.buckets[bucketKey(3)]
	assert.False(t, ok, "bucket 3 must never have been created")
	assert.ElementsMatch(t, []bucketKey{0, 1, 2}, tn.handleBuckets[h], "reverse index must list exactly the 3 overlapping buckets")

	// Confirmed again through the query path's own accessor.
	for _, n := range []int{0, 1, 2} {
		bm := ts.Window("tenant-a", hourMark(n).Add(10*time.Minute), hourMark(n).Add(20*time.Minute))
		assert.True(t, bm.Contains(uint32(h)), "Window for bucket %d must contain h", n)
	}
	bm := ts.Window("tenant-a", hourMark(3).Add(10*time.Minute), hourMark(3).Add(20*time.Minute))
	assert.False(t, bm.Contains(uint32(h)), "Window for bucket 3 must not contain h")
}

// TestTenantSet_BoundaryTimestampMembership is the boundary-timestamp edge
// case WP12's plan calls out explicitly: a timestamp landing exactly on a
// bucket boundary is treated as touching BOTH adjacent buckets (deliberate
// over-inclusion; see bucketKeyLowerBound's doc comment in tenant.go).
func TestTenantSet_BoundaryTimestampMembership(t *testing.T) {
	ts := NewTenantSet()

	t.Run("zero-width block exactly at an hour boundary is a member of both adjacent buckets", func(t *testing.T) {
		h := Handle(10)
		boundary := hourMark(5)
		ts.AddBlock("tenant-a", h, boundary, boundary)

		tn := ts.tenants["tenant-a"]
		assert.ElementsMatch(t, []bucketKey{4, 5}, tn.handleBuckets[h])
	})

	t.Run("start exactly on a boundary, end later, is also a member of the previous bucket", func(t *testing.T) {
		h := Handle(11)
		start := hourMark(5)
		end := hourMark(5).Add(20 * time.Minute)
		ts.AddBlock("tenant-a", h, start, end)

		tn := ts.tenants["tenant-a"]
		assert.ElementsMatch(t, []bucketKey{4, 5}, tn.handleBuckets[h])
	})

	t.Run("end exactly on a boundary, start earlier, is also a member of the next bucket", func(t *testing.T) {
		h := Handle(12)
		start := hourMark(5).Add(40 * time.Minute)
		end := hourMark(6)
		ts.AddBlock("tenant-a", h, start, end)

		tn := ts.tenants["tenant-a"]
		assert.ElementsMatch(t, []bucketKey{5, 6}, tn.handleBuckets[h])
	})

	t.Run("a range strictly inside one bucket, no boundary touch, is a member of only that bucket", func(t *testing.T) {
		h := Handle(13)
		start := hourMark(5).Add(10 * time.Minute)
		end := hourMark(5).Add(50 * time.Minute)
		ts.AddBlock("tenant-a", h, start, end)

		tn := ts.tenants["tenant-a"]
		assert.ElementsMatch(t, []bucketKey{5}, tn.handleBuckets[h])
	})
}

func TestTenantSet_AddBlock_LazilyCreatesTenant(t *testing.T) {
	ts := NewTenantSet()
	ts.mu.RLock()
	_, ok := ts.tenants["tenant-a"]
	ts.mu.RUnlock()
	require.False(t, ok)

	ts.AddBlock("tenant-a", Handle(1), hourMark(0), hourMark(0).Add(time.Minute))

	ts.mu.RLock()
	_, ok = ts.tenants["tenant-a"]
	ts.mu.RUnlock()
	assert.True(t, ok)
}

func TestTenantSet_AddBlock_SameHandleTwiceIsNoOp(t *testing.T) {
	ts := NewTenantSet()
	h := Handle(1)
	ts.AddBlock("tenant-a", h, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))
	// A second AddBlock for the same handle, even with a totally different
	// range, must be ignored — AddBlock is only ever called once per
	// handle's single Pending -> Live transition in real use; this guards
	// that invariant defensively.
	ts.AddBlock("tenant-a", h, hourMark(5).Add(10*time.Minute), hourMark(5).Add(20*time.Minute))

	tn := ts.tenants["tenant-a"]
	assert.ElementsMatch(t, []bucketKey{0}, tn.handleBuckets[h])
}

func TestTenantSet_RemoveBlock_FullyReversesAddBlock(t *testing.T) {
	ts := NewTenantSet()
	h := Handle(1)
	start := hourMark(0).Add(30 * time.Minute)
	end := hourMark(2).Add(30 * time.Minute) // buckets 0, 1, 2

	ts.AddBlock("tenant-a", h, start, end)
	tn := ts.tenants["tenant-a"]
	require.Len(t, tn.handleBuckets[h], 3)

	ts.RemoveBlock("tenant-a", h)

	_, exists := tn.handleBuckets[h]
	assert.False(t, exists, "reverse index entry must be fully removed")
	for _, n := range []int{0, 1, 2} {
		bm, ok := tn.buckets[bucketKey(n)]
		require.True(t, ok, "bucket %d itself must survive (only h's membership is removed)", n)
		assert.False(t, bm.Contains(uint32(h)), "bucket %d must no longer contain h", n)
	}
}

func TestTenantSet_RemoveBlock_UnknownTenantOrHandleIsNoOp(t *testing.T) {
	ts := NewTenantSet()
	ts.RemoveBlock("no-such-tenant", Handle(1)) // must not panic

	ts.AddBlock("tenant-a", Handle(1), hourMark(0), hourMark(0).Add(time.Minute))
	ts.RemoveBlock("tenant-a", Handle(999)) // unknown handle: must not panic or disturb Handle(1)

	tn := ts.tenants["tenant-a"]
	assert.Contains(t, tn.handleBuckets, Handle(1))
}

func TestTenantSet_Window_SubRangeUnionsOnlyOverlappingBuckets(t *testing.T) {
	ts := NewTenantSet()
	h1, h2, h3 := Handle(1), Handle(2), Handle(3)
	ts.AddBlock("tenant-a", h1, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute)) // bucket 0
	ts.AddBlock("tenant-a", h2, hourMark(1).Add(10*time.Minute), hourMark(1).Add(20*time.Minute)) // bucket 1
	ts.AddBlock("tenant-a", h3, hourMark(5).Add(10*time.Minute), hourMark(5).Add(20*time.Minute)) // bucket 5, out of range below

	bm := ts.Window("tenant-a", hourMark(0).Add(10*time.Minute), hourMark(1).Add(20*time.Minute))
	assert.ElementsMatch(t, []uint32{uint32(h1), uint32(h2)}, bm.ToArray())
}

func TestTenantSet_Window_UnscopedUnionsEverything(t *testing.T) {
	ts := NewTenantSet()
	h1, h2 := Handle(1), Handle(2)
	ts.AddBlock("tenant-a", h1, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))
	ts.AddBlock("tenant-a", h2, hourMark(100).Add(10*time.Minute), hourMark(100).Add(20*time.Minute))

	bm := ts.Window("tenant-a", time.Time{}, time.Time{})
	assert.ElementsMatch(t, []uint32{uint32(h1), uint32(h2)}, bm.ToArray())
}

func TestTenantSet_Window_OneSidedUnboundedTreatsThatSideAsUnbounded(t *testing.T) {
	ts := NewTenantSet()
	h1, h2, h3 := Handle(1), Handle(2), Handle(3)
	ts.AddBlock("tenant-a", h1, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))
	ts.AddBlock("tenant-a", h2, hourMark(5).Add(10*time.Minute), hourMark(5).Add(20*time.Minute))
	ts.AddBlock("tenant-a", h3, hourMark(10).Add(10*time.Minute), hourMark(10).Add(20*time.Minute))

	// Unbounded start, bounded end: everything up to and including bucket 5.
	bm := ts.Window("tenant-a", time.Time{}, hourMark(5).Add(20*time.Minute))
	assert.ElementsMatch(t, []uint32{uint32(h1), uint32(h2)}, bm.ToArray())

	// Bounded start, unbounded end: everything from bucket 5 onward.
	bm = ts.Window("tenant-a", hourMark(5).Add(10*time.Minute), time.Time{})
	assert.ElementsMatch(t, []uint32{uint32(h2), uint32(h3)}, bm.ToArray())
}

func TestTenantSet_Window_UnknownTenantReturnsEmptyNotNil(t *testing.T) {
	ts := NewTenantSet()
	bm := ts.Window("no-such-tenant", time.Time{}, time.Time{})
	require.NotNil(t, bm)
	assert.True(t, bm.IsEmpty())
}

func TestTenantSet_Window_NoOverlappingBucketsReturnsEmptyNotNil(t *testing.T) {
	ts := NewTenantSet()
	ts.AddBlock("tenant-a", Handle(1), hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))

	bm := ts.Window("tenant-a", hourMark(50), hourMark(51))
	require.NotNil(t, bm)
	assert.True(t, bm.IsEmpty())
}

func TestTenantSet_DropEmptyBuckets_RemovesOnlyEmptyBuckets(t *testing.T) {
	ts := NewTenantSet()
	h1, h2 := Handle(1), Handle(2)
	ts.AddBlock("tenant-a", h1, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute)) // bucket 0
	ts.AddBlock("tenant-a", h2, hourMark(1).Add(10*time.Minute), hourMark(1).Add(20*time.Minute)) // bucket 1

	ts.RemoveBlock("tenant-a", h1) // bucket 0 now empty; bucket 1 still holds h2

	ts.DropEmptyBuckets("tenant-a")

	tn := ts.tenants["tenant-a"]
	_, ok := tn.buckets[bucketKey(0)]
	assert.False(t, ok, "empty bucket 0 must be dropped")
	bm, ok := tn.buckets[bucketKey(1)]
	require.True(t, ok, "non-empty bucket 1 must survive")
	assert.True(t, bm.Contains(uint32(h2)))
}

func TestTenantSet_DropEmptyBuckets_UnknownTenantIsNoOp(_ *testing.T) {
	ts := NewTenantSet()
	ts.DropEmptyBuckets("no-such-tenant") // must not panic
}

func TestTenantSet_DropTenant(t *testing.T) {
	ts := NewTenantSet()
	ts.AddBlock("tenant-a", Handle(1), hourMark(0), hourMark(0).Add(time.Minute))

	ts.DropTenant("tenant-a")

	bm := ts.Window("tenant-a", time.Time{}, time.Time{})
	assert.True(t, bm.IsEmpty())

	ts.mu.RLock()
	_, ok := ts.tenants["tenant-a"]
	ts.mu.RUnlock()
	assert.False(t, ok)
}

func TestTenantSet_DropTenant_UnknownTenantIsNoOp(_ *testing.T) {
	ts := NewTenantSet()
	ts.DropTenant("no-such-tenant") // must not panic
}

// TestTenantSet_MultipleTenantsAreIsolated guards § Multi-tenant cells'
// isolation claim at the TenantSet level: tenant X's Window must never
// include tenant Y's handles, even when both use overlapping buckets.
func TestTenantSet_MultipleTenantsAreIsolated(t *testing.T) {
	ts := NewTenantSet()
	hX, hY := Handle(1), Handle(2)
	ts.AddBlock("tenant-x", hX, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))
	ts.AddBlock("tenant-y", hY, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))

	bmX := ts.Window("tenant-x", time.Time{}, time.Time{})
	bmY := ts.Window("tenant-y", time.Time{}, time.Time{})

	assert.Equal(t, []uint32{uint32(hX)}, bmX.ToArray())
	assert.Equal(t, []uint32{uint32(hY)}, bmY.ToArray())
}

// TestTenantSet_ConcurrentAddRemoveWindow covers WP12's own stated test
// plan item: concurrent AddBlock/RemoveBlock/Window for the same tenant
// under -race.
func TestTenantSet_ConcurrentAddRemoveWindow(t *testing.T) {
	ts := NewTenantSet()
	const n = 200

	var writers sync.WaitGroup
	for i := 0; i < n; i++ {
		i := i
		writers.Add(1)
		go func() {
			defer writers.Done()
			h := Handle(i + 1)
			bucket := i % 20
			ts.AddBlock("tenant-a", h, hourMark(bucket).Add(10*time.Minute), hourMark(bucket).Add(20*time.Minute))
		}()
	}

	stop := make(chan struct{})
	var readers sync.WaitGroup
	for r := 0; r < 4; r++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
					ts.Window("tenant-a", time.Time{}, time.Time{})
					ts.Window("tenant-a", hourMark(0), hourMark(5))
				}
			}
		}()
	}

	writers.Wait()

	// Concurrently remove every odd handle while readers keep hammering
	// Window.
	var removers sync.WaitGroup
	for i := 0; i < n; i += 2 {
		i := i
		removers.Add(1)
		go func() {
			defer removers.Done()
			ts.RemoveBlock("tenant-a", Handle(i+1))
		}()
	}
	removers.Wait()

	close(stop)
	readers.Wait()

	// Removed handles are i+1 for even i (0, 2, 4, ...) = odd handle values
	// 1, 3, 5, .... Surviving handles are therefore the even ones.
	got := ts.Window("tenant-a", time.Time{}, time.Time{}).ToArray()
	var want []uint32
	for h := 2; h <= n; h += 2 {
		want = append(want, uint32(h))
	}
	assert.ElementsMatch(t, want, got)
}

// TestTenantSet_ExportImport_RoundTripsWindowContents is the WP12 addendum
// (see TenantSetSnapshot's doc comment) test: Export then Import into a
// FRESH TenantSet must reproduce every tenant's Window contents exactly.
// Deliberately asserted via Window (a functional/behavioral check) rather
// than deep-equating the two TenantSets' internal handleBuckets fields
// directly: Import rebuilds the reverse index by iterating a decoded
// bitmap's ToArray() per bucket, so a handle spanning multiple buckets can
// legitimately come back with its handleBuckets entry in a different
// (map-iteration-derived) order than AddBlock's own ascending-bucketKey
// construction -- functionally identical A_T membership, not necessarily
// byte-identical internal slice order.
func TestTenantSet_ExportImport_RoundTripsWindowContents(t *testing.T) {
	original := NewTenantSet()
	original.AddBlock("tenant-a", Handle(1), hourMark(0).Add(10*time.Minute), hourMark(2).Add(10*time.Minute)) // spans buckets 0,1,2
	original.AddBlock("tenant-a", Handle(2), hourMark(5).Add(10*time.Minute), hourMark(5).Add(20*time.Minute))
	original.AddBlock("tenant-b", Handle(3), hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))

	snap, err := original.Export()
	require.NoError(t, err)

	restored := NewTenantSet()
	require.NoError(t, restored.Import(snap))

	for _, tenantID := range []string{"tenant-a", "tenant-b"} {
		wantUnscoped := original.Window(tenantID, time.Time{}, time.Time{}).ToArray()
		gotUnscoped := restored.Window(tenantID, time.Time{}, time.Time{}).ToArray()
		assert.ElementsMatch(t, wantUnscoped, gotUnscoped, "tenant %q unscoped window", tenantID)
	}

	// A scoped window (bucket 5 only) must also round-trip -- confirms the
	// per-bucket membership, not just the union across every bucket.
	wantScoped := original.Window("tenant-a", hourMark(5), hourMark(5).Add(30*time.Minute)).ToArray()
	gotScoped := restored.Window("tenant-a", hourMark(5), hourMark(5).Add(30*time.Minute)).ToArray()
	assert.Equal(t, []uint32{uint32(2)}, wantScoped, "test setup sanity check")
	assert.ElementsMatch(t, wantScoped, gotScoped)
}

// TestTenantSet_Export_IndependentOfLiveState confirms Export's own
// documented guarantee: mutating the live TenantSet after Export must
// never retroactively change the already-taken snapshot.
func TestTenantSet_Export_IndependentOfLiveState(t *testing.T) {
	ts := NewTenantSet()
	ts.AddBlock("tenant-a", Handle(1), hourMark(0), hourMark(0).Add(time.Minute))

	snap, err := ts.Export()
	require.NoError(t, err)

	ts.AddBlock("tenant-a", Handle(2), hourMark(0), hourMark(0).Add(time.Minute))
	ts.RemoveBlock("tenant-a", Handle(1))

	restored := NewTenantSet()
	require.NoError(t, restored.Import(snap))
	got := restored.Window("tenant-a", time.Time{}, time.Time{}).ToArray()
	assert.Equal(t, []uint32{uint32(1)}, got, "the snapshot must reflect state as of Export, not later mutations")
}

// TestTenantSet_Export_EmptyTenantSet confirms Export/Import handle the
// zero-tenant case without panicking, round-tripping to an equally empty
// TenantSet.
func TestTenantSet_Export_EmptyTenantSet(t *testing.T) {
	ts := NewTenantSet()
	snap, err := ts.Export()
	require.NoError(t, err)
	assert.Empty(t, snap.Buckets)

	restored := NewTenantSet()
	require.NoError(t, restored.Import(snap))
	got := restored.Window("any-tenant", time.Time{}, time.Time{})
	assert.True(t, got.IsEmpty())
}
