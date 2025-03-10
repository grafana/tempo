package backendscheduler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/backendscheduler/work/tenantselector"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
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

	work    *work.Work
	workMtx sync.RWMutex

	reader backend.RawReader
	writer backend.RawWriter

	tenantPriority *tenantselector.PriorityQueue
	tenantMtx      sync.RWMutex
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
func New(cfg Config, store storage.Store, overrides overrides.Interface, reader backend.RawReader, writer backend.RawWriter) (*BackendScheduler, error) {
	s := &BackendScheduler{
		cfg:       cfg,
		store:     store,
		overrides: overrides,
		// work:           work.NewPerTenantWork(store, work.FlatPriority),
		work:           work.New(),
		reader:         reader,
		writer:         writer,
		tenantPriority: tenantselector.NewPriorityQueue(),
	}

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)
	return s, nil
}

func (s *BackendScheduler) starting(ctx context.Context) error {
	if s.cfg.Poll {
		s.store.EnablePolling(ctx, s)
	}

	err := s.loadWorkCache(ctx)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return fmt.Errorf("failed to load work cache: %w", err)
	}

	return nil
}

func (s *BackendScheduler) running(ctx context.Context) error {
	level.Info(log.Logger).Log("msg", "backend scheduler running")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	s.prioritizeTenants()

	if err := s.scheduleOnce(ctx); err != nil {
		return fmt.Errorf("failed to schedule initial jobs: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.work.Prune()

			if s.work.Len() < s.cfg.MaxPendingWorkQueue {
				s.prioritizeTenants()

				if err := s.scheduleOnce(ctx); err != nil {
					level.Error(log.Logger).Log("msg", "scheduling cycle failed", "err", err)
					metricSchedulingCycles.WithLabelValues("failed").Inc()
				} else {
					metricSchedulingCycles.WithLabelValues("success").Inc()
				}
			}
		}
	}
}

func (s *BackendScheduler) stopping(_ error) error {
	return s.flushWorkCache(context.Background())
}

func (s *BackendScheduler) prioritizeTenants() {
	tenants := []tenantselector.Tenant{}

	for _, tenantID := range s.store.Tenants() {
		if s.overrides.CompactionDisabled(tenantID) {
			continue
		}

		var (
			blocklist         = s.store.BlockMetas(tenantID)
			window            = s.overrides.MaxCompactionRange(tenantID)
			outstandingBlocks = 0
		)

		if window == 0 {
			window = s.cfg.Compactor.MaxCompactionRange
		}

		// TODO: consider using a different blockselector for this
		blockSelector := blockselector.NewTimeWindowBlockSelector(blocklist,
			window,
			s.cfg.Compactor.MaxCompactionObjects,
			s.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)

		// Measure the outstanding blocks
		for {
			toBeCompacted, _ := blockSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
				break
			}

			outstandingBlocks += len(toBeCompacted)
		}

		tenants = append(tenants, tenantselector.Tenant{
			ID:                         tenantID,
			BlocklistLength:            len(blocklist),
			OutstanidngBlocklistLength: outstandingBlocks,
		})
	}

	ts := tenantselector.NewBlockListWeightedTenantSelector(tenants)

	for _, tenant := range tenants {
		priority := ts.PriorityForTenant(tenant.ID)
		item := tenantselector.NewItem(tenant.ID, priority)
		s.tenantPriority.Push(item)
	}
}

// ScheduleOnce schedules jobs for compaction
func (s *BackendScheduler) scheduleOnce(ctx context.Context) error {
	// TODO: pass a max jobs to schedule
	for _, job := range s.compactions(ctx, 100) {
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

	metricJobsCreated.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
	metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()

	level.Info(log.Logger).Log("msg", "created new job", "job_id", j.ID, "tenant", j.JobDetail.Tenant, "type", j.Type.String())
	return nil
}

// GetJob returns a job by ID
func (s *BackendScheduler) GetJob(_ context.Context, jobID string) (*work.Job, error) {
	j := s.work.GetJob(jobID)
	if j == nil {
		return nil, fmt.Errorf("%w: %s", work.ErrJobNotFound, jobID)
	}

	return j, nil
}

// CompleteJob marks a job as completed
func (s *BackendScheduler) CompleteJob(_ context.Context, jobID string) error {
	j := s.work.GetJob(jobID)
	if j == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	j.Complete()
	metricJobsCompleted.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
	metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "completed job", "job_id", jobID, "type", j.Type.String())
	return nil
}

