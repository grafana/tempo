package work

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

func TestWorkLifecycle(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

	ctx := context.Background()

	err := w.AddJob(&Job{ID: "1"})
	require.NoError(t, err)

	err = w.AddJob(&Job{ID: "1"})
	require.Error(t, err)

	j := w.GetJob("1")
	require.NotNil(t, j)

	jj := w.GetJob("2")
	require.Nil(t, jj)

	err = w.AddJob(&Job{ID: "2"})
	require.NoError(t, err)

	jj = w.GetJob("2")
	require.NotNil(t, jj)

	require.Len(t, w.ListJobs(), 2)
	require.Equal(t, w.Len(), 2)

	jobs := w.ListJobs()
	require.Equal(t, "1", jobs[0].ID)
	require.Equal(t, "2", jobs[1].ID)

	require.Equal(t, j.GetType(), tempopb.JobType_JOB_TYPE_UNSPECIFIED)

	j.Complete()
	time.Sleep(200 * time.Millisecond)
	w.Prune(ctx)
	require.Len(t, w.ListJobs(), 1)

	jj.Fail()
	time.Sleep(200 * time.Millisecond)
	w.Prune(ctx)
	require.Len(t, w.ListJobs(), 0)

	err = w.AddJob(&Job{ID: "3"})
	require.NoError(t, err)

	require.Equal(t, w.Len(), 1)

	j = w.GetJob("3")
	require.NotNil(t, j)
	require.Len(t, w.ListJobs(), 1)
	w.RemoveJob(j.ID)

	require.Len(t, w.ListJobs(), 0)
	require.Equal(t, w.Len(), 0)

	err = w.AddJob(nil)
	require.Error(t, err)
}

func TestTenant(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)
	require.Equal(t, w.Len(), 0)

	j := &Job{ID: "1", JobDetail: tempopb.JobDetail{Tenant: "1"}}
	err := w.AddJob(j)
	require.NoError(t, err)
	require.Equal(t, w.Len(), 1)
	require.Equal(t, j.Tenant(), "1")
}

func TestLen(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)
	require.Equal(t, w.Len(), 0)

	j1 := &Job{ID: "1"}
	err := w.AddJob(j1)
	require.NoError(t, err)
	require.Equal(t, w.Len(), 1)

	j2 := &Job{ID: "2"}
	err = w.AddJob(j2)
	require.NoError(t, err)
	require.Equal(t, w.Len(), 2)

	j2.Complete()
	require.Equal(t, w.Len(), 1)

	j1.Fail()
	require.Equal(t, w.Len(), 0)
}

func TestGetJobForWorker(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

	ctx := context.Background()

	j1 := &Job{ID: "1"}
	err := w.AddJob(j1)
	require.NoError(t, err)
	j1.SetWorkerID("one")
	require.Equal(t, j1.GetWorkerID(), "one")
	j1.Start()

	j2 := &Job{ID: "2"}
	err = w.AddJob(j2)
	require.NoError(t, err)
	j2.SetWorkerID("two")
	j2.Start()

	j1 = w.GetJobForWorker(ctx, "one")
	require.NotNil(t, j1)
	require.Equal(t, "1", j1.ID)
	j1.Complete()

	j1 = w.GetJobForWorker(ctx, "one")
	require.Nil(t, j1)
}

func TestGetJobForType(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

	j := &Job{ID: "x", Type: tempopb.JobType_JOB_TYPE_COMPACTION}
	err := w.AddJob(j)
	require.NoError(t, err)

	j2 := &Job{ID: "y", Type: tempopb.JobType_JOB_TYPE_COMPACTION}
	err = w.AddJob(j2)
	require.NoError(t, err)
	j2.Start()
	j2.SetWorkerID("two")
}

func TestBlocks(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

	j := &Job{ID: "1", Type: tempopb.JobType_JOB_TYPE_COMPACTION, JobDetail: tempopb.JobDetail{Compaction: &tempopb.CompactionDetail{Input: []string{"1"}}}}
	err := w.AddJob(j)
	require.NoError(t, err)

	j = &Job{ID: "2", Type: tempopb.JobType_JOB_TYPE_COMPACTION, JobDetail: tempopb.JobDetail{Compaction: &tempopb.CompactionDetail{Input: []string{"2"}, Output: []string{"3"}}}}
	err = w.AddJob(j)
	require.NoError(t, err)

	// test CompactionInput
	j = &Job{ID: "3", Type: tempopb.JobType_JOB_TYPE_COMPACTION, JobDetail: tempopb.JobDetail{Compaction: &tempopb.CompactionDetail{Input: []string{"4"}}}}
	err = w.AddJob(j)
	require.NoError(t, err)
	require.Equal(t, j.GetCompactionInput(), []string{"4"})

	j.SetCompactionOutput([]string{"5"})
	require.Equal(t, j.GetCompactionOutput(), []string{"5"})
}

