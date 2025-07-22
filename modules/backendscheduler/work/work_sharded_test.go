package work

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestShardedRoundTrip(t *testing.T) {
	shardedWork := NewSharded(Config{})

	// Add multiple jobs to different shards
	jobs := []*Job{
		{ID: "test-job-1"},
		{ID: "test-job-2"},
		{ID: "test-job-3"},
	}

	for _, job := range jobs {
		err := shardedWork.AddJob(job)
		require.NoError(t, err, "failed to add job: %v", err)
	}

	// Verify jobs were added
	require.Equal(t, len(jobs), shardedWork.Len(), "unexpected number of jobs before marshal")

	// Marshal the sharded work
	data, err := shardedWork.Marshal()
	require.NoError(t, err, "failed to marshal sharded work")

	// Unmarshal into a new instance
	newShardedWork := NewSharded(Config{})
	err = newShardedWork.Unmarshal(data)
	require.NoError(t, err, "failed to unmarshal sharded work")

	// Verify the lengths match
	require.Equal(t, shardedWork.Len(), newShardedWork.Len(), "sharded work lengths do not match")

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

func TestMigrationRoundTrip(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		jobs []*Job
	}{
		{
			name: "empty_work",
			jobs: nil,
		},
		{
			name: "single_simple_job",
			jobs: []*Job{
				{ID: "simple-job"},
			},
		},
		{
			name: "multiple_jobs_with_different_types",
			jobs: []*Job{
				{
					ID:   "compaction-job",
					Type: tempopb.JobType_JOB_TYPE_COMPACTION,
					JobDetail: tempopb.JobDetail{
						Tenant: "tenant-a",
						Compaction: &tempopb.CompactionDetail{
							Input: []string{"block-1", "block-2"},
						},
					},
				},
				{
					ID:   "retention-job",
					Type: tempopb.JobType_JOB_TYPE_RETENTION,
					JobDetail: tempopb.JobDetail{
						Tenant:    "tenant-b",
						Retention: &tempopb.RetentionDetail{},
					},
				},
				{
					ID:   "simple-compaction",
					Type: tempopb.JobType_JOB_TYPE_COMPACTION,
					JobDetail: tempopb.JobDetail{
						Tenant: "tenant-c",
					},
				},
			},
		},
		{
			name: "jobs_with_different_statuses_and_timing",
			jobs: []*Job{
				{
					ID:        "pending-job",
					Type:      tempopb.JobType_JOB_TYPE_COMPACTION,
					Status:    tempopb.JobStatus_JOB_STATUS_UNSPECIFIED,
					JobDetail: tempopb.JobDetail{Tenant: "tenant-a"},
				},
				{
					ID:        "running-job",
					Type:      tempopb.JobType_JOB_TYPE_RETENTION,
					Status:    tempopb.JobStatus_JOB_STATUS_RUNNING,
					StartTime: now,
					JobDetail: tempopb.JobDetail{Tenant: "tenant-b"},
				},
				{
					ID:        "completed-job",
					Type:      tempopb.JobType_JOB_TYPE_COMPACTION,
					Status:    tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
					StartTime: now,
					EndTime:   now,
					JobDetail: tempopb.JobDetail{Tenant: "tenant-c"},
				},
				{
					ID:        "failed-job",
					Type:      tempopb.JobType_JOB_TYPE_RETENTION,
					Status:    tempopb.JobStatus_JOB_STATUS_FAILED,
					StartTime: now,
					EndTime:   now,
					JobDetail: tempopb.JobDetail{Tenant: "tenant-d"},
				},
			},
		},
	}

	for _, tt := range tests {
		// Test: Legacy -> Sharded -> Legacy
		t.Run(tt.name+"_legacy_to_sharded_to_legacy", func(t *testing.T) {
			// Create original legacy work
			originalLegacy := New(Config{})
			for _, job := range tt.jobs {
				err := originalLegacy.AddJobPreservingState(job)
				require.NoError(t, err, "failed to add job to original legacy work")
			}

			// Migrate to sharded
			sharded, err := MigrateToSharded(originalLegacy, Config{})
			require.NoError(t, err, "migration to sharded should succeed")

			// Migrate back to legacy
			finalLegacy, err := MigrateFromSharded(sharded)
			require.NoError(t, err, "migration from sharded should succeed")

			// Verify final legacy has same jobs as original
			require.Equal(t, originalLegacy.Len(), finalLegacy.Len(), "final legacy should have same job count as original")

			verifyJobsMatch(t, originalLegacy.ListJobs(), finalLegacy.ListJobs())
		})

		// Test: Sharded -> Legacy -> Sharded
		t.Run(tt.name+"_sharded_to_legacy_to_sharded", func(t *testing.T) {
			// Create original sharded work
			originalSharded := NewSharded(Config{})
			for _, job := range tt.jobs {
				err := originalSharded.AddJobPreservingState(job)
				require.NoError(t, err, "failed to add job to original sharded work")
			}

			// Migrate to legacy
			legacy, err := MigrateFromSharded(originalSharded)
			require.NoError(t, err, "migration from sharded should succeed")

			// Migrate back to sharded
			finalSharded, err := MigrateToSharded(legacy, Config{})
			require.NoError(t, err, "migration to sharded should succeed")

			// Verify final sharded has same jobs as original
			require.Equal(t, originalSharded.Len(), finalSharded.Len(), "final sharded should have same job count as original")

			verifyJobsMatch(t, originalSharded.ListJobs(), finalSharded.ListJobs())
		})
	}
}

// verifyJobsMatch compares two job lists and ensures all fields are preserved
func verifyJobsMatch(t *testing.T, originalJobs, finalJobs []*Job) {
	require.Equal(t, len(originalJobs), len(finalJobs), "job counts should match")

	originalJobsMap := make(map[string]*Job)
	for _, job := range originalJobs {
		originalJobsMap[job.ID] = job
	}

	finalJobsMap := make(map[string]*Job)
	for _, job := range finalJobs {
		finalJobsMap[job.ID] = job
	}

	// Verify all jobs preserved through round trip
	for jobID, originalJob := range originalJobsMap {
		finalJob, exists := finalJobsMap[jobID]
		require.True(t, exists, "job %s should exist after round trip", jobID)
		require.Equal(t, originalJob.ID, finalJob.ID, "job ID should be preserved")
		require.Equal(t, originalJob.Type, finalJob.Type, "job type should be preserved")
		require.Equal(t, originalJob.Status, finalJob.Status, "job status should be preserved")
		require.Equal(t, originalJob.CreatedTime, finalJob.CreatedTime, "job created time should be preserved")
		require.Equal(t, originalJob.StartTime, finalJob.StartTime, "job start time should be preserved")
		require.Equal(t, originalJob.EndTime, finalJob.EndTime, "job end time should be preserved")
		require.Equal(t, originalJob.WorkerID, finalJob.WorkerID, "job worker ID should be preserved")
		require.Equal(t, originalJob.Retries, finalJob.Retries, "job retries should be preserved")
		require.Equal(t, originalJob.JobDetail, finalJob.JobDetail, "job detail should be preserved")
	}
}
