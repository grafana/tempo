package backendscheduler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/backendscheduler/provider"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/jedib0t/go-pretty/v6/table"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
)

var tracer = otel.Tracer("modules/backendscheduler")

// BackendScheduler manages scheduling and execution of backend jobs
type BackendScheduler struct {
	services.Service

	mtx sync.Mutex

	cfg       Config
	store     storage.Store
	overrides overrides.Interface

	work work.Interface

	reader backend.RawReader
	writer backend.RawWriter

	providers []struct {
		provider provider.Provider
		jobs     <-chan *work.Job
	}

	mergedJobs chan *work.Job

	// publishedPendingCounts is the jobs_pending snapshot published on the previous maintenance tick,
	// keyed exactly as work.PendingJobCounts returns it. Comparing the next snapshot against it
	// identifies the tenants and job types whose queue has drained, so their series can be deleted
	// without resetting the whole vector. Touched only from the maintenance loop.
	publishedPendingCounts map[string]map[tempopb.JobType]int
}

// ListJobs returns all jobs in the work cache
func (s *BackendScheduler) ListJobs() []*work.Job {
	return s.work.ListJobs()
}

// RegisterJob delegates to work.Work, satisfying the provider.Scheduler interface.
func (s *BackendScheduler) RegisterJob(job *work.Job) {
	s.work.RegisterJob(job)
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
				s.store,
				s.overrides,
				s.work,
			),
			jobs: nil, // Will be set in running
		},
		{
			provider: provider.NewRedactionProvider(
				s.cfg.ProviderConfig.Redaction,
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
		s.store.EnablePolling(ctx, blocklist.OwnsNothingSharder, true)
	}

	err := s.loadWorkCache(ctx)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return fmt.Errorf("failed to load work cache: %w", err)
	}

	// Load the batch manifest (best-effort; missing file means no active redaction batches).
	if err := s.work.LoadBatchesFromLocal(ctx, s.cfg.LocalWorkPath); err != nil {
		level.Info(log.Logger).Log("msg", "no batch manifest found at startup", "err", err)
	}
	s.dropBatchesWithUnusableWindows()

	wg := sync.WaitGroup{}

	for i := range s.providers {
		s.providers[i].jobs = s.providers[i].provider.Start(ctx)

		wg.Add(1)
		// Start a goroutine to forward jobs from each provider to the merged channel
		go func(jobs <-chan *work.Job) {
			defer wg.Done()

			var job *work.Job

			for {
				var ok bool
				select {
				case job, ok = <-jobs:
					if !ok {
						// Provider closed its channel (it has stopped); nothing more to forward.
						level.Info(log.Logger).Log("msg", "provider channel closed", "provider", i)
						return
					}
				case <-ctx.Done():
					level.Info(log.Logger).Log("msg", "stopping provider", "provider", i)
					// The provider channel is buffered: drain any job it already handed off but we
					// never forwarded, releasing the in-flight count of redaction jobs so it does
					// not leak on shutdown.
					for {
						select {
						case j, ok := <-jobs:
							// A closed channel is always ready, so use the two-value receive and
							// return on close; otherwise this loop would spin forever.
							if !ok {
								return
							}
							if j != nil && j.GetType() == tempopb.JobType_JOB_TYPE_REDACTION {
								s.work.ReleaseRedactionInFlight(j.Tenant())
							}
						default:
							return
						}
					}
				}

				select {
				case s.mergedJobs <- job:
					metricProviderJobsMerged.WithLabelValues(strconv.Itoa(i)).Inc()
				case <-ctx.Done():
					level.Info(log.Logger).Log("msg", "stopping provider", "provider", i)
					// This job was received but not forwarded; if it's a redaction job it was
					// counted in-flight at dequeue and will never reach Next(), so release it.
					if job != nil && job.GetType() == tempopb.JobType_JOB_TYPE_REDACTION {
						s.work.ReleaseRedactionInFlight(job.Tenant())
					}
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

	backendFlushTicker := time.NewTicker(s.cfg.BackendFlushInterval)
	defer backendFlushTicker.Stop()

	// Publish once up front: on the tick alone the metric would be absent for a full
	// MaintenanceInterval after start, so a dashboard or autoscaler reading it right after a restart
	// would see nothing rather than the queue that survived the restart.
	s.recordPendingJobs()

	var err error

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-maintenanceTicker.C:
			s.work.Prune(ctx)
			s.checkPendingRescans(ctx)
			s.cleanupOrphanedBatches(ctx)
			s.recordPendingJobs()
		case <-backendFlushTicker.C:
			err = s.flushWorkCacheToBackend(ctx)
			metricWorkFlushes.Inc()
			if err != nil && !errors.Is(err, context.Canceled) {
				metricWorkFlushesFailed.Inc()
				level.Error(log.Logger).Log("msg", "failed to flush work cache to backend", "error", err)
			}

		}
	}
}

func (s *BackendScheduler) stopping(_ error) error {
	err := s.work.FlushToLocal(context.Background(), s.cfg.LocalWorkPath, nil) // flush all shards
	if err != nil {
		return fmt.Errorf("failed to flush work cache on shutdown: %w", err)
	}

	if err := s.work.FlushBatchesToLocal(context.Background(), s.cfg.LocalWorkPath); err != nil {
		level.Warn(log.Logger).Log("msg", "failed to flush batch manifest on shutdown", "err", err)
	}

	err = s.flushWorkCacheToBackend(context.Background())
	if err != nil {
		return fmt.Errorf("failed to flush work cache to backend on shutdown: %w", err)
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
	j := s.work.GetJobForWorker(ctx, req.WorkerId)
	if j != nil {
		resp := &tempopb.NextJobResponse{
			JobId:  j.ID,
			Type:   j.Type,
			Detail: j.JobDetail,
		}

		// The job exists in memory, but may not have been persisted to disk.
		err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, []string{j.ID})
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

	// Loop so that stale jobs (whose preconditions no longer hold) can be
	// silently discarded and we immediately try the next one, rather than
	// handing an invalid job to a worker.
	for {
		select {
		case j := <-s.mergedJobs:
			if j == nil {
				// Channel closed, no jobs available
				metricJobsNotFound.WithLabelValues(req.WorkerId).Inc()
				return &tempopb.NextJobResponse{}, status.Error(codes.Internal, ErrNilJob.Error())
			}

			span.AddEvent("job received", trace.WithAttributes(
				attribute.String("job_id", j.GetID()),
			))

			// All current job types require a tenant. Legacy global retention jobs
			// emitted by old scheduler binaries have an empty tenant and bypass
			// the per-type precondition checks.
			if j.Tenant() == "" {
				level.Debug(log.Logger).Log("msg", "legacy global job without tenant, passing through",
					"job_id", j.ID, "type", j.GetType().String())
			} else {
				drop := false
				switch j.GetType() {
				case tempopb.JobType_JOB_TYPE_RETENTION:
					// A redaction may have been submitted after this job was emitted.
					// Drop and retry to avoid running retention over a mid-redaction
					// tenant. Gate on the batch barrier (TenantPending), not just
					// in-flight redaction jobs, so a batch in its rescan-wait window
					// (no jobs in flight) still blocks retention.
					if s.work.TenantPending(j.Tenant()) {
						level.Debug(log.Logger).Log("msg", "dropping stale retention job: tenant has active redaction batch",
							"job_id", j.ID, "tenant", j.Tenant())
						metricJobsDropped.WithLabelValues(j.Tenant(), j.GetType().String()).Inc()
						drop = true
					}
				case tempopb.JobType_JOB_TYPE_REDACTION:
					// Resolve trace IDs from the batch manifest.
					// Drop if the batch no longer exists (cancelled or already cleaned up).
					batch := s.work.GetBatch(j.Tenant())
					if batch == nil {
						level.Debug(log.Logger).Log("msg", "dropping redaction job: batch no longer exists",
							"job_id", j.ID, "tenant", j.Tenant())
						metricJobsDropped.WithLabelValues(j.Tenant(), j.GetType().String()).Inc()
						// This job was counted in-flight when NextPendingJob dequeued it; since it is
						// dropped rather than promoted via AddJob, release that count or it leaks.
						s.work.ReleaseRedactionInFlight(j.Tenant())
						drop = true
					} else if j.JobDetail.Redaction != nil {
						// Inject the batch's selector (trace IDs or query) and mode so the
						// worker can resolve and act on the block without re-reading the batch.
						j.JobDetail.Redaction.TraceIds = batch.TraceIds
						j.JobDetail.Redaction.Query = batch.Query
						// A verification job keeps its dry-run intent. Injecting the batch's APPLY
						// here is what a worker that predates the verify field would act on, turning
						// a scan into a rewrite of every block in the pass.
						if !j.JobDetail.Redaction.Verify {
							j.JobDetail.Redaction.Mode = batch.Mode
						} else {
							j.JobDetail.Redaction.Mode = tempopb.RedactionMode_REDACTION_MODE_DRY_RUN
						}
						// A job carrying its own window keeps it: verification derives a narrower
						// window than the batch's, and overwriting it would unbound the scan.
						if j.JobDetail.Redaction.StartTimeUnixNano == 0 && j.JobDetail.Redaction.EndTimeUnixNano == 0 {
							j.JobDetail.Redaction.StartTimeUnixNano = batch.StartTimeUnixNano
							j.JobDetail.Redaction.EndTimeUnixNano = batch.EndTimeUnixNano
						}
					}
				}
				if drop {
					continue
				}
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

			s.work.StartJob(j.ID)
			metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.GetType().String()).Inc()

			err = s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, []string{j.ID})
			if err != nil {
				// Fail without returning the job if we can't update the job cache
				return &tempopb.NextJobResponse{}, status.Error(codes.Internal, ErrFlushFailed.Error())
			}

			span.SetAttributes(attribute.String("job_id", j.ID))

			metricJobsCreated.WithLabelValues(resp.Detail.Tenant, resp.Type.String()).Inc()

			level.Info(log.Logger).Log("msg", "assigned job to worker", "job_id", j.ID, "worker", req.WorkerId)

			return resp, nil
		case <-timeoutCtx.Done():
			span.SetAttributes(attribute.Int("job_q_depth", len(s.mergedJobs)))
			metricJobsNotFound.WithLabelValues(req.WorkerId).Inc()

			return &tempopb.NextJobResponse{}, status.Error(codes.NotFound, ErrNoJobsFound.Error())
		}
	}
}

// UpdateJob implements the BackendSchedulerServer interface
func (s *BackendScheduler) UpdateJob(ctx context.Context, req *tempopb.UpdateJobStatusRequest) (*tempopb.UpdateJobStatusResponse, error) {
	ctx, span := tracer.Start(ctx, "UpdateJob")
	defer span.End()

	j := s.work.GetJob(req.JobId)
	if j == nil {
		return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.NotFound, work.ErrJobNotFound.Error())
	}

	metricJobDuration.WithLabelValues(j.GetType().String()).Observe(time.Since(j.GetCreatedTime()).Seconds())

	switch req.Status {
	case tempopb.JobStatus_JOB_STATUS_RUNNING:
	case tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
		s.work.CompleteJob(req.JobId)
		metricJobsCompleted.WithLabelValues(j.JobDetail.Tenant, j.GetType().String()).Inc()
		metricJobsActive.WithLabelValues(j.JobDetail.Tenant, j.GetType().String()).Dec()
		level.Info(log.Logger).Log("msg", "job completed", "job_id", req.JobId)

		switch j.GetType() {
		case tempopb.JobType_JOB_TYPE_COMPACTION:
			if req.Compaction != nil && req.Compaction.Output != nil {
				s.work.SetJobCompactionOutput(req.JobId, req.Compaction.Output)
			}
		case tempopb.JobType_JOB_TYPE_REDACTION:
			if req.Redaction != nil {
				if j.JobDetail.GetRedaction().GetVerify() {
					// A verify job's mode arrives as the batch's APPLY (Next() overwrites it), so
					// reporting it under that label would credit verification scans to the apply
					// counter and inflate what the redaction claims to have removed.
					recordRedactionVerifyResult(j.Tenant(), req.Redaction.TracesFound)
					if req.Redaction.TracesFound > 0 {
						s.enqueueRedactionForVerifiedBlock(ctx, j.Tenant(),
							j.JobDetail.GetBatchId(), j.JobDetail.GetRedaction().GetBlockId(),
							j.JobDetail.GetRedaction().GetStartTimeUnixNano(),
							j.JobDetail.GetRedaction().GetEndTimeUnixNano())
					}
				} else {
					recordRedactionResult(j.Tenant(), j.JobDetail.GetRedaction().GetMode(), req.Redaction.TracesFound)
				}
				level.Info(log.Logger).Log("msg", "redaction job result",
					"job_id", req.JobId,
					"tenant", j.Tenant(),
					"block_id", j.JobDetail.GetRedaction().GetBlockId(),
					"block_rewrote", req.Redaction.TracesFound > 0,
					"traces_found", req.Redaction.TracesFound)
			}
			s.cleanupBatchIfDone(ctx, j.Tenant())
		}

		err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, []string{req.JobId})
		if err != nil {
			// Fail without returning the job if we can't update the job cache.
			return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.Internal, ErrFlushFailed.Error())
		}

		err = s.applyJobsToBlocklist(ctx, j.Tenant(), []*work.Job{j})
		if err != nil {
			return &tempopb.UpdateJobStatusResponse{}, status.Error(codes.Internal, err.Error())
		}
	case tempopb.JobStatus_JOB_STATUS_FAILED:
		s.work.FailJob(req.JobId)
		metricJobsFailed.WithLabelValues(j.Tenant(), j.GetType().String()).Inc()
		metricJobsActive.WithLabelValues(j.Tenant(), j.GetType().String()).Dec()
		level.Error(log.Logger).Log("msg", "job failed", "job_id", req.JobId, "error", req.Error)

		err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, []string{req.JobId})
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

