package bloomgateway

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
)

// gatewayTestD/gatewayTestF are this file's own (small) leaf-address/
// fingerprint sizing -- deliberately not the package's shared testD/testF
// (events_test.go), so this file's D is legible in isolation (32 leaves)
// without depending on another file's constant staying exactly 4.
const (
	gatewayTestD uint8 = 5 // 32 leaves
	gatewayTestF uint8 = 8
)

// newTestGatewayConfig builds a Config for a full BloomGateway lifecycle
// test: real (small) ring parameters against a shared in-memory KV store
// (ringStore, so multiple instances in one test can share one ring), real
// Kafka parameters against a kfake broker, and a snapshot path under the
// test's own temp dir. Reconciliation.Period is deliberately huge (never
// ticks within a test's lifetime): reconciliation's own behavior is WP19's
// test responsibility, not this file's, and its periodic backendReader.
// Tenants() calls would otherwise confound this file's own reconstruction-
// specific call-count assertions (see the snapshot round-trip test).
func newTestGatewayConfig(t *testing.T, ringStore kv.Client, kafkaAddr, kafkaTopic, snapshotPath string) Config {
	t.Helper()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("bloom-gateway", flag.NewFlagSet("test", flag.ContinueOnError))
	require.NoError(t, cfg.Seed.Set("bloomgateway-test-seed"))

	cfg.D = gatewayTestD
	cfg.F = gatewayTestF
	cfg.NumTokens = 16

	cfg.Ring = newTestRingConfig(t, ringStore)
	// Override newTestRingConfig's own fast (200ms) HeartbeatTimeout
	// (bloomgateway_ring_test.go, WP6's own file, tuned for THAT file's
	// require.Eventually-retried assertions): dskit's ring model stores each
	// instance's heartbeat Timestamp at ONE-SECOND granularity
	// (InstanceDesc.IsHeartbeatHealthy: time.Unix(i.Timestamp, 0)), so a
	// heartbeat timeout finer than ~1s makes GetAllHealthy flicker
	// transiently unhealthy for an actively, perfectly heartbeating solo
	// instance purely from second-boundary rounding -- confirmed directly
	// (a throwaway debug test) before landing this fix. Production's own
	// default (pkg/ring/config.go: 1 minute) is already comfortably above
	// this granularity; this override only matters for tests, which is why
	// it lives here and not in bloomgateway_ring.go.
	cfg.Ring.HeartbeatPeriod = 100 * time.Millisecond
	cfg.Ring.HeartbeatTimeout = 3 * time.Second

	// Override the production-sized ring-stability gate (config.go's
	// RingStabilityWindow/RingStabilityTimeout defaults, 15s/1m -- added
	// for the staggered-cold-start incident, Bug B) down to test speed:
	// left at their defaults, every test that starts a real instance
	// through starting() would need up to 15 REAL, consecutive seconds of
	// unchanged ring state before the gate is satisfied, for no reason --
	// these fixtures are single- and few-instance, in-memory-KV,
	// kfake-backed, and converge fast. This override does NOT make the
	// gate itself millisecond-fast, though: dskit's WaitRingStability
	// polls its own ring-state comparison on a HARDCODED 1s ticker
	// (vendor/.../dskit/ring/util.go's waitStability: "const
	// pollingFrequency = time.Second"), independent of minStability/
	// maxWaiting, so every starting() call that reaches this gate still
	// costs at least one real second (the first poll tick) regardless of
	// how small the configured window is -- this override only avoids
	// needing MULTIPLE consecutive 1s ticks of unchanged ring state to
	// satisfy a much longer window, not the 1s floor itself. Tests that
	// specifically want to exercise staggered-join timing (TestBloomGateway_
	// StaggeredColdStart...) override these again on their own copy; the
	// one test that needs to bypass the poll floor ENTIRELY
	// (TestBloomGateway_StartingPassesConfiguredRingStabilityToWaitRing
	// Stability) does so via g.waitRingStabilityFn instead of via these
	// values.
	cfg.RingStabilityWindow = 20 * time.Millisecond
	cfg.RingStabilityTimeout = 2 * time.Second

	cfg.Kafka = newTestKafkaConfig(kafkaAddr, kafkaTopic)

	cfg.Snapshot.Path = snapshotPath
	cfg.Snapshot.Interval = 0 // ticker disabled by default; tests that need a save call saveSnapshot directly

	// Overrides the production default (a fixed /var/lib/tempo/... path) --
	// starting() creates this directory unconditionally on every real
	// startup (checkShutdownMarker, downscale.go), so every test driving a
	// full BloomGateway through services.StartAndAwaitRunning needs its own
	// scoped, writable directory instead of touching the real machine's
	// filesystem.
	cfg.ShutdownMarkerDir = filepath.Join(t.TempDir(), "shutdown-marker")

	cfg.Reconstruction.Concurrency = 4
	cfg.Reconstruction.RateLimitBytesPerSecond = 1 << 30 // effectively unthrottled for tests

	cfg.Reconciliation.Period = time.Hour // see doc comment above
	cfg.Reconciliation.LagGate = 5 * time.Minute

	cfg.Queue.MaxBytes = 1 << 20
	cfg.Queue.Workers = 2

	require.NoError(t, cfg.Validate())
	return cfg
}

