// This file implements the reconciliation loop (DESIGN.md § Reconciliation):
// a periodic per-tenant diff of the object-store tenant index against this
// instance's block registry, healing every missed-event class -- dropped
// publishes, lost chunks, missed Deletes, producer bugs -- within one
// reconciliation period (§ Reconciliation: "This heals every missed-event
// class ... within one reconciliation period").
//
// Two independent diffs, each gated differently:
//
//   - repair-Add: a block present in the tenant index that this instance's
//     registry either doesn't know about at all, or knows only as
//     BlockPending (AMENDMENT A2 -- heals a block stuck Pending forever from
//     a dropped chunk, or a restart that lost its in-memory chunk-arrival
//     bitset, which is deliberately NOT part of registry.Block nor persisted
//     in a snapshot; see snapshot.go). Lag-gated: suppressed entirely while
//     lagFn() exceeds cfg.LagGate, because an in-flight block that merely
//     *looks* missing/stuck during replay would otherwise trigger a
//     redundant object-store fetch of exactly the data replay is about to
//     deliver for free (§ Reconciliation).
//
//   - synthesize-Delete: a block this instance's registry holds as
//     BlockLive or BlockLiveUnsupportedEncoding that is absent from the
//     tenant index. Never lag-gated (§ Reconciliation: "Delete synthesis is
//     unaffected -- it is correct early and costs no reads"). Fires only
//     after the missing-from-index condition has PERSISTED across the
//     grace window, tracked per-UUID across separate Pass calls (the one
//     piece of cross-pass memory this file holds) -- a block merely racing
//     an in-flight compaction/retention cycle must not be synthesize-
//     deleted on the very first pass that happens to observe it missing.
//
// Both diffs share reconciliationGraceWindow and, for repair-Add, the same
// cell-wide *rate.Limiter and FetchAndApplyBlockColumn helper
// reconstruction.go (WP18) defines -- repair-Adds and reconstruction Adds
// are the SAME code path applying against the SAME shared object-store
// read-rate budget, never a duplicated implementation or a second,
// independent budget (§ Reconciliation: "repair fetches share the cell-wide
// reconstruction rate limit").
package bloomgateway

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/time/rate"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// reconciliationGraceWindow is DESIGN.md's "~2x poll interval" (§
// Reconciliation), used by both diffs (Pass's own doc comment below says so
// explicitly: "past the SAME grace"), but evaluated two different ways:
//
//   - repair-Add gates on how stale the tenant INDEX SNAPSHOT itself is
//     (idx.CreatedAt) -- a single-pass-computable signal, "to avoid racing
//     in-flight Adds" for a block that only just landed in a very fresh
//     index read.
//   - synthesize-Delete gates on how long THIS RECONCILER has continuously
//     observed a UUID missing from the index, tracked in missingSince
//     across passes -- there is no per-block backend timestamp answering
//     "how long has this UUID been absent from the index" the way
//     idx.CreatedAt answers "how fresh is this snapshot", so that half
//     necessarily needs cross-pass memory (see missingSince below); this is
//     the ONE piece of cross-pass state in this file.
//
// DESIGN.md frames this off blocklist_poll (5 min default; § Reconstruction
// step 2 cites the same figure); ReconciliationConfig (config.go) has no
// separate poll-interval field -- this WP's own file list is
// reconciliation.go only, config.go belongs to WP3 -- so this is a fixed
// constant, exactly like reconstruction.go's reconstructionRewindMargin is
// for an analogous "avoid racing in-flight state" margin, rather than a new
// operator knob.
const reconciliationGraceWindow = 10 * time.Minute

// ReconciliationPassStats summarizes one Pass call, returned for
// logging/metrics and for tests to assert on directly -- the same role
// sweep.go's PassStats and reconstruction.go's BatchStats serve for their
// own loops.
//
// Deviation from the implementation plan's exported-API sketch, reported
// per the harness's own instructions: the plan names this type "PassStats"
// verbatim, but sweep.go (WP16, already landed) already declares a
// different type of that exact name for Sweeper.Pass's own return value --
// an unqualified "PassStats" would be a duplicate declaration in this
// package. reconstruction.go (WP18) avoided the same collision by naming
// its own equivalent type BatchStats rather than PassStats; this type is
// named ReconciliationPassStats for the same reason, following that
// precedent rather than renaming WP16's already-landed, differently-shaped
// type.
type ReconciliationPassStats struct {
	RepairAdds         int
	SynthesizedDeletes int
}

