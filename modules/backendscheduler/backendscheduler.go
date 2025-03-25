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
	"github.com/gogo/status"
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
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/blockselector"
	"github.com/jedib0t/go-pretty/v6/table"
	"google.golang.org/grpc/codes"
)

// var tracer = otel.Tracer("modules/backendscheduler")

// BackendScheduler manages scheduling and execution of backend jobs
type BackendScheduler struct {
	services.Service

	cfg       Config
	store     storage.Store
	overrides overrides.Interface

	work *work.Work

	mtx sync.Mutex

	reader backend.RawReader
	writer backend.RawWriter

	curPriority       *tenantselector.PriorityQueue
	curTenant         *tenantselector.Item
	curSelector       blockselector.CompactionBlockSelector
	curTenantJobCount int
}

// New creates a new BackendScheduler
func New(cfg Config, store storage.Store, overrides overrides.Interface, reader backend.RawReader, writer backend.RawWriter) (*BackendScheduler, error) {
	err := ValidateConfig(&cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	s := &BackendScheduler{
		cfg:         cfg,
		store:       store,
		overrides:   overrides,
		work:        work.New(cfg.Work),
		reader:      reader,
		writer:      writer,
		curPriority: tenantselector.NewPriorityQueue(),
	}

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)
	return s, nil
}

func (s *BackendScheduler) starting(ctx context.Context) error {
	if s.cfg.Poll {
		s.store.EnablePolling(ctx, blocklist.OwnsNothingSharder)
	}

	err := s.loadWorkCache(ctx)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return fmt.Errorf("failed to load work cache: %w", err)
	}

	return nil
}

func (s *BackendScheduler) running(ctx context.Context) error {
	level.Info(log.Logger).Log("msg", "backend scheduler running")

	prioritizeTenantsTicker := time.NewTicker(s.cfg.TenantMeasurementInterval)
	defer prioritizeTenantsTicker.Stop()

	maintenanceTicker := time.NewTicker(s.cfg.MaintenanceInterval)
	defer maintenanceTicker.Stop()

	s.measureTenants()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-maintenanceTicker.C:
			s.work.Prune()
		case <-prioritizeTenantsTicker.C:
			s.measureTenants()
		}
	}
}

func (s *BackendScheduler) stopping(_ error) error {
	return s.flushWorkCache(context.Background())
}

func (s *BackendScheduler) measureTenants() {
	for _, tenant := range s.store.Tenants() {
		window := s.overrides.MaxCompactionRange(tenant)

		if window == 0 {
			window = s.cfg.Compactor.MaxCompactionRange
		}

		blockSelector := blockselector.NewTimeWindowBlockSelector(
			s.store.BlockMetas(tenant),
			window,
			s.cfg.Compactor.MaxCompactionObjects,
			s.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)

		tempodb.MeasureOutstandingBlocks(tenant, blockSelector, ownedYes)
	}
}

// prioritizeTenants prioritizes tenants based on the number of outstanding blocks.
func (s *BackendScheduler) prioritizeTenants() {
	tenants := []tenantselector.Tenant{}

	s.curPriority = tenantselector.NewPriorityQueue() // wipe and restart

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

		tenants = append(tenants, tenantselector.Tenant{
			ID:                         tenantID,
			BlocklistLength:            len(blocklist),
			OutstandingBlocklistLength: outstandingBlocks,
		})
	}

	var (
		ts       = tenantselector.NewBlockListWeightedTenantSelector(tenants)
		item     *tenantselector.Item
		priority int
	)

	for _, tenant := range tenants {
		priority = ts.PriorityForTenant(tenant.ID)
		item = tenantselector.NewItem(tenant.ID, priority)
		heap.Push(s.curPriority, item)
	}
}

