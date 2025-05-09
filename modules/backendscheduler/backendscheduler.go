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
	"github.com/gogo/status"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/backendscheduler/provider"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/jedib0t/go-pretty/v6/table"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
)

var tracer = otel.Tracer("modules/backendscheduler")

// BackendScheduler manages scheduling and execution of backend jobs
type BackendScheduler struct {
	services.Service

	cfg       Config
	store     storage.Store
	overrides overrides.Interface

	work *work.Work

	reader backend.RawReader
	writer backend.RawWriter

	providers []struct {
		provider provider.Provider
		jobs     <-chan *work.Job
	}

	mergedJobs chan *work.Job
}

// ListJobs returns all jobs in the work cache
func (s *BackendScheduler) ListJobs() []*work.Job {
	return s.work.ListJobs()
}

// New creates a new BackendScheduler
func New(cfg Config, store storage.Store, overrides overrides.Interface, reader backend.RawReader, writer backend.RawWriter) (*BackendScheduler, error) {
	err := ValidateConfig(&cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	s := &BackendScheduler{
		cfg:        cfg,
		store:      store,
		overrides:  overrides,
		work:       work.New(cfg.Work),
		reader:     reader,
		writer:     writer,
		mergedJobs: make(chan *work.Job, 1),
	}

	// Initialize providers
	s.providers = []struct {
		provider provider.Provider
		jobs     <-chan *work.Job
	}{
		{
			provider: provider.NewCompactionProvider(
				s.cfg.ProviderConfig.Compaction,
				log.Logger,
				s.store,
				s.overrides,
				s.work,
			),
			jobs: nil, // Will be set in running
		},
		{
			provider: provider.NewRetentionProvider(
				s.cfg.ProviderConfig.Retention,
				log.Logger,
				s.work,
			),
			jobs: nil, // Will be set in running
		},
	}

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)
	return s, nil
}

func (s *BackendScheduler) starting(ctx context.Context) error {
	level.Info(log.Logger).Log("msg", "backend scheduler starting")

	if s.cfg.Poll {
		s.store.EnablePolling(ctx, blocklist.OwnsNothingSharder)
	}

	err := s.loadWorkCache(ctx)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return fmt.Errorf("failed to load work cache: %w", err)
	}

	// Start providers and collect job channels
	s.mergedJobs = make(chan *work.Job, len(s.providers))

	wg := sync.WaitGroup{}

	for i := range s.providers {
		s.providers[i].jobs = s.providers[i].provider.Start(ctx)

		wg.Add(1)
		// Start a goroutine to forward jobs from each provider to the merged channel
		go func(jobs <-chan *work.Job) {
			defer wg.Done()

			var job *work.Job

			for {
				select {
				case job = <-jobs:
				case <-ctx.Done():
					level.Info(log.Logger).Log("msg", "stopping provider", "provider", i)
					return
				}

				select {
				case s.mergedJobs <- job:
					metricProviderJobsMerged.WithLabelValues(fmt.Sprintf("%d", i)).Inc()
				case <-ctx.Done():
					level.Info(log.Logger).Log("msg", "stopping provider", "provider", i)
					return
				}
			}
		}(s.providers[i].jobs)
	}

	// Start a goroutine to close the merged channel when all providers are done
	go func() {
		wg.Wait()
		level.Info(log.Logger).Log("msg", "all providers stopped")
		close(s.mergedJobs)
	}()

	return nil
}

func (s *BackendScheduler) running(ctx context.Context) error {
	level.Info(log.Logger).Log("msg", "backend scheduler running")

	maintenanceTicker := time.NewTicker(s.cfg.MaintenanceInterval)
	defer maintenanceTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-maintenanceTicker.C:
			s.work.Prune()
		}
	}
}

func (s *BackendScheduler) stopping(_ error) error {
	err := s.flushWorkCache(context.Background())
	if err != nil {
		return fmt.Errorf("failed to flush work cache on shutdown: %w", err)
	}

	level.Info(log.Logger).Log("msg", "backend scheduler stopping")
	return nil
}

