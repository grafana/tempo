package bloomgateway

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	testD uint8 = 4 // 16 leaves — small enough for unit tests
	testF uint8 = 8
)

// newTestApplier builds an Applier over fresh structures. If completeLeaves
// is true, every leaf is driven nil -> constructing -> complete so InsertLive
// actually stores entries (letting tests assert on leaf contents); when
// false, all leaves stay nil, which is exactly the setup the
// unconditional-commit invariant needs.
func newTestApplier(t *testing.T, completeLeaves bool) (*Applier, *Directory, *Registry, *TenantSet) {
	t.Helper()
	dir := NewDirectory(testD)
	if completeLeaves {
		for i := range uint32(1) << testD {
			leaf, started := dir.BeginConstructing(i)
			require.True(t, started)
			require.NoError(t, dir.Complete(i, leaf))
		}
	}
	reg := NewRegistry()
	tenants := NewTenantSet()
	m := newMetrics(prometheus.NewRegistry())
	seed := HashSeed([]byte("bloom-gateway-events-test-seed"))
	return NewApplier(dir, reg, tenants, seed, testD, testF, m), dir, reg, tenants
}

// traceID encodes n into a canonical 16-byte trace ID (nonzero for n >= 0).
func traceID(n int) []byte {
	id := make([]byte, 16)
	for i := range 8 {
		id[15-i] = byte(uint64(n+1) >> (8 * i))
	}
	return id
}

func traceIDs(ns ...int) [][]byte {
	out := make([][]byte, len(ns))
	for i, n := range ns {
		out[i] = traceID(n)
	}
	return out
}

// chunkFor builds one AddChunk of a block. All chunks of a block repeat the
// block-level metadata (tenant, time range), as a real producer emits them.
func chunkFor(uuid backend.UUID, tenant string, tr timeRange, idx, count uint32, ids [][]byte) *tempopb.BloomGatewayAddChunk {
	return &tempopb.BloomGatewayAddChunk{
		BlockId:           uuid.String(),
		TenantId:          tenant,
		StartTimeUnixNano: tr.start.UnixNano(),
		EndTimeUnixNano:   tr.end.UnixNano(),
		ChunkIndex:        idx,
		ChunkCount:        count,
		TraceIds:          ids,
	}
}

type timeRange struct{ start, end time.Time }

func testTimeRange() timeRange {
	start := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	return timeRange{start: start, end: start.Add(30 * time.Minute)}
}

// handleInWindow reports whether the tenant's unscoped A_T window contains h.
func handleInWindow(ts *TenantSet, tenant string, h Handle) bool {
	return ts.Window(tenant, time.Time{}, time.Time{}).Contains(uint32(h))
}

// windowCardinality reports tenant "t"'s unscoped A_T cardinality -- every
// call site in this file exercises the same single-tenant fixture, so unlike
// handleInWindow (which genuinely varies its tenant across files) this one
// hardcodes it rather than carrying an unused parameter.
func windowCardinality(ts *TenantSet) uint64 {
	return ts.Window("t", time.Time{}, time.Time{}).GetCardinality()
}

func mustHandle(t *testing.T, reg *Registry, uuid backend.UUID) Handle {
	t.Helper()
	b, ok := reg.LookupUUID(uuid)
	require.True(t, ok)
	return b.Handle
}

// TestApply_UnconditionalCommitEvenWithZeroLocalEntries (invariant #3): a
// block whose trace IDs all hash to unowned (nil) leaves still enters the
// registry as Live and appears in A_T. Without this, low-volume tenants'
// blocks would be permanently missing from rejection sets.
func TestApply_UnconditionalCommitEvenWithZeroLocalEntries(t *testing.T) {
	applier, dir, reg, tenants := newTestApplier(t, false /* all leaves nil */)
	tr := testTimeRange()
	uuid := backend.NewUUID()

	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "tenant-a", tr, 0, 1, traceIDs(1, 2, 3))))

	state, ok := reg.State(uuid)
	require.True(t, ok)
	require.Equal(t, BlockLive, state, "block must commit even though every trace ID hit a nil leaf")
	require.True(t, handleInWindow(tenants, "tenant-a", mustHandle(t, reg, uuid)))

	// Confirm the premise: nothing was actually stored in any leaf.
	total := 0
	dir.Range(func(uint32, LeafState) bool { return true })
	for _, id := range traceIDs(1, 2, 3) {
		idx, fp := Address(id, applier.hashSeed, testD, testF)
		h, served := dir.Lookup(idx, uint16(fp))
		require.False(t, served, "leaf must be unserved (nil)")
		total += len(h)
	}
	require.Zero(t, total)
}

