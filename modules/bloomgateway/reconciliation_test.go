// reconciliation_test.go implements WP19's test plan (implementation plan
// §3 WP19): repair-Add (lag-gated, index-snapshot-age-gated, AMENDMENT A2's
// widened Pending/absent eligibility), synthesize-Delete (never lag-gated,
// persisted-across-passes grace), and the shared-code-path/shared-rate-
// limiter properties that make both halves reuse WP18's own machinery
// rather than duplicating it.
package bloomgateway

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/time/rate"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

// zeroLag is a lagFn that never gates repair-Adds -- the default for tests
// not specifically exercising the lag gate.
func zeroLag() time.Duration { return 0 }

// permissiveReconciliationConfig is a ReconciliationConfig whose LagGate is
// generous enough that lag-gating never interferes with a test that isn't
// specifically about it.
func permissiveReconciliationConfig() ReconciliationConfig {
	return ReconciliationConfig{Period: time.Hour, LagGate: time.Hour}
}

// staleTenantIndex builds a *backend.TenantIndex whose CreatedAt is well
// past reconciliationGraceWindow -- the repair-Add path's own
// "avoid racing in-flight Adds" grace -- so tests not specifically
// exercising that gate never trip over it.
func staleTenantIndex(metas ...*backend.BlockMeta) *backend.TenantIndex {
	return &backend.TenantIndex{
		CreatedAt: time.Now().Add(-2 * reconciliationGraceWindow),
		Meta:      metas,
	}
}

// mustState is the test-local one-liner for the lock-safe Registry.State
// accessor, mirroring registry_test.go's mustHandle.
func mustState(t testing.TB, reg *Registry, uuid backend.UUID) BlockState {
	t.Helper()
	s, ok := reg.State(uuid)
	require.True(t, ok, "block %s must be registered", uuid)
	return s
}

