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
	NextPendingJob(jobType tempopb.JobType) *work.Job

	// RegisterJob makes a job visible to other components before it is
	// promoted to the active map via AddJob.
	RegisterJob(job *work.Job)

	// HasJobsForTenant returns true if there are any jobs of the given type in
	// any state (pending, in-flight, or active) for the tenant.
	HasJobsForTenant(tenantID string, jobType tempopb.JobType) bool

	// BusyBlocksForTenant returns a map of blockID -> jobID for every block
	// currently referenced by a pending, registered, or active job for the tenant.
	// Acquires the internal lock exactly once and returns a snapshot.
	BusyBlocksForTenant(tenantID string) map[string]string

	// TenantPending returns true when an exclusive tenant operation exists whose
	// full scope is not yet reflected in the job queue — e.g. a batch manifest
	// was just created before jobs were enqueued, or the system is in a rescan
	// delay window between generations. Distinct from CompactionDisabled (an
	// operator override); this reflects transient internal Work state.
	TenantPending(tenantID string) bool
}
