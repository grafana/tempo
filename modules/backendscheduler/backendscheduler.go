package backendscheduler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/blockselector"
	"github.com/jedib0t/go-pretty/v6/table"
	"go.opentelemetry.io/otel"
)

const (
	ringKey = "backend-scheduler"
)

var tracer = otel.Tracer("modules/backendworker")

// BackendScheduler manages scheduling and execution of backend jobs
type BackendScheduler struct {
	services.Service

	cfg       Config
	store     storage.Store
	overrides overrides.Interface

	// track jobs, keyed by job ID
	jobs    map[string]*Job
	jobsMtx sync.RWMutex
}

// Interface that for future work to create and manage jobs.
type JobProcessor interface {
	GetJob(ctx context.Context) (*Job, error)
	StartJob(ctx context.Context, jobID string) error
	CompleteJob(ctx context.Context, jobID string) error
	FailJob(ctx context.Context, jobID string, err error) error
}

type enablePollingFunc func(context.Context, blocklist.JobSharder) error

// New creates a new BackendScheduler
func New(cfg Config, store storage.Store, overrides overrides.Interface) (*BackendScheduler, error) {
	s := &BackendScheduler{
		cfg:       cfg,
		jobs:      make(map[string]*Job),
		store:     store,
		overrides: overrides,
	}

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)
	return s, nil
}

func (s *BackendScheduler) starting(ctx context.Context) error {
	if s.cfg.Poll {
		s.store.EnablePolling(ctx, s)
	}
	return nil
}

func (s *BackendScheduler) running(ctx context.Context) error {
	level.Info(log.Logger).Log("msg", "backend scheduler running")

	ticker := time.NewTicker(s.cfg.ScheduleInterval)
	defer ticker.Stop()

	if len(s.jobs) == 0 {
		if err := s.ScheduleOnce(ctx); err != nil {
			return fmt.Errorf("failed to schedule initial jobs: %w", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:

			if err := s.ScheduleOnce(ctx); err != nil {
				level.Error(log.Logger).Log("msg", "scheduling cycle failed", "err", err)
				schedulingCycles.WithLabelValues("failed").Inc()
			} else {
				schedulingCycles.WithLabelValues("success").Inc()
			}
		}
	}
}

func (s *BackendScheduler) stopping(_ error) error {
	// TODO: consider flushing job state
	return nil
}

// ScheduleOnce schedules jobs for compaction and performs cleanup.
func (s *BackendScheduler) ScheduleOnce(ctx context.Context) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	// Clean up completed/failed jobs
	for id, job := range s.jobs {
		if job.IsComplete() || job.IsFailed() {
			delete(s.jobs, id)
		}
	}

	if len(s.jobs) > 0 {
		return nil
	}

	for _, job := range s.compactions(ctx) {
		if err := s.createCompactionJob(ctx, job.Tenant, job.Compaction.Input); err != nil {
			return fmt.Errorf("failed to create compaction job: %w", err)
		}
	}

	return nil
}

// CreateJob creates a new job
func (s *BackendScheduler) CreateJob(ctx context.Context, job *Job) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	// Validate max concurrent jobs per tenant
	activeCount := 0
	for _, j := range s.jobs {
		if j.JobDetail.Tenant == job.JobDetail.Tenant && !j.IsComplete() && !j.IsFailed() {
			activeCount++
		}
	}

	s.jobs[job.ID] = job
	jobsCreated.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()
	jobsActive.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()

	level.Info(log.Logger).Log("msg", "created new job", "job_id", job.ID, "tenant", job.JobDetail.Tenant, "type", job.Type.String())
	return nil
}

// GetJob returns a job by ID
func (s *BackendScheduler) GetJob(ctx context.Context, id string) (*Job, error) {
	s.jobsMtx.RLock()
	defer s.jobsMtx.RUnlock()

	job, exists := s.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	return job, nil
}

// CompleteJob marks a job as completed
func (s *BackendScheduler) CompleteJob(ctx context.Context, id string) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Complete()
	jobsCompleted.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()
	jobsActive.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "completed job", "job_id", id, "tenant", job.JobDetail.Tenant, "type", job.Type.String())
	return nil
}

// FailJob marks a job as failed
func (s *BackendScheduler) FailJob(ctx context.Context, id string) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Fail()
	jobsFailed.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()
	jobsActive.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "failed job", "job_id", id, "tenant", job.JobDetail.Tenant, "type", job.Type.String())
	return nil
}

// Next implements the BackendSchedulerServer interface
func (s *BackendScheduler) Next(ctx context.Context, req *tempopb.NextJobRequest) (*tempopb.NextJobResponse, error) {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	// Find jobs that already exist for this worker
	for id, job := range s.jobs {
		if job.Status() == JobStatusPending {
			if job.workerID == req.WorkerId {
				resp := &tempopb.NextJobResponse{
					JobId:  id,
					Type:   job.Type,
					Detail: job.JobDetail,
				}

				level.Info(log.Logger).Log("msg", "assigned previous job to worker", "job_id", id, "worker", req.WorkerId)
				return resp, nil
			}
		}
	}

	// Find next available job
	for id, job := range s.jobs {
		// TODO: check if we have pending jobs for this workerID and hand those out first.

		if job.Status() == JobStatusPending {
			// Honor the request job type if specified
			if req.Type != tempopb.JobType_JOB_TYPE_UNSPECIFIED && job.Type != req.Type {
				continue
			}

			// Create response with job details
			resp := &tempopb.NextJobResponse{
				JobId:  id,
				Type:   job.Type,
				Detail: job.JobDetail,
			}

			// Mark job as running
			job.Start(req.WorkerId)

			level.Info(log.Logger).Log("msg", "assigned job to worker", "job_id", id, "worker", req.WorkerId)
			return resp, nil
		}
	}

	// No jobs available
	return &tempopb.NextJobResponse{}, nil
}

