package work

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
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

// pendingBlockKey returns a stable key for the blocks-pending index (tenant + blockID).
func pendingBlockKey(tenantID, blockID string) string {
	return tenantID + "\x00" + blockID
}

// Shard represents a single shard containing a subset of jobs
type Shard struct {
	Jobs    map[string]*Job `json:"jobs"`
	Pending map[string]*Job `json:"pending_jobs,omitempty"`
	mtx     sync.Mutex
}

type Work struct {
	Shards [ShardCount]*Shard `json:"shards"`
	cfg    Config
	mtx    sync.Mutex // Protects the entire Work structure during Marshal/Unmarshal

	// pendingBlocks indexes (tenantID, blockID) -> jobID for fast BlockPending lookup
	// Not persisted; rebuilt on LoadFromLocal and in Unmarshal from Shard.Pending.
	pendingBlocks          map[string]string `json:"-"`
	currentRedactionTenant string            `json:"-"`
	pendingMtx             sync.Mutex        `json:"-"`
}

func New(cfg Config) Interface {
	sw := &Work{
		cfg: cfg,
	}

	// Initialize all shards
	for i := range ShardCount {
		sw.Shards[i] = &Shard{
			Jobs:    make(map[string]*Job),
			Pending: make(map[string]*Job),
		}
	}
	sw.pendingBlocks = make(map[string]string)

	return sw
}

// AddJob adds a job to the appropriate shard
func (w *Work) AddJob(j *Job) error {
	if j == nil {
		return ErrJobNil
	}

	shard := w.getShard(j.ID)
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

// FlushToLocal writes the work cache to local storage using sharding optimizations
func (w *Work) FlushToLocal(_ context.Context, localPath string, affectedJobIDs []string) error {
	err := os.MkdirAll(localPath, 0o700)
	if err != nil {
		return err
	}

	if len(affectedJobIDs) == 0 {
		// Flush all shards
		return w.flushAllShards(localPath)
	}

	// Flush only affected shards
	return w.flushAffectedShards(localPath, affectedJobIDs)
}

// LoadFromLocal reads the work cache from local storage using sharding approach
func (w *Work) LoadFromLocal(_ context.Context, localPath string) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	// Load from shard files - BackendScheduler already determined this is the right approach
	for i := range ShardCount {
		shardPath := filepath.Join(localPath, FileNameForShard(uint8(i)))

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

	w.rebuildPendingBlocksIndex()
	return nil
}

// StartJob starts a job in the appropriate shard
func (w *Work) StartJob(id string) {
	shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		if j.IsPending() {
			j.Start()
		}
	}
}

// GetJob retrieves a job from the appropriate shard
func (w *Work) GetJob(id string) *Job {
	shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if v, ok := shard.Jobs[id]; ok {
		return v
	}
	return nil
}

// RemoveJob removes a job from the appropriate shard
func (w *Work) RemoveJob(id string) {
	shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	delete(shard.Jobs, id)
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
func (w *Work) CompleteJob(id string) {
	shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		j.Complete()
	}
}

// FailJob marks a job as failed in the appropriate shard
func (w *Work) FailJob(id string) {
	shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	if j, ok := shard.Jobs[id]; ok {
		j.Fail()
	}
}

// SetJobCompactionOutput sets compaction output for a job in the appropriate shard
func (w *Work) SetJobCompactionOutput(id string, output []string) {
	shard := w.getShard(id)
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

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
				Jobs:    make(map[string]*Job),
				Pending: make(map[string]*Job),
			}
		} else {
			if w.Shards[i].Jobs == nil {
				w.Shards[i].Jobs = make(map[string]*Job)
			}
			if w.Shards[i].Pending == nil {
				w.Shards[i].Pending = make(map[string]*Job)
			}
		}
	}

	// Rebuild index; Unmarshal holds all shard locks so we only take pendingMtx.
	w.pendingMtx.Lock()
	w.pendingBlocks = make(map[string]string)
	for i := range ShardCount {
		for _, j := range w.Shards[i].Pending {
			if j.Type == tempopb.JobType_JOB_TYPE_REDACTION && j.JobDetail.Redaction != nil {
				key := pendingBlockKey(j.JobDetail.Tenant, j.JobDetail.Redaction.BlockId)
				w.pendingBlocks[key] = j.ID
			}
		}
	}
	w.pendingMtx.Unlock()

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
func (w *Work) getShard(jobID string) *Shard {
	return w.Shards[w.getShardID(jobID)]
}