// FailJob marks a job as failed
func (s *BackendScheduler) FailJob(_ context.Context, jobID string) error {
	j := s.work.GetJob(jobID)
	if j == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	j.Fail()
	metricJobsFailed.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
	metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "failed job", "job_id", jobID, "tenant", j.JobDetail.Tenant, "type", j.Type.String())
	return nil
}

// Next implements the BackendSchedulerServer interface.  It returns the next queued job for a worker.
func (s *BackendScheduler) Next(ctx context.Context, req *tempopb.NextJobRequest) (*tempopb.NextJobResponse, error) {
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

			// TODO: consider the flow here.  Before we return the job the worker, we
			// should flush the contents to the backend.  What is the faiure
			// condition here?  If we fail to flush the cache, we should not return
			// the job to the worker, but we've recorded it as assigned.  Next Next()
			// call, the worker will receive that same job, but if we crash before
			// then we never returned it so losing the job record is probably fine.

			err := s.flushWorkCache(ctx)
			if err != nil {
				level.Error(log.Logger).Log("msg", "failed to flush work cache", "err", err)
			}

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
		return nil, fmt.Errorf("%w: %s", work.ErrJobNotFound, req.JobId)
	}

	switch req.Status {
	case tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
		j.Complete()
		metricJobsCompleted.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
		level.Info(log.Logger).Log("msg", "job completed", "job_id", req.JobId)

		// FIXME: update the blocklist

	case tempopb.JobStatus_JOB_STATUS_FAILED:
		j.Fail()
		metricJobsFailed.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
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
		// FIXME: constrain this to the tenant instead of looping AllJobs
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

	x.AppendHeader(table.Row{"tenant", "priority"})

	for _, item := range s.tenantPriority.Items() {
		x.AppendRow([]interface{}{item.Value(), item.Priority()})
	}

	x.AppendSeparator()

	// x.SetOutputMirror(w)
	w.WriteHeader(http.StatusOK)
	// w.Header().Set("Content-Type", "plain/text")
	_, _ = io.WriteString(w, x.Render())
}

func (s *BackendScheduler) nextTenant(ctx context.Context) string {
	var tenantID string

	// BUG: don't pop unless Len() > 0

	for tenantID = s.tenantPriority.Pop().(*tenantselector.Item).Value(); !s.overrides.CompactionDisabled(tenantID); {
		item := tenantselector.NewItem(tenantID, 0)
		s.tenantPriority.Push(item)
	}

	return tenantID
}

func (s *BackendScheduler) compactions(ctx context.Context, want int) []tempopb.JobDetail {
	var jobs []tempopb.JobDetail

	// Get the highest priority tenant which has compaction enabled
	tenantID := s.nextTenant(ctx)

	var (
		blocklist = s.store.BlockMetas(tenantID)
		window    = s.overrides.MaxCompactionRange(tenantID)
	)

	if window == 0 {
		window = s.cfg.Compactor.MaxCompactionRange
	}

	blockSelector := blockselector.NewTimeWindowBlockSelector(blocklist,
		window,
		s.cfg.Compactor.MaxCompactionObjects,
		s.cfg.Compactor.MaxBlockBytes,
		blockselector.DefaultMinInputBlocks,
		blockselector.DefaultMaxInputBlocks,
	)

	for {
		if ctx.Err() != nil {
			tempodb.MeasureOutstandingBlocks(tenantID, blockSelector, s.Owns)
			return jobs
		}

		if len(jobs) >= want {
			break
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

	// Measure when we are done with the blockSelector, since calling it again
	// removes the returned blocks.
	tempodb.MeasureOutstandingBlocks(tenantID, blockSelector, s.Owns)

	return jobs
}
