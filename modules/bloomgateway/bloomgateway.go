// This file implements the top-level orchestrator: BloomGateway, the
// services.Service every prior file in this package is wired into (DESIGN.md
// throughout; implementation plan WP20). It owns exactly one piece of logic
// no other file does -- reconciling loaded-or-absent snapshot state against
// the CURRENT ring on startup (§ Snapshots: "owned-range reconciliation
// against the current ring") -- plus (see the addendum below) keeping that
// reconciliation correct for the rest of the instance's life as the ring
// topology changes around it.
//
// Deviation from the implementation plan, reported prominently per the
// harness's own instructions: the plan's starting() sequence (§3 WP20) is
// explicit that it is "the one place this behavior is specified", and it
// only describes a ONE-TIME snapshot-vs-ring reconciliation at cold start.
// That is not sufficient to make this WP's own named deliverable -- the
// multi-instance scale-out test ("new instance joins, acquires, reconstructs,
// previous owner sheds") -- true: the PREVIOUS owner in that scenario is
// already running and never restarts, so a startup-only reconciliation never
// re-fires for it. DESIGN.md's own § Scaling and resharding, Scale-out step 4
// ("Previous owners shed the moved leaves as their ring view updates") also
// describes ongoing behavior a one-shot startup step cannot provide. This
// file therefore adds a small, continuously-running ownership-watch loop
// (runOwnershipWatch, below) alongside the plan's literal starting()
// sequence -- the smallest addition that makes the assigned test scenario
// (and DESIGN.md's own narrative) actually true, not a reinterpretation of
// what starting() itself does.
//
// A second, smaller deviation: the plan's step 3 says to start "Consumer,
// WorkerPool, Sweeper.Run, ReconstructionQueue.Run, Reconciler.Run as one
// services.Manager, watched by a FailureWatcher". Only the ring's own
// Lifecycler+Ring (RingManager.Services()) actually implement
// services.Service; Consumer/WorkerPool/Sweeper/ReconstructionQueue/
// Reconciler do not (they expose Start/Stop or a blocking Run(ctx) instead),
// so they cannot literally be passed to services.NewManager. This file
// starts them directly (Consumer.Start/WorkerPool.Start return immediately
// after spawning their own goroutines; Sweeper.Run/ReconstructionQueue.Run/
// Reconciler.Run are launched in their own goroutines against this
// instance's own long-lived, New()-constructed context) and logs any
// unexpected (non-context-cancellation) error each returns -- preserving the
// plan's INTENT (start them together, watch for failures) without a type
// mismatch. The genuine services.Manager/FailureWatcher machinery is used
// for the ring's own two services exactly as backend-worker's template does.
package bloomgateway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// ringOp is the operation this package resolves leaf ownership under: ACTIVE
// instances only, never extending the replica set (RF=1, no analogue of
// extension needed) -- the same shape backend-worker's own package-level
// ringOp uses, and the one OwnedLeafRanges' own doc comment already
// describes as "this package's ringOp".
var ringOp = ring.NewOp([]ring.InstanceState{ring.ACTIVE}, nil)

// ErrStarting and ErrStopping are readyErr's two non-nil sentinel values
// (live-store's own atomic.Pointer[error] pattern, module-wiring report
// recommendation #2): readyErr starts at &ErrStarting in New(), flips to nil
// once this instance's startup sequence (ring join, plus a snapshot load or
// first reconstruction enqueue -- AMENDMENT A3's surviving "(snapshot loaded
// OR reconstruction enqueued)" clause) has completed, and is set to
// &ErrStopping at the start of stopping(). See CheckReady's own doc comment
// for why the REST of AMENDMENT A3's gate (ring health, reconstruction
// progress) is deliberately NOT folded into this same one-shot pointer.
var (
	ErrStarting = errors.New("bloomgateway: starting")
	ErrStopping = errors.New("bloomgateway: stopping")
)

const (
	// ringActiveWaitTimeout bounds starting()'s wait for this instance to
	// observe itself ACTIVE in its own ring view. Not exposed via Config:
	// neither DESIGN.md nor the plan calls for a knob here, matching how
	// reconciliation.go's reconciliationGraceWindow and reconstruction.go's
	// reconstructionRewindMargin are fixed constants rather than new knobs
	// where the plan doesn't ask for one.
	ringActiveWaitTimeout = 30 * time.Second

	// ringStabilityMinDuration/MaxWaiting bound WaitRingStability's
	// cold-start settle window (DESIGN.md § Availability model's readiness
	// gate; backend-worker's own precedent: "in the event of a cluster cold
	// start... it's better to just wait the ring stability for a short
	// time"). A timeout here is logged and starting() proceeds anyway
	// (matching backend-worker), not treated as fatal.
	ringStabilityMinDuration = 500 * time.Millisecond
	ringStabilityMaxWaiting  = 5 * time.Second

	// consumerLagCheckTimeout bounds the on-demand broker round-trip
	// consumerLag makes on behalf of the Reconciler's lagFn (see its own
	// doc comment below for why this is computed on demand rather than
	// tracked by a background loop).
	consumerLagCheckTimeout = 10 * time.Second

	// statsRefreshInterval paces runStatsLoop (below), which populates the
	// state gauges (blocks_live, entries_total, owned_leaves, snapshot_age_
	// seconds, miss_fp_rate_estimate) from cheap, already-maintained
	// sources -- atomic counters on Directory/Registry and the last-
	// snapshot timestamp -- never a directory walk. Not exposed via
	// Config: neither DESIGN.md nor the plan calls for a knob here, and
	// this loop's cost is independent of cell size (see refreshStats), so
	// a fixed cadence is enough, matching this file's own
	// ownershipReconcileInterval precedent.
	statsRefreshInterval = 15 * time.Second
)