// flushAllShards writes all shards to individual files using atomic operations
func (w *Work) flushAllShards(localPath string) error {
	shards := make(map[int]bool, ShardCount)
	for i := range ShardCount {
		shards[i] = true
	}

	return w.flushShards(localPath, shards)
}

// flushAffectedShards writes only the shards that contain the affected jobs using atomic operations
func (w *Work) flushAffectedShards(localPath string, affectedJobIDs []string) error {
	affectedShards := make(map[int]bool, len(affectedJobIDs))
	for _, jobID := range affectedJobIDs {
		shardID := w.GetShardID(jobID)
		affectedShards[shardID] = true
	}

	return w.flushShards(localPath, affectedShards)
}

func (w *Work) flushShards(localPath string, shards map[int]bool) error {
	var (
		funcErr   error
		filename  string
		shardPath string
		shard     *Shard
	)

	for shardID := range shards {
		shard = w.Shards[shardID]
		funcErr = func(shard *Shard) error {
			shard.mtx.Lock()
			defer shard.mtx.Unlock()

			shardData, err := jsoniter.Marshal(shard)
			if err != nil {
				return err
			}

			filename = FileNameForShard(uint8(shardID))
			shardPath = filepath.Join(localPath, filename)

			err = atomicWriteFile(shardData, shardPath, filename)
			if err != nil {
				return err
			}

			return nil
		}(shard)
		if funcErr != nil {
			return fmt.Errorf("failed to flush shard %d: %w", shardID, funcErr)
		}
	}
	return nil
}

func FileNameForShard(shardID uint8) string {
	return fmt.Sprintf("shard_%03d.json", shardID)
}

// rebuildPendingBlocksIndex rebuilds the pendingBlocks index from all shards' Pending maps.
// Caller must hold w.mtx (e.g. during LoadFromLocal). LoadFromLocal does not hold shard locks here.
func (w *Work) rebuildPendingBlocksIndex() {
	w.pendingMtx.Lock()
	defer w.pendingMtx.Unlock()

	w.pendingBlocks = make(map[string]string)
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for _, j := range shard.Pending {
			if j.Type == tempopb.JobType_JOB_TYPE_REDACTION && j.JobDetail.Redaction != nil {
				key := pendingBlockKey(j.JobDetail.Tenant, j.JobDetail.Redaction.BlockId)
				w.pendingBlocks[key] = j.ID
			}
		}
		shard.mtx.Unlock()
	}
}

// AddPendingJobs adds jobs to the appropriate shards' Pending maps and updates the blocks-pending index.
func (w *Work) AddPendingJobs(jobs []*Job) error {
	if len(jobs) == 0 {
		return nil
	}

	w.pendingMtx.Lock()
	defer w.pendingMtx.Unlock()

	for _, j := range jobs {
		if j == nil {
			continue
		}
		shard := w.getShard(j.ID)
		shard.mtx.Lock()
		if _, ok := shard.Pending[j.ID]; ok {
			shard.mtx.Unlock()
			continue
		}
		j.CreatedTime = time.Now()
		j.Status = tempopb.JobStatus_JOB_STATUS_UNSPECIFIED
		shard.Pending[j.ID] = j
		if j.Type == tempopb.JobType_JOB_TYPE_REDACTION && j.JobDetail.Redaction != nil {
			key := pendingBlockKey(j.JobDetail.Tenant, j.JobDetail.Redaction.BlockId)
			w.pendingBlocks[key] = j.ID
		}
		shard.mtx.Unlock()
	}
	return nil
}

