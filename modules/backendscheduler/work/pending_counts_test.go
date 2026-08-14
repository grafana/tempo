package work

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TestPendingJobCounts covers the queue-depth snapshot that backs the jobs_pending gauge.
//
// Depth is the only signal that can drive scale-up. jobs_active counts work already handed to a
// worker, so it is bounded by the worker count and rises only as capacity rises — reading it to
// decide whether to add capacity is circular. If this undercounts, an autoscaler stops adding
// workers while work is still queued.
func TestPendingJobCounts(t *testing.T) {
	w := New(Config{}).(*Work)

	require.Empty(t, w.PendingJobCounts(), "no jobs means no series at all, not zero-valued ones")

	require.NoError(t, w.AddPendingJobs([]*Job{
		createRedactionJob("r1", "tenant-a", "block-1"),
		createRedactionJob("r2", "tenant-a", "block-2"),
		createRedactionJob("r3", "tenant-b", "block-1"),
		createCompactionJob("c1", "tenant-a", []string{"block-9"}),
	}))

	counts := w.PendingJobCounts()
	require.Equal(t, 2, counts["tenant-a"][tempopb.JobType_JOB_TYPE_REDACTION])
	require.Equal(t, 1, counts["tenant-a"][tempopb.JobType_JOB_TYPE_COMPACTION])
	require.Equal(t, 1, counts["tenant-b"][tempopb.JobType_JOB_TYPE_REDACTION])

	// Dispatching moves a job out of the queue: it is active now, not pending. Counting it in both
	// would hold an autoscaler up after the queue had actually drained.
	require.NotNil(t, w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION))

	counts = w.PendingJobCounts()
	remaining := counts["tenant-a"][tempopb.JobType_JOB_TYPE_REDACTION] + counts["tenant-b"][tempopb.JobType_JOB_TYPE_REDACTION]
	require.Equal(t, 2, remaining, "a dispatched job must leave the pending depth")

	// A drained type disappears rather than reporting zero, which is what lets the gauge's Reset
	// clear the series instead of leaving a stale non-zero value an autoscaler would keep reading.
	for w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION) != nil { //nolint:revive // drain
	}

	counts = w.PendingJobCounts()
	for tenant, byType := range counts {
		_, ok := byType[tempopb.JobType_JOB_TYPE_REDACTION]
		require.False(t, ok, "a drained redaction queue must not appear for tenant %s", tenant)
	}
	require.Equal(t, 1, counts["tenant-a"][tempopb.JobType_JOB_TYPE_COMPACTION], "other job types are unaffected")

	// The dequeue path deletes a queue as it empties, so an empty slice only arises from another
	// path (a rebuild, or a future removal). Reported as-is it would publish a zero-valued series
	// that never clears, which is the same stale-signal problem the gauge's Reset exists to avoid.
	w.pendingMtx.Lock()
	w.pendingByTenant["tenant-c"] = map[tempopb.JobType][]string{tempopb.JobType_JOB_TYPE_RETENTION: {}}
	w.pendingMtx.Unlock()

	_, ok := w.PendingJobCounts()["tenant-c"]
	require.False(t, ok, "an empty queue must produce no series at all")
}