// newTestGatewayCluster builds the shared, cell-wide infrastructure a
// BloomGateway needs (ring KV store, kfake Kafka cluster) once per test,
// returning what New() needs to be pointed at it. Every instance in a test
// shares one of each -- exactly as a real cell's instances share one ring
// and one topic. Partition count is fixed at 4 -- every call site in this
// file uses the same small K, none needs a different topology.
func newTestGatewayCluster(t *testing.T, topic string) (kv.Client, string) {
	t.Helper()
	logger := log.NewNopLogger()
	store, closer := consul.NewInMemoryClient(ring.GetCodec(), logger, nil)
	t.Cleanup(func() { _ = closer.Close() })

	_, addr := testkafka.CreateCluster(t, 4, topic)
	return store, addr
}

// mustNewTestGateway constructs (but does not start) a BloomGateway.
func mustNewTestGateway(t *testing.T, cfg Config, instanceID string, backendReader backend.Reader) *BloomGateway {
	t.Helper()
	g, err := New(cfg, instanceID, backendReader, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	return g
}

// startAndCleanup starts g via the real services.Service lifecycle and
// registers a cleanup to stop it, matching the module-wiring report's own
// "services.Service is embedded, not implemented by hand" convention for
// how every module in this repo is driven in tests.
func startAndCleanup(t *testing.T, g *BloomGateway) {
	t.Helper()
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), g))
	t.Cleanup(func() { _ = services.StopAndAwaitTerminated(context.Background(), g) })
}

// waitReadyTimeout is every call site's budget for CheckReady to turn nil --
// generous relative to this file's small (kfake, inmemory-KV) fixtures, so a
// slow CI box never turns a real bug into a flaky pass.
const waitReadyTimeout = 30 * time.Second

// waitReady polls CheckReady until it reports ready (nil) or waitReadyTimeout.
func waitReady(t *testing.T, g *BloomGateway) {
	t.Helper()
	require.Eventually(t, func() bool {
		return g.CheckReady(context.Background()) == nil
	}, waitReadyTimeout, 10*time.Millisecond, "CheckReady never became true")
}

// allLeavesComplete asserts every leaf in [0, 2^d) is LeafComplete.
func allLeavesComplete(t *testing.T, dir *Directory, d uint8) {
	t.Helper()
	total := uint32(1) << d
	for idx := uint32(0); idx < total; idx++ {
		require.Equalf(t, LeafComplete, dir.State(idx), "leaf %d must be complete", idx)
	}
}

// TestBloomGateway_SingleInstanceLifecycle is this WP's basic full-lifecycle
// deliverable: a single instance (trivially owning the whole ring) starts
// against inmemory KV + testkafka + a fake backend.Reader, becomes ready,
// answers a query, and stops cleanly.
func TestBloomGateway_SingleInstanceLifecycle(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-lifecycle")
	reader := newFakeBackendReader() // no tenants: reconstruction completes trivially

	cfg := newTestGatewayConfig(t, store, addr, "bg-lifecycle", filepath.Join(t.TempDir(), "snapshot.bin"))

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)

	require.Error(t, g.CheckReady(context.Background()), "must not be ready before starting")

	startAndCleanup(t, g)
	waitReady(t, g)

	allLeavesComplete(t, g.dir, cfg.D)
	assert.Zero(t, g.reconstructionQueue.PendingRanges())

	// Sanity: the query path is wired end-to-end. An empty registry/tenant
	// set means an unscoped query for any trace ID must resolve to an
	// empty (not reject-all, not leaf-unavailable) rejection set -- the
	// leaf is complete but the tenant window is empty.
	resp, err := g.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{
		TenantId: "tenant-a",
		TraceId:  traceID(0),
	})
	require.NoError(t, err)
	assert.Zero(t, resp.Flags&FlagLeafUnavailable, "a complete leaf must never set FlagLeafUnavailable")
	assert.Empty(t, resp.Rejected, "an empty tenant window has nothing to reject")

	require.NoError(t, services.StopAndAwaitTerminated(context.Background(), g))
	assert.Error(t, g.CheckReady(context.Background()), "must report not-ready once stopped")
}

