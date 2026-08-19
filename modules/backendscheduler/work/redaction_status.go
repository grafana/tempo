package work

import "github.com/grafana/tempo/pkg/tempopb"

// RedactionJobCounts describes how much of a redaction batch is still outstanding.
type RedactionJobCounts struct {
	// Remaining counts block jobs that are queued, in flight between the queue and a worker,
	// or running.
	Remaining int
	// Running is the subset of Remaining currently assigned to a worker.
	Running int
	// Failed counts terminal failures that Prune has not yet removed.
	Failed int
}

// RedactionJobCounts counts tenantID's redaction jobs belonging to batchID.
//
// All three places a job can be are consulted, because a batch's jobs are spread across them:
// shard.Pending (queued), shard.Jobs (running or terminal), and the in-flight counter (dequeued
// by NextPendingJob, not yet promoted by AddJob, so present in neither map). Counting only one
// map reports a batch as further along than it is.
//
// Successful completions are deliberately not counted. Prune removes terminal jobs after
// PruneAge, so a count of survivors shrinks under a long-running batch; progress is derived as
// the batch's recorded jobs_created minus Remaining, which does not decay.
//
// The result is not a point-in-time snapshot: the shard walk and the in-flight read take
// different locks, so a job transitioning between them mid-walk may be counted once or not at
// all. That is adequate for an operator status view, and the alternative is holding every shard
// lock simultaneously. The in-flight counter is per tenant rather than per batch, which is exact
// while the one-batch-per-tenant invariant holds.
//
// Lock order: each shard lock is taken and released before pendingMtx, never nested inside it --
// AddPendingJobs takes pendingMtx then shard.mtx, so nesting the other way would deadlock.
func (w *Work) RedactionJobCounts(tenantID, batchID string) RedactionJobCounts {
	var counts RedactionJobCounts

	belongs := func(j *Job) bool {
		return j.Type == tempopb.JobType_JOB_TYPE_REDACTION &&
			j.JobDetail.Tenant == tenantID &&
			j.JobDetail.BatchId == batchID
	}

	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()

		for _, j := range shard.Pending {
			if belongs(j) {
				counts.Remaining++
			}
		}

		for _, j := range shard.Jobs {
			if !belongs(j) {
				continue
			}
			switch {
			case j.IsFailed():
				counts.Failed++
			case j.IsComplete():
				// Terminal success: neither outstanding nor counted as progress here.
			default:
				// RUNNING, or UNSPECIFIED between AddJob and StartJob.
				counts.Remaining++
				if j.IsRunning() {
					counts.Running++
				}
			}
		}

		shard.mtx.Unlock()
	}

	w.pendingMtx.Lock()
	counts.Remaining += w.redactionInFlight[tenantID]
	w.pendingMtx.Unlock()

	return counts
}

// RedactionStatus is a locked snapshot of a tenant's redaction batch, taken so the status path
// never reads the live pointer GetBatch returns while another goroutine mutates it.
type RedactionStatus struct {
	BatchID              string
	Mode                 tempopb.RedactionMode
	CreatedAtUnixNano    int64
	StartTimeUnixNano    int64
	EndTimeUnixNano      int64
	JobsCreated          int32
	RescanPending        bool
	QuiesceUntilUnixNano int64
}
