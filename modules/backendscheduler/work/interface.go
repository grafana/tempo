package work

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
)

// Interface defines the common interface for work management
type Interface interface {
	// Job management
	AddJob(j *Job) error
	StartJob(id string)
	GetJob(id string) *Job
	RemoveJob(id string)
	CompleteJob(id string)
	FailJob(id string)
	SetJobCompactionOutput(id string, output []string)

	// Job queries
	ListJobs() []*Job
	GetJobForWorker(ctx context.Context, workerID string) *Job

	// Pending job management (e.g. redaction queue)
	AddPendingJobs(jobs []*Job) error
	ListAllPendingJobs() []*Job
	NextPendingJob(jobType tempopb.JobType) *Job
	// ReleaseRedactionInFlight releases the in-flight count for a dequeued redaction job that is
	// dropped rather than promoted to active (else the counter leaks and wedges the tenant).
	ReleaseRedactionInFlight(tenantID string)

	// PurgePendingRedactionJobs removes one tenant's not-yet-started redaction jobs (cancel),
	// returning the removed job IDs so the caller can report how many were purged.
	PurgePendingRedactionJobs(tenantID string) []string

	// RegisterJob registers a job before it enters the channel pipeline, making it
	// visible to other components. Cleared automatically by AddJob when promoted to active.
	RegisterJob(job *Job)

	// HasJobsForTenant returns true if there are any jobs of the given type in any
	// state (pending queue, in-flight channel, or active map) for the tenant.
	HasJobsForTenant(tenantID string, jobType tempopb.JobType) bool

	// IsBlockBusy returns true if the block is currently referenced by any job in
	// any state (pending, in-flight, or active). Used to skip blocks in selectors
	// and rescans.
	IsBlockBusy(tenantID, blockID string) bool

	// BusyBlocksForTenant returns a map of blockID -> jobID for every block
	// currently referenced by a pending, registered, or active job for the tenant.
	// Acquires pendingMtx exactly once and returns a snapshot.
	BusyBlocksForTenant(tenantID string) map[string]string

	// TenantPending returns true when an exclusive tenant operation exists whose
	// full scope is not yet reflected in the job queue — i.e. an apply-mode redaction batch
	// (just created or in its rescan-wait window). Gates compaction and retention. A dry-run
	// batch is not exclusive and does not make a tenant pending; use GetBatch != nil for a
	// mode-agnostic existence check. Distinct from CompactionDisabled.
	TenantPending(tenantID string) bool

	// Batch management -- shared trace ID list for redaction jobs to avoid per-job copies.
	// GetBatch doubles as the mode-agnostic existence check (one batch per tenant at submission).
	AddBatch(batch *tempopb.RedactionBatch) error
	GetBatch(tenantID string) *tempopb.RedactionBatch
	RemoveBatch(tenantID string)
	ListBatches() []*tempopb.RedactionBatch
	SetBatchRescan(tenantID string, skippedJobIDs []string, rescanAfterUnixNano int64)

	// SetBatchRescanIfCurrent arms the rescan only if the tenant's batch is still the one identified
	// by batchID and is not cancelled, reporting whether it applied. Use it to commit a rescan result
	// computed from an earlier snapshot, so a cancel or a resubmission that landed in the meantime is
	// not overwritten.
	SetBatchRescanIfCurrent(tenantID, batchID string, skippedJobIDs []string, rescanAfterUnixNano int64) bool
	SetBatchQuiesceUntil(tenantID string, untilUnixNano int64)
	// SetBatchCancelled sets the tenant's cancelled flag, publishing it to readers, and reports the
	// value it replaced so a retry can tell whether it made the change. Publish only after
	// PersistBatchCancelled has made the cancel durable.
	SetBatchCancelled(tenantID string, cancelled bool) (previous bool)

	// CommitBatchCancel makes the tenant's cancel durable and then publishes it to readers as one
	// operation: on success it is both on disk and visible, on failure neither, so a failed cancel is
	// never observable and is safe to retry. Reports whether the batch was already cancelled (making a
	// retry idempotent) and returns ErrBatchNotFound if the tenant has no batch.
	CommitBatchCancel(ctx context.Context, tenantID, localPath string) (alreadyCancelled bool, err error)
	BatchQuiescenceState(tenantID string) (quiesceUntilUnixNano int64, rescanPending, dryRun, cancelled, ok bool)
	FlushBatchesToLocal(ctx context.Context, localPath string) error
	LoadBatchesFromLocal(ctx context.Context, localPath string) error

	// Maintenance
	Prune(ctx context.Context)

	// Serialization
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error

	// Local file operations
	FlushToLocal(ctx context.Context, localPath string, affectedJobIDs []string) error
	LoadFromLocal(ctx context.Context, localPath string) error
}

// ShardedWorkInterface extends WorkInterface with sharding-specific methods
type ShardedWorkInterface interface {
	Interface

	// Sharding-specific optimizations
	MarshalShard(shardID int) ([]byte, error)
	UnmarshalShard(shardID int, data []byte) error
	GetShardStats() map[string]any
	GetShardID(jobID string) int
}

var (
	_ Interface            = (*Work)(nil)
	_ ShardedWorkInterface = (*Work)(nil)
)
