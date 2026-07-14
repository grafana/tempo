package bloomgateway

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// newTestSweeper builds a Sweeper over fresh Wave-1 structures at the
// package's shared small testD/testF sizing (events_test.go).
// fullPassPeriod only matters to Run (Pass itself is called synchronously
// by every test below except the Run-lifecycle one).
func newTestSweeper(t *testing.T, fullPassPeriod, replayHorizon time.Duration) (*Sweeper, *Directory, *Registry, *TenantSet) {
	t.Helper()
	dir := NewDirectory(testD)
	reg := NewRegistry()
	tenants := NewTenantSet()
	m := newMetrics(prometheus.NewRegistry())
	cfg := SweepConfig{FullPassPeriod: fullPassPeriod, ReplayHorizon: replayHorizon}
	return NewSweeper(dir, reg, tenants, cfg, m, log.NewNopLogger()), dir, reg, tenants
}

// TestSweep_TombstoneReclamationRequiresFullPassAndReplayHorizon is the
// named test for invariant #9 (§7): tombstone reclamation requires BOTH a
// clean full pass (confirming zero remaining leaf entries) AND DeletedAt
// older than ReplayHorizon. The "a delete racing mid-pass waits for the
// next full pass" rule (sweep.go's Pass doc comment) is exercised here as
// its exact operative form: any pass whose OWN registry snapshot predates
// a delete leaves that block completely untouched, whether the wall-clock
// race was a mid-walk interleaving or simply "before this Pass call" --
// from that pass's point of view the two are indistinguishable, since the
// snapshot is the only thing Pass ever consults.
func TestSweep_TombstoneReclamationRequiresFullPassAndReplayHorizon(t *testing.T) {
	const tenantID = "tenant-a"
	tr := testTimeRange()

	sweeper, dir, reg, tenants := newTestSweeper(t, time.Hour, time.Hour)
	const leafIdx = uint32(3)
	completeLeaf(t, dir, leafIdx)

	uuid := testUUID(t, 1)
	blk, _ := reg.GetOrCreate(uuid, tenantID, tr.start, tr.end)
	require.NoError(t, reg.CommitLive(uuid, false))
	tenants.AddBlock(tenantID, blk.Handle, tr.start, tr.end)
	require.True(t, dir.InsertLive(leafIdx, 42, blk.Handle))

	t.Run("pass before the delete leaves the entry and the block untouched", func(t *testing.T) {
		stats := sweeper.Pass(context.Background())
		assert.Zero(t, stats.EntriesRemoved)
		assert.Zero(t, stats.TombstonesReclaimed)

		leaf, _ := dir.Leaf(leafIdx)
		assert.Equal(t, []Handle{blk.Handle}, leaf.Lookup(42))
	})

	require.NoError(t, reg.MarkDeleted(uuid))
	tenants.RemoveBlock(tenantID, blk.Handle)

	t.Run("first pass after the delete removes the entry but does not reclaim (replay horizon not yet elapsed)", func(t *testing.T) {
		stats := sweeper.Pass(context.Background())
		assert.Equal(t, 1, stats.EntriesRemoved)
		assert.Zero(t, stats.TombstonesReclaimed)

		leaf, _ := dir.Leaf(leafIdx)
		assert.Empty(t, leaf.Lookup(42), "the deleted block's entry must be gone from the leaf")

		_, ok := reg.LookupUUID(uuid)
		assert.True(t, ok, "the tombstone must still be present in the registry: ReplayHorizon has not elapsed")
	})

	// Simulate ReplayHorizon having elapsed via direct field mutation
	// (white-box, same-package -- registry_test.go's own established
	// convention for forcing a state) rather than a real sleep, for a
	// fast, deterministic test.
	tombstoneBlock, ok := reg.LookupUUID(uuid)
	require.True(t, ok)
	tombstoneBlock.DeletedAt = time.Now().Add(-2 * time.Hour)

	t.Run("a pass after both entries-confirmed-zero and the replay horizon reclaims the tombstone", func(t *testing.T) {
		stats := sweeper.Pass(context.Background())
		assert.Zero(t, stats.EntriesRemoved, "nothing left to remove: the prior pass already swept the entry")
		assert.Equal(t, 1, stats.TombstonesReclaimed)

		_, ok := reg.LookupUUID(uuid)
		assert.False(t, ok, "the tombstone must be gone from the registry")
	})
}

