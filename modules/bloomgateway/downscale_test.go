package bloomgateway

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/util/shutdownmarker"
)

// syncBuffer is a mutex-guarded bytes.Buffer. A real BloomGateway's logger
// is shared by many concurrent background goroutines (Kafka consumer, ring
// lifecycler, worker pool, ...), some of which can still be unwinding their
// own shutdown even after services.StopAndAwaitTerminated returns --
// log.NewSyncWriter alone only guards the WRITE side (its mutex wraps calls
// made THROUGH it), which leaves a plain bytes.Buffer's own String() read
// afterward unsynchronized against a write still in flight through a
// different goroutine. This type puts both Write and String behind the
// same lock, which is what TestBloomGateway_GracefulStop_
// PreparedDownscaleUnregisters below actually needs.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// readRingInstanceDesc reads instanceID's raw entry straight out of the
// shared KV store, bypassing any one instance's own (possibly already
// stopped) *ring.Ring reader -- the only way to assert on ring state AFTER
// the BloomGateway whose entry it is has itself been torn down.
func readRingInstanceDesc(t *testing.T, store kv.Client, instanceID string) (ring.InstanceDesc, bool) {
	t.Helper()
	val, err := store.Get(context.Background(), RingKey)
	require.NoError(t, err)
	desc := ring.GetOrCreateRingDesc(val)
	inst, ok := desc.Ingesters[instanceID]
	return inst, ok
}

// TestBloomGateway_PrepareDownscaleHandler_GetPostDelete is the endpoint's
// own contract test, independent of real ring-stopping timing: GET reports
// "set\n"/"unset\n" (mirroring modules/livestore/downscale.go's own
// PrepareDownscaleHandler response shape exactly), POST creates the marker
// and flips SetKeepInstanceInTheRingOnShutdown(false), DELETE reverses
// both.
func TestBloomGateway_PrepareDownscaleHandler_GetPostDelete(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-prepare-handler")
	reader := newFakeBackendReader()
	cfg := newTestGatewayConfig(t, store, addr, "bg-prepare-handler", filepath.Join(t.TempDir(), "snapshot.bin"))

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	startAndCleanup(t, g)
	waitReady(t, g)

	do := func(method string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		g.PrepareDownscaleHandler(rec, httptest.NewRequest(method, "/bloom-gateway/prepare-downscale", nil))
		return rec
	}

	rec := do(http.MethodGet)
	assert.Equal(t, "unset\n", rec.Body.String(), "no marker yet")
	assert.True(t, g.ringManager.Lifecycler.ShouldKeepInstanceInTheRingOnShutdown(), "default is keep-in-ring")

	rec = do(http.MethodPost)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.False(t, g.ringManager.Lifecycler.ShouldKeepInstanceInTheRingOnShutdown(), "POST must flip to unregister-on-stop")
	exists, err := shutdownmarker.Exists(shutdownmarker.GetPath(cfg.ShutdownMarkerDir))
	require.NoError(t, err)
	assert.True(t, exists, "POST must create the marker file")

	rec = do(http.MethodGet)
	assert.Equal(t, "set\n", rec.Body.String())

	rec = do(http.MethodDelete)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.True(t, g.ringManager.Lifecycler.ShouldKeepInstanceInTheRingOnShutdown(), "DELETE must flip back to keep-in-ring")
	exists, err = shutdownmarker.Exists(shutdownmarker.GetPath(cfg.ShutdownMarkerDir))
	require.NoError(t, err)
	assert.False(t, exists, "DELETE must remove the marker file")

	rec = do(http.MethodGet)
	assert.Equal(t, "unset\n", rec.Body.String())

	rec = do(http.MethodPut)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// TestBloomGateway_ShutdownMarker_SurvivesRestart is the named test-plan
// item: a marker left behind by a previous process (POST landed, then the
// pod restarted before the operator's actual removal) must be re-armed by
// checkShutdownMarker, BEFORE the ring subservices even start -- mirroring
// modules/livestore/live_store.go's own "check the marker first thing"
// ordering.
func TestBloomGateway_ShutdownMarker_SurvivesRestart(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-marker-restart")
	reader := newFakeBackendReader()
	cfg := newTestGatewayConfig(t, store, addr, "bg-marker-restart", filepath.Join(t.TempDir(), "snapshot.bin"))

	// Simulate a marker left over from a previous process's prepare-
	// downscale POST -- created directly via the same package Create uses,
	// entirely independent of ever having started a BloomGateway.
	// MkdirAll first: unlike checkShutdownMarker itself, nothing has
	// created cfg.ShutdownMarkerDir yet at this point in the test.
	require.NoError(t, os.MkdirAll(cfg.ShutdownMarkerDir, 0o700))
	require.NoError(t, shutdownmarker.Create(shutdownmarker.GetPath(cfg.ShutdownMarkerDir)))

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	startAndCleanup(t, g)
	waitReady(t, g)

	assert.False(t, g.ringManager.Lifecycler.ShouldKeepInstanceInTheRingOnShutdown(), "an existing marker must re-arm unregister-on-shutdown before the ring even joins")
}

// TestBloomGateway_GracefulStop_KeepsRingEntryActiveByDefault is the named
// test-plan item: a bare graceful stop (no prepare-downscale) must leave
// the KV entry present AND ACTIVE -- neither absent (today's pre-redesign
// behavior) nor stuck LEAVING (the naive "just flip
// KeepInstanceInTheRingOnShutdown" bug the design note's own § 1
// identifies: ring.NewLeaveOnStoppingDelegate transitions to LEAVING
// unconditionally). bloomgateway_ring.go's own doc comment on its delegate
// chain explains why the fix is to use NO stopping delegate at all, rather
// than a conditional replacement -- BasicLifecycler.stopping() already
// reads KeepInstanceInTheRingOnShutdown natively.
func TestBloomGateway_GracefulStop_KeepsRingEntryActiveByDefault(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-keep-in-ring")
	reader := newFakeBackendReader()
	cfg := newTestGatewayConfig(t, store, addr, "bg-keep-in-ring", filepath.Join(t.TempDir(), "snapshot.bin"))

	g := mustNewTestGateway(t, cfg, "bloom-gateway-0", reader)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), g))
	waitReady(t, g)

	require.NoError(t, services.StopAndAwaitTerminated(context.Background(), g))

	inst, ok := readRingInstanceDesc(t, store, "bloom-gateway-0")
	require.True(t, ok, "a bare graceful stop must NOT unregister the instance from the ring")
	assert.Equal(t, ring.ACTIVE, inst.GetState(), "a bare graceful stop must NOT transition the instance to LEAVING")
}

