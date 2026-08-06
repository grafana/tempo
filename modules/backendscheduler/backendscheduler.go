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

	var err error

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-maintenanceTicker.C:
			s.work.Prune(ctx)
			s.checkPendingRescans(ctx)
			s.cleanupOrphanedBatches(ctx)
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
					// The batch carries the selector this job needs, and is resolved by tenant. Drop the
					// job unless that batch is still the one it was created for and still wants the work.
					// Dropping happens before any work begins, so no partial output exists and the 1:1
					// block in:out invariant holds; jobs already running on a worker still finish.
					batch := s.work.GetBatch(j.Tenant())
					dropReason := ""
					switch {
					case batch == nil:
						dropReason = "batch_missing"
					case batch.Cancelled:
						// A cancel's purge only reaches jobs still in the pending queue, so a job dequeued
						// just before the cancel would otherwise be assigned and redact a block the
						// operator asked to stop.
						dropReason = "batch_cancelled"
					case j.JobDetail.BatchId != "" && j.JobDetail.BatchId != batch.BatchId:
						// A job left over from an earlier batch, still travelling the provider channel when
						// that batch was removed and a new one submitted. Without this check it would be
						// handed the new batch's selector and mode, rewriting its block under a selector it
						// was never scheduled for — possibly in apply mode when it belonged to a dry run.
						dropReason = "batch_superseded"
					}
					if dropReason != "" {
						level.Debug(log.Logger).Log("msg", "dropping redaction job",
							"job_id", j.ID, "tenant", j.Tenant(), "reason", dropReason)
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
						j.JobDetail.Redaction.Mode = batch.Mode
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
				recordRedactionResult(j.Tenant(), j.JobDetail.GetRedaction().GetMode(), req.Redaction.TracesFound)
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
	// Exactly one selector: an explicit trace ID list or a TraceQL query, never both.
	// (The proto reserves a single-member oneof for query; XOR is enforced here until
	// trace_ids is migrated into the oneof.)
	querySel := req.GetQuery()
	hasIDs := len(req.TraceIds) > 0
	hasQuery := querySel.GetQuery() != "" // nil-safe
	switch {
	case hasIDs && hasQuery:
		return nil, status.Error(codes.InvalidArgument, "trace_ids and query are mutually exclusive")
	case !hasIDs && !hasQuery:
		return nil, status.Error(codes.InvalidArgument, "one of trace_ids or query must be set")
	case hasQuery:
		if err := validateRedactionQuery(querySel.Query); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	// Reject unknown modes rather than defaulting them to APPLY: since only DRY_RUN is
	// checked downstream, an unrecognized value would otherwise fall through to a
	// destructive rewrite. Fail closed.
	switch req.Mode {
	case tempopb.RedactionMode_REDACTION_MODE_APPLY, tempopb.RedactionMode_REDACTION_MODE_DRY_RUN:
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("unknown redaction mode %d", int32(req.Mode)))
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

	skippedJobSet := make(map[string]struct{})
	filtered := metas[:0:0]
	for _, meta := range metas {
		if jobID, busy := busyBlocks[meta.BlockID.String()]; busy {
			skippedJobSet[jobID] = struct{}{}
			continue
		}
		filtered = append(filtered, meta)
	}
	skippedBlocks := len(metas) - len(filtered)
	if skippedBlocks > 0 {
		level.Warn(log.Logger).Log("msg", "skipping blocks in active compaction jobs during redaction submission",
			"tenant", tenant,
			"skipped_blocks", skippedBlocks,
			"skipped_compaction_jobs", len(skippedJobSet),
			"total_blocks", len(metas))
	}
	metas = filtered

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
		"blocks_skipped_compacting", skippedBlocks,
		"trace_count", len(req.TraceIds))

	return &tempopb.SubmitRedactionResponse{
		BatchId:     batchID,
		JobsCreated: int32(len(jobs)),
	}, nil
}

