package backendscheduler

import (
	"context"
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
)

func TestBackendScheduler(t *testing.T) {
	cfg := Config{
		ScheduleInterval: 100 * time.Millisecond,
	}

	// tmpDir := t.TempDir()
	// writer := setupBackend(t, tmpDir)

	var (
		ctx   = context.Background()
		store = newStore(ctx, t)
	)

	limits, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			// Global: overrides.GlobalOverrides{
			// 	MaxBytesPerTrace: maxBytes,
			// },
			// Ingestion: overrides.IngestionOverrides{
			// 	MaxLocalTracesPerUser: 4,
			// },
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	t.Run("no tenants and no jobs", func(t *testing.T) {
		bs, err := New(cfg, store, limits)
		require.NoError(t, err)

		err = bs.ScheduleOnce(ctx)
		require.NoError(t, err)

		resp, err := bs.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "", resp.JobId)
	})

	t.Run("one tenant has a jobs", func(t *testing.T) {
		bs, err := New(cfg, store, limits)
		require.NoError(t, err)

		j := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: "test-tenant",
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err = bs.CreateJob(ctx, j)
		require.NoError(t, err)

		err = bs.ScheduleOnce(ctx)
		require.NoError(t, err)

		resp, err := bs.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, j.ID, resp.JobId)
	})

	t.Run("a request for a compaction job returns only a compaction job type", func(t *testing.T) {
		bs, err := New(cfg, store, limits)
		require.NoError(t, err)

		j1 := &work.Job{
			ID: uuid.New().String(),
			// Type: tempopb.JobType_JOB_TYPE_UNSPECIFIED,
			JobDetail: tempopb.JobDetail{
				Tenant: "test-tenant",
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		j2 := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: "test-tenant",
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err = bs.CreateJob(ctx, j1)
		require.NoError(t, err)

		err = bs.CreateJob(ctx, j2)
		require.NoError(t, err)

		err = bs.ScheduleOnce(ctx)
		require.NoError(t, err)

		resp, err := bs.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, j2.ID, resp.JobId)
		require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, resp.Type)

		// Returns the same job if called again with the same worker ID
		resp, err = bs.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: "test-worker",
			Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, j2.ID, resp.JobId)
	})

	t.Run("handles multiple workers", func(t *testing.T) {
		bs, err := New(cfg, store, limits)
		require.NoError(t, err)

		tenant := "test-tenant"

		j1 := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: tenant,
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err = bs.CreateJob(ctx, j1)
		require.NoError(t, err)

		j2 := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: tenant,
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err = bs.CreateJob(ctx, j2)
		require.NoError(t, err)

		j3 := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: tenant,
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err = bs.CreateJob(ctx, j3)
		require.NoError(t, err)

		j4 := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: tenant,
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err = bs.CreateJob(ctx, j4)
		require.NoError(t, err)

		err = bs.ScheduleOnce(ctx)
		require.NoError(t, err)

		// Write test blocks for multiple jobs
		// tenant := "test-tenant"
		// blockIDs := writeTestBlocks(t, ctx, store, tenant, 6) // 3 jobs with 2 blocks each
		// require.Len(t, blockIDs, 6)

		// Create multiple jobs
		// for i := 0; i < 3; i++ {
		// 	jobBlocks := blockIDs[i*2 : (i+1)*2]
		// 	err := bs.CreateCompactionJob(ctx, tenant, jobBlocks, fmt.Sprintf("output-block-%d", i))
		// 	require.NoError(t, err)
		// }

		// Different workers should get different jobs
		worker1Jobs := make(map[string]*tempopb.NextJobResponse)
		worker2Jobs := make(map[string]*tempopb.NextJobResponse)

		for i := 0; i < 2; i++ {
			resp, err := bs.Next(ctx, &tempopb.NextJobRequest{
				WorkerId: "worker1",
				Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
			})
			require.NoError(t, err)
			require.NotNil(t, resp)
			worker1Jobs[resp.JobId] = resp

			resp, err = bs.Next(ctx, &tempopb.NextJobRequest{
				WorkerId: "worker2",
				Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
			})
			require.NoError(t, err)
			if resp != nil {
				worker2Jobs[resp.JobId] = resp
			}
		}

		require.NotEmpty(t, worker1Jobs)
		require.NotEmpty(t, worker2Jobs)

		// Verify jobs were distributed
		for id := range worker1Jobs {
			_, exists := worker2Jobs[id]
			require.False(t, exists, "same job assigned to multiple workers")
		}

		for id := range worker2Jobs {
			_, exists := worker1Jobs[id]
			require.False(t, exists, "same job assigned to multiple workers")
		}

		// Mark jobs failed or complete
		resp, err := bs.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  j1.ID,
			Status: tempopb.JobStatus_JOB_STATUS_FAILED,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		resp, err = bs.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  j1.ID,
			Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// unknown job id
		resp, err = bs.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  uuid.New().String(),
			Status: tempopb.JobStatus_JOB_STATUS_FAILED,
		})
		require.Error(t, err)
		require.Nil(t, resp)

		// Completed and failed jobs are not cleaned up by this point.
		currentJobs := bs.ListJobs(ctx, tenant)
		require.Len(t, currentJobs, 4)
	})

	t.Run("CRUD operation testing", func(t *testing.T) {
		tenant := "test-tenant"

		bs, err := New(cfg, store, limits)
		require.NoError(t, err)

		j1 := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: tenant,
				Compaction: &tempopb.CompactionDetail{
					Input: []string{uuid.New().String(), uuid.New().String()},
				},
			},
		}

		err = bs.CreateJob(ctx, j1)
		require.NoError(t, err)

		currentJobs := bs.ListJobs(ctx, tenant)
		require.Len(t, currentJobs, 1)
		require.Equal(t, j1.ID, currentJobs[0].ID)
		require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, currentJobs[0].Type)
		require.Equal(t, tenant, currentJobs[0].JobDetail.Tenant)

		resp, err := bs.GetJob(ctx, j1.ID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, j1.ID, resp.ID)
		require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, resp.Type)
		require.Equal(t, tenant, resp.JobDetail.Tenant)

		err = bs.CompleteJob(ctx, j1.ID)
		require.NoError(t, err)

		resp, err = bs.GetJob(ctx, j1.ID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, work.JobStatusCompleted, resp.Status())

		err = bs.FailJob(ctx, j1.ID)
		require.NoError(t, err)

		resp, err = bs.GetJob(ctx, j1.ID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, work.JobStatusFailed, resp.Status())
	})
}