// ownershipReconcileInterval paces runOwnershipWatch (see the package doc
// comment above). A var, not a const: this package's own tests override it
// to a much smaller value so multi-instance scale-out convergence doesn't
// have to wait out a production-sized interval; production leaves it at its
// default. Not exposed via Config -- this is this WP's own addition, not a
// plan-specified knob.
var ownershipReconcileInterval = time.Second

// BloomGateway is the top-level bloom gateway service (DESIGN.md throughout).
// It embeds services.Service (assigned in New(), matching every other
// module in this repo: module-wiring report convention "services.Service is
// embedded, not implemented by hand") and wires together every prior file in
// this package.
type BloomGateway struct {
	services.Service

	cfg        Config
	instanceID string
	logger     log.Logger
	metrics    *metrics

	seed            []byte
	seedFingerprint uint64

	ringManager *RingManager

	dir     *Directory
	reg     *Registry
	tenants *TenantSet
	applier *Applier

	consumer            *Consumer
	workerPool          *WorkerPool
	sweeper             *Sweeper
	snapshotter         *Snapshotter
	reconstructionQueue *ReconstructionQueue
	reconciler          *Reconciler
	server              *Server

	backendReader backend.Reader
	limiter       *rate.Limiter

	// lastOwnedRanges is runOwnershipWatch's own diff baseline (and is
	// seeded once by reconcileStartup) -- touched only from that one
	// goroutine, so it needs no lock of its own.
	lastOwnedRanges []LeafRange

	// lastSnapshotUnixNano is the UnixNano of this instance's most recent
	// successful snapshot Load (reconcileStartup) or Save (saveSnapshot) --
	// the snapshot_age_seconds gauge's source (§ Metrics: "age of the most
	// recently loaded or saved snapshot"). Zero is a sentinel for "no
	// snapshot yet this process" (matches Handle's InvalidHandle=0 and
	// BlockState's BlockPending-is-zero-value conventions elsewhere in this
	// package): a real UnixNano timestamp for "now" is never legitimately
	// zero, so the two cases are unambiguous. refreshStats reports NaN for
	// the zero case rather than 0 -- 0 would read as "just snapshotted",
	// the opposite of the truth, and would also make a naive `snapshot_age_
	// seconds > threshold` alert wrongly stay quiet forever on an instance
	// that has NEVER produced a snapshot (e.g. snapshotting disabled via
	// Config.Snapshot.Interval <= 0) instead of the intended "no data yet"
	// treatment a NaN gives that same alert expression.
	lastSnapshotUnixNano atomic.Int64

	// ctx/cancel is this instance's own long-lived background context
	// (live-store's own New()-time context.WithCancel pattern), decoupled
	// from whatever ctx dskit's services.Service machinery passes to
	// starting()/running() for any one phase -- every background loop
	// started in starting() runs against this ctx and is torn down in
	// stopping() by calling cancel and waiting on bgWG.
	ctx    context.Context
	cancel context.CancelFunc
	bgWG   sync.WaitGroup

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	readyErr atomic.Pointer[error]
}

var _ tempopb.BloomGatewayServer = (*BloomGateway)(nil)

// New builds a BloomGateway. It constructs every prior file's component
// (ring, directory, registry, tenant set, applier, consumer, worker pool,
// sweeper, snapshotter, reconstruction queue, reconciler, query server) and
// wires them together, but starts nothing -- StartAsync (dskit's
// services.Service) drives starting()/running()/stopping() below.
//
// instanceID must match the StatefulSet pod-naming shape "name-N"
// (NewRingManager's own requirement, surfaced here as an ordinary
// constructor error rather than a later, less specific failure).
func New(cfg Config, instanceID string, backendReader backend.Reader, logger log.Logger, reg prometheus.Registerer) (*BloomGateway, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("bloomgateway: invalid config: %w", err)
	}

	m := newMetrics(reg)

	ringManager, err := NewRingManager(cfg.Ring, instanceID, cfg.Ring.InstanceZone, cfg.NumTokens, logger, reg)
	if err != nil {
		return nil, fmt.Errorf("bloomgateway: %w", err)
	}

	dir := NewDirectory(cfg.D)
	registry := NewRegistry()
	tenants := NewTenantSet()

	seed := []byte(cfg.Seed.String())
	hashSeed := HashSeed(seed)
	seedFingerprint := SeedFingerprint(seed)

	applier := NewApplier(dir, registry, tenants, hashSeed, cfg.D, cfg.F, m)

	consumer, err := NewConsumer(cfg.Kafka, instanceID, cfg.Queue.MaxBytes, logger, reg)
	if err != nil {
		return nil, fmt.Errorf("bloomgateway: %w", err)
	}
	workerPool := NewWorkerPool(cfg.Queue.Workers, consumer.Records(), applier, logger, m)
	sweeper := NewSweeper(dir, registry, tenants, cfg.Sweep, m, logger)
	snapshotter := NewSnapshotter(m)

	// The shared cell-wide object-store read-rate budget: constructed
	// exactly ONCE here and passed to both NewReconstructionQueue and
	// NewReconciler (their own constructor doc comments explain why a
	// second, independently-built limiter from the same config value would
	// silently double the effective rate). Burst is floored at
	// estimatedBlockColumnBytes (reconstruction.go's own documented
	// requirement: "the limiter's burst must be >= estimatedBlockColumnBytes,
	// or every real... call this queue makes would fail immediately").
	burst := int(cfg.Reconstruction.RateLimitBytesPerSecond)
	if burst < estimatedBlockColumnBytes {
		burst = estimatedBlockColumnBytes
	}
	limiter := rate.NewLimiter(rate.Limit(cfg.Reconstruction.RateLimitBytesPerSecond), burst)

	reconstructionQueue := NewReconstructionQueue(dir, applier, consumer, backendReader, cfg.Reconstruction, limiter, m, logger)
	server := NewServer(dir, registry, tenants, seed, cfg.D, cfg.F, m)

	ctx, cancel := context.WithCancel(context.Background())

	g := &BloomGateway{
		cfg:                 cfg,
		instanceID:          instanceID,
		logger:              logger,
		metrics:             m,
		seed:                seed,
		seedFingerprint:     seedFingerprint,
		ringManager:         ringManager,
		dir:                 dir,
		reg:                 registry,
		tenants:             tenants,
		applier:             applier,
		consumer:            consumer,
		workerPool:          workerPool,
		sweeper:             sweeper,
		snapshotter:         snapshotter,
		reconstructionQueue: reconstructionQueue,
		server:              server,
		backendReader:       backendReader,
		limiter:             limiter,
		ctx:                 ctx,
		cancel:              cancel,
	}
	// NewReconciler's lagFn needs a method value on g, so g must already
	// exist -- reconciler is therefore constructed after, and assigned onto
	// g directly, rather than via the struct literal above.
	g.reconciler = NewReconciler(registry, applier, backendReader, cfg.Reconciliation, g.consumerLagFn, limiter, m, logger)

	g.readyErr.Store(&ErrStarting)
	g.Service = services.NewBasicService(g.starting, g.running, g.stopping)
	return g, nil
}