// TestReconciliation_RepairsStuckPendingBlock is AMENDMENT A2's named test:
// the repair-Add condition is "in the tenant index AND (absent from the
// registry OR present with State == BlockPending) past the grace window" --
// widened from the plan's original "absent from the registry" alone, so
// that a block stuck Pending forever (a dropped final chunk, or a restart
// that lost its transient chunk-arrival bitset -- see events.go's
// chunkProgress, deliberately not persisted) actually heals, as DESIGN.md's
// own § Reconciliation claims. Every registry state is covered in one
// table, so the eligible/ineligible boundary is exercised in exactly one
// place.
//
// The "must never be re-fetched" subtests below use a vparquet5.
// VersionString meta -- a REAL encoding.TraceIDProjector -- so that, if the
// eligibility check is ever wrong and the block is mistakenly re-fetched,
// FetchAndApplyBlockColumn's real vparquet5.OpenTraceIDReader call reaches
// fakeBackendReader's panicking Read/ReadRange/StreamReader (reconstruction
// _test.go's own "fail loudly, not silently" fixture design) instead of
// silently succeeding or no-op'ing.
func TestReconciliation_RepairsStuckPendingBlock(t *testing.T) {
	tr := testTimeRange()

	t.Run("absent from the registry entirely is repaired", func(t *testing.T) {
		applier, _, reg, _ := newTestApplier(t, true)
		id := testUUID(t, 1)
		reader := newFakeBackendReader()
		reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(id, "tenant-a", tr, vparquet3.VersionString)))

		rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
		stats := rec.Pass(context.Background(), "tenant-a")

		assert.Equal(t, BlockLiveUnsupportedEncoding, mustState(t, reg, id), "the repair must have registered the previously-unknown block")
		// applied == false for this fixture (unsupported encoding, see
		// FetchAndApplyBlockColumn), so RepairAdds -- which only counts
		// applied==true outcomes, matching reconstruction.go's own
		// Applied/NotApplied split -- stays 0 even though the repair ran;
		// the state transition above is this subtest's proof that it did.
		assert.Equal(t, 0, stats.RepairAdds)
	})

	t.Run("present as BlockPending (stuck: dropped chunk or lost bitset) is repaired", func(t *testing.T) {
		applier, _, reg, _ := newTestApplier(t, true)
		id := testUUID(t, 2)
		// Simulates a block stuck Pending: some earlier chunk landed
		// (creating the registry entry), but the completing chunk never
		// arrived (dropped by the producer, or the bitset tracking it was
		// lost across a restart -- AMENDMENT A2's own motivating scenario).
		reg.GetOrCreate(id, "tenant-a", tr.start, tr.end)
		require.Equal(t, BlockPending, mustState(t, reg, id))

		reader := newFakeBackendReader()
		reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(id, "tenant-a", tr, vparquet3.VersionString)))

		rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
		rec.Pass(context.Background(), "tenant-a")

		assert.Equal(t, BlockLiveUnsupportedEncoding, mustState(t, reg, id), "a stuck-Pending block must be healed out of Pending, not left stuck forever")
	})

	t.Run("present as BlockLive is left untouched, never re-fetched", func(t *testing.T) {
		applier, _, reg, tenants := newTestApplier(t, true)
		id := testUUID(t, 3)
		require.NoError(t, applier.ApplyAddChunk(chunkFor(id, "tenant-a", tr, 0, 1, traceIDs(30))))
		require.Equal(t, BlockLive, mustState(t, reg, id))
		require.True(t, handleInWindow(tenants, "tenant-a", mustHandle(t, reg, id)))

		reader := newFakeBackendReader()
		reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(id, "tenant-a", tr, vparquet5.VersionString)))

		rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
		rec.Pass(context.Background(), "tenant-a")

		assert.Equal(t, BlockLive, mustState(t, reg, id), "an already-Live block must never be re-fetched or demoted")
		assert.True(t, handleInWindow(tenants, "tenant-a", mustHandle(t, reg, id)))
	})

	t.Run("present as BlockLiveUnsupportedEncoding is left untouched, never re-fetched (never repair-looped)", func(t *testing.T) {
		applier, _, reg, _ := newTestApplier(t, true)
		id := testUUID(t, 4)
		require.NoError(t, applier.CommitUnsupportedEncoding(id, "tenant-a", tr.start, tr.end))
		require.Equal(t, BlockLiveUnsupportedEncoding, mustState(t, reg, id))

		reader := newFakeBackendReader()
		reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(id, "tenant-a", tr, vparquet5.VersionString)))

		rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
		rec.Pass(context.Background(), "tenant-a")

		assert.Equal(t, BlockLiveUnsupportedEncoding, mustState(t, reg, id), "an already-LiveUnsupportedEncoding block must never be re-fetched")
	})

	t.Run("present as BlockDeleted is left untouched, never resurrected", func(t *testing.T) {
		applier, _, reg, _ := newTestApplier(t, true)
		id := testUUID(t, 5)
		require.NoError(t, applier.ApplyAddChunk(chunkFor(id, "tenant-a", tr, 0, 1, traceIDs(31))))
		require.NoError(t, applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: id.String()}))
		require.Equal(t, BlockDeleted, mustState(t, reg, id))

		reader := newFakeBackendReader()
		reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(id, "tenant-a", tr, vparquet5.VersionString)))

		rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
		rec.Pass(context.Background(), "tenant-a")

		assert.Equal(t, BlockDeleted, mustState(t, reg, id), "a Deleted block is terminal -- reconciliation must never resurrect it")
	})
}

// TestReconciliation_RepairAddsLagGated: repair-Adds are suppressed
// entirely while lagFn() exceeds cfg.LagGate, and resume once it drops back
// (§ Reconciliation: "The loop skips repair-Adds until lag is back under
// threshold"). Delete synthesis is a completely separate code path
// (synthesizeDeletes) and is asserted elsewhere (TestReconciliation_
// SynthesizeDeleteRequiresPersistenceAcrossPasses) to be unaffected by lag.
func TestReconciliation_RepairAddsLagGated(t *testing.T) {
	applier, _, reg, _ := newTestApplier(t, true)
	tr := testTimeRange()
	id := testUUID(t, 1)

	reader := newFakeBackendReader()
	reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(id, "tenant-a", tr, vparquet3.VersionString)))

	lag := 20 * time.Minute // starts above the gate below
	lagFn := func() time.Duration { return lag }
	cfg := ReconciliationConfig{Period: time.Hour, LagGate: 5 * time.Minute}

	rec := NewReconciler(reg, applier, reader, cfg, lagFn, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())

	stats := rec.Pass(context.Background(), "tenant-a")
	assert.Equal(t, 0, stats.RepairAdds)
	_, ok := reg.LookupUUID(id)
	assert.False(t, ok, "while lag exceeds the gate, repair-Adds must be suppressed entirely -- the block must stay unknown")

	lag = 1 * time.Minute // drops back under the gate

	rec.Pass(context.Background(), "tenant-a")
	assert.Equal(t, BlockLiveUnsupportedEncoding, mustState(t, reg, id), "once lag drops back under the gate, the previously-suppressed repair must resume")
}

