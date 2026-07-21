package bloomgateway

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDirectory_UnservedLeavesNeverAnswer is the named test for invariant #1
// and half of invariant #7 (§7): Lookup on a nil or constructing index must
// always return ok=false, and MUST NOT ever return a non-nil handles slice
// — an unserved leaf answering at all, even emptily-but-truthily, would be
// indistinguishable from a served empty-complete leaf at the type level if
// this weren't asserted directly.
func TestDirectory_UnservedLeavesNeverAnswer(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(5)

	t.Run("nil", func(t *testing.T) {
		require.Equal(t, LeafNil, dir.State(idx))
		handles, ok := dir.Lookup(idx, 123)
		assert.False(t, ok)
		assert.Nil(t, handles)
	})

	t.Run("constructing", func(t *testing.T) {
		leaf, started := dir.BeginConstructing(idx)
		require.True(t, started)
		require.NotNil(t, leaf)

		// Even after live writes accumulate into the constructing leaf,
		// it must still refuse to answer — completeness, not mere
		// non-emptiness, gates serving (DESIGN.md § Design constraints:
		// "a leaf is never served from partial state").
		applied := dir.InsertLive(idx, 123, Handle(1))
		require.True(t, applied)

		require.Equal(t, LeafConstructing, dir.State(idx))
		handles, ok := dir.Lookup(idx, 123)
		assert.False(t, ok)
		assert.Nil(t, handles)
	})
}

// TestDirectory_EmptyCompleteLeafDistinctFromNil is the named test for the
// other half of invariant #7 (§7): an owned, complete, empty leaf legitimately
// rejects everything in-window, which the wire protocol represents as
// ok=true with handles=nil — genuinely different from ok=false, even though
// both surface a nil handles slice in Go.
func TestDirectory_EmptyCompleteLeafDistinctFromNil(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(3)

	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.NoError(t, dir.Complete(idx, leaf))
	require.Equal(t, LeafComplete, dir.State(idx))

	handles, ok := dir.Lookup(idx, 42)
	assert.True(t, ok, "a complete leaf must answer, even with no match")
	assert.Nil(t, handles, "a complete-but-empty leaf's zero-match answer has no handles")

	// Contrast directly against a genuinely nil (never-owned) index: same
	// nil handles slice, different ok.
	nilIdx := uint32(7)
	nilHandles, nilOK := dir.Lookup(nilIdx, 42)
	assert.False(t, nilOK)
	assert.Nil(t, nilHandles)
	assert.Equal(t, handles, nilHandles, "both surface nil handles in Go, but ok must differ")
	assert.NotEqual(t, ok, nilOK)
}

// TestDirectory_FullLifecycle drives nil -> constructing -> complete -> nil
// end to end, asserting each transition's effect on Lookup/InsertLive at
// every step (WP10's own "full lifecycle state machine" test plan item).
func TestDirectory_FullLifecycle(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(9)

	// 1. nil: writes dropped, queries empty.
	require.Equal(t, LeafNil, dir.State(idx))
	assert.False(t, dir.InsertLive(idx, 1, Handle(1)))
	_, ok := dir.Lookup(idx, 1)
	assert.False(t, ok)

	// 2. constructing: writes accumulate, queries still empty.
	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.Equal(t, LeafConstructing, dir.State(idx))

	assert.True(t, dir.InsertLive(idx, 1, Handle(1)))
	_, ok = dir.Lookup(idx, 1)
	assert.False(t, ok, "still unserved while constructing, even though the write was applied")

	// A second BeginConstructing on the same (already constructing) index
	// is a benign no-op and must not replace the accumulating leaf.
	sameLeaf, startedAgain := dir.BeginConstructing(idx)
	assert.False(t, startedAgain)
	assert.Nil(t, sameLeaf)
	handlesInLeaf := leaf.Lookup(1)
	assert.Equal(t, []Handle{1}, handlesInLeaf, "the original constructing leaf must still hold the write applied above")

	// 3. complete: now served, and reflects every write applied since
	// BeginConstructing (here, trivially, because leaf IS that same
	// object).
	require.NoError(t, dir.Complete(idx, leaf))
	require.Equal(t, LeafComplete, dir.State(idx))

	handles, ok := dir.Lookup(idx, 1)
	require.True(t, ok)
	assert.Equal(t, []Handle{1}, handles)

	// Live writes continue to apply once complete.
	assert.True(t, dir.InsertLive(idx, 2, Handle(2)))
	handles, ok = dir.Lookup(idx, 2)
	require.True(t, ok)
	assert.Equal(t, []Handle{2}, handles)

	// 4. shed: complete -> nil, atomically stops both serving and writes.
	dir.Shed(idx)
	require.Equal(t, LeafNil, dir.State(idx))
	assert.False(t, dir.InsertLive(idx, 3, Handle(3)))
	_, ok = dir.Lookup(idx, 3)
	assert.False(t, ok)
}