func newStore(ctx context.Context, t testing.TB) storage.Store {
	return newStoreWithLogger(ctx, t, test.NewTestingLogger(t))
}

func newStoreWithLogger(ctx context.Context, t testing.TB, log log.Logger) storage.Store {
	tmpDir := t.TempDir()

	_, _, _, err := local.New(&local.Config{
		Path: tmpDir,
	})
	require.NoError(t, err)

	// w := backend.NewWriter(ww)
	//
	// w.WriteBlockMeta(ctx, &backend.BlockMeta{
	// 	BlockID:           backend.NewUUID(),
	// 	TenantID:          "test-tenant",
	// 	StartTime:         time.Now().Add(-time.Hour),
	// 	EndTime:           time.Now(),
	// 	TotalObjects:      1,
	// 	ReplicationFactor: 1,
	// 	Version:           encoding.LatestEncoding().Version(),
	// 	Encoding:          backend.EncNone,
	// })
	//
	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local: &local.Config{
				Path: tmpDir,
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
				Filepath: tmpDir,
			},
			BlocklistPoll: 100 * time.Millisecond,
		},
	}, nil, log)
	require.NoError(t, err)

	s.EnablePolling(ctx, &ownsEverythingSharder{})

	// NOTE: Call EnableCompaction to set the overrides, but pass a canceled
	// context so we don't run the compaction and retention loops.
	// canceldCtx, cancel := context.WithCancel(ctx)
	// cancel()

	// err = s.EnableCompaction(canceldCtx, &tempodb.CompactorConfig{
	// 	ChunkSizeBytes:          10,
	// 	MaxCompactionRange:      24 * time.Hour,
	// 	BlockRetention:          0,
	// 	CompactedBlockRetention: 0,
	// 	MaxCompactionObjects:    1000,
	// 	MaxBlockBytes:           100_000_000, // Needs to be sized appropriately for the test data or no jobs will get scheduled.
	// }, &ownsEverythingSharder{}, &mockOverrides{})
	// require.NoError(t, err)
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

// OwnsEverythingSharder owns everything.
var OwnsEverythingSharder = ownsEverythingSharder{}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}