// startRingOnly starts just g's ring subservices (Lifecycler+Ring) and
// waits for this instance to observe itself ACTIVE -- everything
// reconcileStartup needs, without going through the full starting()
// sequence (which would also start ReconstructionQueue.Run, racing this
// test's own attempt to observe an enqueued-but-unclaimed range).
func startRingOnly(t *testing.T, g *BloomGateway) {
	t.Helper()
	ctx := context.Background()
	for _, svc := range g.ringManager.Services() {
		require.NoError(t, services.StartAndAwaitRunning(ctx, svc))
	}
	t.Cleanup(func() {
		for _, svc := range g.ringManager.Services() {
			_ = services.StopAndAwaitTerminated(context.Background(), svc)
		}
	})
	require.NoError(t, ring.WaitInstanceState(ctx, g.ringManager.Ring, g.instanceID, ring.ACTIVE))
}

// TestBloomGateway_CheckReady_RequiresReconstructionQueueDrained is
// AMENDMENT A3's named deliverable: between ReconstructionQueue.Enqueue and
// the queue's Run loop actually calling BeginConstructing, the enqueued
// ranges' leaves are still LeafNil -- so "zero LeafConstructing slots"
// alone (the plan's original gate) is satisfied trivially and would
// wrongly report ready. This test drives reconcileStartup directly WITHOUT
// ever starting the reconstruction queue's Run loop, so the ranges it
// enqueues are guaranteed to sit unclaimed for the whole test -- no gate/
// timing games needed to observe the gap deterministically.
func TestBloomGateway_CheckReady_RequiresReconstructionQueueDrained(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-a3")
	reader := newFakeBackendReader()

	cfg := newTestGatewayConfig(t, store, addr, "bg-a3", filepath.Join(t.TempDir(), "snapshot.bin"))
	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)

	ctx := context.Background()
	require.Error(t, g.CheckReady(ctx), "not ready before the ring is even joined (readyErr is still ErrStarting)")

	startRingOnly(t, g)
	require.Error(t, g.CheckReady(ctx), "readyErr is still ErrStarting: reconcileStartup has not run yet")

	offsets, err := g.reconcileStartup()
	require.NoError(t, err)
	assert.Nil(t, offsets, "a snapshotless cold start returns nil offsets (AtStart on every partition)")
	require.Positive(t, g.reconstructionQueue.PendingRanges(), "a fresh, snapshotless single instance must have enqueued its full owned range")
	assert.Equal(t, LeafNil, g.dir.State(0), "BeginConstructing has not run for any of it yet -- the plan's ORIGINAL gate would wrongly pass here")

	// Simulate having reached the point starting() itself would: readyErr
	// only ever flips to nil there. AMENDMENT A3's own point is that this
	// alone must still not be enough.
	g.readyErr.Store(nil)
	require.Error(t, g.CheckReady(ctx), "AMENDMENT A3: ranges are enqueued but not yet claimed by BeginConstructing, so this must NOT be ready")

	stats, err := g.reconstructionQueue.RunBatch(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1<<cfg.D, stats.LeavesStarted)
	assert.Zero(t, g.reconstructionQueue.PendingRanges())
	assert.NoError(t, g.CheckReady(ctx), "once the batch has actually claimed and completed every range, CheckReady must pass")
}

