package work

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TestPurgePendingRedactionJobsIsTenantScoped verifies cancel's core primitive removes only the
// target tenant's pending redaction jobs, leaving other tenants' concurrent redactions untouched.
func TestPurgePendingRedactionJobsIsTenantScoped(t *testing.T) {
	w := New(Config{}).(*Work)
	require.NoError(t, w.AddPendingJobs([]*Job{
		createRedactionJob("a1", "tenant-a", "blk-a1"),
		createRedactionJob("a2", "tenant-a", "blk-a2"),
		createRedactionJob("a3", "tenant-a", "blk-a3"),
		createRedactionJob("b1", "tenant-b", "blk-b1"),
		createRedactionJob("b2", "tenant-b", "blk-b2"),
	}))
	require.Equal(t, 3, countPendingForTenant(w, "tenant-a"))
	require.Equal(t, 2, countPendingForTenant(w, "tenant-b"))

	purged := w.PurgePendingRedactionJobs("tenant-a")

	require.Equal(t, 3, purged, "returns the number of pending jobs removed")
	require.Equal(t, 0, countPendingForTenant(w, "tenant-a"), "target tenant's pending redaction jobs are gone")
	require.False(t, w.HasJobsForTenant("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION))
	require.Equal(t, 2, countPendingForTenant(w, "tenant-b"), "a concurrent tenant's redaction is untouched")
	require.True(t, w.HasJobsForTenant("tenant-b", tempopb.JobType_JOB_TYPE_REDACTION))
}

// TestPurgePendingRedactionJobsFreesBlocks verifies purged jobs release their pending-block index
// entries, so the blocks are no longer reported busy.
func TestPurgePendingRedactionJobsFreesBlocks(t *testing.T) {
	w := New(Config{}).(*Work)
	require.NoError(t, w.AddPendingJobs([]*Job{
		createRedactionJob("a1", "tenant-a", "blk-a1"),
	}))
	require.True(t, w.IsBlockBusy("tenant-a", "blk-a1"))
	w.PurgePendingRedactionJobs("tenant-a")
	require.False(t, w.IsBlockBusy("tenant-a", "blk-a1"), "purged job's block is no longer busy")
}
