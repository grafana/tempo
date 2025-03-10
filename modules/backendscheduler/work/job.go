package work

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

	// TODO: use proto status?
	status      JobStatus
	mtx         sync.RWMutex
	createdTime time.Time
	startTime   time.Time
	endTime     time.Time
	workerID    string
}

func (j *Job) Start(id string) {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	j.workerID = id
	j.status = JobStatusRunning
	j.startTime = time.Now()
}

func (j *Job) Complete() {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	j.status = JobStatusCompleted
	j.endTime = time.Now()
}

func (j *Job) Fail() {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	j.status = JobStatusFailed
	j.endTime = time.Now()
}

func (j *Job) IsComplete() bool {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.status == JobStatusCompleted
}

func (j *Job) IsFailed() bool {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.status == JobStatusFailed
}

func (j *Job) IsRunning() bool {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.status == JobStatusRunning
}

func (j *Job) Status() JobStatus {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.status
}

func (j *Job) CreatedTime() time.Time {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.createdTime
}

func (j *Job) StartTime() time.Time {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.startTime
}

func (j *Job) EndTime() time.Time {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.endTime
}

func (j *Job) WorkerID() string {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.workerID
}
