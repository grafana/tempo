package backendscheduler

import (
	"context"
	"flag"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	proto "github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var tenant = "test-tenant"

func TestBackendScheduler(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.BackendFlushInterval = 100 * time.Millisecond

	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, tmpDir)
	)

	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	t.Run("next with no jobs returns correct errors", func(t *testing.T) {
		s, err := New(cfg, store, limits, rr, ww)
		require.NoError(t, err)

		resp, err := s.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
		})
		require.Error(t, err)
		errStatus, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, errStatus.Code(), codes.NotFound)

		require.NotNil(t, resp)
		require.Equal(t, "", resp.JobId)
	})

	tenantCount := 5
	tenantBlockIDs := make(map[string][]backend.UUID)

	// Push some data to a few tenants
	for i := range tenantCount {
		testTenant := tenant + strconv.Itoa(i)
		tenantBlockIDs[testTenant] = writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 10)
	}

	time.Sleep(500 * time.Millisecond)

	t.Run("jobs need doing", func(t *testing.T) {
		s, err := New(cfg, store, limits, rr, ww)
		require.NoError(t, err)

		// Start the scheduler
		err = s.starting(ctx)
		require.NoError(t, err)

		go func() {
			// Start a goroutine to run the service
			err = s.running(ctx)
			require.NoError(t, err)
		}()

		defer func() {
			err := s.stopping(nil)
			require.NoError(t, err)
		}()

		// Let the providers start
		time.Sleep(200 * time.Millisecond)

		resp, err := s.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotEqual(t, "", resp.JobId)
		require.NotEqual(t, "", resp.Detail.Tenant)

		t.Run("requesting the next job with the same worker returns the same job", func(t *testing.T) {
			resp2, err := s.Next(ctx, &tempopb.NextJobRequest{
				WorkerId: "test-worker",
			})
			require.NoError(t, err)
			require.NotNil(t, resp2)
			require.Equal(t, resp.JobId, resp2.JobId)
		})

		updateResp, err := s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  resp.JobId,
			Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
			Compaction: &tempopb.CompactionDetail{
				Output: []string{uuid.New().String()},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)

		// Once the job is updated, ensure that the store no longer contains the job input block IDs
		metas := store.BlockMetas(resp.Detail.Tenant)

		for _, b := range resp.Detail.Compaction.Input {
			require.Contains(t, tenantBlockIDs[resp.Detail.Tenant], backend.MustParse(b))

			for _, m := range metas {
				require.NotEqual(t, m.BlockID, backend.MustParse(b))
			}
		}

		t.Run("the store does not contain the metas which have been compacted", func(t *testing.T) {
			metas := store.BlockMetas(resp.Detail.Tenant)

			for _, m := range metas {
				for _, b := range resp.Detail.Compaction.Input {
					require.NotContains(t, m.BlockID.String(), b)
				}
			}
		})

		resp, err = s.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotEqual(t, "", resp.JobId)

		updateResp, err = s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  resp.JobId,
			Status: tempopb.JobStatus_JOB_STATUS_FAILED,
			Compaction: &tempopb.CompactionDetail{
				Output: []string{uuid.New().String()},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)

		// Give some time for s to flush jobs to the backend
		time.Sleep(500 * time.Millisecond)

		t.Run("jobs are reloaded from cache", func(t *testing.T) {
			s2, err := New(cfg, store, limits, rr, ww)
			require.NoError(t, err)

			err = s2.starting(ctx)
			require.NoError(t, err)

			// Ensure that the jobs are the same
			for _, job := range s.work.ListJobs() {
				j := s2.work.GetJob(job.ID)
				require.NotNil(t, j)
				equalJobs(t, job, j)
			}

			for _, job := range s2.work.ListJobs() {
				j := s.work.GetJob(job.ID)
				require.NotNil(t, j)
				equalJobs(t, job, j)
			}
		})

		t.Run("jobs are reloaded from backend if local cache errors", func(t *testing.T) {
			cfg.LocalWorkPath = tmpDir + "/non-existent-path"

			s3, err := New(cfg, store, limits, rr, ww)
			require.NoError(t, err)

			err = s3.starting(ctx)
			require.NoError(t, err)

			// Ensure that the jobs are the same
			for _, job := range s.work.ListJobs() {
				j := s3.work.GetJob(job.ID)
				require.NotNil(t, j)
				equalJobs(t, job, j)
			}

			for _, job := range s3.work.ListJobs() {
				j := s.work.GetJob(job.ID)
				require.NotNil(t, j)
				equalJobs(t, job, j)
			}
		})

		// Drain all the jobs
		for i := 0; i < tenantCount*15; i++ {
			resp, err = s.Next(ctx, &tempopb.NextJobRequest{
				WorkerId: "test-worker",
			})
			if err != nil {
				statusErr, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.NotFound, statusErr.Code())
				break
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			updateResp, err = s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
				JobId:  resp.JobId,
				Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
				Compaction: &tempopb.CompactionDetail{
					Output: []string{uuid.New().String()},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, updateResp)
		}
	})
}

