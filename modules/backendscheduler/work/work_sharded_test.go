package work

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestShardedRoundTrip(t *testing.T) {
	shardedWork := New(Config{})

	// Add multiple jobs to different shards
	jobs := []*Job{
		createTestJob("test-job-1", tempopb.JobType_JOB_TYPE_COMPACTION),
		createTestJob("test-job-2", tempopb.JobType_JOB_TYPE_COMPACTION),
		createTestJob("test-job-3", tempopb.JobType_JOB_TYPE_RETENTION),
	}

	for _, job := range jobs {
		err := shardedWork.AddJob(job)
		require.NoError(t, err, "failed to add job: %v", err)
	}

	// Verify jobs were added
	require.Equal(t, len(jobs), len(shardedWork.ListJobs()), "unexpected number of jobs before marshal")

	// Marshal the sharded work
	data, err := shardedWork.Marshal()
	require.NoError(t, err, "failed to marshal sharded work")

	// Unmarshal into a new instance
	newShardedWork := New(Config{})
	err = newShardedWork.Unmarshal(data)
	require.NoError(t, err, "failed to unmarshal sharded work")

	// Verify the lengths match
	require.Equal(t, len(shardedWork.ListJobs()), len(newShardedWork.ListJobs()), "sharded work lengths do not match")

	// Verify all jobs are present in the new instance
	originalJobs := shardedWork.ListJobs()
	newJobs := newShardedWork.ListJobs()

	require.Equal(t, len(originalJobs), len(newJobs), "job counts don't match")

	// Create maps for easier comparison
	originalJobMap := make(map[string]*Job)
	for _, job := range originalJobs {
		originalJobMap[job.ID] = job
	}

	newJobMap := make(map[string]*Job)
	for _, job := range newJobs {
		newJobMap[job.ID] = job
	}

	// Verify all original jobs exist in the new instance
	for jobID, originalJob := range originalJobMap {
		newJob, exists := newJobMap[jobID]
		require.True(t, exists, "job %s should exist in unmarshaled instance", jobID)
		require.Equal(t, originalJob.ID, newJob.ID, "job IDs should match")
	}
}

func TestAddJob(t *testing.T) {
	work := New(Config{})

	t.Run("successful add", func(t *testing.T) {
		job := createTestJob("test-job", tempopb.JobType_JOB_TYPE_COMPACTION)
		err := work.AddJob(job)
		require.NoError(t, err)

		// Verify job was added with correct state
		retrievedJob := work.GetJob("test-job")
		require.NotNil(t, retrievedJob)
		require.Equal(t, "test-job", retrievedJob.ID)
		require.Equal(t, tempopb.JobStatus_JOB_STATUS_UNSPECIFIED, retrievedJob.Status)
		require.False(t, retrievedJob.CreatedTime.IsZero())
	})

	t.Run("nil job", func(t *testing.T) {
		err := work.AddJob(nil)
		require.Error(t, err)
		require.Equal(t, ErrJobNil, err)
	})

	t.Run("duplicate job", func(t *testing.T) {
		job1 := createTestJob("duplicate-job", tempopb.JobType_JOB_TYPE_COMPACTION)
		job2 := createTestJob("duplicate-job", tempopb.JobType_JOB_TYPE_COMPACTION)

		err1 := work.AddJob(job1)
		require.NoError(t, err1)

		err2 := work.AddJob(job2)
		require.Error(t, err2)
		require.Equal(t, ErrJobAlreadyExists, err2)
	})
}