// TestApply_DeleteIsTerminal_NoResurrection (invariant #4): once a block is
// Deleted, replayed Adds — including the chunk that would have completed it —
// never resurrect it into A_T. Covers both orderings: Delete after a full
// commit, and Delete racing ahead of the completing chunk.
func TestApply_DeleteIsTerminal_NoResurrection(t *testing.T) {
	t.Run("delete after commit, then replay", func(t *testing.T) {
		applier, _, reg, tenants := newTestApplier(t, true)
		tr := testTimeRange()
		uuid := backend.NewUUID()
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 1, traceIDs(1, 2))))
		h := mustHandle(t, reg, uuid)
		require.NoError(t, applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: uuid.String()}))

		// Replay the original completing chunk.
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 1, traceIDs(1, 2))))

		state, _ := reg.State(uuid)
		require.Equal(t, BlockDeleted, state)
		require.False(t, handleInWindow(tenants, "t", h))
	})

	t.Run("delete races ahead of completing chunk", func(t *testing.T) {
		applier, _, reg, tenants := newTestApplier(t, true)
		tr := testTimeRange()
		uuid := backend.NewUUID()
		// Two-chunk block: apply only chunk 0 (still Pending).
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 2, traceIDs(1))))
		require.NoError(t, applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: uuid.String()}))
		h := mustHandle(t, reg, uuid)
		// The completing chunk arrives after the Delete.
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 1, 2, traceIDs(2))))

		state, _ := reg.State(uuid)
		require.Equal(t, BlockDeleted, state)
		require.False(t, handleInWindow(tenants, "t", h))
	})
}

// TestApply_IdempotentUnderRedeliveryAndReorder (invariant #5): applying the
// same block's chunks in random permutations, with duplicates, from many
// goroutines, always converges to the same state — Live, exactly one A_T
// entry, and each trace ID present exactly once in its leaf.
func TestApply_IdempotentUnderRedeliveryAndReorder(t *testing.T) {
	const (
		nChunks   = 6
		perChunk  = 4
		goroutine = 8
	)
	applier, dir, reg, tenants := newTestApplier(t, true)
	tr := testTimeRange()
	uuid := backend.NewUUID()

	// Build the full chunk set; chunk i carries trace IDs [i*perChunk, ...).
	base := make([]*tempopb.BloomGatewayAddChunk, nChunks)
	var allIDs [][]byte
	for i := range nChunks {
		ids := make([][]byte, perChunk)
		for j := range perChunk {
			n := i*perChunk + j
			ids[j] = traceID(n)
			allIDs = append(allIDs, traceID(n))
		}
		base[i] = chunkFor(uuid, "t", tr, uint32(i), nChunks, ids)
	}

	var wg sync.WaitGroup
	for g := range goroutine {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed))
			// A shuffled sequence with each chunk duplicated once to force
			// redelivery.
			seq := make([]*tempopb.BloomGatewayAddChunk, 0, nChunks*2)
			seq = append(seq, base...)
			seq = append(seq, base...)
			rng.Shuffle(len(seq), func(a, b int) { seq[a], seq[b] = seq[b], seq[a] })
			for _, c := range seq {
				require.NoError(t, applier.ApplyAddChunk(c))
			}
		}(int64(g) + 1)
	}
	wg.Wait()

	state, _ := reg.State(uuid)
	require.Equal(t, BlockLive, state)
	h := mustHandle(t, reg, uuid)
	require.Equal(t, uint64(1), windowCardinality(tenants), "exactly one A_T entry despite concurrency and redelivery")
	require.True(t, handleInWindow(tenants, "t", h))

	// Every trace ID present exactly once in its leaf (insert-if-absent).
	for _, id := range allIDs {
		idx, fp := Address(id, applier.hashSeed, testD, testF)
		handles, served := dir.Lookup(idx, uint16(fp))
		require.True(t, served)
		count := 0
		for _, got := range handles {
			if got == h {
				count++
			}
		}
		require.Equal(t, 1, count, "trace id must appear exactly once, no redelivery duplicates")
	}
}

