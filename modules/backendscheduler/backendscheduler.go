package backendscheduler

import (
	"container/heap"
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
		cfg:            cfg,
		store:          store,
		overrides:      overrides,
		work:           work.New(cfg.Work),
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

	scheduleTicker := time.NewTicker(time.Second)
	defer scheduleTicker.Stop()

	prioritizeTenantsTicker := time.NewTicker(time.Minute)
	defer prioritizeTenantsTicker.Stop()

	s.prioritizeTenants()

	if err := s.scheduleOnce(ctx, s.cfg.MaxPendingWorkQueue); err != nil {
		return fmt.Errorf("failed to schedule initial jobs: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-prioritizeTenantsTicker.C:
			s.prioritizeTenants()
		case <-scheduleTicker.C:
			s.work.Prune()
			if workLen := s.work.Len(); workLen < s.cfg.MinPendingWorkQueue {
				toAdd := s.cfg.MaxPendingWorkQueue - workLen
				if err := s.scheduleOnce(ctx, toAdd); err != nil {
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
	s.tenantMtx.Lock()
	defer s.tenantMtx.Unlock()

	tenants := []tenantselector.Tenant{}

	for _, tenantID := range s.store.Tenants() {
		if s.overrides.CompactionDisabled(tenantID) {
			continue
		}

		var (
			blocklist         = s.store.BlockMetas(tenantID)
			window            = s.overrides.MaxCompactionRange(tenantID)
			outstandingBlocks = 0
			toBeCompacted     []*backend.BlockMeta
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
			toBeCompacted, _ = blockSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
				break
			}

			outstandingBlocks += len(toBeCompacted)
		}

		// Measure the last time this tenant was worked
		lastCompactedTime := time.Time{}
		t := s.store.LastCompacted(tenantID)
		if t != nil {
			lastCompactedTime = *t
		}

		if !lastCompactedTime.IsZero() {
			lastCompactedTime = s.lastWorkForTenant(tenantID)
		}

		tenants = append(tenants, tenantselector.Tenant{
			ID:                         tenantID,
			BlocklistLength:            len(blocklist),
			OutstanidngBlocklistLength: outstandingBlocks,
			LastWork:                   lastCompactedTime,
		})
	}

	var (
		ts       = tenantselector.NewBlockListWeightedTenantSelector(tenants)
		items    = s.tenantPriority.UpdatePriority(ts)
		item     *tenantselector.Item
		priority int
	)

	for _, tenant := range tenants {
		if _, ok := items[tenant.ID]; !ok {
			priority = ts.PriorityForTenant(tenant.ID)
			item = tenantselector.NewItem(tenant.ID, priority)
			heap.Push(s.tenantPriority, item)
		}
	}
}

func (s *BackendScheduler) lastWorkForTenant(tenantID string) time.Time {
	s.workMtx.RLock()
	defer s.workMtx.RUnlock()

	// Get the most recent time
	var lastWork time.Time
	for _, j := range s.work.ListJobs() {
		if j.JobDetail.Tenant == tenantID {
			if j.GetEndTime().After(lastWork) {
				lastWork = j.GetEndTime()
			}
		}
	}

	return lastWork
}

// ScheduleOnce schedules jobs for compaction
func (s *BackendScheduler) scheduleOnce(ctx context.Context, toAdd int) error {
	for _, job := range s.compactions(ctx, toAdd) {
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

	// TODO: flush the cache here

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

	// TODO: flush the cache here

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

	// TODO: flush the cache here

	j.Fail()
	metricJobsFailed.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
	metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "failed job", "job_id", jobID, "tenant", j.JobDetail.Tenant, "type", j.Type.String())
	return nil
}

// Next implements the BackendSchedulerServer interface.  It returns the next queued job for a worker.
func (s *BackendScheduler) Next(ctx context.Context, req *tempopb.NextJobRequest) (*tempopb.NextJobResponse, error) {
	// Find jobs that already exist for this worker
	for id, j := range s.work.ListJobs() {
		if j.IsPending() || j.IsRunning() {
			if j.GetWorkerID() == req.WorkerId {
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
	for id, j := range s.work.ListJobs() {
		if j.IsPending() {
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

			j.SetWorkerID(req.WorkerId)

			err := s.flushWorkCache(ctx)
			if err != nil {
				// Fail without returning the job if we can't update the job cache
				return &tempopb.NextJobResponse{}, fmt.Errorf("failed to flush work cache: %w", err)
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
	case tempopb.JobStatus_JOB_STATUS_RUNNING:
		j.Start()
		metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
		level.Info(log.Logger).Log("msg", "job started", "job_id", req.JobId, "worker_id", j.GetWorkerID())
	case tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
		j.Complete()
		metricJobsCompleted.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
		metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Dec()
		level.Info(log.Logger).Log("msg", "job completed", "job_id", req.JobId)

		j.JobDetail.Compaction.Output = req.Compaction.Output

		var (
			metas     = s.store.BlockMetas(j.JobDetail.Tenant)
			oldBlocks []*backend.BlockMeta
		)

		for _, b := range j.JobDetail.Compaction.Input {
			for _, m := range metas {
				if m.BlockID.String() == b {
					oldBlocks = append(oldBlocks, m)
				}
			}
		}

		s.store.MarkBlocklistCompacted(j.JobDetail.Tenant, oldBlocks, nil)

	case tempopb.JobStatus_JOB_STATUS_FAILED:
		j.Fail()
		metricJobsFailed.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Inc()
		metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.Type.String()).Dec()
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
	return s.work.ListJobs()
}

// CreateCompactionJob creates a new compaction job for the given tenant and blocks.
// Must be called under jobsMtx lock.
func (s *BackendScheduler) createCompactionJob(ctx context.Context, tenantID string, input []string) error {
	// Skip blocks which already have a job
	for _, blockID := range input {
		for _, j := range s.work.ListJobs() {
			if j.JobDetail.Tenant == tenantID {
				switch j.Type {
				case tempopb.JobType_JOB_TYPE_COMPACTION:
					for _, b := range j.JobDetail.Compaction.Input {
						if b == blockID {
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

	return s.flushWorkCache(ctx)
}

func (s *BackendScheduler) Owns(_ string) bool {
	return true
}

func (s *BackendScheduler) StatusHandler(w http.ResponseWriter, _ *http.Request) {
	s.tenantMtx.RLock()
	defer s.tenantMtx.RUnlock()

	x := table.NewWriter()
	x.AppendHeader(table.Row{"tenant", "input", "output", "status", "worker", "created", "start", "end"})

	for _, j := range s.work.ListJobs() {
		x.AppendRows([]table.Row{
			{j.JobDetail.Tenant, j.JobDetail.Compaction.Input, j.JobDetail.Compaction.Output, j.GetStatus().String(), j.GetWorkerID(), j.GetCreatedTime(), j.GetStartTime(), j.GetEndTime()},
		})
	}

	x.AppendSeparator()

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, x.Render())

	x = table.NewWriter()
	x.AppendHeader(table.Row{"tenant", "priority", "last_work", "blocks"})

	for _, item := range s.tenantPriority.Items() {
		x.AppendRow([]interface{}{item.Value(), item.Priority(), s.store.LastCompacted(item.Value()), len(s.store.BlockMetas(item.Value()))})
	}

	x.AppendSeparator()

	// x.SetOutputMirror(w)
	// w.Header().Set("Content-Type", "plain/text")
	_, _ = io.WriteString(w, x.Render())
}

func (s *BackendScheduler) nextTenant(ctx context.Context) *tenantselector.Item {
	s.tenantMtx.RLock()
	defer s.tenantMtx.RUnlock()

	if s.tenantPriority.Len() > 0 {
		for tenant := heap.Pop(s.tenantPriority).(*tenantselector.Item); !s.overrides.CompactionDisabled(tenant.Value()); {
			heap.Push(s.tenantPriority, tenant)
			s.tenantPriority.Update(tenant, tenant.Value(), 0)
			return tenant
		}
	}

	return nil
}

func (s *BackendScheduler) compactions(ctx context.Context, want int) []tempopb.JobDetail {
	var (
		jobs      []tempopb.JobDetail
		tenant    = s.nextTenant(ctx)
		tenantID  = tenant.Value()
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

	if len(jobs) > 0 {
		level.Info(log.Logger).Log("msg", "compaction jobs scheduled", "jobs", len(jobs), "tenant", tenantID)
	}

	return jobs
}