// TestBloomGateway_SnapshotRoundTrip_ColdStartVsSnapshotBackedRestart is this
// WP's named "snapshot-present vs snapshot-absent cold starts" deliverable,
// as one coherent round trip: phase 1 is a genuinely snapshotless cold
// start (must run a real reconstruction pass -- observed directly via the
// fake backend reader's tenant-index call count and RunBatch's own
// BatchStats, since an empty registry/tenant set alone can't distinguish
// "reconstruction ran and found nothing" from "reconstruction never ran");
// phase 2 restarts a fresh instance (same instance ID, same ring, same
// snapshot path) and must load directly from the saved snapshot, never
// re-running a reconstruction pass.
//
// This drives reconcileStartup + RunBatch directly (startRingOnly, not the
// full service) rather than going through CheckReady/waitReady: the
// Reconciler's own Run loop unconditionally makes one immediate
// tenant-index pass on its very first iteration REGARDLESS of
// cfg.Reconciliation.Period (reconciliation.go's Run: runOnce is called
// before the first sleepCtx wait) -- discovered empirically while writing
// this test (an initial version asserting on the full service's call count
// flaked against this confound). Driving reconstruction directly sidesteps
// it entirely, since neither the Reconciler nor any other background loop
// is ever started in this test.
func TestBloomGateway_SnapshotRoundTrip_ColdStartVsSnapshotBackedRestart(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-snapshot-roundtrip")

	reader := newFakeBackendReader()
	tr := testTimeRange()
	reader.setTenantIndex("tenant-a", &backend.TenantIndex{
		CreatedAt: time.Now(),
		Meta: []*backend.BlockMeta{
			// vparquet3 (unsupported encoding): FetchAndApplyBlockColumn
			// resolves it via CommitUnsupportedEncoding, exercising a real
			// tenant-index-driven reconstruction pass without needing an
			// actual on-disk vparquet5 block.
			blockMetaFixture(testUUID(t, 1), "tenant-a", tr, vparquet3.VersionString),
		},
	})

	snapshotPath := filepath.Join(t.TempDir(), "snapshot.bin")
	cfg := newTestGatewayConfig(t, store, addr, "bg-snapshot-roundtrip", snapshotPath)
	ctx := context.Background()
	total := uint32(1) << cfg.D

	// Phase 1: no snapshot file exists yet.
	g1 := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	t.Cleanup(func() { _ = g1.consumer.Close() })
	startRingOnly(t, g1)

	offsets, err := g1.reconcileStartup()
	require.NoError(t, err)
	assert.Nil(t, offsets, "a snapshotless cold start returns nil offsets")

	stats, err := g1.reconstructionQueue.RunBatch(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, total, stats.LeavesStarted, "phase 1 must reconstruct every owned leaf")
	assert.Equal(t, 1, stats.Blocks, "phase 1 must have read the tenant index for real")
	allLeavesComplete(t, g1.dir, cfg.D)

	callsAfterPhase1 := reader.tenantIndexCallCount("tenant-a")
	assert.Equal(t, 1, callsAfterPhase1)

	require.NoError(t, g1.saveSnapshot())

	// Phase 2: a fresh instance, same instance ID/ring/snapshot path.
	g2 := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	t.Cleanup(func() { _ = g2.consumer.Close() })
	startRingOnly(t, g2)

	offsets, err = g2.reconcileStartup()
	require.NoError(t, err)
	assert.NotNil(t, offsets, "a successfully loaded snapshot must return its recorded resume offsets")

	stats, err = g2.reconstructionQueue.RunBatch(ctx)
	require.NoError(t, err)
	assert.Zero(t, stats.LeavesStarted, "every owned leaf was already loaded from the snapshot; nothing left to (re)construct -- BeginConstructing no-oped for every index in the drained range")
	allLeavesComplete(t, g2.dir, cfg.D)

	assert.Equal(t, callsAfterPhase1, reader.tenantIndexCallCount("tenant-a"),
		"a fully-covered snapshot load must trigger NO further tenant-index reads")

	// The registry/tenant state (NOT ownership-scoped, § Event processing)
	// must also have round-tripped: the block reconstruction discovered in
	// phase 1 must still be known (as unsupported-encoding, never
	// rejectable) in phase 2, without a second fetch.
	_, ok := g2.reg.LookupUUID(testUUID(t, 1))
	assert.True(t, ok, "the block registry must survive the snapshot round trip")
}

