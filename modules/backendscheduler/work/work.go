package work

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
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

// splitPendingBlockKey is the inverse of pendingBlockKey. It splits key into its
// (tenantID, blockID) components. Callers must only call this on a non-empty key
// returned by Job.PendingBlockKey.
func splitPendingBlockKey(key string) (tenantID, blockID string) {
	tenantID, blockID, _ = strings.Cut(key, "\x00")
	return tenantID, blockID
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

	// pendingBlocks indexes (tenantID, blockID) -> jobID for fast pending-block lookup.
	// Not persisted; rebuilt on LoadFromLocal and in Unmarshal from Shard.Pending.
	pendingBlocks map[string]string

	// pendingByTenant indexes tenant -> job type -> ordered queue of job IDs for O(1)
	// HasJobsForTenant checks and O(1) NextPendingJob dequeues.
	// Not persisted; rebuilt on LoadFromLocal and Unmarshal.
	pendingByTenant map[string]map[tempopb.JobType][]string
	pendingMtx      sync.Mutex

	// redactionInFlight counts redaction jobs per tenant that have been popped from
	// the pending queue by NextPendingJob but not yet promoted to the active map
	// via AddJob. Not persisted; reset to 0 on restart (channels are empty after
	// restart so the count is naturally 0). Guarded by pendingMtx.
	redactionInFlight map[string]int

	// registeredJobs tracks jobs registered by providers before they enter the channel
	// pipeline. Cleared in AddJob when the job is promoted to active. Not persisted.
	// Guarded by pendingMtx.
	registeredJobs map[string]*Job

	// workerJobs indexes workerID -> jobID for every job currently assigned to a
	// worker that has not yet completed or failed. Populated in AddJob (the caller
	// must call SetWorkerID before AddJob), cleared by CompleteJob, FailJob, and the
	// dead-job timeout path in Prune. Not persisted; rebuilt on LoadFromLocal and
	// Unmarshal from the active job map. Guarded by pendingMtx.
	workerJobs map[string]string

	// runningBlocks indexes (tenantID, blockID) -> *Job for every block referenced
	// by a registered or active job. Populated by RegisterJob; entries persist until
	// CompleteJob or FailJob. Guarded by pendingMtx. Not persisted; rebuilt by
	// rebuildPendingIndexes after loading the work cache.
	runningBlocks map[string]*Job

	// runningByTenant counts active (RUNNING or UNSPECIFIED) jobs in shard.Jobs per
	// (tenant, jobType). Eliminates the 256-shard scan in HasJobsForTenant.
	// Guarded by pendingMtx. Not persisted; rebuilt by rebuildPendingIndexes.
	runningByTenant map[string]map[tempopb.JobType]int

	// pendingBlocksByTenant is a secondary index over pendingBlocks:
	// tenantID -> blockID -> jobID. Eliminates the O(N) prefix scan in BusyBlocksForTenant.
	// Guarded by pendingMtx. Not persisted; rebuilt by rebuildPendingIndexes.
	pendingBlocksByTenant map[string]map[string]string

	// runningBlocksByTenant is a secondary index over runningBlocks:
	// tenantID -> blockID -> jobID. Eliminates the O(N) prefix scan in BusyBlocksForTenant.
	// Guarded by pendingMtx. Not persisted; rebuilt by rebuildPendingIndexes.
	runningBlocksByTenant map[string]map[string]string

	// batches holds the active redaction batch per tenant (trace ID list shared across jobs).
	batches *batchStore
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
	sw.pendingByTenant = make(map[string]map[tempopb.JobType][]string)
	sw.redactionInFlight = make(map[string]int)
	sw.registeredJobs = make(map[string]*Job)
	sw.workerJobs = make(map[string]string)
	sw.runningBlocks = make(map[string]*Job)
	sw.runningByTenant = make(map[string]map[tempopb.JobType]int)
	sw.pendingBlocksByTenant = make(map[string]map[string]string)
	sw.runningBlocksByTenant = make(map[string]map[string]string)
	sw.batches = newBatchStore()

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

	w.pendingMtx.Lock()
	// Clear registered job now that it is promoted to active.
	delete(w.registeredJobs, j.ID)
	// If this redaction job was previously in-flight (popped from pending but not
	// yet active), decrement the counter now that it has been promoted to active.
	if j.GetType() == tempopb.JobType_JOB_TYPE_REDACTION {
		if w.redactionInFlight[j.Tenant()] > 0 {
			w.redactionInFlight[j.Tenant()]--
		}
	}
	// Index the worker -> job assignment so GetJobForWorker is O(1).
	if wid := j.GetWorkerID(); wid != "" {
		w.workerJobs[wid] = j.ID
	}
	// Track active job count per (tenant, type) to avoid shard scans in HasJobsForTenant.
	w.incRunningByTenant(j.Tenant(), j.GetType())
	w.pendingMtx.Unlock()

	return nil
}

