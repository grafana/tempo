package work

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShardedRoundTrip(t *testing.T) {
	shardedWork := New(Config{})

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
	newShardedWork := New(Config{})
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