func TestJobOperations(t *testing.T) {
	work := New(Config{})
	job := createTestJob("ops-test", tempopb.JobType_JOB_TYPE_COMPACTION)
	err := work.AddJob(job)
	require.NoError(t, err)

	t.Run("start job", func(t *testing.T) {
		work.StartJob("ops-test")
		retrievedJob := work.GetJob("ops-test")
		require.Equal(t, tempopb.JobStatus_JOB_STATUS_RUNNING, retrievedJob.Status)
		require.False(t, retrievedJob.StartTime.IsZero())
	})

	t.Run("complete job", func(t *testing.T) {
		work.CompleteJob("ops-test")
		retrievedJob := work.GetJob("ops-test")
		require.Equal(t, tempopb.JobStatus_JOB_STATUS_SUCCEEDED, retrievedJob.Status)
		require.False(t, retrievedJob.EndTime.IsZero())
	})

	t.Run("set compaction output", func(t *testing.T) {
		output := []string{"output1", "output2"}
		work.SetJobCompactionOutput("ops-test", output)
		retrievedJob := work.GetJob("ops-test")
		require.Equal(t, output, retrievedJob.GetCompactionOutput())
	})

	t.Run("fail job", func(t *testing.T) {
		failJob := createTestJob("fail-test", tempopb.JobType_JOB_TYPE_COMPACTION)
		err = work.AddJob(failJob)
		require.NoError(t, err)
		work.StartJob("fail-test")

		work.FailJob("fail-test")
		retrievedJob := work.GetJob("fail-test")
		require.Equal(t, tempopb.JobStatus_JOB_STATUS_FAILED, retrievedJob.Status)
		require.False(t, retrievedJob.EndTime.IsZero())
	})

	t.Run("remove job", func(t *testing.T) {
		removeJob := createTestJob("remove-test", tempopb.JobType_JOB_TYPE_COMPACTION)
		err = work.AddJob(removeJob)
		require.NoError(t, err)

		// Verify job exists
		require.NotNil(t, work.GetJob("remove-test"))

		// Remove job
		work.RemoveJob("remove-test")

		// Verify job is gone
		require.Nil(t, work.GetJob("remove-test"))
	})
}

func TestListJobs(t *testing.T) {
	work := New(Config{})

	// Add jobs with different creation times
	jobs := []*Job{
		createTestJob("job-1", tempopb.JobType_JOB_TYPE_COMPACTION),
		createTestJob("job-2", tempopb.JobType_JOB_TYPE_COMPACTION),
		createTestJob("job-3", tempopb.JobType_JOB_TYPE_COMPACTION),
	}

	var err error

	for i, job := range jobs {
		err = work.AddJob(job)
		require.NoError(t, err)
		// Manually set creation time for testing sorting
		retrievedJob := work.GetJob(job.ID)
		retrievedJob.CreatedTime = time.Now().Add(time.Duration(i) * time.Second)
	}

	listedJobs := work.ListJobs()
	require.Equal(t, len(jobs), len(listedJobs))

	// Verify jobs are sorted by creation time
	for i := 1; i < len(listedJobs); i++ {
		require.True(t, listedJobs[i-1].CreatedTime.Before(listedJobs[i].CreatedTime) ||
			listedJobs[i-1].CreatedTime.Equal(listedJobs[i].CreatedTime))
	}
}

func TestGetJobForWorker(t *testing.T) {
	var (
		work = New(Config{})
		ctx  = context.Background()
		err  error
	)

	// Add jobs for different workers
	job1 := createTestJob("worker1-job", tempopb.JobType_JOB_TYPE_COMPACTION)
	err = work.AddJob(job1)
	require.NoError(t, err)
	work.StartJob(job1.ID) // Sets status to RUNNING
	retrievedJob1 := work.GetJob(job1.ID)
	retrievedJob1.SetWorkerID("worker-1")

	job2 := createTestJob("worker2-job", tempopb.JobType_JOB_TYPE_COMPACTION)
	err = work.AddJob(job2)
	require.NoError(t, err)
	retrievedJob2 := work.GetJob(job2.ID)
	retrievedJob2.SetWorkerID("worker-2")
	// job2 remains in UNSPECIFIED status

	// Test finding job for worker-1
	foundJob := work.GetJobForWorker(ctx, "worker-1")
	require.NotNil(t, foundJob)
	require.Equal(t, "worker1-job", foundJob.ID)

	// Test finding job for worker-2
	foundJob = work.GetJobForWorker(ctx, "worker-2")
	require.NotNil(t, foundJob)
	require.Equal(t, "worker2-job", foundJob.ID)

	// Test worker with no jobs
	foundJob = work.GetJobForWorker(ctx, "nonexistent-worker")
	require.Nil(t, foundJob)
}

