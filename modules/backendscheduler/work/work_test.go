package work

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestWork(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

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

	j.Complete()
	time.Sleep(200 * time.Millisecond)
	w.Prune()
	require.Len(t, w.ListJobs(), 1)

	jj.Fail()
	time.Sleep(200 * time.Millisecond)
	w.Prune()
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

	j1 := &Job{ID: "1"}
	err := w.AddJob(j1)
	require.NoError(t, err)
	j1.SetWorkerID("one")
	j1.Start()

	j2 := &Job{ID: "2"}
	err = w.AddJob(j2)
	require.NoError(t, err)
	j2.SetWorkerID("two")
	j2.Start()

	j1 = w.GetJobForWorker("one")
	require.NotNil(t, j1)
	require.Equal(t, "1", j1.ID)
	j1.Complete()

	j1 = w.GetJobForWorker("one")
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

	j = w.GetJobForType(j.Type)
	require.NotNil(t, j)
	require.Equal(t, "x", j.ID)
	j.Complete()

	j = w.GetJobForType(j.Type)
	require.Nil(t, j)
}

func TestHasBlocks(t *testing.T) {
	w := New(Config{PruneAge: 100 * time.Millisecond})
	require.NotNil(t, w)

	j := &Job{ID: "1", JobDetail: tempopb.JobDetail{Compaction: &tempopb.CompactionDetail{Input: []string{"1"}}}}
	w.AddJob(j)
	require.True(t, w.HasBlocks([]string{"1"}))
	require.False(t, w.HasBlocks([]string{"2"}))
}
