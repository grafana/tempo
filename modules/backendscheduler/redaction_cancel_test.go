package backendscheduler

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
)

func pendingRedactionJob(id, tenant, blockID string) *work.Job {
	return &work.Job{
		ID:   id,
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			Redaction: &tempopb.RedactionDetail{BlockId: blockID},
		},
	}
}

// TestCancelledBatchRemovedImmediately verifies a cancelled batch, once drained, is removed at
// once (like a dry-run) rather than entering quiescence — nothing more will be written, so there
// is no cleanup-window race to hold compaction for.
func TestCancelledBatchRemovedImmediately(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "t-cancel"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
	}))
	s.work.SetBatchCancelled(tenant, true)

	s.cleanupBatchIfDone(ctx, tenant)
	require.Nil(t, s.work.GetBatch(tenant), "a cancelled batch is removed immediately once drained")
}

// TestCancelRedactionPurgesPendingAndMarksCancelled verifies the handler stops the backlog: it
// marks the batch cancelled, purges the pending jobs, and — while an in-flight job is still
// draining — leaves the batch in place (so that job finishes and in:out stays 1:1).
func TestCancelRedactionPurgesPendingAndMarksCancelled(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "t-cancel2"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "batch-2", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		RescanAfterUnixNano: time.Now().Add(time.Hour).UnixNano(),
	}))
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{
		pendingRedactionJob("j1", tenant, "blk1"),
		pendingRedactionJob("j2", tenant, "blk2"),
		pendingRedactionJob("j3", tenant, "blk3"),
	}))
	// Dispatch one job so it is in flight (popped from pending) — the batch must not be removed
	// while it is still draining.
	require.NotNil(t, s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION))

	resp, err := s.CancelRedaction(user.InjectOrgID(ctx, tenant), &tempopb.CancelRedactionRequest{})
	require.NoError(t, err)
	require.Equal(t, "batch-2", resp.BatchId)
	require.Equal(t, int32(2), resp.PendingPurged, "the two still-pending jobs are purged")

	b := s.work.GetBatch(tenant)
	require.NotNil(t, b, "batch remains while the in-flight job drains")
	require.True(t, b.Cancelled, "batch is marked cancelled")
	require.Zero(t, b.RescanAfterUnixNano, "cancel clears any armed rescan")
	require.True(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION), "the in-flight job keeps the batch active")
}

// TestCancelRedactionRemovesDrainedBatchImmediately verifies that when a cancel leaves no in-flight
// work (only pending jobs, now purged), the batch is removed at once and compaction resumes —
// without waiting for a maintenance tick.
func TestCancelRedactionRemovesDrainedBatchImmediately(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "t-cancel3"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "batch-3", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
	}))
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{
		pendingRedactionJob("k1", tenant, "blkx"),
	}))

	resp, err := s.CancelRedaction(user.InjectOrgID(ctx, tenant), &tempopb.CancelRedactionRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.PendingPurged)
	require.Nil(t, s.work.GetBatch(tenant), "a cancel with only pending work removes the batch immediately")
	require.False(t, s.work.TenantPending(tenant), "compaction resumes right away")
}

// TestCancelRedactionFailsIfPurgeNotPersisted verifies that if the purge can't be flushed to the
// work cache, the RPC fails and the batch is left in place — so the operator can retry rather than
// believe the backlog is stopped, and the batch isn't removed unless the purge is durable.
func TestCancelRedactionFailsIfPurgeNotPersisted(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	// Point the work path at a child of a regular file so FlushToLocal's MkdirAll fails.
	notADir := filepath.Join(t.TempDir(), "notadir")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0o600))
	s.cfg.LocalWorkPath = filepath.Join(notADir, "sub")

	tenant := "t-flushfail"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
	}))
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{pendingRedactionJob("j1", tenant, "blk1")}))

	_, err := s.CancelRedaction(user.InjectOrgID(ctx, tenant), &tempopb.CancelRedactionRequest{})
	require.Error(t, err, "cancel must fail if it could not be persisted")
	require.Equal(t, codes.Internal, status.Code(err))

	b := s.work.GetBatch(tenant)
	require.NotNil(t, b, "batch is left in place for retry when persistence failed")
	require.False(t, b.Cancelled, "the cancelled flag is reverted on a persistence failure, so a maintenance tick can't remove the batch before the cancel is durable")
}

// TestCancelledBatchStaleRescanClearedNotRun verifies checkPendingRescans never rescans a
// cancelled batch: it clears any stale armed rescan (e.g. reloaded from a manifest written before
// the cancel cleared it) instead of enqueuing jobs, then the batch drains and is removed.
func TestCancelledBatchStaleRescanClearedNotRun(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "t-cancel-stale-rescan"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Cancelled:               true,
		SkippedCompactionJobIds: []string{"stale-job"},
		RescanAfterUnixNano:     time.Now().Add(-time.Hour).UnixNano(),
	}))

	s.checkPendingRescans(ctx)

	b := s.work.GetBatch(tenant)
	require.NotNil(t, b)
	require.Zero(t, b.RescanAfterUnixNano, "a cancelled batch's stale rescan is cleared, not run")
	require.False(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION), "no redaction jobs are enqueued for a cancelled batch")

	s.cleanupOrphanedBatches(ctx)
	require.Nil(t, s.work.GetBatch(tenant), "cancelled batch drains once its stale rescan is cleared")
}

// TestCheckPendingRescansConcurrentWithCancel runs the maintenance rescan sweep concurrently with
// cancel-path mutations for the same tenant. Under -race it guards against reading batch fields
// (RescanAfterUnixNano, Cancelled) off the live ListBatches pointer while the locked setters write
// them — ListBatches must hand out a snapshot copy.
func TestCheckPendingRescansConcurrentWithCancel(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "t-rescan-race"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		SkippedCompactionJobIds: []string{"j"},
		RescanAfterUnixNano:     time.Now().Add(-time.Hour).UnixNano(),
	}))

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				s.checkPendingRescans(ctx)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			s.work.SetBatchCancelled(tenant, i%2 == 0)
			s.work.SetBatchRescan(tenant, []string{"j"}, time.Now().Add(-time.Hour).UnixNano())
		}
		close(stop)
	}()

	wg.Wait()
}

// TestCancelRedactionNoBatchReturnsNotFound verifies cancelling a tenant with no active redaction
// is a clean NotFound.
func TestCancelRedactionNoBatchReturnsNotFound(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	_, err := s.CancelRedaction(user.InjectOrgID(ctx, "t-none"), &tempopb.CancelRedactionRequest{})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}