// SubmitRedaction implements the BackendSchedulerServer interface. The tenant is sourced
// exclusively from the authenticated request context (X-Scope-OrgID header); any tenant_id
// field on the request body is ignored. This prevents a cross-tenant escalation where an
// authenticated caller could supply a different tenant in the body and trigger redaction
// against that tenant's blocks. The method snapshots the tenant's block list and enqueues
// one pending job per block. Trace IDs are stored in a shared batch manifest rather than
// in each job to avoid copying the list across potentially millions of pending jobs.
func (s *BackendScheduler) SubmitRedaction(ctx context.Context, req *tempopb.SubmitRedactionRequest) (*tempopb.SubmitRedactionResponse, error) {
	_, span := tracer.Start(ctx, "SubmitRedaction")
	defer span.End()

	tenant, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		if errors.Is(err, user.ErrNoOrgID) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	querySel := req.GetQuery()
	if err := validateRedactionRequest(req, querySel); err != nil {
		return nil, err
	}

	if s.overrides.CompactionDisabled(tenant) {
		return nil, status.Error(codes.FailedPrecondition, "compaction is disabled for this tenant")
	}

	// One batch per tenant, any mode. GetBatch is the mode-agnostic existence check; a dry-run
	// does not make TenantPending true, so guarding on TenantPending here would wrongly admit a
	// second submission over a running dry-run.
	if s.work.GetBatch(tenant) != nil {
		return nil, status.Error(codes.AlreadyExists, "a redaction is already in progress for this tenant")
	}

	batchID := uuid.New().String()
	span.SetAttributes(
		attribute.String("tenant", tenant),
		attribute.String("batch_id", batchID),
		attribute.Int("trace_count", len(req.TraceIds)),
	)

	// Snapshot the block list for this tenant. One pending job is created per block;
	// the worker checks whether the block actually contains any of the trace IDs.
	metas := s.store.BlockMetas(tenant)
	if len(metas) == 0 {
		return nil, status.Error(codes.NotFound, "no blocks found for tenant")
	}

	// Build a map of block ID -> job ID for all blocks currently referenced by
	// any job. Blocks in active compaction may disappear before a redaction worker
	// can process them — their contents will be merged into a new output block not
	// yet covered by any pending redaction job. We record the job IDs so the
	// maintenance loop can look up output blocks once compaction finishes.
	// Since GetBatch returned nil above (no batch of any mode), there are no active redaction
	// jobs for this tenant, so busy blocks are exclusively from other providers.
	busyBlocks := s.work.BusyBlocksForTenant(tenant)

	// The block count at the instant of selection. Held in its own variable because `metas` is
	// reassigned to the filtered set below, and because it is the denominator every count in the audit
	// record is relative to: without it, jobs_created cannot be reconciled against anything. It will not
	// match tempodb_blocklist_length, which is only written at poll completion while compaction mutates
	// the live list in between.
	totalBlocks := len(metas)

	skippedJobSet := make(map[string]struct{})
	filtered := metas[:0:0]
	// Counted separately from the busy-block skip so the two reasons are never conflated in the logs.
	var outOfWindowBlocks, indeterminateBlocks int
	for _, meta := range metas {
		// Skip blocks whose data range falls outside the requested window (no job created). Keyed
		// on the block's real StartTime/EndTime — see blockOverlapsWindow on why not CompactedTime.
		overlaps, indeterminate := blockOverlapsWindow(meta, req.StartTimeUnixNano, req.EndTimeUnixNano)
		if !overlaps {
			outOfWindowBlocks++
			continue
		}
		if jobID, busy := busyBlocks[meta.BlockID.String()]; busy {
			skippedJobSet[jobID] = struct{}{}
			continue
		}
		// Counted here rather than at the overlap test so it means "enqueued on trust", matching the
		// warning below. An indeterminate block that is also mid-compaction is deferred, not included,
		// and is already reported by the busy-block warning.
		if indeterminate {
			indeterminateBlocks++
		}
		filtered = append(filtered, meta)
	}
	// Blocks held by an active compaction are a coverage gap the operator may need to act on, so they
	// warn. Out-of-window blocks are the feature working as asked, so they are counted separately.
	busySkippedBlocks := totalBlocks - len(filtered) - outOfWindowBlocks
	if busySkippedBlocks > 0 {
		level.Warn(log.Logger).Log("msg", "skipping blocks in active compaction jobs during redaction submission",
			"tenant", tenant,
			"skipped_blocks", busySkippedBlocks,
			"skipped_compaction_jobs", len(skippedJobSet),
			"total_blocks", totalBlocks)
	}
	if indeterminateBlocks > 0 {
		// Included, not skipped: their range could not be judged, so they were taken on trust.
		level.Warn(log.Logger).Log("msg", "redaction included blocks with an unusable data range",
			"tenant", tenant, "blocks", indeterminateBlocks, "total_blocks", totalBlocks)
	}
	metas = filtered

	// An empty selection with nothing deferred means the window matched no data — almost always a mistyped
	// bound. Refuse rather than create a batch: an empty batch reports jobs_created:0 as success, holds
	// compaction off for a quiescence cycle, and rejects the corrected resubmission with AlreadyExists.
	//
	// Deferred blocks are the exception, because an apply-mode batch must survive for its rescan. A
	// dry-run arms no rescan, so it gets no exemption — and its own code, since the blocks did overlap and
	// "no blocks overlap" would send the operator to widen the window instead of retrying.
	if len(metas) == 0 && len(skippedJobSet) > 0 && req.Mode.IsDryRun() {
		level.Info(log.Logger).Log("msg", "dry-run redaction deferred every in-window block to compaction; refusing rather than creating a batch that never scans",
			"tenant", tenant, "busy_blocks", busySkippedBlocks, "compaction_jobs", len(skippedJobSet))
		return nil, status.Errorf(codes.FailedPrecondition,
			"all %d in-window blocks are being compacted; retry the dry-run shortly", busySkippedBlocks)
	}
	if len(metas) == 0 && len(skippedJobSet) == 0 {
		level.Info(log.Logger).Log("msg", "redaction submission matched no blocks in the requested window",
			"tenant", tenant, "out_of_window_blocks", outOfWindowBlocks,
			"total_blocks", totalBlocks)
		return nil, status.Error(codes.NotFound, "no blocks overlap the requested time window")
	}

	// Record what will actually be touched: the requested window can extend well past the data, so the
	// covered range is the honest blast radius.
	coveredStart, coveredEnd, haveStart, haveEnd := coveredRange(metas)
	span.SetAttributes(
		attribute.Int64("window_start_unix_nano", req.StartTimeUnixNano),
		attribute.Int64("window_end_unix_nano", req.EndTimeUnixNano),
		attribute.String("covered_start", coveredRangeLabel(coveredStart, haveStart)),
		attribute.String("covered_end", coveredRangeLabel(coveredEnd, haveEnd)),
		attribute.Int("total_blocks", totalBlocks),
		attribute.Int("out_of_window_blocks", outOfWindowBlocks),
		attribute.Int("indeterminate_blocks", indeterminateBlocks),
	)

	jobs := make([]*work.Job, 0, len(metas))
	for _, meta := range metas {
		jobs = append(jobs, &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_REDACTION,
			JobDetail: tempopb.JobDetail{
				Tenant:  tenant,
				BatchId: batchID,
				Redaction: &tempopb.RedactionDetail{
					BlockId: meta.BlockID.String(),
					// TraceIds intentionally empty here — populated from batch in Next().
				},
			},
		})
	}

	batch := &tempopb.RedactionBatch{
		BatchId:           batchID,
		TenantId:          tenant,
		TraceIds:          req.TraceIds,
		Query:             querySel,
		Mode:              req.Mode,
		StartTimeUnixNano: req.StartTimeUnixNano,
		EndTimeUnixNano:   req.EndTimeUnixNano,
		CreatedAtUnixNano: time.Now().UnixNano(),
	}
	// Only apply-mode batches arm a rescan. A dry-run rewrites nothing, so there is no output
	// block to re-cover once a skipped compaction finishes; a rescan would only re-count and
	// inflate the dry-run metric. Blocks busy at submission simply go uncounted (a minor,
	// documented preview undercount).
	if len(skippedJobSet) > 0 && !req.Mode.IsDryRun() {
		skippedJobIDs := make([]string, 0, len(skippedJobSet))
		for id := range skippedJobSet {
			skippedJobIDs = append(skippedJobIDs, id)
		}
		batch.SkippedCompactionJobIds = skippedJobIDs
		batch.RescanAfterUnixNano = time.Now().Add(s.cfg.ProviderConfig.Redaction.RescanDelay).UnixNano()
	}

	// Store batch first, then jobs. On job failure, roll back the batch so the
	// tenant is not permanently locked out of future submissions.
	if err := s.work.AddBatch(batch); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.work.AddPendingJobs(jobs); err != nil {
		s.work.RemoveBatch(tenant)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Persist batch manifest and affected shards. Both are best-effort here;
	// the data is safely in memory and will be flushed again on shutdown.
	if err := s.work.FlushBatchesToLocal(ctx, s.cfg.LocalWorkPath); err != nil {
		level.Warn(log.Logger).Log("msg", "failed to flush batch manifest", "err", err)
	}
	affectedIDs := make([]string, len(jobs))
	for i, j := range jobs {
		affectedIDs[i] = j.ID
	}
	if err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, affectedIDs); err != nil {
		level.Warn(log.Logger).Log("msg", "failed to flush job shards", "err", err)
	}

	level.Info(log.Logger).Log("msg", "redaction batch submitted",
		"tenant", tenant,
		"batch_id", batchID,
		"jobs_created", len(jobs),
		"total_blocks", totalBlocks,
		"blocks_skipped_compacting", busySkippedBlocks,
		"blocks_out_of_window", outOfWindowBlocks,
		"covered_start", coveredRangeLabel(coveredStart, haveStart),
		"covered_end", coveredRangeLabel(coveredEnd, haveEnd),
		"trace_count", len(req.TraceIds))

	return &tempopb.SubmitRedactionResponse{
		BatchId:     batchID,
		JobsCreated: int32(len(jobs)),
	}, nil
}