// TestBloomGateway_GracefulStop_PreparedDownscaleUnregisters is the named
// test-plan item's other half: once prepared via POST, the NEXT graceful
// stop must revert to this package's pre-redesign net effect -- the
// instance leaves the ring -- exactly as every stop did before 2026-07-16.
// Unlike the pre-redesign path, this does NOT go through LEAVING first
// (bloomgateway_ring.go's own doc comment: there is no way to reach it
// from a delegate outside the ring package once stopping() has begun, and
// none is needed here). This test captures this instance's own log output
// (live-store's own "replace the logger with one writing to a buffer"
// pattern, modules/livestore/instance_test.go) specifically to guard
// against a regression of that class: an earlier revision of this
// package's delegate chain attempted the LEAVING transition anyway via the
// only mechanism available from outside the ring package, which always
// failed and logged an error on every single prepared stop. A correct
// implementation logs nothing of the kind.
func TestBloomGateway_GracefulStop_PreparedDownscaleUnregisters(t *testing.T) {
	store, addr := newTestGatewayCluster(t, "bg-prepared-stop")
	reader := newFakeBackendReader()
	cfg := newTestGatewayConfig(t, store, addr, "bg-prepared-stop", filepath.Join(t.TempDir(), "snapshot.bin"))

	logBuf := &syncBuffer{}
	g, err := New(cfg, "bloom-gateway-0", reader, log.NewLogfmtLogger(logBuf), prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), g))
	waitReady(t, g)

	rec := httptest.NewRecorder()
	g.PrepareDownscaleHandler(rec, httptest.NewRequest(http.MethodPost, "/bloom-gateway/prepare-downscale", nil))
	require.Equal(t, http.StatusNoContent, rec.Code)

	require.NoError(t, services.StopAndAwaitTerminated(context.Background(), g))

	_, ok := readRingInstanceDesc(t, store, "bloom-gateway-0")
	assert.False(t, ok, "a stop after prepare-downscale must unregister the instance from the ring")
	assert.NotContains(t, logBuf.String(), "failed to change instance state", "a prepared stop must never attempt (and fail) a LEAVING transition -- see bloomgateway_ring.go's own doc comment on why no stopping delegate exists at all")
}

