package work

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func createRedactionJob(id, tenantID, blockID string, traceIDs [][]byte) *Job {
	return &Job{
		ID:   id,
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: tenantID,
			Redaction: &tempopb.RedactionDetail{
				BlockId:  blockID,
				TraceIds: traceIDs,
			},
		},
	}
}

func TestAddPendingJobs(t *testing.T) {
	w := New(Config{}).(*Work)

	t.Run("add and list", func(t *testing.T) {
		jobs := []*Job{
			createRedactionJob("r1", "tenant-a", "block-1", nil),
			createRedactionJob("r2", "tenant-a", "block-2", nil),
			createRedactionJob("r3", "tenant-b", "block-1", nil),
		}
		err := w.AddPendingJobs(jobs)
		require.NoError(t, err)

		listA := w.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION)
		require.Len(t, listA, 2)
		listB := w.ListPendingJobs("tenant-b", tempopb.JobType_JOB_TYPE_REDACTION)
		require.Len(t, listB, 1)
		require.True(t, w.HasPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION))
		require.True(t, w.HasPendingJobs("tenant-b", tempopb.JobType_JOB_TYPE_REDACTION))
		require.False(t, w.HasPendingJobs("tenant-c", tempopb.JobType_JOB_TYPE_REDACTION))
	})

	t.Run("idempotent add same job id", func(t *testing.T) {
		w2 := New(Config{}).(*Work)
		j := createRedactionJob("same-id", "t", "b", nil)
		require.NoError(t, w2.AddPendingJobs([]*Job{j}))
		require.NoError(t, w2.AddPendingJobs([]*Job{j})) // same job again
		require.Len(t, w2.ListPendingJobs("t", tempopb.JobType_JOB_TYPE_REDACTION), 1)
	})
}

func TestBlockPending(t *testing.T) {
	w := New(Config{}).(*Work)

	require.False(t, w.BlockPending("tenant-a", "block-1"))

	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1", nil),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	require.True(t, w.BlockPending("tenant-a", "block-1"))
	require.False(t, w.BlockPending("tenant-a", "block-2"))
	require.False(t, w.BlockPending("tenant-b", "block-1"))

	w.RemovePending("r1")
	require.False(t, w.BlockPending("tenant-a", "block-1"))
}

func TestRemovePending(t *testing.T) {
	w := New(Config{}).(*Work)
	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1", nil),
		createRedactionJob("r2", "tenant-a", "block-2", nil),
	}
	require.NoError(t, w.AddPendingJobs(jobs))
	require.Len(t, w.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION), 2)

	w.RemovePending("r1")
	require.Len(t, w.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION), 1)
	require.False(t, w.BlockPending("tenant-a", "block-1"))
	require.True(t, w.BlockPending("tenant-a", "block-2"))

	w.RemovePending("r2")
	require.Len(t, w.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION), 0)
	require.False(t, w.BlockPending("tenant-a", "block-2"))
}

func TestPopNextPendingRedactionJob_DrainsOneTenant(t *testing.T) {
	w := New(Config{}).(*Work)
	jobs := []*Job{
		createRedactionJob("r-a1", "tenant-a", "block-1", nil),
		createRedactionJob("r-a2", "tenant-a", "block-2", nil),
		createRedactionJob("r-b1", "tenant-b", "block-1", nil),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	// Pop should drain tenant-a first (order of first pop is undefined, but then same tenant)
	var poppedTenants []string
	for range 3 {
		j := w.PopNextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
		require.NotNil(t, j)
		poppedTenants = append(poppedTenants, j.JobDetail.Tenant)
	}
	// tenant-a should be drained before we see tenant-b
	require.Equal(t, "tenant-a", poppedTenants[0])
	require.Equal(t, "tenant-a", poppedTenants[1])
	require.Equal(t, "tenant-b", poppedTenants[2])

	require.Nil(t, w.PopNextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION))
	require.Len(t, w.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION), 0)
	require.Len(t, w.ListPendingJobs("tenant-b", tempopb.JobType_JOB_TYPE_REDACTION), 0)
}

func TestPendingRoundTrip_FlushAndLoad(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	w := New(Config{}).(*Work)

	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1", nil),
		createRedactionJob("r2", "tenant-a", "block-2", nil),
	}
	require.NoError(t, w.AddPendingJobs(jobs))
	jobIDs := []string{"r1", "r2"}
	require.NoError(t, w.FlushToLocal(ctx, tmpDir, jobIDs))

	// Load into new instance
	w2 := New(Config{}).(*Work)
	require.NoError(t, w2.LoadFromLocal(ctx, tmpDir))

	require.Len(t, w2.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION), 2)
	require.True(t, w2.BlockPending("tenant-a", "block-1"))
	require.True(t, w2.BlockPending("tenant-a", "block-2"))
}

func TestPendingRoundTrip_MarshalUnmarshal(t *testing.T) {
	w := New(Config{}).(*Work)
	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1", nil),
		createRedactionJob("r2", "tenant-b", "block-1", nil),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	data, err := w.Marshal()
	require.NoError(t, err)

	w2 := New(Config{}).(*Work)
	require.NoError(t, w2.Unmarshal(data))

	require.Len(t, w2.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION), 1)
	require.Len(t, w2.ListPendingJobs("tenant-b", tempopb.JobType_JOB_TYPE_REDACTION), 1)
	require.True(t, w2.BlockPending("tenant-a", "block-1"))
	require.True(t, w2.BlockPending("tenant-b", "block-1"))
}

func TestPendingAndActiveJobs_Isolated(t *testing.T) {
	w := New(Config{}).(*Work)

	// Add to active Jobs
	active := createTestJob("active-1", tempopb.JobType_JOB_TYPE_COMPACTION)
	require.NoError(t, w.AddJob(active))

	// Add to Pending
	pending := createRedactionJob("pending-1", "tenant-a", "block-1", nil)
	require.NoError(t, w.AddPendingJobs([]*Job{pending}))

	require.Len(t, w.ListJobs(), 1)
	require.Len(t, w.ListPendingJobs("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION), 1)
	require.NotNil(t, w.GetJob("active-1"))
	require.Nil(t, w.GetJob("pending-1"))
}

func TestLoadFromLocal_RebuildsPendingIndex(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	w := New(Config{}).(*Work)
	jobs := []*Job{
		createRedactionJob("r1", "t", "b1", nil),
	}
	require.NoError(t, w.AddPendingJobs(jobs))
	require.NoError(t, w.FlushToLocal(ctx, tmpDir, []string{"r1"}))

	w2 := New(Config{}).(*Work)
	require.NoError(t, w2.LoadFromLocal(ctx, tmpDir))

	require.True(t, w2.BlockPending("t", "b1"))
	require.Len(t, w2.ListPendingJobs("t", tempopb.JobType_JOB_TYPE_REDACTION), 1)
}