func TestPrune(t *testing.T) {
	cfg := Config{
		PruneAge:       time.Hour,
		DeadJobTimeout: 30 * time.Minute,
	}
	work := New(cfg)
	ctx := context.Background()

	now := time.Now()

	var err error

	// Add old completed job (should be pruned)
	oldCompleted := createTestJob("old-completed", tempopb.JobType_JOB_TYPE_COMPACTION)
	err = work.AddJob(oldCompleted)
	require.NoError(t, err)
	work.CompleteJob(oldCompleted.ID) // Sets status to SUCCEEDED
	retrievedOldCompleted := work.GetJob(oldCompleted.ID)
	retrievedOldCompleted.EndTime = now.Add(-2 * time.Hour)

	// Add recent completed job (should not be pruned)
	recentCompleted := createTestJob("recent-completed", tempopb.JobType_JOB_TYPE_COMPACTION)
	err = work.AddJob(recentCompleted)
	require.NoError(t, err)
	work.CompleteJob(recentCompleted.ID) // Sets status to SUCCEEDED
	retrievedRecentCompleted := work.GetJob(recentCompleted.ID)
	retrievedRecentCompleted.EndTime = now.Add(-10 * time.Minute)

	// Add old running job (should be failed)
	oldRunning := createTestJob("old-running", tempopb.JobType_JOB_TYPE_COMPACTION)
	err = work.AddJob(oldRunning)
	require.NoError(t, err)
	work.StartJob(oldRunning.ID) // Sets status to RUNNING
	retrievedOldRunning := work.GetJob(oldRunning.ID)
	retrievedOldRunning.StartTime = now.Add(-time.Hour)

	// Add recent running job (should remain running)
	recentRunning := createTestJob("recent-running", tempopb.JobType_JOB_TYPE_COMPACTION)
	err = work.AddJob(recentRunning)
	require.NoError(t, err)
	work.StartJob(recentRunning.ID) // Sets status to RUNNING
	retrievedRecentRunning := work.GetJob(recentRunning.ID)
	retrievedRecentRunning.StartTime = now.Add(-10 * time.Minute)

	// Run prune
	work.Prune(ctx)

	// Verify results
	require.Nil(t, work.GetJob("old-completed"), "old completed job should be pruned")
	require.NotNil(t, work.GetJob("recent-completed"), "recent completed job should remain")

	oldRunningJob := work.GetJob("old-running")
	require.NotNil(t, oldRunningJob, "old running job should still exist")
	require.Equal(t, tempopb.JobStatus_JOB_STATUS_FAILED, oldRunningJob.Status, "old running job should be failed")

	recentRunningJob := work.GetJob("recent-running")
	require.NotNil(t, recentRunningJob, "recent running job should remain")
	require.Equal(t, tempopb.JobStatus_JOB_STATUS_RUNNING, recentRunningJob.Status, "recent running job should stay running")
}