// TestBloomGateway_ReconcileStartup_PartialSnapshotCoverageReconstructsTheRest
// covers the plan's "still enqueues newly owned ones" clause for a stable
// single instance: a hand-built snapshot covering only HALF the owned
// leaves as complete (simulating a partial/stale save, or leaves this
// instance didn't yet own last time it saved) must still end up with EVERY
// leaf complete after startup -- the covered half loaded directly, the
// other half reconstructed.
func TestBloomGateway_ReconcileStartup_PartialSnapshotCoverageReconstructsTheRest(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-partial-snapshot")
	reader := newFakeBackendReader() // no tenants: the reconstructed half completes trivially

	snapshotPath := filepath.Join(t.TempDir(), "snapshot.bin")
	cfg := newTestGatewayConfig(t, store, addr, "bg-partial-snapshot", snapshotPath)
	total := uint32(1) << cfg.D
	half := total / 2

	seed := []byte(cfg.Seed.String())
	completeLeaves := make(map[uint32]*Leaf, half)
	for idx := uint32(0); idx < half; idx++ {
		leaf := NewLeaf()
		leaf.InsertIfAbsent(uint16(idx), Handle(1)) // distinguishable content, asserted below
		completeLeaves[idx] = leaf
	}
	state := &State{
		D:               cfg.D,
		F:               cfg.F,
		SeedFingerprint: SeedFingerprint(seed),
		Tokens:          []uint32{0},
		Offsets:         map[int32]int64{},
		CompleteLeaves:  completeLeaves,
		Tenants:         TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}
	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))
	require.NoError(t, sn.Save(snapshotPath, state))

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), g))
	t.Cleanup(func() { _ = services.StopAndAwaitTerminated(context.Background(), g) })

	waitReady(t, g)
	allLeavesComplete(t, g.dir, cfg.D)

	// The first half's content must be exactly what was loaded (not
	// silently re-reconstructed/overwritten), the second half must have
	// been reconstructed fresh (empty, since the fake backend has no
	// blocks).
	for idx := uint32(0); idx < half; idx++ {
		handles, ok := g.dir.Lookup(idx, uint16(idx))
		require.True(t, ok)
		assert.Contains(t, handles, Handle(1), "leaf %d must retain its snapshot-loaded content", idx)
	}
	for idx := half; idx < total; idx++ {
		_, ok := g.dir.Lookup(idx, 0)
		require.True(t, ok, "leaf %d must be complete (reconstructed, not merely absent)", idx)
	}
}

