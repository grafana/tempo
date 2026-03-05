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
	PopNextPendingJob(jobType tempopb.JobType) *Job

	// RegisterInFlight registers a job as in-flight before it enters the channel
	// pipeline. Cleared automatically by AddJob when promoted to active.
	RegisterInFlight(job *Job)

	// HasJobsForTenant returns true if there are any jobs of the given type in any
	// state (pending queue, in-flight channel, or active map) for the tenant.
	HasJobsForTenant(tenantID string, jobType tempopb.JobType) bool

	// IsBlockBusy returns true if the block is currently referenced by any job in
	// any state (pending, in-flight, or active). Used to skip blocks in selectors
	// and rescans.
	IsBlockBusy(tenantID, blockID string) bool

	// BlocksUnderCompaction returns a map of blockID -> jobID for all blocks
	// currently being compacted for the tenant, across active and in-flight states.
	// Used by SubmitRedaction to build the skip+rescan set.
	BlocksUnderCompaction(tenantID string) map[string]string

	// Batch management -- shared trace ID list for redaction jobs to avoid per-job copies.
	AddBatch(batch *tempopb.RedactionBatch) error
	GetBatch(tenantID string) *tempopb.RedactionBatch
	RemoveBatch(tenantID string)
	HasActiveBatchForTenant(tenantID string) bool
	ListBatches() []*tempopb.RedactionBatch
	ClearBatchRescan(tenantID string)
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
