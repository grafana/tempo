package provider

import (
	"context"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
)

// Provider defines the interface for job providers
type Provider interface {
	// Start begins the provider's job generation and returns a channel of jobs
	Start(ctx context.Context) <-chan *work.Job
}

// Scheduler interface defines the methods providers need from the scheduler
type Scheduler interface {
	ListJobs() []*work.Job

	// Pending job queries.  Compaction uses BlockPending to skip blocks that are
	// pending redaction; retention uses HasPendingJobs to skip tenants.
	ListPendingJobs(tenantID string, jobType tempopb.JobType) []*work.Job
	HasPendingJobs(tenantID string, jobType tempopb.JobType) bool
	BlockPending(tenantID, blockID string) bool
	PopNextPendingJob(jobType tempopb.JobType) *work.Job
}
