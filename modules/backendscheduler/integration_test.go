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

	// Verify file structure based on implementation
	workPath := cfg.LocalWorkPath
	if useSharding {
		// Should have shard files
		foundShardFiles := 0
		for i := range work.ShardCount {
			shardPath := newScheduler.filenameForShard(uint8(i))
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

	// Create scheduler with original work implementation
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
	err = originalScheduler.work.FlushToLocal(ctx, originalScheduler.cfg.LocalWorkPath, nil)
	require.NoError(t, err)

	// Stop original scheduler
	err = originalScheduler.stopping(nil)
	require.NoError(t, err)

	// Create new scheduler with sharding enabled
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
		shardPath := shardedScheduler.filenameForShard(uint8(i))
		if _, err = os.Stat(shardPath); err == nil {
			foundShardFiles++
		}
	}
	require.Greater(t, foundShardFiles, 0, "Should have created shard files")

	t.Logf("Successfully migrated to %d shard files", foundShardFiles)
}

// TestRollback tests that we can move between sharded and non-sharded work
// implementations while maintaining job state.
func TestMigrations(t *testing.T) {
	cases := []struct {
		name           string
		withNewTempDir bool // A new temp dir is created for the second shceduler to force loading from the backend
		shardedFirst   bool // The first scheduler created is sharded
		shardedSecond  bool // The second scheduler created is sharded
		// runThird       bool // If true, a third scheduler is created to verify rollback
		// shardedThird   bool
	}{
		{
			name:           "sharded/original: local rollback",
			withNewTempDir: false,
			shardedFirst:   true,
			shardedSecond:  false,
		},
		// { // TODO: remove me.  This test was ensuring that if we started unsharded, we would load from the backend, but I think that if we don't have local state, we should not expect to load it from the backend, since a migration should clean this up.
		//
		// 	name:           "started sharded with new local dir, rollback from backend",
		// 	withNewTempDir: true, // Use a fresh temp dir to force load from backend
		// 	shardedFirst:   true,
		// 	shardedSecond:  false,
		// },
		// {
		// 	name:           "started sharded with new local dir, rollback from backend, run third",
		// 	withNewTempDir: true, // Use a fresh temp dir to force load from backend
		// 	shardedFirst:   true,
		// 	shardedSecond:  false,
		// 	runThird:       true,  // Run a third scheduler to verify rollback
		// 	shardedThird:   false, // The third scheduler should be sharded
		// },

		{
			name:           "start original, rollback from local",
			withNewTempDir: false,
			shardedFirst:   false,
			shardedSecond:  true,
		},
		// Removed: start_sharded_with_new_local_dir,_rollback_from_backend
		// This scenario doesn't make sense in practice - you wouldn't have
		// legacy format in backend but expect sharded format to load from it
		{
			name:           "sharded/sharded: local rollback",
			withNewTempDir: false,
			shardedFirst:   true,
			shardedSecond:  true,
		},
		{
			name:           "sharded/sharded: backend rollback",
			withNewTempDir: true,
			shardedFirst:   true,
			shardedSecond:  true,
		},
		{
			name:           "original/original: local rollback",
			withNewTempDir: false,
			shardedFirst:   false,
			shardedSecond:  false,
		},
		{
			name:           "original/original: backend rollback",
			withNewTempDir: true,
			shardedFirst:   false,
			shardedSecond:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storeDir := t.TempDir()

			t.Logf("Using store directory: %s", storeDir)

			// Common setup for both schedulers
			var (
				ctx, cancel   = context.WithCancel(context.Background())
				store, rr, ww = newStore(ctx, t, storeDir)
			)

			defer func() {
				cancel()
				store.Shutdown()
			}()

			limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
			require.NoError(t, err)

			// Create the first scheduler
			firstConfig := Config{}
			firstConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			firstConfig.Work.Sharded = tc.shardedFirst
			firstConfig.LocalWorkPath = t.TempDir()

			firstJobs := testSchedulerWithConfig(ctx, t, firstConfig, store, limits, rr, ww, true)
			require.NotEmpty(t, firstJobs, "Should have jobs to test rollback")
			t.Logf("First scheduler created with %d jobs", len(firstJobs))

			secondConfig := Config{}
			secondConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			secondConfig.Work.Sharded = tc.shardedSecond
			secondConfig.LocalWorkPath = firstConfig.LocalWorkPath
			if tc.withNewTempDir {
				// Use a fresh local directory to simulate loading from backend
				secondConfig.LocalWorkPath = t.TempDir()
			}

			secondJobs := testSchedulerWithConfig(ctx, t, secondConfig, store, limits, rr, ww, false)
			// time.Sleep(100 * time.Second)
			require.NotEmpty(t, secondJobs, "Should have jobs to test rollback")

			testJobsEqual(t, firstJobs, secondJobs)

			// if tc.runThird {
			// 	// Create a third scheduler to verify rollback
			// 	thirdConfig := Config{}
			// 	thirdConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			// 	thirdConfig.Work.Sharded = tc.shardedThird
			// 	thirdConfig.LocalWorkPath = secondConfig.LocalWorkPath // Use the same path as the second scheduler
			//
			// 	thirdJobs := testSchedulerWithConfig(ctx, t, thirdConfig, store, limits, rr, ww, false)
			// 	require.NotEmpty(t, thirdJobs, "Should have jobs after rollback")
			//
			// 	testJobsEqual(t, firstJobs, thirdJobs)
			// }
		})
	}
}

