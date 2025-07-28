package work

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("modules/backendscheduler/work")

const (
	// ShardCount defines the number of shards (256 = 2^8)
	ShardCount = 256
	// ShardMask for fast bit masking
	ShardMask = ShardCount - 1 // 0xFF
)

// Shard represents a single shard containing a subset of jobs
type Shard struct {
	Jobs map[string]*Job `json:"jobs"`
	mtx  sync.Mutex
}

type Work struct {
	Shards [ShardCount]*Shard `json:"shards"`
	cfg    Config
	mtx    sync.Mutex // Protects the entire Work structure during Marshal/Unmarshal
}

// TODO: consider returning an error if the mkdir fails
func New(cfg Config) (Interface, error) {
	sw := &Work{
		cfg: cfg,
	}

	err := os.MkdirAll(cfg.LocalWorkPath, 0o700)
	if err != nil {
		return nil, err
	}

	// Initialize all shards
	for i := range ShardCount {
		sw.Shards[i] = &Shard{
			Jobs: make(map[string]*Job),
		}
	}

	// TODO: create the LocatWorkDirectory if it doesn't exist

	return sw, nil
}

// AddJob adds a job to the appropriate shard
func (w *Work) AddJob(ctx context.Context, j *Job, workerID string) error {
	if j == nil {
		return ErrJobNil
	}

	if workerID == "" {
		return ErrEmptyWorkerID
	}

	shardID, shard := w.getShard(j.ID)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if _, ok := shard.Jobs[j.ID]; ok {
		return ErrJobAlreadyExists
	}

	j.CreatedTime = time.Now()
	j.Status = tempopb.JobStatus_JOB_STATUS_UNSPECIFIED
	j.WorkerID = workerID

	shard.Jobs[j.ID] = j

	return w.flushShard(ctx, shard, shardID)
}

// FlushToLocal writes the work cache to local storage using sharding optimizations
func (w *Work) FlushToLocal(ctx context.Context, affectedJobIDs []string) error {
	_, span := tracer.Start(ctx, "FlushToLocal")
	defer span.End()

	err := os.MkdirAll(w.cfg.LocalWorkPath, 0o700)
	if err != nil {
		return err
	}

	if len(affectedJobIDs) == 0 {
		// Flush all shards
		return w.flushAllShards(ctx)
	}

	// Flush only affected shards
	return w.flushAffectedShards(ctx, affectedJobIDs)
}

// LoadFromLocal reads the work cache from local storage using sharding approach
func (w *Work) LoadFromLocal(ctx context.Context) error {
	_, span := tracer.Start(ctx, "LoadFromLocal")
	defer span.End()

	w.mtx.Lock()
	defer w.mtx.Unlock()

	// Load from shard files - BackendScheduler already determined this is the right approach
	for i := range ShardCount {
		shardPath := filepath.Join(w.cfg.LocalWorkPath, FileNameForShard(i))

		data, err := os.ReadFile(shardPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Empty shard is OK
			}
			return err
		}

		err = w.UnmarshalShard(i, data)
		if err != nil {
			return err
		}
	}

	return nil
}

// StartJob starts a job in the appropriate shard
func (w *Work) StartJob(ctx context.Context, id string) error {
	shardID, shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		if j.IsPending() {
			j.Start()
		}
	}

	return w.flushShard(ctx, shard, shardID)
}

// GetJob retrieves a job from the appropriate shard
func (w *Work) GetJob(id string) *Job {
	_, shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if v, ok := shard.Jobs[id]; ok {
		return v
	}
	return nil
}

// RemoveJob removes a job from the appropriate shard
func (w *Work) RemoveJob(ctx context.Context, id string) {
	_, shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	delete(shard.Jobs, id)

	// return w.flushShard(ctx, shard, shardID)
}

// ListJobs returns all jobs across all shards, sorted by creation time
func (w *Work) ListJobs() []*Job {
	var jobCount int

	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		jobCount += len(shard.Jobs)
		shard.mtx.Unlock()
	}

	allJobs := make([]*Job, 0, jobCount)

	// Collect jobs from all shards
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()

		for _, j := range shard.Jobs {
			allJobs = append(allJobs, j)
		}

		shard.mtx.Unlock()
	}

	return allJobs
}

