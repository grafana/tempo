package backendscheduler

import (
	"context"
	"flag"
	"sync"
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
	// A positive interval makes the quiesce-until deadline comfortably in the future; tests
	// force expiry by rewinding the deadline rather than sleeping.
	cfg.MaintenanceInterval = time.Minute

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

// TestBatchQuiescence verifies a completed batch records a quiesce-until deadline (keeping the
// tenant's compaction blocked) and is removed only once that deadline passes, not immediately.
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
	require.Positive(t, b.QuiesceUntilUnixNano, "should record a future quiesce-until deadline")
	require.True(t, s.work.TenantPending(tenant), "compaction must stay blocked during quiescence")

	// A sweep before the deadline is a no-op: still present and blocking, and not rewritten.
	s.cleanupOrphanedBatches(ctx)
	b = s.work.GetBatch(tenant)
	require.NotNil(t, b, "batch must remain while within the quiescence window")
	require.True(t, s.work.TenantPending(tenant))

	// Once the deadline passes, the next sweep removes it and re-enables compaction.
	s.work.SetBatchQuiesceUntil(tenant, time.Now().Add(-time.Second).UnixNano())
	s.cleanupOrphanedBatches(ctx)
	require.Nil(t, s.work.GetBatch(tenant), "batch must be removed once the deadline passes")
	require.False(t, s.work.TenantPending(tenant), "compaction must be re-enabled after quiescence")
}

// TestQuiescenceConcurrentAccess exercises the maintenance-tick quiescence sweep concurrently
// with job-completion mutations. Run under -race, it guards against reading batch fields off a
// live pointer without the batch-store lock (the tick and UpdateJob run on different goroutines
// in production).
func TestQuiescenceConcurrentAccess(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "tenant-race"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
	}))

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Maintenance-tick sweep (reads quiescence state, may decrement/remove).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				s.cleanupOrphanedBatches(ctx)
			}
		}
	}()

	// Job-completion path mutating quiescence for the same tenant.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			s.cleanupBatchIfDone(ctx, tenant)
			s.work.SetBatchQuiesceUntil(tenant, int64(i))
		}
		close(stop)
	}()

	wg.Wait()
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
	require.Zero(t, b.QuiesceUntilUnixNano, "must not enter quiescence while a rescan is pending")
}