func TestShardingMethods(t *testing.T) {
	work := New(Config{}).(*Work)
	var err error

	t.Run("shard ID consistency", func(t *testing.T) {
		jobID := "test-job-123"
		shardID1 := work.GetShardID(jobID)
		shardID2 := work.GetShardID(jobID)
		require.Equal(t, shardID1, shardID2, "shard ID should be consistent")
		require.Less(t, shardID1, ShardCount, "shard ID should be within valid range")
	})

	t.Run("marshal shard", func(t *testing.T) {
		// Add job to a specific shard
		job := createTestJob("shard-test", tempopb.JobType_JOB_TYPE_COMPACTION)
		err = work.AddJob(job)
		require.NoError(t, err)

		shardID := work.GetShardID("shard-test")
		data, err := work.MarshalShard(shardID)
		require.NoError(t, err)
		require.NotEmpty(t, data)
	})

	t.Run("unmarshal shard", func(t *testing.T) {
		// Create shard data
		job := createTestJob("unmarshal-test", tempopb.JobType_JOB_TYPE_COMPACTION)
		err = work.AddJob(job)
		require.NoError(t, err)

		shardID := work.GetShardID("unmarshal-test")
		data, err := work.MarshalShard(shardID)
		require.NoError(t, err)

		// Create new work instance and unmarshal specific shard
		newWork := New(Config{}).(*Work)
		err = newWork.UnmarshalShard(shardID, data)
		require.NoError(t, err)

		// Verify job exists in the correct shard
		retrievedJob := newWork.GetJob("unmarshal-test")
		require.NotNil(t, retrievedJob)
		require.Equal(t, "unmarshal-test", retrievedJob.ID)
	})

	t.Run("shard stats", func(t *testing.T) {
		// Clear and add specific jobs for stats testing
		statsWork := New(Config{}).(*Work)

		var err error

		// Add jobs to create non-uniform distribution
		for i := range 10 {
			job := createTestJob(fmt.Sprintf("stats-job-%d", i), tempopb.JobType_JOB_TYPE_COMPACTION)
			err = statsWork.AddJob(job)
			require.NoError(t, err, "failed to add job for stats")
		}

		stats := statsWork.GetShardStats()
		require.Contains(t, stats, "total_jobs")
		require.Contains(t, stats, "total_shards")
		require.Contains(t, stats, "non_empty_shards")
		require.Contains(t, stats, "avg_jobs_per_shard")
		require.Contains(t, stats, "avg_jobs_per_active_shard")
		require.Contains(t, stats, "shard_sizes")

		require.Equal(t, 10, stats["total_jobs"])
		require.Equal(t, ShardCount, stats["total_shards"])
		require.Greater(t, stats["non_empty_shards"], 0)
	})
}

func TestLocalFileOperations(t *testing.T) {
	work := New(Config{})
	ctx := context.Background()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "work-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Add test jobs
	jobs := []*Job{
		createTestJob("local-job-1", tempopb.JobType_JOB_TYPE_COMPACTION),
		createTestJob("local-job-2", tempopb.JobType_JOB_TYPE_COMPACTION),
		createTestJob("local-job-3", tempopb.JobType_JOB_TYPE_COMPACTION),
	}

	for _, job := range jobs {
		err := work.AddJob(job)
		require.NoError(t, err)
	}

	t.Run("flush all to local", func(t *testing.T) {
		err := work.FlushToLocal(ctx, tmpDir, nil)
		require.NoError(t, err)

		// Verify shard files were created
		files, err := filepath.Glob(filepath.Join(tmpDir, "shard_*.json"))
		require.NoError(t, err)
		require.NotEmpty(t, files)
	})

	t.Run("flush affected to local", func(t *testing.T) {
		affectedDir := filepath.Join(tmpDir, "affected")
		err := os.MkdirAll(affectedDir, 0o700)
		require.NoError(t, err)

		// Flush only specific jobs
		affectedJobs := []string{"local-job-1", "local-job-3"}
		err = work.FlushToLocal(ctx, affectedDir, affectedJobs)
		require.NoError(t, err)

		// Verify only affected shard files were created
		files, err := filepath.Glob(filepath.Join(affectedDir, "shard_*.json"))
		require.NoError(t, err)
		require.NotEmpty(t, files)
	})

	t.Run("load from local", func(t *testing.T) {
		// Create new work instance and load from saved files
		newWork := New(Config{})
		err := newWork.LoadFromLocal(ctx, tmpDir)
		require.NoError(t, err)

		// Verify all jobs were loaded
		require.Equal(t, len(work.ListJobs()), len(newWork.ListJobs()))

		originalJobs := work.ListJobs()
		loadedJobs := newWork.ListJobs()

		require.Equal(t, len(originalJobs), len(loadedJobs))

		// Verify job IDs match
		originalIDs := make(map[string]bool)
		for _, job := range originalJobs {
			originalIDs[job.ID] = true
		}

		for _, job := range loadedJobs {
			require.True(t, originalIDs[job.ID], "loaded job %s should exist in original", job.ID)
		}
	})

	t.Run("load from nonexistent path", func(t *testing.T) {
		nonexistentWork := New(Config{})
		err := nonexistentWork.LoadFromLocal(ctx, "/nonexistent/path")
		require.NoError(t, err) // Should succeed with empty shards
		require.Equal(t, 0, len(nonexistentWork.ListJobs()))
	})
}