// Prune removes old completed/failed jobs from all shards
func (w *Work) Prune(ctx context.Context) {
	_, span := tracer.Start(ctx, "ShardedPrune")
	defer span.End()

	// TODO: consider using a more efficient pruning strategy

	// Prune each shard independently for better concurrency
	var wg sync.WaitGroup
	for i := range ShardCount {
		wg.Add(1)
		go func(shardIndex int) {
			defer wg.Done()
			shard := w.Shards[shardIndex]

			shard.mtx.Lock()
			defer shard.mtx.Unlock()

			for id, j := range shard.Jobs {
				switch j.GetStatus() {
				case tempopb.JobStatus_JOB_STATUS_FAILED, tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
					if time.Since(j.GetEndTime()) > w.cfg.PruneAge {
						delete(shard.Jobs, id)
					}
				case tempopb.JobStatus_JOB_STATUS_RUNNING:
					if time.Since(j.GetStartTime()) > w.cfg.DeadJobTimeout {
						j.Fail()
					}
				}
			}
		}(i)
	}
	wg.Wait()
}

// GetJobForWorker finds a job for a specific worker across all shards
func (w *Work) GetJobForWorker(ctx context.Context, workerID string) *Job {
	_, span := tracer.Start(ctx, "ShardedGetJobForWorker")
	defer span.End()

	var jj *Job

	// Search across all shards for this worker's jobs
	for i := range ShardCount {
		shard := w.Shards[i]

		jj = func() *Job {
			shard.mtx.Lock()
			defer shard.mtx.Unlock()

			for _, j := range shard.Jobs {
				if j.GetWorkerID() != workerID {
					continue
				}

				switch j.GetStatus() {
				case tempopb.JobStatus_JOB_STATUS_UNSPECIFIED, tempopb.JobStatus_JOB_STATUS_RUNNING:
					return j
				}
			}

			return nil
		}()

		if jj != nil {
			return jj
		}
	}

	return nil
}

// CompleteJob marks a job as completed in the appropriate shard
func (w *Work) CompleteJob(ctx context.Context, id string) {
	_, shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		j.Complete()
	}

	// return w.flushShard(ctx, shard, shardID)
}

// FailJob marks a job as failed in the appropriate shard
func (w *Work) FailJob(ctx context.Context, id string) {
	shardID, shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()
	defer w.flushShard(ctx, shard, shardID)

	if j, ok := shard.Jobs[id]; ok {
		j.Fail()
	}
}

// SetJobCompactionOutput sets compaction output for a job in the appropriate shard
func (w *Work) SetJobCompactionOutput(ctx context.Context, id string, output []string) {
	shardID, shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()
	defer w.flushShard(ctx, shard, shardID)

	if j, ok := shard.Jobs[id]; ok {
		j.SetCompactionOutput(output)
	}
}

// Marshal serializes all shards to JSON with proper locking
func (w *Work) Marshal() ([]byte, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	// Lock all shards in order to prevent race conditions during marshaling
	for i := range ShardCount {
		w.Shards[i].mtx.Lock()
	}
	defer func() {
		// Unlock all shards
		for i := range ShardCount {
			w.Shards[i].mtx.Unlock()
		}
	}()

	// For now, marshal the entire structure for compatibility
	return jsoniter.Marshal(w)
}

// MarshalShard marshals only a specific shard
func (w *Work) MarshalShard(shardID int) ([]byte, error) {
	if shardID >= ShardCount {
		return nil, fmt.Errorf("invalid shard ID: %d", shardID)
	}

	shard := w.Shards[shardID]
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	return jsoniter.Marshal(shard)
}

// Unmarshal deserializes JSON to all shards with proper locking
func (w *Work) Unmarshal(data []byte) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	// Lock all shards in order to prevent race conditions during unmarshaling
	for i := range ShardCount {
		w.Shards[i].mtx.Lock()
	}
	defer func() {
		// Unlock all shards
		for i := range ShardCount {
			w.Shards[i].mtx.Unlock()
		}
	}()

	err := jsoniter.Unmarshal(data, w)
	if err != nil {
		return err
	}

	// Ensure all shards are properly initialized (in case any were nil after unmarshaling)
	for i := range ShardCount {
		if w.Shards[i] == nil {
			w.Shards[i] = &Shard{
				Jobs: make(map[string]*Job),
			}
		} else if w.Shards[i].Jobs == nil {
			w.Shards[i].Jobs = make(map[string]*Job)
		}
	}

	return nil
}