func equalJobs(t *testing.T, expected, actual *work.Job) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.CreatedTime.Unix(), actual.CreatedTime.Unix())
	require.Equal(t, expected.StartTime.Unix(), actual.StartTime.Unix())
	require.Equal(t, expected.EndTime.Unix(), actual.EndTime.Unix())
	require.Equal(t, expected.WorkerID, actual.WorkerID)
	require.Equal(t, expected.Retries, actual.Retries)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.JobDetail, actual.JobDetail)
}

func newStore(ctx context.Context, t testing.TB, tmpDir string) (storage.Store, backend.RawReader, backend.RawWriter) {
	rr, ww, _, err := local.New(&local.Config{
		Path: tmpDir + "/traces",
	})
	require.NoError(t, err)

	return newStoreWithLogger(ctx, t, test.NewTestingLogger(t), tmpDir), rr, ww
}

func newStoreWithLogger(ctx context.Context, t testing.TB, log log.Logger, tmpDir string) storage.Store {
	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local: &local.Config{
				Path: tmpDir + "/traces",
			},
			Block: &common.BlockConfig{
				BloomFP:             0.01,
				BloomShardSizeBytes: 100_000,
				Version:             encoding.LatestEncoding().Version(),
			},
			WAL: &wal.Config{
				Filepath: tmpDir + "/wal",
			},
			BlocklistPoll: 100 * time.Millisecond,
		},
	}, nil, log)
	require.NoError(t, err)

	s.EnablePolling(ctx, &ownsEverythingSharder{}, false)

	return s
}

func TestProtoMarshaler(t *testing.T) {
	_, err := proto.Marshal(&tempopb.JobDetail{
		Compaction: &tempopb.CompactionDetail{
			Input: []string{"input1", "input2"},
		},
	})
	require.NoError(t, err)

	detail := tempopb.JobDetail{
		Tenant: "test",
		Compaction: &tempopb.CompactionDetail{
			Input: []string{"input1", "input2"},
		},
	}

	_, err = proto.Marshal(&tempopb.NextJobResponse{
		JobId:  uuid.New().String(),
		Type:   tempopb.JobType_JOB_TYPE_COMPACTION,
		Detail: detail,
	})
	require.NoError(t, err)
}

func writeTenantBlocks(ctx context.Context, t *testing.T, w backend.Writer, tenant string, count int) []backend.UUID {
	var (
		err      error
		blockIDs []backend.UUID
	)

	for range count {
		meta := &backend.BlockMeta{
			BlockID:  backend.NewUUID(),
			TenantID: tenant,
			Version:  encoding.DefaultEncoding().Version(),
		}

		blockIDs = append(blockIDs, meta.BlockID)

		err = w.WriteBlockMeta(ctx, meta)
		require.NoError(t, err)
	}

	return blockIDs
}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}