// cleanupOrphanedBatches sweeps all active batches once per maintenance tick and advances
// each batch's quiescence: a completed batch enters quiescence, a quiescing batch counts down
// (and is removed at zero), and a re-activated batch leaves quiescence. Called after each Prune
// tick because Prune transitions timed-out running jobs to FAILED by calling j.Fail() directly,
// bypassing the UpdateJob path.
func (s *BackendScheduler) cleanupOrphanedBatches(ctx context.Context) {
	changed := false
	for _, batch := range s.work.ListBatches() {
		// batch.TenantId is immutable; advanceQuiescence reads the mutable fields under lock.
		if s.advanceQuiescence(ctx, batch.TenantId) {
			changed = true
		}
	}
	// The manifest is a single global file; flush once per tick if anything changed rather
	// than once per mutated batch.
	if changed {
		s.flushBatches(ctx)
	}
}

// quiescenceSweeps is how many maintenance sweeps' worth of time a completed redaction batch is
// held before removal. Entry records a deadline of now + quiescenceSweeps × MaintenanceInterval;
// the batch is removed on the first tick at or after it. Holding the batch keeps the tenant's
// compaction disabled (TenantPending stays true) long enough for the rescan to catch any block a
// compaction produced just as the last redaction job finished, closing the cleanup-window race.
// A deadline (vs a decrementing counter) is static — stable across pod restarts / work reloads —
// and lets the batch stay unwritten between entry and removal.
const quiescenceSweeps = 2

