package bloomgateway

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/tempodb/backend"
)

// tombstone is one Deleted block captured in a single Pass's registry
// snapshot (step 1 below) -- exactly the bookkeeping the tombstone-reclaim
// check (step 3) needs, without a second registry read.
type tombstone struct {
	uuid      backend.UUID
	handle    Handle
	deletedAt time.Time
}

// Sweeper is the background incremental leaf sweep and tombstone
// reclamation (DESIGN.md § Garbage collection, invariant #9, §7): entries
// referencing deleted blocks are exact but lazy -- they remain in leaves
// until this walks the directory and removes them, and a deleted block's
// registry entry itself is only reclaimed once a pass confirms zero
// remaining leaf entries AND the delete is older than the replay horizon.
type Sweeper struct {
	dir     *Directory
	reg     *Registry
	tenants *TenantSet
	cfg     SweepConfig
	metrics *metrics
	logger  log.Logger
}

// NewSweeper builds a Sweeper over dir/reg/tenants.
func NewSweeper(dir *Directory, reg *Registry, tenants *TenantSet, cfg SweepConfig, m *metrics, logger log.Logger) *Sweeper {
	return &Sweeper{
		dir:     dir,
		reg:     reg,
		tenants: tenants,
		cfg:     cfg,
		metrics: m,
		logger:  logger,
	}
}

// PassStats summarizes one Pass call, returned for logging/metrics and for
// tests to assert on directly.
type PassStats struct {
	EntriesVisited      int
	EntriesRemoved      int
	TombstonesReclaimed int
}

// Run paces repeated full passes to run roughly every cfg.FullPassPeriod
// (DESIGN.md § Garbage collection: "run continuously with a full-pass
// period of ~1-2h"), blocking until ctx is done. Pass itself is a
// synchronous, complete walk of the whole directory (see its own doc
// comment) -- DESIGN.md's own cost model says a full pass is "seconds of
// one core" even at the reference ~2.5x10^9-entries/instance scale, so
// pacing at the PASS level (run one, then wait) is what keeps the sweep's
// CPU cost negligible without needing to fragment a single pass across
// multiple ticks: every leaf's own stripe lock is still only ever held for
// one slot's critical section at a time (directory.go's Range), which is
// the actual "never block writers for long" property DESIGN.md cares
// about.
func (s *Sweeper) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		stats := s.Pass(ctx)
		level.Debug(s.logger).Log("msg", "bloomgateway: sweep pass complete", "entries_visited", stats.EntriesVisited, "entries_removed", stats.EntriesRemoved, "tombstones_reclaimed", stats.TombstonesReclaimed)

		timer := time.NewTimer(s.cfg.FullPassPeriod)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// Pass runs exactly one full pass synchronously (so tests/benchmarks don't