// Next implements the BackendSchedulerServer interface.  It returns the next queued job for a worker.
func (s *BackendScheduler) Next(ctx context.Context, req *tempopb.NextJobRequest) (*tempopb.NextJobResponse, error) {
	ctx, span := tracer.Start(ctx, "Next")
	defer span.End()

	span.SetAttributes(attribute.String("worker_id", req.WorkerId))

	// Find jobs that already exist for this worker
	j := s.work.GetJobForWorker(req.WorkerId)
	if j != nil {
		resp := &tempopb.NextJobResponse{
			JobId:  j.ID,
			Type:   j.Type,
			Detail: j.JobDetail,
		}

		// The job exists in memory, but may not have been persisted to the backend.
		err := s.flushWorkCache(ctx)
		if err != nil {
			// Fail without returning the job if we can't update the job cache.
			return &tempopb.NextJobResponse{}, status.Error(codes.Internal, ErrFlushFailed.Error())
		}

		span.SetAttributes(attribute.String("job_id", j.ID))

		metricJobsRetry.WithLabelValues(j.JobDetail.Tenant, j.GetType().String(), j.GetWorkerID()).Inc()

		level.Info(log.Logger).Log("msg", "assigned previous job to worker", "job_id", j.ID, "worker", req.WorkerId)

		return resp, nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, s.cfg.JobTimeout)
	defer cancel()

	// Try to get a job from the merged channel
	select {
	case j := <-s.mergedJobs:
		if j == nil {
			// Channel closed, no jobs available
			metricJobsNotFound.WithLabelValues(req.WorkerId).Inc()
			return &tempopb.NextJobResponse{}, status.Error(codes.Internal, ErrNilJob.Error())
		}

		resp := &tempopb.NextJobResponse{
			JobId:  j.ID,
			Type:   j.Type,
			Detail: j.JobDetail,
		}

		j.SetWorkerID(req.WorkerId)
		err := s.work.AddJob(j)
		if err != nil {
			return &tempopb.NextJobResponse{}, status.Error(codes.Internal, err.Error())
		}

		j.Start()
		metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.GetType().String()).Inc()

		err = s.flushWorkCache(ctx)
		if err != nil {
			// Fail without returning the job if we can't update the job cache
			return &tempopb.NextJobResponse{}, status.Error(codes.Internal, ErrFlushFailed.Error())
		}

		span.SetAttributes(attribute.String("job_id", j.ID))

		metricJobsCreated.WithLabelValues(j.Tenant(), j.GetType().String()).Inc()

		level.Info(log.Logger).Log("msg", "assigned job to worker", "job_id", j.ID, "worker", req.WorkerId)

		return resp, nil
	case <-timeoutCtx.Done():
		span.SetAttributes(attribute.Int("job_q_depth", len(s.mergedJobs)))
		metricJobsNotFound.WithLabelValues(req.WorkerId).Inc()

		return &tempopb.NextJobResponse{}, status.Error(codes.NotFound, ErrNoJobsFound.Error())
	}
}

