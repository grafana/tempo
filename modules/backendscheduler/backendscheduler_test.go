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
	cfg := Config{
		TenantMeasurementInterval: 100 * time.Millisecond,
	}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	tmpDir := t.TempDir()

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

	_ = backend.NewReader(rr)
	w := backend.NewWriter(ww)

	tenantCount := 5

	// Push some data to a few tenants
	for i := 0; i < tenantCount; i++ {
		testTenant := tenant + strconv.Itoa(i)
		writeTenantBlocks(ctx, t, w, testTenant, 10)
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
				IndexDownsampleBytes: 2,
				BloomFP:              0.01,
				BloomShardSizeBytes:  100_000,
				Version:              encoding.LatestEncoding().Version(),
				Encoding:             backend.EncLZ4_1M,
				IndexPageSizeBytes:   1000,
			},
			WAL: &wal.Config{
				Filepath: tmpDir + "/wal",
			},
			BlocklistPoll: 100 * time.Millisecond,
		},
	}, nil, log)
	require.NoError(t, err)

	s.EnablePolling(ctx, &ownsEverythingSharder{})

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

func writeTenantBlocks(ctx context.Context, t *testing.T, w backend.Writer, tenant string, count int) {
	var err error
	for i := 0; i < count; i++ {
		meta := &backend.BlockMeta{
			BlockID:  backend.NewUUID(),
			TenantID: tenant,
		}

		err = w.WriteBlockMeta(ctx, meta)
		require.NoError(t, err)
	}
}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}

func TestProviderBasedScheduling(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.TenantMeasurementInterval = 100 * time.Millisecond
	cfg.ProviderConfig.Retention.Interval = 100 * time.Millisecond

	tmpDir := t.TempDir()

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

	// Push some data to a few tenants
	tenantCount := 3
	for i := 0; i < tenantCount; i++ {
		testTenant := tenant + strconv.Itoa(i)
		writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 5)
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