func TestConcurrency(t *testing.T) {
	work := New(Config{})
	ctx := context.Background()

	// Test concurrent operations
	t.Run("concurrent adds", func(t *testing.T) {
		const numGoroutines = 10
		const jobsPerGoroutine = 10

		// Add jobs concurrently
		done := make(chan struct{})
		for i := range numGoroutines {
			go func(goroutineID int) {
				defer func() { done <- struct{}{} }()

				for j := range jobsPerGoroutine {
					job := createTestJob(fmt.Sprintf("concurrent-%d-%d", goroutineID, j), tempopb.JobType_JOB_TYPE_COMPACTION)
					err := work.AddJob(job)
					require.NoError(t, err)
				}
			}(i)
		}

		// Wait for all goroutines
		for range numGoroutines {
			<-done
		}

		// Verify all jobs were added (should be numGoroutines * jobsPerGoroutine)
		require.Equal(t, numGoroutines*jobsPerGoroutine, len(work.ListJobs()))
	})

	t.Run("concurrent operations", func(t *testing.T) {
		var err error

		// Add initial jobs
		for i := range 50 {
			job := createTestJob(fmt.Sprintf("ops-job-%d", i), tempopb.JobType_JOB_TYPE_COMPACTION)
			err = work.AddJob(job)
			require.NoError(t, err)
		}

		// Perform concurrent operations
		const numWorkers = 5
		done := make(chan struct{})

		for i := range numWorkers {
			go func(workerID int) {
				defer func() { done <- struct{}{} }()

				// Perform various operations
				jobs := work.ListJobs()
				if len(jobs) > 0 {
					job := jobs[workerID%len(jobs)]
					work.StartJob(job.ID)
					work.GetJob(job.ID)
					if workerID%2 == 0 {
						work.CompleteJob(job.ID)
					} else {
						work.FailJob(job.ID)
					}
				}
			}(i)
		}

		// Wait for all workers
		for range numWorkers {
			<-done
		}

		// Verify operations completed without panic
		jobs := work.ListJobs()
		require.NotEmpty(t, jobs)
	})

	t.Run("concurrent prune", func(t *testing.T) {
		cfg := Config{
			PruneAge:       time.Minute,
			DeadJobTimeout: time.Minute,
		}
		pruneWork := New(cfg)

		var err error

		// Add jobs with old timestamps
		for i := range 20 {
			job := createTestJob(fmt.Sprintf("prune-job-%d", i), tempopb.JobType_JOB_TYPE_COMPACTION)
			err = pruneWork.AddJob(job)
			require.NoError(t, err)
			pruneWork.CompleteJob(job.ID) // Sets status to SUCCEEDED
			retrievedJob := pruneWork.GetJob(job.ID)
			retrievedJob.EndTime = time.Now().Add(-2 * time.Hour)
		}

		// Run prune concurrently
		const numPruners = 3
		done := make(chan struct{})

		for range numPruners {
			go func() {
				defer func() { done <- struct{}{} }()
				pruneWork.Prune(ctx)
			}()
		}

		// Wait for all pruners
		for range numPruners {
			<-done
		}

		// Verify jobs were pruned without race conditions
		require.Equal(t, 0, len(pruneWork.ListJobs()))
	})
}