// UpdateJob implements the BackendSchedulerServer interface
func (s *BackendScheduler) UpdateJob(ctx context.Context, req *tempopb.UpdateJobStatusRequest) (*tempopb.UpdateJobStatusResponse, error) {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	job, exists := s.jobs[req.JobId]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", req.JobId)
	}

	switch req.Status {
	case tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
		job.Complete()
		jobsCompleted.WithLabelValues(job.JobDetail.Tenant, job.JobDetail.Tenant).Inc()
		level.Info(log.Logger).Log("msg", "job completed", "job_id", req.JobId)

		// TODO: update the blocklist?

	case tempopb.JobStatus_JOB_STATUS_FAILED:
		job.Fail()
		jobsFailed.WithLabelValues(job.JobDetail.Tenant, job.JobDetail.Tenant).Inc()
		level.Error(log.Logger).Log("msg", "job failed", "job_id", req.JobId, "error", req.Error)

	default:
		return nil, fmt.Errorf("invalid job status: %v", req.Status)
	}

	return &tempopb.UpdateJobStatusResponse{
		Success: true,
	}, nil
}

// ListJobs returns all jobs for a given tenant
func (s *BackendScheduler) ListJobs(ctx context.Context, tenantID string) ([]*Job, error) {
	s.jobsMtx.RLock()
	defer s.jobsMtx.RUnlock()

	var jobs []*Job
	for _, job := range s.jobs {
		if job.JobDetail.Tenant == tenantID {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

// CreateCompactionJob creates a new compaction job for the given tenant and blocks.
// Must be called under jobsMtx lock.
func (s *BackendScheduler) createCompactionJob(ctx context.Context, tenantID string, input []string) error {
	// Skip blocks which already have a job
	for _, blockID := range input {
		for _, j := range s.jobs {
			if j.JobDetail.Tenant == tenantID {
				switch j.Type {
				case tempopb.JobType_JOB_TYPE_COMPACTION:
					for _, b := range j.JobDetail.Compaction.Input {
						if b == blockID {
							// TODO: consider continue when we have a stripe of like-blocks.
							return nil
						}
					}
				default:
					continue
				}
			}
		}
	}

	jobID := uuid.New().String()

	job := &Job{
		ID:   jobID,
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: tenantID,
			Compaction: &tempopb.CompactionDetail{
				Input: input,
			},
		},
	}

	// Store the job
	s.jobs[jobID] = job

	// Update metrics
	// TODO: two metrics for the same job smells
	jobsCreated.WithLabelValues(tenantID, job.Type.String()).Inc()
	jobsActive.WithLabelValues(tenantID, job.Type.String()).Inc()

	return nil
}

func (s *BackendScheduler) Owns(_ string) bool {
	return true
}

func (s *BackendScheduler) StatusHandler(w http.ResponseWriter, _ *http.Request) {
	jobs := s.compactions(context.Background())

	x := table.NewWriter()
	x.AppendHeader(table.Row{"tenant", "blocks"})

	for _, e := range jobs {
		x.AppendRows([]table.Row{
			{e.Tenant, e.Compaction.Input},
		})
	}

	x.AppendSeparator()

	// x.SetOutputMirror(w)
	w.WriteHeader(http.StatusOK)
	// w.Header().Set("Content-Type", "plain/text")
	_, _ = io.WriteString(w, x.Render())
}

func (w *BackendScheduler) compactions(ctx context.Context) []tempopb.JobDetail {
	var jobs []tempopb.JobDetail

	tenants := w.store.Tenants()
	if len(tenants) == 0 {
		return jobs
	}

	sort.Slice(tenants, func(i, j int) bool { return tenants[i] < tenants[j] })

	for _, tenantID := range tenants {
		if w.overrides.CompactionDisabled(tenantID) {
			continue
		}

		blocklist := w.store.BlockMetas(tenantID)
		window := w.overrides.MaxCompactionRange(tenantID)
		if window == 0 {
			window = w.cfg.Compactor.MaxCompactionRange
		}

		spew.Dump("window minutes", window.Minutes())

		// TODO: A new implementation of the blockSelector would be appropriate
		// here.  Return a stripe of blocks matching eachother.  These can be
		// broken into jobs by the caller.  Currently only 4 blocks are returned
		// which leads to a single job per tenant.

		spew.Dump("max block bytes", w.cfg.Compactor.MaxBlockBytes)
		spew.Dump("max compction objects", w.cfg.Compactor.MaxCompactionObjects)

		blockSelector := blockselector.NewTimeWindowBlockSelector(blocklist,
			window,
			w.cfg.Compactor.MaxCompactionObjects,
			w.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)

		// TODO:
		// measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)

		for {
			if ctx.Err() != nil {
				spew.Dump("context error", ctx.Err())
				return jobs
			}

			toBeCompacted, _ := blockSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
				spew.Dump("no blocks to compact", tenantID)
				break
			}

			input := make([]string, 0, len(toBeCompacted))
			for _, b := range toBeCompacted {
				input = append(input, b.BlockID.String())
			}

			compaction := &tempopb.CompactionDetail{
				Input: input,
			}

			job := tempopb.JobDetail{
				Tenant: tenantID,
				// Detail: &tempopb.JobDetail_Compaction{Compaction: compaction},
				Compaction: compaction,
			}

			jobs = append(jobs, job)
		}

		// TODO:
		// measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)
	}

	return jobs
}