// RegisterJob registers a job before it enters the channel pipeline, making it
// visible to other components (e.g. HasJobsForTenant, BusyBlocksForTenant).
// Call this immediately after creating a job, before sending it to the jobs channel.
// The registration is cleared automatically when AddJob promotes the job to active.
func (w *Work) RegisterJob(job *Job) {
	w.pendingMtx.Lock()
	w.registeredJobs[job.ID] = job
	for _, key := range runningBlockKeys(job) {
		w.runningBlocks[key] = job
	}
	w.addRunningBlocksByTenant(job)
	w.pendingMtx.Unlock()
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

	w.rebuildPendingIndexes()
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

// Prune removes old completed/failed jobs from all shards and transitions
// timed-out running jobs to FAILED. Index cleanup (runningBlocks, workerJobs)
// for timed-out jobs is performed after all shards are processed.
func (w *Work) Prune(ctx context.Context) {
	_, span := tracer.Start(ctx, "ShardedPrune")
	defer span.End()

	// Pre-allocate one slot per shard so goroutines can collect timed-out jobs
	// without a shared mutex — each goroutine writes only to its own slice.
	timedOut := make([][]*Job, ShardCount)

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
						timedOut[shardIndex] = append(timedOut[shardIndex], j)
					}
				}
			}
		}(i)
	}
	wg.Wait()

	// Clean up indexes for all timed-out jobs. CompleteJob/FailJob normally handle
	// this, but Prune calls j.Fail() directly to avoid re-acquiring the shard lock.
	w.pendingMtx.Lock()
	for _, shardJobs := range timedOut {
		for _, j := range shardJobs {
			for _, key := range runningBlockKeys(j) {
				if w.runningBlocks[key] == j {
					delete(w.runningBlocks, key)
				}
			}
			if wid := j.GetWorkerID(); wid != "" {
				if w.workerJobs[wid] == j.ID {
					delete(w.workerJobs, wid)
				}
			}
			// Timed-out jobs were confirmed RUNNING in the shard scan above.
			w.removeRunningBlocksByTenant(j)
			w.decRunningByTenant(j.Tenant(), j.GetType())
		}
	}
	w.pendingMtx.Unlock()
}

// GetJobForWorker returns the active job assigned to the given worker, or nil
// if none exists. Uses the O(1) workerJobs index rather than scanning all shards.
func (w *Work) GetJobForWorker(ctx context.Context, workerID string) *Job {
	_, span := tracer.Start(ctx, "ShardedGetJobForWorker")
	defer span.End()

	w.pendingMtx.Lock()
	jobID, ok := w.workerJobs[workerID]
	w.pendingMtx.Unlock()

	if !ok {
		return nil
	}

	j := w.GetJob(jobID)
	if j == nil {
		return nil
	}
	switch j.GetStatus() {
	case tempopb.JobStatus_JOB_STATUS_UNSPECIFIED, tempopb.JobStatus_JOB_STATUS_RUNNING:
		return j
	}
	return nil
}

