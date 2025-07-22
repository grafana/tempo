package work

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	jsoniter "github.com/json-iterator/go"
)

const (
	// ShardCount defines the number of shards (256 = 2^8, perfect for uint8)
	ShardCount = 256
	// ShardMask for fast bit masking instead of modulo
	ShardMask = ShardCount - 1 // 0xFF
)

// Shard represents a single shard containing a subset of jobs
type Shard struct {
	Jobs map[string]*Job `json:"jobs"`
	mtx  sync.Mutex
}

// ShardedWork is a sharded version of Work for improved performance
type ShardedWork struct {
	Shards [ShardCount]*Shard `json:"shards"`
	cfg    Config
}

// NewSharded creates a new sharded work instance
func NewSharded(cfg Config) *ShardedWork {
	sw := &ShardedWork{
		cfg: cfg,
	}

	// Initialize all shards
	for i := range ShardCount {
		sw.Shards[i] = &Shard{
			Jobs: make(map[string]*Job),
		}
	}

	return sw
}

// getShardID returns the shard ID for a given job ID
func (sw *ShardedWork) getShardID(jobID string) uint8 {
	h := fnv.New32a()
	h.Write([]byte(jobID))
	return uint8(h.Sum32() & ShardMask)
}

// GetShardID returns the shard ID for a given job ID
func (sw *ShardedWork) GetShardID(jobID string) uint8 {
	return sw.getShardID(jobID)
}

// getShard returns the shard for a given job ID
func (sw *ShardedWork) getShard(jobID string) *Shard {
	return sw.Shards[sw.getShardID(jobID)]
}

// AddJob adds a job to the appropriate shard
func (sw *ShardedWork) AddJob(j *Job) error {
	if j == nil {
		return ErrJobNil
	}

	shard := sw.getShard(j.ID)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if _, ok := shard.Jobs[j.ID]; ok {
		return ErrJobAlreadyExists
	}

	j.CreatedTime = time.Now()
	j.Status = tempopb.JobStatus_JOB_STATUS_UNSPECIFIED

	shard.Jobs[j.ID] = j
	return nil
}

// AddJobPreservingState adds a job while preserving its existing state
// This is used for migration to preserve job status, timing, etc.
func (sw *ShardedWork) AddJobPreservingState(j *Job) error {
	if j == nil {
		return ErrJobNil
	}

	shard := sw.getShard(j.ID)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if _, ok := shard.Jobs[j.ID]; ok {
		return ErrJobAlreadyExists
	}

	// Preserve all existing state - don't modify the job
	shard.Jobs[j.ID] = j
	return nil
}

// StartJob starts a job in the appropriate shard
func (sw *ShardedWork) StartJob(id string) {
	shard := sw.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		if j.IsPending() {
			j.Start()
		}
	}
}

// GetJob retrieves a job from the appropriate shard
func (sw *ShardedWork) GetJob(id string) *Job {
	shard := sw.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if v, ok := shard.Jobs[id]; ok {
		return v
	}
	return nil
}

// RemoveJob removes a job from the appropriate shard
func (sw *ShardedWork) RemoveJob(id string) {
	shard := sw.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	delete(shard.Jobs, id)
}

// ListJobs returns all jobs across all shards, sorted by creation time
func (sw *ShardedWork) ListJobs() []*Job {
	var allJobs []*Job

	// Collect jobs from all shards
	for i := range ShardCount {
		shard := sw.Shards[i]
		shard.mtx.Lock()

		for _, j := range shard.Jobs {
			allJobs = append(allJobs, j)
		}

		shard.mtx.Unlock()
	}

	// Sort jobs by creation time
	sort.Slice(allJobs, func(i, j int) bool {
		return allJobs[i].GetCreatedTime().Before(allJobs[j].GetCreatedTime())
	})

	return allJobs
}

// Prune removes old completed/failed jobs from all shards
func (sw *ShardedWork) Prune(ctx context.Context) {
	_, span := tracer.Start(ctx, "ShardedPrune")
	defer span.End()

	// Prune each shard independently for better concurrency
	var wg sync.WaitGroup
	for i := range ShardCount {
		wg.Add(1)
		go func(shardIndex int) {
			defer wg.Done()
			shard := sw.Shards[shardIndex]

			shard.mtx.Lock()
			defer shard.mtx.Unlock()

			for id, j := range shard.Jobs {
				switch j.GetStatus() {
				case tempopb.JobStatus_JOB_STATUS_FAILED, tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
					if time.Since(j.GetEndTime()) > sw.cfg.PruneAge {
						delete(shard.Jobs, id)
					}
				case tempopb.JobStatus_JOB_STATUS_RUNNING:
					if time.Since(j.GetStartTime()) > sw.cfg.DeadJobTimeout {
						j.Fail()
					}
				}
			}
		}(i)
	}
	wg.Wait()
}

// Len returns the total number of pending jobs across all shards
func (sw *ShardedWork) Len() int {
	var count int
	for i := range ShardCount {
		shard := sw.Shards[i]
		shard.mtx.Lock()

		for _, j := range shard.Jobs {
			if !j.IsPending() {
				continue
			}
			count++
		}

		shard.mtx.Unlock()
	}
	return count
}

