package work

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TestReleaseRedactionInFlight verifies a redaction job dequeued via NextPendingJob (now counted
// in-flight) can be released without ever reaching AddJob — the case of a job dropped at
// assignment. Without the release, the counter leaks and HasJobsForTenant stays true forever.
func TestReleaseRedactionInFlight(t *testing.T) {
	w := New(Config{}).(*Work)
	tenant := "t"
	require.NoError(t, w.AddPendingJobs([]*Job{createRedactionJob("j1", tenant, "blk1")}))

	j := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	require.True(t, w.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION), "dequeued job is counted in-flight")

	// The job is dropped at assignment (never promoted via AddJob): release its in-flight count.
	w.ReleaseRedactionInFlight(tenant)
	require.False(t, w.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION), "released job no longer counts; no leak")
}

// TestNextPendingJobCountsInFlightAtomically verifies a dequeued redaction job is counted in-flight
// immediately (dequeue and increment are one critical section), so a concurrent HasJobsForTenant
// can't observe the job as gone-but-not-yet-counted.
func TestNextPendingJobCountsInFlightAtomically(t *testing.T) {
	w := New(Config{}).(*Work)
	tenant := "t"
	require.NoError(t, w.AddPendingJobs([]*Job{createRedactionJob("j1", tenant, "blk1")}))

	require.NotNil(t, w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION))
	require.True(t, w.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION), "job counted the instant it leaves the pending queue")
}

// TestRedactionInFlightAccountingNoLeakUnderRace drains a queue by dequeuing then dropping each job
// (the leak-prone path) while another goroutine hammers HasJobsForTenant. Under -race it guards the
// counter/queue access; functionally it asserts the in-flight count returns to zero — no leak.
func TestRedactionInFlightAccountingNoLeakUnderRace(t *testing.T) {
	w := New(Config{}).(*Work)
	tenant := "t"
	const n = 500
	jobs := make([]*Job, n)
	for i := range jobs {
		jobs[i] = createRedactionJob(fmt.Sprintf("j%d", i), tenant, fmt.Sprintf("blk%d", i))
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				w.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION)
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			j := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
			if j == nil {
				break
			}
			w.ReleaseRedactionInFlight(j.Tenant()) // dropped, not promoted
		}
		close(stop)
	}()
	wg.Wait()

	require.False(t, w.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION), "every dequeued job was released; in-flight count must be back to zero")
}
