package work

import (
	"context"
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
