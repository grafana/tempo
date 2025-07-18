package backendscheduler

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

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
		name        string
		useSharding bool
	}{
		{"original_work", false},
		{"sharded_work", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create config with or without sharding
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			cfg.Work.Sharded = tt.useSharding
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

			testJobOperations(ctx, t, scheduler, tt.useSharding)

			testPersistenceAndRecovery(ctx, t, scheduler, cfg, store, limits, rr, ww, tt.useSharding)
		})
	}
}

func testJobOperations(ctx context.Context, t *testing.T, scheduler *BackendScheduler, useSharding bool) {
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
	require.Len(t, allJobs, jobCount)

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
	if useSharding {
		if shardedWork, ok := work.AsSharded(scheduler.work); ok {
			stats := shardedWork.GetShardStats()
			require.NotNil(t, stats)

			totalJobs, exists := stats["total_jobs"]
			require.True(t, exists)
			require.Equal(t, jobCount, totalJobs)

			t.Logf("Shard distribution stats: %+v", stats)
		} else {
			t.Error("Expected sharded work implementation but got original")
		}
	}
}

func testPersistenceAndRecovery(ctx context.Context, t *testing.T, originalScheduler *BackendScheduler, cfg Config, store storage.Store, limits overrides.Interface, rr backend.RawReader, ww backend.RawWriter, useSharding bool) {
	// Get initial jobs
	initialJobs := originalScheduler.ListJobs()
	require.NotEmpty(t, initialJobs, "Should have jobs to test persistence")

	// Force a flush to disk
	err := originalScheduler.flushWorkCacheOptimized(ctx, nil) // Flush all
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

	// Verify file structure based on implementation
	workPath := cfg.LocalWorkPath
	if useSharding {
		// Should have shard files
		foundShardFiles := 0
		for i := range work.ShardCount {
			shardPath := workPath + "/" + fmt.Sprintf("shard_%03d.json", i)
			if _, err := os.Stat(shardPath); err == nil {
				foundShardFiles++
			}
		}
		require.Greater(t, foundShardFiles, 0, "Should have at least some shard files")
		t.Logf("Found %d shard files", foundShardFiles)
	} else {
		// Should have original work.json
		workFile := workPath + "/work.json"
		_, err := os.Stat(workFile)
		require.NoError(t, err, "Should have work.json file")
	}
}

// TestShardedMigration tests migration from original to sharded format
func TestShardedMigration(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: Create scheduler with original work implementation
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Work.Sharded = false // Start with original
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

	// Create original scheduler
	originalScheduler, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	err = originalScheduler.starting(ctx)
	require.NoError(t, err)

	jobCount := 10

	// Add some jobs
	for i := range jobCount {
		job := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: fmt.Sprintf("tenant-%d", i%3),
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String()},
				},
			},
		}
		err = originalScheduler.work.AddJob(job)
		require.NoError(t, err)
	}

	originalJobs := originalScheduler.ListJobs()
	require.Len(t, originalJobs, jobCount)

	// Flush to create work.json
	err = originalScheduler.flushWorkCacheOptimized(ctx, nil)
	require.NoError(t, err)

	// Stop original scheduler
	err = originalScheduler.stopping(nil)
	require.NoError(t, err)

	// Step 2: Create new scheduler with sharding enabled
	cfg.Work.Sharded = true // Enable sharding

	shardedScheduler, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	// This should trigger migration
	err = shardedScheduler.starting(ctx)
	require.NoError(t, err)

	// Verify migration worked
	migratedJobs := shardedScheduler.ListJobs()
	require.Len(t, migratedJobs, jobCount)

	// Verify all jobs are present
	originalJobMap := make(map[string]*work.Job)
	for _, job := range originalJobs {
		originalJobMap[job.ID] = job
	}

	for _, migratedJob := range migratedJobs {
		originalJob, exists := originalJobMap[migratedJob.ID]
		require.True(t, exists)
		require.Equal(t, originalJob.Tenant(), migratedJob.Tenant())
	}

	// Verify sharded work is being used
	require.True(t, work.IsSharded(shardedScheduler.work))

	// Verify shard files were created
	foundShardFiles := 0
	for i := range 256 {
		shardPath := tmpDir + "/work/" + fmt.Sprintf("shard_%03d.json", i)
		if _, err = os.Stat(shardPath); err == nil {
			foundShardFiles++
		}
	}
	require.Greater(t, foundShardFiles, 0, "Should have created shard files")

	// Verify backup file was created
	backupPath := tmpDir + "/work/work.json.backup"
	_, err = os.Stat(backupPath)
	require.NoError(t, err, "Should have created backup of original work.json")

	t.Logf("Successfully migrated to %d shard files", foundShardFiles)
}

// TestPerformanceComparison compares performance between implementations
func TestPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	const numJobs = 1000
	const numOperations = 100

	implementations := []struct {
		name        string
		useSharding bool
	}{
		{"original", false},
		{"sharded", true},
	}

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			cfg.Work.Sharded = impl.useSharding
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

			// Pre-populate with jobs
			jobIDs := make([]string, numJobs)
			for i := range numJobs {
				job := &work.Job{
					ID:   uuid.New().String(),
					Type: tempopb.JobType_JOB_TYPE_COMPACTION,
					JobDetail: tempopb.JobDetail{
						Tenant: fmt.Sprintf("tenant-%d", i%10),
						Compaction: &tempopb.CompactionDetail{
							Input: []string{uuid.New().String()},
						},
					},
				}
				err := scheduler.work.AddJob(job)
				require.NoError(t, err)
				jobIDs[i] = job.ID
			}

			// Measure flush performance
			start := time.Now()
			for i := range numOperations {
				// Simulate updating a single job (typical Next/UpdateJob pattern)
				affectedJob := jobIDs[i%len(jobIDs)]
				err := scheduler.flushWorkCacheOptimized(ctx, []string{affectedJob})
				require.NoError(t, err)
			}
			elapsed := time.Since(start)

			avgPerOp := elapsed / numOperations
			t.Logf("%s: %d operations took %v (avg %v per operation)",
				impl.name, numOperations, elapsed, avgPerOp)

			// Sharded should be significantly faster
			if impl.useSharding {
				require.Less(t, avgPerOp, time.Millisecond,
					"Sharded implementation should be sub-millisecond per operation")
			}
		})
	}
}
