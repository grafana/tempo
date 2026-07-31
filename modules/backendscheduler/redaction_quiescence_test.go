package backendscheduler

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

// newQuiescenceScheduler builds a scheduler with no providers running (we drive the
// batch lifecycle directly).
func newQuiescenceScheduler(t *testing.T) (context.Context, *BackendScheduler) {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	// cancel BEFORE Shutdown: Shutdown waits on the polling goroutine, which only exits
	// once the ctx is cancelled (a single cleanup keeps that order).
	t.Cleanup(func() {
		cancel()
		store.Shutdown()
	})

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)
	return ctx, s
}

// TestBatchQuiescence verifies a completed batch is held for two maintenance ticks (keeping
// the tenant's compaction blocked) before removal, rather than removed immediately.
func TestBatchQuiescence(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "tenant-quiescence"

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
	}))

	// Completion (no jobs, no rescan): enter quiescence, batch NOT removed, compaction still blocked.
	s.cleanupBatchIfDone(ctx, tenant)
	b := s.work.GetBatch(tenant)
	require.NotNil(t, b, "batch must not be removed the instant jobs finish")
	require.Equal(t, int32(2), b.QuiescenceTicksRemaining, "should enter quiescence with 2 ticks")
	require.True(t, s.work.TenantPending(tenant), "compaction must stay blocked during quiescence")

	// Tick 1: decrement, still present and blocking.
	s.cleanupOrphanedBatches(ctx)
	b = s.work.GetBatch(tenant)
	require.NotNil(t, b)
	require.Equal(t, int32(1), b.QuiescenceTicksRemaining)
	require.True(t, s.work.TenantPending(tenant))

	// Tick 2: reaches zero -> removed, compaction re-enabled.
	s.cleanupOrphanedBatches(ctx)
	require.Nil(t, s.work.GetBatch(tenant), "batch must be removed once quiescence elapses")
	require.False(t, s.work.TenantPending(tenant), "compaction must be re-enabled after quiescence")
}

// TestBatchQuiescenceWaitsForRescan verifies a batch with a pending rescan does not enter
// quiescence (the rescan must run first).
func TestBatchQuiescenceWaitsForRescan(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "tenant-rescan"

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		RescanAfterUnixNano: time.Now().Add(time.Hour).UnixNano(),
	}))

	s.cleanupBatchIfDone(ctx, tenant)
	b := s.work.GetBatch(tenant)
	require.NotNil(t, b, "batch with a pending rescan must not be removed")
	require.Equal(t, int32(0), b.QuiescenceTicksRemaining, "must not enter quiescence while a rescan is pending")
}
