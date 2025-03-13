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

func TestBackendScheduler(t *testing.T) {
	cfg := Config{
		ScheduleInterval:       100 * time.Millisecond,
		TenantPriorityInterval: 100 * time.Millisecond,
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

	t.Run("no tenants and no jobs", func(t *testing.T) {
		t.Skip()
		bs, err := New(cfg, store, limits, rr, ww)
		require.NoError(t, err)

		err = bs.scheduleOnce(ctx, 10)
		require.NoError(t, err)

		resp, err := bs.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "", resp.JobId)
	})
}

var tenant = "test-tenant"

func TestBackendScheduler_gRPC(t *testing.T) {
	cfg := Config{
		ScheduleInterval:       100 * time.Millisecond,
		TenantPriorityInterval: 100 * time.Millisecond,
	}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	tmpDir := t.TempDir()

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, tmpDir)
	)
	defer cancel()
	defer store.Shutdown()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	t.Run("next with no jobs returns correct errors", func(t *testing.T) {
		s, err := New(cfg, store, limits, rr, ww)
		require.NoError(t, err)

		err = s.scheduleOnce(ctx, 10)
		require.NoError(t, err)

		resp, err := s.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
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

		s.prioritizeTenants()
		err = s.scheduleOnce(ctx, 10)
		require.NoError(t, err)

		s.prioritizeTenants()

		resp, err := s.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotEqual(t, "", resp.JobId)
		require.NotEqual(t, "", resp.Detail.Tenant)
		tenant := resp.Detail.Tenant

		updateResp, err := s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  resp.JobId,
			Status: tempopb.JobStatus_JOB_STATUS_RUNNING,
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)

		updateResp, err = s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  resp.JobId,
			Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
			Compaction: &tempopb.CompactionDetail{
				Output: []string{uuid.New().String()},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)

		resp, err = s.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
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

		since := time.Since(s.lastWorkForTenant(tenant))
		require.True(t, since < 2*time.Second)
	})
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
