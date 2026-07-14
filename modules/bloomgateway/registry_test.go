package bloomgateway

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

// testUUID returns a deterministic, distinct backend.UUID for test index n
// — human-legible in failure output, and stable across runs (unlike
// backend.NewUUID(), which is random).
func testUUID(t testing.TB, n int) backend.UUID {
	t.Helper()
	return backend.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", n))
}

// TestRegistry_HandlesNeverReused is the named test for DESIGN.md §
// Representation notes' "never reclaimed" guarantee: a handle, once
// allocated, must never be handed out again for the lifetime of the
// Registry — including after the block it named has been deleted AND
// reclaimed.
func TestRegistry_HandlesNeverReused(t *testing.T) {
	r := NewRegistry()

	seen := map[Handle]bool{}
	allocate := func(n int) {
		b, isNew := r.GetOrCreate(testUUID(t, n), "tenant-a", time.Now(), time.Now())
		require.True(t, isNew)
		require.False(t, seen[b.Handle], "handle %d allocated twice", b.Handle)
		seen[b.Handle] = true
	}

	for n := 0; n < 5; n++ {
		allocate(n)
	}

	// Delete and reclaim the first three.
	for n := 0; n < 3; n++ {
		u := testUUID(t, n)
		require.NoError(t, r.MarkDeleted(u))
		r.Reclaim(u)
		_, ok := r.LookupUUID(u)
		require.False(t, ok, "uuid %d must be gone from the registry after Reclaim", n)
	}

	// Allocate many more blocks; none of their handles may collide with
	// anything ever allocated before, including the reclaimed ones.
	for n := 5; n < 50; n++ {
		allocate(n)
	}

	assert.NotContains(t, seen, InvalidHandle, "InvalidHandle must never be allocated")
}

// TestRegistry_DeleteIsTerminal is the named test for invariant #4 (§7):
// once a block is BlockDeleted, no CommitLive call — in any variant, any
// number of times — may resurrect it, and MarkDeleted redelivery is
// idempotent.
func TestRegistry_DeleteIsTerminal(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)

	b, isNew := r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
	require.True(t, isNew)
	require.NoError(t, r.CommitLive(uuid, false))
	require.Equal(t, BlockLive, b.State)

	require.NoError(t, r.MarkDeleted(uuid))
	require.Equal(t, BlockDeleted, b.State)
	deletedAt := b.DeletedAt
	require.False(t, deletedAt.IsZero())

	require.NoError(t, r.CommitLive(uuid, false))
	assert.Equal(t, BlockDeleted, b.State, "CommitLive(unsupportedEncoding=false) after MarkDeleted must be a no-op")

	require.NoError(t, r.CommitLive(uuid, true))
	assert.Equal(t, BlockDeleted, b.State, "CommitLive(unsupportedEncoding=true) after MarkDeleted must also be a no-op")

	// Redelivered Delete is idempotent and must not disturb the original
	// DeletedAt (the sweep's replay-horizon check needs it stable).
	require.NoError(t, r.MarkDeleted(uuid))
	assert.Equal(t, BlockDeleted, b.State)
	assert.Equal(t, deletedAt, b.DeletedAt)
}

// TestRegistry_StateTransitionTable is the transition-table test covering
// AMENDMENT A1's edge: Live -> LiveUnsupportedEncoding is now legal
// (demotion), and LiveUnsupportedEncoding -> Live remains illegal.
func TestRegistry_StateTransitionTable(t *testing.T) {
	tests := []struct {
		name                string
		from                BlockState
		unsupportedEncoding bool
		wantState           BlockState
		wantErr             bool
	}{
		{"pending to live", BlockPending, false, BlockLive, false},
		{"pending to live-unsupported-encoding", BlockPending, true, BlockLiveUnsupportedEncoding, false},
		{"live to live is a no-op", BlockLive, false, BlockLive, false},
		{"live to live-unsupported-encoding is a legal demotion (AMENDMENT A1)", BlockLive, true, BlockLiveUnsupportedEncoding, false},
		{"live-unsupported-encoding to live-unsupported-encoding is a no-op", BlockLiveUnsupportedEncoding, true, BlockLiveUnsupportedEncoding, false},
		{"live-unsupported-encoding to live is ILLEGAL (AMENDMENT A1)", BlockLiveUnsupportedEncoding, false, BlockLiveUnsupportedEncoding, true},
		{"deleted to live is a no-op (tombstone terminal)", BlockDeleted, false, BlockDeleted, false},
		{"deleted to live-unsupported-encoding is a no-op (tombstone terminal)", BlockDeleted, true, BlockDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			uuid := testUUID(t, 1)
			b, _ := r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
			b.State = tt.from // force the starting state directly; same-package test

			err := r.CommitLive(uuid, tt.unsupportedEncoding)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantState, b.State)
		})
	}
}

