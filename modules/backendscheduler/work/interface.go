package work

import (
	"context"
)

// Interface defines the common interface for work management
// Both Work and ShardedWork implement this interface
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
	Len() int
	GetJobForWorker(ctx context.Context, workerID string) *Job

	// Maintenance
	Prune(ctx context.Context)

	// Serialization
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error

	// Migration helper - preserves existing job state including status
	AddJobPreservingState(j *Job) error
}

// ShardedWorkInterface extends WorkInterface with sharding-specific methods
// This allows the backend scheduler to optionally use sharding optimizations
type ShardedWorkInterface interface {
	Interface

	// Sharding-specific optimizations
	MarshalShard(shardID uint8) ([]byte, error)
	MarshalAffectedShards(jobIDs []string) (map[uint8][]byte, error)
	UnmarshalShard(shardID uint8, data []byte) error
	GetShardStats() map[string]any
	GetShardID(jobID string) uint8
}

// Ensure both implementations satisfy the interface at compile time
var (
	_ Interface            = (*Work)(nil)
	_ Interface            = (*ShardedWork)(nil)
	_ ShardedWorkInterface = (*ShardedWork)(nil)
)