// Reconciler is the periodic tenant-index-vs-registry diff loop (DESIGN.md §
// Reconciliation). It holds no *TenantSet or *Directory reference of its
// own: every effect it has on either flows exclusively through applier --
// ApplyDelete for synthesize-Delete, and FetchAndApplyBlockColumn (which
// itself calls applier.ApplyAddChunk/CommitUnsupportedEncoding) for
// repair-Add -- the same single apply path every other write, live or
// synthetic, in this package goes through.
type Reconciler struct {
	reg           *Registry
	applier       *Applier
	backendReader backend.Reader
	cfg           ReconciliationConfig
	lagFn         func() time.Duration
	limiter       *rate.Limiter
	metrics       *metrics
	logger        log.Logger

	mu sync.Mutex
	// missingSince tracks, per tenant then per UUID, the wall-clock time
	// this Reconciler FIRST observed that UUID Live/LiveUnsupportedEncoding
	// in the registry but absent from the tenant index. A UUID is dropped
	// from its tenant's map the moment it is no longer observed missing
	// (reappeared in the index, left the registry's live view some other
	// way, or was just synthesize-deleted) -- so a stale timestamp can
	// never outlive the condition it was tracking.
	//
	// Nested per-tenant (matching the exported-API sketch's own "per-tenant
	// state" phrasing) rather than one flat map[backend.UUID]time.Time:
	// Pass only ever enumerates and prunes ITS OWN tenantID's blocks, so
	// nesting is what keeps one tenant's Pass call from ever touching
	// another tenant's tracked entries, even though backend.UUID values are
	// in fact globally unique on their own.
	missingSince map[string]map[backend.UUID]time.Time
}

// NewReconciler builds a Reconciler. limiter must be the SAME *rate.Limiter
// instance passed to NewReconstructionQueue (constructed once by the
// top-level orchestrator) -- passing each its own limiter built from the
// same config value would silently double the effective cell-wide
// object-store read-rate budget, since two independent *rate.Limiter
// objects share no state (see reconstruction.go's identical note on
// ReconstructionQueue.limiter).
func NewReconciler(reg *Registry, applier *Applier, backendReader backend.Reader, cfg ReconciliationConfig, lagFn func() time.Duration, limiter *rate.Limiter, m *metrics, logger log.Logger) *Reconciler {
	return &Reconciler{
		reg:           reg,
		applier:       applier,
		backendReader: backendReader,
		cfg:           cfg,
		lagFn:         lagFn,
		limiter:       limiter,
		metrics:       m,
		logger:        logger,
		missingSince:  make(map[string]map[backend.UUID]time.Time),
	}
}

// Run ticks every cfg.Period, running one Pass per known tenant, until ctx
// is done. This is a long-lived background loop (like sweep.go's Run and
// reconstruction.go's Run), not a one-shot operation a caller awaits.
func (r *Reconciler) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		r.runOnce(ctx)

		if !sleepCtx(ctx, r.cfg.Period) {
			return ctx.Err()
		}
	}
}

// runOnce runs one Pass per tenant currently known to the backend. A
// failure enumerating tenants is logged and retried next tick, not fatal to
// the loop -- reconciliation is itself a safety net, so its own transient
// failures must not stop it from trying again next period.
func (r *Reconciler) runOnce(ctx context.Context) {
	tenantIDs, err := r.backendReader.Tenants(ctx)
	if err != nil {
		level.Warn(r.logger).Log("msg", "bloomgateway: reconciliation: list tenants failed; will retry next period", "err", err)
		return
	}

	for _, tenantID := range tenantIDs {
		if ctx.Err() != nil {
			return
		}

		stats := r.Pass(ctx, tenantID)
		if stats.RepairAdds > 0 || stats.SynthesizedDeletes > 0 {
			// Nonzero steady-state repair rates indicate a broken producer,
			// not a working safety net (§ Reconciliation) -- worth an Info
			// line, not just the counter, so an operator scanning logs
			// during an incident sees which tenant(s) are affected.
			level.Info(r.logger).Log("msg", "bloomgateway: reconciliation pass repaired blocks", "tenant", tenantID, "repair_adds", stats.RepairAdds, "synthesized_deletes", stats.SynthesizedDeletes)
		}
	}
}

