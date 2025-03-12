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