// TestApply_ATMembershipImpliesLive (invariant #2, write side): a block is
// absent from A_T for as long as it is Pending, no matter how many chunks
// have landed; only the completing chunk makes it appear, atomically with the
// Live transition.
func TestApply_ATMembershipImpliesLive(t *testing.T) {
	applier, _, reg, tenants := newTestApplier(t, true)
	tr := testTimeRange()
	uuid := backend.NewUUID()

	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 3, traceIDs(1))))
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 1, 3, traceIDs(2))))
	h := mustHandle(t, reg, uuid)
	state, _ := reg.State(uuid)
	require.Equal(t, BlockPending, state, "still pending with 2 of 3 chunks")
	require.False(t, handleInWindow(tenants, "t", h), "pending block must not be in A_T")

	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 2, 3, traceIDs(3))))
	state, _ = reg.State(uuid)
	require.Equal(t, BlockLive, state)
	require.True(t, handleInWindow(tenants, "t", h))
}

// TestApply_InvalidChunkDroppedWhole: a chunk carrying one malformed trace ID
// applies none of its IDs and does not mark itself seen (so a count=1 block
// stays Pending, not Live), and returns an error.
func TestApply_InvalidChunkDroppedWhole(t *testing.T) {
	for _, tc := range []struct {
		name string
		bad  []byte
	}{
		{"empty", []byte{}},
		{"too long", make([]byte, 17)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			applier, dir, reg, _ := newTestApplier(t, true)
			tr := testTimeRange()
			uuid := backend.NewUUID()
			ids := [][]byte{traceID(1), tc.bad, traceID(2)}

			err := applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 1, ids))
			require.Error(t, err)

			// A chunk that fails validation leaves no trace: validation
			// precedes GetOrCreate, so the block must never even be created,
			// and none of the valid IDs in the same chunk are inserted.
			_, ok := reg.State(uuid)
			require.False(t, ok, "a chunk that fails whole-chunk validation must never create a registry entry")
			for _, good := range [][]byte{traceID(1), traceID(2)} {
				idx, fp := Address(good, applier.hashSeed, testD, testF)
				handles, _ := dir.Lookup(idx, uint16(fp))
				require.Empty(t, handles, "no ID from a dropped chunk may be inserted")
			}
		})
	}
}

// TestApply_UnrecognizedVersionDroppedNotApplied: DecodeEvent rejects any
// envelope version other than the supported one, before Type is dispatched.
func TestApply_UnrecognizedVersionDroppedNotApplied(t *testing.T) {
	applier, _, _, _ := newTestApplier(t, true)
	for _, v := range []uint32{0, 2, 99} {
		event := &tempopb.BloomGatewayEvent{
			Version:  v,
			Type:     tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK,
			AddChunk: chunkFor(backend.NewUUID(), "t", testTimeRange(), 0, 1, traceIDs(1)),
		}
		raw, err := event.Marshal()
		require.NoError(t, err)
		_, err = applier.DecodeEvent(raw)
		require.Error(t, err, "version %d must be rejected", v)
	}

	// The supported version decodes and dispatches.
	good := &tempopb.BloomGatewayEvent{
		Version:  supportedEventVersion,
		Type:     tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK,
		AddChunk: chunkFor(backend.NewUUID(), "t", testTimeRange(), 0, 1, traceIDs(1)),
	}
	raw, err := good.Marshal()
	require.NoError(t, err)
	decoded, err := applier.DecodeEvent(raw)
	require.NoError(t, err)
	require.NoError(t, applier.Apply(decoded))
}