// starting implements the plan's numbered sequence (§3 WP20), plus this
// file's own ownership-watch addition (package doc comment above):
//
//  1. Start the ring sub-services; block on ring.WaitInstanceState(ACTIVE) +
//     ring.WaitRingStability (§ Availability model's readiness gate).
//  2. reconcileStartup: attempt Snapshotter.Load and diff against the
//     CURRENT ring, or fall back to full reconstruction -- see its own doc
//     comment for the exact branches.
//  3. Start Consumer, WorkerPool, and this instance's own background loops
//     (Sweeper, ReconstructionQueue, Reconciler, the ownership watch, and
//     the snapshot ticker) against g.ctx.
//  4. (folded into 3): the snapshot ticker's own Pause -> Save -> Resume
//     sequence, §0 D9.
//  5. readyErr clears once this point is reached -- see CheckReady's own
//     doc comment for how the REST of AMENDMENT A3's gate is enforced live,
//     not cached here.
func (g *BloomGateway) starting(ctx context.Context) (err error) {
	defer func() {
		if err != nil && g.subservices != nil {
			if stopErr := services.StopManagerAndAwaitStopped(context.Background(), g.subservices); stopErr != nil {
				level.Error(g.logger).Log("msg", "bloomgateway: failed to stop ring subservices after a failed start", "err", stopErr)
			}
		}
	}()

	g.subservices, err = services.NewManager(g.ringManager.Services()...)
	if err != nil {
		return fmt.Errorf("bloomgateway: creating ring subservices manager: %w", err)
	}
	g.subservicesWatcher = services.NewFailureWatcher()
	g.subservicesWatcher.WatchManager(g.subservices)

	if err := services.StartManagerAndAwaitHealthy(ctx, g.subservices); err != nil {
		return fmt.Errorf("bloomgateway: starting ring subservices: %w", err)
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, ringActiveWaitTimeout)
	activeErr := ring.WaitInstanceState(waitCtx, g.ringManager.Ring, g.instanceID, ring.ACTIVE)
	waitCancel()
	if activeErr != nil {
		return fmt.Errorf("bloomgateway: waiting for this instance to become ACTIVE in the ring: %w", activeErr)
	}
	level.Info(g.logger).Log("msg", "bloomgateway: instance is ACTIVE in the ring", "instance", g.instanceID)

	stabilityCtx, stabilityCancel := context.WithTimeout(ctx, ringStabilityMaxWaiting)
	stabilityErr := ring.WaitRingStability(stabilityCtx, g.ringManager.Ring, ringOp, ringStabilityMinDuration, ringStabilityMaxWaiting)
	stabilityCancel()
	if stabilityErr != nil {
		level.Warn(g.logger).Log("msg", "bloomgateway: ring topology did not stabilize within the max wait; proceeding anyway", "err", stabilityErr)
	}

	offsets, err := g.reconcileStartup()
	if err != nil {
		return fmt.Errorf("bloomgateway: reconciling startup state against the ring: %w", err)
	}

	if err := g.consumer.Start(g.ctx, offsets); err != nil {
		return fmt.Errorf("bloomgateway: starting kafka consumer: %w", err)
	}
	g.workerPool.Start(g.ctx)

	for _, loop := range []func(context.Context){
		g.sweeper.Run,
		g.runReconstructionLoop,
		g.runReconciliationLoop,
		g.runOwnershipWatch,
		g.runSnapshotTicker,
		g.runStatsLoop,
	} {
		loop := loop
		g.bgWG.Add(1)
		go func() {
			defer g.bgWG.Done()
			loop(g.ctx)
		}()
	}

	g.readyErr.Store(nil)
	level.Info(g.logger).Log("msg", "bloomgateway: startup sequence complete")
	return nil
}