func TestDirectory_BeginConstructing_NoOpIfNotNil(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(1)

	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.NoError(t, dir.Complete(idx, leaf))

	// Already complete: BeginConstructing must not demote it back to
	// constructing or replace its leaf.
	got, started := dir.BeginConstructing(idx)
	assert.False(t, started)
	assert.Nil(t, got)
	assert.Equal(t, LeafComplete, dir.State(idx))
}

func TestDirectory_Complete_ErrorsIfNotConstructing(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(1)

	// Never begun: Complete on a nil slot is a caller-contract violation.
	err := dir.Complete(idx, NewLeaf())
	assert.Error(t, err)
	assert.Equal(t, LeafNil, dir.State(idx), "a failed Complete must not mutate the slot")

	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.NoError(t, dir.Complete(idx, leaf))

	// Already complete: calling Complete again must error, not silently
	// re-swap.
	err = dir.Complete(idx, NewLeaf())
	assert.Error(t, err)
	assert.Equal(t, LeafComplete, dir.State(idx))
}

func TestDirectory_Shed_NoOpIfAlreadyNil(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(2)
	require.Equal(t, LeafNil, dir.State(idx))

	dir.Shed(idx) // must not panic
	assert.Equal(t, LeafNil, dir.State(idx))
}

func TestDirectory_Shed_FromConstructingAbandonsBackfill(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(2)

	_, started := dir.BeginConstructing(idx)
	require.True(t, started)

	dir.Shed(idx)
	assert.Equal(t, LeafNil, dir.State(idx))

	// The abandoned range must be re-acquirable from scratch (nil, not
	// stuck constructing).
	leaf, started := dir.BeginConstructing(idx)
	assert.True(t, started)
	assert.Equal(t, 0, leaf.Len())
}

// TestDirectory_Leaf_ReturnsCurrentObjectAndStateForEveryLifecycleStage
// covers the Leaf accessor added as a WP16 (sweep.go) gap-fix (see its doc
// comment in directory.go): it must return (nil, LeafNil) before
// ownership, the SAME object BeginConstructing handed back while
// constructing (so a caller can read entries a concurrent backfill is
// accumulating), the object Complete swapped in once complete, and
// (nil, LeafNil) again after Shed.
func TestDirectory_Leaf_ReturnsCurrentObjectAndStateForEveryLifecycleStage(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(8)

	leaf, state := dir.Leaf(idx)
	assert.Nil(t, leaf)
	assert.Equal(t, LeafNil, state)

	constructing, started := dir.BeginConstructing(idx)
	require.True(t, started)
	gotLeaf, gotState := dir.Leaf(idx)
	assert.Same(t, constructing, gotLeaf, "must be the SAME object live writes are accumulating into")
	assert.Equal(t, LeafConstructing, gotState)

	require.NoError(t, dir.Complete(idx, constructing))
	gotLeaf, gotState = dir.Leaf(idx)
	assert.Same(t, constructing, gotLeaf)
	assert.Equal(t, LeafComplete, gotState)

	replacement := NewLeaf()
	dir.Swap(idx, replacement)
	gotLeaf, gotState = dir.Leaf(idx)
	assert.Same(t, replacement, gotLeaf, "Swap must be immediately visible to Leaf")
	assert.Equal(t, LeafComplete, gotState)

	dir.Shed(idx)
	gotLeaf, gotState = dir.Leaf(idx)
	assert.Nil(t, gotLeaf)
	assert.Equal(t, LeafNil, gotState)
}