// CancelRedaction cancels the authenticated tenant's in-progress redaction. It marks the batch
// cancelled, clears any armed rescan, and purges the not-yet-started pending jobs so the backlog
// stops. Jobs already dispatched finish on their own — cancel never interrupts a block rewrite, so
// block in:out stays 1:1 — after which the batch is removed without quiescence and compaction
// resumes. The tenant is read from the request header, never a body field, so cancel cannot cross
// tenants and concurrent per-tenant redactions stay isolated.
func (s *BackendScheduler) CancelRedaction(ctx context.Context, _ *tempopb.CancelRedactionRequest) (*tempopb.CancelRedactionResponse, error) {
	ctx, span := tracer.Start(ctx, "CancelRedaction")
	defer span.End()

	tenant, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		if errors.Is(err, user.ErrNoOrgID) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	span.SetAttributes(attribute.String("tenant", tenant))

	batch := s.work.GetBatch(tenant)
	if batch == nil {
		return nil, status.Error(codes.NotFound, "no redaction in progress for this tenant")
	}
	batchID := batch.BatchId

	// Commit the cancel before publishing it. Persisting does not touch the in-memory batch, so if it
	// fails nothing has changed and nothing has seen the cancel: the RPC reports failure honestly and
	// the operator can retry. Publishing first would be unsafe, because the readers of the flag act
	// irreversibly — Next() discards a dequeued job and checkPendingRescans clears the skipped-block
	// list — so a subsequent write failure could not be undone, and we would report a failed cancel
	// after work had already been dropped.
	if err := s.work.PersistBatchCancelled(ctx, tenant, s.cfg.LocalWorkPath); err != nil {
		level.Error(log.Logger).Log("msg", "failed to persist redaction cancel; nothing changed, safe to retry", "tenant", tenant, "err", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to persist redaction cancel; nothing changed, retry: %v", err))
	}
	if alreadyCancelled := s.work.SetBatchCancelled(tenant, true); alreadyCancelled {
		level.Info(log.Logger).Log("msg", "redaction already cancelled; completing idempotently", "tenant", tenant, "batch_id", batchID)
	}

	// The cancel is durable and published, so it is now forward-only: there is no state left that a
	// failure below could roll back to, and none is needed. Everything from here self-heals from the
	// durable flag if it is interrupted — pending jobs that survive on disk are dropped at assignment
	// (Next), and an armed rescan that survives is cleared by checkPendingRescans — so these writes are
	// latency optimizations, not correctness gates. Failing the RPC for them would tell the operator to
	// retry something already in effect.
	s.work.SetBatchRescan(tenant, nil, 0)
	purgedIDs := s.work.PurgePendingRedactionJobs(tenant)
	// Flush unconditionally, even when this call purged nothing: a retry finds the jobs already gone
	// from memory and so gets no IDs back, while they are still on disk. With no IDs to target this
	// flushes every shard, persisting whatever an earlier attempt removed.
	if err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, purgedIDs); err != nil {
		level.Warn(log.Logger).Log("msg", "redaction cancel purge not yet persisted; jobs will be dropped at assignment and the purge re-persisted on a later flush", "tenant", tenant, "err", err)
	}
	if err := s.work.FlushBatchesToLocal(ctx, s.cfg.LocalWorkPath); err != nil {
		level.Warn(log.Logger).Log("msg", "redaction cancel rescan clear not yet persisted; a stale armed rescan is cleared on a later maintenance tick", "tenant", tenant, "err", err)
	}

	// Start the quiescence countdown if nothing is outstanding: jobs dispatched before the cancel may
	// have rewritten blocks, so the tenant's compaction stays held until the blocklist has polled them.
	s.cleanupBatchIfDone(ctx, tenant)

	span.SetAttributes(
		attribute.String("batch_id", batchID),
		attribute.Int("pending_purged", len(purgedIDs)),
	)
	level.Info(log.Logger).Log("msg", "redaction cancelled",
		"tenant", tenant, "batch_id", batchID, "pending_purged", len(purgedIDs))

	return &tempopb.CancelRedactionResponse{
		BatchId:       batchID,
		PendingPurged: int32(len(purgedIDs)),
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
		if s.advanceQuiescence(batch.TenantId) {
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
	quiesceUntil, rescanPending, dryRun, _, ok := s.work.BatchQuiescenceState(tenantID)
	if !ok || s.redactionBatchActive(tenantID, rescanPending) {
		return
	}
	// A dry-run rewrites nothing, so no output block exists for the blocklist to catch up to and
	// there is no cleanup-window race to hold compaction open for. Remove at once.
	//
	// A cancelled batch does NOT qualify: cancel abandons only work that had not started, while jobs
	// already dispatched finish and rewrite their blocks. Such a batch has produced output blocks the
	// blocklist may not have polled yet, so it quiesces on the normal path below — otherwise
	// compaction could pick up a pre-redaction input block and resurrect the redacted traces.
	if dryRun {
		s.work.RemoveBatch(tenantID)
		s.flushBatches(ctx)
		level.Info(log.Logger).Log("msg", "redaction batch removed without quiescence", "tenant", tenantID, "reason", "dry_run")
		return
	}
	if quiesceUntil == 0 {
		s.enterQuiescence(tenantID)
		s.flushBatches(ctx)
	}
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
func (s *BackendScheduler) advanceQuiescence(tenantID string) (changed bool) {
	quiesceUntil, rescanPending, dryRun, _, ok := s.work.BatchQuiescenceState(tenantID)
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
		// A dry-run does not quiesce (it produced no output block); remove on the tick that finds it
		// drained. cleanupBatchIfDone usually handles this on job completion; this covers a batch
		// that drained without a completion callback. Cancelled batches quiesce like any other
		// apply-mode batch — see cleanupBatchIfDone.
		s.work.RemoveBatch(tenantID)
		level.Info(log.Logger).Log("msg", "redaction batch removed without quiescence", "tenant", tenantID, "reason", "dry_run")
		return true
	}
	if quiesceUntil == 0 {
		s.enterQuiescence(tenantID)
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
	changed := false
	for _, batch := range s.work.ListBatches() {
		if batch.RescanAfterUnixNano == 0 {
			continue
		}
		// A cancelled batch never rescans — it is abandoning the blocks it skipped. Clear the armed
		// rescan instead of running it, whatever its deadline: this check must precede the due-time
		// gate below, because a rescan armed for the future is exactly what a crash between the
		// cancel's two manifest flushes leaves behind. Leaving it armed keeps rescanPending — and so
		// the batch, and so the tenant's compaction block — alive for the rest of the rescan delay
		// with no work outstanding.
		if batch.Cancelled {
			s.work.SetBatchRescan(batch.TenantId, nil, 0)
			changed = true
			continue
		}
		if now < batch.RescanAfterUnixNano {
			continue
		}
		s.performRescan(ctx, batch)
	}
	if changed {
		s.flushBatches(ctx)
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
			} else {
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

	// Commit the rescan state, but only if the batch is still the one this scan started from. Everything
	// above ran off a snapshot and can take a long time, so a cancel may have landed meanwhile — and a
	// cancelled batch must not have its rescan re-armed, nor may a resubmitted batch inherit this scan's
	// result. If the guard rejects, abandon the whole result including any jobs found ready to enqueue:
	// the cancel (or the new batch) decides what happens to those blocks now.
	var rescanAfterNano int64
	if len(rearmIDs) > 0 {
		rescanAfterNano = time.Now().Add(s.cfg.ProviderConfig.Redaction.RescanDelay).UnixNano()
	}
	if !s.work.SetBatchRescanIfCurrent(tenantID, batchID, rearmIDs, rescanAfterNano) {
		level.Info(log.Logger).Log(
			"msg", "redaction rescan abandoned: batch was cancelled or replaced while the rescan ran",
			"tenant", tenantID, "batch_id", batchID, "discarded_ready_jobs", len(allReadyJobs),
		)
		return
	}

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
