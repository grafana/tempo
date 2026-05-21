package work

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func createCompactionJob(id, tenantID string, blockIDs []string) *Job {
	return &Job{
		ID:   id,
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: tenantID,
			Compaction: &tempopb.CompactionDetail{
				Input: blockIDs,
			},
		},
	}
}

func createRedactionJob(id, tenantID, blockID string) *Job {
	return &Job{
		ID:   id,
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: tenantID,
			Redaction: &tempopb.RedactionDetail{
				BlockId: blockID,
			},
		},
	}
}

// countPendingForTenant counts pending redaction jobs for the given tenant.
func countPendingForTenant(w *Work, tenantID string) int {
	count := 0
	for _, j := range w.ListAllPendingJobs() {
		if j.JobDetail.Tenant == tenantID && j.Type == tempopb.JobType_JOB_TYPE_REDACTION {
			count++
		}
	}
	return count
}

func TestAddPendingJobs(t *testing.T) {
	w := New(Config{}).(*Work)

	t.Run("add and list", func(t *testing.T) {
		jobs := []*Job{
			createRedactionJob("r1", "tenant-a", "block-1"),
			createRedactionJob("r2", "tenant-a", "block-2"),
			createRedactionJob("r3", "tenant-b", "block-1"),
		}
		err := w.AddPendingJobs(jobs)
		require.NoError(t, err)

		require.Equal(t, 2, countPendingForTenant(w, "tenant-a"))
		require.Equal(t, 1, countPendingForTenant(w, "tenant-b"))
		require.True(t, w.HasJobsForTenant("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION))
		require.True(t, w.HasJobsForTenant("tenant-b", tempopb.JobType_JOB_TYPE_REDACTION))
		require.False(t, w.HasJobsForTenant("tenant-c", tempopb.JobType_JOB_TYPE_REDACTION))
	})

	t.Run("idempotent add same job id", func(t *testing.T) {
		w2 := New(Config{}).(*Work)
		j := createRedactionJob("same-id", "t", "b")
		require.NoError(t, w2.AddPendingJobs([]*Job{j}))
		require.NoError(t, w2.AddPendingJobs([]*Job{j})) // same job again
		require.Equal(t, 1, countPendingForTenant(w2, "t"))
	})
}

func TestListAllPendingJobs(t *testing.T) {
	w := New(Config{}).(*Work)

	// Empty initially.
	require.Empty(t, w.ListAllPendingJobs())

	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1"),
		createRedactionJob("r2", "tenant-a", "block-2"),
		createRedactionJob("r3", "tenant-b", "block-1"),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	all := w.ListAllPendingJobs()
	require.Len(t, all, 3)

	// All returned jobs must be from the pending set (not the active jobs map).
	ids := make(map[string]bool, len(all))
	for _, j := range all {
		ids[j.ID] = true
	}
	require.True(t, ids["r1"])
	require.True(t, ids["r2"])
	require.True(t, ids["r3"])

	// Active jobs must not appear.
	active := createRedactionJob("active-1", "tenant-a", "block-3")
	require.NoError(t, w.AddJob(active))
	require.Len(t, w.ListAllPendingJobs(), 3)

	// After popping one job the count drops.
	popped := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, popped)
	require.Len(t, w.ListAllPendingJobs(), 2)
}

func TestIsBlockBusy(t *testing.T) {
	w := New(Config{}).(*Work)

	require.False(t, w.IsBlockBusy("tenant-a", "block-1"))

	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1"),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	require.True(t, w.IsBlockBusy("tenant-a", "block-1"))
	require.False(t, w.IsBlockBusy("tenant-a", "block-2"))
	require.False(t, w.IsBlockBusy("tenant-b", "block-1"))

	// Popping removes the block from the pending index.
	popped := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, popped)
	require.False(t, w.IsBlockBusy("tenant-a", "block-1"))
}

func TestNextPendingJob_DrainsPendingQueue(t *testing.T) {
	w := New(Config{}).(*Work)
	jobs := []*Job{
		createRedactionJob("r-a1", "tenant-a", "block-1"),
		createRedactionJob("r-a2", "tenant-a", "block-2"),
		createRedactionJob("r-b1", "tenant-b", "block-1"),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	seen := make(map[string]bool)
	for range 3 {
		j := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
		require.NotNil(t, j)
		seen[j.ID] = true
	}
	require.True(t, seen["r-a1"])
	require.True(t, seen["r-a2"])
	require.True(t, seen["r-b1"])

	require.Nil(t, w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION))
	require.Empty(t, w.ListAllPendingJobs())
}

