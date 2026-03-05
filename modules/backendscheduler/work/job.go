package work

import (
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

type Job struct {
	ID          string            `json:"id"`
	Type        tempopb.JobType   `json:"type"`
	JobDetail   tempopb.JobDetail `json:"job_detail"`
	Status      tempopb.JobStatus `json:"status"`
	mtx         sync.Mutex
	CreatedTime time.Time `json:"created_time"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	WorkerID    string    `json:"worker_id"`
	Retries     int       `json:"retries"`
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
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_SUCCEEDED
}

func (j *Job) IsFailed() bool {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_FAILED
}

func (j *Job) IsPending() bool {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_UNSPECIFIED
}

func (j *Job) IsRunning() bool {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.Status == tempopb.JobStatus_JOB_STATUS_RUNNING
}

func (j *Job) GetID() string {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.ID
}

func (j *Job) GetStatus() tempopb.JobStatus {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.Status
}

func (j *Job) GetCreatedTime() time.Time {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.CreatedTime
}

func (j *Job) GetStartTime() time.Time {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.StartTime
}

func (j *Job) GetEndTime() time.Time {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.EndTime
}

func (j *Job) GetType() tempopb.JobType {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.Type
}

func (j *Job) SetWorkerID(id string) {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	j.WorkerID = id
}

func (j *Job) GetWorkerID() string {
	j.mtx.Lock()
	defer j.mtx.Unlock()
	return j.WorkerID
}

func (j *Job) Tenant() string {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	return j.JobDetail.Tenant
}

func (j *Job) GetCompactionInput() []string {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	switch j.Type {
	case tempopb.JobType_JOB_TYPE_COMPACTION:
		return j.JobDetail.Compaction.Input
	default:
		return nil
	}
}

func (j *Job) GetCompactionOutput() []string {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	switch j.Type {
	case tempopb.JobType_JOB_TYPE_COMPACTION:
		return j.JobDetail.Compaction.Output
	default:
		return nil
	}
}

func (j *Job) SetCompactionOutput(blocks []string) {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	switch j.Type {
	case tempopb.JobType_JOB_TYPE_COMPACTION:
		j.JobDetail.Compaction.Output = blocks
	default:
		return
	}
}

// GetRedactionBlockID returns the block ID for redaction jobs, or empty string otherwise.
func (j *Job) GetRedactionBlockID() string {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	if j.Type != tempopb.JobType_JOB_TYPE_REDACTION || j.JobDetail.Redaction == nil {
		return ""
	}
	return j.JobDetail.Redaction.BlockId
}

// PendingBlockKey returns the blocks-pending index key for this job, or empty
// string if this job type does not claim a block. The key is used by Work to
// maintain the pendingBlocks index for fast IsBlockBusy lookups (O(1) pending
// check). Adding a new job type that needs block-pending tracking only requires
// a new branch here; work.go stays type-agnostic.
func (j *Job) PendingBlockKey() string {
	j.mtx.Lock()
	defer j.mtx.Unlock()

	switch j.Type {
	case tempopb.JobType_JOB_TYPE_REDACTION:
		if j.JobDetail.Redaction == nil {
			return ""
		}
		return j.JobDetail.Tenant + "\x00" + j.JobDetail.Redaction.BlockId
	default:
		return ""
	}
}
