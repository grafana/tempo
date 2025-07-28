package work

import (
	"context"
)

// Interface defines the common interface for work management
type Interface interface {
	// Job management
	AddJob(ctx context.Context, j *Job, workerID string) error
	StartJob(ctx context.Context, id string) error
	GetJob(id string) *Job
	RemoveJob(ctx context.Context, id string)
	CompleteJob(ctx context.Context, id string)
	FailJob(ctx context.Context, id string)
	SetJobCompactionOutput(ctx context.Context, id string, output []string)

	// Job queries
	ListJobs() []*Job
	GetJobForWorker(ctx context.Context, workerID string) *Job

	// Maintenance
	Prune(ctx context.Context)

	// Serialization
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error

	// Local file operations
	FlushToLocal(ctx context.Context, affectedJobIDs []string) error
	LoadFromLocal(ctx context.Context) error

	HasLocalData() bool
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