// redactionBatchActive reports whether a redaction batch still has outstanding work -- jobs in
// any state (pending/in-flight/active) or a pending rescan. A batch that is not active is a
// candidate for quiescence. Both the job-completion path (cleanupBatchIfDone) and the tick path
// (advanceQuiescence) share this one predicate so their notions of "done" cannot drift.
// rescanPending is supplied by the caller from a locked batch-store snapshot; batch fields are
// never read off a live pointer without the store lock (data race).
func (s *BackendScheduler) redactionBatchActive(tenantID string, rescanPending bool) bool {
	return s.work.HasJobsForTenant(tenantID, tempopb.JobType_JOB_TYPE_REDACTION) || rescanPending
}

// cleanupBatchIfDone is called when a batch's redaction jobs may have all finished (e.g. on job
// completion via UpdateJob). Rather than removing the batch immediately -- which would re-enable
// compaction before the rescan can cover a late compaction output -- it enters quiescence by
// recording a quiesce-until deadline; a later maintenance tick (advanceQuiescence) removes it.
//
// "In-flight" jobs (popped from the pending queue, travelling the provider channel, not yet
// promoted to the active map) still count via redactionBatchActive, so the batch is not treated
// as done while any are outstanding.
func (s *BackendScheduler) cleanupBatchIfDone(ctx context.Context, tenantID string) {
	quiesceUntil, rescanPending, dryRun, ok := s.work.BatchQuiescenceState(tenantID)
	if !ok || s.redactionBatchActive(tenantID, rescanPending) {
		return
	}
	if dryRun {
		// A dry-run writes nothing and never disabled compaction, so there is no cleanup-window
		// race to hold open: remove the batch at once and free the tenant's slot.
		s.work.RemoveBatch(tenantID)
		s.flushBatches(ctx)
		level.Info(log.Logger).Log("msg", "dry-run redaction batch complete, removed", "tenant", tenantID)
		return
	}
	if quiesceUntil == 0 {
		s.settleDrainedBatch(ctx, tenantID)
		s.flushBatches(ctx)
	}
}