func TestRegistry_CommitLive_UnknownUUIDErrors(t *testing.T) {
	r := NewRegistry()
	err := r.CommitLive(testUUID(t, 1), false)
	assert.Error(t, err)
}

func TestRegistry_MarkDeleted_UnknownUUIDIsNoOp(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 99)

	err := r.MarkDeleted(uuid)
	require.NoError(t, err)

	_, ok := r.LookupUUID(uuid)
	assert.False(t, ok, "MarkDeleted on an unknown uuid must not create a registry entry")
}

func TestRegistry_GetOrCreate_RecordsStartEndOnlyWhenActuallyNew(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	start1, end1 := time.Unix(1000, 0), time.Unix(2000, 0)
	start2, end2 := time.Unix(9000, 0), time.Unix(9999, 0)

	b1, isNew1 := r.GetOrCreate(uuid, "tenant-a", start1, end1)
	require.True(t, isNew1)
	assert.Equal(t, start1, b1.StartTime)
	assert.Equal(t, end1, b1.EndTime)

	// A second chunk of the same block arriving with (irrelevant, or even
	// wrong) start/end must not perturb the recorded values — only the
	// creating call's start/end are ever recorded.
	b2, isNew2 := r.GetOrCreate(uuid, "tenant-a", start2, end2)
	assert.False(t, isNew2)
	assert.Same(t, b1, b2)
	assert.Equal(t, start1, b2.StartTime)
	assert.Equal(t, end1, b2.EndTime)
}

func TestRegistry_LookupHandle_LookupUUID(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	b, _ := r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())

	gotByHandle, ok := r.LookupHandle(b.Handle)
	require.True(t, ok)
	assert.Same(t, b, gotByHandle)

	gotByUUID, ok := r.LookupUUID(uuid)
	require.True(t, ok)
	assert.Same(t, b, gotByUUID)

	_, ok = r.LookupHandle(Handle(999999))
	assert.False(t, ok)
	_, ok = r.LookupUUID(testUUID(t, 999))
	assert.False(t, ok)
}

func TestRegistry_Range_VisitsEveryBlockRegardlessOfState(t *testing.T) {
	r := NewRegistry()
	u0, u1, u2 := testUUID(t, 0), testUUID(t, 1), testUUID(t, 2)
	r.GetOrCreate(u0, "tenant-a", time.Now(), time.Now()) // stays Pending
	r.GetOrCreate(u1, "tenant-a", time.Now(), time.Now())
	require.NoError(t, r.CommitLive(u1, false)) // Live
	r.GetOrCreate(u2, "tenant-a", time.Now(), time.Now())
	require.NoError(t, r.CommitLive(u2, false))
	require.NoError(t, r.MarkDeleted(u2)) // Deleted

	visited := map[backend.UUID]BlockState{}
	r.Range(func(b *Block) bool {
		visited[b.UUID] = b.State
		return true
	})

	assert.Equal(t, map[backend.UUID]BlockState{
		u0: BlockPending,
		u1: BlockLive,
		u2: BlockDeleted,
	}, visited)
}

func TestRegistry_Range_EarlyStop(t *testing.T) {
	r := NewRegistry()
	for n := 0; n < 5; n++ {
		r.GetOrCreate(testUUID(t, n), "tenant-a", time.Now(), time.Now())
	}

	count := 0
	r.Range(func(*Block) bool {
		count++
		return count < 2
	})
	assert.Equal(t, 2, count)
}

