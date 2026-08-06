package backendscheduler

import (
	"fmt"
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
	rescanAt := time.Now().Add(time.Hour).UnixNano()
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		// An armed rescan (blocks were busy compacting at submit) must survive a failed cancel,
		// or the deferred redaction of those blocks would be silently lost.
		SkippedCompactionJobIds: []string{"busy-job"},
		RescanAfterUnixNano:     rescanAt,
	}))
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{pendingRedactionJob("j1", tenant, "blk1")}))

	_, err := s.CancelRedaction(user.InjectOrgID(ctx, tenant), &tempopb.CancelRedactionRequest{})
	require.Error(t, err, "cancel must fail if it could not be persisted")
	require.Equal(t, codes.Internal, status.Code(err))

	b := s.work.GetBatch(tenant)
	require.NotNil(t, b, "batch is left in place for retry when persistence failed")
	require.False(t, b.Cancelled, "the cancelled flag is reverted on a persistence failure, so a maintenance tick can't remove the batch before the cancel is durable")
	require.Equal(t, rescanAt, b.RescanAfterUnixNano, "a failed cancel leaves the armed rescan intact, so deferred redaction work is not lost")
	require.Equal(t, []string{"busy-job"}, b.SkippedCompactionJobIds, "the rescan's skipped-job list is preserved on rollback")
}

// TestCancelRedactionRetryPersistsAlreadyPurgedJobs verifies a retried cancel still persists the
// purge. If the first attempt purged the jobs in memory but failed to flush, the retry's purge finds
// nothing left to remove and returns no IDs — the flush must not be skipped on that account, or the
// RPC reports success while the pending jobs are still on disk and a scheduler restart would reload
// them and resume redacting after a "successful" cancel.
func TestCancelRedactionRetryPersistsAlreadyPurgedJobs(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "t-cancel-retry"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "batch-retry", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
	}))
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{
		pendingRedactionJob("r1", tenant, "blk1"),
		pendingRedactionJob("r2", tenant, "blk2"),
	}))
	// Persist the pre-cancel state, so the pending jobs are on disk.
	require.NoError(t, s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, nil))

	// Simulate the first attempt: the purge applied in memory but its flush failed, so the jobs are
	// gone from memory while still present on disk. The retry's purge therefore returns no IDs.
	require.Len(t, s.work.PurgePendingRedactionJobs(tenant), 2)

	resp, err := s.CancelRedaction(user.InjectOrgID(ctx, tenant), &tempopb.CancelRedactionRequest{})
	require.NoError(t, err)
	require.Zero(t, resp.PendingPurged, "nothing left to purge in memory on the retry")

	// The retry must have persisted the purge: a fresh Work loading from disk sees no pending jobs.
	reloaded := work.New(work.Config{})
	require.NoError(t, reloaded.LoadFromLocal(ctx, s.cfg.LocalWorkPath))
	require.Empty(t, reloaded.ListAllPendingJobs(), "a successful cancel must leave no pending redaction jobs on disk, even when the purge was already applied in memory by a failed attempt")
}

// TestNextDropsJobFromCancelledBatch verifies a job that left the pending queue just before a cancel
// is not handed to a worker afterwards. The cancel's purge only reaches jobs still in the pending
// queue, and the batch itself stays until in-flight work drains, so without an explicit check this
// already-dequeued job would be assigned and would redact a block the operator asked to stop. It is
// dropped at assignment — before any work begins, so no output block is orphaned and the 1:1 block
// in:out invariant is unaffected — and its in-flight count is released so the batch can be cleaned up.
func TestNextDropsJobFromCancelledBatch(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	// Keep the assignment wait short: the job under test is dropped, and no other job follows.
	s.cfg.JobTimeout = 200 * time.Millisecond

	tenant := "t-cancel-next"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "batch-next", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		TraceIds: [][]byte{{0x01}},
	}))
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{pendingRedactionJob("n1", tenant, "blk1")}))

	// The job is dequeued (counted in-flight) and then the tenant's redaction is cancelled, so the
	// purge cannot reach it.
	j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	s.work.SetBatchCancelled(tenant, true)
	require.NotNil(t, s.work.GetBatch(tenant), "the batch is still present while in-flight work drains")

	s.mergedJobs <- j

	_, err := s.Next(ctx, &tempopb.NextJobRequest{WorkerId: "w1"})
	require.Error(t, err, "no job is assigned from a cancelled batch")
	require.Equal(t, codes.NotFound, status.Code(err))

	require.False(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION),
		"the dropped job's in-flight count is released, so the cancelled batch can drain and be removed")
}

// TestNextConcurrentWithCancelNoRace drives job assignment against a concurrent cancel — the
// feature's intended use: an operator cancels while workers are still polling for work. Next()
// consults the tenant's batch to decide whether to assign or drop, and CancelRedaction mutates that
// batch, so the two must not touch the same batch memory unsynchronized. Meaningful under -race.
func TestNextConcurrentWithCancelNoRace(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	// Short wait: a dropped job leaves Next() waiting for another that never comes.
	s.cfg.JobTimeout = 20 * time.Millisecond

	tenant := "t-cancel-race"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "batch-race", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		TraceIds: [][]byte{{0x01}},
	}))

	const n = 30
	jobs := make([]*work.Job, n)
	for i := range jobs {
		jobs[i] = pendingRedactionJob(fmt.Sprintf("rj%d", i), tenant, fmt.Sprintf("blk%d", i))
	}
	require.NoError(t, s.work.AddPendingJobs(jobs))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
			if j == nil {
				return
			}
			s.mergedJobs <- j
			// A fresh worker ID each time, so Next() re-reads the batch instead of replaying
			// an already-assigned job via GetJobForWorker.
			_, _ = s.Next(ctx, &tempopb.NextJobRequest{WorkerId: fmt.Sprintf("w%d", i)})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			s.work.SetBatchCancelled(tenant, i%2 == 0)
		}
	}()
	wg.Wait()
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