// settleDrainedBatch decides what becomes of a batch whose jobs have drained: verify it, or enter
// quiescence if verification has nothing left to do.
//
// Both the job-completion path (cleanupBatchIfDone) and the maintenance tick (advanceQuiescence)
// route through here, for the same reason they share redactionBatchActive: if the two decide
// independently they drift. They already did -- quiescence entered from the completion path skipped
// verification entirely, so it only ever ran for a batch that drained without a completion
// callback, which is the timeout case rather than the normal one.
func (s *BackendScheduler) settleDrainedBatch(ctx context.Context, tenantID string) {
	if s.runVerification(ctx, tenantID) {
		return
	}
	s.enterQuiescence(tenantID)
}

// enterQuiescence records the quiesce-until deadline (now + quiescenceSweeps × MaintenanceInterval)
// once, leaving compaction disabled until then. It mutates in-memory state only; the caller flushes.
func (s *BackendScheduler) enterQuiescence(tenantID string) {
	until := time.Now().Add(quiescenceSweeps * s.cfg.MaintenanceInterval)
	s.work.SetBatchQuiesceUntil(tenantID, until.UnixNano())
	level.Info(log.Logger).Log("msg", "redaction batch complete, entering quiescence", "tenant", tenantID, "quiesce_until", until.Format(time.RFC3339))
}