// TestDirectory_CloneLeaf_IndependentCopyAndNilState covers CloneLeaf's
// sequential contract: nil for an unowned index (with its state), and
// otherwise a deep copy that shares no memory with the live leaf (mutating
// one must never affect the other) -- the concurrent-safety half (why this
// method exists at all, instead of Leaf+an external Clone()) is covered by
// TestDirectory_CloneLeaf_SafeConcurrentWithInsertLive below and by
// sweep_test.go's TestSweep_ConcurrentPassWithLiveWrites, which is what
// originally caught the race this method fixes.
func TestDirectory_CloneLeaf_IndependentCopyAndNilState(t *testing.T) {
	dir := NewDirectory(4)

	nilIdx := uint32(1)
	clone, state := dir.CloneLeaf(nilIdx)
	assert.Nil(t, clone)
	assert.Equal(t, LeafNil, state)

	idx := uint32(2)
	completeLeaf(t, dir, idx)
	require.True(t, dir.InsertLive(idx, 7, Handle(1)))

	clone, state = dir.CloneLeaf(idx)
	require.Equal(t, LeafComplete, state)
	require.NotNil(t, clone)
	assert.Equal(t, []Handle{1}, clone.Lookup(7))

	// Mutate the clone; the live leaf (observed via Lookup) must be
	// unaffected, and vice versa (insert into the live leaf; the
	// already-taken clone must not see it).
	clone.InsertIfAbsent(7, Handle(2))
	liveHandles, ok := dir.Lookup(idx, 7)
	require.True(t, ok)
	assert.Equal(t, []Handle{1}, liveHandles, "mutating the clone must not affect the live leaf")

	require.True(t, dir.InsertLive(idx, 9, Handle(3)))
	assert.Nil(t, clone.Lookup(9), "a live write after CloneLeaf must not retroactively appear in the already-taken clone")
}

// TestDirectory_CloneLeaf_SafeConcurrentWithInsertLive is the direct
// regression test for the race CloneLeaf was added to fix (see its doc
// comment in directory.go): repeatedly cloning a leaf while another
// goroutine hammers InsertLive on the same index must never race, unlike
// Leaf()+an external Clone() call.
func TestDirectory_CloneLeaf_SafeConcurrentWithInsertLive(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(3)
	completeLeaf(t, dir, idx)

	stop := make(chan struct{})
	var writer sync.WaitGroup
	writer.Add(1)
	go func() {
		defer writer.Done()
		var i uint16
		for {
			select {
			case <-stop:
				return
			default:
			}
			dir.InsertLive(idx, i, Handle(i)+1)
			i++
		}
	}()

	for i := 0; i < 500; i++ {
		clone, state := dir.CloneLeaf(idx)
		assert.Equal(t, LeafComplete, state)
		assert.NotNil(t, clone)
	}

	close(stop)
	writer.Wait()
}

func TestDirectory_Swap_ReplacesLeafWithoutChangingState(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(6)

	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.NoError(t, dir.Complete(idx, leaf))
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))

	// Sweep-style compaction: clone, filter, swap back in.
	replacement := leaf.Clone()
	removed := replacement.RemoveWhere(func(Handle) bool { return false })
	require.Equal(t, 1, removed)

	dir.Swap(idx, replacement)

	assert.Equal(t, LeafComplete, dir.State(idx), "Swap must not change lifecycle state")
	handles, ok := dir.Lookup(idx, 1)
	require.True(t, ok)
	assert.Nil(t, handles, "the swapped-in (filtered) leaf must be the one Lookup now sees")
}

func TestDirectory_Range_VisitsOnlyNonNilSlotsInOrder(t *testing.T) {
	dir := NewDirectory(4) // 16 slots

	leaves := map[uint32]*Leaf{}
	for _, idx := range []uint32{2, 5, 9} {
		leaf, started := dir.BeginConstructing(idx)
		require.True(t, started)
		leaves[idx] = leaf
	}
	// idx 5 goes all the way to complete; 2 and 9 stay constructing.
	require.NoError(t, dir.Complete(5, leaves[5]))

	var visited []uint32
	states := map[uint32]LeafState{}
	dir.Range(func(idx uint32, state LeafState) bool {
		visited = append(visited, idx)
		states[idx] = state
		return true
	})

	assert.Equal(t, []uint32{2, 5, 9}, visited, "Range must visit non-nil slots in increasing idx order and skip nil ones")
	assert.Equal(t, LeafConstructing, states[2])
	assert.Equal(t, LeafComplete, states[5])
	assert.Equal(t, LeafConstructing, states[9])
}

func TestDirectory_Range_EarlyStop(t *testing.T) {
	dir := NewDirectory(4)
	for _, idx := range []uint32{1, 2, 3} {
		_, started := dir.BeginConstructing(idx)
		require.True(t, started)
	}

	var visited []uint32
	dir.Range(func(idx uint32, _ LeafState) bool {
		visited = append(visited, idx)
		return idx != 2 // stop right after visiting idx 2
	})

	assert.Equal(t, []uint32{1, 2}, visited)
}

