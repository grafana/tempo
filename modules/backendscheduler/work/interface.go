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
	// full scope is not yet reflected in the job queue — e.g. a batch was just
	// created or the system is in a rescan delay window. Distinct from
	// CompactionDisabled; this reflects transient internal state.
	TenantPending(tenantID string) bool

	// Batch management -- shared trace ID list for redaction jobs to avoid per-job copies.
	AddBatch(batch *tempopb.RedactionBatch) error
	GetBatch(tenantID string) *tempopb.RedactionBatch
	RemoveBatch(tenantID string)
	ListBatches() []*tempopb.RedactionBatch
	SetBatchRescan(tenantID string, skippedJobIDs []string, rescanAfterUnixNano int64)
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
