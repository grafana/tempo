package backendscheduler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/blockselector"
	"github.com/jedib0t/go-pretty/v6/table"
)

// var tracer = otel.Tracer("modules/backendscheduler")

// BackendScheduler manages scheduling and execution of backend jobs
type BackendScheduler struct {
	services.Service

	cfg       Config
	store     storage.Store
	overrides overrides.Interface

	work *work.Queue
}

var _ JobProcessor = (*BackendScheduler)(nil)

// Interface that for future API work to create and manage jobs.
type JobProcessor interface {
	GetJob(ctx context.Context, jobID string) (*work.Job, error)
	CompleteJob(ctx context.Context, jobID string) error
	FailJob(ctx context.Context, jobID string) error
	CreateJob(ctx context.Context, job *work.Job) error
	ListJobs(ctx context.Context) []*work.Job
}

// New creates a new BackendScheduler
func New(cfg Config, store storage.Store, overrides overrides.Interface) (*BackendScheduler, error) {
	s := &BackendScheduler{
		cfg:       cfg,
		store:     store,
		overrides: overrides,
		work:      work.NewQueue(),
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

	if len(s.work.Jobs()) == 0 {
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
				metricSchedulingCycles.WithLabelValues("failed").Inc()
			} else {
				metricSchedulingCycles.WithLabelValues("success").Inc()
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
	s.work.Prune()

	for _, job := range s.compactions(ctx) {
		if err := s.createCompactionJob(ctx, job.Tenant, job.Compaction.Input); err != nil {
			return fmt.Errorf("failed to create compaction job: %w", err)
		}
	}

	return nil
}

// CreateJob creates a new job
func (s *BackendScheduler) CreateJob(_ context.Context, j *work.Job) error {
	err := s.work.AddJob(j)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	activeCount := 0
	for _, jj := range s.work.Jobs() {
		if !jj.IsComplete() && !jj.IsFailed() {
			activeCount++
		}
	}

	metricJobsCreated.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
	metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()

	level.Info(log.Logger).Log("msg", "created new job", "job_id", j.ID, "tenant", j.JobDetail.Tenant, "type", j.Type.String())
	return nil
}

// GetJob returns a job by ID
func (s *BackendScheduler) GetJob(_ context.Context, id string) (*work.Job, error) {
	j := s.work.GetJob(id)
	if j == nil {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	return j, nil
}

// CompleteJob marks a job as completed
func (s *BackendScheduler) CompleteJob(_ context.Context, id string) error {
	j := s.work.GetJob(id)
	if j == nil {
		return fmt.Errorf("job not found: %s", id)
	}

	j.Complete()
	metricJobsCompleted.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
	metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "completed job", "job_id", id, "tenant", j.JobDetail.Tenant, "type", j.Type.String())
	return nil
}

// FailJob marks a job as failed
func (s *BackendScheduler) FailJob(_ context.Context, id string) error {
	j := s.work.GetJob(id)
	if j == nil {
		return fmt.Errorf("job not found: %s", id)
	}

	j.Fail()
	metricJobsFailed.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
	metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "failed job", "job_id", id, "tenant", j.JobDetail.Tenant, "type", j.Type.String())
	return nil
}

// Next implements the BackendSchedulerServer interface
func (s *BackendScheduler) Next(_ context.Context, req *tempopb.NextJobRequest) (*tempopb.NextJobResponse, error) {
	// Find jobs that already exist for this worker
	for id, j := range s.work.Jobs() {
		if j.Status() == work.JobStatusPending || j.Status() == work.JobStatusRunning {
			if j.WorkerID() == req.WorkerId {
				resp := &tempopb.NextJobResponse{
					JobId:  j.ID,
					Type:   j.Type,
					Detail: j.JobDetail,
				}

				level.Info(log.Logger).Log("msg", "assigned previous job to worker", "job_id", id, "worker", req.WorkerId)
				return resp, nil
			}
		}
	}

	// Find next available job
	for id, j := range s.work.Jobs() {
		if j.Status() == work.JobStatusPending {
			// Honor the request job type if specified
			if req.Type != tempopb.JobType_JOB_TYPE_UNSPECIFIED && j.Type != req.Type {
				continue
			}

			// Create response with job details
			resp := &tempopb.NextJobResponse{
				JobId:  j.ID,
				Type:   j.Type,
				Detail: j.JobDetail,
			}

			// Mark job as running
			j.Start(req.WorkerId)

			level.Info(log.Logger).Log("msg", "assigned job to worker", "job_id", id, "worker", req.WorkerId)
			return resp, nil
		}
	}

	// No jobs available
	return &tempopb.NextJobResponse{}, nil
}

// UpdateJob implements the BackendSchedulerServer interface
func (s *BackendScheduler) UpdateJob(_ context.Context, req *tempopb.UpdateJobStatusRequest) (*tempopb.UpdateJobStatusResponse, error) {
	j := s.work.GetJob(req.JobId)
	if j == nil {
		return nil, fmt.Errorf("job not found: %s", req.JobId)
	}

	switch req.Status {
	case tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
		j.Complete()
		metricJobsCompleted.WithLabelValues(j.JobDetail.Tenant, j.JobDetail.Tenant).Inc()
		level.Info(log.Logger).Log("msg", "job completed", "job_id", req.JobId)

		// TODO: update the blocklist?

	case tempopb.JobStatus_JOB_STATUS_FAILED:
		j.Fail()
		metricJobsFailed.WithLabelValues(j.JobDetail.Tenant, j.JobDetail.Tenant).Inc()
		level.Error(log.Logger).Log("msg", "job failed", "job_id", req.JobId, "error", req.Error)

	default:
		return nil, fmt.Errorf("invalid job status: %v", req.Status)
	}

	return &tempopb.UpdateJobStatusResponse{
		Success: true,
	}, nil
}

// ListJobs returns all jobs for a given tenant
func (s *BackendScheduler) ListJobs(_ context.Context) []*work.Job {
	return s.work.Jobs()
}

// CreateCompactionJob creates a new compaction job for the given tenant and blocks.
// Must be called under jobsMtx lock.
func (s *BackendScheduler) createCompactionJob(_ context.Context, tenantID string, input []string) error {
	// Skip blocks which already have a job
	for _, blockID := range input {
		for _, j := range s.work.Jobs() {
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

	job := &work.Job{
		ID:   jobID,
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: tenantID,
			Compaction: &tempopb.CompactionDetail{
				Input: input,
			},
		},
	}

	err := s.work.AddJob(job)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	// Update metrics
	metricJobsCreated.WithLabelValues(tenantID, job.Type.String()).Inc()
	metricJobsActive.WithLabelValues(tenantID, job.Type.String()).Inc()

	return nil
}

func (s *BackendScheduler) Owns(_ string) bool {
	return true
}

func (s *BackendScheduler) StatusHandler(w http.ResponseWriter, _ *http.Request) {
	x := table.NewWriter()
	x.AppendHeader(table.Row{"tenant", "blocks", "status", "worker", "created", "start", "end"})

	for _, j := range s.work.Jobs() {
		x.AppendRows([]table.Row{
			{j.JobDetail.Tenant, j.JobDetail.Compaction.Input, j.Status(), j.WorkerID(), j.CreatedTime(), j.StartTime(), j.EndTime()},
		})
	}

	x.AppendSeparator()

	// x.SetOutputMirror(w)
	w.WriteHeader(http.StatusOK)
	// w.Header().Set("Content-Type", "plain/text")
	_, _ = io.WriteString(w, x.Render())
}

func (s *BackendScheduler) compactions(ctx context.Context) []tempopb.JobDetail {
	var jobs []tempopb.JobDetail

	tenants := s.store.Tenants()
	if len(tenants) == 0 {
		return jobs
	}

	sort.Slice(tenants, func(i, j int) bool { return tenants[i] < tenants[j] })

	for _, tenantID := range tenants {
		if s.overrides.CompactionDisabled(tenantID) {
			continue
		}

		blocklist := s.store.BlockMetas(tenantID)
		window := s.overrides.MaxCompactionRange(tenantID)
		if window == 0 {
			window = s.cfg.Compactor.MaxCompactionRange
		}

		// TODO: A new implementation of the blockSelector would be appropriate
		// here.  Return a stripe of blocks matching eachother.  These can be
		// broken into jobs by the caller.

		blockSelector := blockselector.NewTimeWindowBlockSelector(blocklist,
			window,
			s.cfg.Compactor.MaxCompactionObjects,
			s.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)

		// TODO:
		// measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)

		for {
			if ctx.Err() != nil {
				return jobs
			}

			toBeCompacted, _ := blockSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
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
				Tenant:     tenantID,
				Compaction: compaction,
			}

			jobs = append(jobs, job)
		}

		// TODO:
		// measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)
	}

	return jobs
}