// advanceQuiescence advances one batch's quiescence on a maintenance tick: a batch re-activated
// by new jobs or a pending rescan leaves quiescence; a done batch not yet quiescing enters it; a
// quiescing batch is removed once its deadline passes. Between entry and the deadline it makes no
// change (returns false), so the manifest is not rewritten on every tick. The caller flushes once
// per tick if anything changed.
func (s *BackendScheduler) advanceQuiescence(ctx context.Context, tenantID string) (changed bool) {
	quiesceUntil, rescanPending, dryRun, ok := s.work.BatchQuiescenceState(tenantID)
	if !ok {
		return false
	}
	if s.redactionBatchActive(tenantID, rescanPending) {
		if quiesceUntil != 0 {
			s.work.SetBatchQuiesceUntil(tenantID, 0)
			return true
		}
		return false
	}
	if dryRun {
		// Dry-run batches do not quiesce (nothing was written); remove on the tick that finds
		// them done. Normally cleanupBatchIfDone already handled this on job completion; this
		// covers a batch that went done without a completion callback (e.g. all jobs dropped).
		s.work.RemoveBatch(tenantID)
		level.Info(log.Logger).Log("msg", "dry-run redaction batch complete, removed", "tenant", tenantID)
		return true
	}
	if quiesceUntil == 0 {
		// Verify before quiescing, not after: quiescence is the window that keeps compaction off so
		// the batch can still act, and a pass that finds a gap needs to enqueue work rather than
		// discover it once teardown has already re-enabled compaction.
		s.settleDrainedBatch(ctx, tenantID)
		return true
	}
	if time.Now().UnixNano() < quiesceUntil {
		// Still within the quiescence window; nothing to change this tick.
		return false
	}
	s.work.RemoveBatch(tenantID)
	level.Info(log.Logger).Log("msg", "redaction batch quiescence complete, manifest removed", "tenant", tenantID)
	return true
}

// flushBatches persists the (single, global) batch manifest, best-effort.
func (s *BackendScheduler) flushBatches(ctx context.Context) {
	if err := s.work.FlushBatchesToLocal(ctx, s.cfg.LocalWorkPath); err != nil {
		level.Warn(log.Logger).Log("msg", "failed to flush batch manifest", "err", err)
	}
}

// checkPendingRescans is called on each maintenance tick. It looks for batches whose
// rescan window has elapsed, looks up the output blocks from the skipped compaction
// jobs, and enqueues new pending redaction jobs for those blocks.
func (s *BackendScheduler) checkPendingRescans(ctx context.Context) {
	now := time.Now().UnixNano()
	for _, batch := range s.work.ListBatches() {
		if batch.RescanAfterUnixNano == 0 || now < batch.RescanAfterUnixNano {
			continue
		}
		s.performRescan(ctx, batch)
	}
}