// UnmarshalShard deserializes JSON to a specific shard
func (w *Work) UnmarshalShard(shardID int, data []byte) error {
	if shardID >= ShardCount {
		return fmt.Errorf("invalid shard ID: %d", shardID)
	}

	shard := w.Shards[shardID]
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	return jsoniter.Unmarshal(data, shard)
}

// GetShardStats returns statistics about job distribution across shards
func (w *Work) GetShardStats() map[string]any {
	stats := make(map[string]any)
	shardSizes := make([]int, ShardCount)
	totalJobs := 0
	nonEmptyShards := 0

	for i := range ShardCount {
		shard := w.Shards[i]
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

func (w *Work) getShardID(jobID string) int {
	h := fnv.New32a()
	h.Write([]byte(jobID))
	return int(h.Sum32() & ShardMask)
}

// GetShardID returns the shard ID for a given job ID
func (w *Work) GetShardID(jobID string) int {
	return w.getShardID(jobID)
}

// getShard returns the shard for a given job ID
func (w *Work) getShard(jobID string) (int, *Shard) {
	id := w.getShardID(jobID)
	return id, w.Shards[id]
}

// flushAllShards writes all shards to individual files using atomic operations
func (w *Work) flushAllShards(ctx context.Context) error {
	shards := make(map[int]bool, ShardCount)
	for i := range ShardCount {
		shards[i] = true
	}

	return w.flushShards(ctx, shards)
}

// flushAffectedShards writes only the shards that contain the affected jobs using atomic operations
func (w *Work) flushAffectedShards(ctx context.Context, affectedJobIDs []string) error {
	affectedShards := make(map[int]bool, len(affectedJobIDs))
	for _, jobID := range affectedJobIDs {
		shardID := w.GetShardID(jobID)
		affectedShards[shardID] = true
	}

	return w.flushShards(ctx, affectedShards)
}

func (w *Work) flushShards(ctx context.Context, shards map[int]bool) error {
	_, span := tracer.Start(ctx, "flushShards")
	defer span.End()

	var (
		funcErr error
		shard   *Shard
	)

	for shardID := range shards {
		shard = w.Shards[shardID]
		funcErr = func(shard *Shard) error {
			shard.mtx.Lock()
			defer shard.mtx.Unlock()

			return w.flushShard(ctx, shard, shardID)
		}(shard)
		if funcErr != nil {
			return fmt.Errorf("failed to flush shard %d: %w", shardID, funcErr)
		}
	}
	return nil
}

// flushShard must be called under shard lock
func (w *Work) flushShard(ctx context.Context, shard *Shard, id int) error {
	shardData, err := jsoniter.Marshal(shard)
	if err != nil {
		return err
	}

	filename := FileNameForShard(id)
	shardPath := filepath.Join(w.cfg.LocalWorkPath, filename)

	err = atomicWriteFile(shardData, shardPath)
	if err != nil {
		return err
	}

	return nil
}

func FileNameForShard(shardID int) string {
	return fmt.Sprintf("shard_%03d.json", shardID)
}

// HasShardFiles checks if any shard files exist
func (w *Work) HasLocalData() bool {
	// Check if the directory exists and create it if not.
	if w.cfg.LocalWorkPath == "" {
		return false // No local work path configured
	}
	// Ensure the local work path exists
	if _, err := os.Stat(w.cfg.LocalWorkPath); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		err := os.MkdirAll(w.cfg.LocalWorkPath, 0o700)
		if err != nil {
			level.Error(log.Logger).Log("msg", "failed to create local work path", "path", w.cfg.LocalWorkPath, "error", err)
			return false
		}
	}

	for i := range ShardCount {
		if _, err := os.Stat(w.FilePathForShard(i)); err == nil {
			return true // Found at least one shard file
		}
	}
	return false
}

func (w *Work) FilePathForShard(shardID int) string {
	return filepath.Join(w.cfg.LocalWorkPath, FileNameForShard(shardID))
}
