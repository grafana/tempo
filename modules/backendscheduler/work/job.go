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

type Job struct {
	ID          string            `json:"id"`
	Type        tempopb.JobType   `json:"type"`
	JobDetail   tempopb.JobDetail `json:"job_detail"`
	Status      tempopb.JobStatus `json:"status"`
	mtx         sync.RWMutex
	CreatedTime time.Time `json:"created_time"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	WorkerID    string    `json:"worker_id"`

	// TODO: implement retries with max
	Retries int `json:"retries"`
}

func (j *Job) Start() {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	j.Status = tempopb.JobStatus_JOB_STATUS_RUNNING
	j.StartTime = time.Now()
}

func (j *Job) Complete() {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	j.Status = tempopb.JobStatus_JOB_STATUS_SUCCEEDED
	j.EndTime = time.Now()
}

func (j *Job) Fail() {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	j.Status = tempopb.JobStatus_JOB_STATUS_FAILED
	j.EndTime = time.Now()
}

func (j *Job) IsComplete() bool {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_SUCCEEDED
}

func (j *Job) IsFailed() bool {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_FAILED
}

func (j *Job) IsPending() bool {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_UNSPECIFIED
}

func (j *Job) IsRunning() bool {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_RUNNING
}

func (j *Job) GetStatus() tempopb.JobStatus {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.Status
}

func (j *Job) GetCreatedTime() time.Time {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.CreatedTime
}

func (j *Job) GetStartTime() time.Time {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.StartTime
}

func (j *Job) GetEndTime() time.Time {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.EndTime
}

func (j *Job) SetWorkerID(id string) {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	j.WorkerID = id
}

func (j *Job) GetWorkerID() string {
	j.mtx.RLock()
	defer j.mtx.RUnlock()
	return j.WorkerID
}

// OnBlock returns true if the job is operating on a block.
func (j *Job) OnBlock(id string) bool {
	for _, b := range j.JobDetail.Compaction.Input {
		if b == id {
			return true
		}
	}

	for _, b := range j.JobDetail.Compaction.Output {
		if b == id {
			return true
		}
	}

	return false
}