func TestDeadJobTimeout(t *testing.T) {
	w := New(Config{PruneAge: time.Hour, DeadJobTimeout: 100 * time.Millisecond})
	require.NotNil(t, w)

	ctx := context.Background()

	j := &Job{ID: "1", JobDetail: tempopb.JobDetail{Compaction: &tempopb.CompactionDetail{Input: []string{"1"}}}}
	err := w.AddJob(j)
	require.NoError(t, err)
	j.Start()

	time.Sleep(200 * time.Millisecond)
	w.Prune(ctx)
	require.Equal(t, w.Len(), 0)
	require.Equal(t, j.GetStatus(), tempopb.JobStatus_JOB_STATUS_FAILED)
}

func TestMarshal(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

	j := &Job{ID: "1", JobDetail: tempopb.JobDetail{Compaction: &tempopb.CompactionDetail{Input: []string{"1"}}}}
	err := w.AddJob(j)
	require.NoError(t, err)

	b, err := w.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	w2 := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w2)

	err = w2.Unmarshal(b)
	require.NoError(t, err)

	require.Equal(t, w.Len(), w2.Len())
	require.Equal(t, w.GetJob("1").ID, w2.GetJob("1").ID)
}

func TestJsonMarshal(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

	j := &Job{ID: "1", JobDetail: tempopb.JobDetail{Compaction: &tempopb.CompactionDetail{Input: []string{"1"}}}}
	err := w.AddJob(j)
	require.NoError(t, err)

	b, err := jsoniter.Marshal(w)
	require.NoError(t, err)
	require.NotEmpty(t, b)

	var w2 Work
	err = jsoniter.Unmarshal(b, &w2)
	require.NoError(t, err)
	require.Equal(t, w.Len(), w2.Len())
	require.Equal(t, w.GetJob("1").ID, w2.GetJob("1").ID)
	require.Equal(t, w.GetJob("1").GetCompactionInput(), w2.GetJob("1").GetCompactionInput())
}

// TestConcurrentFlushToLocal verifies that concurrent FlushToLocal calls don't corrupt files
func TestConcurrentFlushToLocal(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create work instances with different data
	works := make([]*Work, 5)
	for i := range works {
		works[i] = New(Config{})
		// Add a unique job to each work instance
		job := &Job{
			ID:   fmt.Sprintf("job-%d", i),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		}
		err := works[i].AddJob(job)
		require.NoError(t, err)
	}

	// Launch concurrent FlushToLocal operations
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workIndex int) {
			defer wg.Done()
			work := works[workIndex%len(works)]

			// Multiple flushes from each goroutine
			for j := 0; j < 3; j++ {
				err := work.FlushToLocal(ctx, tmpDir, nil)
				require.NoError(t, err)
				time.Sleep(1 * time.Millisecond) // Small delay to increase chance of race
			}
		}(i)
	}

	wg.Wait()

	// Verify the final file is valid and not corrupted
	finalWork := New(Config{})
	err := finalWork.LoadFromLocal(ctx, tmpDir)
	require.NoError(t, err)

	// The file should contain valid data from one of the work instances
	require.Equal(t, 1, finalWork.Len(), "Final work should contain exactly one job")

	// Verify the job is from one of our original work instances
	jobs := finalWork.ListJobs()
	require.Len(t, jobs, 1)

	jobID := jobs[0].ID
	found := false
	for i := range works {
		if jobID == fmt.Sprintf("job-%d", i) {
			found = true
			break
		}
	}
	require.True(t, found, "Final job should be from one of the original work instances")

	// Ensure no temporary files are left behind
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.tmp"))
	require.NoError(t, err)
	require.Empty(t, files, "No temporary files should remain")
}

// TestAtomicWriteFileConcurrency verifies that atomicWriteFile itself handles concurrent access safely
func TestAtomicWriteFileConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "test.json")

	// Different data to write from each goroutine
	testData := [][]byte{
		[]byte(`{"test": "data1"}`),
		[]byte(`{"test": "data2"}`),
		[]byte(`{"test": "data3"}`),
		[]byte(`{"test": "data4"}`),
		[]byte(`{"test": "data5"}`),
	}

	// Launch concurrent atomicWriteFile operations
	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(dataIndex int) {
			defer wg.Done()
			data := testData[dataIndex%len(testData)]

			// Multiple writes from each goroutine to the same target file
			for j := 0; j < 3; j++ {
				err := atomicWriteFile(data, targetFile, "test.json")
				require.NoError(t, err)
				time.Sleep(1 * time.Millisecond) // Small delay to increase chance of race
			}
		}(i)
	}

	wg.Wait()

	// Verify the final file exists and contains valid JSON
	finalData, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	require.NotEmpty(t, finalData, "Final file should not be empty")

	// The file should contain valid JSON from one of our test data sets
	var result map[string]interface{}
	err = jsoniter.Unmarshal(finalData, &result)
	require.NoError(t, err, "Final file should contain valid JSON")

	// Verify the content is from one of our original data sets
	testValue, exists := result["test"]
	require.True(t, exists, "JSON should have 'test' field")

	found := false
	for _, data := range testData {
		var expected map[string]interface{}
		err = jsoniter.Unmarshal(data, &expected)
		require.NoError(t, err)
		if expected["test"] == testValue {
			found = true
			break
		}
	}
	require.True(t, found, "Final content should match one of the original data sets")

	// Ensure no temporary files are left behind
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.tmp*"))
	require.NoError(t, err)
	require.Empty(t, files, "No temporary files should remain after atomicWriteFile")
}