func TestPendingRoundTrip_FlushAndLoad(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	w := New(Config{}).(*Work)

	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1"),
		createRedactionJob("r2", "tenant-a", "block-2"),
	}
	require.NoError(t, w.AddPendingJobs(jobs))
	jobIDs := []string{"r1", "r2"}
	require.NoError(t, w.FlushToLocal(ctx, tmpDir, jobIDs))

	// Load into new instance
	w2 := New(Config{}).(*Work)
	require.NoError(t, w2.LoadFromLocal(ctx, tmpDir))

	require.Equal(t, 2, countPendingForTenant(w2, "tenant-a"))
	require.True(t, w2.IsBlockBusy("tenant-a", "block-1"))
	require.True(t, w2.IsBlockBusy("tenant-a", "block-2"))
}

func TestPendingRoundTrip_MarshalUnmarshal(t *testing.T) {
	w := New(Config{}).(*Work)
	jobs := []*Job{
		createRedactionJob("r1", "tenant-a", "block-1"),
		createRedactionJob("r2", "tenant-b", "block-1"),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	data, err := w.Marshal()
	require.NoError(t, err)

	w2 := New(Config{}).(*Work)
	require.NoError(t, w2.Unmarshal(data))

	// Shard-scan based checks.
	require.Equal(t, 1, countPendingForTenant(w2, "tenant-a"))
	require.Equal(t, 1, countPendingForTenant(w2, "tenant-b"))
	require.Len(t, w2.ListAllPendingJobs(), 2)

	// pendingBlocks index rebuilt correctly.
	require.True(t, w2.IsBlockBusy("tenant-a", "block-1"))
	require.True(t, w2.IsBlockBusy("tenant-b", "block-1"))

	// pendingByTenant index rebuilt correctly (used by HasJobsForTenant and NextPendingJob).
	require.True(t, w2.HasJobsForTenant("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION))
	require.True(t, w2.HasJobsForTenant("tenant-b", tempopb.JobType_JOB_TYPE_REDACTION))
	require.False(t, w2.HasJobsForTenant("tenant-c", tempopb.JobType_JOB_TYPE_REDACTION))

	j := w2.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	// After popping, the pending queue for this tenant is empty.
	require.Zero(t, countPendingForTenant(w2, j.Tenant()))
	// The block is no longer in the pending index (removed on pop).
	require.False(t, w2.IsBlockBusy(j.Tenant(), j.GetRedactionBlockID()))
}

func TestPendingAndActiveJobs_Isolated(t *testing.T) {
	w := New(Config{}).(*Work)

	// Add to active Jobs
	active := createTestJob("active-1", tempopb.JobType_JOB_TYPE_COMPACTION)
	require.NoError(t, w.AddJob(active))

	// Add to Pending
	pending := createRedactionJob("pending-1", "tenant-a", "block-1")
	require.NoError(t, w.AddPendingJobs([]*Job{pending}))

	require.Len(t, w.ListJobs(), 1)
	require.Equal(t, 1, countPendingForTenant(w, "tenant-a"))
	require.NotNil(t, w.GetJob("active-1"))
	require.Nil(t, w.GetJob("pending-1"))
}

func TestIsBlockBusyRunningLifecycle(t *testing.T) {
	w := New(Config{}).(*Work)

	// 1. AddPendingJobs → block is in pendingBlocks.
	j := createRedactionJob("r1", "tenant-a", "block-1")
	require.NoError(t, w.AddPendingJobs([]*Job{j}))
	require.True(t, w.IsBlockBusy("tenant-a", "block-1"))

	// 2. NextPendingJob → pendingBlocks cleared, runningBlocks not yet set (gap).
	popped := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, popped)
	require.False(t, w.IsBlockBusy("tenant-a", "block-1"))

	// 3. RegisterJob → runningBlocks set.
	w.RegisterJob(popped)
	require.True(t, w.IsBlockBusy("tenant-a", "block-1"))

	// Unrelated block/tenant must not be affected.
	require.False(t, w.IsBlockBusy("tenant-a", "block-2"))
	require.False(t, w.IsBlockBusy("tenant-b", "block-1"))

	// 4. AddJob → registeredJobs cleared, runningBlocks unchanged.
	require.NoError(t, w.AddJob(popped))
	require.True(t, w.IsBlockBusy("tenant-a", "block-1"))

	// 5. StartJob → no index change.
	w.StartJob(popped.ID)
	require.True(t, w.IsBlockBusy("tenant-a", "block-1"))

	// 6. CompleteJob → runningBlocks cleared; block no longer busy.
	w.CompleteJob(popped.ID)
	require.False(t, w.IsBlockBusy("tenant-a", "block-1"))
}

func TestBusyBlocksForTenant(t *testing.T) {
	w := New(Config{}).(*Work)

	// Empty initially.
	require.Empty(t, w.BusyBlocksForTenant("tenant-a"))

	// Pending redaction job → block appears in snapshot.
	j := createRedactionJob("r1", "tenant-a", "block-1")
	require.NoError(t, w.AddPendingJobs([]*Job{j}))
	busy := w.BusyBlocksForTenant("tenant-a")
	require.Contains(t, busy, "block-1")
	// Unrelated tenant must not bleed in.
	require.Empty(t, w.BusyBlocksForTenant("tenant-b"))

	// Popping removes the block from the pending index.
	popped := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, popped)
	// The gap between pop and RegisterJob: block is not in the snapshot.
	require.NotContains(t, w.BusyBlocksForTenant("tenant-a"), "block-1")

	// RegisterJob re-adds via runningBlocks.
	w.RegisterJob(popped)
	require.Contains(t, w.BusyBlocksForTenant("tenant-a"), "block-1")

	// Promote to active via AddJob; block still covered by runningBlocks.
	require.NoError(t, w.AddJob(popped))
	require.Contains(t, w.BusyBlocksForTenant("tenant-a"), "block-1")

	// CompleteJob releases runningBlocks.
	w.CompleteJob(popped.ID)
	require.NotContains(t, w.BusyBlocksForTenant("tenant-a"), "block-1")

	// Compaction job: multiple input blocks tracked via runningBlocks.
	cj := createCompactionJob("c1", "tenant-a", []string{"blk-x", "blk-y", "blk-z"})
	w.RegisterJob(cj)
	busyC := w.BusyBlocksForTenant("tenant-a")
	require.Contains(t, busyC, "blk-x")
	require.Contains(t, busyC, "blk-y")
	require.Contains(t, busyC, "blk-z")
	// Other tenant still empty.
	require.Empty(t, w.BusyBlocksForTenant("tenant-b"))
}

func TestIsBlockBusyCompactionLifecycle(t *testing.T) {
	w := New(Config{}).(*Work)

	blocks := []string{"block-a", "block-b", "block-c"}
	j := createCompactionJob("c1", "tenant-x", blocks)

	assertBusy := func(want bool) {
		t.Helper()
		for _, bid := range blocks {
			require.Equal(t, want, w.IsBlockBusy("tenant-x", bid), "block %s", bid)
		}
	}

	// Before registration nothing is busy.
	assertBusy(false)

	// RegisterJob → all input blocks marked running.
	w.RegisterJob(j)
	assertBusy(true)

	// AddJob → registeredJobs cleared, runningBlocks unchanged.
	require.NoError(t, w.AddJob(j))
	assertBusy(true)

	// StartJob → no index change.
	w.StartJob(j.ID)
	assertBusy(true)

	// CompleteJob → all blocks released.
	w.CompleteJob(j.ID)
	assertBusy(false)

	// FailJob path: register a second job and verify FailJob also clears runningBlocks.
	j2 := createCompactionJob("c2", "tenant-x", blocks)
	w.RegisterJob(j2)
	assertBusy(true)
	require.NoError(t, w.AddJob(j2))
	w.FailJob(j2.ID)
	assertBusy(false)
}

func TestLoadFromLocal_RebuildsPendingIndex(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	w := New(Config{}).(*Work)
	jobs := []*Job{
		createRedactionJob("r1", "t", "b1"),
	}
	require.NoError(t, w.AddPendingJobs(jobs))
	require.NoError(t, w.FlushToLocal(ctx, tmpDir, []string{"r1"}))

	w2 := New(Config{}).(*Work)
	require.NoError(t, w2.LoadFromLocal(ctx, tmpDir))

	require.True(t, w2.IsBlockBusy("t", "b1"))
	require.True(t, w2.HasJobsForTenant("t", tempopb.JobType_JOB_TYPE_REDACTION))
}

// BenchmarkHasJobsForTenant measures HasJobsForTenant with 1000 tenants each having
// active compaction jobs — the old path scanned all 256 shards on every call.
func BenchmarkHasJobsForTenant(b *testing.B) {
	const numTenants = 1000
	w := New(Config{}).(*Work)

	for i := range numTenants {
		tenant := fmt.Sprintf("tenant-%04d", i)
		j := createCompactionJob(fmt.Sprintf("job-%04d", i), tenant, []string{fmt.Sprintf("blk-%04d", i)})
		j.SetWorkerID("worker-1")
		w.RegisterJob(j)
		_ = w.AddJob(j)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		// Query the last tenant — worst case for the old shard scan.
		w.HasJobsForTenant("tenant-0999", tempopb.JobType_JOB_TYPE_COMPACTION)
	}
}

// BenchmarkBusyBlocksForTenant measures BusyBlocksForTenant with blocks spread across
// many tenants — the old path prefix-scanned all entries on every call.
func BenchmarkBusyBlocksForTenant(b *testing.B) {
	const (
		numTenants      = 100
		blocksPerTenant = 100
	)
	w := New(Config{}).(*Work)

	for i := range numTenants {
		tenant := fmt.Sprintf("tenant-%04d", i)
		var blockIDs []string
		for k := range blocksPerTenant {
			blockIDs = append(blockIDs, fmt.Sprintf("blk-%04d-%04d", i, k))
		}
		j := createCompactionJob(fmt.Sprintf("job-%04d", i), tenant, blockIDs)
		w.RegisterJob(j)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		w.BusyBlocksForTenant("tenant-0050")
	}
}