// reconcileStartup is the plan's step 2 (§3 WP20), factored out into its own
// method specifically so a test can drive it directly -- without also
// starting the reconstruction queue's Run loop -- to deterministically
// exercise AMENDMENT A3's gap (ranges enqueued but not yet claimed by
// BeginConstructing).
//
// Returns the offsets Consumer.Start should resume from (nil means every
// partition starts at AtStart(), the plan's "any other load failure" branch).
//
// Registry and TenantSet state are NOT ownership-scoped (§ Event processing:
// "commit is unconditional on local entry count" -- every instance's
// registry/A_T reflects every block it has ever seen on the topic,
// regardless of which leaves it owns), so a successful load imports them
// wholesale, unconditionally. Only the Directory (leaf-address-partitioned
// by construction) and the snapshot's ConstructingRanges are filtered by
// current ownership.
func (g *BloomGateway) reconcileStartup() (map[int32]int64, error) {
	ownedRanges, err := g.currentOwnedRanges()
	if err != nil {
		return nil, fmt.Errorf("resolving owned leaf ranges: %w", err)
	}

	state, loadErr := g.snapshotter.Load(g.cfg.Snapshot.Path, g.cfg.D, g.cfg.F, g.seedFingerprint)
	if loadErr != nil {
		if errors.Is(loadErr, ErrSnapshotMismatch) {
			level.Warn(g.logger).Log("msg", "bloomgateway: snapshot mismatch; discarding and reconstructing", "err", loadErr)
		} else {
			level.Info(g.logger).Log("msg", "bloomgateway: no usable snapshot; reconstructing", "err", loadErr)
		}
		g.reconstructionQueue.Enqueue(ownedRanges)
		g.lastOwnedRanges = ownedRanges
		return nil, nil
	}

	level.Info(g.logger).Log("msg", "bloomgateway: loaded snapshot", "path", g.cfg.Snapshot.Path, "complete_leaves", len(state.CompleteLeaves), "constructing_ranges", len(state.ConstructingRanges))
	// snapshot_age_seconds' age is measured from this instant, not from
	// whatever the snapshot's own on-disk contents claim (the format has no
	// "saved at" field at all) -- DESIGN.md's own "loaded OR saved" wording,
	// § Metrics.
	g.lastSnapshotUnixNano.Store(time.Now().UnixNano())

	if err := g.reg.Import(state.Blocks); err != nil {
		return nil, fmt.Errorf("importing registry from snapshot: %w", err)
	}
	if err := g.tenants.Import(state.Tenants); err != nil {
		return nil, fmt.Errorf("importing tenant state from snapshot: %w", err)
	}

	ownedIdx := func(idx uint32) bool { return leafRangesContain(ownedRanges, idx) }
	for idx, leaf := range state.CompleteLeaves {
		if !ownedIdx(idx) {
			// No longer owned by this instance: never loaded. A freshly
			// constructed Directory starts every slot at LeafNil, so
			// simply not loading it IS the shed (Directory.Shed on an
			// already-nil slot is documented as a no-op) -- there is
			// nothing to Shed away here.
			continue
		}
		if _, started := g.dir.BeginConstructing(idx); started {
			if cerr := g.dir.Complete(idx, leaf); cerr != nil {
				level.Warn(g.logger).Log("msg", "bloomgateway: completing snapshot-loaded leaf failed", "leaf", idx, "err", cerr)
			}
		}
	}

	// Owned ranges absent from (or only partially covered by) the
	// snapshot's CompleteLeaves need a real reconstruction pass;
	// BeginConstructing no-ops (Enqueue/RunBatch's own documented contract)
	// for every index just loaded above, so enqueuing the FULL owned range
	// set here costs nothing extra for those and correctly queues the rest.
	g.reconstructionQueue.Enqueue(ownedRanges)
	// Ranges that were still mid-flight at snapshot-save time never
	// persisted their (necessarily partial) leaf content -- only the bare
	// range (snapshot.go's own documented "constructing/pending ranges
	// re-enqueued on load") -- filtered down to the portion still owned
	// (a topology change while this instance was down may have moved some
	// of it elsewhere).
	g.reconstructionQueue.Enqueue(intersectOwnedRanges(state.ConstructingRanges, ownedIdx))

	g.lastOwnedRanges = ownedRanges
	return state.Offsets, nil
}

// running blocks until ctx is done or the ring subservices report a
// failure -- the generic dskit shape every ring-backed module in this repo
// uses (module-wiring report convention: "the failure channel is selected
// alongside ctx.Done() inside running()"). This instance's OWN background
// loops (sweep, reconstruction, reconciliation, ownership watch, snapshot
// ticker) are not services.Service values and so have no analogous failure
// channel to select on here -- see the package doc comment's second
// deviation note for why, and their own wrapper functions (below) for how
// their errors are handled instead (logged, never propagated: they are
// designed to retry forever until ctx is cancelled).
func (g *BloomGateway) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-g.subservicesWatcher.Chan():
		return fmt.Errorf("bloomgateway: ring subservice failed: %w", err)
	}
}

// stopping tears down everything starting() brought up, in roughly reverse
// order: readyErr flips to ErrStopping immediately (live-store's own
// pattern) so CheckReady stops reporting ready the instant shutdown begins,
// then this instance's own background loops are cancelled and awaited,
// then the write path (workerPool/consumer) is stopped, then the ring
// subservices.
func (g *BloomGateway) stopping(failureCase error) error {
	g.readyErr.Store(&ErrStopping)

	g.cancel()
	g.workerPool.Stop()
	if err := g.consumer.Close(); err != nil {
		level.Warn(g.logger).Log("msg", "bloomgateway: failed to close kafka consumer", "err", err)
	}
	g.bgWG.Wait()

	if g.subservices != nil {
		if err := services.StopManagerAndAwaitStopped(context.Background(), g.subservices); err != nil {
			level.Warn(g.logger).Log("msg", "bloomgateway: failed to stop ring subservices", "err", err)
		}
	}

	return failureCase
}

