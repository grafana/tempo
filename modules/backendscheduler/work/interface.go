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
	RemovePending(jobID string)
	ListPendingJobs(tenantID string, jobType tempopb.JobType) []*Job
	ListAllPendingJobs() []*Job
	HasPendingJobs(tenantID string, jobType tempopb.JobType) bool
	BlockPending(tenantID, blockID string) bool
	PopNextPendingJob(jobType tempopb.JobType) *Job

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