// TestBloomGateway_GracefulRestart_DoesNotReassignOwnership is the named
// test-plan item exercising the whole feature end-to-end across two real
// instances: an unprepared graceful stop must not make the survivor
// observe a newly owned range (no reconstruction work at all) while the
// restarting instance is down, and the full restart cycle must leave
// ownership and the survivor's reconstruction queue exactly as they were.
func TestBloomGateway_GracefulRestart_DoesNotReassignOwnership(t *testing.T) {
	// Fast ownership-reconcile ticks (TestBloomGateway_MultiInstanceScaleOut's
	// own precedent) so both initial convergence and the "did NOT reassign"
	// assertion below get many real ticks within the test's short window,
	// rather than relying on the production 1s default barely firing once.
	prevInterval := ownershipReconcileInterval
	ownershipReconcileInterval = 20 * time.Millisecond
	t.Cleanup(func() { ownershipReconcileInterval = prevInterval })

	store, addr := newTestGatewayCluster(t, "bg-graceful-restart")
	reader := newFakeBackendReader() // no tenants: any reconstruction completes trivially fast, but none is expected at all

	cfg0 := newTestGatewayConfig(t, store, addr, "bg-graceful-restart", filepath.Join(t.TempDir(), "g0.bin"))
	total := uint32(1) << cfg0.D

	g0 := mustNewTestGateway(t, cfg0, "bloom-gateway-0", reader)
	startAndCleanup(t, g0)
	waitReady(t, g0)
	allLeavesComplete(t, g0.dir, cfg0.D)

	cfg1 := newTestGatewayConfig(t, store, addr, "bg-graceful-restart", filepath.Join(t.TempDir(), "g1.bin"))
	g1 := mustNewTestGateway(t, cfg1, "bloom-gateway-1", reader)
	startAndCleanup(t, g1)
	waitReady(t, g1)

	require.Eventually(t, func() bool {
		for idx := range total {
			s0, s1 := g0.dir.State(idx) == LeafComplete, g1.dir.State(idx) == LeafComplete
			if s0 == s1 { // both true (double-served) or both false (unserved)
				return false
			}
		}
		return true
	}, 30*time.Second, 20*time.Millisecond, "every leaf must converge to exactly one owner before the restart under test")

	ownedBefore, err := g1.currentOwnedRanges()
	require.NoError(t, err)
	require.NotEmpty(t, ownedBefore, "instance 1 must own a non-trivial share before g0 restarts")

	// Graceful stop, WITHOUT prepare-downscale -- bloomgateway_ring.go's
	// KeepInstanceInTheRingOnShutdown wiring (this package's new default)
	// must keep g0's ring entry ACTIVE, same tokens, for the whole down
	// window.
	require.NoError(t, services.StopAndAwaitTerminated(context.Background(), g0))

	// Well within newTestGatewayConfig's own 3s ring heartbeat timeout: g1
	// must not observe a newly owned range and must not enqueue any
	// reconstruction for g0's share while g0 is down.
	time.Sleep(time.Second)
	ownedDuring, err := g1.currentOwnedRanges()
	require.NoError(t, err)
	assert.Equal(t, ownedBefore, ownedDuring, "a graceful, unprepared restart must not shift ownership even while the instance is down")
	assert.Zero(t, g1.reconstructionQueue.PendingRanges(), "g1 must not enqueue reconstruction for g0's ranges during a graceful, unprepared restart")

	// g0 "restarts": a fresh process, same instance ID/tokens/config, as a
	// real StatefulSet pod reschedule would produce.
	g0b := mustNewTestGateway(t, cfg0, "bloom-gateway-0", reader)
	startAndCleanup(t, g0b)
	waitReady(t, g0b)

	ownedAfter, err := g1.currentOwnedRanges()
	require.NoError(t, err)
	assert.Equal(t, ownedBefore, ownedAfter, "ownership must be exactly as it was before the restart cycle")
	assert.Zero(t, g1.reconstructionQueue.PendingRanges(), "the full restart cycle must never have triggered any reconstruction on the survivor")
}