func TestRegistry_Reclaim_UnknownUUIDIsNoOp(t *testing.T) {
	r := NewRegistry()
	r.Reclaim(testUUID(t, 1)) // must not panic
}

// TestRegistry_Reclaim_ConcurrentWithLookup covers WP11's own stated risk:
// Reclaim racing a concurrent LookupUUID/LookupHandle for the same block
// must not panic, and must never leave the registry with a "half-removed"
// block — present in one index but not the other. Run under -race.
//
// The atomicity check reads both indexes under a single RLock acquisition
// (white-box, same-package field access) rather than via two separate
// LookupUUID/LookupHandle calls: two independently-locked calls each see a
// consistent snapshot of their OWN index, but Reclaim could interleave
// between the two calls, making a pair of independently-locked reads
// disagree even though the registry itself was never actually
// inconsistent at any single instant — that would be a false positive in
// the test, not a bug in Reclaim.
func TestRegistry_Reclaim_ConcurrentWithLookup(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	b, _ := r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
	h := b.Handle
	require.NoError(t, r.MarkDeleted(uuid))

	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			r.mu.RLock()
			_, okUUID := r.byUUID[uuid]
			_, okHandle := r.byHandle[h]
			r.mu.RUnlock()
			assert.Equal(t, okUUID, okHandle, "uuid and handle indexes must always agree at any single instant: never half-removed")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			r.Reclaim(uuid) // idempotent; every call after the first is a no-op
		}
		close(stop)
	}()

	wg.Wait()

	_, ok := r.LookupUUID(uuid)
	assert.False(t, ok)
	_, ok = r.LookupHandle(h)
	assert.False(t, ok)
}

// TestRegistry_GetOrCreate_ConcurrentSameUUID is the idempotent-creation
// guard directly relevant to invariant #5 (§7): concurrent GetOrCreate
// calls for the same uuid from many goroutines must yield exactly one
// Block/handle, matching AMENDMENT A2's "multiple chunks of the same block
// can arrive on different workers concurrently."
func TestRegistry_GetOrCreate_ConcurrentSameUUID(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)

	const n = 100
	results := make([]*Block, n)
	isNewFlags := make([]bool, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, isNew := r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
			results[i] = b
			isNewFlags[i] = isNew
		}()
	}
	wg.Wait()

	newCount := 0
	for i := 0; i < n; i++ {
		assert.Same(t, results[0], results[i], "every concurrent GetOrCreate for the same uuid must return the identical *Block")
		if isNewFlags[i] {
			newCount++
		}
	}
	assert.Equal(t, 1, newCount, "exactly one caller must observe isNew=true")
}

func TestRegistry_ResolveHandles(t *testing.T) {
	r := NewRegistry()
	u1, u2 := testUUID(t, 1), testUUID(t, 2)
	b1, _ := r.GetOrCreate(u1, "tenant-a", time.Now(), time.Now())
	b2, _ := r.GetOrCreate(u2, "tenant-a", time.Now(), time.Now())

	got := r.ResolveHandles([]Handle{b1.Handle, b2.Handle, Handle(999999)})
	require.Len(t, got, 3)
	assert.Equal(t, u1, got[0])
	assert.Equal(t, u2, got[1])
	assert.Equal(t, backend.UUID{}, got[2], "an unregistered handle resolves to the zero UUID")
}

func TestRegistry_ResolveHandles_Empty(t *testing.T) {
	r := NewRegistry()
	got := r.ResolveHandles(nil)
	assert.Empty(t, got)
}