func TestSubmitRedactionValidation(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, tmpDir)
	)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	testTenant := "tenant-validation"
	writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 1)
	time.Sleep(300 * time.Millisecond)

	tenantCtx := user.InjectOrgID(ctx, testTenant)

	validReq := &tempopb.SubmitRedactionRequest{
		TraceIds: [][]byte{[]byte(uuid.New().String())},
	}

	tests := []struct {
		name     string
		ctx      context.Context
		req      *tempopb.SubmitRedactionRequest
		wantCode codes.Code
	}{
		{
			name:     "missing context tenant",
			ctx:      ctx, // no org ID injected — simulates unauthenticated caller
			req:      &tempopb.SubmitRedactionRequest{TraceIds: [][]byte{[]byte("trace1")}},
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "empty trace_ids",
			ctx:      tenantCtx,
			req:      &tempopb.SubmitRedactionRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "duplicate submission",
			ctx:      tenantCtx,
			req:      validReq,
			wantCode: codes.AlreadyExists,
		},
	}

	// Regression: a body tenant_id different from the context tenant must be ignored —
	// jobs must be created for the authenticated tenant, not the body tenant.
	otherTenant := "tenant-other"
	writeTenantBlocks(ctx, t, backend.NewWriter(ww), otherTenant, 1)
	time.Sleep(300 * time.Millisecond)
	crossReq := &tempopb.SubmitRedactionRequest{
		TenantId: otherTenant, // attacker-supplied body tenant
		TraceIds: [][]byte{[]byte(uuid.New().String())},
	}
	crossResp, err := s.SubmitRedaction(tenantCtx, crossReq)
	require.NoError(t, err)
	require.Positive(t, crossResp.JobsCreated, "jobs must be created for the authenticated tenant")
	require.False(t, s.work.TenantPending(otherTenant), "body tenant_id must not be used")
	require.True(t, s.work.TenantPending(testTenant), "authenticated tenant must have a pending batch")
	// Clean up so the duplicate-submission test case fires correctly below.
	s.work.RemoveBatch(testTenant)

	// Seed an active batch so the duplicate-submission case fires.
	_, err = s.SubmitRedaction(tenantCtx, validReq)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.SubmitRedaction(tt.ctx, tt.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestSubmitRedactionAndRescan(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	// RescanDelay = 0: RescanAfterUnixNano is set to time.Now() at submission,
	// so checkPendingRescans fires as soon as we call it (with a brief sleep).
	cfg.ProviderConfig.Redaction.RescanDelay = 0

	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, tmpDir)
	)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	testTenant := "tenant-redact"

	// Write 5 blocks and wait for the blocklist poll to pick them up.
	blockIDs := writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 5)
	time.Sleep(300 * time.Millisecond)

	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)
	// Do NOT call s.starting — it would launch the RedactionProvider goroutine which
	// immediately drains pending jobs, racing with our IsBlockBusy assertions below.
	// The store blocklist is already populated via newStore + time.Sleep above.

	// Simulate a running compaction job covering the first two blocks.
	// RegisterJob must be called before AddJob so the block keys are indexed in
	// runningBlocks (matching production flow from the compaction provider).
	compJob := &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: testTenant,
			Compaction: &tempopb.CompactionDetail{
				Input: []string{blockIDs[0].String(), blockIDs[1].String()},
			},
		},
	}
	s.work.RegisterJob(compJob)
	require.NoError(t, s.work.AddJob(compJob))
	s.work.StartJob(compJob.ID)

	// Submit the redaction. Blocks 0 and 1 are in active compaction and must be
	// skipped; blocks 2, 3, 4 must receive pending jobs.
	resp, err := s.SubmitRedaction(user.InjectOrgID(ctx, testTenant), &tempopb.SubmitRedactionRequest{
		TraceIds: [][]byte{[]byte(uuid.New().String())},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, int32(3), resp.JobsCreated)

	// Build a map of blocks that received pending redaction jobs.
	pendingBlockSet := make(map[string]bool)
	for _, j := range s.work.ListAllPendingJobs() {
		pendingBlockSet[j.GetRedactionBlockID()] = true
	}

	// Blocks in active compaction must NOT have pending redaction jobs.
	require.False(t, pendingBlockSet[blockIDs[0].String()], "block 0 in active compaction should not have pending redaction")
	require.False(t, pendingBlockSet[blockIDs[1].String()], "block 1 in active compaction should not have pending redaction")

	// Remaining blocks must have pending redaction jobs.
	require.True(t, pendingBlockSet[blockIDs[2].String()], "block 2 should have pending redaction")
	require.True(t, pendingBlockSet[blockIDs[3].String()], "block 3 should have pending redaction")
	require.True(t, pendingBlockSet[blockIDs[4].String()], "block 4 should have pending redaction")

	// The batch must record the skipped compaction job ID and a rescan deadline.
	batch := s.work.GetBatch(testTenant)
	require.NotNil(t, batch)
	require.Equal(t, []string{compJob.ID}, batch.SkippedCompactionJobIds)
	require.Positive(t, batch.RescanAfterUnixNano)

	// Mark the compaction job done and record one output block.
	outputBlock := uuid.New().String()
	s.work.SetJobCompactionOutput(compJob.ID, []string{outputBlock})
	s.work.CompleteJob(compJob.ID)

	// Trigger the rescan. RescanDelay is 0, so after a brief sleep the
	// rescan deadline is in the past and checkPendingRescans fires.
	time.Sleep(time.Millisecond)
	s.checkPendingRescans(ctx)

	// The compaction output block must now have a pending redaction job.
	require.True(t, s.work.IsBlockBusy(testTenant, outputBlock))

	// The rescan deadline must have been cleared.
	batch = s.work.GetBatch(testTenant)
	require.NotNil(t, batch)
	require.Zero(t, batch.RescanAfterUnixNano)
}

