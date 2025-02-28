package backendscheduler

import (
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

// type JobType int
//
// const (
// 	JobTypeUnknown JobType = iota
// 	JobTypeCompaction
// )
//
// func (t JobType) String() string {
// 	switch t {
// 	case JobTypeCompaction:
// 		return "compaction"
// 	default:
// 		return "unknown"
// 	}
// }

type JobStatus int

const (
	JobStatusPending JobStatus = iota
	JobStatusRunning
	JobStatusCompleted
	JobStatusFailed
)

type Job struct {
	ID        string          `json:"id"`
	Type      tempopb.JobType `json:"type"`
	JobDetail tempopb.JobDetail

	// TODO: use proto staus
	status     JobStatus
	statusLock sync.RWMutex
	startTime  time.Time
	endTime    time.Time
	workerID   string
}

func (j *Job) Start(id string) {
	j.statusLock.Lock()
	defer j.statusLock.Unlock()

	j.workerID = id
	j.status = JobStatusRunning
	j.startTime = time.Now()
}

func (j *Job) Complete() {
	j.statusLock.Lock()
	defer j.statusLock.Unlock()

	j.status = JobStatusCompleted
	j.endTime = time.Now()
}

func (j *Job) Fail() {
	j.statusLock.Lock()
	defer j.statusLock.Unlock()

	j.status = JobStatusFailed
	j.endTime = time.Now()
}

func (j *Job) IsComplete() bool {
	j.statusLock.RLock()
	defer j.statusLock.RUnlock()
	return j.status == JobStatusCompleted
}

func (j *Job) IsFailed() bool {
	j.statusLock.RLock()
	defer j.statusLock.RUnlock()
	return j.status == JobStatusFailed
}

func (j *Job) IsRunning() bool {
	j.statusLock.RLock()
	defer j.statusLock.RUnlock()
	return j.status == JobStatusRunning
}

func (j *Job) Status() JobStatus {
	j.statusLock.RLock()
	defer j.statusLock.RUnlock()
	return j.status
}