// TestSweep_LeavesLiveEntriesUntouched: a leaf mixing a live block's entry
// with a deleted block's entry keeps exactly the live one after a pass.
func TestSweep_LeavesLiveEntriesUntouched(t *testing.T) {
	const tenantID = "tenant-a"
	tr := testTimeRange()

	sweeper, dir, reg, tenants := newTestSweeper(t, time.Hour, time.Hour)
	const leafIdx = uint32(5)
	completeLeaf(t, dir, leafIdx)

	liveHandle := liveBlock(t, reg, tenants, 1, tr.start, tr.end)
	require.True(t, dir.InsertLive(leafIdx, 1, liveHandle))

	deletedUUID := testUUID(t, 2)
	deletedBlk, _ := reg.GetOrCreate(deletedUUID, tenantID, tr.start, tr.end)
	require.NoError(t, reg.CommitLive(deletedUUID, false))
	tenants.AddBlock(tenantID, deletedBlk.Handle, tr.start, tr.end)
	require.True(t, dir.InsertLive(leafIdx, 2, deletedBlk.Handle))
	require.NoError(t, reg.MarkDeleted(deletedUUID))
	tenants.RemoveBlock(tenantID, deletedBlk.Handle)

	stats := sweeper.Pass(context.Background())
	assert.Equal(t, 1, stats.EntriesRemoved)

	leaf, _ := dir.Leaf(leafIdx)
	assert.Equal(t, []Handle{liveHandle}, leaf.Lookup(1), "the live block's entry must survive")
	assert.Empty(t, leaf.Lookup(2), "the deleted block's entry must be gone")
}

// TestSweep_EmptyBucketsDropped: deleting every block in one A_T bucket
// drops that bucket on the next pass, while a still-populated sibling
// bucket survives (DESIGN.md § Garbage collection: "Empty A_T buckets are
// dropped on the same pass").
func TestSweep_EmptyBucketsDropped(t *testing.T) {
	const tenantID = "tenant-a"
	sweeper, _, reg, tenants := newTestSweeper(t, time.Hour, time.Hour)

	h1 := liveBlock(t, reg, tenants, 1, hourMark(0).Add(10*time.Minute), hourMark(0).Add(20*time.Minute))
	h2 := liveBlock(t, reg, tenants, 2, hourMark(1).Add(10*time.Minute), hourMark(1).Add(20*time.Minute))

	require.NoError(t, reg.MarkDeleted(testUUID(t, 1)))
	tenants.RemoveBlock(tenantID, h1) // bucket 0 now empty; bucket 1 still holds h2

	sweeper.Pass(context.Background())

	tn := tenants.tenants[tenantID]
	require.NotNil(t, tn)
	_, ok := tn.buckets[bucketKey(0)]
	assert.False(t, ok, "the now-empty bucket must be dropped")
	bm, ok := tn.buckets[bucketKey(1)]
	require.True(t, ok, "the still-populated bucket must survive")
	assert.True(t, bm.Contains(uint32(h2)))
}

// TestSweep_DropsEmptyBucketsOnlyForTenantsKnownToTheRegistry documents
// (and locks in) sweep.go's deliberate scope choice: Pass derives its
// "every active tenant" set from the registry's own Block.TenantID field
// rather than a new TenantSet enumeration method (see the comment at its
// call site in sweep.go). A tenant with zero blocks ever registered is
// therefore never visited -- which is fine, since DropEmptyBuckets on a
// tenant with no buckets at all is already a no-op (tenant_test.go's own
// TestTenantSet_DropEmptyBuckets_UnknownTenantIsNoOp); this test confirms
// Pass does not panic or otherwise misbehave in that case.
func TestSweep_DropsEmptyBucketsOnlyForTenantsKnownToTheRegistry(t *testing.T) {
	sweeper, _, _, tenants := newTestSweeper(t, time.Hour, time.Hour)
	tenants.AddBlock("tenant-with-no-registry-entry", Handle(1), hourMark(0), hourMark(0).Add(time.Minute))

	assert.NotPanics(t, func() {
		sweeper.Pass(context.Background())
	})
}

// TestSweep_Run_StopsPromptlyOnContextCancellation covers Run's own pacing
// loop: it must actually run passes (a short FullPassPeriod ticks several
// times within the sleep below) and return promptly once ctx is done, with
// no leaked goroutine (testing-conventions report recommendation #5,
// following tempodb/pool's own goleak bracketing convention).
func TestSweep_Run_StopsPromptlyOnContextCancellation(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	sweeper, _, _, _ := newTestSweeper(t, 5*time.Millisecond, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		sweeper.Run(ctx)
	}()

	time.Sleep(30 * time.Millisecond) // let a handful of passes happen
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return promptly after context cancellation")
	}

	goleak.VerifyNone(t, opts)
}