// TestBloomGateway_SaveSnapshot_ConcurrentWithDirectoryWrites is the save
// path's own concurrent-writes-under-race test plan item (2026-07-16 OOM
// fix, DESIGN.md § Snapshots amendment): buildSnapshotState no longer
// clones every owned leaf up front while holding the directory's own
// iteration state -- this exercises the new streaming shape by hammering
// saveSnapshot in a tight loop on one goroutine while another concurrently
// mutates (and reshuffles the lifecycle state of) the very same directory
// saveSnapshot is streaming from, mirroring sweep_test.go's own
// TestSweep_ConcurrentPassWithLiveWrites. There is no precise before/after
// assertion -- the point is that -race catches nothing and the LAST
// successful save still loads cleanly despite the concurrent churn.
func TestBloomGateway_SaveSnapshot_ConcurrentWithDirectoryWrites(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-save-concurrency")
	reader := newFakeBackendReader()
	snapshotPath := filepath.Join(t.TempDir(), "snapshot.bin")
	cfg := newTestGatewayConfig(t, store, addr, "bg-save-concurrency", snapshotPath)

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	t.Cleanup(func() { _ = g.consumer.Close() })
	startRingOnly(t, g)

	total := uint32(1) << cfg.D
	for idx := uint32(0); idx < total; idx++ {
		leaf, started := g.dir.BeginConstructing(idx)
		require.True(t, started)
		require.NoError(t, g.dir.Complete(idx, leaf))
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for ctx.Err() == nil {
			if !assert.NoError(t, g.saveSnapshot()) {
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var i uint16
		for ctx.Err() == nil {
			idx := uint32(i) % total
			g.dir.InsertLive(idx, i, Handle(i)+1)
			if i%7 == 0 {
				// Ownership churn: shed the leaf saveSnapshot may be
				// mid-clone on, then immediately reconstruct it -- exactly
				// the "flips away from complete and back" pattern
				// directoryLeafSource.Clone's ok=false branch exists for.
				g.dir.Shed(idx)
				if leaf, started := g.dir.BeginConstructing(idx); started {
					_ = g.dir.Complete(idx, leaf)
				}
			}
			i++
		}
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()
	wg.Wait()

	got, err := g.snapshotter.Load(snapshotPath, cfg.D, cfg.F, g.seedFingerprint)
	require.NoError(t, err, "the last successful save must still load cleanly despite concurrent directory writers")
	assert.LessOrEqual(t, len(got.CompleteLeaves), int(total))
}

// TestBloomGateway_MultiInstanceScaleOut is this WP's own named
// first-class deliverable: a real topology change (2-3 gateways sharing
// one in-memory KV ring and one kfake cluster) exercised end-to-end -- a
// new instance joins, acquires its share of the ring, reconstructs it, and
// the PREVIOUS owner (already running, never restarted) sheds what it no
// longer owns. This is the completeness invariant's closest thing to an
// integration test across a real topology change (§7 invariant #1's
// "also exercised by" column), and the one scenario that only this WP's
// own runOwnershipWatch addition (package doc comment, bloomgateway.go)
// makes true: nothing about an already-running instance's own startup
// sequence would otherwise ever notice a later-joining peer.
func TestBloomGateway_MultiInstanceScaleOut(t *testing.T) {
	// Fast ownership-reconcile ticks so convergence doesn't have to wait
	// out the production-sized default.
	prevInterval := ownershipReconcileInterval
	ownershipReconcileInterval = 20 * time.Millisecond
	t.Cleanup(func() { ownershipReconcileInterval = prevInterval })

	store, addr := newTestGatewayCluster(t, "bg-scaleout")
	reader := newFakeBackendReader() // shared "object store": no tenants, so any instance's reconstruction completes trivially fast

	cfg0 := newTestGatewayConfig(t, store, addr, "bg-scaleout", filepath.Join(t.TempDir(), "g0.bin"))
	total := uint32(1) << cfg0.D

	g0 := mustNewTestGateway(t, cfg0, "bloom-gateway-0", reader)
	startAndCleanup(t, g0)
	waitReady(t, g0)
	allLeavesComplete(t, g0.dir, cfg0.D)

	// cfg1 shares the SAME cell-wide parameters (D/F/seed, from
	// newTestGatewayConfig's own fixed values) and the SAME ring/topic as
	// cfg0 -- only the instance ID and snapshot path differ, exactly as two
	// pods of one StatefulSet would.
	cfg1 := newTestGatewayConfig(t, store, addr, "bg-scaleout", filepath.Join(t.TempDir(), "g1.bin"))
	g1 := mustNewTestGateway(t, cfg1, "bloom-gateway-1", reader)
	startAndCleanup(t, g1)
	waitReady(t, g1)

	// Convergence: every leaf ends up served by EXACTLY one of the two
	// instances -- never both (a real bug), never neither (stuck
	// unconverged) -- and instance 0 must have shed whatever it no longer
	// owns via its own ownership-watch loop, since it was never restarted.
	require.Eventually(t, func() bool {
		for idx := uint32(0); idx < total; idx++ {
			s0, s1 := g0.dir.State(idx) == LeafComplete, g1.dir.State(idx) == LeafComplete
			if s0 == s1 { // both true (double-served) or both false (unserved)
				return false
			}
		}
		return true
	}, 30*time.Second, 20*time.Millisecond, "every leaf must converge to exactly one owner")

	// The scale-out must be REAL, not a no-op: instance 1 must own (and
	// have reconstructed) a non-trivial share.
	rs, err := g0.ringManager.Ring.GetAllHealthy(ringOp)
	require.NoError(t, err)
	require.Len(t, rs.Instances, 2)
	ranges1 := OwnedLeafRanges(rs.Instances, "bloom-gateway-1", cfg0.D)
	assert.NotEmpty(t, ranges1, "the new instance must own a non-trivial share of the ring")

	var leaves0, leaves1 int
	for idx := uint32(0); idx < total; idx++ {
		if g0.dir.State(idx) == LeafComplete {
			leaves0++
		}
		if g1.dir.State(idx) == LeafComplete {
			leaves1++
		}
	}
	assert.EqualValues(t, total, leaves0+leaves1, "every leaf must be served by exactly one instance")
	assert.Positive(t, leaves1, "instance 1 must actually be serving its share")
	assert.Less(t, leaves0, int(total), "instance 0 must have shed at least one leaf it no longer owns")
}

// TestBloomGateway_StaggeredColdStartConvergesWithoutManualIntervention is
// the bug report's own "Staggered join" test: several instances join with
// overlapping, staggered start times -- the shape of a StatefulSet cold
// start where pods do not all reach ACTIVE simultaneously (the live
// incident this guards against had two waves of 8, ~4 minutes apart) -- and
// must all converge to their fair share and reach ready, with every leaf
// served by exactly one instance and none left permanently unserved, with
// no manual restart or intervention.
//
// What this exercises, precisely: Bug A's in-flight-batch re-scoping (an
// early joiner's oversized claim shrinks instead of resurrecting shed
// leaves) plus runOwnershipWatch's steady-state convergence (shed +
// newlyOwned, and the self-heal backstop). It deliberately does NOT
// validate Bug B's ring-stability gate itself -- dskit's WaitRingStability
// polls its own ring-state comparison on a HARDCODED 1s ticker (vendor/
// .../dskit/ring/util.go's waitStability: "const pollingFrequency =
// time.Second"), independent of the configured minStability/maxWaiting
// values, so this test's config (RingStabilityWindow/Timeout overridden to
// 20ms/2s in newTestGatewayConfig, for every OTHER test's sake) can neither
// exercise nor rule out the gate's real behavior: proved empirically in
// review -- neutering the gate entirely (window/timeout=1ms, i.e. an
// instant, always-fired timeout that skips waiting altogether) leaves this
// test passing unchanged, because Bug A's re-scoping and the ownership
// watch converge correctly regardless of how oversized the initial claim
// was. See TestBloomGateway_StartingPassesConfiguredRingStabilityToWaitRing
// Stability below for the gate's own wiring proof, which does not depend on
// dskit's poll cadence at all.
func TestBloomGateway_StaggeredColdStartConvergesWithoutManualIntervention(t *testing.T) {
	// Fast ownership-reconcile ticks (matching TestBloomGateway_
	// MultiInstanceScaleOut's own override) so convergence, and the
	// self-heal pass riding the same ticker, don't have to wait out
	// production-sized intervals.
	prevInterval := ownershipReconcileInterval
	ownershipReconcileInterval = 20 * time.Millisecond
	t.Cleanup(func() { ownershipReconcileInterval = prevInterval })

	const n = 4
	store, addr := newTestGatewayCluster(t, "bg-staggered")
	reader := newFakeBackendReader() // no tenants: every instance's reconstruction completes trivially fast

	gateways := make([]*BloomGateway, n)
	for i := range n {
		cfg := newTestGatewayConfig(t, store, addr, "bg-staggered", filepath.Join(t.TempDir(), fmt.Sprintf("g%d.bin", i)))
		gateways[i] = mustNewTestGateway(t, cfg, fmt.Sprintf("bloom-gateway-%d", i), reader)
	}
	t.Cleanup(func() {
		for _, g := range gateways {
			_ = services.StopAndAwaitTerminated(context.Background(), g)
		}
	})

	// Start every instance concurrently, staggered by a small, overlapping
	// delay -- unlike sequential start+waitReady (every other multi-
	// instance test here), this deliberately lets later joiners register
	// while earlier ones are still mid-startup, the exact race Bug A/B
	// guard against.
	var wg sync.WaitGroup
	for i, g := range gateways {
		delay := time.Duration(i) * 60 * time.Millisecond
		wg.Go(func() {
			time.Sleep(delay)
			require.NoError(t, services.StartAndAwaitRunning(context.Background(), g))
		})
	}
	wg.Wait()

	for _, g := range gateways {
		waitReady(t, g)
	}

	total := uint32(1) << gateways[0].cfg.D

	// Convergence: every leaf ends up LeafComplete on EXACTLY one
	// instance -- no gap (an early or late joiner permanently missing its
	// fair share, Bug C's failure mode) and no double-service, once every
	// instance reports ready. This is "total constructing across
	// instances converges to 2^D" in its strongest form: fully complete,
	// not merely accounted for.
	owner := make([]int, total)
	for idx := range owner {
		owner[idx] = -1
	}
	for gi, g := range gateways {
		for idx := range total {
			if g.dir.State(idx) == LeafComplete {
				require.Equalf(t, -1, owner[idx], "leaf %d double-served by instance %d and %d", idx, owner[idx], gi)
				owner[idx] = gi
			}
		}
	}
	for idx, gi := range owner {
		assert.NotEqualf(t, -1, gi, "leaf %d has no owner: some instance's fair share was never reconstructed", idx)
	}
}

// TestBloomGateway_StartingPassesConfiguredRingStabilityToWaitRingStability
// is Bug B's deterministic wiring proof: it asserts cfg.RingStabilityWindow/
// RingStabilityTimeout actually reach starting()'s ring-stability call, byte
// for byte, with no dependency on dskit's own hardcoded 1s poll floor
// (util.go's waitStability, see the staggered-join test's doc comment
// above) for either correctness or timing. g.waitRingStabilityFn (its own
// field doc comment) is overridden to record the arguments it was called
// with and return nil immediately -- skipping dskit's real polling loop
// entirely, so this test costs no more than an ordinary single-instance
// startup and asserts nothing about real time.
func TestBloomGateway_StartingPassesConfiguredRingStabilityToWaitRingStability(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-stability-wiring")
	reader := newFakeBackendReader()

	cfg := newTestGatewayConfig(t, store, addr, "bg-stability-wiring", filepath.Join(t.TempDir(), "snapshot.bin"))
	cfg.RingStabilityWindow = 7 * time.Second
	cfg.RingStabilityTimeout = 42 * time.Second

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)

	var mu sync.Mutex
	var calls int
	var gotOp ring.Operation
	var gotMinStability, gotMaxWaiting time.Duration
	g.waitRingStabilityFn = func(_ context.Context, r ring.ReadRing, op ring.Operation, minStability, maxWaiting time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		calls++
		assert.Same(t, g.ringManager.Ring, r, "starting() must pass its own ring manager's Ring")
		gotOp = op
		gotMinStability = minStability
		gotMaxWaiting = maxWaiting
		return nil // skip dskit's real polling loop entirely -- no dependency on its 1s poll floor
	}

	startAndCleanup(t, g)
	waitReady(t, g)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls, "starting() must call the ring-stability gate exactly once")
	assert.Equal(t, ringOp, gotOp, "must gate on this package's own ringOp (ACTIVE-only, RF=1)")
	assert.Equal(t, cfg.RingStabilityWindow, gotMinStability, "cfg.RingStabilityWindow must reach WaitRingStability's minStability argument")
	assert.Equal(t, cfg.RingStabilityTimeout, gotMaxWaiting, "cfg.RingStabilityTimeout must reach WaitRingStability's maxWaiting argument")
}

// TestBloomGateway_SelfHealRecoversRingOwnedLeafMissingFromDirectory is the
// bug report's own "Pod-7 scenario" test: reconcileOwnership's per-tick diff
// assumes g.lastOwnedRanges accurately reflects what the directory currently
// holds. If something else (a failed reconstruction batch's Abandon cleanup
// discarding a still-owned leaf alongside genuinely-shed ones, or any other
// path with the same shape) reverts a ring-owned leaf to nil WITHOUT the
// ring's own view of ownership moving, the hot tick's diff sees nothing new
// -- forever, since lastOwnedRanges already "contains" what the ring
// reports every subsequent tick. Only the coarser self-heal pass, which
// compares the ring's current answer directly against directory state
// rather than against an incremental diff baseline, can notice and repair
// this without a manual restart.
func TestBloomGateway_SelfHealRecoversRingOwnedLeafMissingFromDirectory(t *testing.T) {
	prevInterval := ownershipReconcileInterval
	ownershipReconcileInterval = 20 * time.Millisecond
	t.Cleanup(func() { ownershipReconcileInterval = prevInterval })

	store, addr := newTestGatewayCluster(t, "bg-selfheal")
	reader := newFakeBackendReader()

	cfg := newTestGatewayConfig(t, store, addr, "bg-selfheal", filepath.Join(t.TempDir(), "snapshot.bin"))

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	startAndCleanup(t, g)
	waitReady(t, g)
	allLeavesComplete(t, g.dir, cfg.D)

	// Simulate the incident directly: a sub-range this instance still
	// legitimately owns per the ring (ring membership never changes in
	// this test) is reverted to nil by something OTHER than an ownership
	// change -- exactly what a failed batch's overly-broad Abandon cleanup
	// does to a leaf that was never actually shed. lastOwnedRanges is
	// deliberately left untouched: the ring's own answer never looks
	// different, so the hot tick's incremental diff has nothing to react
	// to -- this IS the gap selfHealOwnership exists to close.
	const goneEnd = uint32(4) // leaves [0, goneEnd) are the simulated gap
	for idx := range goneEnd {
		g.dir.Shed(idx)
	}

	// Documents the gap directly: several hot ticks alone must NOT repair
	// this (reconcileOwnership's own diff has no way to).
	time.Sleep(10 * ownershipReconcileInterval)
	for idx := range goneEnd {
		require.Equal(t, LeafNil, g.dir.State(idx), "sanity: the hot-tick diff alone cannot see this gap")
	}

	// The self-heal pass, running at selfHealCheckEvery's coarser cadence
	// alongside the same watch loop, must notice the ring still assigns
	// these leaves to this instance and re-enqueue + reconstruct them --
	// no manual restart, no other intervention.
	require.Eventually(t, func() bool {
		for idx := range goneEnd {
			if g.dir.State(idx) != LeafComplete {
				return false
			}
		}
		return true
	}, 30*time.Second, 20*time.Millisecond,
		"the self-heal pass must re-enqueue and reconstruct a ring-owned leaf missing from the directory")

	allLeavesComplete(t, g.dir, cfg.D)
}
