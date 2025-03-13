package work

import (
	"sort"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

type Work struct {
	Jobs    map[string]*Job `json:"jobs"`
	jobsMtx sync.RWMutex
	cfg     Config
}

func New(cfg Config) *Work {
	return &Work{
		// track jobs, keyed by job ID
		Jobs: make(map[string]*Job),
		cfg:  cfg,
	}
}

func (q *Work) AddJob(j *Job) error {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	if _, ok := q.Jobs[j.ID]; ok {
		return ErrJobAlreadyExists
	}

	j.CreatedTime = time.Now()
	j.Status = tempopb.JobStatus_JOB_STATUS_UNSPECIFIED

	q.Jobs[j.ID] = j

	return nil
}

func (q *Work) GetJob(id string) *Job {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	if v, ok := q.Jobs[id]; ok {
		return v
	}

	return nil
}

func (q *Work) RemoveJob(id string) {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	delete(q.Jobs, id)
}

func (q *Work) ListJobs() []*Job {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	jobs := make([]*Job, 0, len(q.Jobs))
	for _, j := range q.Jobs {
		jobs = append(jobs, j)
	}

	// sort jobs by creation time
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].GetCreatedTime().Before(jobs[j].GetCreatedTime())
	})

	return jobs
}

func (q *Work) Prune() {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	for id, j := range q.Jobs {
		switch j.GetStatus() {
		case tempopb.JobStatus_JOB_STATUS_FAILED, tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
			// Keep the completed jobs around a while so as not to recreate them
			// before the blocklist has been updated.
			if time.Since(j.GetEndTime()) > q.cfg.PruneAge {
				delete(q.Jobs, id)
			}
		}
	}
}

// Len returns the jobs which are not complete or failed.
func (q *Work) Len() int {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	var count int
	for _, j := range q.Jobs {
		if j.IsComplete() || j.IsFailed() {
			continue
		}
		count++
	}

	return count
}

func (q *Work) GetJobForWorker(workerID string) *Job {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	for _, j := range q.Jobs {
		if j.IsRunning() && j.WorkerID == workerID {
			return j
		}
	}

	return nil
}

func (q *Work) GetJobForType(jobType tempopb.JobType) *Job {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	for _, j := range q.Jobs {
		if j.WorkerID != "" {
			continue
		}

		if j.IsPending() {
			// Honor the request job type if specified
			if jobType != tempopb.JobType_JOB_TYPE_UNSPECIFIED && j.Type != jobType {
				continue
			}

			return j
		}
	}

	return nil
}

// HasBlocks returns true if the worker is currently working on any of the provided block IDs.
func (q *Work) HasBlocks(ids []string) bool {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	for _, j := range q.Jobs {
		for _, id := range ids {
			if j.OnBlock(id) {
				return true
			}
		}
	}

	return false
}