// CompleteJob marks a job as completed in the appropriate shard
func (w *Work) CompleteJob(id string) {
	shard := w.getShard(id)
	shard.mtx.Lock()
	var j *Job
	var wasActive bool
	if jj, ok := shard.Jobs[id]; ok {
		s := jj.GetStatus()
		wasActive = s == tempopb.JobStatus_JOB_STATUS_RUNNING || s == tempopb.JobStatus_JOB_STATUS_UNSPECIFIED
		jj.Complete()
		j = jj
	}
	shard.mtx.Unlock()

	if j != nil {
		w.pendingMtx.Lock()
		for _, key := range runningBlockKeys(j) {
			delete(w.runningBlocks, key)
		}
		if wid := j.GetWorkerID(); wid != "" {
			delete(w.workerJobs, wid)
		}
		w.removeRunningBlocksByTenant(j)
		if wasActive {
			w.decRunningByTenant(j.Tenant(), j.GetType())
		}
		w.pendingMtx.Unlock()
	}
}

// FailJob marks a job as failed in the appropriate shard
func (w *Work) FailJob(id string) {
	shard := w.getShard(id)
	shard.mtx.Lock()
	var j *Job
	var wasActive bool
	if jj, ok := shard.Jobs[id]; ok {
		s := jj.GetStatus()
		wasActive = s == tempopb.JobStatus_JOB_STATUS_RUNNING || s == tempopb.JobStatus_JOB_STATUS_UNSPECIFIED
		jj.Fail()
		j = jj
	}
	shard.mtx.Unlock()

	if j != nil {
		w.pendingMtx.Lock()
		for _, key := range runningBlockKeys(j) {
			delete(w.runningBlocks, key)
		}
		if wid := j.GetWorkerID(); wid != "" {
			delete(w.workerJobs, wid)
		}
		w.removeRunningBlocksByTenant(j)
		if wasActive {
			w.decRunningByTenant(j.Tenant(), j.GetType())
		}
		w.pendingMtx.Unlock()
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
	return json.Marshal(w)
}

// MarshalShard marshals only a specific shard
func (w *Work) MarshalShard(shardID int) ([]byte, error) {
	if shardID >= ShardCount {
		return nil, fmt.Errorf("invalid shard ID: %d", shardID)
	}

	shard := w.Shards[shardID]
	shard.mtx.Lock()
	defer shard.mtx.Unlock()

	return json.Marshal(shard)
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

	err := json.Unmarshal(data, w)
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

	// Rebuild indexes; Unmarshal holds all shard locks so we only take pendingMtx here.
	w.pendingMtx.Lock()
	defer w.pendingMtx.Unlock()
	w.pendingBlocks = make(map[string]string)
	w.pendingByTenant = make(map[string]map[tempopb.JobType][]string)
	w.pendingBlocksByTenant = make(map[string]map[string]string)
	for i := range ShardCount {
		for _, j := range w.Shards[i].Pending {
			if key := j.PendingBlockKey(); key != "" {
				w.pendingBlocks[key] = j.ID
				tenant := j.JobDetail.Tenant
				_, blockID := splitPendingBlockKey(key)
				if w.pendingBlocksByTenant[tenant] == nil {
					w.pendingBlocksByTenant[tenant] = make(map[string]string)
				}
				w.pendingBlocksByTenant[tenant][blockID] = j.ID
			}
			tenant := j.JobDetail.Tenant
			if w.pendingByTenant[tenant] == nil {
				w.pendingByTenant[tenant] = make(map[tempopb.JobType][]string)
			}
			w.pendingByTenant[tenant][j.Type] = append(w.pendingByTenant[tenant][j.Type], j.ID)
		}
	}

	w.workerJobs = make(map[string]string)
	w.runningBlocks = make(map[string]*Job)
	w.runningByTenant = make(map[string]map[tempopb.JobType]int)
	w.runningBlocksByTenant = make(map[string]map[string]string)
	for i := range ShardCount {
		for _, j := range w.Shards[i].Jobs {
			switch j.GetStatus() {
			case tempopb.JobStatus_JOB_STATUS_UNSPECIFIED,
				tempopb.JobStatus_JOB_STATUS_RUNNING:
				for _, key := range runningBlockKeys(j) {
					w.runningBlocks[key] = j
				}
				if wid := j.GetWorkerID(); wid != "" {
					w.workerJobs[wid] = j.ID
				}
				w.incRunningByTenant(j.Tenant(), j.GetType())
				w.addRunningBlocksByTenant(j)
			}
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

	return json.Unmarshal(data, shard)
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

			shardData, err := json.Marshal(shard)
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

// rebuildPendingIndexes rebuilds the pendingBlocks, pendingByTenant, and runningBlocks indexes
// from all shards' Pending and Jobs maps. Caller must hold w.mtx (e.g. during LoadFromLocal).
func (w *Work) rebuildPendingIndexes() {
	w.pendingMtx.Lock()
	defer w.pendingMtx.Unlock()

	w.pendingBlocks = make(map[string]string)
	w.pendingByTenant = make(map[string]map[tempopb.JobType][]string)
	w.pendingBlocksByTenant = make(map[string]map[string]string)

	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for _, j := range shard.Pending {
			if key := j.PendingBlockKey(); key != "" {
				w.pendingBlocks[key] = j.ID
				tenant := j.JobDetail.Tenant
				_, blockID := splitPendingBlockKey(key)
				if w.pendingBlocksByTenant[tenant] == nil {
					w.pendingBlocksByTenant[tenant] = make(map[string]string)
				}
				w.pendingBlocksByTenant[tenant][blockID] = j.ID
			}
			tenant := j.JobDetail.Tenant
			if w.pendingByTenant[tenant] == nil {
				w.pendingByTenant[tenant] = make(map[tempopb.JobType][]string)
			}
			w.pendingByTenant[tenant][j.Type] = append(w.pendingByTenant[tenant][j.Type], j.ID)
		}
		shard.mtx.Unlock()
	}

	w.workerJobs = make(map[string]string)
	w.runningBlocks = make(map[string]*Job)
	w.runningByTenant = make(map[string]map[tempopb.JobType]int)
	w.runningBlocksByTenant = make(map[string]map[string]string)
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for _, j := range shard.Jobs {
			switch j.GetStatus() {
			case tempopb.JobStatus_JOB_STATUS_UNSPECIFIED,
				tempopb.JobStatus_JOB_STATUS_RUNNING:
				for _, key := range runningBlockKeys(j) {
					w.runningBlocks[key] = j
				}
				if wid := j.GetWorkerID(); wid != "" {
					w.workerJobs[wid] = j.ID
				}
				w.incRunningByTenant(j.Tenant(), j.GetType())
				w.addRunningBlocksByTenant(j)
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
		if key := j.PendingBlockKey(); key != "" {
			w.pendingBlocks[key] = j.ID
			tenant := j.Tenant()
			_, blockID := splitPendingBlockKey(key)
			if w.pendingBlocksByTenant[tenant] == nil {
				w.pendingBlocksByTenant[tenant] = make(map[string]string)
			}
			w.pendingBlocksByTenant[tenant][blockID] = j.ID
		}
		shard.mtx.Unlock()

		// Maintain per-tenant queue index (protected by pendingMtx, already held).
		tenant := j.JobDetail.Tenant
		if w.pendingByTenant[tenant] == nil {
			w.pendingByTenant[tenant] = make(map[tempopb.JobType][]string)
		}
		w.pendingByTenant[tenant][j.Type] = append(w.pendingByTenant[tenant][j.Type], j.ID)
	}
	return nil
}

// ListAllPendingJobs returns all pending jobs across all shards and tenants.
func (w *Work) ListAllPendingJobs() []*Job {
	var out []*Job
	for i := range ShardCount {
		shard := w.Shards[i]
		shard.mtx.Lock()
		for _, j := range shard.Pending {
			out = append(out, j)
		}
		shard.mtx.Unlock()
	}
	return out
}

// NextPendingJob removes and returns one pending job of the given type,
// selecting from any tenant with pending work. Uses the pendingByTenant index
// for O(tenants) selection and O(1) dequeue, skipping stale entries.
func (w *Work) NextPendingJob(jobType tempopb.JobType) *Job {
	for {
		w.pendingMtx.Lock()
		var tenantID string
		var jobID string
		for tenant, typeMap := range w.pendingByTenant {
			if len(typeMap[jobType]) > 0 {
				tenantID = tenant
				jobID = typeMap[jobType][0]
				newQueue := typeMap[jobType][1:]
				if len(newQueue) == 0 {
					delete(typeMap, jobType)
					if len(typeMap) == 0 {
						delete(w.pendingByTenant, tenant)
					}
				} else {
					typeMap[jobType] = newQueue
				}
				break
			}
		}
		w.pendingMtx.Unlock()

		if tenantID == "" {
			return nil
		}

		// Retrieve and delete from the shard.
		shard := w.getShard(jobID)
		shard.mtx.Lock()
		j := shard.Pending[jobID]
		delete(shard.Pending, jobID)
		shard.mtx.Unlock()

		if j != nil {
			w.removePendingBlockIndex(j)
			// Track that this redaction job is now in-flight: it has been removed
			// from the pending queue but not yet promoted to the active map.
			if j.GetType() == tempopb.JobType_JOB_TYPE_REDACTION {
				w.pendingMtx.Lock()
				w.redactionInFlight[j.Tenant()]++
				w.pendingMtx.Unlock()
			}
			return j
		}
		// j is nil: stale index entry, skip and retry.
	}
}

// hasRedactionInFlight returns true if there are redaction jobs for the tenant
// that have been popped from the pending queue but not yet promoted to active.
// Caller must hold pendingMtx.
func (w *Work) hasRedactionInFlight(tenantID string) bool {
	return w.redactionInFlight[tenantID] > 0
}

// HasJobsForTenant returns true if there are any jobs of the given type in any
// state (pending queue, registered, or active map) for the tenant.
func (w *Work) HasJobsForTenant(tenantID string, jobType tempopb.JobType) bool {
	w.pendingMtx.Lock()
	hasPending := len(w.pendingByTenant[tenantID][jobType]) > 0
	hasRunning := w.runningByTenant[tenantID][jobType] > 0
	hasInFlight := false
	if jobType == tempopb.JobType_JOB_TYPE_REDACTION {
		hasInFlight = w.hasRedactionInFlight(tenantID)
	}
	if !hasInFlight && !hasRunning {
		for _, j := range w.registeredJobs {
			if j.Tenant() == tenantID && j.GetType() == jobType {
				hasInFlight = true
				break
			}
		}
	}
	w.pendingMtx.Unlock()

	return hasPending || hasRunning || hasInFlight
}

// IsBlockBusy returns true if the block is currently referenced by any pending
// or running job. Uses O(1) map lookups against pendingBlocks and runningBlocks.
func (w *Work) IsBlockBusy(tenantID, blockID string) bool {
	key := pendingBlockKey(tenantID, blockID)
	w.pendingMtx.Lock()
	_, inPending := w.pendingBlocks[key]
	_, inRunning := w.runningBlocks[key]
	w.pendingMtx.Unlock()
	return inPending || inRunning
}

// BusyBlocksForTenant returns a map of blockID -> jobID for every block
// currently referenced by a pending, registered, or active job for the tenant.
// Acquires pendingMtx exactly once and returns a snapshot.
func (w *Work) BusyBlocksForTenant(tenantID string) map[string]string {
	result := make(map[string]string)
	w.pendingMtx.Lock()
	for blockID, jobID := range w.pendingBlocksByTenant[tenantID] {
		result[blockID] = jobID
	}
	for blockID, jobID := range w.runningBlocksByTenant[tenantID] {
		result[blockID] = jobID
	}
	w.pendingMtx.Unlock()
	return result
}

// TenantPending returns true when an exclusive tenant operation exists whose
// full scope is not yet reflected in the job queue. Delegates to the batch store.
func (w *Work) TenantPending(tenantID string) bool {
	return w.batches.hasActive(tenantID)
}

// runningBlockKeys returns pendingBlockKey strings for every block referenced by j.
func runningBlockKeys(j *Job) []string {
	tenant := j.Tenant()
	var keys []string
	for _, bid := range j.GetCompactionInput() {
		keys = append(keys, pendingBlockKey(tenant, bid))
	}
	if bid := j.GetRedactionBlockID(); bid != "" {
		keys = append(keys, pendingBlockKey(tenant, bid))
	}
	return keys
}

// incRunningByTenant increments the runningByTenant count for (tenant, jobType).
// Caller must hold pendingMtx.
func (w *Work) incRunningByTenant(tenant string, jobType tempopb.JobType) {
	if w.runningByTenant[tenant] == nil {
		w.runningByTenant[tenant] = make(map[tempopb.JobType]int)
	}
	w.runningByTenant[tenant][jobType]++
}

// decRunningByTenant decrements the runningByTenant count for (tenant, jobType),
// cleaning up empty entries. Caller must hold pendingMtx.
func (w *Work) decRunningByTenant(tenant string, jobType tempopb.JobType) {
	if m := w.runningByTenant[tenant]; m != nil {
		m[jobType]--
		if m[jobType] <= 0 {
			delete(m, jobType)
			if len(m) == 0 {
				delete(w.runningByTenant, tenant)
			}
		}
	}
}

// addRunningBlocksByTenant adds j's blocks to the runningBlocksByTenant secondary index.
// Caller must hold pendingMtx.
func (w *Work) addRunningBlocksByTenant(j *Job) {
	tenant := j.Tenant()
	if w.runningBlocksByTenant[tenant] == nil {
		w.runningBlocksByTenant[tenant] = make(map[string]string)
	}
	for _, bid := range j.GetCompactionInput() {
		w.runningBlocksByTenant[tenant][bid] = j.ID
	}
	if bid := j.GetRedactionBlockID(); bid != "" {
		w.runningBlocksByTenant[tenant][bid] = j.ID
	}
}

// removeRunningBlocksByTenant removes j's blocks from the runningBlocksByTenant secondary index,
// only deleting entries that still point to this job. Caller must hold pendingMtx.
func (w *Work) removeRunningBlocksByTenant(j *Job) {
	tenant := j.Tenant()
	tb := w.runningBlocksByTenant[tenant]
	if tb == nil {
		return
	}
	for _, bid := range j.GetCompactionInput() {
		if tb[bid] == j.ID {
			delete(tb, bid)
		}
	}
	if bid := j.GetRedactionBlockID(); bid != "" {
		if tb[bid] == j.ID {
			delete(tb, bid)
		}
	}
	if len(tb) == 0 {
		delete(w.runningBlocksByTenant, tenant)
	}
}

func (w *Work) removePendingBlockIndex(j *Job) {
	key := j.PendingBlockKey()
	if key == "" {
		return
	}
	tenant := j.Tenant()
	w.pendingMtx.Lock()
	defer w.pendingMtx.Unlock()
	delete(w.pendingBlocks, key)
	_, blockID := splitPendingBlockKey(key)
	if tb := w.pendingBlocksByTenant[tenant]; tb != nil {
		delete(tb, blockID)
		if len(tb) == 0 {
			delete(w.pendingBlocksByTenant, tenant)
		}
	}
}