// CheckReady implements the readiness gate AMENDMENT A3 specifies: ring
// ACTIVE AND (snapshot loaded OR reconstruction enqueued) AND zero
// LeafConstructing slots AND ReconstructionQueue.PendingRanges() == 0 --
// strengthened by one more live check this WP's own testing found
// necessary (see below).
//
// readyErr (live-store's own atomic.Pointer[error] pattern) carries ONLY
// the one-time "has this instance's startup sequence completed" latch --
// the plan's original "(snapshot loaded OR reconstruction enqueued)"
// clause, which reconcileStartup satisfies unconditionally on every
// success path (it always either imports a snapshot or enqueues a full
// reconstruction). It never flips back to non-nil except on shutdown.
//
// The rest of the gate is read LIVE on every call, deliberately not cached:
// ring health and reconstruction progress can regress at any point over
// the service's life (a scale event acquiring new ranges, a heartbeat
// miss), and a stale cached "yes, ready" is exactly what would let a
// StatefulSet rollout stack two simultaneous reconstructions -- the
// failure mode the readiness gate exists to prevent (§ Availability
// model).
//
// Deviation/strengthening found by this WP's own testing, reported per the
// harness's own instructions: AMENDMENT A3's gate as specified --
// PendingRanges()==0 AND zero LeafConstructing -- has a narrower residual
// race than the one A3 itself closes. ReconstructionQueue.drain() clears
// q.pending (making PendingRanges() report 0) THE INSTANT a batch is
// claimed, but BeginConstructing is only called for that batch's leaves
// AFTERWARD, in a loop (processBatch step 1) that takes real, nonzero time
// to run for every index. Between drain() returning and that loop's FIRST
// (or, for a still-in-progress leaf, ANY not-yet-reached) iteration, an
// owned leaf can be genuinely LeafNil while BOTH of AMENDMENT A3's own
// conditions already read as satisfied -- caught directly by this WP's own
// multi-instance test flaking under -count=5 before this fix landed. The
// fix generalizes the intent correctly: instead of "zero constructing" and
// "nothing pending" as two separate proxies, check directly that EVERY
// leaf this instance currently owns (per OwnedLeafRanges) is
// LeafComplete -- neither LeafNil nor LeafConstructing passes. This
// subsumes both of AMENDMENT A3's own conditions (a nil-or-constructing
// owned leaf is never complete) and closes the gap the two proxies missed
// individually. Cost: O(owned leaves) -- Directory.State per index in the
// (small, ring-bounded) owned-range list, actually CHEAPER than the
// previous constructingLeaves() helper's O(2^D) full-directory Range walk,
// and removed together with this change.
func (g *BloomGateway) CheckReady(context.Context) error {
	if errp := g.readyErr.Load(); errp != nil {
		if err := *errp; err != nil {
			return err
		}
	}

	ringState, err := g.ringManager.Ring.GetInstanceState(g.instanceID)
	if err != nil {
		return fmt.Errorf("bloomgateway: not ready: resolving own ring state: %w", err)
	}
	if ringState != ring.ACTIVE {
		return fmt.Errorf("bloomgateway: not ready: instance is %s in the ring, not ACTIVE", ringState)
	}

	if pending := g.reconstructionQueue.PendingRanges(); pending > 0 {
		return fmt.Errorf("bloomgateway: not ready: %d leaf ranges enqueued for reconstruction but not yet claimed", pending)
	}

	ownedRanges, err := g.currentOwnedRanges()
	if err != nil {
		return fmt.Errorf("bloomgateway: not ready: resolving owned leaf ranges: %w", err)
	}
	for _, r := range ownedRanges {
		for idx := r.Start; idx < r.End; idx++ {
			if s := g.dir.State(idx); s != LeafComplete {
				return fmt.Errorf("bloomgateway: not ready: leaf %d is %v, not complete", idx, s)
			}
		}
	}

	return nil
}

// Query implements tempopb.BloomGatewayServer directly, delegating to the
// internal query.Server, so cmd/tempo can register *BloomGateway with no
// wrapper type.
func (g *BloomGateway) Query(ctx context.Context, req *tempopb.BloomGatewayQueryRequest) (*tempopb.BloomGatewayQueryResponse, error) {
	return g.server.Query(ctx, req)
}

// Ring exposes the read ring for HTTP status-page mounting and a future
// query-frontend client pool (module-wiring report recommendation #3).
func (g *BloomGateway) Ring() *ring.Ring {
	return g.ringManager.Ring
}

// currentOwnedRanges resolves this instance's currently-owned leaf ranges
// against the live ring -- reconcileStartup's and runOwnershipWatch's
// shared entrypoint into OwnedLeafRanges (bloomgateway_ring.go).
func (g *BloomGateway) currentOwnedRanges() ([]LeafRange, error) {
	rs, err := g.ringManager.Ring.GetAllHealthy(ringOp)
	if err != nil {
		return nil, err
	}
	return OwnedLeafRanges(rs.Instances, g.instanceID, g.cfg.D), nil
}

// leafRangesContain reports whether idx falls within any of ranges --
// sorted and non-overlapping, exactly OwnedLeafRanges' own return contract
// (bloomgateway_ring.go). A binary search: ranges are bounded by this
// instance's token count (<=512, ring-lifecycler report), never by 2^D, so
// this stays cheap regardless of cell size.
func leafRangesContain(ranges []LeafRange, idx uint32) bool {
	i := sort.Search(len(ranges), func(i int) bool { return ranges[i].End > idx })
	return i < len(ranges) && ranges[i].Start <= idx
}

