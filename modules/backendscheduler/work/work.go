package work

import (
	"sort"
	"sync"
	"time"
)

type Work struct {
	jobs    map[string]*Job
	jobsMtx sync.RWMutex
}

func New() *Work {
	return &Work{
		// track jobs, keyed by job ID
		jobs: make(map[string]*Job),
	}
}

func (q *Work) AddJob(j *Job) error {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	if _, ok := q.jobs[j.ID]; ok {
		return ErrJobAlreadyExists
	}

	j.createdTime = time.Now()

	q.jobs[j.ID] = j

	return nil
}

func (q *Work) GetJob(id string) *Job {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	if v, ok := q.jobs[id]; ok {
		return v
	}

	return nil
}

func (q *Work) RemoveJob(id string) {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	delete(q.jobs, id)
}

func (q *Work) Jobs() []*Job {
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

func (q *Work) Prune() {
	q.jobsMtx.Lock()
	defer q.jobsMtx.Unlock()

	for id, j := range q.jobs {
		switch j.Status() {
		case JobStatusCompleted, JobStatusFailed:
			// Keep the completed jobs around a while so as not to recreate them
			// before the blocklist has been updated.
			if time.Since(j.EndTime()) > time.Hour {
				delete(q.jobs, id)
			}
		}
	}
}

func (q *Work) Len() int {
	q.jobsMtx.RLock()
	defer q.jobsMtx.RUnlock()

	var count int
	for _, j := range q.jobs {
		if j.Status() == JobStatusCompleted || j.Status() == JobStatusFailed {
			continue
		}
		count++
	}

	return len(q.jobs)
}