// TestApply_CommitUnsupportedEncodingDemotesLiveBlock (AMENDMENT A1): a block
// made Live by live Adds, then encountered as unsupported during a backfill,
// is demoted to LiveUnsupportedEncoding and removed from A_T so it can never
// be rejected. A never-seen block goes straight to the unsupported state.
func TestApply_CommitUnsupportedEncodingDemotesLiveBlock(t *testing.T) {
	t.Run("demote a live block", func(t *testing.T) {
		applier, _, reg, tenants := newTestApplier(t, true)
		tr := testTimeRange()
		uuid := backend.NewUUID()
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 1, traceIDs(1, 2))))
		h := mustHandle(t, reg, uuid)
		require.True(t, handleInWindow(tenants, "t", h), "precondition: live block is in A_T")

		require.NoError(t, applier.CommitUnsupportedEncoding(uuid, "t", tr.start, tr.end))
		state, _ := reg.State(uuid)
		require.Equal(t, BlockLiveUnsupportedEncoding, state)
		require.False(t, handleInWindow(tenants, "t", h), "demoted block must be gone from A_T")
	})

	t.Run("register a fresh unsupported block", func(t *testing.T) {
		applier, _, reg, tenants := newTestApplier(t, true)
		tr := testTimeRange()
		uuid := backend.NewUUID()
		require.NoError(t, applier.CommitUnsupportedEncoding(uuid, "t", tr.start, tr.end))
		state, ok := reg.State(uuid)
		require.True(t, ok)
		require.Equal(t, BlockLiveUnsupportedEncoding, state)
		require.Equal(t, uint64(0), windowCardinality(tenants), "unsupported block is never in A_T")
	})
}

// TestApply_SyntheticSingleChunkRepairCompletesAlongsidePartialGrouping
// (AMENDMENT A2): a synthetic count=1 repair completes a block immediately,
// even while a partial multi-chunk grouping from the live stream is in flight
// for the same UUID; the stale grouping is discarded, and its leftover chunks
// arriving afterward are harmless no-ops.
func TestApply_SyntheticSingleChunkRepairCompletesAlongsidePartialGrouping(t *testing.T) {
	applier, _, reg, tenants := newTestApplier(t, true)
	tr := testTimeRange()
	uuid := backend.NewUUID()

	// Partial 3-chunk grouping from the live stream.
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 3, traceIDs(1))))
	state, _ := reg.State(uuid)
	require.Equal(t, BlockPending, state)

	// Synthetic single-chunk repair completes it immediately.
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 1, traceIDs(2))))
	state, _ = reg.State(uuid)
	require.Equal(t, BlockLive, state)
	h := mustHandle(t, reg, uuid)
	require.True(t, handleInWindow(tenants, "t", h))

	// The remaining chunks of the abandoned 3-chunk grouping are harmless.
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 1, 3, traceIDs(3))))
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 2, 3, traceIDs(4))))
	require.Equal(t, uint64(1), windowCardinality(tenants))
}

// TestApply_LateChunksAfterCommitAreHarmless (AMENDMENT A2): redelivering a
// chunk after the block has already committed leaves state unchanged and does
// not double-add to A_T.
func TestApply_LateChunksAfterCommitAreHarmless(t *testing.T) {
	applier, _, reg, tenants := newTestApplier(t, true)
	tr := testTimeRange()
	uuid := backend.NewUUID()
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 2, traceIDs(1))))
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 1, 2, traceIDs(2))))
	require.Equal(t, uint64(1), windowCardinality(tenants))

	// Redeliver both chunks after commit.
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 2, traceIDs(1))))
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 1, 2, traceIDs(2))))

	state, _ := reg.State(uuid)
	require.Equal(t, BlockLive, state)
	require.Equal(t, uint64(1), windowCardinality(tenants))
}

// TestApply_DeleteUnknownBlockIsNoop: a Delete for a block this instance has
// never seen is a silent no-op, not an error.
func TestApply_DeleteUnknownBlockIsNoop(t *testing.T) {
	applier, _, _, _ := newTestApplier(t, true)
	require.NoError(t, applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: backend.NewUUID().String()}))
}