// Pass runs one tenant's reconciliation synchronously (so tests don't have
// to wait wall-clock cfg.Period) -- the same "Pass alongside Run" shape
// sweep.go and reconstruction.go (RunBatch) both use for the identical
// reason: a background-loop-only API gives tests no deterministic way to
// assert on one diff's outcome.
//
// A TenantIndex read failure is logged and this tenant is skipped entirely
// for this pass (zero-value ReconciliationPassStats) -- there is nothing to
// diff against without it, and the next Run tick tries again.
func (r *Reconciler) Pass(ctx context.Context, tenantID string) ReconciliationPassStats {
	var stats ReconciliationPassStats

	idx, err := r.backendReader.TenantIndex(ctx, tenantID)
	if err != nil {
		level.Warn(r.logger).Log("msg", "bloomgateway: reconciliation: read tenant index failed; skipping tenant this pass", "tenant", tenantID, "err", err)
		return stats
	}

	stats.RepairAdds = r.repairAdds(ctx, tenantID, idx)

	inIndex := make(map[backend.UUID]struct{}, len(idx.Meta))
	for _, meta := range idx.Meta {
		inIndex[meta.BlockID] = struct{}{}
	}
	stats.SynthesizedDeletes = r.synthesizeDeletes(tenantID, inIndex)

	return stats
}

// repairAdds implements the repair-Add half of Pass (AMENDMENT A2): every
// block in idx.Meta that this instance's registry either doesn't know at
// all, or knows only as BlockPending (never one already Live,
// LiveUnsupportedEncoding, or Deleted -- those are not repair-eligible), is
// backfilled via the SAME FetchAndApplyBlockColumn helper reconstruction.go
// (WP18) uses for its own column pass -- asserted directly by
// TestReconciliation_RepairsStuckPendingBlock and
// TestReconciliation_SharesRateLimiterWithReconstruction, not just assumed
// by calling it.
//
// Two independent gates, checked once per call (not per block): lag-gating
// (repair-Adds are SKIPPED ENTIRELY while lagFn() exceeds cfg.LagGate --
// Delete synthesis is never lag-gated, see synthesizeDeletes) and the
// index-snapshot-age grace (reconciliationGraceWindow's own doc comment
// explains why this half uses idx.CreatedAt rather than cross-pass memory).
func (r *Reconciler) repairAdds(ctx context.Context, tenantID string, idx *backend.TenantIndex) int {
	if lag := r.lagFn(); lag > r.cfg.LagGate {
		level.Debug(r.logger).Log("msg", "bloomgateway: reconciliation: repair-adds suppressed: consumer lag exceeds grace threshold", "tenant", tenantID, "lag", lag, "gate", r.cfg.LagGate)
		return 0
	}
	if age := time.Since(idx.CreatedAt); age < reconciliationGraceWindow {
		level.Debug(r.logger).Log("msg", "bloomgateway: reconciliation: repair-adds skipped: tenant index snapshot too fresh to trust a diff against", "tenant", tenantID, "age", age, "grace", reconciliationGraceWindow)
		return 0
	}

	repaired := 0
	for _, meta := range idx.Meta {
		if ctx.Err() != nil {
			return repaired
		}

		state, known := r.reg.State(meta.BlockID)
		if known && state != BlockPending {
			continue // Live, LiveUnsupportedEncoding, or Deleted: not repair-eligible (AMENDMENT A2)
		}

		if err := r.limiter.WaitN(ctx, estimatedBlockColumnBytes); err != nil {
			level.Warn(r.logger).Log("msg", "bloomgateway: reconciliation: rate limiter wait failed; aborting this pass's remaining repairs", "tenant", tenantID, "block", meta.BlockID, "err", err)
			return repaired
		}

		applied, ferr := FetchAndApplyBlockColumn(ctx, r.backendReader, meta, r.applier, r.metrics)
		if ferr != nil {
			level.Warn(r.logger).Log("msg", "bloomgateway: reconciliation: repair-add failed", "tenant", tenantID, "block", meta.BlockID, "err", ferr)
			continue
		}
		if applied {
			repaired++
			r.metrics.reconciliationRepairsTotal.WithLabelValues("add").Inc()
		}
	}
	return repaired
}

