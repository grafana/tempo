package work

import (
	"context"
)

// Interface defines the common interface for work management
type Interface interface {
	// Job management
	AddJob(ctx context.Context, j *Job, workerID string) error
	StartJob(ctx context.Context, id string) error
	RemoveJob(ctx context.Context, id string) error
	CompleteJob(ctx context.Context, id string) error
	FailJob(ctx context.Context, id string) error
	SetJobCompactionOutput(ctx context.Context, id string, output []string) error

	// Job queries
	GetJobForWorker(ctx context.Context, workerID string) *Job
	GetJob(id string) *Job
	ListJobs() []*Job

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