// TestDirectory_ConcurrentInsertLiveAndLookup_SharedStripe regression-guards
// the striping scheme: two different leaves that happen to share a stripe
// (idx mod 1024) must not corrupt each other under concurrent InsertLive/
// Lookup from many goroutines. Run under -race.
func TestDirectory_ConcurrentInsertLiveAndLookup_SharedStripe(t *testing.T) {
	const d = 11 // 2^11 = 2048 slots > 1024 stripes: guarantees a collision
	dir := NewDirectory(d)

	idxA := uint32(3)
	idxB := idxA + directoryStripes
	require.Equal(t, stripeFor(idxA), stripeFor(idxB), "test setup must actually exercise a shared stripe")
	require.NotEqual(t, idxA, idxB)

	for _, idx := range []uint32{idxA, idxB} {
		leaf, started := dir.BeginConstructing(idx)
		require.True(t, started)
		require.NoError(t, dir.Complete(idx, leaf))
	}

	const perLeaf = 300
	var writers sync.WaitGroup
	for _, idx := range []uint32{idxA, idxB} {
		idx := idx
		for i := 0; i < perLeaf; i++ {
			i := i
			writers.Add(1)
			go func() {
				defer writers.Done()
				applied := dir.InsertLive(idx, uint16(i), Handle(i+1))
				assert.True(t, applied)
			}()
		}
	}

	// Concurrent readers hammering both leaves throughout the writes
	// above — exercising RLock/Lock interleaving on the shared stripe, not
	// asserting on the necessarily-racy intermediate results.
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
					dir.Lookup(idxA, 0)
					dir.Lookup(idxB, 0)
				}
			}
		}()
	}

	writers.Wait()
	close(stop)
	readers.Wait()

	for _, idx := range []uint32{idxA, idxB} {
		for i := 0; i < perLeaf; i++ {
			handles, ok := dir.Lookup(idx, uint16(i))
			require.True(t, ok)
			assert.Contains(t, handles, Handle(i+1), "idx=%d fp=%d", idx, i)
		}
	}
}

// TestDirectory_Shed_SequentialNoWriteAfterReturn is the deterministic half
// of "Shed is observed atomically by a concurrent InsertLive": once Shed has
// returned, the very next InsertLive/Lookup call must see LeafNil, with no
// window of staleness.
func TestDirectory_Shed_SequentialNoWriteAfterReturn(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(4)

	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.NoError(t, dir.Complete(idx, leaf))
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))

	dir.Shed(idx)

	applied := dir.InsertLive(idx, 2, Handle(2))
	assert.False(t, applied, "no write may apply the instant after Shed returns")
	_, ok := dir.Lookup(idx, 1)
	assert.False(t, ok, "no query may be answered the instant after Shed returns, even for a fp inserted before the shed")
}

// TestDirectory_Shed_ConcurrentInsertLiveNoTornState stresses Shed against
// a hammering InsertLive goroutine under -race: once Shed returns, no
// InsertLive call that returns afterward may report applied=true, and the
// slot must deterministically read back as LeafNil with no torn
// state/leaf-pointer combination visible to Lookup.
func TestDirectory_Shed_ConcurrentInsertLiveNoTornState(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(4)

	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.NoError(t, dir.Complete(idx, leaf))

	var shedReturned boolFlag
	appliedAfterShedReturned := boolFlag{}

	stop := make(chan struct{})
	var writer sync.WaitGroup
	writer.Add(1)
	go func() {
		defer writer.Done()
		var i uint16
		for {
			select {
			case <-stop:
				return
			default:
			}
			applied := dir.InsertLive(idx, i, Handle(i)+1)
			if shedReturned.get() && applied {
				appliedAfterShedReturned.set()
			}
			i++
		}
	}()

	time.Sleep(2 * time.Millisecond)
	dir.Shed(idx)
	shedReturned.set()
	time.Sleep(2 * time.Millisecond)

	close(stop)
	writer.Wait()

	assert.False(t, appliedAfterShedReturned.get(), "no InsertLive call may report applied=true once Shed has returned")
	assert.Equal(t, LeafNil, dir.State(idx))
	_, ok := dir.Lookup(idx, 0)
	assert.False(t, ok)
}