// intersectOwnedRanges returns the subset of idx values in ranges that
// ownedIdx approves, re-coalesced -- iterating leaf-by-leaf within the
// (expected-narrow) input ranges only, never the whole directory. Used to
// filter a snapshot's ConstructingRanges down to only the leaves still
// owned by the current ring (a range may have been partially reassigned to
// another instance while this one was down).
func intersectOwnedRanges(ranges []LeafRange, ownedIdx func(uint32) bool) []LeafRange {
	var out []LeafRange
	for _, r := range ranges {
		var start uint32
		open := false
		for idx := r.Start; idx < r.End; idx++ {
			switch {
			case ownedIdx(idx) && !open:
				start, open = idx, true
			case !ownedIdx(idx) && open:
				out = append(out, LeafRange{Start: start, End: idx})
				open = false
			}
		}
		if open {
			out = append(out, LeafRange{Start: start, End: r.End})
		}
	}
	return out
}

// rangeSetDifference returns the portions of a not covered by any range in
// b -- a minus b, as a coalesced range list. Both inputs are expected
// sorted and non-overlapping (OwnedLeafRanges' own contract); this is a
// straightforward O(len(a)*len(b)) subtraction rather than an optimized
// merge, since both lists are bounded by instance token counts (<=512)
// regardless of cell size -- runOwnershipWatch's own diff primitive,
// deliberately avoiding an O(2^D) per-leaf scan to detect newly owned
// ranges every tick.
func rangeSetDifference(a, b []LeafRange) []LeafRange {
	var out []LeafRange
	for _, ra := range a {
		pieces := []LeafRange{ra}
		for _, rb := range b {
			var next []LeafRange
			for _, p := range pieces {
				if rb.End <= p.Start || rb.Start >= p.End {
					next = append(next, p) // no overlap
					continue
				}
				if rb.Start > p.Start {
					next = append(next, LeafRange{Start: p.Start, End: rb.Start})
				}
				if rb.End < p.End {
					next = append(next, LeafRange{Start: rb.End, End: p.End})
				}
			}
			pieces = next
		}
		out = append(out, pieces...)
	}
	return out
}

// runOwnershipWatch is this file's own addition (package doc comment
// above): periodically recomputes this instance's owned leaf ranges against
// the live ring and reacts to any change -- shedding leaves no longer
// owned, enqueuing reconstruction for leaves newly owned since the last
// tick. This is what makes DESIGN.md's own Scale-out step 4 ("Previous
// owners shed the moved leaves as their ring view updates") and this WP's
// own multi-instance scale-out test true for an instance that never
// restarts.
func (g *BloomGateway) runOwnershipWatch(ctx context.Context) {
	ticker := time.NewTicker(ownershipReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.reconcileOwnership()
		}
	}
}

// reconcileOwnership is one runOwnershipWatch tick. Shedding walks
// g.dir.Range, bounded by however many leaves this instance CURRENTLY
// holds (the same order of cost as e.g. the sweep's own directory pass,
// not the whole 2^D space); detecting newly owned ranges instead uses
// rangeSetDifference against g.lastOwnedRanges, a small range-list
// subtraction independent of D, specifically so a healthy, unchanged
// instance's steady-state tick costs O(owned leaves) once for shedding and
// O(ranges) for the diff -- never enqueuing (and therefore never
// perturbing CheckReady's PendingRanges() check) when nothing has actually
// changed since the last tick.
func (g *BloomGateway) reconcileOwnership() {
	ownedRanges, err := g.currentOwnedRanges()
	if err != nil {
		level.Warn(g.logger).Log("msg", "bloomgateway: ownership reconcile: resolving owned ranges failed; will retry next tick", "err", err)
		return
	}

	// Shed only the ranges that were owned last tick but are not owned now,
	// computed as a bounded range-list difference — NOT a full 2^D directory
	// walk. At the reference D=25 the directory has ~33.6M slots; walking all
	// of them every ownershipReconcileInterval (1s default) purely to find
	// the usually-empty shed set would burn a core continuously in steady
	// state, contradicting DESIGN.md's "CPU is not a sizing axis" and needling
	// the load-bearing RF=1 consumer-lag budget. A healthy, unchanged instance
	// now does O(ranges) work per tick and touches no leaf at all.
	shedRanges := rangeSetDifference(g.lastOwnedRanges, ownedRanges)
	shed := 0
	for _, r := range shedRanges {
		for idx := r.Start; idx < r.End; idx++ {
			g.dir.Shed(idx)
			shed++
		}
	}
	if shed > 0 {
		level.Info(g.logger).Log("msg", "bloomgateway: ownership reconcile: shed leaves no longer owned", "count", shed)
	}

	newlyOwned := rangeSetDifference(ownedRanges, g.lastOwnedRanges)
	g.lastOwnedRanges = ownedRanges
	if len(newlyOwned) > 0 {
		level.Info(g.logger).Log("msg", "bloomgateway: ownership reconcile: enqueuing newly owned ranges", "ranges", len(newlyOwned))
		g.reconstructionQueue.Enqueue(newlyOwned)
	}
}

// runReconstructionLoop and runReconciliationLoop wrap the two blocking
// Run(ctx) error loops with uniform logging (package doc comment's second
// deviation note): both are designed to retry internally forever and only
// ever return once ctx is done, so a non-cancellation error here would be
// a genuine surprise worth logging loudly, not silently swallowing.
func (g *BloomGateway) runReconstructionLoop(ctx context.Context) {
	if err := g.reconstructionQueue.Run(ctx); err != nil && ctx.Err() == nil {
		level.Error(g.logger).Log("msg", "bloomgateway: reconstruction queue loop exited unexpectedly", "err", err)
	}
}