// GetJobForWorker finds a job for a specific worker across all shards
func (sw *ShardedWork) GetJobForWorker(ctx context.Context, workerID string) *Job {
	_, span := tracer.Start(ctx, "ShardedGetJobForWorker")
	defer span.End()

	// Search across all shards for this worker's jobs
	for i := range ShardCount {
		shard := sw.Shards[i]
		shard.mtx.Lock()

		for _, j := range shard.Jobs {
			if j.GetWorkerID() != workerID {
				continue
			}

			switch j.GetStatus() {
			case tempopb.JobStatus_JOB_STATUS_UNSPECIFIED, tempopb.JobStatus_JOB_STATUS_RUNNING:
				shard.mtx.Unlock()
				return j
			}
		}

		shard.mtx.Unlock()
	}

	return nil
}

// CompleteJob marks a job as completed in the appropriate shard
func (sw *ShardedWork) CompleteJob(id string) {
	shard := sw.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		j.Complete()
	}
}

// FailJob marks a job as failed in the appropriate shard
func (sw *ShardedWork) FailJob(id string) {
	shard := sw.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		j.Fail()
	}
}

// SetJobCompactionOutput sets compaction output for a job in the appropriate shard
func (sw *ShardedWork) SetJobCompactionOutput(id string, output []string) {
	shard := sw.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		j.SetCompactionOutput(output)
	}
}

// Marshal serializes all shards to JSON
// NOTE: This is the current full-marshal to maintain the interface compatibility.
// In practice, we'd implement MarshalShard() for single-shard operations
func (sw *ShardedWork) Marshal() ([]byte, error) {
	// For now, marshal the entire structure for compatibility
	return jsoniter.Marshal(sw)
}

// MarshalShard marshals only a specific shard - this is the optimization!
func (sw *ShardedWork) MarshalShard(shardID uint8) ([]byte, error) {
	if int(shardID) >= ShardCount {
		return nil, fmt.Errorf("invalid shard ID: %d", shardID)
	}

	shard := sw.Shards[shardID]
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	return jsoniter.Marshal(shard)
}

// MarshalAffectedShards marshals only the shards that contain the given job IDs
// This is the key optimization - only marshal what changed!
func (sw *ShardedWork) MarshalAffectedShards(jobIDs []string) (map[uint8][]byte, error) {
	// Determine which shards are affected
	affectedShards := make(map[uint8]bool)
	for _, jobID := range jobIDs {
		shardID := sw.getShardID(jobID)
		affectedShards[shardID] = true
	}

	// Marshal only affected shards
	result := make(map[uint8][]byte)
	for shardID := range affectedShards {
		data, err := sw.MarshalShard(shardID)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal shard %d: %w", shardID, err)
		}
		result[shardID] = data
	}

	return result, nil
}

// Unmarshal deserializes JSON to all shards
func (sw *ShardedWork) Unmarshal(data []byte) error {
	err := jsoniter.Unmarshal(data, sw)
	if err != nil {
		return err
	}

	// Ensure all shards are properly initialized (in case any were nil after unmarshaling)
	for i := range ShardCount {
		if sw.Shards[i] == nil {
			sw.Shards[i] = &Shard{
				Jobs: make(map[string]*Job),
			}
		} else if sw.Shards[i].Jobs == nil {
			sw.Shards[i].Jobs = make(map[string]*Job)
		}
	}

	return nil
}

// UnmarshalShard deserializes JSON to a specific shard
func (sw *ShardedWork) UnmarshalShard(shardID uint8, data []byte) error {
	if int(shardID) >= ShardCount {
		return fmt.Errorf("invalid shard ID: %d", shardID)
	}

	shard := sw.Shards[shardID]
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	return jsoniter.Unmarshal(data, shard)
}

// GetShardStats returns statistics about job distribution across shards
func (sw *ShardedWork) GetShardStats() map[string]any {
	stats := make(map[string]any)
	shardSizes := make([]int, ShardCount)
	totalJobs := 0
	nonEmptyShards := 0

	for i := range ShardCount {
		shard := sw.Shards[i]
		shard.mtx.Lock()
		size := len(shard.Jobs)
		shard.mtx.Unlock()

		shardSizes[i] = size
		totalJobs += size
		if size > 0 {
			nonEmptyShards++
		}
	}

	// Calculate distribution statistics
	if totalJobs > 0 {
		avgJobsPerShard := float64(totalJobs) / float64(ShardCount)
		avgJobsPerActiveShard := float64(totalJobs) / float64(nonEmptyShards)

		stats["total_jobs"] = totalJobs
		stats["total_shards"] = ShardCount
		stats["non_empty_shards"] = nonEmptyShards
		stats["avg_jobs_per_shard"] = avgJobsPerShard
		stats["avg_jobs_per_active_shard"] = avgJobsPerActiveShard
		stats["shard_sizes"] = shardSizes
	}

	return stats
}
