package work

import (
	"sort"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	jsoniter "github.com/json-iterator/go"
)

type Work struct {
	Jobs map[string]*Job `json:"jobs"`
	mtx  sync.RWMutex
	cfg  Config
}

func New(cfg Config) *Work {
	return &Work{
		// track jobs, keyed by job ID
		Jobs: make(map[string]*Job),
		cfg:  cfg,
	}
}

func (q *Work) AddJob(j *Job) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if j == nil {
		return ErrJobNil
	}

	if _, ok := q.Jobs[j.ID]; ok {
		return ErrJobAlreadyExists
	}

	j.CreatedTime = time.Now()
	j.Status = tempopb.JobStatus_JOB_STATUS_UNSPECIFIED

	q.Jobs[j.ID] = j

	return nil
}

func (q *Work) GetJob(id string) *Job {
	q.mtx.RLock()
	defer q.mtx.RUnlock()

	if v, ok := q.Jobs[id]; ok {
		return v
	}

	return nil
}

func (q *Work) RemoveJob(id string) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	delete(q.Jobs, id)
}

func (q *Work) ListJobs() []*Job {
	q.mtx.RLock()

	jobs := make([]*Job, 0, len(q.Jobs))
	for _, j := range q.Jobs {
		jobs = append(jobs, j)
	}

	// Not defered to unlock while sorting
	q.mtx.RUnlock()

	// sort jobs by creation time
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].GetCreatedTime().Before(jobs[j].GetCreatedTime())
	})

	return jobs
}

func (q *Work) Prune() {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	for id, j := range q.Jobs {
		switch j.GetStatus() {
		case tempopb.JobStatus_JOB_STATUS_FAILED, tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
			// Keep the completed jobs around a while so as not to recreate them
			// before the blocklist has been updated.
			if time.Since(j.GetEndTime()) > q.cfg.PruneAge {
				delete(q.Jobs, id)
			}
		case tempopb.JobStatus_JOB_STATUS_RUNNING:
			// Fail jobs which have been running for too long
			if time.Since(j.GetStartTime()) > q.cfg.DeadJobTimeout {
				j.Fail()
			}
		}
	}
}

// Len returns the jobs which are pending execution.
func (q *Work) Len() int {
	q.mtx.RLock()
	defer q.mtx.RUnlock()

	var count int
	for _, j := range q.Jobs {
		if !j.IsPending() {
			continue
		}
		count++
	}

	return count
}

func (q *Work) GetJobForWorker(workerID string) *Job {
	q.mtx.RLock()
	defer q.mtx.RUnlock()

	for _, j := range q.Jobs {
		if (j.IsRunning() || j.IsPending()) && j.WorkerID == workerID {
			return j
		}
	}

	return nil
}

func (q *Work) Marshal() ([]byte, error) {
	q.mtx.RLock()
	defer q.mtx.RUnlock()

	return jsoniter.Marshal(q)
}

func (q *Work) Unmarshal(data []byte) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	return jsoniter.Unmarshal(data, q)
}