// TestRescanSkipsRunningJob verifies that performRescan does not drop blocks when the
// skipped compaction job is still RUNNING at rescan time. The batch must be re-armed
// at the same generation; only when the job completes and rescan fires again should the
// output block receive a pending redaction job.
func TestRescanSkipsRunningJob(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.ProviderConfig.Redaction.RescanDelay = 0

	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, tmpDir)
	)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	testTenant := "tenant-rescan-running"

	blockIDs := writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 3)
	time.Sleep(300 * time.Millisecond)

	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	// Simulate a running compaction job covering the first two blocks.
	compJob := &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: testTenant,
			Compaction: &tempopb.CompactionDetail{
				Input: []string{blockIDs[0].String(), blockIDs[1].String()},
			},
		},
	}
	s.work.RegisterJob(compJob)
	require.NoError(t, s.work.AddJob(compJob))
	s.work.StartJob(compJob.ID)

	// Submit the redaction; blocks 0+1 under compaction are skipped.
	resp, err := s.SubmitRedaction(user.InjectOrgID(ctx, testTenant), &tempopb.SubmitRedactionRequest{
		TraceIds: [][]byte{[]byte(uuid.New().String())},
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.JobsCreated) // only block 2

	batch := s.work.GetBatch(testTenant)
	require.NotNil(t, batch)
	require.Equal(t, []string{compJob.ID}, batch.SkippedCompactionJobIds)

	// Fire rescan while the compaction job is still RUNNING (not complete).
	// The batch must be re-armed with the same job ID and the same generation.
	time.Sleep(time.Millisecond)
	s.checkPendingRescans(ctx)

	batch = s.work.GetBatch(testTenant)
	require.NotNil(t, batch, "batch must not be removed while rescan is pending")
	require.Equal(t, []string{compJob.ID}, batch.SkippedCompactionJobIds, "still-running job must remain in skipped list")
	require.Positive(t, batch.RescanAfterUnixNano, "rescan deadline must be re-armed")

	// The output block must not yet have a pending redaction job.
	outputBlock := uuid.New().String()
	require.False(t, s.work.IsBlockBusy(testTenant, outputBlock))

	// Now complete the compaction job with one output block.
	s.work.SetJobCompactionOutput(compJob.ID, []string{outputBlock})
	s.work.CompleteJob(compJob.ID)

	// Second rescan: job is now complete, output block must be enqueued.
	time.Sleep(time.Millisecond)
	s.checkPendingRescans(ctx)

	require.True(t, s.work.IsBlockBusy(testTenant, outputBlock), "output block must have a pending redaction job after rescan")

	batch = s.work.GetBatch(testTenant)
	require.NotNil(t, batch)
	require.Zero(t, batch.RescanAfterUnixNano, "rescan deadline must be cleared after successful enqueue")
}

func TestProviderBasedScheduling(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.ProviderConfig.Retention.Interval = 100 * time.Millisecond

	tmpDir := t.TempDir()
	cfg.LocalWorkPath = t.TempDir()

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, tmpDir)
	)

	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	tenantBlockIDs := make(map[string][]backend.UUID)

	// Push some data to a few tenants
	tenantCount := 3
	for i := range tenantCount {
		testTenant := tenant + strconv.Itoa(i)
		tenantBlockIDs[testTenant] = writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 5)
	}

	time.Sleep(500 * time.Millisecond)

	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	// Start the service
	err = s.starting(ctx)
	require.NoError(t, err)

	// Start a goroutine to run the service
	go func() {
		// Start a goroutine to run the service
		err = s.running(ctx)
		require.NoError(t, err)
	}()

	defer func() {
		err := s.stopping(nil)
		require.NoError(t, err)
	}()

	// Wait for providers to start and generate jobs
	time.Sleep(200 * time.Millisecond)

	// Request jobs and verify they're coming from providers
	var compactionJobs, retentionJobs int
	for i := 0; i < 10; i++ {
		resp, err := s.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker-" + strconv.Itoa(i),
		})
		if err != nil {
			statusErr, ok := status.FromError(err)
			require.True(t, ok)
			if statusErr.Code() == codes.NotFound {
				// No more jobs, break
				break
			}
			require.NoError(t, err)
		}

		require.NotNil(t, resp)
		require.NotEmpty(t, resp.JobId)

		switch resp.Type {
		case tempopb.JobType_JOB_TYPE_COMPACTION:
			compactionJobs++
		case tempopb.JobType_JOB_TYPE_RETENTION:
			retentionJobs++
		}

		// Complete the job
		updateResp, err := s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  resp.JobId,
			Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)
	}

	// We should have at least one retention job and some compaction jobs
	require.GreaterOrEqual(t, retentionJobs, 1)
	require.GreaterOrEqual(t, compactionJobs, 1)
}