func TestEdgeCases(t *testing.T) {
	work := New(Config{})

	t.Run("operations on nonexistent jobs", func(t *testing.T) {
		// These should not panic
		work.StartJob("nonexistent")
		work.CompleteJob("nonexistent")
		work.FailJob("nonexistent")
		work.SetJobCompactionOutput("nonexistent", []string{"output"})
		work.RemoveJob("nonexistent")

		job := work.GetJob("nonexistent")
		require.Nil(t, job)
	})

	t.Run("empty work operations", func(t *testing.T) {
		emptyWork := New(Config{})

		require.Equal(t, 0, len(emptyWork.ListJobs()))
		require.Empty(t, emptyWork.ListJobs())

		ctx := context.Background()
		foundJob := emptyWork.GetJobForWorker(ctx, "any-worker")
		require.Nil(t, foundJob)

		// Prune should not panic on empty work
		emptyWork.Prune(ctx)

		// Marshal/unmarshal empty work
		data, err := emptyWork.Marshal()
		require.NoError(t, err)

		newWork := New(Config{})
		err = newWork.Unmarshal(data)
		require.NoError(t, err)
		require.Equal(t, 0, len(newWork.ListJobs()))
	})

	t.Run("unmarshal invalid data", func(t *testing.T) {
		newWork := New(Config{})
		err := newWork.Unmarshal([]byte("invalid json"))
		require.Error(t, err)
	})
}

// TestFullMarshalUnmarshal tests the full work cache serialization across all shards
func TestFullMarshalUnmarshal(t *testing.T) {
	work := New(Config{})
	var err error

	// Create jobs with variety in states for comprehensive testing
	totalJobs := 20
	expectedJobsByID := make(map[string]*Job)
	pendingJobCount := 0

	for i := range totalJobs {
		jobID := fmt.Sprintf("marshal-test-job-%d", i)
		job := createTestJob(jobID, tempopb.JobType_JOB_TYPE_COMPACTION)

		// Add variety in job states
		switch i % 4 {
		case 0:
			err = work.AddJob(job)
			require.NoError(t, err)
			pendingJobCount++ // This job remains pending
		case 1:
			err = work.AddJob(job)
			require.NoError(t, err)
			work.StartJob(jobID)
			// Running jobs are not counted as "pending" by Len()
		case 2:
			err = work.AddJob(job)
			require.NoError(t, err)
			work.StartJob(jobID)
			work.CompleteJob(jobID)
			// Completed jobs are not counted as "pending" by Len()
		case 3:
			err = work.AddJob(job)
			require.NoError(t, err)
			work.StartJob(jobID)
			work.FailJob(jobID)
			// Failed jobs are not counted as "pending" by Len()
		}

		expectedJobsByID[jobID] = work.GetJob(jobID)
	}

	t.Logf("Created %d total jobs (%d pending)", totalJobs, pendingJobCount)
	originalJobCount := len(work.ListJobs())
	require.Equal(t, totalJobs, originalJobCount, "Should have created all jobs")

	// Test full marshal
	data, err := work.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	t.Logf("Marshaled data size: %d bytes", len(data))

	// Create new work instance and unmarshal
	newWork := New(Config{})
	err = newWork.Unmarshal(data)
	require.NoError(t, err)

	// Verify all jobs were restored correctly
	require.Equal(t, originalJobCount, len(newWork.ListJobs()))

	// Verify each job's data and state
	for jobID, expectedJob := range expectedJobsByID {
		actualJob := newWork.GetJob(jobID)
		require.NotNil(t, actualJob, "Job %s should exist after unmarshal", jobID)
		require.Equal(t, expectedJob.ID, actualJob.ID)
		require.Equal(t, expectedJob.Type, actualJob.Type)
		require.Equal(t, expectedJob.Status, actualJob.Status)
		require.Equal(t, expectedJob.WorkerID, actualJob.WorkerID)
		require.Equal(t, expectedJob.Tenant(), actualJob.Tenant())
		require.Equal(t, expectedJob.GetCompactionInput(), actualJob.GetCompactionInput())
	}

	// Verify ListJobs returns same jobs (order doesn't matter)
	originalJobs := work.ListJobs()
	newJobs := newWork.ListJobs()
	require.Equal(t, len(originalJobs), len(newJobs))

	originalJobIDs := make(map[string]bool)
	for _, job := range originalJobs {
		originalJobIDs[job.ID] = true
	}

	for _, job := range newJobs {
		require.True(t, originalJobIDs[job.ID], "Job %s should exist in original set", job.ID)
	}
}