// TestRegistry_ImportPreservesExactHandlesWithGaps is Import's named
// deliverable (bloomgateway.go's snapshot-restore path): handles must round
// -trip EXACTLY, including a gap left by a block reclaimed before the
// snapshot was taken -- a plain replay of GetOrCreate against a fresh
// Registry cannot reproduce this (its own allocator is sequential and gap-
// free), which is the whole reason Import exists.
func TestRegistry_ImportPreservesExactHandlesWithGaps(t *testing.T) {
	start := time.Now()
	blocks := []Block{
		{UUID: testUUID(t, 1), TenantID: "tenant-a", StartTime: start, EndTime: start, State: BlockLive, Handle: Handle(1)},
		// Handle 2 deliberately missing -- simulates a block reclaimed by
		// the sweep before the snapshot this Import call is restoring was
		// taken.
		{UUID: testUUID(t, 2), TenantID: "tenant-a", StartTime: start, EndTime: start, State: BlockDeleted, Handle: Handle(3), DeletedAt: start},
		{UUID: testUUID(t, 3), TenantID: "tenant-b", StartTime: start, EndTime: start, State: BlockLiveUnsupportedEncoding, Handle: Handle(7)},
	}

	r := NewRegistry()
	require.NoError(t, r.Import(blocks))

	for _, want := range blocks {
		got, ok := r.LookupHandle(want.Handle)
		require.True(t, ok, "handle %d must resolve after import", want.Handle)
		assert.Equal(t, want.UUID, got.UUID)

		byUUID, ok := r.LookupUUID(want.UUID)
		require.True(t, ok)
		assert.Equal(t, want.Handle, byUUID.Handle, "uuid %s must resolve back to its exact imported handle, not a re-allocated one", want.UUID)
	}

	// The allocator must resume strictly above the highest imported handle
	// (7 here), regardless of the gap at 2 -- a subsequent GetOrCreate must
	// never collide with an imported handle.
	newBlock, isNew := r.GetOrCreate(testUUID(t, 4), "tenant-a", start, start)
	require.True(t, isNew)
	assert.Greater(t, uint32(newBlock.Handle), uint32(Handle(7)))
}

func TestRegistry_ImportRejectsInvalidInput(t *testing.T) {
	start := time.Now()

	t.Run("invalid handle", func(t *testing.T) {
		r := NewRegistry()
		err := r.Import([]Block{{UUID: testUUID(t, 1), Handle: InvalidHandle}})
		require.Error(t, err)
	})

	t.Run("duplicate uuid", func(t *testing.T) {
		r := NewRegistry()
		u := testUUID(t, 1)
		err := r.Import([]Block{
			{UUID: u, Handle: Handle(1), StartTime: start, EndTime: start},
			{UUID: u, Handle: Handle(2), StartTime: start, EndTime: start},
		})
		require.Error(t, err)
	})

	t.Run("duplicate handle", func(t *testing.T) {
		r := NewRegistry()
		err := r.Import([]Block{
			{UUID: testUUID(t, 1), Handle: Handle(1), StartTime: start, EndTime: start},
			{UUID: testUUID(t, 2), Handle: Handle(1), StartTime: start, EndTime: start},
		})
		require.Error(t, err)
	})
}

// TestRegistry_ImportReplacesWholesale confirms Import discards any
// pre-existing state rather than merging -- mirroring TenantSet.Import's own
// documented "replaces wholesale" contract.
func TestRegistry_ImportReplacesWholesale(t *testing.T) {
	r := NewRegistry()
	pre, _ := r.GetOrCreate(testUUID(t, 99), "tenant-a", time.Now(), time.Now())
	require.NotEqual(t, Handle(999), pre.Handle, "test sanity: the imported handle below must not collide with pre's")

	require.NoError(t, r.Import([]Block{{UUID: testUUID(t, 1), Handle: Handle(999), StartTime: time.Now(), EndTime: time.Now()}}))

	_, ok := r.LookupUUID(pre.UUID)
	assert.False(t, ok, "Import must discard state present before the call")
	_, ok = r.LookupHandle(pre.Handle)
	assert.False(t, ok, "pre's handle must not survive Import either, since it belonged to a block Import discarded")
}