// boolFlag is a tiny mutex-guarded bool for the test above — deliberately
// not a bare bool (which -race would rightly flag as a data race across
// goroutines) and not go.uber.org/atomic.Bool (either would work; a mutex
// keeps this test's synchronization obviously correct without pulling in
// another import for a single flag).
type boolFlag struct {
	mu sync.Mutex
	v  bool
}

func (f *boolFlag) set() {
	f.mu.Lock()
	f.v = true
	f.mu.Unlock()
}

func (f *boolFlag) get() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.v
}

// TestDirectory_EntryTotal_CountsAppliedInsertsOnly is the named test for
// InsertLive's hot-path accounting contract (directory.go's own doc
// comment): EntryTotal must grow by exactly one per ACTUALLY-inserted (fp,
// h) pair, and redelivery of an already-present pair (InsertIfAbsent
// returning false) must not double-count.
func TestDirectory_EntryTotal_CountsAppliedInsertsOnly(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(1)
	completeLeaf(t, dir, idx)
	assert.Zero(t, dir.EntryTotal())

	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	assert.EqualValues(t, 1, dir.EntryTotal())

	require.True(t, dir.InsertLive(idx, 2, Handle(2)))
	assert.EqualValues(t, 2, dir.EntryTotal())

	// Redelivery of the exact same (fp, handle) pair must not increment --
	// InsertIfAbsent itself reports no insert happened.
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	assert.EqualValues(t, 2, dir.EntryTotal(), "redelivered (fp, handle) pair must not be double-counted")

	// A different leaf's inserts accumulate onto the same instance-wide total.
	otherIdx := uint32(2)
	completeLeaf(t, dir, otherIdx)
	require.True(t, dir.InsertLive(otherIdx, 5, Handle(5)))
	assert.EqualValues(t, 3, dir.EntryTotal())

	// A drop (nil slot) must not be counted at all.
	nilIdx := uint32(3)
	require.False(t, dir.InsertLive(nilIdx, 9, Handle(9)))
	assert.EqualValues(t, 3, dir.EntryTotal())
}

// TestDirectory_EntryTotal_CompactLeafDecrementsByRemoved covers CompactLeaf's
// entries_total bookkeeping: the counter must drop by exactly the number of
// entries RemoveWhere actually removed, never by visited (which includes
// surviving entries).
func TestDirectory_EntryTotal_CompactLeafDecrementsByRemoved(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(1)
	completeLeaf(t, dir, idx)

	for i := uint16(0); i < 5; i++ {
		require.True(t, dir.InsertLive(idx, i, Handle(i)+1))
	}
	require.EqualValues(t, 5, dir.EntryTotal())

	// Keep only even handles (1, 3, 5 survive as odd Handle values below);
	// this removes handles 2 and 4 (Handle(i)+1 for i=1,3) -- 2 of 5.
	visited, removed, compacted := dir.CompactLeaf(idx, func(h Handle) bool { return h%2 == 1 })
	require.True(t, compacted)
	assert.Equal(t, 5, visited)
	assert.Equal(t, 2, removed)
	assert.EqualValues(t, 3, dir.EntryTotal(), "entries_total must drop by exactly removed, not visited")
}

// TestDirectory_EntryTotal_ShedDiscardsLeafEntries covers Shed's accounting:
// discarding ownership of a complete leaf must subtract every entry it held.
func TestDirectory_EntryTotal_ShedDiscardsLeafEntries(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(1)
	completeLeaf(t, dir, idx)
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	require.True(t, dir.InsertLive(idx, 2, Handle(2)))
	require.EqualValues(t, 2, dir.EntryTotal())

	dir.Shed(idx)
	assert.Zero(t, dir.EntryTotal(), "shedding a leaf must give back every entry it held")
}

// TestDirectory_EntryTotal_AbandonDiscardsAccumulatedLiveWrites covers
// Abandon's accounting: a failed reconstruction batch's constructing leaf
// may have already accumulated live writes (reconstruction never pauses
// live application) before the batch fails; Abandon must give those entries
// back, not leave them stranded in the total forever.
func TestDirectory_EntryTotal_AbandonDiscardsAccumulatedLiveWrites(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(1)

	_, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	require.True(t, dir.InsertLive(idx, 2, Handle(2)))
	require.EqualValues(t, 2, dir.EntryTotal())

	require.True(t, dir.Abandon(idx))
	assert.Zero(t, dir.EntryTotal(), "abandoning a constructing leaf must give back its accumulated entries")
}

