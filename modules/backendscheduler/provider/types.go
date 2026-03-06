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
	HasActiveBatchForTenant(tenantID string) bool
	PopNextPendingJob(jobType tempopb.JobType) *work.Job

	// RegisterJob makes a job visible to other components before it is
	// promoted to the active map via AddJob.
	RegisterJob(job *work.Job)

	// HasJobsForTenant returns true if there are any jobs of the given type in
	// any state (pending, in-flight, or active) for the tenant.
	HasJobsForTenant(tenantID string, jobType tempopb.JobType) bool

	// IsBlockBusy returns true if the block is referenced by any job in any
	// state. Used to skip blocks in selectors and rescans.
	IsBlockBusy(tenantID, blockID string) bool

	// BlocksUnderCompaction returns a blockID -> jobID map for all blocks being
	// compacted for the tenant across active and in-flight states.
	BlocksUnderCompaction(tenantID string) map[string]string
}