func (g *BloomGateway) runReconciliationLoop(ctx context.Context) {
	if err := g.reconciler.Run(ctx); err != nil && ctx.Err() == nil {
		level.Error(g.logger).Log("msg", "bloomgateway: reconciliation loop exited unexpectedly", "err", err)
	}
}

// runSnapshotTicker is the plan's step 4 (§3 WP20): every cfg.Snapshot.
// Interval, Pause -> Snapshotter.Save -> Resume (§ Snapshots consistency,
// §0 D9). Interval <= 0 disables snapshotting entirely (config.go's own
// documented default-on behavior; an operator can opt out, at the cost of
// every restart becoming a full reconstruction).
func (g *BloomGateway) runSnapshotTicker(ctx context.Context) {
	if g.cfg.Snapshot.Interval <= 0 {
		return
	}
	ticker := time.NewTicker(g.cfg.Snapshot.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = g.saveSnapshot() // logged internally; nothing more to do here but try again next cycle
		}
	}
}

// saveSnapshot performs exactly one Pause -> Save -> Resume cycle. Pause
// quiesces the live Kafka apply path, reducing churn during the pass; it is
// NOT relied on for correctness of the leaf reads, because it does not stop
// the sweep/reconstruction/reconciliation writers. buildSnapshotState
// therefore takes consistent under-lock copies of each leaf via CloneLeaf
// (see there), so a concurrent writer on any of those paths cannot corrupt or
// tear the serialized state.
// saveSnapshot returns any error encountered (tests assert on it directly);
// runSnapshotTicker's own call site discards it, having already logged
// internally below -- there is nothing more a periodic ticker can usefully
// do with it beyond trying again next cycle.
func (g *BloomGateway) saveSnapshot() error {
	g.workerPool.Pause()
	defer g.workerPool.Resume()

	state, err := g.buildSnapshotState()
	if err != nil {
		level.Warn(g.logger).Log("msg", "bloomgateway: snapshot: failed to assemble state; skipping this cycle", "err", err)
		return err
	}
	if err := g.snapshotter.Save(g.cfg.Snapshot.Path, state); err != nil {
		level.Warn(g.logger).Log("msg", "bloomgateway: snapshot: save failed", "err", err)
		return err
	}
	// snapshot_age_seconds' other update point (see reconcileStartup's Load
	// counterpart above) -- DESIGN.md's own "age of the most recently
	// loaded OR saved snapshot" wording, § Metrics.
	g.lastSnapshotUnixNano.Store(time.Now().UnixNano())
	level.Info(g.logger).Log("msg", "bloomgateway: snapshot saved", "path", g.cfg.Snapshot.Path)
	return nil
}

// buildSnapshotState assembles a snapshot.State from the live structures.
// Must only be called while the worker pool is paused (saveSnapshot's own
// contract) -- see its doc comment for why.
func (g *BloomGateway) buildSnapshotState() (*State, error) {
	complete := make(map[uint32]*Leaf)
	var constructingIdx []uint32
	g.dir.Range(func(idx uint32, _ LeafState) bool {
		// CloneLeaf, not Leaf: the worker-pool Pause() around this only stops
		// the live Kafka apply path, NOT the sweep, reconstruction, or
		// reconciliation goroutines, which also write leaves. Leaf() hands
		// back the live *Leaf with the stripe lock already released, so
		// serializing it would race those writers (a data race on the
		// leaf's backing slices, and a torn snapshot). CloneLeaf takes a
		// consistent deep copy under the stripe lock, so what we serialize is
		// an atomic point-in-time view no concurrent writer can corrupt.
		leaf, state := g.dir.CloneLeaf(idx)
		switch state {
		case LeafComplete:
			complete[idx] = leaf
		case LeafConstructing:
			constructingIdx = append(constructingIdx, idx)
		case LeafNil:
			// Range never visits LeafNil slots; unreachable in practice,
			// kept only so this switch is exhaustive over LeafState.
		}
		return true
	})

	var blocks []Block
	g.reg.Range(func(b *Block) bool {
		blocks = append(blocks, *b)
		return true
	})

	tenantSnap, err := g.tenants.Export()
	if err != nil {
		return nil, fmt.Errorf("exporting tenant state: %w", err)
	}

	return &State{
		D:                  g.cfg.D,
		F:                  g.cfg.F,
		SeedFingerprint:    g.seedFingerprint,
		Tokens:             g.ringManager.Lifecycler.GetTokens(),
		Offsets:            g.workerPool.AppliedOffsets(),
		CompleteLeaves:     complete,
		ConstructingRanges: coalesceConsecutive(constructingIdx),
		Blocks:             blocks,
		Tenants:            tenantSnap,
	}, nil
}

// coalesceConsecutive turns a strictly ascending (Directory.Range's own
// "increasing idx order" guarantee) slice of individual leaf indices into
// coalesced [Start,End) ranges.
func coalesceConsecutive(idxs []uint32) []LeafRange {
	if len(idxs) == 0 {
		return nil
	}
	var out []LeafRange
	start, prev := idxs[0], idxs[0]
	for _, idx := range idxs[1:] {
		if idx == prev+1 {
			prev = idx
			continue
		}
		out = append(out, LeafRange{Start: start, End: prev + 1})
		start, prev = idx, idx
	}
	return append(out, LeafRange{Start: start, End: prev + 1})
}