// TestMarshalUnmarshalEdgeCases tests edge cases and error conditions
func TestMarshalUnmarshalEdgeCases(t *testing.T) {
	t.Run("empty work cache", func(t *testing.T) {
		work := New(Config{})

		// Marshal empty cache
		data, err := work.Marshal()
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal into new instance
		newWork := New(Config{})
		err = newWork.Unmarshal(data)
		require.NoError(t, err)
		require.Equal(t, 0, len(newWork.ListJobs()))
	})

	t.Run("invalid json unmarshal", func(t *testing.T) {
		work := New(Config{})

		err := work.Unmarshal([]byte("invalid json"))
		require.Error(t, err)
	})
}

// TestConcurrentMarshalUnmarshal tests that Marshal/Unmarshal are safe with concurrent shard operations
func TestConcurrentMarshalUnmarshal(t *testing.T) {
	work := New(Config{})

	// Add initial jobs
	for i := range 10 {
		job := createTestJob(fmt.Sprintf("marshal-concurrent-job-%d", i), tempopb.JobType_JOB_TYPE_COMPACTION)
		err := work.AddJob(job)
		require.NoError(t, err)
	}

	// Run concurrent operations
	done := make(chan struct{})
	numWorkers := 5

	// Workers that continuously modify shards
	for i := range numWorkers {
		go func(workerID int) {
			defer func() { done <- struct{}{} }()

			for j := range 20 {
				jobID := fmt.Sprintf("worker-%d-job-%d", workerID, j)
				job := createTestJob(jobID, tempopb.JobType_JOB_TYPE_COMPACTION)

				err := work.AddJob(job)
				require.NoError(t, err)

				work.StartJob(jobID)
				work.CompleteJob(jobID)
				work.RemoveJob(jobID)
			}
		}(i)
	}

	// Worker that continuously marshals/unmarshals
	go func() {
		defer func() { done <- struct{}{} }()

		for range 10 {
			// Marshal current state
			data, err := work.Marshal()
			require.NoError(t, err)
			require.NotEmpty(t, data)

			// Create new work instance and unmarshal
			newWork := New(Config{})
			err = newWork.Unmarshal(data)
			require.NoError(t, err)

			// Basic validation
			require.NotNil(t, newWork)
		}
	}()

	// Wait for all workers to complete
	for range numWorkers + 1 {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out - possible deadlock")
		}
	}

	// Final validation - work should still be functional
	finalJob := createTestJob("final-test", tempopb.JobType_JOB_TYPE_COMPACTION)
	err := work.AddJob(finalJob)
	require.NoError(t, err)

	retrievedJob := work.GetJob("final-test")
	require.NotNil(t, retrievedJob)
	require.Equal(t, "final-test", retrievedJob.ID)
}

func createTestJob(id string, jobType tempopb.JobType) *Job {
	return &Job{
		ID:   id,
		Type: jobType,
		JobDetail: tempopb.JobDetail{
			Tenant: "test-tenant",
			Compaction: &tempopb.CompactionDetail{
				Input: []string{"block1", "block2"},
			},
		},
	}
}