// Next implements the BackendSchedulerServer interface.  It returns the next queued job for a worker.
func (s *BackendScheduler) Next(ctx context.Context, req *tempopb.NextJobRequest) (*tempopb.NextJobResponse, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

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

		metricJobsRetry.WithLabelValues(j.JobDetail.Tenant, j.GetType().String(), j.GetWorkerID()).Inc()

		level.Info(log.Logger).Log("msg", "assigned previous job to worker", "job_id", j.ID, "worker", req.WorkerId)

		return resp, nil
	}

	// only one job type right now, just grab the next comapction job. later we can add code to grab the next whatever type of job
	j = s.nextCompactionJob(ctx)
	if j != nil {
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

		metricJobsCreated.WithLabelValues(j.Tenant(), j.GetType().String()).Inc()

		level.Info(log.Logger).Log("msg", "assigned job to worker", "job_id", j.ID, "worker", req.WorkerId)

		return resp, nil
	}

	return &tempopb.NextJobResponse{}, status.Error(codes.NotFound, ErrNoJobsFound.Error())
}

// UpdateJob implements the BackendSchedulerServer interface
func (s *BackendScheduler) UpdateJob(ctx context.Context, req *tempopb.UpdateJobStatusRequest) (*tempopb.UpdateJobStatusResponse, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

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

		var (
			metas     = s.store.BlockMetas(j.Tenant())
			oldBlocks []*backend.BlockMeta
		)

		for _, b := range j.GetCompactionInput() {
			for _, m := range metas {
				if m.BlockID.String() == b {
					oldBlocks = append(oldBlocks, m)
				}
			}
		}

		err = s.store.MarkBlocklistCompacted(j.Tenant(), oldBlocks, nil)
		if err != nil {
			return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.Internal, "failed to mark compacted blocks on in-memory blocklist")
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

func (s *BackendScheduler) StatusHandler(w http.ResponseWriter, _ *http.Request) {
	x := table.NewWriter()
	x.AppendHeader(table.Row{"tenant", "jobID", "input", "output", "status", "worker", "created", "start", "end"})

	for _, j := range s.work.ListJobs() {
		x.AppendRows([]table.Row{
			{
				j.Tenant(),
				j.GetID(),
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

// returns the next compabeingBackendScheduler-0290053e1ction job. this could probably be rewritten cleverly to use yields
// and fewer struct vars
func (s *BackendScheduler) nextCompactionJob(_ context.Context) *work.Job {
	var prioritized bool

	reset := func() {
		s.curSelector = nil
		s.curTenant = nil
		s.curTenantJobCount = 0
	}

	for {
		// do we have an current tenant?
		if s.curSelector != nil {

			if s.curTenantJobCount >= s.cfg.MaxJobsPerTenant {
				reset()
				continue
			}

			toBeCompacted, _ := s.curSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
				// we have drained this selector to the next tenant!
				reset()
				continue
			}

			input := make([]string, 0, len(toBeCompacted))
			for _, b := range toBeCompacted {
				input = append(input, b.BlockID.String())
			}

			compaction := &tempopb.CompactionDetail{
				Input: input,
			}

			s.curTenantJobCount++

			return &work.Job{
				ID:   uuid.New().String(),
				Type: tempopb.JobType_JOB_TYPE_COMPACTION,
				JobDetail: tempopb.JobDetail{
					Tenant:     s.curTenant.Value(),
					Compaction: compaction,
				},
			}
		}

		// we don't have a current tenant, get the next one

		if s.curPriority.Len() == 0 {
			if prioritized {
				// no compactions are needed and we've already prioritized once
				return nil
			}
			s.prioritizeTenants()
			prioritized = true
		}

		if s.curPriority.Len() == 0 {
			// no tenants = no jobs
			return nil
		}

		s.curTenant = heap.Pop(s.curPriority).(*tenantselector.Item)
		if s.curTenant == nil {
			return nil
		}

		// set the block selector and go to the top to find a job for this tenant
		s.curSelector = blockselector.NewTimeWindowBlockSelector(
			s.store.BlockMetas(s.curTenant.Value()),
			s.cfg.Compactor.MaxCompactionRange,
			s.cfg.Compactor.MaxCompactionObjects,
			s.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)
	}
}

func ownedYes(_ string) bool {
	return true
}