// synthesizeDeletes implements the Delete-synthesis half of Pass: every
// block this instance's registry holds as BlockLive or
// BlockLiveUnsupportedEncoding (AMENDMENT A1 / §7 invariant #10 -- a
// LiveUnsupportedEncoding block missing from the index is synthesize-
// deleted exactly like a Live one, never repair-looped: it is not
// repair-eligible in repairAdds above, since its registry State is neither
// "unknown" nor BlockPending) for tenantID, but absent from inIndex, is a
// synthesize-Delete CANDIDATE. Never lag-gated (§ Reconciliation: "Delete
// synthesis is unaffected -- it is correct early and costs no reads").
//
// A candidate only actually fires once missingSince shows it has been
// continuously observed missing for at least reconciliationGraceWindow,
// persisted across separate Pass calls -- never on the pass that first
// observes it (that pass only starts the clock). This is the file's one
// piece of cross-pass memory; getting it wrong has two distinct failure
// modes (both covered by
// TestReconciliation_SynthesizeDeleteRequiresPersistenceAcrossPasses):
// re-deriving "first observed" from scratch every call (comparing against
// the SAME now used for the timestamp) can never synthesize anything, since
// the elapsed duration is always exactly zero; skipping the persistence
// check entirely acts one pass too early -- exactly the
// racing-in-flight-compaction/retention scenario the grace window exists to
// protect against.
func (r *Reconciler) synthesizeDeletes(tenantID string, inIndex map[backend.UUID]struct{}) int {
	var missingNow []backend.UUID
	r.reg.Range(func(b *Block) bool {
		if b.TenantID != tenantID {
			return true
		}
		if b.State != BlockLive && b.State != BlockLiveUnsupportedEncoding {
			return true
		}
		if _, present := inIndex[b.UUID]; !present {
			missingNow = append(missingNow, b.UUID)
		}
		return true
	})

	r.mu.Lock()
	defer r.mu.Unlock()

	tracked := r.missingSince[tenantID]
	if tracked == nil {
		tracked = make(map[backend.UUID]time.Time)
	}

	// Forget any previously tracked UUID no longer observed missing this
	// pass (reappeared in the index, or left the registry's live view) --
	// otherwise a stale timestamp could linger and later apply to an
	// unrelated recurrence.
	stillMissing := make(map[backend.UUID]struct{}, len(missingNow))
	for _, uuid := range missingNow {
		stillMissing[uuid] = struct{}{}
	}
	for uuid := range tracked {
		if _, ok := stillMissing[uuid]; !ok {
			delete(tracked, uuid)
		}
	}

	now := time.Now()
	synthesized := 0
	for _, uuid := range missingNow {
		firstSeen, ok := tracked[uuid]
		if !ok {
			tracked[uuid] = now // first observation: start the clock, do NOT act this pass
			continue
		}
		if now.Sub(firstSeen) < reconciliationGraceWindow {
			continue // still within grace: a single pass's snapshot alone must never act
		}

		if err := r.applier.ApplyDelete(&tempopb.BloomGatewayDelete{BlockId: uuid.String()}); err != nil {
			level.Warn(r.logger).Log("msg", "bloomgateway: reconciliation: synthesize-delete failed", "tenant", tenantID, "block", uuid, "err", err)
			continue
		}
		delete(tracked, uuid)
		synthesized++
		r.metrics.reconciliationRepairsTotal.WithLabelValues("delete").Inc()
	}

	if len(tracked) == 0 {
		delete(r.missingSince, tenantID)
	} else {
		r.missingSince[tenantID] = tracked
	}

	return synthesized
}