// TestDirectory_EntryTotal_CompleteAccountsDeltaNotDoubleCounting covers
// Complete's two distinct shapes (directory.go's own doc comment):
//  1. the reconstruction queue's normal path, completing with the SAME
//     *Leaf object BeginConstructing handed back -- already fully counted
//     via InsertLive, so this must be a zero-delta no-op; and
//  2. bloomgateway.go's snapshot-restore path, completing with a DIFFERENT,
//     already-populated *Leaf that was never counted via InsertLive at
//     all -- this must add exactly that leaf's length, once.
func TestDirectory_EntryTotal_CompleteAccountsDeltaNotDoubleCounting(t *testing.T) {
	t.Run("same object as BeginConstructing: zero-delta", func(t *testing.T) {
		dir := NewDirectory(4)
		idx := uint32(1)

		leaf, started := dir.BeginConstructing(idx)
		require.True(t, started)
		require.True(t, dir.InsertLive(idx, 1, Handle(1)))
		require.True(t, dir.InsertLive(idx, 2, Handle(2)))
		require.EqualValues(t, 2, dir.EntryTotal())

		require.NoError(t, dir.Complete(idx, leaf))
		assert.EqualValues(t, 2, dir.EntryTotal(), "completing with the same already-counted object must not change the total")
	})

	t.Run("different, pre-populated object (snapshot restore): adds its length once", func(t *testing.T) {
		dir := NewDirectory(4)
		idx := uint32(1)

		_, started := dir.BeginConstructing(idx)
		require.True(t, started)
		assert.Zero(t, dir.EntryTotal())

		restored := NewLeaf()
		require.True(t, restored.InsertIfAbsent(1, Handle(1)))
		require.True(t, restored.InsertIfAbsent(2, Handle(2)))
		require.True(t, restored.InsertIfAbsent(3, Handle(3)))

		require.NoError(t, dir.Complete(idx, restored))
		assert.EqualValues(t, 3, dir.EntryTotal(), "completing with a pre-populated snapshot-loaded leaf must add its full length exactly once")
	})
}

// TestDirectory_LeafStateCounts_TransitionsOnConstructCompleteShedAbandon is
// the named test for owned_leaves{state}'s bookkeeping: LeafStateCounts must
// track BeginConstructing/Complete/Shed/Abandon exactly, without a Range
// walk.
func TestDirectory_LeafStateCounts_TransitionsOnConstructCompleteShedAbandon(t *testing.T) {
	dir := NewDirectory(4)

	constructing, complete := dir.LeafStateCounts()
	assert.Zero(t, constructing)
	assert.Zero(t, complete)

	leafA, started := dir.BeginConstructing(uint32(1))
	require.True(t, started)
	constructing, complete = dir.LeafStateCounts()
	assert.EqualValues(t, 1, constructing)
	assert.Zero(t, complete)

	_, started = dir.BeginConstructing(uint32(2))
	require.True(t, started)
	constructing, complete = dir.LeafStateCounts()
	assert.EqualValues(t, 2, constructing)
	assert.Zero(t, complete)

	require.NoError(t, dir.Complete(uint32(1), leafA))
	constructing, complete = dir.LeafStateCounts()
	assert.EqualValues(t, 1, constructing, "completing idx 1 must not affect idx 2's constructing count")
	assert.EqualValues(t, 1, complete)

	// Abandon idx 2 (still constructing): constructing count drops, complete
	// stays untouched.
	require.True(t, dir.Abandon(uint32(2)))
	constructing, complete = dir.LeafStateCounts()
	assert.Zero(t, constructing)
	assert.EqualValues(t, 1, complete)

	// Shed idx 1 (complete): complete count drops back to zero.
	dir.Shed(uint32(1))
	constructing, complete = dir.LeafStateCounts()
	assert.Zero(t, constructing)
	assert.Zero(t, complete)
}

// TestDirectory_EntryLen_SafelyReadsConstructingLeafLength covers the
// accessor sweep.go's end-of-pass recount uses for constructing leaves
// (CompactLeaf only reports complete leaves' lengths).
func TestDirectory_EntryLen_SafelyReadsConstructingLeafLength(t *testing.T) {
	dir := NewDirectory(4)
	idx := uint32(1)

	n, state := dir.EntryLen(idx)
	assert.Zero(t, n)
	assert.Equal(t, LeafNil, state)

	_, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	require.True(t, dir.InsertLive(idx, 2, Handle(2)))

	n, state = dir.EntryLen(idx)
	assert.Equal(t, 2, n)
	assert.Equal(t, LeafConstructing, state)
}
