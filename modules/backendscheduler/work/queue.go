package work

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrJobAlreadyExists = errors.New("job already exists")

type Queue struct {
	jobs    map[string]*Job
	jobsMtx sync.RWMutex
}

func NewQueue() *Queue {
	return &Queue{
		// track jobs, keyed by job ID
		jobs: make(map[string]*Job),
	}
}

func (q *Queue) AddJob(j *Job) error {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	if _, ok := q.jobs[j.ID]; ok {
		return ErrJobAlreadyExists
	}

	j.createdTime = time.Now()

	q.jobs[j.ID] = j

	return nil
}

func (q *Queue) GetJob(id string) *Job {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	if v, ok := q.jobs[id]; ok {
		return v
	}

	return nil
}

func (q *Queue) RemoveJob(id string) {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	delete(q.jobs, id)
}

func (q *Queue) Jobs() []*Job {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	jobs := make([]*Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		jobs = append(jobs, j)
	}

	// sort jobs by creation time
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedTime().Before(jobs[j].CreatedTime())
	})

	return jobs
}

func (q *Queue) Prune() {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	for id, j := range q.jobs {
		if j.Status() == JobStatusCompleted || j.Status() == JobStatusFailed {
			delete(q.jobs, id)
		}
	}
}