// performRescan handles one batch that is ready for a rescan.
//
// It loops over the skipped compaction job IDs, classifying each job as still-running
// (output not yet available) or complete (output blocks known). For complete jobs it
// further classifies output blocks as ready (enqueue a redaction job) or still being
// compacted (discover the covering job and advance to that inline).
//
// The loop resolves cascaded compaction chains in a single maintenance tick. It only
// bails out and re-arms when it encounters a still-running job whose output is not yet
// available. MaxRescanGenerations bounds the depth to prevent infinite chains.
func (s *BackendScheduler) performRescan(ctx context.Context, batch *tempopb.RedactionBatch) {
	tenantID := batch.TenantId
	batchID := batch.BatchId
	maxRetries := s.cfg.ProviderConfig.Redaction.MaxRescanGenerations

	currentJobIDs := batch.SkippedCompactionJobIds
	var allReadyJobs []*work.Job
	var rearmIDs []string // non-empty → re-arm batch for next tick
	resolved := false     // true on clean exit (done or scheduled for re-arm)

	// Validated on both paths into the store (SubmitRedaction, dropBatchesWithUnusableWindows).
	rescanWindow := tempodb.RedactionWindow{StartNano: batch.StartTimeUnixNano, EndNano: batch.EndTimeUnixNano}

	// Output blocks are discovered by ID, so re-applying the window means looking their metas up. Snapshot
	// once: BlockMetas returns a private slice copy and metas are not mutated after publication, so the
	// pointers cannot be read torn even though the poller mutates the blocklist concurrently. A block
	// missing from the snapshot resolves to unknown and is enqueued, so staleness can only over-include.
	// Skipped entirely for an unwindowed batch, the common case.
	var metaByID map[string]*backend.BlockMeta
	if !rescanWindow.IsZero() {
		metas := s.store.BlockMetas(tenantID)
		metaByID = make(map[string]*backend.BlockMeta, len(metas))
		for _, m := range metas {
			metaByID[m.BlockID.String()] = m
		}
	}

	for range maxRetries {
		// skippedSet doubles as a "not-yet-found" tracker: delete each entry on match;
		// any IDs remaining after the loop were never found (pruned/failed externally).
		skippedSet := make(map[string]struct{}, len(currentJobIDs))
		for _, id := range currentJobIDs {
			skippedSet[id] = struct{}{}
		}

		var stillRunningIDs, outputBlockIDs []string
		for _, j := range s.work.ListJobs() {
			if _, ok := skippedSet[j.ID]; !ok {
				continue
			}
			delete(skippedSet, j.ID)
			if !j.IsComplete() {
				stillRunningIDs = append(stillRunningIDs, j.ID)
			} else {
				outputBlockIDs = append(outputBlockIDs, j.GetCompactionOutput()...)
			}
		}
		for id := range skippedSet {
			level.Warn(log.Logger).Log(
				"msg", "redaction rescan: skipped compaction job not found; it may have been pruned or failed externally -- operator should resubmit if traces are still present",
				"tenant", tenantID, "batch_id", batchID, "job_id", id,
			)
		}

		// Classify output blocks: ready to redact vs still being compacted.
		busyBlocks := s.work.BusyBlocksForTenant(tenantID)
		var nextJobIDs []string
		for _, blockID := range outputBlockIDs {
			if jobID, busy := busyBlocks[blockID]; busy {
				nextJobIDs = append(nextJobIDs, jobID)
				continue
			}

			// Skip an output block lying wholly outside the window. This is an I/O saving, not a safety
			// property: an output merging an in-window input still overlaps, and the per-block scan bound
			// is what actually limits deletion. An unknown block is treated as in scope.
			if m, known := metaByID[blockID]; known {
				if overlaps, _ := blockOverlapsWindow(m, rescanWindow.StartNano, rescanWindow.EndNano); !overlaps {
					level.Debug(log.Logger).Log("msg", "redaction rescan: output block outside the window; not enqueued",
						"tenant", tenantID, "batch_id", batchID, "block_id", blockID)
					continue
				}
			}

			allReadyJobs = append(allReadyJobs, &work.Job{
				ID:   uuid.New().String(),
				Type: tempopb.JobType_JOB_TYPE_REDACTION,
				JobDetail: tempopb.JobDetail{
					Tenant:  tenantID,
					BatchId: batchID,
					Redaction: &tempopb.RedactionDetail{
						BlockId: blockID,
					},
				},
			})
		}

		level.Info(log.Logger).Log(
			"msg", "redaction rescan: iteration",
			"tenant", tenantID, "batch_id", batchID,
			"still_running", len(stillRunningIDs), "output_blocks", len(outputBlockIDs),
			"ready_jobs", len(allReadyJobs), "next_job_ids", len(nextJobIDs),
		)

		if len(stillRunningIDs) > 0 {
			// Can't proceed inline: wait for next tick.
			rearmIDs = append(rearmIDs, stillRunningIDs...)
			rearmIDs = append(rearmIDs, nextJobIDs...)
			resolved = true
			break
		}

		if len(nextJobIDs) == 0 {
			// All complete, no further compaction chains: nothing more to do.
			resolved = true
			break
		}

		// All current jobs are complete but their output is still being compacted:
		// advance to the next generation inline.
		currentJobIDs = nextJobIDs
	}

	if !resolved {
		level.Warn(log.Logger).Log(
			"msg", "redaction rescan: cascaded compaction depth exceeded -- operator should resubmit if traces are still present",
			"tenant", tenantID, "batch_id", batchID, "max_retries", maxRetries,
		)
	}

	// Commit rescan state update under the batch store's write lock.
	var rescanAfterNano int64
	if len(rearmIDs) > 0 {
		rescanAfterNano = time.Now().Add(s.cfg.ProviderConfig.Redaction.RescanDelay).UnixNano()
	}
	s.work.SetBatchRescan(tenantID, rearmIDs, rescanAfterNano)

	if len(allReadyJobs) == 0 && len(rearmIDs) == 0 {
		s.cleanupBatchIfDone(ctx, tenantID)
		return
	}

	if len(allReadyJobs) > 0 {
		if err := s.work.AddPendingJobs(allReadyJobs); err != nil {
			level.Error(log.Logger).Log("msg", "redaction rescan: failed to add pending jobs", "tenant", tenantID, "err", err)
			return
		}
	}

	if err := s.work.FlushBatchesToLocal(ctx, s.cfg.LocalWorkPath); err != nil {
		level.Warn(log.Logger).Log("msg", "redaction rescan: failed to flush batch manifest", "tenant", tenantID, "err", err)
	}
	if len(allReadyJobs) > 0 {
		affectedIDs := make([]string, len(allReadyJobs))
		for i, j := range allReadyJobs {
			affectedIDs[i] = j.ID
		}
		if err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, affectedIDs); err != nil {
			level.Warn(log.Logger).Log("msg", "redaction rescan: failed to flush job shards", "tenant", tenantID, "err", err)
		}
	}
}

