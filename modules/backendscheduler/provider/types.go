package provider

import (
	"context"

	"github.com/grafana/tempo/modules/backendscheduler/work"
)

// Provider defines the interface for job providers
type Provider interface {
	// Start begins the provider's job generation and returns a channel of jobs
	Start(ctx context.Context) <-chan *work.Job
}

// Scheduler interface defines the methods providers need from the scheduler
type Scheduler interface {
	ListJobs() []*work.Job
	HasBlocks([]string) bool
}