// TestApply_Metrics is a smoke test that the apply/delete counters move.
func TestApply_Metrics(t *testing.T) {
	reg := NewRegistry()
	tenants := NewTenantSet()
	dir := NewDirectory(testD)
	promReg := prometheus.NewRegistry()
	m := newMetrics(promReg)
	applier := NewApplier(dir, reg, tenants, HashSeed([]byte("s")), testD, testF, m)
	tr := testTimeRange()
	uuid := backend.NewUUID()

	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 1, traceIDs(1))))
	require.Error(t, applier.ApplyAddChunk(chunkFor(backend.NewUUID(), "t", tr, 0, 1, [][]byte{{}})))
	require.NoError(t, applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: uuid.String()}))

	require.Equal(t, float64(1), promtestCounter(t, m.addsTotal, "applied"))
	require.Equal(t, float64(1), promtestCounter(t, m.addsTotal, "dropped"))
}

func promtestCounter(t *testing.T, vec *prometheus.CounterVec, label string) float64 {
	t.Helper()
	c, err := vec.GetMetricWithLabelValues(label)
	require.NoError(t, err)
	return testutil.ToFloat64(c)
}

// TestApply_CommitBlockUndoesAddWhenUnsupportedEncodingWonRace (review
// finding, invariants #2/#10): a still-Pending block can be demoted straight
// to BlockLiveUnsupportedEncoding by a concurrent CommitUnsupportedEncoding
// (reconciliation) before that same block's own completing chunk reaches
// commitBlock — reachable because recordChunk's "still Pending" check and its
// eventual commitBlock call are separated by the chunkMu section (events.go),
// leaving a window for another goroutine to run entirely in between. When the
// completing chunk's commitBlock lands after losing that race, it must still
// undo its speculative A_T add, or the handle is stuck in A_T forever even
// though the registry correctly says LiveUnsupportedEncoding.
func TestApply_CommitBlockUndoesAddWhenUnsupportedEncodingWonRace(t *testing.T) {
	applier, _, reg, tenants := newTestApplier(t, true)
	tr := testTimeRange()
	uuid := backend.NewUUID()

	// Chunk 0 of 2 lands: block Pending, handle allocated, not yet in A_T.
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 2, traceIDs(1))))
	block, ok := reg.LookupUUID(uuid)
	require.True(t, ok)
	require.Equal(t, BlockPending, block.State)

	// The race: CommitUnsupportedEncoding wins first, demoting the
	// still-forming block directly from Pending.
	require.NoError(t, applier.CommitUnsupportedEncoding(uuid, "t", tr.start, tr.end))
	state, _ := reg.State(uuid)
	require.Equal(t, BlockLiveUnsupportedEncoding, state)

	// The completing chunk's recordChunk had already decided to complete
	// (its "still Pending" check ran before the above) and now reaches
	// commitBlock.
	applier.commitBlock(uuid, "t", block, false)

	state, _ = reg.State(uuid)
	require.Equal(t, BlockLiveUnsupportedEncoding, state, "must not resurrect to Live")
	require.False(t, handleInWindow(tenants, "t", block.Handle),
		"handle must not be stuck in A_T after losing the race to CommitUnsupportedEncoding")
}

