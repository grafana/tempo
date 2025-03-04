package backendworker

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	backendscheduler_client "github.com/grafana/tempo/modules/backendscheduler/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	tempo_util "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"go.opentelemetry.io/otel"
)

const (
	reasonCompactorDiscardedSpans = "trace_too_large_to_compact"
)

type BackendWorker struct {
	services.Service

	cfg              Config
	store            storage.Store
	overrides        overrides.Interface
	backendScheduler tempopb.BackendSchedulerClient

	workerID string
}

var tracer = otel.Tracer("modules/backendworker")

func New(cfg Config, schedulerClientCfg backendscheduler_client.Config, store storage.Store, overrides overrides.Interface) (*BackendWorker, error) {
	s := &BackendWorker{
		cfg:       cfg,
		store:     store,
		overrides: overrides,
	}

	workerID, err := os.Hostname()
	s.workerID = workerID

	schedulerClient, err := backendscheduler_client.New(cfg.BackendSchedulerAddr, schedulerClientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend scheduler client: %w", err)
	}
	s.backendScheduler = schedulerClient

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)
	return s, nil
}

func (w *BackendWorker) starting(ctx context.Context) error {
	if w.cfg.Poll {
		w.store.EnablePolling(ctx, w)
	}

	return nil
}

func (w *BackendWorker) running(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	level.Info(log.Logger).Log("msg", "backend scheduler running")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processCompactionJobs(ctx); err != nil {
				// TODO: backoff
				level.Error(log.Logger).Log("msg", "error processing compaction jobs", "err", err)
			}
		}
	}
}

func (w *BackendWorker) processCompactionJobs(ctx context.Context) error {
	// Request next job
	// FIXME: the org ID is not used by the backend scheduler, but it is required by the request.
	resp, err := w.backendScheduler.Next(user.InjectOrgID(ctx, w.workerID), &tempopb.NextJobRequest{
		WorkerId: w.workerID,
		Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
	})
	if err != nil {
		return fmt.Errorf("error getting next job: %w", err)
	}

	if resp == nil || resp.JobId == "" {
		return nil // No jobs available
	}

	if resp.Detail.Tenant == "" {
		return w.failJob(ctx, resp.JobId, "received job with empty tenant")
	}

	level.Debug(log.Logger).Log("msg", "received job", "job_id", resp.JobId, "tenant", resp.Detail.Tenant)

	blockMetas := w.store.BlockMetas(resp.Detail.Tenant)

	// Collect the metas which match the IDs in the job
	var sourceMetas []*backend.BlockMeta
	for _, blockMeta := range blockMetas {
		for _, blockID := range resp.Detail.Compaction.Input {
			if blockMeta.BlockID.String() == blockID {
				sourceMetas = append(sourceMetas, blockMeta)
			}
		}
	}

	// Execute compaction using existing logic
	// err = w.store.Compact(ctx, sourceMetas, resp.Detail.Tenant)
	err = w.compact(ctx, sourceMetas, resp.Detail.Tenant)
	if err != nil {
		return w.failJob(ctx, resp.JobId, fmt.Sprintf("error compacting blocks: %v", err))
	}

	// Mark job as complete
	_, err = w.backendScheduler.UpdateJob(user.InjectOrgID(ctx, w.workerID), &tempopb.UpdateJobStatusRequest{
		JobId:  resp.JobId,
		Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
	})
	if err != nil {
		return fmt.Errorf("failed marking job %q as complete: %w", resp.JobId, err)
	}

	return nil
}

func (w *BackendWorker) stopping(_ error) error {
	return nil
}

func (w *BackendWorker) failJob(ctx context.Context, jobID string, errMsg string) error {
	level.Error(log.Logger).Log("msg", "job failed", "job_id", jobID, "error", errMsg)

	_, err := w.backendScheduler.UpdateJob(user.InjectOrgID(ctx, w.workerID), &tempopb.UpdateJobStatusRequest{
		JobId:  jobID,
		Status: tempopb.JobStatus_JOB_STATUS_FAILED,
		Error:  errMsg,
	})
	if err != nil {
		return fmt.Errorf("failed marking job %q as failed: %w", jobID, err)
	}

	return fmt.Errorf("%s", errMsg)
}

func (w *BackendWorker) compact(ctx context.Context, blockMetas []*backend.BlockMeta, tenantID string) error {
	return w.store.CompactWithConfig(ctx, blockMetas, tenantID, &w.cfg.Compactor, w, w)
}

// Combine implements tempodb.CompactorSharder
func (w *BackendWorker) Combine(dataEncoding string, tenantID string, objs ...[]byte) ([]byte, bool, error) {
	combinedObj, wasCombined, err := model.StaticCombiner.Combine(dataEncoding, objs...)
	if err != nil {
		return nil, false, err
	}

	maxBytes := w.overrides.MaxBytesPerTrace(tenantID)
	if maxBytes == 0 || len(combinedObj) < maxBytes {
		return combinedObj, wasCombined, nil
	}

	// technically neither of these conditions should ever be true, we are adding them as guard code
	// for the following logic
	if len(objs) == 0 {
		return []byte{}, wasCombined, nil
	}
	if len(objs) == 1 {
		return objs[0], wasCombined, nil
	}

	totalDiscarded := countSpans(dataEncoding, objs[1:]...)
	overrides.RecordDiscardedSpans(totalDiscarded, reasonCompactorDiscardedSpans, tenantID)
	return objs[0], wasCombined, nil
}

// Copied from compactor module.  Centralize?
func countSpans(dataEncoding string, objs ...[]byte) (total int) {
	var traceID string
	decoder, err := model.NewObjectDecoder(dataEncoding)
	if err != nil {
		return 0
	}

	for _, o := range objs {
		t, err := decoder.PrepareForRead(o)
		if err != nil {
			continue
		}

		for _, b := range t.ResourceSpans {
			for _, ilm := range b.ScopeSpans {
				if len(ilm.Spans) > 0 && traceID == "" {
					traceID = tempo_util.TraceIDToHexString(ilm.Spans[0].TraceId)
				}
				total += len(ilm.Spans)
			}
		}
	}

	level.Debug(log.Logger).Log("msg", "max size of trace exceeded", "traceId", traceID, "discarded_span_count", total)

	return
}

func (w *BackendWorker) Owns(_ string) bool {
	return false
}

func (w *BackendWorker) RecordDiscardedSpans(count int, tenantID string, traceID string, rootSpanName string, rootServiceName string) {
	level.Warn(log.Logger).Log("msg", "max size of trace exceeded", "tenant", tenantID, "traceId", traceID,
		"rootSpanName", rootSpanName, "rootServiceName", rootServiceName, "discarded_span_count", count)
	overrides.RecordDiscardedSpans(count, reasonCompactorDiscardedSpans, tenantID)
}

// BlockRetentionForTenant implements CompactorOverrides
func (w *BackendWorker) BlockRetentionForTenant(tenantID string) time.Duration {
	return w.overrides.BlockRetention(tenantID)
}

// CompactionDisabledForTenant implements CompactorOverrides
func (w *BackendWorker) CompactionDisabledForTenant(tenantID string) bool {
	return w.overrides.CompactionDisabled(tenantID)
}

func (w *BackendWorker) MaxBytesPerTraceForTenant(tenantID string) int {
	return w.overrides.MaxBytesPerTrace(tenantID)
}

func (w *BackendWorker) MaxCompactionRangeForTenant(tenantID string) time.Duration {
	return w.overrides.MaxCompactionRange(tenantID)
}
