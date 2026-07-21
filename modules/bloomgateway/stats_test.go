package bloomgateway

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestGatewayForStats builds a minimal *BloomGateway exposing only the
// fields refreshStats reads (dir, reg, cfg, metrics, lastSnapshotUnixNano)
// -- bypassing New() entirely, since refreshStats needs no ring, Kafka, or
// background-service machinery. Mirrors this package's own established
// convention of building components directly for a unit test (e.g.
// newTestSweeper, newTestApplier, newTestServer) rather than standing up
// the full service just to exercise one piece of it.
func newTestGatewayForStats(t *testing.T) (*BloomGateway, *Directory, *Registry) {
	t.Helper()
	dir := NewDirectory(testD)
	reg := NewRegistry()
	m := newMetrics(prometheus.NewRegistry())
	cfg := Config{D: testD, F: testF}
	return &BloomGateway{dir: dir, reg: reg, cfg: cfg, metrics: m}, dir, reg
}

// TestBloomGateway_RefreshStats_BlocksLiveAndEntriesTotal covers
// refreshStats' two registry/directory-sourced gauges end to end.
func TestBloomGateway_RefreshStats_BlocksLiveAndEntriesTotal(t *testing.T) {
	g, dir, reg := newTestGatewayForStats(t)

	idx := uint32(1)
	completeLeaf(t, dir, idx)
	require.True(t, dir.InsertLive(idx, 1, Handle(1)))
	require.True(t, dir.InsertLive(idx, 2, Handle(2)))

	uuid := testUUID(t, 1)
	reg.GetOrCreate(uuid, "tenant-a", time.Now(), time.Now())
	require.NoError(t, reg.CommitLive(uuid, false))

	g.refreshStats()

	assert.Equal(t, float64(1), testutil.ToFloat64(g.metrics.blocksLive))
	assert.Equal(t, float64(2), testutil.ToFloat64(g.metrics.entriesTotal))
}

// TestBloomGateway_RefreshStats_OwnedLeavesByState covers the
// owned_leaves{state} gauge.
func TestBloomGateway_RefreshStats_OwnedLeavesByState(t *testing.T) {
	g, dir, _ := newTestGatewayForStats(t)

	completeLeaf(t, dir, uint32(1))
	_, started := dir.BeginConstructing(uint32(2))
	require.True(t, started)

	g.refreshStats()

	assert.Equal(t, float64(1), testutil.ToFloat64(g.metrics.ownedLeaves.WithLabelValues("complete")))
	assert.Equal(t, float64(1), testutil.ToFloat64(g.metrics.ownedLeaves.WithLabelValues("constructing")))
}

// TestBloomGateway_RefreshStats_SnapshotAge covers snapshot_age_seconds'
// two states: NaN before any snapshot, and a real (approximately correct)
// age once lastSnapshotUnixNano is set.
func TestBloomGateway_RefreshStats_SnapshotAge(t *testing.T) {
	t.Run("NaN before any snapshot", func(t *testing.T) {
		g, _, _ := newTestGatewayForStats(t)
		g.refreshStats()
		assert.True(t, math.IsNaN(testutil.ToFloat64(g.metrics.snapshotAgeSeconds)), "snapshot_age_seconds must be NaN, not 0, before this process has ever loaded or saved a snapshot")
	})

	t.Run("computed age after a snapshot timestamp is recorded", func(t *testing.T) {
		g, _, _ := newTestGatewayForStats(t)
		const age = 90 * time.Second
		g.lastSnapshotUnixNano.Store(time.Now().Add(-age).UnixNano())

		g.refreshStats()

		got := testutil.ToFloat64(g.metrics.snapshotAgeSeconds)
		assert.InDelta(t, age.Seconds(), got, 2, "snapshot_age_seconds must reflect elapsed time since the recorded snapshot instant")
	})
}

// TestBloomGateway_RefreshStats_MissFPRateFormula covers the derived
// miss_fp_rate_estimate = entries_total / 2^(d+f) formula (DESIGN.md §
// Sizing) directly against a hand-computed expectation.
func TestBloomGateway_RefreshStats_MissFPRateFormula(t *testing.T) {
	g, dir, _ := newTestGatewayForStats(t)

	idx := uint32(1)
	completeLeaf(t, dir, idx)
	for i := uint16(0); i < 10; i++ {
		require.True(t, dir.InsertLive(idx, i, Handle(i)+1))
	}

	g.refreshStats()

	want := 10.0 / math.Pow(2, float64(g.cfg.D)+float64(g.cfg.F))
	assert.InDelta(t, want, testutil.ToFloat64(g.metrics.missFPRateEstimate), 1e-12)
}

// TestBloomGateway_RunStatsLoop_PopulatesImmediatelyAndStopsOnCancel covers
// runStatsLoop's own two behaviors that refreshStats' unit tests above
// cannot: gauges are populated immediately on start (not only after the
// first tick), and the loop returns promptly once ctx is cancelled.
func TestBloomGateway_RunStatsLoop_PopulatesImmediatelyAndStopsOnCancel(t *testing.T) {
	g, dir, _ := newTestGatewayForStats(t)
	completeLeaf(t, dir, uint32(1))
	require.True(t, dir.InsertLive(uint32(1), 1, Handle(1)))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		g.runStatsLoop(ctx)
	}()

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(g.metrics.entriesTotal) == 1
	}, time.Second, time.Millisecond, "runStatsLoop must populate gauges immediately, not just after the first 15s tick")

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runStatsLoop did not return promptly after context cancellation")
	}
}