// TestCleanupOrphanedBatchesAfterDeadJobTimeout verifies that
// cleanupOrphanedBatches removes a batch whose redaction jobs were all
// transitioned to FAILED by the Prune dead-job timeout path.
//
// Regression test for the bug where Prune called j.Fail() directly (to avoid
// re-acquiring the shard lock), bypassing UpdateJob → cleanupBatchIfDone, and
// leaving the batch in batchStore permanently. The fix adds
// cleanupOrphanedBatches to the maintenance tick after each Prune call.
func TestCleanupOrphanedBatchesAfterDeadJobTimeout(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	// RescanDelay = 0 and no compaction jobs racing with submission, so the
	// batch has RescanAfterUnixNano == 0 and cleanupBatchIfDone won't block on it.
	cfg.ProviderConfig.Redaction.RescanDelay = 0

	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	testTenant := "tenant-orphan-batch"
	writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 2)
	time.Sleep(300 * time.Millisecond)

	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)
	// Do NOT call s.starting — it would launch background goroutines that race
	// with the manual job lifecycle below.

	// Submit a redaction. With 2 blocks, we expect 2 pending jobs.
	tenantCtx := user.InjectOrgID(ctx, testTenant)
	resp, err := s.SubmitRedaction(tenantCtx, &tempopb.SubmitRedactionRequest{
		TraceIds: [][]byte{[]byte(uuid.New().String())},
	})
	require.NoError(t, err)
	require.Greater(t, int(resp.JobsCreated), 0)
	require.True(t, s.work.TenantPending(testTenant), "batch must be active after submission")

	// Simulate workers picking up all pending redaction jobs. For each job:
	// pop from pending → register → assign worker → promote to active → start.
	// This mirrors the production path in backendscheduler.Next().
	for i := 0; ; i++ {
		j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
		if j == nil {
			break
		}
		s.work.RegisterJob(j)
		j.SetWorkerID("worker-" + strconv.Itoa(i))
		require.NoError(t, s.work.AddJob(j))
		s.work.StartJob(j.ID)
		// Back-date the start time so Prune sees the job as past DeadJobTimeout
		// (default 24h). Use 25h to give a clear margin.
		s.work.GetJob(j.ID).StartTime = time.Now().Add(-25 * time.Hour)
	}

	require.True(t, s.work.HasJobsForTenant(testTenant, tempopb.JobType_JOB_TYPE_REDACTION),
		"jobs must be running before prune")

	// Prune transitions timed-out running jobs to FAILED and cleans up indexes,
	// but does NOT call cleanupBatchIfDone — that is the bug this test covers.
	s.work.Prune(ctx)

	require.False(t, s.work.HasJobsForTenant(testTenant, tempopb.JobType_JOB_TYPE_REDACTION),
		"no active jobs should remain after prune")
	require.NotNil(t, s.work.GetBatch(testTenant),
		"batch must still exist after prune alone (orphaned batch bug)")

	// cleanupOrphanedBatches is the fix: it sweeps all batches and removes any
	// whose jobs have all finished. This is now called on every maintenance tick.
	s.cleanupOrphanedBatches(ctx)

	require.Nil(t, s.work.GetBatch(testTenant),
		"batch must be removed after cleanupOrphanedBatches")
	require.False(t, s.work.TenantPending(testTenant),
		"tenant must not be blocked after batch cleanup")

	// A new redaction submission must succeed — the tenant is no longer locked out.
	_, err = s.SubmitRedaction(tenantCtx, &tempopb.SubmitRedactionRequest{
		TraceIds: [][]byte{[]byte(uuid.New().String())},
	})
	require.NoError(t, err, "SubmitRedaction must not return AlreadyExists after batch cleanup")
}