func (s *BackendScheduler) replayWorkOnBlocklist(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "replayWorkOnBlocklist")
	defer span.End()

	var (
		err           error
		tenant        string
		jobStatus     tempopb.JobStatus
		perTenantJobs = make(map[string][]*work.Job)
	)

	// Get all the input blocks which have been successfully compacted
	for _, j := range s.work.ListJobs() {
		tenant = j.Tenant()
		jobStatus = j.GetStatus()

		// count the active jobs and update the metric
		if jobStatus == tempopb.JobStatus_JOB_STATUS_RUNNING {
			metricJobsActive.WithLabelValues(tenant, j.GetType().String()).Inc()
		}

		if jobStatus != tempopb.JobStatus_JOB_STATUS_SUCCEEDED {
			continue
		}

		if _, ok := perTenantJobs[tenant]; !ok {
			perTenantJobs[tenant] = make([]*work.Job, 0, 1000)
		}

		perTenantJobs[tenant] = append(perTenantJobs[tenant], j)
	}

	for tenant, jobs := range perTenantJobs {
		err = s.applyJobsToBlocklist(ctx, tenant, jobs)
		if err != nil {
			return fmt.Errorf("failed to load blocklist jobs for tenant %s: %w", tenant, err)
		}
	}

	return nil
}

// applyJobsToBlocklist processes the jobs and applies their results to the in-memory blocklist.
func (s *BackendScheduler) applyJobsToBlocklist(ctx context.Context, tenant string, jobs []*work.Job) error {
	_, span := tracer.Start(ctx, "loadBlocklistJobsForTenant")
	defer span.End()

	var (
		metas     = s.store.BlockMetas(tenant)
		oldBlocks []*backend.BlockMeta
		u         backend.UUID
		err       error
		m         *backend.BlockMeta
		ok        bool
	)

	span.SetAttributes(
		attribute.String("tenant", tenant),
		attribute.Int("job_count", len(jobs)),
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

			if m, ok = foundMetaInMetas(metas, u); ok {
				oldBlocks = append(oldBlocks, m)
			}
		}
	}

	err = s.store.MarkBlocklistCompacted(tenant, oldBlocks, nil)
	if err != nil {
		return fmt.Errorf("failed to mark compacted blocks on in-memory blocklist: %w", err)
	}

	return nil
}

func foundMetaInMetas(metas []*backend.BlockMeta, u backend.UUID) (*backend.BlockMeta, bool) {
	for _, m := range metas {
		if m.BlockID == u {
			return m, true
		}
	}
	return nil, false
}

func (s *BackendScheduler) StatusHandler(w http.ResponseWriter, _ *http.Request) {
	// Active jobs table
	active := table.NewWriter()
	active.SetTitle("Active Jobs")
	active.AppendHeader(table.Row{"tenant", "job_id", "type", "status", "worker", "created", "start", "end"})

	jobs := s.work.ListJobs()
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].GetCreatedTime().After(jobs[j].GetCreatedTime())
	})
	for _, j := range jobs {
		active.AppendRow(table.Row{
			j.Tenant(),
			j.GetID(),
			j.GetType().String(),
			j.GetStatus().String(),
			j.GetWorkerID(),
			j.GetCreatedTime().Format(time.RFC3339),
			j.GetStartTime().Format(time.RFC3339),
			j.GetEndTime().Format(time.RFC3339),
		})
	}

	// Pending jobs table (redaction queue)
	pending := table.NewWriter()
	pending.SetTitle("Pending Jobs")
	pending.AppendHeader(table.Row{"tenant", "job_id", "type", "block_id", "batch_id"})

	for _, j := range s.work.ListAllPendingJobs() {
		blockID := ""
		if j.JobDetail.Redaction != nil {
			blockID = j.JobDetail.Redaction.BlockId
		}
		pending.AppendRow(table.Row{
			j.Tenant(),
			j.GetID(),
			j.GetType().String(),
			blockID,
			j.JobDetail.BatchId,
		})
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, active.Render())
	_, _ = io.WriteString(w, "\n\n")
	_, _ = io.WriteString(w, pending.Render())
}