// UpdateJob implements the BackendSchedulerServer interface
func (s *BackendScheduler) UpdateJob(ctx context.Context, req *tempopb.UpdateJobStatusRequest) (*tempopb.UpdateJobStatusResponse, error) {
	j := s.work.GetJob(req.JobId)
	if j == nil {
		return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.NotFound, work.ErrJobNotFound.Error())
	}

	switch req.Status {
	case tempopb.JobStatus_JOB_STATUS_RUNNING:
	case tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
		j.Complete()
		metricJobsCompleted.WithLabelValues(j.JobDetail.Tenant, j.GetType().String()).Inc()
		metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.GetType().String()).Dec()
		level.Info(log.Logger).Log("msg", "job completed", "job_id", req.JobId)

		if req.Compaction != nil && req.Compaction.Output != nil {
			j.SetCompactionOutput(req.Compaction.Output)
		}

		err := s.flushWorkCache(ctx)
		if err != nil {
			// Fail without returning the job if we can't update the job cache.
			return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.Internal, ErrFlushFailed.Error())
		}

		err = s.loadBlocklistJobsForTenant(j.Tenant(), []*work.Job{j})
		if err != nil {
			return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.Internal, err.Error())
		}
	case tempopb.JobStatus_JOB_STATUS_FAILED:
		j.Fail()
		metricJobsFailed.WithLabelValues(j.Tenant(), j.GetType().String()).Inc()
		metricJobsActive.WithLabelValues(j.Tenant(), j.GetType().String()).Dec()
		level.Error(log.Logger).Log("msg", "job failed", "job_id", req.JobId, "error", req.Error)

		err := s.flushWorkCache(ctx)
		if err != nil {
			// Fail without returning the job if we can't update the job cache.
			return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.Internal, ErrFlushFailed.Error())
		}

	default:
		return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.InvalidArgument, "invalid job status")
	}

	return &tempopb.UpdateJobStatusResponse{
		Success: true,
	}, nil
}

func (s *BackendScheduler) replayWorkOnBlocklist() {
	var (
		err           error
		tenant        string
		perTenantJobs = make(map[string][]*work.Job)
	)

	// Get all the input blocks which have been successfully compacted
	for _, j := range s.work.ListJobs() {
		tenant = j.Tenant()

		if j.GetStatus() != tempopb.JobStatus_JOB_STATUS_SUCCEEDED {
			continue
		}

		if _, ok := perTenantJobs[tenant]; !ok {
			perTenantJobs[tenant] = make([]*work.Job, 0, 1000)
		}

		perTenantJobs[tenant] = append(perTenantJobs[tenant], j)
	}

	for tenant, jobs := range perTenantJobs {
		err = s.loadBlocklistJobsForTenant(tenant, jobs)
		if err != nil {
			level.Error(log.Logger).Log("msg", "failed to load blocklist jobs for tenant", "tenant", tenant, "error", err)
		}
	}
}

func (s *BackendScheduler) loadBlocklistJobsForTenant(tenant string, jobs []*work.Job) error {
	var (
		metas     = s.store.BlockMetas(tenant)
		oldBlocks []*backend.BlockMeta
		u         backend.UUID
		err       error
	)

	for _, j := range jobs {
		if j.GetStatus() != tempopb.JobStatus_JOB_STATUS_SUCCEEDED {
			continue
		}

		for _, b := range j.GetCompactionInput() {
			u, err = backend.ParseUUID(b)
			if err != nil {
				level.Error(log.Logger).Log("msg", "failed to parse block ID", "block_id", b, "error", err)
				continue
			}

			for _, m := range metas {
				if m.BlockID == u {
					oldBlocks = append(oldBlocks, m)
					break
				}
			}
		}
	}

	err = s.store.MarkBlocklistCompacted(tenant, oldBlocks, nil)
	if err != nil {
		return fmt.Errorf("failed to mark compacted blocks on in-memory blocklist: %w", err)
	}

	return nil
}

func (s *BackendScheduler) StatusHandler(w http.ResponseWriter, _ *http.Request) {
	x := table.NewWriter()
	x.AppendHeader(table.Row{"tenant", "jobID", "type", "input", "output", "status", "worker", "created", "start", "end"})

	for _, j := range s.work.ListJobs() {
		x.AppendRows([]table.Row{
			{
				j.Tenant(),
				j.GetID(),
				j.GetType().String(),
				j.GetCompactionInput(),
				j.GetCompactionOutput(),
				j.GetStatus().String(),
				j.GetWorkerID(),
				j.GetCreatedTime(),
				j.GetStartTime(),
				j.GetEndTime(),
			},
		})
	}

	x.AppendSeparator()

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, x.Render())
}