// TestReconciliation_RepairAddsGatedByIndexSnapshotAge: repair-Adds are
// also skipped while the tenant INDEX SNAPSHOT itself is too fresh
// (reconciliationGraceWindow's own doc comment explains why this half uses
// idx.CreatedAt rather than cross-pass memory) -- "to avoid racing
// in-flight Adds" for a block that only just landed in a very fresh index
// read (§ Reconciliation).
func TestReconciliation_RepairAddsGatedByIndexSnapshotAge(t *testing.T) {
	applier, _, reg, _ := newTestApplier(t, true)
	tr := testTimeRange()
	id := testUUID(t, 1)
	meta := blockMetaFixture(id, "tenant-a", tr, vparquet3.VersionString)

	reader := newFakeBackendReader()
	reader.setTenantIndex("tenant-a", &backend.TenantIndex{CreatedAt: time.Now(), Meta: []*backend.BlockMeta{meta}})

	rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())

	rec.Pass(context.Background(), "tenant-a")
	_, ok := reg.LookupUUID(id)
	assert.False(t, ok, "a freshly-built index snapshot must not trigger a repair yet")

	reader.setTenantIndex("tenant-a", staleTenantIndex(meta))
	rec.Pass(context.Background(), "tenant-a")
	assert.Equal(t, BlockLiveUnsupportedEncoding, mustState(t, reg, id), "once the index snapshot is stale enough, the repair must proceed")
}

// TestReconciliation_SynthesizeDeleteRequiresPersistenceAcrossPasses covers
// both failure modes the plan's own risk note warns about: an
// implementation that re-derives "first observed" from scratch every call
// (comparing the same instant against itself) could never synthesize a
// Delete; one that skips the persistence check entirely would synthesize
// one pass too early, defeating the grace window's protection against a
// block merely racing an in-flight compaction/retention cycle. Parametrized
// over the block's registry state (BlockLive and BlockLiveUnsupportedEncoding)
// so the same test also covers "a LiveUnsupportedEncoding block missing
// from the index is Delete-synthesized exactly like a Live one" (§7
// invariant #10's WP19-side exercise, complementing WP18's own
// reconstruction-side test).
func TestReconciliation_SynthesizeDeleteRequiresPersistenceAcrossPasses(t *testing.T) {
	for _, unsupported := range []bool{false, true} {
		name := "Live"
		if unsupported {
			name = "LiveUnsupportedEncoding"
		}

		t.Run(name, func(t *testing.T) {
			applier, _, reg, tenants := newTestApplier(t, true)
			tr := testTimeRange()
			id := testUUID(t, 1)

			require.NoError(t, applier.ApplyAddChunk(chunkFor(id, "tenant-a", tr, 0, 1, traceIDs(1))))
			if unsupported {
				require.NoError(t, applier.CommitUnsupportedEncoding(id, "tenant-a", tr.start, tr.end))
			}
			wantState := BlockLive
			if unsupported {
				wantState = BlockLiveUnsupportedEncoding
			}
			require.Equal(t, wantState, mustState(t, reg, id), "precondition")
			require.Equal(t, !unsupported, handleInWindow(tenants, "tenant-a", mustHandle(t, reg, id)), "precondition: A_T membership tracks Live vs LiveUnsupportedEncoding")

			reader := newFakeBackendReader()
			// The block is NOT in this tenant's index on even the FIRST
			// pass: it's already missing (simulating retention having
			// cleared it while this instance's own Delete never arrived,
			// § Reconciliation's crash-window scenario).
			reader.setTenantIndex("tenant-a", staleTenantIndex())

			rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())

			stats := rec.Pass(context.Background(), "tenant-a")
			assert.Equal(t, 0, stats.SynthesizedDeletes, "must NOT fire on the pass that first observes the block missing")
			assert.Equal(t, wantState, mustState(t, reg, id), "the block must be untouched after only one observation")

			// Simulate the grace window having elapsed since that first
			// observation -- white-box, same-package field mutation
			// (registry_test.go/sweep_test.go's own established convention
			// for forcing time-based state) rather than a real sleep.
			rec.mu.Lock()
			rec.missingSince["tenant-a"][id] = time.Now().Add(-2 * reconciliationGraceWindow)
			rec.mu.Unlock()

			stats = rec.Pass(context.Background(), "tenant-a")
			assert.Equal(t, 1, stats.SynthesizedDeletes, "once the missing condition has persisted past the grace window, the Delete must synthesize")
			assert.Equal(t, BlockDeleted, mustState(t, reg, id))
			assert.False(t, handleInWindow(tenants, "tenant-a", mustHandle(t, reg, id)))
			assert.Equal(t, float64(1), promtestCounter(t, applier.metrics.reconciliationRepairsTotal, "delete"))
		})
	}
}