// runStatsLoop periodically populates the state gauges that were previously
// declared in metrics.go but never populated by any production code:
// blocks_live, entries_total, owned_leaves{state}, snapshot_age_seconds,
// miss_fp_rate_estimate. Every source refreshStats reads is either an
// atomic counter (Directory.EntryTotal/LeafStateCounts, Registry.LiveCount)
// or a plain field (lastSnapshotUnixNano) -- NEVER a directory walk. That
// distinction is load-bearing, not a style preference: a prior defect
// (Phase C #5, STATE.md) was exactly an O(2^D) directory walk on a
// 1-second ticker (runOwnershipWatch, before its fix); this loop must not
// reintroduce that shape on ITS tick, so every value it reads has to
// already be maintained incrementally elsewhere (directory.go, registry.go)
// before this loop can simply read it.
func (g *BloomGateway) runStatsLoop(ctx context.Context) {
	g.refreshStats() // populate immediately, not just after the first tick
	ticker := time.NewTicker(statsRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.refreshStats()
		}
	}
}

// refreshStats sets every gauge runStatsLoop is responsible for from its
// cheap, already-maintained source. See each gauge's own comment below for
// why its particular source is safe to read on a 15s cadence.
func (g *BloomGateway) refreshStats() {
	// blocks_live: Registry's own incrementally-maintained live-block
	// counter (registry.go's CommitLive/MarkDeleted exactly-once
	// transition points), not a Range walk over the ~100k-block registry.
	g.metrics.blocksLive.Set(float64(g.reg.LiveCount()))

	// entries_total: Directory's atomic counter, self-healed once per
	// complete Sweeper.Pass (sweep.go) against any incremental-accounting
	// drift -- see Directory.EntryTotal's own doc comment.
	entries := g.dir.EntryTotal()
	g.metrics.entriesTotal.Set(float64(entries))

	// owned_leaves{state}: Directory's own leaf-state transition counters
	// (BeginConstructing/Complete/Shed/Abandon), the same bookkeeping
	// CheckReady's readiness gate already relies on existing per-leaf
	// (dir.State), just aggregated instead of scanned.
	constructing, complete := g.dir.LeafStateCounts()
	g.metrics.ownedLeaves.WithLabelValues("constructing").Set(float64(constructing))
	g.metrics.ownedLeaves.WithLabelValues("complete").Set(float64(complete))

	// snapshot_age_seconds: NaN before this process has ever loaded or
	// saved a snapshot (lastSnapshotUnixNano's documented zero sentinel,
	// see its field comment) rather than 0 -- 0 would misreport "just
	// snapshotted" and would make a naive `snapshot_age_seconds >
	// threshold` alert wrongly silent on an instance that has produced no
	// snapshot at all (including one with snapshotting disabled via
	// Config.Snapshot.Interval <= 0), instead of correctly reporting "no
	// data".
	if nanos := g.lastSnapshotUnixNano.Load(); nanos == 0 {
		g.metrics.snapshotAgeSeconds.Set(math.NaN())
	} else {
		g.metrics.snapshotAgeSeconds.Set(time.Since(time.Unix(0, nanos)).Seconds())
	}

	// miss_fp_rate_estimate: DESIGN.md § Sizing's own closed-form estimate,
	// pairs / 2^(d+f) -- entries_total already IS this instance's pair
	// count (one leaf entry per (trace, live block) pair, leaf.go), so this
	// is a single derived line, free after the sources above.
	g.metrics.missFPRateEstimate.Set(float64(entries) / math.Pow(2, float64(g.cfg.D)+float64(g.cfg.F)))
}

// consumerLagFn adapts consumerLag to reconciliation.go's lagFn func()
// time.Duration contract (no context parameter).
func (g *BloomGateway) consumerLagFn() time.Duration {
	ctx, cancel := context.WithTimeout(context.Background(), consumerLagCheckTimeout)
	defer cancel()
	return g.consumerLag(ctx)
}

// consumerLag is this WP's own addition, not specified by the plan: neither
// Consumer (consumer.go) nor its Record type exposes a ready-made lag-in-
// time primitive (no per-record timestamp is threaded through the bounded
// queue, by that file's own design), yet Reconciler.NewReconciler requires
// a func() time.Duration lagFn to gate repair-Adds (§ Reconciliation). This
// resolves to a conservative binary signal built entirely from Consumer's
// existing exported surface (OffsetsAtOrBefore, CurrentFetchOffsets): 0
// while every partition has been fetched at least up to "now" per the
// broker, or LagGate+1s (guaranteed to exceed any configured gate) the
// moment any partition is behind, or the broker call itself fails --
// repair-Adds are suppressed for the ENTIRE duration the consumer is
// behind (or unreachable) at all, which is at least as conservative as
// DESIGN.md's own "skips repair-Adds until lag is back under threshold"
// requires. A broker round-trip per call is acceptable: reconciliation.
// Pass calls this at most once per tenant per cfg.Reconciliation.Period
// (15 min default), never a hot path.
func (g *BloomGateway) consumerLag(ctx context.Context) time.Duration {
	behind := g.cfg.Reconciliation.LagGate + time.Second

	end, err := g.consumer.OffsetsAtOrBefore(ctx, time.Now())
	if err != nil {
		// Unknown lag: fail safe toward "assume behind" -- suppressing a
		// repair-Add costs only freshness (healed next period once the
		// broker answers again); wrongly ALLOWING one during a real broker
		// outage costs a redundant object-store fetch race exactly when
		// recovery needs it least (§ Reconciliation).
		return behind
	}

	current := g.consumer.CurrentFetchOffsets()
	for partition, endOffset := range end {
		if current[partition] < endOffset {
			return behind
		}
	}
	return 0
}
