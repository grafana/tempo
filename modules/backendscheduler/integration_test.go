package backendscheduler

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// TestShardedIntegration verifies that sharded work can be used as a drop-in replacement
func TestShardedIntegration(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"sharded_work"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create config with or without sharding
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			cfg.LocalWorkPath = tmpDir + "/work"

			var (
				ctx, cancel   = context.WithCancel(context.Background())
				store, rr, ww = newStore(ctx, t, tmpDir)
			)
			defer func() {
				cancel()
				store.Shutdown()
			}()

			limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
			require.NoError(t, err)

			scheduler, err := New(cfg, store, limits, rr, ww)
			require.NoError(t, err)

			err = scheduler.starting(ctx)
			require.NoError(t, err)

			testJobOperations(ctx, t, scheduler)

			testPersistenceAndRecovery(ctx, t, scheduler, cfg, store, limits, rr, ww)
		})
	}
}

func testJobOperations(ctx context.Context, t *testing.T, scheduler *BackendScheduler) {
	// Add some jobs
	jobCount := 5
	jobIDs := make([]string, jobCount)

	for i := range jobCount {
		// The scheduler will create jobs through providers, but for testing we'll add them directly
		job := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: "test-tenant",
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err := scheduler.work.AddJob(job)
		require.NoError(t, err)
		jobIDs[i] = job.ID
	}

	// Test job retrieval
	allJobs := scheduler.ListJobs()
	require.Len(t, allJobs, jobCount, "Should have added jobs to the scheduler")

	// Mark all jobs completed
	for _, jobID := range jobIDs {
		// Start the job
		scheduler.work.StartJob(jobID)

		// Complete the job
		_, err := scheduler.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  jobID,
			Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
		})
		require.NoError(t, err)
	}

	// Verify all jobs are completed
	allJobs = scheduler.ListJobs()
	for _, job := range allJobs {
		require.Equal(t, tempopb.JobStatus_JOB_STATUS_SUCCEEDED, job.GetStatus())
	}

	require.Len(t, allJobs, jobCount, "Should have same number of jobs after completion")

	// If sharding is enabled, verify sharding-specific functionality
	stats := scheduler.work.(*work.Work).GetShardStats()
	require.NotNil(t, stats)

	totalJobs, exists := stats["total_jobs"]
	require.True(t, exists)
	require.Equal(t, jobCount, totalJobs)

	t.Logf("Shard distribution stats: %+v", stats)
}

func testPersistenceAndRecovery(ctx context.Context, t *testing.T, originalScheduler *BackendScheduler, cfg Config, store storage.Store, limits overrides.Interface, rr backend.RawReader, ww backend.RawWriter) {
	// Get initial jobs
	initialJobs := originalScheduler.ListJobs()
	require.NotEmpty(t, initialJobs, "Should have jobs to test persistence")

	// Force a flush to disk
	err := originalScheduler.work.FlushToLocal(ctx, originalScheduler.cfg.LocalWorkPath, nil) // Flush all
	require.NoError(t, err)

	// Create a new scheduler instance (simulating restart)
	newScheduler, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	err = newScheduler.starting(ctx)
	require.NoError(t, err)

	// Verify jobs were recovered
	recoveredJobs := newScheduler.ListJobs()
	require.Len(t, recoveredJobs, len(initialJobs))

	// Verify job details match
	initialJobMap := make(map[string]*work.Job)
	for _, job := range initialJobs {
		initialJobMap[job.ID] = job
	}

	for _, recoveredJob := range recoveredJobs {
		originalJob, exists := initialJobMap[recoveredJob.ID]
		require.True(t, exists, "Recovered job should exist in original jobs")
		require.Equal(t, originalJob.GetStatus(), recoveredJob.GetStatus())
		require.Equal(t, originalJob.Tenant(), recoveredJob.Tenant())
	}

	// Should have shard files
	foundShardFiles := 0
	for i := range work.ShardCount {
		shardPath := newScheduler.filepathForShard(uint8(i))
		if _, err := os.Stat(shardPath); err == nil {
			foundShardFiles++
		}
	}
	require.Greater(t, foundShardFiles, 0, "Should have at least some shard files")
	t.Logf("Found %d shard files", foundShardFiles)
}