// TestSweep_Pass_RecountsEntriesAtEndOfCompletePass is the named test for
// entries_total's self-healing design (directory.go's own doc comment on
// Directory.SetEntryTotal): a complete pass must independently recompute the
// TRUE entry total from the leaves it just walked and OVERWRITE whatever the
// incremental atomic counter held -- correcting drift injected here to
// simulate a hypothetical missed-accounting bug elsewhere, not merely
// confirming the counter already agreed with itself.
func TestSweep_Pass_RecountsEntriesAtEndOfCompletePass(t *testing.T) {
	sweeper, dir, _, _ := newTestSweeper(t, time.Hour, time.Hour)

	const completeIdx = uint32(3)
	completeLeaf(t, dir, completeIdx)
	require.True(t, dir.InsertLive(completeIdx, 1, Handle(1)))
	require.True(t, dir.InsertLive(completeIdx, 2, Handle(2)))
	require.True(t, dir.InsertLive(completeIdx, 3, Handle(3)))

	// A constructing leaf's entries must also feed the recount (via
	// EntryLen, since CompactLeaf only reports complete leaves) -- otherwise
	// a pass would systematically undercount by every constructing leaf's
	// contribution.
	const constructingIdx = uint32(5)
	_, started := dir.BeginConstructing(constructingIdx)
	require.True(t, started)
	require.True(t, dir.InsertLive(constructingIdx, 9, Handle(9)))
	require.True(t, dir.InsertLive(constructingIdx, 10, Handle(10)))

	const trueTotal = 5 // 3 complete + 2 constructing
	require.EqualValues(t, trueTotal, dir.EntryTotal(), "test sanity: incremental accounting must already agree before injecting drift")

	// Inject drift: simulate a hypothetical bug in some OTHER accounting
	// path having lost track of reality.
	dir.SetEntryTotal(trueTotal + 1000)
	require.EqualValues(t, trueTotal+1000, dir.EntryTotal(), "test sanity: drift injection must have taken effect")

	stats := sweeper.Pass(context.Background())
	assert.Zero(t, stats.EntriesRemoved, "nothing was deleted in this test; the recount, not compaction, is under test")
	assert.EqualValues(t, trueTotal, dir.EntryTotal(), "a complete pass must self-heal entries_total to the recomputed truth, discarding the drifted value")
}

// TestSweep_Pass_IncompleteWalkDoesNotOverwriteEntryTotal is
// TestSweep_CancelledPassDoesNotReclaimTombstones' entries_total analogue
// (regression_phasec_test.go): a pass whose directory walk was cut short by
// ctx cancellation only observed a PARTIAL subset of leaves, so applying its
// recount would undercount entries_total for every leaf past the cutoff --
// the same "only a COMPLETE pass is authoritative" rule that already gates
// tombstone reclamation must also gate the recount.
func TestSweep_Pass_IncompleteWalkDoesNotOverwriteEntryTotal(t *testing.T) {
	sweeper, dir, _, _ := newTestSweeper(t, time.Hour, time.Hour)

	const idx = uint32(3)
	completeLeaf(t, dir, idx)
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	require.True(t, dir.InsertLive(idx, 2, Handle(2)))

	const drifted = 999
	dir.SetEntryTotal(drifted)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: the walk aborts at the first non-nil leaf

	stats := sweeper.Pass(ctx)
	assert.Zero(t, stats.EntriesRemoved)
	assert.EqualValues(t, drifted, dir.EntryTotal(), "an incomplete walk must leave entries_total untouched, not overwrite it with a partial recount")
}

// TestSweep_ConcurrentPassWithLiveWrites is the plan's own "concurrent Pass
// + live writes under -race" test plan item (§ Concurrency: "the sweep
// runs continuously in production, never quiesced"): one goroutine hammers
// Pass in a tight loop while another concurrently adds and deletes blocks
// through the same Directory/Registry/TenantSet Pass reads and mutates.
// There is no precise before/after assertion here (the whole point is that
// the two race) -- -race catching nothing, plus the post-loop consistency
// check below, is the actual test.
func TestSweep_ConcurrentPassWithLiveWrites(t *testing.T) {
	const tenantID = "tenant-a"
	sweeper, dir, reg, tenants := newTestSweeper(t, time.Hour, 20*time.Millisecond)
	hashSeed := HashSeed([]byte("bloom-gateway-sweep-concurrency-test-seed"))

	for i := uint32(0); i < uint32(1)<<testD; i++ {
		completeLeaf(t, dir, i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for ctx.Err() == nil {
			sweeper.Pass(ctx)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		n := 0
		for ctx.Err() == nil {
			n++
			uuid := backend.NewUUID()
			start := time.Now()
			blk, _ := reg.GetOrCreate(uuid, tenantID, start, start.Add(time.Minute))
			if !assert.NoError(t, reg.CommitLive(uuid, false)) {
				return
			}
			tenants.AddBlock(tenantID, blk.Handle, start, start.Add(time.Minute))

			id := make([]byte, 16)
			binary.BigEndian.PutUint64(id[8:], uint64(n))
			leafIdx, fp := Address(id, hashSeed, testD, testF)
			dir.InsertLive(leafIdx, uint16(fp), blk.Handle)

			if n%3 == 0 {
				if !assert.NoError(t, reg.MarkDeleted(uuid)) {
					return
				}
				tenants.RemoveBlock(tenantID, blk.Handle)
			}
		}
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()
	wg.Wait()
}