func testSchedulerWithConfig(ctx context.Context, t *testing.T, cfg Config, store storage.Store, limits overrides.Interface, rr backend.RawReader, ww backend.RawWriter, pushJobs bool) []*work.Job {
	scheduler, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	// This will reload the existing work
	err = scheduler.starting(ctx)
	require.NoError(t, err)

	defer func() {
		err = scheduler.stopping(nil)
		require.NoError(t, err)
	}()

	// Verify sharding implemetation matches the desired configuration
	require.Equal(t, cfg.Work.Sharded, work.IsSharded(scheduler.work))

	// Flush the work cache locally to create work.json file (simulate normal operation)
	err = scheduler.work.FlushToLocal(ctx, scheduler.cfg.LocalWorkPath, nil)
	require.NoError(t, err)

	err = scheduler.flushWorkCacheToBackend(ctx)
	require.NoError(t, err)

	// We should only push jobs if the flag is set, which should only be true on
	// the first scheduler.  The rest will validate we can reload this work set.
	if pushJobs {
		t.Logf("Pushing jobs to scheduler")
		testJobOperations(ctx, t, scheduler, cfg.Work.Sharded)
		t.Logf("Pushed %d jobs to scheduler", len(scheduler.work.ListJobs()))
	}

	if cfg.Work.Sharded {
		// Verify that shard files were created
		foundShardFiles := 0
		for i := range work.ShardCount {
			shardPath := scheduler.filenameForShard(uint8(i))
			if _, err = os.Stat(shardPath); err == nil {
				foundShardFiles++
			}
		}
		require.Greater(t, foundShardFiles, 0, "Should have at least some shard files")
		t.Logf("Found %d shard files", foundShardFiles)

		// Verify that no legacy work.json file exists
		legacyWorkFile := cfg.LocalWorkPath + "/work.json"
		_, err = os.Stat(legacyWorkFile)
		require.True(t, os.IsNotExist(err), "Should not have legacy work.json file when using sharded work")
	} else {
		// Verify that a legacy work.json file was created during rollback
		legacyWorkFile := cfg.LocalWorkPath + "/work.json"
		_, err = os.Stat(legacyWorkFile)
		require.NoError(t, err, "Should have created legacy work.json file during rollback")

		// Verify that no shard files exist
		for i := range work.ShardCount {
			shardPath := scheduler.filenameForShard(uint8(i))
			_, err = os.Stat(shardPath)
			require.True(t, os.IsNotExist(err), "Should not have shard files when using original work implementation")
		}
	}

	return scheduler.ListJobs()
}

func testJobsEqual(t *testing.T, a, b []*work.Job) {
	require.Len(t, a, len(b), "Job lists should have same length")

	jobMap := make(map[string]*work.Job)
	for _, job := range a {
		jobMap[job.ID] = job
	}

	for _, job := range b {
		originalJob, exists := jobMap[job.ID]
		require.True(t, exists, "Job should exist in original jobs")
		require.Equal(t, originalJob.Tenant(), job.Tenant(), "Job tenant should match")
		require.Equal(t, originalJob.Type, job.Type, "Job type should match")
	}
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
				err := scheduler.work.FlushToLocal(ctx, scheduler.cfg.LocalWorkPath, []string{affectedJob})
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