// TestRegistry_LiveCount_TracksCommitAndDelete is the named test for
// blocks_live's exactly-once transition points (CommitLive's BlockPending
// branch, MarkDeleted's wasLive-gated decrement).
func TestRegistry_LiveCount_TracksCommitAndDelete(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	assert.Zero(t, r.LiveCount())

	r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
	assert.Zero(t, r.LiveCount(), "a still-Pending block must not be counted live")

	require.NoError(t, r.CommitLive(uuid, false))
	assert.EqualValues(t, 1, r.LiveCount())

	require.NoError(t, r.MarkDeleted(uuid))
	assert.Zero(t, r.LiveCount(), "deleting a live block must give back its count")
}

// TestRegistry_LiveCount_DemotionDoesNotDoubleCountOrDrop covers AMENDMENT
// A1's Live -> LiveUnsupportedEncoding demotion edge: it moves BETWEEN two
// states blocksLive already counts as "live" (see the gauge's Help text in
// metrics.go), so it must neither increment (double-count) nor decrement
// (undercount) LiveCount.
func TestRegistry_LiveCount_DemotionDoesNotDoubleCountOrDrop(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())

	require.NoError(t, r.CommitLive(uuid, false))
	require.EqualValues(t, 1, r.LiveCount())

	require.NoError(t, r.CommitLive(uuid, true)) // demotion
	assert.EqualValues(t, 1, r.LiveCount(), "demoting Live -> LiveUnsupportedEncoding must not change the live count")

	require.NoError(t, r.CommitLive(uuid, true)) // no-op redelivery
	assert.EqualValues(t, 1, r.LiveCount())
}

// TestRegistry_LiveCount_PendingDeleteNeverCounted covers the case
// MarkDeleted's wasLive check exists for: a Delete racing ahead of a
// still-chunking (still-Pending) block's remaining chunks must not
// decrement LiveCount, since CommitLive never incremented it in the first
// place -- decrementing anyway would silently underflow the gauge.
func TestRegistry_LiveCount_PendingDeleteNeverCounted(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
	assert.Zero(t, r.LiveCount())

	require.NoError(t, r.MarkDeleted(uuid))
	assert.Zero(t, r.LiveCount(), "deleting a still-Pending block must not decrement an already-zero count")
}

// TestRegistry_LiveCount_RedeliveredMarkDeletedIsIdempotent covers
// MarkDeleted's own guard (b.State == BlockDeleted -> no-op): redelivery of
// the same Delete must decrement LiveCount exactly once, never per call.
func TestRegistry_LiveCount_RedeliveredMarkDeletedIsIdempotent(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
	require.NoError(t, r.CommitLive(uuid, false))

	require.NoError(t, r.MarkDeleted(uuid))
	require.NoError(t, r.MarkDeleted(uuid))
	require.NoError(t, r.MarkDeleted(uuid))
	assert.Zero(t, r.LiveCount(), "redelivered Delete must decrement exactly once, never go negative")
}

// TestRegistry_LiveCount_ImportRecomputesFromImportedState covers Import's
// bulk-replace path: liveCount must reflect exactly the imported blocks'
// State (Live and LiveUnsupportedEncoding count; Pending and Deleted do
// not), since imported blocks bypass CommitLive/MarkDeleted entirely.
func TestRegistry_LiveCount_ImportRecomputesFromImportedState(t *testing.T) {
	r := NewRegistry()
	uuid := testUUID(t, 1)
	r.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
	require.NoError(t, r.CommitLive(uuid, false))
	require.EqualValues(t, 1, r.LiveCount(), "test sanity: pre-import state must not leak into the post-import count")

	start := time.Now()
	require.NoError(t, r.Import([]Block{
		{UUID: testUUID(t, 10), Handle: Handle(1), State: BlockPending, StartTime: start, EndTime: start},
		{UUID: testUUID(t, 11), Handle: Handle(2), State: BlockLive, StartTime: start, EndTime: start},
		{UUID: testUUID(t, 12), Handle: Handle(3), State: BlockLiveUnsupportedEncoding, StartTime: start, EndTime: start},
		{UUID: testUUID(t, 13), Handle: Handle(4), State: BlockDeleted, StartTime: start, EndTime: start, DeletedAt: start},
	}))

	assert.EqualValues(t, 2, r.LiveCount(), "only the imported Live and LiveUnsupportedEncoding blocks count as live")
}
