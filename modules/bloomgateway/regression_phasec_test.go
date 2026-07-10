package bloomgateway

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

// These tests guard the correctness fixes made in response to the Phase C
// adversarial design review. Each is written to FAIL against the pre-fix code
// and pass against the fix, so a future regression is caught.

// TestDirectory_CompactLeafRemovesOnlyUnkept covers the new atomic
// read-modify-write primitive that replaced the sweep's racy
// CloneLeaf-then-Swap. It removes exactly the unkept handles from a complete
// leaf, and refuses to touch nil/constructing slots.
func TestDirectory_CompactLeafRemovesOnlyUnkept(t *testing.T) {
	dir := NewDirectory(testD)
	const idx = uint32(3)
	completeLeaf(t, dir, idx)

	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	require.True(t, dir.InsertLive(idx, 2, Handle(2)))
	require.True(t, dir.InsertLive(idx, 3, Handle(3)))

	keep := func(h Handle) bool { return h != 2 }
	visited, removed, compacted := dir.CompactLeaf(idx, keep)
	require.True(t, compacted)
	require.Equal(t, 3, visited)
	require.Equal(t, 1, removed)

	h1, ok := dir.Lookup(idx, 1)
	require.True(t, ok)
	require.Equal(t, []Handle{1}, h1)
	h2, _ := dir.Lookup(idx, 2)
	require.Empty(t, h2, "the unkept handle must be gone")
	h3, _ := dir.Lookup(idx, 3)
	require.Equal(t, []Handle{3}, h3)

	t.Run("nil slot is not compacted", func(t *testing.T) {
		_, _, compacted := dir.CompactLeaf(9, keep)
		require.False(t, compacted)
	})
	t.Run("constructing slot is not compacted", func(t *testing.T) {
		const cidx = uint32(4)
		_, started := dir.BeginConstructing(cidx)
		require.True(t, started)
		_, _, compacted := dir.CompactLeaf(cidx, keep)
		require.False(t, compacted, "only complete leaves may be compacted")
	})
}

// TestDirectory_AbandonRevertsOnlyConstructing covers the rollback primitive
// the reconstruction batch uses on failure: it reverts a constructing leaf to
// nil, and is a no-op on any other state (so it can never clobber a complete
// leaf some other episode finished).
func TestDirectory_AbandonRevertsOnlyConstructing(t *testing.T) {
	dir := NewDirectory(testD)

	const constructingIdx = uint32(1)
	_, started := dir.BeginConstructing(constructingIdx)
	require.True(t, started)
	require.True(t, dir.Abandon(constructingIdx))
	require.Equal(t, LeafNil, dir.State(constructingIdx))

	const completeIdx = uint32(2)
	completeLeaf(t, dir, completeIdx)
	require.False(t, dir.Abandon(completeIdx), "a complete leaf must not be abandonable")
	require.Equal(t, LeafComplete, dir.State(completeIdx))

	require.False(t, dir.Abandon(5), "a nil slot abandon is a no-op")
}

// TestSweep_CancelledPassDoesNotReclaimTombstones guards the fix for the
// Phase C finding that a context-cancelled (shutdown) sweep pass, whose
// directory walk stopped early, still ran the reclamation loop — reclaiming a
// tombstone without having confirmed its leaf entries were removed, reopening
// the resurrection-by-replay hole. A pass whose walk did not complete must
// reclaim nothing.
func TestSweep_CancelledPassDoesNotReclaimTombstones(t *testing.T) {
	const tenantID = "tenant-a"
	tr := testTimeRange()
	// ReplayHorizon 0: the tombstone is immediately reclamation-eligible on
	// age, so ONLY the incomplete-walk guard can hold reclamation back.
	sweeper, dir, reg, _ := newTestSweeper(t, time.Hour, 0)
	const leafIdx = uint32(3)
	completeLeaf(t, dir, leafIdx)

	uuid := backend.NewUUID()
	blk, _ := reg.GetOrCreate(uuid, tenantID, tr.start, tr.end)
	require.NoError(t, reg.CommitLive(uuid, false))
	require.True(t, dir.InsertLive(leafIdx, 42, blk.Handle))
	require.NoError(t, reg.MarkDeleted(uuid))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: the walk aborts at the first non-nil leaf

	stats := sweeper.Pass(ctx)
	require.Zero(t, stats.TombstonesReclaimed, "a cancelled/incomplete pass must reclaim nothing")
	_, ok := reg.LookupUUID(uuid)
	require.True(t, ok, "the tombstone must remain until a complete pass reclaims it")
}

// TestSweep_ConcurrentInsertSurvivesCompaction is the regression guard for the
// headline Phase C must-fix: the sweep's old CloneLeaf-then-Swap dropped live
// entries inserted concurrently in the window between the clone and the swap
// (a false negative). One goroutine hammers Pass (with a steady supply of
// deleted-block entries to remove, so every pass exercises the compaction
// path) while another inserts many distinct live entries into the same leaf.
// Every live entry must survive. Run under -race.
func TestSweep_ConcurrentInsertSurvivesCompaction(t *testing.T) {
	const tenantID = "tenant-a"
	tr := testTimeRange()
	sweeper, dir, reg, tenants := newTestSweeper(t, time.Hour, time.Hour)
	const leafIdx = uint32(3)
	completeLeaf(t, dir, leafIdx)

	// Deleted-block handles whose entries the sweeper removes each pass; the
	// inserter re-adds them periodically so removals (and thus the
	// compaction/Swap path) keep happening throughout the run.
	const nDeleted = 8
	deletedHandles := make([]Handle, nDeleted)
	deletedFPs := make([]uint16, nDeleted)
	for i := range nDeleted {
		uuid := backend.NewUUID()
		blk, _ := reg.GetOrCreate(uuid, tenantID, tr.start, tr.end)
		require.NoError(t, reg.CommitLive(uuid, false))
		require.NoError(t, reg.MarkDeleted(uuid))
		deletedHandles[i] = blk.Handle
		deletedFPs[i] = uint16(1000 + i)
	}

	const nLive = 1500
	liveHandles := make([]Handle, nLive)
	liveFPs := make([]uint16, nLive)
	for i := range nLive {
		liveHandles[i] = liveBlock(t, reg, tenants, 100000+i, tr.start, tr.end)
		liveFPs[i] = uint16(20000 + i) // 20000..21499, distinct from deleted FPs, within uint16
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				sweeper.Pass(context.Background())
			}
		}
	}()

	for i := range nLive {
		dir.InsertLive(leafIdx, liveFPs[i], liveHandles[i])
		if i%16 == 0 {
			for j := range nDeleted {
				dir.InsertLive(leafIdx, deletedFPs[j], deletedHandles[j])
			}
		}
	}
	close(done)
	wg.Wait()

	for i := range nLive {
		handles, ok := dir.Lookup(leafIdx, liveFPs[i])
		require.True(t, ok)
		found := false
		for _, h := range handles {
			if h == liveHandles[i] {
				found = true
				break
			}
		}
		require.Truef(t, found, "live entry %d (fp=%d, handle=%d) was lost during concurrent compaction", i, liveFPs[i], liveHandles[i])
	}
}