// TestApply_DeleteOrderingClosesRaceWithCompletingChunk (review finding,
// invariant #2): ApplyDelete's ordering — MarkDeleted, then RemoveBlock —
// must make the Deleted state visible BEFORE RemoveBlock ever runs, so that
// any completing chunk racing in from that point onward is caught by
// ApplyAddChunk's own tombstone fast-path (never even reaching commitBlock),
// closing the race entirely rather than merely narrowing it. The reverse
// order (RemoveBlock, then MarkDeleted — the original bug) leaves a real
// gap: RemoveBlock fires as a wasted no-op while the block is still Pending,
// a completing chunk lands unobstructed and genuinely goes Live, and by the
// time MarkDeleted finally fires, nothing calls RemoveBlock again.
//
// The gap (or its absence) is only a couple of instructions wide in real
// code, far too narrow for real goroutine scheduling to reliably land in;
// this uses a test-only hook (applyDeleteRaceHook, matching
// modules/frontend/queue's testHookBeforeWaiting convention) to deterministically
// run a completing chunk exactly between ApplyDelete's two effects.
func TestApply_DeleteOrderingClosesRaceWithCompletingChunk(t *testing.T) {
	applier, _, reg, tenants := newTestApplier(t, true)
	tr := testTimeRange()
	uuid := backend.NewUUID()

	// Two-chunk block, only chunk 0 applied: still Pending, handle
	// allocated, not yet in A_T.
	require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 2, traceIDs(1))))
	h := mustHandle(t, reg, uuid)

	var hookRan bool
	applier.applyDeleteRaceHook = func() {
		hookRan = true
		// The Deleted state must already be visible here -- this is what
		// makes a completing chunk racing in from this point onward
		// harmless via the tombstone fast-path, never reaching commitBlock.
		state, _ := reg.State(uuid)
		require.Equal(t, BlockDeleted, state,
			"ApplyDelete's first effect must be the one that makes Deleted visible, before RemoveBlock runs")

		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 1, 2, traceIDs(2))))
	}

	require.NoError(t, applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: uuid.String()}))
	require.True(t, hookRan, "precondition: the race hook must have fired")

	state, _ := reg.State(uuid)
	require.Equal(t, BlockDeleted, state)
	require.False(t, handleInWindow(tenants, "t", h),
		"handle must never be stuck in A_T when a completing chunk races into the gap between ApplyDelete's two effects")
}

// TestApply_ChunkProgressDroppedWhenBlockDeletedMidChunking (review finding):
// a block Deleted before its chunkProgress grouping completes must not leak
// that grouping for the life of the process. Covers both cleanup points —
// ApplyDelete itself, and a later chunk's tombstone fast-path catching a
// grouping orphaned by a Delete that bypassed ApplyDelete's own cleanup
// (isolated here via a direct reg.MarkDeleted, to attribute the cleanup to
// ApplyAddChunk's fast-path specifically rather than ApplyDelete's).
func TestApply_ChunkProgressDroppedWhenBlockDeletedMidChunking(t *testing.T) {
	applier, _, reg, _ := newTestApplier(t, true)
	tr := testTimeRange()

	groupingExists := func(uuid backend.UUID, count uint32) bool {
		applier.chunkMu.Lock()
		defer applier.chunkMu.Unlock()
		_, ok := applier.chunkProgress[chunkKey{uuid: uuid, count: count}]
		return ok
	}

	t.Run("ApplyDelete drops the orphaned grouping", func(t *testing.T) {
		uuid := backend.NewUUID()
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 3, traceIDs(1))))
		require.True(t, groupingExists(uuid, 3), "precondition: a partial grouping exists")

		require.NoError(t, applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: uuid.String()}))
		require.False(t, groupingExists(uuid, 3), "delete must drop the orphaned grouping")
	})

	t.Run("tombstone fast-path drops a grouping left by an earlier delete path", func(t *testing.T) {
		uuid := backend.NewUUID()
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 0, 3, traceIDs(2))))
		require.True(t, groupingExists(uuid, 3), "precondition: a partial grouping exists")

		// Mark deleted directly, bypassing ApplyDelete's own chunkProgress
		// cleanup, to isolate ApplyAddChunk's tombstone fast-path cleanup.
		require.NoError(t, reg.MarkDeleted(uuid))
		require.True(t, groupingExists(uuid, 3), "precondition: still orphaned before the redelivered chunk")

		// A redelivered chunk 1 hits the tombstone fast-path.
		require.NoError(t, applier.ApplyAddChunk(chunkFor(uuid, "t", tr, 1, 3, traceIDs(3))))
		require.False(t, groupingExists(uuid, 3), "tombstone fast-path must drop the orphaned grouping")

		state, _ := reg.State(uuid)
		require.Equal(t, BlockDeleted, state)
	})
}