// TestBloomGateway_AutoForgetExpiry_EventuallyMovesOwnership is the named
// test-plan item for the death backstop (design note §4): an instance that
// is stopped and NEVER comes back must still, eventually, have its
// keyspace reassigned -- driven by the (test-shortened) ring heartbeat
// timeout alone, per the design note's own finding that reassignment and
// KV forgetting are independent mechanisms -- and its stale KV entry must
// eventually be purged outright by AutoForgetDelegate, strictly after
// reassignment already happened (Validate's own RingAutoForgetTimeout >
// Ring.HeartbeatTimeout ordering constraint, config.go).
func TestBloomGateway_AutoForgetExpiry_EventuallyMovesOwnership(t *testing.T) {
	prevInterval := ownershipReconcileInterval
	ownershipReconcileInterval = 20 * time.Millisecond
	t.Cleanup(func() { ownershipReconcileInterval = prevInterval })

	store, addr := newTestGatewayCluster(t, "bg-auto-forget")
	reader := newFakeBackendReader() // no tenants: reassigned ranges reconstruct trivially fast

	cfg0 := newTestGatewayConfig(t, store, addr, "bg-auto-forget", filepath.Join(t.TempDir(), "g0.bin"))
	// Both floored well above dskit's own documented 1-second heartbeat-
	// timestamp granularity (newTestGatewayConfig's own doc comment above:
	// InstanceDesc.IsHeartbeatHealthy rounds Timestamp via time.Unix(t, 0))
	// -- a HeartbeatTimeout below that floor makes GetAllHealthy flicker
	// unhealthy for a perfectly heartbeating instance from second-boundary
	// rounding alone, which is exactly what made this test flaky before
	// this fix (spurious churn on g0 itself, not the reassignment under
	// test).
	cfg0.Ring.HeartbeatTimeout = 2 * time.Second
	cfg0.RingAutoForgetTimeout = 5 * time.Second
	total := uint32(1) << cfg0.D

	g0 := mustNewTestGateway(t, cfg0, "bloom-gateway-0", reader)
	startAndCleanup(t, g0)
	waitReady(t, g0)
	allLeavesComplete(t, g0.dir, cfg0.D)

	cfg1 := newTestGatewayConfig(t, store, addr, "bg-auto-forget", filepath.Join(t.TempDir(), "g1.bin"))
	cfg1.Ring.HeartbeatTimeout = cfg0.Ring.HeartbeatTimeout
	cfg1.RingAutoForgetTimeout = cfg0.RingAutoForgetTimeout
	g1 := mustNewTestGateway(t, cfg1, "bloom-gateway-1", reader)
	startAndCleanup(t, g1)
	waitReady(t, g1)

	require.Eventually(t, func() bool {
		for idx := range total {
			s0, s1 := g0.dir.State(idx) == LeafComplete, g1.dir.State(idx) == LeafComplete
			if s0 == s1 {
				return false
			}
		}
		return true
	}, 30*time.Second, 20*time.Millisecond, "every leaf must converge to exactly one owner before the test's own stop")

	// Stopped WITHOUT preparing for downscale, and never restarted: stays
	// ACTIVE-but-stale in the ring until the heartbeat timeout excludes it.
	require.NoError(t, services.StopAndAwaitTerminated(context.Background(), g0))

	require.Eventually(t, func() bool {
		for idx := range total {
			if g1.dir.State(idx) != LeafComplete {
				return false
			}
		}
		return true
	}, 15*time.Second, 20*time.Millisecond, "g1 must eventually reconstruct g0's share once the ring heartbeat timeout elapses, even though g0 never unregistered")

	require.Eventually(t, func() bool {
		_, ok := readRingInstanceDesc(t, store, "bloom-gateway-0")
		return !ok
	}, 20*time.Second, 20*time.Millisecond, "the stale entry must eventually be purged from the KV store by AutoForgetDelegate")
}
