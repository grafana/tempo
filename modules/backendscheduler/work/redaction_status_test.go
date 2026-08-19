package work

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

func batchJob(id, tenantID, batchID, blockID string) *Job {
	return &Job{
		ID:   id,
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:  tenantID,
			BatchId: batchID,
			Redaction: &tempopb.RedactionDetail{
				BlockId: blockID,
			},
		},
	}
}

// TestRedactionJobCountsSpansBothJobMaps is the reason this method exists: pending jobs live in
// shard.Pending and active/terminal jobs in shard.Jobs, so anything walking only one of them
// (ListJobs, ListAllPendingJobs) undercounts a batch mid-flight.
func TestRedactionJobCountsSpansBothJobMaps(t *testing.T) {
	w := New(Config{}).(*Work)
	const tenant, batch = "t", "b"

	require.NoError(t, w.AddPendingJobs([]*Job{
		batchJob("queued1", tenant, batch, "blk1"),
		batchJob("queued2", tenant, batch, "blk2"),
	}))

	running := batchJob("running", tenant, batch, "blk3")
	running.SetWorkerID("w1")
	require.NoError(t, w.AddJob(running))
	w.StartJob("running")

	done := batchJob("done", tenant, batch, "blk4")
	done.SetWorkerID("w2")
	require.NoError(t, w.AddJob(done))
	w.StartJob("done")
	w.CompleteJob("done")

	failed := batchJob("failed", tenant, batch, "blk5")
	failed.SetWorkerID("w3")
	require.NoError(t, w.AddJob(failed))
	w.StartJob("failed")
	w.FailJob("failed")

	got := w.RedactionJobCounts(tenant, batch)
	require.Equal(t, 3, got.Remaining, "two queued plus one running")
	require.Equal(t, 1, got.Running, "only the job assigned to a worker")
	require.Equal(t, 1, got.Failed)
}

// TestRedactionJobCountsIncludesInFlight covers the window where a job has been dequeued by
// NextPendingJob but not yet promoted by AddJob: it is in neither shard map, so a count built
// only from the two maps loses it and reports the batch closer to done than it is.
func TestRedactionJobCountsIncludesInFlight(t *testing.T) {
	w := New(Config{}).(*Work)
	const tenant, batch = "t", "b"

	require.NoError(t, w.AddPendingJobs([]*Job{batchJob("j1", tenant, batch, "blk1")}))
	require.Equal(t, 1, w.RedactionJobCounts(tenant, batch).Remaining)

	j := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)

	require.Equal(t, 1, w.RedactionJobCounts(tenant, batch).Remaining,
		"a dequeued-but-unpromoted job is still outstanding work")
}

// TestRedactionJobCountsUnaffectedByPrune is why progress is reported as created-minus-remaining
// rather than as a count of completions: Prune deletes terminal jobs while a long batch is still
// running, so surviving completions are not a stable numerator.
func TestRedactionJobCountsUnaffectedByPrune(t *testing.T) {
	w := New(Config{}).(*Work)
	const tenant, batch = "t", "b"

	require.NoError(t, w.AddPendingJobs([]*Job{batchJob("queued", tenant, batch, "blk1")}))

	done := batchJob("done", tenant, batch, "blk2")
	done.SetWorkerID("w1")
	require.NoError(t, w.AddJob(done))
	w.StartJob("done")
	w.CompleteJob("done")

	before := w.RedactionJobCounts(tenant, batch)
	w.Prune(context.Background())
	require.Nil(t, w.GetJob("done"), "PruneAge 0 retires the completed job immediately")

	require.Equal(t, before.Remaining, w.RedactionJobCounts(tenant, batch).Remaining,
		"pruning a completed job must not change how much work is left")
}

// TestRedactionJobCountsScopedToTenantAndBatch guards against counting a different tenant's work,
// or a stale batch's, into this batch's progress.
func TestRedactionJobCountsScopedToTenantAndBatch(t *testing.T) {
	w := New(Config{}).(*Work)

	require.NoError(t, w.AddPendingJobs([]*Job{
		batchJob("mine", "t", "b", "blk1"),
		batchJob("other-tenant", "other", "b", "blk2"),
		batchJob("other-batch", "t", "b2", "blk3"),
	}))

	got := w.RedactionJobCounts("t", "b")
	require.Equal(t, 1, got.Remaining, "only this tenant's jobs for this batch")
}