// have to wait wall-clock FullPassPeriod). Algorithm (DESIGN.md § Garbage
// collection):
//
//  1. Snapshot the set of currently-Deleted handles from the registry
//     ONCE, at the very start -- this is what makes every later step in
//     THIS pass safe: tombstone terminality (invariant #4, §7) guarantees
//     no NEW entries for an already-Deleted handle can ever appear, so a
//     handle captured here is confirmed garbage for the rest of this
//     pass's duration. A block whose Delete races in AFTER this snapshot
//     (whether that's mid-walk or simply between this Pass call and the
//     next) is a genuinely different case from this pass's point of view:
//     it is absent from the snapshot, so this pass leaves its entries
//     untouched and its tombstone unreclaimed -- deferred entirely to
//     whichever future pass's OWN snapshot first captures it (named test:
//     TestSweep_TombstoneReclamationRequiresFullPassAndReplayHorizon).
//  2. For every COMPLETE leaf, clone it, RemoveWhere(not in the step-1
//     set), and Swap the filtered clone back in if anything was removed
//     (§ Mutation modes' copy-rewire-place-back). Constructing leaves are
//     left alone: they are still accumulating a concurrent backfill, and
//     any deleted-block garbage in them is harmless until they flip
//     complete and a later pass visits them.
//  3. For every block that was ALREADY Deleted at step 1's snapshot AND
//     whose DeletedAt is older than cfg.ReplayHorizon, Registry.Reclaim
//     it. A block whose delete is too recent keeps its tombstone for a
//     later pass -- this is what stops a stale in-flight Add on the topic
//     from ever resurrecting a reclaimed tombstone (§7 invariant #9).
//  4. Drop now-empty A_T buckets for every tenant this instance currently
//     knows of any block for (derived from the registry's own TenantID
//     field, not a new TenantSet enumeration method -- see the comment at
//     its call site below for why).
func (s *Sweeper) Pass(ctx context.Context) PassStats {
	passStart := time.Now()

	deletedHandles := make(map[Handle]struct{})
	var tombstones []tombstone
	tenantIDs := make(map[string]struct{})

	s.reg.Range(func(b *Block) bool {
		tenantIDs[b.TenantID] = struct{}{}
		if b.State == BlockDeleted {
			deletedHandles[b.Handle] = struct{}{}
			tombstones = append(tombstones, tombstone{uuid: b.UUID, handle: b.Handle, deletedAt: b.DeletedAt})
		}
		return true
	})

	keep := func(h Handle) bool {
		_, deleted := deletedHandles[h]
		return !deleted
	}

	var stats PassStats
	walkComplete := true
	s.dir.Range(func(idx uint32, _ LeafState) bool {
		if ctx.Err() != nil {
			walkComplete = false
			return false
		}

		// CompactLeaf filters the live leaf in place under a single stripe
		// write lock. This is load-bearing, not a style choice: the earlier
		// CloneLeaf-then-Swap shape had a lost-update race — a concurrent
		// InsertLive landing between the clone (lock released) and the Swap
		// (lock re-taken) wrote into the live leaf, and the Swap of the
		// stale filtered copy silently discarded it, dropping a live entry
		// (a false negative — the forbidden error class). Doing the
		// read-modify-write atomically under the write lock closes that
		// window: InsertLive takes the same lock, so it either happens fully
		// before or fully after this compaction, never lost across it.
		visited, removed, compacted := s.dir.CompactLeaf(idx, keep)
		if !compacted {
			return true // nil/constructing slot: left for a later pass
		}
		stats.EntriesVisited += visited
		stats.EntriesRemoved += removed
		return true
	})

	// Tombstone reclamation is only safe once THIS pass has confirmed zero
	// remaining leaf entries for the block — which requires the directory
	// walk above to have actually visited every leaf. If ctx cancellation
	// (shutdown) cut the walk short, some leaves went unvisited, so their
	// entries for a deleted block may still be present; reclaiming the
	// registry tombstone now would reopen the resurrection-by-replay hole
	// (§7 invariant #9). Defer all reclamation to a future, complete pass.
	if !walkComplete {
		s.metrics.sweepPassDurationSeconds.Observe(time.Since(passStart).Seconds())
		s.metrics.sweepEntriesRemovedTotal.Add(float64(stats.EntriesRemoved))
		return stats
	}

	now := time.Now()
	for _, ts := range tombstones {
		if now.Sub(ts.deletedAt) < s.cfg.ReplayHorizon {
			continue // not yet past the replay horizon; try again next pass
		}
		s.reg.Reclaim(ts.uuid)
		stats.TombstonesReclaimed++
	}

	// tenantIDs is derived from the registry (every tenant with at least
	// one known block, of any state) rather than a new TenantSet
	// enumeration method: this keeps sweep.go self-contained within this
	// WP's own file (tenant.go stays untouched here, unlike the WP17
	// TenantSet.Export/Import addendum, which the plan explicitly calls
	// for). The one gap this leaves -- a tenant whose last block has
	// already been fully reclaimed keeps any already-empty A_T buckets
	// forever -- is harmless: an empty roaring bitmap sitting in a map
	// costs a handful of bytes, dwarfed by everything else this design
	// budgets memory for, and DESIGN.md's own "sweep is where tenant
	// deletion converges" already assigns whole-tenant A_T teardown to
	// TenantSet.DropTenant, a different, explicit signal this WP does not
	// own.
	for tenantID := range tenantIDs {
		s.tenants.DropEmptyBuckets(tenantID)
	}

	s.metrics.sweepPassDurationSeconds.Observe(time.Since(passStart).Seconds())
	s.metrics.sweepEntriesRemovedTotal.Add(float64(stats.EntriesRemoved))

	return stats
}
