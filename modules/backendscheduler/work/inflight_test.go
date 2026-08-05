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

// TestNextPendingJobCountsInFlightOnDequeue verifies a dequeued redaction job is counted in-flight
// as soon as it leaves the pending queue, so HasJobsForTenant keeps reporting the tenant busy before
// the job is promoted via AddJob.
//
// Note: this asserts the count is set on dequeue; it does NOT prove the increment shares the dequeue's
// critical section (the TOCTOU the fix closes). That atomicity is a structural property — increment
// and dequeue are under one pendingMtx hold in NextPendingJob — and is not observable through the
// public API, since HasJobsForTenant serializes on the same mutex and so can never see the interior
// of a single critical section either way. It is verified by inspection, not by this test.
func TestNextPendingJobCountsInFlightOnDequeue(t *testing.T) {
	w := New(Config{}).(*Work)
	tenant := "t"
	require.NoError(t, w.AddPendingJobs([]*Job{createRedactionJob("j1", tenant, "blk1")}))

	require.NotNil(t, w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION))
	require.True(t, w.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION), "job counted the instant it leaves the pending queue")
}

// TestAddJobDuplicateReleasesInFlight covers the leak path where a redaction job is dequeued (and
// thus counted in-flight) but AddJob then finds an identical job ID already active and returns
// ErrJobAlreadyExists before the promote-path decrement. Reachable in normal operation because
// AddPendingJobs dedups only against shard.Pending, so an active ID can be re-enqueued and
// re-dequeued. Without releasing on the duplicate path the count leaks permanently and
// HasJobsForTenant stays true forever, wedging the tenant's future redactions.
func TestAddJobDuplicateReleasesInFlight(t *testing.T) {
	w := New(Config{}).(*Work)
	tenant := "t"

	// An identical job ID is already active.
	require.NoError(t, w.AddJob(createRedactionJob("dup", tenant, "blkA")))

	// Re-enqueue the same ID and dequeue it: NextPendingJob counts it in-flight.
	require.NoError(t, w.AddPendingJobs([]*Job{createRedactionJob("dup", tenant, "blkB")}))
	dup := w.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, dup)

	// AddJob rejects the duplicate; it must still release the in-flight count it did not promote.
	require.ErrorIs(t, w.AddJob(dup), ErrJobAlreadyExists)

	w.pendingMtx.Lock()
	inFlight := w.redactionInFlight[tenant]
	w.pendingMtx.Unlock()
	require.Zero(t, inFlight, "duplicate AddJob must release the unpromoted in-flight count, else the tenant wedges")
}

// TestRedactionInFlightAccountingNoLeakUnderRace drains a queue by dequeuing then dropping each job
// (the leak-prone path) while another goroutine hammers HasJobsForTenant. The functional assertion
// is that every dequeued-then-dropped job releases its count, so the in-flight total returns to
// zero. Running under -race additionally validates that the concurrent NextPendingJob /
// ReleaseRedactionInFlight / HasJobsForTenant callers keep the shared counter and queue maps
// correctly synchronized under pendingMtx.
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