// TestReconciliation_SharesRateLimiterWithReconstruction: repair fetches
// must draw from the SAME cell-wide *rate.Limiter reconstruction.go's own
// fetches draw from (§ Reconciliation: "repair fetches share the cell-wide
// reconstruction rate limit") -- not a second, independent budget that
// would silently double the effective cell-wide rate.
func TestReconciliation_SharesRateLimiterWithReconstruction(t *testing.T) {
	applier, _, reg, _ := newTestApplier(t, true)
	tr := testTimeRange()
	id := testUUID(t, 1)

	reader := newFakeBackendReader()
	reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(id, "tenant-a", tr, vparquet3.VersionString)))

	// Exactly one fetch's worth of budget, with a zero refill rate ("a zero
	// Limit allows no events", vendor/golang.org/x/time/rate/rate.go) -- the
	// SAME *rate.Limiter type NewReconstructionQueue takes (reconstruction.
	// go), standing in for the shared cell-wide budget.
	limiter := rate.NewLimiter(0, int(estimatedBlockColumnBytes))

	rec := NewReconciler(reg, applier, reader, permissiveReconciliationConfig(), zeroLag, limiter, applier.metrics, log.NewNopLogger())
	rec.Pass(context.Background(), "tenant-a")

	require.Equal(t, BlockLiveUnsupportedEncoding, mustState(t, reg, id), "sanity: the repair must actually have run and drawn from the limiter")

	// The repair above must have spent the limiter's entire budget: a
	// subsequent attempt to draw the SAME estimatedBlockColumnBytes cost --
	// exactly what reconstruction.go's fetchAndApplyOne draws before every
	// one of ITS OWN fetches -- must now fail immediately, proving the
	// budget is genuinely shared rather than doubled by an independent
	// limiter.
	assert.False(t, limiter.AllowN(time.Now(), int(estimatedBlockColumnBytes)), "the repair fetch must have consumed the shared limiter's entire budget")
}

// TestReconciliation_Run_ProcessesAllTenantsAndStopsOnCancellation: Run
// ticks across every tenant the backend knows about, and returns promptly
// once ctx is cancelled -- the same lifecycle contract every other
// background loop in this package (sweep.go, reconstruction.go, worker.go,
// consumer.go) is held to.
func TestReconciliation_Run_ProcessesAllTenantsAndStopsOnCancellation(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	applier, _, reg, _ := newTestApplier(t, true)
	tr := testTimeRange()
	idA, idB := testUUID(t, 1), testUUID(t, 2)

	reader := newFakeBackendReader()
	reader.setTenantIndex("tenant-a", staleTenantIndex(blockMetaFixture(idA, "tenant-a", tr, vparquet3.VersionString)))
	reader.setTenantIndex("tenant-b", staleTenantIndex(blockMetaFixture(idB, "tenant-b", tr, vparquet3.VersionString)))

	cfg := ReconciliationConfig{Period: 5 * time.Millisecond, LagGate: time.Hour}
	rec := NewReconciler(reg, applier, reader, cfg, zeroLag, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = rec.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		_, okA := reg.LookupUUID(idA)
		_, okB := reg.LookupUUID(idB)
		return okA && okB
	}, 2*time.Second, 5*time.Millisecond, "Run must reconcile every tenant the backend reports")

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return promptly after context cancellation")
	}

	goleak.VerifyNone(t, opts)
}