// RemovePending removes a job from the appropriate shard's Pending map and the blocks-pending index.
func (w *Work) RemovePending(jobID string) {
	shard := w.getShard(jobID)
	shard.mtx.Lock()
	j, ok := shard.Pending[jobID]
	shard.mtx.Unlock()
	if !ok {
		return
	}

	w.pendingMtx.Lock()
	defer w.pendingMtx.Unlock()

	shard.mtx.Lock()
	delete(shard.Pending, jobID)
	if j.Type == tempopb.JobType_JOB_TYPE_REDACTION && j.JobDetail.Redaction != nil {
		key := pendingBlockKey(j.JobDetail.Tenant, j.JobDetail.Redaction.BlockId)
		delete(w.pendingBlocks, key)
	}
	shard.mtx.Unlock()
}

// ListPendingJobs returns all pending jobs for the given tenant and job type across all shards.
func (w *Work) ListPendingJobs(tenantID string, jobType tempopb.JobType) []*Job {
	var out []*Job
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for _, j := range shard.Pending {
			if j.JobDetail.Tenant == tenantID && j.Type == jobType {
				out = append(out, j)
			}
		}
		shard.mtx.Unlock()
	}
	return out
}

// HasPendingJobs returns true if there are any pending jobs for the given tenant and job type.
func (w *Work) HasPendingJobs(tenantID string, jobType tempopb.JobType) bool {
	return len(w.ListPendingJobs(tenantID, jobType)) > 0
}

// BlockPending returns true if the given (tenantID, blockID) has a pending job
func (w *Work) BlockPending(tenantID, blockID string) bool {
	w.pendingMtx.Lock()
	_, ok := w.pendingBlocks[pendingBlockKey(tenantID, blockID)]
	w.pendingMtx.Unlock()
	return ok
}

// PopNextPendingJob removes and returns one pending job of the given type. For redaction, it drains
// one tenant at a time (returns jobs for the same tenant until none remain, then switches to another).
func (w *Work) PopNextPendingJob(jobType tempopb.JobType) *Job {
	if jobType != tempopb.JobType_JOB_TYPE_REDACTION {
		return w.popAnyPendingJob(jobType)
	}
	return w.popNextPendingRedactionJob()
}

func (w *Work) popAnyPendingJob(jobType tempopb.JobType) *Job {
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for id, j := range shard.Pending {
			if j.Type == jobType {
				delete(shard.Pending, id)
				shard.mtx.Unlock()
				w.removePendingBlockIndex(j)
				return j
			}
		}
		shard.mtx.Unlock()
	}
	return nil
}

func (w *Work) popNextPendingRedactionJob() *Job {
	w.pendingMtx.Lock()
	currentTenant := w.currentRedactionTenant
	w.pendingMtx.Unlock()

	// Try current tenant first
	if currentTenant != "" {
		if j := w.popOnePendingJobForTenant(currentTenant, tempopb.JobType_JOB_TYPE_REDACTION); j != nil {
			return j
		}
		w.pendingMtx.Lock()
		w.currentRedactionTenant = ""
		w.pendingMtx.Unlock()
	}

	// Find any tenant with pending redaction and set as current
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for _, j := range shard.Pending {
			if j.Type == tempopb.JobType_JOB_TYPE_REDACTION && j.JobDetail.Tenant != "" {
				tenant := j.JobDetail.Tenant
				shard.mtx.Unlock()
				w.pendingMtx.Lock()
				w.currentRedactionTenant = tenant
				w.pendingMtx.Unlock()
				return w.popOnePendingJobForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION)
			}
		}
		shard.mtx.Unlock()
	}
	return nil
}

func (w *Work) popOnePendingJobForTenant(tenantID string, jobType tempopb.JobType) *Job {
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for id, j := range shard.Pending {
			if j.JobDetail.Tenant == tenantID && j.Type == jobType {
				delete(shard.Pending, id)
				shard.mtx.Unlock()
				w.removePendingBlockIndex(j)
				return j
			}
		}
		shard.mtx.Unlock()
	}
	return nil
}

func (w *Work) removePendingBlockIndex(j *Job) {
	if j.Type != tempopb.JobType_JOB_TYPE_REDACTION || j.JobDetail.Redaction == nil {
		return
	}
	w.pendingMtx.Lock()
	defer w.pendingMtx.Unlock()
	key := pendingBlockKey(j.JobDetail.Tenant, j.JobDetail.Redaction.BlockId)
	delete(w.pendingBlocks, key)
}
